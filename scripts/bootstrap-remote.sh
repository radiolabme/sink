#!/bin/bash
# bootstrap-remote.sh - Bootstrap Sink on remote hosts via SSH
#
# Usage: ./bootstrap-remote.sh <ssh-target> <config-source> [options]
#
# Examples:
#   ./bootstrap-remote.sh user@host setup.json
#   ./bootstrap-remote.sh user@host https://configs.example.com/setup.json
#   ./bootstrap-remote.sh user@host http://example.com/setup.json --sha256 abc123...

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
  cat << EOF
Sink Remote Bootstrap - Deploy and execute Sink on remote hosts

Usage: $0 <ssh-target> <config-source> [options]

Arguments:
  ssh-target      SSH target (user@host or user@host:port)
  config-source   Local file path or URL (http:// or https://)

Options:
  --sha256 <hash>   SHA256 checksum (required for HTTP, optional for HTTPS)
  --binary <path>   Path to sink binary (default: ./bin/sink)
  --no-cleanup      Don't remove files after execution
  --dry-run         Preview actions without executing
  --platform <os>   Override platform detection (darwin, linux, windows)
  --yes             Skip confirmation prompt
  --verbose         Show detailed output
  --help            Show this help message

Security:
  - HTTPS URLs: Verified via TLS certificate validation
  - HTTP URLs:  MUST provide --sha256 for verification
  - SSH:        All transfers encrypted via SSH

Examples:
  # Local config file
  $0 user@host setup.json

  # HTTPS URL (TLS verified)
  $0 user@host https://configs.example.com/prod.json

  # HTTP URL with SHA256 verification
  $0 user@host http://configs.example.com/setup.json \\
    --sha256 \$(sha256sum setup.json | cut -d' ' -f1)

  # Custom binary location
  $0 user@host setup.json --binary ~/Downloads/sink

  # Dry run (preview only)
  $0 user@host setup.json --dry-run

  # Multiple hosts (bash loop)
  for host in host{1..5}; do
    $0 user@\$host setup.json
  done

EOF
  exit 1
}

log_info() {
  echo -e "${BLUE}ℹ${NC}  $1"
}

log_success() {
  echo -e "${GREEN}✅${NC} $1"
}

log_warning() {
  echo -e "${YELLOW}⚠${NC}  $1"
}

log_error() {
  echo -e "${RED}❌${NC} $1" >&2
}

log_step() {
  echo -e "${BLUE}▶${NC}  $1"
}

# Verify GitHub URL pinning
verify_github_pin() {
  local url="$1"
  
  # Extract GitHub info from raw.githubusercontent.com or github.com URLs
  if [[ "$url" =~ raw\.githubusercontent\.com/([^/]+)/([^/]+)/([^/]+)/ ]]; then
    local owner="${BASH_REMATCH[1]}"
    local repo="${BASH_REMATCH[2]}"
    local ref="${BASH_REMATCH[3]}"
    
    # Check if ref is a semantic version tag (vX.Y.Z format)
    if [[ "$ref" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
      log_success "GitHub: Pinned to release tag '$ref' ✓"
      return 0
    fi
    
    # Check if ref is a commit SHA (40 hex chars or 7+ char short SHA)
    if [[ "$ref" =~ ^[0-9a-f]{40}$ ]]; then
      log_success "GitHub: Pinned to commit '${ref:0:8}...' ✓"
      return 0
    elif [[ "$ref" =~ ^[0-9a-f]{7,}$ ]]; then
      log_success "GitHub: Pinned to short commit '${ref}' ✓"
      return 0
    fi
    
    # Check if ref is a mutable branch
    if [[ "$ref" =~ ^(main|master|develop|dev|staging|production|trunk)$ ]]; then
      log_warning "GitHub: Using MUTABLE branch '$ref'"
      log_warning "Branch content can change - consider pinning to a tag or commit"
      log_info "Example: https://raw.githubusercontent.com/$owner/$repo/v1.0.0/..."
      return 0
    fi
    
    # Unknown ref format - assume it might be a tag
    log_info "GitHub: Using ref '$ref' (assuming tag or branch)"
    return 0
  fi
  
  # Check github.com/owner/repo/releases/download/tag/file format
  if [[ "$url" =~ github\.com/([^/]+)/([^/]+)/releases/download/([^/]+)/ ]]; then
    local owner="${BASH_REMATCH[1]}"
    local repo="${BASH_REMATCH[2]}"
    local tag="${BASH_REMATCH[3]}"
    
    log_success "GitHub Release: Pinned to '$tag' ✓✓"
    log_info "Repository: $owner/$repo"
    return 0
  fi
  
  # Not a GitHub URL - no verification needed
  return 0
}

# Auto-fetch checksum from GitHub
auto_fetch_checksum() {
  local config_url="$1"
  
  # Try .sha256 file first
  local checksum_url="${config_url}.sha256"
  
  if [ "$VERBOSE" = true ]; then
    log_info "Attempting to auto-fetch checksum from: $checksum_url"
  fi
  
  if local checksum_content=$(curl -fsSL "$checksum_url" 2>/dev/null); then
    # Extract just the hash (handles both "hash filename" and "hash" formats)
    local checksum=$(echo "$checksum_content" | grep -Eo '^[0-9a-f]{64}' | head -1)
    if [ -n "$checksum" ]; then
      log_success "Auto-fetched SHA256 from .sha256 file"
      if [ "$VERBOSE" = true ]; then
        log_info "Checksum: ${checksum:0:16}..."
      fi
      echo "$checksum"
      return 0
    fi
  fi
  
  return 1
}

# Parse arguments
SSH_TARGET="${1:-}"
CONFIG_SOURCE="${2:-}"

if [ "$#" -lt 2 ]; then
  usage
fi

shift 2

SHA256=""
BINARY="./bin/sink"
CLEANUP=true
DRY_RUN=false
PLATFORM_OVERRIDE=""
AUTO_YES=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --sha256)
      SHA256="$2"
      shift 2
      ;;
    --binary)
      BINARY="$2"
      shift 2
      ;;
    --no-cleanup)
      CLEANUP=false
      shift
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --platform)
      PLATFORM_OVERRIDE="$2"
      shift 2
      ;;
    --yes|-y)
      AUTO_YES=true
      shift
      ;;
    --verbose|-v)
      VERBOSE=true
      shift
      ;;
    --help|-h)
      usage
      ;;
    *)
      log_error "Unknown option: $1"
      usage
      ;;
  esac
done

# Validate inputs
if [ -z "$SSH_TARGET" ]; then
  log_error "SSH target required"
  usage
fi

if [ -z "$CONFIG_SOURCE" ]; then
  log_error "Config source required"
  usage
fi

if [ ! -f "$BINARY" ]; then
  log_error "Binary not found at: $BINARY"
  log_info "Build it with: make build"
  exit 1
fi

# Generate unique temp names
RANDOM_ID="sink-$$-$RANDOM"
REMOTE_SINK="/tmp/$RANDOM_ID"
REMOTE_CONFIG="/tmp/$RANDOM_ID.json"

# Cleanup function
cleanup() {
  local exit_code=$?
  if [ "$CLEANUP" = true ] && [ "$DRY_RUN" = false ]; then
    if [ "$VERBOSE" = true ]; then
      log_step "Cleaning up remote files..."
    fi
    ssh "$SSH_TARGET" "rm -f '$REMOTE_SINK' '$REMOTE_CONFIG'" 2>/dev/null || true
  fi
  exit $exit_code
}
trap cleanup EXIT INT TERM

# Test SSH connectivity
test_ssh() {
  if [ "$VERBOSE" = true ]; then
    log_step "Testing SSH connectivity to $SSH_TARGET..."
  fi
  
  if ! ssh -o ConnectTimeout=5 -o BatchMode=yes "$SSH_TARGET" "echo 'SSH OK'" >/dev/null 2>&1; then
    if ! ssh -o ConnectTimeout=5 "$SSH_TARGET" "echo 'SSH OK'" >/dev/null 2>&1; then
      log_error "Cannot connect to $SSH_TARGET"
      log_info "Check SSH keys, credentials, and network connectivity"
      exit 1
    fi
  fi
  
  if [ "$VERBOSE" = true ]; then
    log_success "SSH connection verified"
  fi
}

# Main execution
main() {
  echo ""
  echo "🚀 Sink Remote Bootstrap"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "   Target: $SSH_TARGET"
  echo "   Config: $CONFIG_SOURCE"
  if [ "$DRY_RUN" = true ]; then
    echo "   Mode:   DRY RUN (preview only)"
  fi
  echo ""

  # Test SSH
  test_ssh

  # Step 1: Transfer binary
  log_step "Transferring sink binary..."
  if [ "$DRY_RUN" = true ]; then
    log_info "[DRY-RUN] Would transfer: $BINARY → $SSH_TARGET:$REMOTE_SINK"
  else
    local binary_size=$(du -h "$BINARY" | cut -f1)
    if [ "$VERBOSE" = true ]; then
      log_info "Binary size: $binary_size"
    fi
    
    if scp -q "$BINARY" "$SSH_TARGET:$REMOTE_SINK" 2>/dev/null; then
      ssh "$SSH_TARGET" "chmod +x '$REMOTE_SINK'" 2>/dev/null
      log_success "Binary transferred ($binary_size)"
    else
      log_error "Failed to transfer binary"
      exit 1
    fi
  fi

  # Step 2: Handle config
  if [[ "$CONFIG_SOURCE" =~ ^https?:// ]]; then
    # URL-based config
    local protocol=$(echo "$CONFIG_SOURCE" | cut -d: -f1)
    
    # Verify GitHub URL pinning if applicable
    verify_github_pin "$CONFIG_SOURCE"
    
    # Auto-fetch checksum if not provided and it's HTTPS
    if [ -z "$SHA256" ] && [ "$protocol" = "https" ]; then
      if [[ "$CONFIG_SOURCE" =~ github\.com|githubusercontent\.com ]]; then
        if SHA256=$(auto_fetch_checksum "$CONFIG_SOURCE"); then
          log_info "Using auto-fetched checksum for verification"
        fi
      fi
    fi
    
    # Validate HTTP requires SHA256
    if [ "$protocol" = "http" ] && [ -z "$SHA256" ]; then
      log_error "HTTP URLs require --sha256 for verification"
      log_info "Generate SHA256: sha256sum your-config.json"
      log_info "Then use: --sha256 \$(sha256sum your-config.json | cut -d' ' -f1)"
      exit 1
    fi
    
    log_step "Downloading config from URL..."
    if [ "$protocol" = "http" ]; then
      log_warning "Using HTTP (not secure) - SHA256 verification required"
    fi
    
    if [ "$DRY_RUN" = true ]; then
      log_info "[DRY-RUN] Would download: $CONFIG_SOURCE → $REMOTE_CONFIG"
    else
      if ssh "$SSH_TARGET" "curl -fsSL -o '$REMOTE_CONFIG' '$CONFIG_SOURCE'" 2>/dev/null; then
        log_success "Config downloaded"
      else
        log_error "Failed to download config from $CONFIG_SOURCE"
        exit 1
      fi
    fi
    
    # Verify SHA256 if provided
    if [ -n "$SHA256" ]; then
      log_step "Verifying SHA256 checksum..."
      if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] Would verify SHA256: ${SHA256:0:16}..."
      else
        local verify_script="
          ACTUAL=\$(sha256sum '$REMOTE_CONFIG' | cut -d' ' -f1)
          if [ \"\$ACTUAL\" != '$SHA256' ]; then
            echo 'MISMATCH'
            echo \"\$ACTUAL\"
            exit 1
          fi
          echo 'OK'
        "
        
        local result=$(ssh "$SSH_TARGET" "$verify_script" 2>&1)
        if echo "$result" | grep -q "^OK"; then
          log_success "SHA256 verified: ${SHA256:0:16}..."
        else
          log_error "SHA256 mismatch!"
          log_error "Expected: $SHA256"
          log_error "Actual:   $(echo "$result" | tail -1)"
          exit 1
        fi
      fi
    elif [ "$protocol" = "https" ]; then
      log_success "HTTPS - verified via TLS"
    fi
  else
    # Local file
    if [ ! -f "$CONFIG_SOURCE" ]; then
      log_error "Config file not found: $CONFIG_SOURCE"
      exit 1
    fi
    
    log_step "Transferring config file..."
    local config_size=$(du -h "$CONFIG_SOURCE" | cut -f1)
    if [ "$VERBOSE" = true ]; then
      log_info "Config size: $config_size"
    fi
    
    if [ "$DRY_RUN" = true ]; then
      log_info "[DRY-RUN] Would transfer: $CONFIG_SOURCE → $SSH_TARGET:$REMOTE_CONFIG"
    else
      if scp -q "$CONFIG_SOURCE" "$SSH_TARGET:$REMOTE_CONFIG" 2>/dev/null; then
        log_success "Config transferred ($config_size)"
      else
        log_error "Failed to transfer config"
        exit 1
      fi
    fi
  fi

  # Step 3: Validate config remotely (optional but recommended)
  if [ "$DRY_RUN" = false ]; then
    log_step "Validating config on remote host..."
    if ssh "$SSH_TARGET" "'$REMOTE_SINK' validate '$REMOTE_CONFIG'" 2>&1 | grep -q "✅"; then
      log_success "Config is valid"
    else
      log_error "Config validation failed"
      exit 1
    fi
  fi

  # Step 4: Execute
  echo ""
  log_step "Executing sink on remote host..."
  
  if [ "$DRY_RUN" = true ]; then
    log_info "[DRY-RUN] Would execute: $REMOTE_SINK execute $REMOTE_CONFIG"
    if [ -n "$PLATFORM_OVERRIDE" ]; then
      log_info "[DRY-RUN] Platform override: $PLATFORM_OVERRIDE"
    fi
    if [ "$AUTO_YES" = true ]; then
      log_info "[DRY-RUN] Auto-confirm: yes"
    fi
  else
    local exec_cmd="'$REMOTE_SINK' execute '$REMOTE_CONFIG'"
    if [ -n "$PLATFORM_OVERRIDE" ]; then
      exec_cmd="$exec_cmd --platform '$PLATFORM_OVERRIDE'"
    fi
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Execute with or without auto-confirmation
    if [ "$AUTO_YES" = true ]; then
      # Send 'yes' to stdin for auto-confirmation
      echo "yes" | ssh -t "$SSH_TARGET" "$exec_cmd" || {
        log_error "Execution failed on remote host"
        exit 1
      }
    else
      # Interactive mode - user confirms on remote
      ssh -t "$SSH_TARGET" "$exec_cmd" || {
        log_error "Execution failed on remote host"
        exit 1
      }
    fi
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  fi

  echo ""
  log_success "Bootstrap complete!"
  
  if [ "$CLEANUP" = false ]; then
    echo ""
    log_info "Files left on remote (--no-cleanup):"
    log_info "  Binary: $REMOTE_SINK"
    log_info "  Config: $REMOTE_CONFIG"
  fi
}

# Run main
main
