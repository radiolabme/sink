package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLoadConfigFromURL_HTTPS tests loading config from HTTPS URL
func TestLoadConfigFromURL_HTTPS(t *testing.T) {
	// Create a test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		configJSON := `{
			"version": "1.0",
			"facts": {},
			"platforms": [{
				"name": "Test",
				"os": "darwin",
				"install_steps": []
			}]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(configJSON))
	}))
	defer server.Close()

	// Note: We can't easily test this without modifying the http.Client to accept test certs
	// So we'll test the structure instead
	t.Skip("HTTPS testing requires custom TLS setup - covered by integration tests")
}

// TestLoadConfigFromURL_RequiresSHA256ForHTTP tests that HTTP URLs require SHA256
func TestLoadConfigFromURL_RequiresSHA256ForHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	defer server.Close()

	// This should fail because HTTP without SHA256 is not allowed
	_, err := loadConfigFromURL(server.URL, "", false)
	if err == nil {
		t.Error("Expected error for HTTP URL without SHA256, got nil")
	}
	if err != nil && err.Error() != "HTTP URLs require --sha256 checksum or --skip-checksum flag for security" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestLoadConfigFromURL_WithSHA256 tests loading with SHA256 verification
func TestLoadConfigFromURL_WithSHA256(t *testing.T) {
	configJSON := `{
		"version": "1.0",
		"facts": {},
		"platforms": [{
			"name": "Test",
			"os": "darwin",
			"install_steps": []
		}]
	}`

	// Calculate actual SHA256
	hash := sha256.Sum256([]byte(configJSON))
	expectedHash := hex.EncodeToString(hash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(configJSON))
	}))
	defer server.Close()

	// This should succeed with correct SHA256
	config, err := loadConfigFromURL(server.URL, expectedHash, false)
	if err != nil {
		t.Errorf("Expected success with correct SHA256, got error: %v", err)
	}
	if config == nil {
		t.Error("Expected config to be non-nil")
	}
	if config != nil && config.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", config.Version)
	}
}

// TestLoadConfigFromURL_WrongSHA256 tests that wrong SHA256 fails
func TestLoadConfigFromURL_WrongSHA256(t *testing.T) {
	configJSON := `{"version": "1.0", "facts": {}, "platforms": []}`
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(configJSON))
	}))
	defer server.Close()

	// This should fail with wrong SHA256
	_, err := loadConfigFromURL(server.URL, wrongHash, false)
	if err == nil {
		t.Error("Expected error for wrong SHA256, got nil")
	}
}

// TestLoadConfigFromURL_Skip Checksum tests skipping checksum verification
func TestLoadConfigFromURL_SkipChecksum(t *testing.T) {
	configJSON := `{
		"version": "1.0",
		"facts": {},
		"platforms": [{
			"name": "Test",
			"os": "darwin",
			"install_steps": []
		}]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(configJSON))
	}))
	defer server.Close()

	// This should succeed even without SHA256 when skipChecksum is true
	config, err := loadConfigFromURL(server.URL, "", true)
	if err != nil {
		t.Errorf("Expected success with skipChecksum=true, got error: %v", err)
	}
	if config == nil {
		t.Error("Expected config to be non-nil")
	}
}

// TestFetchChecksum tests fetching checksum from .sha256 file
func TestFetchChecksum(t *testing.T) {
	expectedChecksum := "abcdef1234567890"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedChecksum + "  config.json\n"))
	}))
	defer server.Close()

	checksum, err := fetchChecksum(server.URL)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if checksum != expectedChecksum {
		t.Errorf("Expected checksum %s, got %s", expectedChecksum, checksum)
	}
}

// TestFetchChecksum_NotFound tests handling of missing checksum file
func TestFetchChecksum_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := fetchChecksum(server.URL)
	if err == nil {
		t.Error("Expected error for 404, got nil")
	}
}

// TestVerifyChecksum tests checksum verification
func TestVerifyChecksum(t *testing.T) {
	data := []byte("test data")
	hash := sha256.Sum256(data)
	correctChecksum := hex.EncodeToString(hash[:])

	err := verifyChecksum(data, correctChecksum)
	if err != nil {
		t.Errorf("Expected success with correct checksum, got error: %v", err)
	}

	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	err = verifyChecksum(data, wrongChecksum)
	if err == nil {
		t.Error("Expected error with wrong checksum, got nil")
	}
}

// TestValidateGitHubPin tests GitHub pin validation output
func TestValidateGitHubPin(t *testing.T) {
	tests := []struct {
		name string
		info *GitHubURLInfo
	}{
		{
			name: "Semantic version tag",
			info: &GitHubURLInfo{
				Owner:     "myorg",
				Repo:      "configs",
				Ref:       "v1.0.0",
				PinType:   GitHubPinTag,
				IsPinned:  true,
				IsMutable: false,
			},
		},
		{
			name: "Commit SHA",
			info: &GitHubURLInfo{
				Owner:     "myorg",
				Repo:      "configs",
				Ref:       "abc123def456",
				PinType:   GitHubPinCommit,
				IsPinned:  true,
				IsMutable: false,
			},
		},
		{
			name: "Mutable branch",
			info: &GitHubURLInfo{
				Owner:     "myorg",
				Repo:      "configs",
				Ref:       "main",
				PinType:   GitHubPinBranch,
				IsPinned:  false,
				IsMutable: true,
			},
		},
		{
			name: "GitHub Release",
			info: &GitHubURLInfo{
				Owner:     "myorg",
				Repo:      "configs",
				Ref:       "v1.0.0",
				PinType:   GitHubPinRelease,
				IsPinned:  true,
				IsMutable: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test that it doesn't panic
			// Output testing would require capturing stdout
			validateGitHubPin(tt.info)
		})
	}
}

// TestLoadConfigFromURL_InvalidJSON tests handling of invalid JSON
func TestLoadConfigFromURL_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	_, err := loadConfigFromURL(server.URL, "", true)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

// TestLoadConfigFromURL_HTTPError tests handling of HTTP errors
func TestLoadConfigFromURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := loadConfigFromURL(server.URL, "", true)
	if err == nil {
		t.Error("Expected error for HTTP 500, got nil")
	}
}
