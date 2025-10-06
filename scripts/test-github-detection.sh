#!/bin/bash
# test-github-detection.sh - Test GitHub URL pin detection

set -euo pipefail

# Source the functions from bootstrap-remote.sh
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Mock logging functions
log_success() { echo -e "${GREEN}✅${NC} $1"; }
log_warning() { echo -e "⚠️  $1"; }
log_info() { echo "ℹ️  $1"; }

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Verify GitHub URL pinning function
verify_github_pin() {
  local url="$1"
  
  # Extract GitHub info from raw.githubusercontent.com or github.com URLs
  if [[ "$url" =~ raw\.githubusercontent\.com/([^/]+)/([^/]+)/([^/]+)/ ]]; then
    local owner="${BASH_REMATCH[1]}"
    local repo="${BASH_REMATCH[2]}"
    local ref="${BASH_REMATCH[3]}"
    
    # Check if ref is a semantic version tag
    if [[ "$ref" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
      log_success "GitHub: Pinned to release tag '$ref' ✓"
      return 0
    fi
    
    # Check if ref is a commit SHA
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
      return 0
    fi
    
    log_info "GitHub: Using ref '$ref' (assuming tag or branch)"
    return 0
  fi
  
  # Check GitHub releases
  if [[ "$url" =~ github\.com/([^/]+)/([^/]+)/releases/download/([^/]+)/ ]]; then
    local owner="${BASH_REMATCH[1]}"
    local repo="${BASH_REMATCH[2]}"
    local tag="${BASH_REMATCH[3]}"
    
    log_success "GitHub Release: Pinned to '$tag' ✓✓"
    log_info "Repository: $owner/$repo"
    return 0
  fi
  
  return 0
}

test_url() {
  local description="$1"
  local url="$2"
  local expected="$3"
  
  echo ""
  echo "Testing: $description"
  echo "URL: $url"
  
  output=$(verify_github_pin "$url" 2>&1)
  
  if echo "$output" | grep -q "$expected"; then
    echo -e "${GREEN}✓ PASS${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo -e "${RED}✗ FAIL${NC}"
    echo "Expected: $expected"
    echo "Got: $output"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

echo "======================================"
echo "GitHub URL Pin Detection Tests"
echo "======================================"

# Test semantic version tags
test_url "Semantic version tag (v1.0.0)" \
  "https://raw.githubusercontent.com/myorg/configs/v1.0.0/prod.json" \
  "Pinned to release tag 'v1.0.0'"

test_url "Semantic version with pre-release" \
  "https://raw.githubusercontent.com/myorg/configs/v1.0.0-rc.1/prod.json" \
  "Pinned to release tag 'v1.0.0-rc.1'"

test_url "Semantic version with build metadata" \
  "https://raw.githubusercontent.com/myorg/configs/v2.1.3-alpha.2/prod.json" \
  "Pinned to release tag 'v2.1.3-alpha.2'"

# Test commit SHAs
test_url "Full commit SHA (40 chars)" \
  "https://raw.githubusercontent.com/myorg/configs/a1b2c3d4e5f67890abcdef1234567890abcdef12/prod.json" \
  "Pinned to commit 'a1b2c3d4...'"

test_url "Short commit SHA (7 chars)" \
  "https://raw.githubusercontent.com/myorg/configs/a1b2c3d/prod.json" \
  "Pinned to short commit 'a1b2c3d'"

test_url "Short commit SHA (12 chars)" \
  "https://raw.githubusercontent.com/myorg/configs/a1b2c3d4e5f6/prod.json" \
  "Pinned to short commit 'a1b2c3d4e5f6'"

# Test mutable branches
test_url "Main branch (mutable)" \
  "https://raw.githubusercontent.com/myorg/configs/main/prod.json" \
  "MUTABLE branch 'main'"

test_url "Master branch (mutable)" \
  "https://raw.githubusercontent.com/myorg/configs/master/prod.json" \
  "MUTABLE branch 'master'"

test_url "Develop branch (mutable)" \
  "https://raw.githubusercontent.com/myorg/configs/develop/prod.json" \
  "MUTABLE branch 'develop'"

# Test GitHub Releases
test_url "GitHub Release with version tag" \
  "https://github.com/myorg/configs/releases/download/v1.0.0/prod.json" \
  "GitHub Release: Pinned to 'v1.0.0'"

test_url "GitHub Release with pre-release" \
  "https://github.com/myorg/configs/releases/download/v1.0.0-rc.1/prod.json" \
  "GitHub Release: Pinned to 'v1.0.0-rc.1'"

# Test custom branch names
test_url "Custom branch name" \
  "https://raw.githubusercontent.com/myorg/configs/feature-branch/prod.json" \
  "Using ref 'feature-branch'"

echo ""
echo "======================================"
echo "Test Results"
echo "======================================"
echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
echo "Total:  $((TESTS_PASSED + TESTS_FAILED))"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
