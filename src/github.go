package main

import (
	"regexp"
)

// GitHubPinType represents the type of GitHub reference
type GitHubPinType int

const (
	GitHubPinUnknown GitHubPinType = iota
	GitHubPinTag
	GitHubPinCommit
	GitHubPinBranch
	GitHubPinRelease
)

// GitHubURLInfo contains parsed information from a GitHub URL
type GitHubURLInfo struct {
	Owner     string
	Repo      string
	Ref       string
	PinType   GitHubPinType
	IsPinned  bool
	IsMutable bool
}

// ParseGitHubURL parses a GitHub URL and extracts pinning information
func ParseGitHubURL(url string) (*GitHubURLInfo, bool) {
	// Pattern for raw.githubusercontent.com URLs
	rawPattern := regexp.MustCompile(`raw\.githubusercontent\.com/([^/]+)/([^/]+)/([^/]+)/`)
	if matches := rawPattern.FindStringSubmatch(url); matches != nil {
		info := &GitHubURLInfo{
			Owner: matches[1],
			Repo:  matches[2],
			Ref:   matches[3],
		}
		info.PinType = determineRefType(info.Ref)
		info.IsPinned = info.PinType == GitHubPinTag || info.PinType == GitHubPinCommit
		info.IsMutable = info.PinType == GitHubPinBranch
		return info, true
	}

	// Pattern for github.com/owner/repo/releases/download/tag/file
	releasePattern := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/releases/download/([^/]+)/`)
	if matches := releasePattern.FindStringSubmatch(url); matches != nil {
		info := &GitHubURLInfo{
			Owner:     matches[1],
			Repo:      matches[2],
			Ref:       matches[3],
			PinType:   GitHubPinRelease,
			IsPinned:  true,
			IsMutable: false,
		}
		return info, true
	}

	return nil, false
}

// determineRefType determines the type of Git reference
func determineRefType(ref string) GitHubPinType {
	// Semantic version tag: vX.Y.Z or vX.Y.Z-prerelease
	semverPattern := regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$`)
	if semverPattern.MatchString(ref) {
		return GitHubPinTag
	}

	// Full commit SHA (40 hex chars)
	fullSHAPattern := regexp.MustCompile(`^[0-9a-f]{40}$`)
	if fullSHAPattern.MatchString(ref) {
		return GitHubPinCommit
	}

	// Short commit SHA (7+ hex chars)
	shortSHAPattern := regexp.MustCompile(`^[0-9a-f]{7,}$`)
	if shortSHAPattern.MatchString(ref) {
		return GitHubPinCommit
	}

	// Common mutable branch names
	mutableBranches := []string{"main", "master", "develop", "dev", "staging", "production", "trunk"}
	for _, branch := range mutableBranches {
		if ref == branch {
			return GitHubPinBranch
		}
	}

	// Unknown - could be a custom branch or tag
	return GitHubPinUnknown
}

// Test semantic version tags
