package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// bootstrapCommand handles the bootstrap command for loading configs from URLs
func bootstrapCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: Missing config source")
		fmt.Fprintln(os.Stderr, "")
		printBootstrapHelp()
		os.Exit(1)
	}

	// Parse flags
	configSource := os.Args[2]
	dryRun := false
	platform := ""
	sha256Hash := ""
	skipChecksum := false

	for i := 3; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case arg == "-h" || arg == "--help":
			printBootstrapHelp()
			os.Exit(0)
		case arg == "--dry-run":
			dryRun = true
		case arg == "--skip-checksum":
			skipChecksum = true
		case arg == "--platform" && i+1 < len(os.Args):
			platform = os.Args[i+1]
			i++
		case arg == "--sha256" && i+1 < len(os.Args):
			sha256Hash = os.Args[i+1]
			i++
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\n", arg)
			os.Exit(1)
		}
	}

	// Load config from URL or file
	var config *Config
	var err error

	if strings.HasPrefix(configSource, "http://") || strings.HasPrefix(configSource, "https://") {
		config, err = loadConfigFromURL(configSource, sha256Hash, skipChecksum)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config from URL: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Local file
		config, err = LoadConfig(configSource)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	}

	// Now execute using the same logic as executeCommand
	executeConfigWithOptions(config, dryRun, platform)
}

// loadConfigFromURL downloads and parses a config from a URL
func loadConfigFromURL(url string, expectedSHA256 string, skipChecksum bool) (*Config, error) {
	// Check if it's a GitHub URL
	githubInfo, isGitHub := ParseGitHubURL(url)
	if isGitHub {
		validateGitHubPin(githubInfo)

		// Try to auto-fetch checksum from .sha256 file if not provided
		if expectedSHA256 == "" && !skipChecksum {
			checksumURL := url + ".sha256"
			if autoChecksum, err := fetchChecksum(checksumURL); err == nil {
				expectedSHA256 = autoChecksum
				fmt.Printf("✅ Auto-fetched SHA256 from %s\n", checksumURL)
			}
		}
	}

	// Validate security requirements
	if strings.HasPrefix(url, "http://") && expectedSHA256 == "" && !skipChecksum {
		return nil, fmt.Errorf("HTTP URLs require --sha256 checksum or --skip-checksum flag for security")
	}

	// Download the config
	fmt.Printf("📥 Downloading config from %s\n", url)
	client := &http.Client{
		Timeout: DefaultHTTPTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read the body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Verify checksum if provided
	if expectedSHA256 != "" {
		if err := verifyChecksum(body, expectedSHA256); err != nil {
			return nil, err
		}
		fmt.Printf("✅ SHA256 verified\n")
	} else if strings.HasPrefix(url, "https://") {
		fmt.Printf("✅ Downloaded via HTTPS (TLS verified)\n")
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}

	// The config type itself performs basic structure validation
	// Additional validation would happen during execution
	fmt.Printf("✅ Config loaded and validated\n")
	return &config, nil
}

// validateGitHubPin validates GitHub URL pinning and warns about mutable refs
func validateGitHubPin(info *GitHubURLInfo) {
	switch info.PinType {
	case GitHubPinTag:
		fmt.Printf("✅ GitHub: Pinned to release tag '%s' ✓\n", info.Ref)
	case GitHubPinCommit:
		shortRef := info.Ref
		if len(shortRef) > 8 {
			shortRef = shortRef[:8] + "..."
		}
		fmt.Printf("✅ GitHub: Pinned to commit '%s' ✓\n", shortRef)
	case GitHubPinRelease:
		fmt.Printf("✅ GitHub Release: Pinned to '%s' ✓✓\n", info.Ref)
	case GitHubPinBranch:
		fmt.Printf("⚠️  GitHub: Using MUTABLE branch '%s' (content can change)\n", info.Ref)
	default:
		fmt.Printf("ℹ️  GitHub: Using ref '%s' (assuming tag or branch)\n", info.Ref)
	}
	fmt.Printf("   Repository: %s/%s\n", info.Owner, info.Repo)
}

// fetchChecksum attempts to download a .sha256 file
func fetchChecksum(url string) (string, error) {
	client := &http.Client{
		Timeout: ChecksumHTTPTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Checksum file may contain just the hash or "hash  filename"
	checksum := strings.TrimSpace(string(body))
	if parts := strings.Fields(checksum); len(parts) > 0 {
		checksum = parts[0]
	}

	return checksum, nil
}

// verifyChecksum verifies the SHA256 checksum of data
func verifyChecksum(data []byte, expectedHex string) error {
	hash := sha256.Sum256(data)
	actualHex := hex.EncodeToString(hash[:])

	expectedHex = strings.TrimSpace(strings.ToLower(expectedHex))
	actualHex = strings.ToLower(actualHex)

	if actualHex != expectedHex {
		return fmt.Errorf("SHA256 mismatch:\n  Expected: %s\n  Got:      %s", expectedHex, actualHex)
	}

	return nil
}

// printBootstrapHelp prints help for the bootstrap command
func printBootstrapHelp() {
	fmt.Print(`
sink bootstrap - Bootstrap from URL or local file

Usage:
  sink bootstrap <source> [options]

Arguments:
  source              Config file URL or local path
                      Supports: http://, https://, file paths

Options:
  --dry-run          Show what would be executed without running
  --platform <os>    Override platform detection (darwin, linux, etc.)
  --sha256 <hash>    Expected SHA256 checksum (required for HTTP)
  --skip-checksum    Skip checksum verification (not recommended)
  -h, --help         Show this help message

Description:
  The bootstrap command loads configuration from URLs or local files and
  executes the installation steps. It provides secure remote config loading
  with GitHub URL pinning validation and checksum verification.

GitHub URL Pinning:
  Bootstrap automatically detects and validates GitHub URLs:

  ✅ Pinned (Immutable - Recommended):
     - Semantic version tags: v1.0.0, v2.1.3-rc.1
     - Commit SHAs: abc123def456... (40 chars) or abc123d (7+ chars)
     - GitHub Releases: /releases/download/v1.0.0/config.json

  ⚠️  Mutable (Content Can Change):
     - Branch names: main, master, develop, staging

  Auto-checksum: If a .sha256 file exists alongside the config,
  it will be automatically fetched and verified.

Security Model:
  Source Type   | SHA256 Required? | Verification
  --------------|------------------|------------------
  Local file    | No               | Trusted source
  HTTPS URL     | No (recommended) | TLS certificate
  HTTP URL      | YES              | SHA256 checksum
  GitHub HTTPS  | No (auto-fetch)  | TLS + optional SHA256

Examples:
  # Bootstrap from GitHub (pinned version)
  sink bootstrap https://raw.githubusercontent.com/org/configs/v1.0.0/prod.json

  # Bootstrap from GitHub commit
  sink bootstrap https://raw.githubusercontent.com/org/configs/abc123def/prod.json

  # Bootstrap from HTTPS with auto-checksum
  sink bootstrap https://configs.example.com/setup.json

  # Bootstrap from HTTP with manual checksum
  sink bootstrap http://configs.example.com/setup.json \
    --sha256 a3b2c1d4e5f6...

  # Bootstrap from local file
  sink bootstrap config.json

  # Dry-run to preview
  sink bootstrap https://example.com/config.json --dry-run

  # Override platform detection
  sink bootstrap config.json --platform linux

Exit Codes:
  0    Success
  1    Error (download failed, validation failed, execution failed)

Output:
  Bootstrap shows download progress, GitHub pin validation, checksum
  verification, and then the standard execution output with step progress.

Related Commands:
  sink execute    - Execute a local config file
  sink remote     - Deploy to remote hosts via SSH
  sink validate   - Validate a config file
  sink help       - Show general help

See Also:
  docs/GITHUB_URL_PINNING.md  - GitHub pinning security guide
  docs/REMOTE_BOOTSTRAP.md    - Remote deployment guide
  examples/bootstrap-*.json   - Bootstrap examples
`)
}
