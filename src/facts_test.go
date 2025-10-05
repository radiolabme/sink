package main

import (
	"errors"
	"strings"
	"testing"
)

// TestFactGathering tests the fact gathering system
func TestFactGathering(t *testing.T) {
	// Create a mock transport that returns predetermined values
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"uname -s | tr '[:upper:]' '[:lower:]'": {stdout: "darwin\n", exitCode: 0},
			"uname -m":                              {stdout: "x86_64\n", exitCode: 0},
			"hostname":                              {stdout: "test-host\n", exitCode: 0},
			"whoami":                                {stdout: "testuser\n", exitCode: 0},
			"echo $HOME":                            {stdout: "/home/testuser\n", exitCode: 0},
			"command -v brew >/dev/null && echo true || echo false": {stdout: "true\n", exitCode: 0},
		},
	}

	factDefs := map[string]FactDef{
		"os": {
			Command: "uname -s | tr '[:upper:]' '[:lower:]'",
			Export:  "SINK_OS",
		},
		"arch": {
			Command: "uname -m",
			Transform: map[string]string{
				"x86_64":  "amd64",
				"aarch64": "arm64",
				"arm64":   "arm64",
			},
			Export: "SINK_ARCH",
		},
		"hostname": {
			Command: "hostname",
			Export:  "SINK_HOSTNAME",
		},
		"user": {
			Command: "whoami",
			Export:  "SINK_USER",
		},
		"home": {
			Command: "echo $HOME",
			Export:  "SINK_HOME",
		},
		"has_brew": {
			Command: "command -v brew >/dev/null && echo true || echo false",
			Type:    "boolean",
			Export:  "SINK_HAS_BREW",
		},
	}

	gatherer := NewFactGatherer(factDefs, mockTransport)
	facts, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Test expected facts
	tests := []struct {
		name     string
		key      string
		expected interface{}
	}{
		{"os fact", "os", "darwin"},
		{"arch fact with transform", "arch", "amd64"},
		{"hostname fact", "hostname", "test-host"},
		{"user fact", "user", "testuser"},
		{"home fact", "home", "/home/testuser"},
		{"boolean fact", "has_brew", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if val, ok := facts[tt.key]; !ok {
				t.Errorf("fact '%s' not found", tt.key)
			} else if val != tt.expected {
				t.Errorf("fact '%s' = %v, want %v", tt.key, val, tt.expected)
			}
		})
	}
}

// TestFactTransform tests value transformation
func TestFactTransform(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		transform map[string]string
		strict    bool
		expected  string
		wantErr   bool
	}{
		{
			name:      "exact match",
			input:     "x86_64",
			transform: map[string]string{"x86_64": "amd64"},
			expected:  "amd64",
		},
		{
			name:      "no match, not strict",
			input:     "unknown",
			transform: map[string]string{"x86_64": "amd64"},
			strict:    false,
			expected:  "unknown",
		},
		{
			name:      "no match, strict",
			input:     "unknown",
			transform: map[string]string{"x86_64": "amd64"},
			strict:    true,
			wantErr:   true,
		},
		{
			name:     "no transform",
			input:    "value",
			expected: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := applyTransform(tt.input, tt.transform, tt.strict)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("got %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

// TestFactTypeCoercion tests type coercion for facts
func TestFactTypeCoercion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		typ      string
		expected interface{}
		wantErr  bool
	}{
		{"string type", "hello", "string", "hello", false},
		{"boolean true", "true", "boolean", true, false},
		{"boolean false", "false", "boolean", false, false},
		{"boolean invalid", "maybe", "boolean", nil, true},
		{"integer valid", "42", "integer", int64(42), false},
		{"integer invalid", "not-a-number", "integer", nil, true},
		{"default to string", "value", "", "value", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := coerceType(tt.input, tt.typ)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("got %v (type %T), want %v (type %T)", result, result, tt.expected, tt.expected)
				}
			}
		})
	}
}

// TestFactPlatformFiltering tests that facts are only gathered on specified platforms
func TestFactPlatformFiltering(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"uname -s | tr '[:upper:]' '[:lower:]'": {stdout: "darwin\n", exitCode: 0},
			"linux-only command":                    {stdout: "should-not-run\n", exitCode: 0},
		},
	}

	factDefs := map[string]FactDef{
		"os": {
			Command: "uname -s | tr '[:upper:]' '[:lower:]'",
		},
		"linux_only": {
			Command:   "linux-only command",
			Platforms: []string{"linux"},
		},
	}

	gatherer := NewFactGatherer(factDefs, mockTransport)
	gatherer.currentOS = "darwin" // Simulate running on macOS

	facts, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	if _, ok := facts["os"]; !ok {
		t.Error("expected 'os' fact to be gathered")
	}

	if _, ok := facts["linux_only"]; ok {
		t.Error("expected 'linux_only' fact to be skipped on darwin")
	}
}

// TestFactExport tests exporting facts as environment variables
func TestFactExport(t *testing.T) {
	facts := Facts{
		"os":       "darwin",
		"arch":     "amd64",
		"has_brew": true,
	}

	factDefs := map[string]FactDef{
		"os":        {Export: "SINK_OS"},
		"arch":      {Export: "SINK_ARCH"},
		"has_brew":  {Export: "SINK_HAS_BREW"},
		"no_export": {Export: ""}, // Should not be exported
	}

	gatherer := &FactGatherer{definitions: factDefs}
	exports := gatherer.Export(facts)

	expected := []string{
		"SINK_OS=darwin",
		"SINK_ARCH=amd64",
		"SINK_HAS_BREW=true",
	}

	if len(exports) != len(expected) {
		t.Errorf("expected %d exports, got %d", len(expected), len(exports))
	}

	for _, exp := range expected {
		found := false
		for _, got := range exports {
			if got == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected export %q not found in %v", exp, exports)
		}
	}
}

// TestFactRequiredValidation tests that required facts fail if they cannot be gathered
func TestFactRequiredValidation(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"working command": {stdout: "ok\n", exitCode: 0},
			"failing command": {stdout: "", exitCode: 1, err: errors.New("command failed")},
		},
	}

	tests := []struct {
		name     string
		factDefs map[string]FactDef
		wantErr  bool
	}{
		{
			name: "optional fact fails",
			factDefs: map[string]FactDef{
				"optional": {
					Command:  "failing command",
					Required: false,
				},
			},
			wantErr: false,
		},
		{
			name: "required fact fails",
			factDefs: map[string]FactDef{
				"required": {
					Command:  "failing command",
					Required: true,
				},
			},
			wantErr: true,
		},
		{
			name: "required fact succeeds",
			factDefs: map[string]FactDef{
				"required": {
					Command:  "working command",
					Required: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gatherer := NewFactGatherer(tt.factDefs, mockTransport)
			_, err := gatherer.Gather()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestFactOutputTrimming tests that fact values are trimmed
func TestFactOutputTrimming(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"echo test": {stdout: "  test  \n\n", exitCode: 0},
		},
	}

	factDefs := map[string]FactDef{
		"trimmed": {Command: "echo test"},
	}

	gatherer := NewFactGatherer(factDefs, mockTransport)
	facts, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	if val := facts["trimmed"]; val != "test" {
		t.Errorf("expected 'test', got %q", val)
	}
}

// MockTransport is a mock implementation of Transport for testing
type MockTransport struct {
	responses map[string]MockResponse
	onRun     func(cmd string)
}

type MockResponse struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
}

func (m *MockTransport) Run(cmd string) (stdout, stderr string, exitCode int, err error) {
	// Normalize command (trim spaces)
	cmd = strings.TrimSpace(cmd)

	if m.onRun != nil {
		m.onRun(cmd)
	}

	if resp, ok := m.responses[cmd]; ok {
		return resp.stdout, resp.stderr, resp.exitCode, resp.err
	}
	// Default response for unmocked commands
	return "", "command not mocked", 127, nil
}
