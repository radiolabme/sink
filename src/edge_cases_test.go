package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// edge_cases_test.go - Additional tests for uncovered error paths using standard Unix tools
// These tests target the gaps identified in coverage analysis without external dependencies

// TestTransportErrorCombinations tests the uncovered error path where
// a command fails with BOTH err != nil AND stderr != ""
func TestTransportErrorCombinations(t *testing.T) {
	transport := NewLocalTransport()

	tests := []struct {
		name         string
		command      string
		wantExitCode int
		wantStderr   bool
		wantStdout   bool
	}{
		{
			name:         "stderr with non-zero exit",
			command:      "sh -c 'echo error >&2; exit 1'",
			wantExitCode: 1,
			wantStderr:   true,
		},
		{
			name:         "both stdout and stderr",
			command:      "sh -c 'echo out; echo err >&2; exit 2'",
			wantExitCode: 2,
			wantStdout:   true,
			wantStderr:   true,
		},
		{
			name:         "false exits 1",
			command:      "false",
			wantExitCode: 1,
		},
		{
			name:         "command not found",
			command:      "nonexistent_command_xyz123",
			wantExitCode: 127,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitCode, _ := transport.Run(tt.command)

			if exitCode != tt.wantExitCode {
				t.Errorf("exitCode = %d, want %d", exitCode, tt.wantExitCode)
			}

			if tt.wantStdout && stdout == "" {
				t.Error("expected stdout but got none")
			}

			if tt.wantStderr && stderr == "" {
				t.Error("expected stderr but got none")
			}
		})
	}
}

// TestExecutorCommandWithStderr tests the executor.executeCommand path where
// stderr is included in the error message (currently uncovered)
func TestExecutorCommandWithStderr(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	step := InstallStep{
		Name: "Command with detailed stderr",
		Step: CommandStep{
			Command: "sh -c 'echo \"detailed error message\" >&2; exit 1'",
		},
	}

	result := executor.ExecuteStep(step, nil)

	if result.Error == "" {
		t.Fatal("expected error but got none")
	}

	// This covers the uncovered branch: errorMsg = fmt.Sprintf("%s\nstderr: %s", errorMsg, stderr)
	if !strings.Contains(result.Error, "stderr:") {
		t.Errorf("expected stderr in error message, got: %s", result.Error)
	}

	if !strings.Contains(result.Error, "detailed error message") {
		t.Errorf("expected detailed error in message, got: %s", result.Error)
	}
}

// TestFactsRequiredFailure tests the required fact failure path in facts.go
func TestFactsRequiredFailure(t *testing.T) {
	transport := NewLocalTransport()

	facts := map[string]FactDef{
		"REQUIRED_FACT": {
			Command:  "false", // Always fails
			Required: true,
		},
	}

	gatherer := NewFactGatherer(facts, transport)
	_, err := gatherer.Gather()

	if err == nil {
		t.Fatal("expected error for required fact failure")
	}

	if !strings.Contains(err.Error(), "required fact") {
		t.Errorf("expected 'required fact' in error, got: %v", err)
	}
}

// TestFactsStrictTransformFailure tests strict transform validation failure
func TestFactsStrictTransformFailure(t *testing.T) {
	transport := NewLocalTransport()

	facts := map[string]FactDef{
		"STRICT_FACT": {
			Command: "echo unknown_value",
			Type:    "string",
			Transform: map[string]string{
				"known": "mapped",
			},
			Strict:   true,
			Required: true,
		},
	}

	gatherer := NewFactGatherer(facts, transport)
	_, err := gatherer.Gather()

	if err == nil {
		t.Fatal("expected error for strict transform failure")
	}

	if !strings.Contains(err.Error(), "transform failed") {
		t.Errorf("expected transform error, got: %v", err)
	}
}

// TestConfigLoadFilesystemErrors tests LoadConfig error paths
func TestConfigLoadFilesystemErrors(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "nonexistent file",
			path:    "/tmp/nonexistent-sink-config-xyz123.json",
			wantErr: "failed to read config",
		},
		{
			name:    "directory instead of file",
			path:    "/tmp",
			wantErr: "failed to read config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(tt.path)
			if err == nil {
				t.Fatal("expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// TestConfigLoadMalformedJSON tests the json.Unmarshal error branch in LoadConfig
func TestConfigLoadMalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	badJSON := filepath.Join(tmpDir, "bad.json")

	// Create a file with invalid JSON
	err := os.WriteFile(badJSON, []byte(`{"invalid": ]`), 0644)
	if err != nil {
		t.Fatalf("couldn't create test file: %v", err)
	}

	_, err = LoadConfig(badJSON)
	if err == nil {
		t.Fatal("expected error for malformed JSON but got none")
	}

	// This covers the uncovered json.Unmarshal error branch
	if !strings.Contains(err.Error(), "parse") && !strings.Contains(err.Error(), "JSON") {
		t.Errorf("expected JSON parse error, got: %v", err)
	}
}

// TestTemplateInterpolationErrors tests the template execution error paths
func TestTemplateInterpolationErrors(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	tests := []struct {
		name    string
		command string
		facts   Facts
		wantErr string
	}{
		{
			name:    "invalid template syntax",
			command: "echo {{.invalid-syntax}}",
			facts:   Facts{"os": "darwin"},
			wantErr: "template",
		},
		{
			name:    "template with pipe error",
			command: "echo {{.os | invalidFunc}}",
			facts:   Facts{"os": "darwin"},
			wantErr: "template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := InstallStep{
				Name: tt.name,
				Step: CommandStep{
					Command: tt.command,
				},
			}

			result := executor.ExecuteStep(step, tt.facts)

			if result.Error == "" {
				t.Fatal("expected template error but got none")
			}

			if !strings.Contains(result.Error, tt.wantErr) {
				t.Errorf("expected error containing %q, got: %s", tt.wantErr, result.Error)
			}
		})
	}
}

// TestRemediationStepFailureWithStderr tests remediation failure with stderr output
func TestRemediationStepFailureWithStderr(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	step := InstallStep{
		Name: "Remediation with stderr",
		Step: CheckRemediateStep{
			Check: "false", // Always fails
			OnMissing: []RemediationStep{
				{
					Name:    "Failing remediation",
					Command: "sh -c 'echo \"remediation error\" >&2; exit 1'",
				},
			},
		},
	}

	result := executor.ExecuteStep(step, nil)

	if result.Error == "" {
		t.Fatal("expected remediation failure error")
	}

	if !strings.Contains(result.Error, "remediation failed") {
		t.Errorf("expected 'remediation failed' in error, got: %s", result.Error)
	}

	if len(result.RemediationSteps) == 0 {
		t.Error("expected remediation steps in result")
	}
}

// TestValidationEdgeCases tests config validation edge cases
func TestValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty install steps",
			config: &Config{
				Version: "1.0.0",
				Platforms: []Platform{
					{
						OS:           "darwin",
						Match:        "*",
						Name:         "macOS",
						InstallSteps: []InstallStep{}, // Empty!
					},
				},
			},
			wantErr: true,
			errMsg:  "install_steps or distributions",
		},
		{
			name: "invalid fact export name",
			config: &Config{
				Version: "1.0.0",
				Facts: map[string]FactDef{
					"invalid-name": { // Hyphens not allowed in exports
						Command: "echo test",
						Export:  "INVALID_NAME",
					},
				},
				Platforms: []Platform{
					{
						OS:    "darwin",
						Match: "*",
						Name:  "macOS",
						InstallSteps: []InstallStep{
							{Name: "test", Step: CommandStep{Command: "echo ok"}},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "fact name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error but got none")
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got: %v", tt.errMsg, err)
			}
		})
	}
}

// TestCheckErrorStepWithOutput tests CheckErrorStep with command output
func TestCheckErrorStepWithOutput(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	step := InstallStep{
		Name: "Check with output",
		Step: CheckErrorStep{
			Check: "sh -c 'echo some output; exit 1'",
			Error: "Check failed with output",
		},
	}

	result := executor.ExecuteStep(step, nil)

	if result.Error == "" {
		t.Fatal("expected error from CheckErrorStep")
	}

	// Verify error message includes our configured error
	if !strings.Contains(result.Error, "Check failed") {
		t.Errorf("expected configured error message, got: %s", result.Error)
	}
}

// TestFactsOptionalSkipOnFailure tests that optional facts are skipped on failure
func TestFactsOptionalSkipOnFailure(t *testing.T) {
	transport := NewLocalTransport()

	facts := map[string]FactDef{
		"OPTIONAL_FACT": {
			Command:  "sh -c 'echo error >&2; exit 1'",
			Required: false, // Should not fail overall gather
		},
		"REQUIRED_FACT": {
			Command:  "echo success",
			Required: true,
		},
	}

	gatherer := NewFactGatherer(facts, transport)
	result, err := gatherer.Gather()

	if err != nil {
		t.Fatalf("optional fact failure should not cause overall failure: %v", err)
	}

	// Optional fact should be missing
	if _, exists := result["OPTIONAL_FACT"]; exists {
		t.Error("expected optional failed fact to be missing from results")
	}

	// Required fact should be present
	if _, exists := result["REQUIRED_FACT"]; !exists {
		t.Error("expected required fact to be present")
	}
}
