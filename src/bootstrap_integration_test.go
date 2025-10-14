package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBootstrapIntegration tests end-to-end bootstrap functionality
func TestBootstrapIntegration(t *testing.T) {
	t.Run("LocalFile", testBootstrapLocalFile)
	t.Run("HTTPSWithChecksum", testBootstrapHTTPSWithChecksum)
	t.Run("HTTPSWithAutoChecksum", testBootstrapHTTPSWithAutoChecksum)
	t.Run("HTTPRequiresChecksum", testBootstrapHTTPRequiresChecksum)
	t.Run("DryRunMode", testBootstrapDryRun)
	t.Run("InvalidConfig", testBootstrapInvalidConfig)
	t.Run("PlatformOverride", testBootstrapPlatformOverride)
	t.Run("GitHubPinnedVersion", testBootstrapGitHubPinned)
	t.Run("GitHubCommitSHA", testBootstrapGitHubCommit)
	t.Run("ChecksumMismatch", testBootstrapChecksumMismatch)
}

// testBootstrapLocalFile tests bootstrapping from a local file
func testBootstrapLocalFile(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")

	config := `{
		"version": "1.0.0",
		"facts": {},
		"platforms": [
			{
				"os": "darwin",
				"match": "Darwin",
				"name": "macOS Test",
				"install_steps": [
					{
						"name": "Test step",
						"command": "echo 'test output'",
						"check": "true"
					}
				]
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(config), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test loading config from local file
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load local config: %v", err)
	}

	if loadedConfig.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", loadedConfig.Version)
	}

	if len(loadedConfig.Platforms) != 1 {
		t.Errorf("Expected 1 platform, got %d", len(loadedConfig.Platforms))
	}
}

// testBootstrapHTTPSWithChecksum tests HTTP download with manual checksum verification
// Note: HTTPS with proper TLS validation is tested separately in bootstrap_test.go
func testBootstrapHTTPSWithChecksum(t *testing.T) {
	// Create test config
	config := `{
		"version": "1.0.0",
		"facts": {},
		"platforms": [
			{
				"os": "linux",
				"match": "Linux",
				"name": "Linux Test",
				"install_steps": [
					{
						"name": "Test step",
						"command": "echo 'http test with checksum'",
						"check": "true"
					}
				]
			}
		]
	}`

	// Calculate expected SHA256
	expectedSHA256 := calculateSHA256([]byte(config))

	// Create test server (using HTTP for integration testing)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(config))
	}))
	defer server.Close()

	// Test loading with correct checksum
	loadedConfig, err := loadConfigFromURL(server.URL, expectedSHA256, false)
	if err != nil {
		t.Fatalf("Failed to load config with checksum: %v", err)
	}

	if loadedConfig.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", loadedConfig.Version)
	}
}

// testBootstrapHTTPSWithAutoChecksum tests HTTP with automatic .sha256 file fetching
// Note: This tests the auto-checksum functionality without TLS complexity
func testBootstrapHTTPSWithAutoChecksum(t *testing.T) {
	config := `{
		"version": "1.0.0",
		"facts": {},
		"platforms": [
			{
				"os": "linux",
				"match": "Linux",
				"name": "Linux Checksum Test",
				"install_steps": [
					{
						"name": "Test step",
						"command": "echo 'auto checksum test'",
						"check": "true"
					}
				]
			}
		]
	}`

	expectedSHA256 := calculateSHA256([]byte(config))

	// Create test server that serves both config and .sha256 file
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sha256") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedSHA256))
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(config))
		}
	}))
	defer server.Close()

	// Test auto-checksum fetching (must append .sha256 to the path, not the full URL)
	checksumURL := server.URL + "/.sha256" // Add .sha256 as a path component
	checksum, err := fetchChecksum(checksumURL)
	if err != nil {
		t.Fatalf("Failed to fetch auto-checksum: %v", err)
	}

	if checksum != expectedSHA256 {
		t.Errorf("Expected checksum %s, got %s", expectedSHA256, checksum)
	}

	// Test loading with explicit checksum (HTTP requires explicit checksum)
	loadedConfig, err := loadConfigFromURL(server.URL, expectedSHA256, false)
	if err != nil {
		t.Fatalf("Failed to load config with checksum: %v", err)
	}

	if loadedConfig.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", loadedConfig.Version)
	}
}

// testBootstrapHTTPRequiresChecksum tests that HTTP URLs require checksums
func testBootstrapHTTPRequiresChecksum(t *testing.T) {
	config := `{
		"version": "1.0.0",
		"facts": {},
		"platforms": [
			{
				"os": "linux",
				"match": "Linux",
				"name": "HTTP Test",
				"install_steps": [
					{
						"name": "Test step",
						"command": "true",
						"check": "true"
					}
				]
			}
		]
	}`

	// Create HTTP (not HTTPS) server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(config))
	}))
	defer server.Close()

	// Test that HTTP without checksum fails
	_, err := loadConfigFromURL(server.URL, "", false)
	if err == nil {
		t.Error("Expected error for HTTP URL without checksum")
	}

	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("Expected checksum-related error, got: %v", err)
	}

	// Test that HTTP with correct checksum succeeds
	expectedSHA256 := calculateSHA256([]byte(config))
	loadedConfig, err := loadConfigFromURL(server.URL, expectedSHA256, false)
	if err != nil {
		t.Fatalf("Failed to load HTTP config with checksum: %v", err)
	}

	if loadedConfig.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", loadedConfig.Version)
	}
}

// testBootstrapDryRun tests dry-run mode functionality
func testBootstrapDryRun(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "dry-run-config.json")

	config := `{
		"version": "1.0.0",
		"facts": {
			"test_fact": {
				"command": "echo 'fact_value'"
			}
		},
		"platforms": [
			{
				"os": "darwin",
				"match": "Darwin",
				"name": "macOS Dry Run Test",
				"install_steps": [
					{
						"name": "Dry run test step",
						"command": "echo 'This should not execute'",
						"check": "false"
					}
				]
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(config), 0644)
	if err != nil {
		t.Fatalf("Failed to create dry-run config: %v", err)
	}

	// Load config and test dry-run execution
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create executor in dry-run mode
	transport := NewLocalTransport()
	executor := NewExecutor(transport)
	executor.DryRun = true

	// Gather facts (should work in dry-run)
	gatherer := NewFactGatherer(loadedConfig.Facts, transport)
	facts, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather facts in dry-run: %v", err)
	}

	if len(facts) == 0 {
		t.Error("Expected facts to be gathered in dry-run mode")
	}

	// Execute platform (should not actually run commands)
	platform := loadedConfig.Platforms[0]
	results := executor.ExecutePlatform(platform, facts)

	if len(results) != len(platform.InstallSteps) {
		t.Errorf("Expected %d results, got %d", len(platform.InstallSteps), len(results))
	}

	for _, result := range results {
		if result.Error != "" {
			t.Errorf("Dry-run should not produce errors, got: %s", result.Error)
		}
	}
}

// testBootstrapInvalidConfig tests handling of invalid configurations
func testBootstrapInvalidConfig(t *testing.T) {
	invalidConfigs := []struct {
		name   string
		config string
		error  string
	}{
		{
			name:   "Invalid JSON",
			config: `{"version": "1.0.0", "invalid": json}`,
			error:  "invalid character",
		},
		{
			name:   "Missing required fields",
			config: `{"version": "1.0.0", "facts": {}, "platforms": []}`,
			error:  "at least one platform is required",
		},
		{
			name:   "Invalid platform",
			config: `{"version": "1.0.0", "facts": {}, "platforms": [{"name": "test"}]}`,
			error:  "os",
		},
	}

	for _, tc := range invalidConfigs {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server with invalid config
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tc.config))
			}))
			defer server.Close()

			// Test that loading fails appropriately
			_, err := loadConfigFromURL(server.URL, "", true) // Skip checksum for invalid JSON
			if err == nil {
				t.Error("Expected error for invalid config")
			}

			if !strings.Contains(err.Error(), tc.error) {
				t.Errorf("Expected error containing '%s', got: %v", tc.error, err)
			}
		})
	}
}

// testBootstrapPlatformOverride tests platform override functionality
func testBootstrapPlatformOverride(t *testing.T) {
	config := `{
		"version": "1.0.0",
		"facts": {},
		"platforms": [
			{
				"os": "linux",
				"match": "Linux",
				"name": "Linux Override Test",
				"install_steps": [
					{
						"name": "Linux-specific step",
						"command": "echo 'linux command'",
						"check": "true"
					}
				]
			},
			{
				"os": "darwin",
				"match": "Darwin",
				"name": "macOS Override Test",
				"install_steps": [
					{
						"name": "macOS-specific step",
						"command": "echo 'macos command'",
						"check": "true"
					}
				]
			}
		]
	}`

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "platform-override-config.json")

	err := os.WriteFile(configPath, []byte(config), 0644)
	if err != nil {
		t.Fatalf("Failed to create platform override config: %v", err)
	}

	// Load config
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test platform selection with override
	targetOS := "linux" // Override to Linux regardless of actual OS
	var selectedPlatform *Platform
	for i := range loadedConfig.Platforms {
		if loadedConfig.Platforms[i].OS == targetOS {
			selectedPlatform = &loadedConfig.Platforms[i]
			break
		}
	}

	if selectedPlatform == nil {
		t.Fatalf("Failed to find platform for override OS: %s", targetOS)
	}

	if selectedPlatform.OS != "linux" {
		t.Errorf("Expected linux platform, got %s", selectedPlatform.OS)
	}

	if selectedPlatform.Name != "Linux Override Test" {
		t.Errorf("Expected 'Linux Override Test', got %s", selectedPlatform.Name)
	}
}

// testBootstrapGitHubPinned tests GitHub URL pinning with version tags
func testBootstrapGitHubPinned(t *testing.T) {
	testCases := []struct {
		name        string
		url         string
		expectPin   bool
		expectType  GitHubPinType
		expectError bool
	}{
		{
			name:       "Semantic version tag",
			url:        "https://raw.githubusercontent.com/myorg/configs/v1.0.0/config.json",
			expectPin:  true,
			expectType: GitHubPinTag,
		},
		{
			name:       "Release candidate tag",
			url:        "https://raw.githubusercontent.com/myorg/configs/v2.1.0-rc.1/config.json",
			expectPin:  true,
			expectType: GitHubPinTag,
		},
		{
			name:       "Branch reference (mutable)",
			url:        "https://raw.githubusercontent.com/myorg/configs/main/config.json",
			expectPin:  false,
			expectType: GitHubPinBranch,
		},
		{
			name:       "Commit SHA",
			url:        "https://raw.githubusercontent.com/myorg/configs/a1b2c3d4e5f67890abcdef1234567890abcdef12/config.json",
			expectPin:  true,
			expectType: GitHubPinCommit,
		},
		{
			name:        "Invalid GitHub URL",
			url:         "https://raw.githubusercontent.com/invalid",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test GitHub pinning
			info, ok := ParseGitHubURL(tc.url)

			if tc.expectError {
				if ok {
					t.Error("Expected parsing to fail for invalid URL")
				}
				return
			}

			if !ok {
				t.Fatalf("Failed to parse GitHub URL: %s", tc.url)
			}

			if info.IsPinned != tc.expectPin {
				t.Errorf("Expected pinned=%v, got %v", tc.expectPin, info.IsPinned)
			}

			if info.PinType != tc.expectType {
				t.Errorf("Expected pin type %v, got %v", tc.expectType, info.PinType)
			}

			// Test validation output (should not panic)
			validateGitHubPin(info)
		})
	}
}

// testBootstrapGitHubCommit tests GitHub commit SHA pinning
func testBootstrapGitHubCommit(t *testing.T) {
	commitURLs := []struct {
		url       string
		expectSHA string
	}{
		{
			url:       "https://raw.githubusercontent.com/org/repo/a1b2c3d4e5f67890abcdef1234567890abcdef12/config.json",
			expectSHA: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
		},
		{
			url:       "https://raw.githubusercontent.com/org/repo/abc123d/config.json",
			expectSHA: "abc123d",
		},
	}

	for _, tc := range commitURLs {
		t.Run(tc.expectSHA, func(t *testing.T) {
			info, ok := ParseGitHubURL(tc.url)
			if !ok {
				t.Fatalf("Failed to parse GitHub commit URL: %s", tc.url)
			}

			if info.PinType != GitHubPinCommit {
				t.Errorf("Expected commit pin type, got %v", info.PinType)
			}

			if info.Ref != tc.expectSHA {
				t.Errorf("Expected SHA %s, got %s", tc.expectSHA, info.Ref)
			}

			if !info.IsPinned {
				t.Error("Commit SHAs should be considered pinned")
			}

			if info.IsMutable {
				t.Error("Commit SHAs should be immutable")
			}
		})
	}
}

// testBootstrapChecksumMismatch tests checksum verification failure
func testBootstrapChecksumMismatch(t *testing.T) {
	config := `{"version": "1.0.0", "facts": {}, "platforms": []}`
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(config))
	}))
	defer server.Close()

	// Test that wrong checksum fails
	_, err := loadConfigFromURL(server.URL, wrongChecksum, false)
	if err == nil {
		t.Error("Expected error for checksum mismatch")
	}

	if !strings.Contains(err.Error(), "checksum") && !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("Expected checksum mismatch error, got: %v", err)
	}
}

// Helper function to calculate SHA256 checksum
func calculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// Benchmark tests for performance validation
func BenchmarkBootstrapLocalFile(b *testing.B) {
	// Create temporary config file
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "bench-config.json")

	config := `{
		"version": "1.0.0",
		"facts": {"test": "echo test"},
		"platforms": [
			{
				"os": "darwin",
				"name": "Benchmark Test",
				"install_steps": [
					{"name": "Step 1", "command": "true", "check": "true"},
					{"name": "Step 2", "command": "true", "check": "true"},
					{"name": "Step 3", "command": "true", "check": "true"}
				]
			}
		]
	}`

	err := os.WriteFile(configPath, []byte(config), 0644)
	if err != nil {
		b.Fatalf("Failed to create benchmark config: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configPath)
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}

func BenchmarkBootstrapHTTPS(b *testing.B) {
	config := `{
		"version": "1.0.0",
		"facts": {},
		"platforms": [
			{
				"os": "linux",
				"match": "Linux",
				"name": "HTTPS Benchmark",
				"install_steps": [
					{"name": "Test", "command": "true", "check": "true"}
				]
			}
		]
	}`

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(config))
	}))
	defer server.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := loadConfigFromURL(server.URL, "", true) // Skip checksum for benchmark
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}

// Integration test with real execution (commented out by default to avoid side effects)
/*
func TestBootstrapRealExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real execution test in short mode")
	}

	// Create a safe test config that only echoes
	config := `{
		"version": "1.0.0",
		"facts": {
			"current_user": "whoami",
			"current_dir": "pwd"
		},
		"platforms": [
			{
				"os": "darwin",
				"name": "Real Execution Test",
				"install_steps": [
					{
						"name": "Safe echo test",
						"command": "echo 'Integration test successful'",
						"check": "true"
					}
				]
			}
		]
	}`

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "real-execution-config.json")

	err := os.WriteFile(configPath, []byte(config), 0644)
	if err != nil {
		t.Fatalf("Failed to create real execution config: %v", err)
	}

	// Load and execute config (with platform override to ensure consistent testing)
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Execute with dry-run disabled for real execution test
	transport := NewLocalTransport()
	executor := NewExecutor(transport)
	executor.DryRun = false

	gatherer := NewFactGatherer(loadedConfig.Facts, transport)
	facts, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather facts: %v", err)
	}

	platform := loadedConfig.Platforms[0]
	results := executor.ExecutePlatform(platform, facts)

	for _, result := range results {
		if result.Error != "" {
			t.Errorf("Real execution failed: %s", result.Error)
		}
	}
}
*/

// Test timeout handling
func TestBootstrapTimeout(t *testing.T) {
	// Create test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Short delay for testing
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"version": "1.0.0",
			"facts": {},
			"platforms": [{
				"name": "Test",
				"os": "linux",
				"match": "Linux",
				"install_steps": [{"name": "test", "command": "true", "check": "true"}]
			}]
		}`))
	}))
	defer server.Close()

	// Note: This test assumes the HTTP client has a reasonable timeout
	// In a real implementation, you'd want to configure timeouts
	_, err := loadConfigFromURL(server.URL, "", true)
	if err != nil {
		// Timeout errors are acceptable here
		if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Errorf("Expected timeout-related error or success, got: %v", err)
		}
	}
}
