package main

import (
	"strings"
	"testing"
)

// TestExecutorCommandStep tests execution of simple command steps
func TestExecutorCommandStep(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"echo hello": {stdout: "hello\n", exitCode: 0},
			"exit 1":     {stdout: "", exitCode: 1},
		},
	}

	tests := []struct {
		name      string
		step      InstallStep
		wantError bool
	}{
		{
			name: "successful command",
			step: InstallStep{
				Name: "Test echo",
				Step: CommandStep{Command: "echo hello"},
			},
			wantError: false,
		},
		{
			name: "failing command",
			step: InstallStep{
				Name: "Failing command",
				Step: CommandStep{Command: "exit 1"},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(mockTransport)
			result := executor.ExecuteStep(tt.step, nil)

			if tt.wantError {
				if result.Error == "" {
					t.Error("expected error but got none")
				}
			} else {
				if result.Error != "" {
					t.Errorf("unexpected error: %s", result.Error)
				}
			}
		})
	}
}

// TestExecutorCheckErrorStep tests check-error step execution
func TestExecutorCheckErrorStep(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"command -v brew":   {stdout: "/usr/local/bin/brew\n", exitCode: 0},
			"command -v nonext": {stdout: "", exitCode: 1},
		},
	}

	tests := []struct {
		name      string
		step      InstallStep
		wantError bool
		errorMsg  string
	}{
		{
			name: "check passes",
			step: InstallStep{
				Name: "Check Homebrew",
				Step: CheckErrorStep{
					Check: "command -v brew",
					Error: "Homebrew required",
				},
			},
			wantError: false,
		},
		{
			name: "check fails",
			step: InstallStep{
				Name: "Check nonexistent",
				Step: CheckErrorStep{
					Check: "command -v nonext",
					Error: "Tool not found",
				},
			},
			wantError: true,
			errorMsg:  "Tool not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(mockTransport)
			result := executor.ExecuteStep(tt.step, nil)

			if tt.wantError {
				if result.Error == "" {
					t.Error("expected error but got none")
				}
				if tt.errorMsg != "" && !strings.Contains(result.Error, tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, result.Error)
				}
			} else {
				if result.Error != "" {
					t.Errorf("unexpected error: %s", result.Error)
				}
			}
		})
	}
}

// TestExecutorCheckRemediateStep tests check-remediate step execution
func TestExecutorCheckRemediateStep(t *testing.T) {
	t.Run("check fails, remediation succeeds", func(t *testing.T) {
		// Create a stateful mock that changes behavior after remediation
		callCount := 0
		statefulTransport := &StatefulMockTransport{
			runFunc: func(cmd string) (string, string, int, error) {
				cmd = strings.TrimSpace(cmd)

				// First call to "command -v snap" fails, subsequent calls succeed
				if cmd == "command -v snap" {
					callCount++
					if callCount == 1 {
						return "", "", 1, nil // Not installed initially
					}
					return "/usr/bin/snap", "", 0, nil // Installed after remediation
				}

				// Remediation command succeeds
				if cmd == "sudo apt-get install snapd" {
					return "Installing...\n", "", 0, nil
				}

				// Discovery commands for context
				if strings.Contains(cmd, "hostname") || strings.Contains(cmd, "whoami") ||
					strings.Contains(cmd, "pwd") || strings.Contains(cmd, "uname") {
					return "test", "", 0, nil
				}

				return "", "command not mocked", 127, nil
			},
		}

		step := InstallStep{
			Name: "Check snapd",
			Step: CheckRemediateStep{
				Check: "command -v snap",
				OnMissing: []RemediationStep{
					{
						Name:    "Install snapd",
						Command: "sudo apt-get install snapd",
					},
				},
			},
		}

		executor := NewExecutor(statefulTransport)
		result := executor.ExecuteStep(step, nil)

		if result.Error != "" {
			t.Errorf("unexpected error: %s", result.Error)
		}
		if result.Status != "success" {
			t.Errorf("expected status 'success', got %q", result.Status)
		}
		if len(result.RemediationSteps) == 0 {
			t.Error("expected remediation steps but got none")
		}
	})

	t.Run("check passes, no remediation", func(t *testing.T) {
		mockTransport := &MockTransport{
			responses: map[string]MockResponse{
				"command -v brew": {stdout: "/usr/local/bin/brew\n", exitCode: 0},
			},
		}

		step := InstallStep{
			Name: "Check Homebrew",
			Step: CheckRemediateStep{
				Check:     "command -v brew",
				OnMissing: []RemediationStep{},
			},
		}

		executor := NewExecutor(mockTransport)
		result := executor.ExecuteStep(step, nil)

		if result.Error != "" {
			t.Errorf("unexpected error: %s", result.Error)
		}
		if len(result.RemediationSteps) > 0 {
			t.Errorf("expected no remediation but got %d steps", len(result.RemediationSteps))
		}
	})

	t.Run("check fails, remediation fails", func(t *testing.T) {
		mockTransport := &MockTransport{
			responses: map[string]MockResponse{
				"command -v snap":              {stdout: "", exitCode: 1},
				"brew install failing-package": {stdout: "", exitCode: 1},
			},
		}

		step := InstallStep{
			Name: "Check Homebrew",
			Step: CheckRemediateStep{
				Check: "command -v snap",
				OnMissing: []RemediationStep{
					{
						Name:    "Install failing package",
						Command: "brew install failing-package",
					},
				},
			},
		}

		executor := NewExecutor(mockTransport)
		result := executor.ExecuteStep(step, nil)

		if result.Error == "" {
			t.Error("expected error but got none")
		}
		if len(result.RemediationSteps) == 0 {
			t.Error("expected remediation steps to be recorded")
		}
	})
}

// TestExecutorErrorOnlyStep tests error-only step execution
func TestExecutorErrorOnlyStep(t *testing.T) {
	mockTransport := &MockTransport{}

	step := InstallStep{
		Name: "Unsupported platform",
		Step: ErrorOnlyStep{
			Error: "This platform is not supported",
		},
	}

	executor := NewExecutor(mockTransport)
	result := executor.ExecuteStep(step, nil)

	if result.Error == "" {
		t.Error("expected error but got none")
	}
	if !strings.Contains(result.Error, "not supported") {
		t.Errorf("expected error containing 'not supported', got %q", result.Error)
	}
}

// TestCheckRemediateWithRecheck tests that remediation is verified by re-running check
func TestCheckRemediateWithRecheck(t *testing.T) {
	t.Run("remediation succeeds and recheck passes", func(t *testing.T) {
		transport := NewLocalTransport()
		executor := NewExecutor(transport)

		// Create a test file that acts as our "state"
		testFile := "/tmp/sink-test-recheck"
		transport.Run("rm -f " + testFile)

		step := InstallStep{
			Name: "Create test file",
			Step: CheckRemediateStep{
				Check: "test -f " + testFile,
				OnMissing: []RemediationStep{
					{
						Name:    "Touch file",
						Command: "touch " + testFile,
					},
				},
			},
		}

		result := executor.ExecuteStep(step, nil)

		// Should succeed because remediation creates the file
		if result.Error != "" {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if result.Status != "success" {
			t.Errorf("expected status 'success', got %q", result.Status)
		}

		// Cleanup
		transport.Run("rm -f " + testFile)
	})

	t.Run("remediation succeeds but recheck fails", func(t *testing.T) {
		transport := NewLocalTransport()
		executor := NewExecutor(transport)

		step := InstallStep{
			Name: "Remediation that doesn't fix check",
			Step: CheckRemediateStep{
				// Check for a file that will never exist
				Check: "test -f /tmp/sink-nonexistent-file-xyz",
				OnMissing: []RemediationStep{
					{
						Name:    "Do something unrelated",
						Command: "echo 'this does not create the file'",
					},
				},
			},
		}

		result := executor.ExecuteStep(step, nil)

		// Should fail because remediation didn't fix the check
		if result.Error == "" {
			t.Error("expected error but got none")
		}
		if !strings.Contains(result.Error, "check still fails") {
			t.Errorf("expected error about check still failing, got: %s", result.Error)
		}
		if result.Status != "failed" {
			t.Errorf("expected status 'failed', got %q", result.Status)
		}
		if len(result.RemediationSteps) == 0 {
			t.Error("expected remediation steps to be recorded")
		}
	})
}

// TestExecutorWithFacts tests step execution with fact interpolation
func TestExecutorWithFacts(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"echo OS is darwin": {stdout: "OS is darwin\n", exitCode: 0},
		},
	}

	facts := Facts{
		"os": "darwin",
	}

	step := InstallStep{
		Name: "Echo OS",
		Step: CommandStep{Command: "echo OS is {{.os}}"},
	}

	executor := NewExecutor(mockTransport)
	result := executor.ExecuteStep(step, facts)

	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}

	// The command should have been interpolated with facts
	if !strings.Contains(result.Output, "OS is darwin") {
		t.Errorf("expected output containing 'OS is darwin', got %q", result.Output)
	}
}

// TestExecutorFullPlatformExecution tests executing all steps for a platform
func TestExecutorFullPlatformExecution(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"command -v brew":        {stdout: "/usr/local/bin/brew\n", exitCode: 0},
			"brew install some-tool": {stdout: "Installing...\n", exitCode: 0},
			"some-tool --version":    {stdout: "1.0.0\n", exitCode: 0},
		},
	}

	platform := Platform{
		OS:    "darwin",
		Match: "darwin*",
		Name:  "macOS",
		InstallSteps: []InstallStep{
			{
				Name: "Check Homebrew",
				Step: CheckErrorStep{
					Check: "command -v brew",
					Error: "Homebrew required",
				},
			},
			{
				Name: "Install tool",
				Step: CommandStep{Command: "brew install some-tool"},
			},
			{
				Name: "Verify installation",
				Step: CommandStep{Command: "some-tool --version"},
			},
		},
	}

	executor := NewExecutor(mockTransport)
	results := executor.ExecutePlatform(platform, nil)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	for i, result := range results {
		if result.Error != "" {
			t.Errorf("step %d failed: %s", i, result.Error)
		}
		if result.Status != "success" {
			t.Errorf("step %d status = %s, want success", i, result.Status)
		}
	}
}

// TestExecutorStopOnError tests that execution stops on first error
func TestExecutorStopOnError(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"echo step1": {stdout: "step1\n", exitCode: 0},
			"exit 1":     {stdout: "", exitCode: 1},
			"echo step3": {stdout: "step3\n", exitCode: 0},
		},
	}

	platform := Platform{
		OS:    "darwin",
		Match: "darwin*",
		Name:  "macOS",
		InstallSteps: []InstallStep{
			{
				Name: "Step 1",
				Step: CommandStep{Command: "echo step1"},
			},
			{
				Name: "Failing step",
				Step: CommandStep{Command: "exit 1"},
			},
			{
				Name: "Step 3",
				Step: CommandStep{Command: "echo step3"},
			},
		},
	}

	executor := NewExecutor(mockTransport)
	results := executor.ExecutePlatform(platform, nil)

	// Should have 2 results: success and failure, but not the third
	if len(results) != 2 {
		t.Errorf("expected 2 results (stop on error), got %d", len(results))
	}

	if results[0].Status != "success" {
		t.Errorf("step 0 status = %s, want success", results[0].Status)
	}

	if results[1].Status != "failed" {
		t.Errorf("step 1 status = %s, want failed", results[1].Status)
	}
}

// TestExecutorIdempotency tests that steps can be run multiple times
func TestExecutorIdempotency(t *testing.T) {
	callCount := make(map[string]int)

	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"idempotent-command": {stdout: "ok\n", exitCode: 0},
		},
		onRun: func(cmd string) {
			callCount[cmd]++
		},
	}

	step := InstallStep{
		Name: "Idempotent step",
		Step: CommandStep{Command: "idempotent-command"},
	}

	executor := NewExecutor(mockTransport)

	// Run the same step twice
	result1 := executor.ExecuteStep(step, nil)
	result2 := executor.ExecuteStep(step, nil)

	if result1.Error != "" {
		t.Errorf("first run failed: %s", result1.Error)
	}
	if result2.Error != "" {
		t.Errorf("second run failed: %s", result2.Error)
	}

	if callCount["idempotent-command"] != 2 {
		t.Errorf("expected command to be called 2 times, got %d", callCount["idempotent-command"])
	}
}

// TestExecutorDryRun tests dry-run mode where no commands are actually executed
func TestExecutorDryRun(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"dangerous-command": {stdout: "should not run\n", exitCode: 0},
		},
	}

	step := InstallStep{
		Name: "Dangerous step",
		Step: CommandStep{Command: "dangerous-command"},
	}

	executor := NewExecutor(mockTransport)
	executor.DryRun = true

	result := executor.ExecuteStep(step, nil)

	if result.Status != "skipped" {
		t.Errorf("expected status 'skipped' in dry-run, got %s", result.Status)
	}
	if result.Error != "" {
		t.Errorf("unexpected error in dry-run: %s", result.Error)
	}
}

// TestExecutorEventEmission tests that execution events are emitted
func TestExecutorEventEmission(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"echo test": {stdout: "test\n", exitCode: 0},
		},
	}

	events := []ExecutionEvent{}
	executor := NewExecutor(mockTransport)
	executor.OnEvent = func(event ExecutionEvent) {
		events = append(events, event)
	}

	step := InstallStep{
		Name: "Test step",
		Step: CommandStep{Command: "echo test"},
	}

	executor.ExecuteStep(step, nil)

	if len(events) < 2 {
		t.Errorf("expected at least 2 events (start and end), got %d", len(events))
	}

	// First event should be "running"
	if events[0].Status != "running" {
		t.Errorf("first event status = %s, want running", events[0].Status)
	}

	// Last event should be "success"
	lastEvent := events[len(events)-1]
	if lastEvent.Status != "success" {
		t.Errorf("last event status = %s, want success", lastEvent.Status)
	}
}

// Enhanced MockTransport with call tracking
type MockTransportWithTracking struct {
	responses map[string]MockResponse
	calls     []string
	onRun     func(cmd string)
}

func (m *MockTransportWithTracking) Run(cmd string) (stdout, stderr string, exitCode int, err error) {
	cmd = strings.TrimSpace(cmd)
	m.calls = append(m.calls, cmd)

	if m.onRun != nil {
		m.onRun(cmd)
	}

	if resp, ok := m.responses[cmd]; ok {
		return resp.stdout, resp.stderr, resp.exitCode, resp.err
	}
	return "", "command not mocked", 127, nil
}

// StatefulMockTransport allows custom run logic for stateful tests
type StatefulMockTransport struct {
	runFunc func(cmd string) (stdout, stderr string, exitCode int, err error)
}

func (s *StatefulMockTransport) Run(cmd string) (stdout, stderr string, exitCode int, err error) {
	return s.runFunc(cmd)
}
