package main

import (
	"strings"
	"testing"
	"time"
)

// TestRetryMechanism tests the retry-until-success feature
func TestRetryMechanism(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		retry         *string
		timeout       *string
		expectSuccess bool
		expectOutput  string
		minDuration   time.Duration
		maxDuration   time.Duration
	}{
		{
			name:          "no retry - immediate success",
			command:       "echo 'success'",
			retry:         nil,
			timeout:       nil,
			expectSuccess: true,
			expectOutput:  "success",
			minDuration:   0,
			maxDuration:   1 * time.Second,
		},
		{
			name:          "no retry - immediate failure",
			command:       "exit 1",
			retry:         nil,
			timeout:       nil,
			expectSuccess: false,
			expectOutput:  "",
			minDuration:   0,
			maxDuration:   1 * time.Second,
		},
		{
			name:          "retry until - quick success",
			command:       "touch /tmp/sink-test-retry && test -f /tmp/sink-test-retry",
			retry:         stringPtr("until"),
			timeout:       stringPtr("5s"),
			expectSuccess: true,
			expectOutput:  "Ready after",
			minDuration:   0,
			maxDuration:   2 * time.Second,
		},
		{
			name:          "retry until - timeout",
			command:       "false",
			retry:         stringPtr("until"),
			timeout:       stringPtr("2s"),
			expectSuccess: false,
			expectOutput:  "Timeout after 2s",
			minDuration:   2 * time.Second,
			maxDuration:   3 * time.Second,
		},
		{
			name:          "retry until - default timeout (60s not tested, but we test format)",
			command:       "echo 'immediate success'",
			retry:         stringPtr("until"),
			timeout:       nil, // Should default to 60s
			expectSuccess: true,
			expectOutput:  "Ready after",
			minDuration:   0,
			maxDuration:   1 * time.Second,
		},
		{
			name:          "retry until - custom timeout format",
			command:       "echo 'success'",
			retry:         stringPtr("until"),
			timeout:       stringPtr("1m"),
			expectSuccess: true,
			expectOutput:  "Ready after",
			minDuration:   0,
			maxDuration:   1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cleanup before test
			transport := NewLocalTransport()
			transport.Run("rm -f /tmp/sink-test-retry")

			executor := NewExecutor(transport)
			step := CommandStep{
				Command: tt.command,
				Retry:   tt.retry,
				Timeout: tt.timeout,
			}

			startTime := time.Now()
			result := executor.executeCommand("test-step", step, Facts{})
			elapsed := time.Since(startTime)

			// Check success/failure
			if tt.expectSuccess && result.Status != "success" {
				t.Errorf("Expected success, got %s with error: %s", result.Status, result.Error)
			}
			if !tt.expectSuccess && result.Status != "failed" {
				t.Errorf("Expected failure, got %s", result.Status)
			}

			// Check output contains expected string
			if tt.expectOutput != "" {
				if !strings.Contains(result.Output, tt.expectOutput) && !strings.Contains(result.Error, tt.expectOutput) {
					t.Errorf("Expected output/error to contain '%s', got output: %s, error: %s", tt.expectOutput, result.Output, result.Error)
				}
			}

			// Check duration bounds
			if elapsed < tt.minDuration {
				t.Errorf("Expected at least %s, took %s", tt.minDuration, elapsed)
			}
			if elapsed > tt.maxDuration {
				t.Errorf("Expected at most %s, took %s", tt.maxDuration, elapsed)
			}

			// Cleanup after test
			transport.Run("rm -f /tmp/sink-test-retry")
		})
	}
}

// TestRetryInvalidTimeout tests error handling for invalid timeout formats
func TestRetryInvalidTimeout(t *testing.T) {
	executor := NewExecutor(NewLocalTransport())

	tests := []struct {
		name    string
		timeout string
	}{
		{"invalid format", "invalid"},
		{"negative timeout", "-10s"},
		{"just a number", "5000"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := CommandStep{
				Command: "echo 'test'",
				Retry:   stringPtr("until"),
				Timeout: &tt.timeout,
			}

			result := executor.executeCommand("test-step", step, Facts{})

			// Empty string should use default (and succeed)
			if tt.timeout == "" {
				if result.Status != "success" {
					t.Errorf("Expected success with default timeout, got %s: %s", result.Status, result.Error)
				}
			} else {
				// Invalid formats should either error or succeed with default
				// Current implementation defaults to 60s on invalid, so it succeeds
				if result.Status == "failed" && !strings.Contains(result.Error, "invalid timeout") {
					// If it fails, it should be due to invalid timeout, not command failure
					t.Errorf("Expected invalid timeout error or success, got: %s", result.Error)
				}
			}
		})
	}
}

// TestRetryInRemediationSteps tests retry in remediation steps
func TestRetryInRemediationSteps(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	// Cleanup
	transport.Run("rm -f /tmp/sink-remediation-retry-test")

	remStep := RemediationStep{
		Name:    "wait for file",
		Command: "test -f /tmp/sink-remediation-retry-test || (sleep 2 && touch /tmp/sink-remediation-retry-test && exit 1)",
		Retry:   stringPtr("until"),
		Timeout: stringPtr("5s"),
	}

	startTime := time.Now()
	result := executor.executeRemediation(remStep, Facts{})
	elapsed := time.Since(startTime)

	if result.Status != "success" {
		t.Errorf("Expected success, got %s: %s", result.Status, result.Error)
	}

	if elapsed < 1*time.Second {
		t.Errorf("Expected at least 1s (retry delay), took %s", elapsed)
	}

	if elapsed > 5*time.Second {
		t.Errorf("Expected at most 5s, took %s", elapsed)
	}

	if !strings.Contains(result.Output, "Ready after") {
		t.Errorf("Expected 'Ready after' in output, got: %s", result.Output)
	}

	// Cleanup
	transport.Run("rm -f /tmp/sink-remediation-retry-test")
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
