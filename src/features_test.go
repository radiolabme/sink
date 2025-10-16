package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestTimeoutConfigParsing tests parsing of timeout configurations
func TestTimeoutConfigParsing(t *testing.T) {
	tests := []struct {
		name           string
		timeoutJSON    string
		expectInterval string
		expectErrCode  *int
		expectError    bool
	}{
		{
			name:           "simple string timeout",
			timeoutJSON:    `"30s"`,
			expectInterval: "30s",
			expectErrCode:  nil,
			expectError:    false,
		},
		{
			name:           "timeout object with interval only",
			timeoutJSON:    `{"interval": "5m"}`,
			expectInterval: "5m",
			expectErrCode:  nil,
			expectError:    false,
		},
		{
			name:           "timeout object with interval and error_code",
			timeoutJSON:    `{"interval": "2m", "error_code": 124}`,
			expectInterval: "2m",
			expectErrCode:  intPtr(124),
			expectError:    false,
		},
		{
			name:           "timeout object with custom error code",
			timeoutJSON:    `{"interval": "10s", "error_code": 143}`,
			expectInterval: "10s",
			expectErrCode:  intPtr(143),
			expectError:    false,
		},
		{
			name:          "invalid timeout format",
			timeoutJSON:   `123`,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interval, errCode, err := ParseTimeout(json.RawMessage(tt.timeoutJSON))

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if interval != tt.expectInterval {
				t.Errorf("expected interval %q, got %q", tt.expectInterval, interval)
			}

			if (tt.expectErrCode == nil) != (errCode == nil) {
				t.Errorf("error code mismatch: expected %v, got %v", tt.expectErrCode, errCode)
			}

			if tt.expectErrCode != nil && errCode != nil && *tt.expectErrCode != *errCode {
				t.Errorf("expected error code %d, got %d", *tt.expectErrCode, *errCode)
			}
		})
	}
}

// TestCommandStepWithVerbose tests verbose output for commands
func TestCommandStepWithVerbose(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	cmd := CommandStep{
		Command: "echo 'test output'",
		Verbose: true,
	}

	result := executor.executeCommand("test step", cmd, Facts{})

	if result.Status != "success" {
		t.Errorf("expected success, got %s", result.Status)
	}
}

// TestCommandStepWithSleep tests sleep functionality
func TestCommandStepWithSleep(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	sleepDuration := "100ms"
	cmd := CommandStep{
		Command: "echo 'test'",
		Sleep:   &sleepDuration,
	}

	start := time.Now()
	result := executor.executeCommand("test step", cmd, Facts{})
	elapsed := time.Since(start)

	if result.Status != "success" {
		t.Errorf("expected success, got %s: %s", result.Status, result.Error)
	}

	// Sleep should have taken at least 100ms
	if elapsed < 100*time.Millisecond {
		t.Errorf("expected sleep to take at least 100ms, took %v", elapsed)
	}
}

// TestCommandStepWithTimeoutObject tests timeout with custom error code
func TestCommandStepWithTimeoutObject(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	retry := "until"
	timeoutJSON := json.RawMessage(`{"interval": "200ms", "error_code": 124}`)

	// Command that will always fail
	cmd := CommandStep{
		Command: "exit 1",
		Retry:   &retry,
		Timeout: timeoutJSON,
	}

	result := executor.executeCommand("test step", cmd, Facts{})

	if result.Status != "failed" {
		t.Errorf("expected failed, got %s", result.Status)
	}

	// Should use custom error code 124
	if result.ExitCode != 124 {
		t.Errorf("expected exit code 124, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Error, "Timeout") {
		t.Errorf("expected timeout error, got: %s", result.Error)
	}
}

// TestRemediationStepWithVerbose tests verbose output for remediation
func TestRemediationStepWithVerbose(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	remStep := RemediationStep{
		Name:    "echo test",
		Command: "echo 'test'",
		Verbose: true,
	}

	result := executor.executeRemediation(remStep, Facts{})

	if result.Status != "success" {
		t.Errorf("expected success, got %s", result.Status)
	}
}

// TestRemediationStepWithSleep tests sleep in remediation
func TestRemediationStepWithSleep(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	sleepDuration := "100ms"
	remStep := RemediationStep{
		Name:    "start service",
		Command: "echo 'started'",
		Sleep:   &sleepDuration,
	}

	start := time.Now()
	result := executor.executeRemediation(remStep, Facts{})
	elapsed := time.Since(start)

	if result.Status != "success" {
		t.Errorf("expected success, got %s: %s", result.Status, result.Error)
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("expected sleep to take at least 100ms, took %v", elapsed)
	}
}

// TestFactDefWithVerbose tests verbose output for facts
func TestFactDefWithVerbose(t *testing.T) {
	transport := NewLocalTransport()

	facts := map[string]FactDef{
		"os": {
			Command: "uname",
			Verbose: true,
		},
	}

	gatherer := NewFactGatherer(facts, transport)
	result, err := gatherer.Gather()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["os"] == nil {
		t.Errorf("expected os fact to be gathered")
	}
}

// TestFactDefWithSleep tests sleep after fact gathering
func TestFactDefWithSleep(t *testing.T) {
	transport := NewLocalTransport()

	sleepDuration := "100ms"
	facts := map[string]FactDef{
		"test_fact": {
			Command: "echo 'test'",
			Sleep:   &sleepDuration,
		},
	}

	gatherer := NewFactGatherer(facts, transport)
	start := time.Now()
	result, err := gatherer.Gather()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["test_fact"] != "test" {
		t.Errorf("expected 'test', got %v", result["test_fact"])
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("expected sleep to take at least 100ms, took %v", elapsed)
	}
}

// TestInvalidSleepDuration tests error handling for invalid sleep durations
func TestInvalidSleepDuration(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	invalidSleep := "invalid"
	cmd := CommandStep{
		Command: "echo 'test'",
		Sleep:   &invalidSleep,
	}

	result := executor.executeCommand("test step", cmd, Facts{})

	if result.Status != "failed" {
		t.Errorf("expected failed due to invalid sleep, got %s", result.Status)
	}

	if !strings.Contains(result.Error, "sleep error") {
		t.Errorf("expected sleep error, got: %s", result.Error)
	}
}

// TestCombinedFeatures tests using verbose, sleep, and timeout together
func TestCombinedFeatures(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	sleepDuration := "50ms"
	timeoutJSON := json.RawMessage(`{"interval": "5s", "error_code": 124}`)

	cmd := CommandStep{
		Command: "echo 'done'",
		Verbose: true,
		Sleep:   &sleepDuration,
		Timeout: timeoutJSON,
	}

	start := time.Now()
	result := executor.executeCommand("test step", cmd, Facts{})
	elapsed := time.Since(start)

	if result.Status != "success" {
		t.Errorf("expected success, got %s: %s", result.Status, result.Error)
	}

	if elapsed < 50*time.Millisecond {
		t.Errorf("expected at least 50ms elapsed time for sleep")
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
