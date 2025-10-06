package main

import (
	"strings"
	"testing"
)

// Test semantic version tags
func TestGitHubPinDetection_SemanticVersionTag(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedRef string
		expectedPin GitHubPinType
		isPinned    bool
		isMutable   bool
	}{
		{
			name:        "Simple semantic version v1.0.0",
			url:         "https://raw.githubusercontent.com/myorg/configs/v1.0.0/prod.json",
			expectedRef: "v1.0.0",
			expectedPin: GitHubPinTag,
			isPinned:    true,
			isMutable:   false,
		},
		{
			name:        "Semantic version with pre-release",
			url:         "https://raw.githubusercontent.com/myorg/configs/v1.0.0-rc.1/prod.json",
			expectedRef: "v1.0.0-rc.1",
			expectedPin: GitHubPinTag,
			isPinned:    true,
			isMutable:   false,
		},
		{
			name:        "Semantic version with alpha",
			url:         "https://raw.githubusercontent.com/myorg/configs/v2.1.3-alpha.2/prod.json",
			expectedRef: "v2.1.3-alpha.2",
			expectedPin: GitHubPinTag,
			isPinned:    true,
			isMutable:   false,
		},
		{
			name:        "Semantic version with beta",
			url:         "https://raw.githubusercontent.com/myorg/configs/v3.0.0-beta.1/prod.json",
			expectedRef: "v3.0.0-beta.1",
			expectedPin: GitHubPinTag,
			isPinned:    true,
			isMutable:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := ParseGitHubURL(tt.url)
			if !ok {
				t.Fatalf("Failed to parse GitHub URL: %s", tt.url)
			}

			if info.Ref != tt.expectedRef {
				t.Errorf("Expected ref %q, got %q", tt.expectedRef, info.Ref)
			}

			if info.PinType != tt.expectedPin {
				t.Errorf("Expected pin type %v, got %v", tt.expectedPin, info.PinType)
			}

			if info.IsPinned != tt.isPinned {
				t.Errorf("Expected isPinned %v, got %v", tt.isPinned, info.IsPinned)
			}

			if info.IsMutable != tt.isMutable {
				t.Errorf("Expected isMutable %v, got %v", tt.isMutable, info.IsMutable)
			}
		})
	}
}

// Test commit SHAs
func TestGitHubPinDetection_CommitSHA(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedRef string
		expectedPin GitHubPinType
		isPinned    bool
	}{
		{
			name:        "Full commit SHA (40 chars)",
			url:         "https://raw.githubusercontent.com/myorg/configs/a1b2c3d4e5f67890abcdef1234567890abcdef12/prod.json",
			expectedRef: "a1b2c3d4e5f67890abcdef1234567890abcdef12",
			expectedPin: GitHubPinCommit,
			isPinned:    true,
		},
		{
			name:        "Short commit SHA (7 chars)",
			url:         "https://raw.githubusercontent.com/myorg/configs/a1b2c3d/prod.json",
			expectedRef: "a1b2c3d",
			expectedPin: GitHubPinCommit,
			isPinned:    true,
		},
		{
			name:        "Short commit SHA (12 chars)",
			url:         "https://raw.githubusercontent.com/myorg/configs/a1b2c3d4e5f6/prod.json",
			expectedRef: "a1b2c3d4e5f6",
			expectedPin: GitHubPinCommit,
			isPinned:    true,
		},
		{
			name:        "Short commit SHA (8 chars)",
			url:         "https://raw.githubusercontent.com/myorg/configs/abcdef01/prod.json",
			expectedRef: "abcdef01",
			expectedPin: GitHubPinCommit,
			isPinned:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := ParseGitHubURL(tt.url)
			if !ok {
				t.Fatalf("Failed to parse GitHub URL: %s", tt.url)
			}

			if info.Ref != tt.expectedRef {
				t.Errorf("Expected ref %q, got %q", tt.expectedRef, info.Ref)
			}

			if info.PinType != tt.expectedPin {
				t.Errorf("Expected pin type %v, got %v", tt.expectedPin, info.PinType)
			}

			if info.IsPinned != tt.isPinned {
				t.Errorf("Expected isPinned %v, got %v", tt.isPinned, info.IsPinned)
			}

			if info.IsMutable {
				t.Errorf("Commit SHA should not be mutable")
			}
		})
	}
}

// Test mutable branches
func TestGitHubPinDetection_MutableBranches(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedRef string
		expectedPin GitHubPinType
		isMutable   bool
	}{
		{
			name:        "Main branch",
			url:         "https://raw.githubusercontent.com/myorg/configs/main/prod.json",
			expectedRef: "main",
			expectedPin: GitHubPinBranch,
			isMutable:   true,
		},
		{
			name:        "Master branch",
			url:         "https://raw.githubusercontent.com/myorg/configs/master/prod.json",
			expectedRef: "master",
			expectedPin: GitHubPinBranch,
			isMutable:   true,
		},
		{
			name:        "Develop branch",
			url:         "https://raw.githubusercontent.com/myorg/configs/develop/prod.json",
			expectedRef: "develop",
			expectedPin: GitHubPinBranch,
			isMutable:   true,
		},
		{
			name:        "Dev branch",
			url:         "https://raw.githubusercontent.com/myorg/configs/dev/prod.json",
			expectedRef: "dev",
			expectedPin: GitHubPinBranch,
			isMutable:   true,
		},
		{
			name:        "Staging branch",
			url:         "https://raw.githubusercontent.com/myorg/configs/staging/prod.json",
			expectedRef: "staging",
			expectedPin: GitHubPinBranch,
			isMutable:   true,
		},
		{
			name:        "Production branch",
			url:         "https://raw.githubusercontent.com/myorg/configs/production/prod.json",
			expectedRef: "production",
			expectedPin: GitHubPinBranch,
			isMutable:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := ParseGitHubURL(tt.url)
			if !ok {
				t.Fatalf("Failed to parse GitHub URL: %s", tt.url)
			}

			if info.Ref != tt.expectedRef {
				t.Errorf("Expected ref %q, got %q", tt.expectedRef, info.Ref)
			}

			if info.PinType != tt.expectedPin {
				t.Errorf("Expected pin type %v, got %v", tt.expectedPin, info.PinType)
			}

			if info.IsPinned {
				t.Errorf("Mutable branch should not be considered pinned")
			}

			if !info.IsMutable {
				t.Errorf("Expected branch to be mutable")
			}
		})
	}
}

// Test GitHub Releases
func TestGitHubPinDetection_GitHubReleases(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedRef string
		isPinned    bool
	}{
		{
			name:        "GitHub Release with version tag",
			url:         "https://github.com/myorg/configs/releases/download/v1.0.0/prod.json",
			expectedRef: "v1.0.0",
			isPinned:    true,
		},
		{
			name:        "GitHub Release with pre-release",
			url:         "https://github.com/myorg/configs/releases/download/v1.0.0-rc.1/prod.json",
			expectedRef: "v1.0.0-rc.1",
			isPinned:    true,
		},
		{
			name:        "GitHub Release with beta",
			url:         "https://github.com/myorg/configs/releases/download/v2.0.0-beta.3/config.json",
			expectedRef: "v2.0.0-beta.3",
			isPinned:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := ParseGitHubURL(tt.url)
			if !ok {
				t.Fatalf("Failed to parse GitHub URL: %s", tt.url)
			}

			if info.PinType != GitHubPinRelease {
				t.Errorf("Expected pin type GitHubPinRelease, got %v", info.PinType)
			}

			if info.Ref != tt.expectedRef {
				t.Errorf("Expected ref %q, got %q", tt.expectedRef, info.Ref)
			}

			if !info.IsPinned {
				t.Errorf("GitHub Release should be considered pinned")
			}

			if info.IsMutable {
				t.Errorf("GitHub Release should not be mutable")
			}
		})
	}
}

// Test custom branch names
func TestGitHubPinDetection_CustomBranches(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedRef string
	}{
		{
			name:        "Feature branch",
			url:         "https://raw.githubusercontent.com/myorg/configs/feature-branch/prod.json",
			expectedRef: "feature-branch",
		},
		{
			name:        "Release branch",
			url:         "https://raw.githubusercontent.com/myorg/configs/release-1.x/prod.json",
			expectedRef: "release-1.x",
		},
		{
			name:        "Bugfix branch",
			url:         "https://raw.githubusercontent.com/myorg/configs/bugfix/issue-123/config.json",
			expectedRef: "bugfix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := ParseGitHubURL(tt.url)
			if !ok {
				t.Fatalf("Failed to parse GitHub URL: %s", tt.url)
			}

			if info.Ref != tt.expectedRef {
				t.Errorf("Expected ref %q, got %q", tt.expectedRef, info.Ref)
			}

			if info.PinType != GitHubPinUnknown {
				t.Errorf("Expected pin type GitHubPinUnknown for custom branch, got %v", info.PinType)
			}
		})
	}
}

// Test repository and owner extraction
func TestGitHubPinDetection_RepositoryInfo(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedOwner string
		expectedRepo  string
	}{
		{
			name:          "Standard org repo",
			url:           "https://raw.githubusercontent.com/myorg/configs/v1.0.0/prod.json",
			expectedOwner: "myorg",
			expectedRepo:  "configs",
		},
		{
			name:          "User repo",
			url:           "https://raw.githubusercontent.com/johndoe/my-configs/main/setup.json",
			expectedOwner: "johndoe",
			expectedRepo:  "my-configs",
		},
		{
			name:          "Repo with numbers",
			url:           "https://raw.githubusercontent.com/org123/config456/v2.0.0/test.json",
			expectedOwner: "org123",
			expectedRepo:  "config456",
		},
		{
			name:          "GitHub Release",
			url:           "https://github.com/radiolabme/sink/releases/download/v1.0.0/config.json",
			expectedOwner: "radiolabme",
			expectedRepo:  "sink",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := ParseGitHubURL(tt.url)
			if !ok {
				t.Fatalf("Failed to parse GitHub URL: %s", tt.url)
			}

			if info.Owner != tt.expectedOwner {
				t.Errorf("Expected owner %q, got %q", tt.expectedOwner, info.Owner)
			}

			if info.Repo != tt.expectedRepo {
				t.Errorf("Expected repo %q, got %q", tt.expectedRepo, info.Repo)
			}
		})
	}
}

// Test non-GitHub URLs
func TestGitHubPinDetection_NonGitHubURLs(t *testing.T) {
	tests := []string{
		"https://example.com/config.json",
		"https://configs.mycompany.com/prod.json",
		"http://localhost:8080/config.json",
		"file:///path/to/config.json",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			info, ok := ParseGitHubURL(url)
			if ok {
				t.Errorf("Non-GitHub URL should not be parsed as GitHub: %s (got %+v)", url, info)
			}
		})
	}
}

// Test PinType string representation
func (pt GitHubPinType) String() string {
	switch pt {
	case GitHubPinTag:
		return "tag"
	case GitHubPinCommit:
		return "commit"
	case GitHubPinBranch:
		return "branch"
	case GitHubPinRelease:
		return "release"
	default:
		return "unknown"
	}
}

// Test validation of pinning requirements
func TestGitHubPinDetection_ValidationRules(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		requireImmutable bool
		shouldPass       bool
	}{
		{
			name:             "Tag is immutable",
			url:              "https://raw.githubusercontent.com/myorg/configs/v1.0.0/prod.json",
			requireImmutable: true,
			shouldPass:       true,
		},
		{
			name:             "Commit is immutable",
			url:              "https://raw.githubusercontent.com/myorg/configs/abc123def456/prod.json",
			requireImmutable: true,
			shouldPass:       true,
		},
		{
			name:             "Branch is mutable - should fail when immutable required",
			url:              "https://raw.githubusercontent.com/myorg/configs/main/prod.json",
			requireImmutable: true,
			shouldPass:       false,
		},
		{
			name:             "Branch is ok when immutable not required",
			url:              "https://raw.githubusercontent.com/myorg/configs/main/prod.json",
			requireImmutable: false,
			shouldPass:       true,
		},
		{
			name:             "Release is immutable",
			url:              "https://github.com/myorg/configs/releases/download/v1.0.0/prod.json",
			requireImmutable: true,
			shouldPass:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := ParseGitHubURL(tt.url)
			if !ok {
				t.Fatalf("Failed to parse GitHub URL: %s", tt.url)
			}

			passes := !tt.requireImmutable || !info.IsMutable
			if passes != tt.shouldPass {
				t.Errorf("Validation rule failed: requireImmutable=%v, isMutable=%v, expected pass=%v, got pass=%v",
					tt.requireImmutable, info.IsMutable, tt.shouldPass, passes)
			}
		})
	}
}

// Benchmark GitHub URL parsing
func BenchmarkParseGitHubURL(b *testing.B) {
	urls := []string{
		"https://raw.githubusercontent.com/myorg/configs/v1.0.0/prod.json",
		"https://raw.githubusercontent.com/myorg/configs/abc123def456/prod.json",
		"https://raw.githubusercontent.com/myorg/configs/main/prod.json",
		"https://github.com/myorg/configs/releases/download/v1.0.0/prod.json",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		url := urls[i%len(urls)]
		ParseGitHubURL(url)
	}
}

// Helper function to format GitHub pin info for display
func FormatGitHubPinInfo(info *GitHubURLInfo) string {
	var sb strings.Builder

	sb.WriteString("GitHub URL: ")
	sb.WriteString(info.Owner)
	sb.WriteString("/")
	sb.WriteString(info.Repo)
	sb.WriteString(" @ ")
	sb.WriteString(info.Ref)
	sb.WriteString(" (")
	sb.WriteString(info.PinType.String())
	sb.WriteString(")")

	if info.IsPinned {
		sb.WriteString(" ✓ PINNED")
	}

	if info.IsMutable {
		sb.WriteString(" ⚠ MUTABLE")
	}

	return sb.String()
}

// Test formatting output
func TestFormatGitHubPinInfo(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{
			url:      "https://raw.githubusercontent.com/myorg/configs/v1.0.0/prod.json",
			expected: "GitHub URL: myorg/configs @ v1.0.0 (tag) ✓ PINNED",
		},
		{
			url:      "https://raw.githubusercontent.com/myorg/configs/main/prod.json",
			expected: "GitHub URL: myorg/configs @ main (branch) ⚠ MUTABLE",
		},
		{
			url:      "https://github.com/myorg/configs/releases/download/v1.0.0/prod.json",
			expected: "GitHub URL: myorg/configs @ v1.0.0 (release) ✓ PINNED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			info, ok := ParseGitHubURL(tt.url)
			if !ok {
				t.Fatalf("Failed to parse GitHub URL: %s", tt.url)
			}

			output := FormatGitHubPinInfo(info)
			if output != tt.expected {
				t.Errorf("Expected output:\n%s\nGot:\n%s", tt.expected, output)
			}
		})
	}
}
