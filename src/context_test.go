package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestExecutorContextDiscovery tests that execution context is discovered
func TestExecutorContextDiscovery(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	ctx := executor.GetContext()

	// Verify all fields are populated
	if ctx.Host == "" {
		t.Error("Host should be populated")
	}

	if ctx.User == "" {
		t.Error("User should be populated")
	}

	if ctx.WorkDir == "" {
		t.Error("WorkDir should be populated")
	}

	if ctx.OS == "" {
		t.Error("OS should be populated")
	}

	if ctx.Arch == "" {
		t.Error("Arch should be populated")
	}

	if ctx.Transport != "local" {
		t.Errorf("Transport should be 'local', got: %s", ctx.Transport)
	}

	if ctx.Timestamp == "" {
		t.Error("Timestamp should be populated")
	}
}

// TestExecutorContextFields tests specific context fields
func TestExecutorContextFields(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	ctx := executor.GetContext()

	// OS should be Darwin on macOS
	if ctx.OS != "Darwin" && ctx.OS != "Linux" && ctx.OS != "Windows_NT" {
		t.Errorf("Unexpected OS: %s", ctx.OS)
	}

	// Arch should be reasonable
	validArchs := []string{"x86_64", "arm64", "amd64", "i386", "i686"}
	validArch := false
	for _, arch := range validArchs {
		if ctx.Arch == arch {
			validArch = true
			break
		}
	}
	if !validArch {
		t.Errorf("Unexpected architecture: %s", ctx.Arch)
	}

	// WorkDir should be an absolute path
	if !strings.HasPrefix(ctx.WorkDir, "/") && !strings.Contains(ctx.WorkDir, ":\\") {
		t.Errorf("WorkDir should be absolute path, got: %s", ctx.WorkDir)
	}
}

// TestExecutorContextInEvents tests that context is included in events
func TestExecutorContextInEvents(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	// Capture events
	var capturedEvents []ExecutionEvent
	executor.OnEvent = func(event ExecutionEvent) {
		capturedEvents = append(capturedEvents, event)
	}

	// Execute a simple step
	step := InstallStep{
		Name: "Test Step",
		Step: CommandStep{
			Command: "echo test",
		},
	}

	executor.ExecuteStep(step, nil)

	// Verify events have context
	if len(capturedEvents) == 0 {
		t.Fatal("Expected events to be emitted")
	}

	for i, event := range capturedEvents {
		if event.Context.Host == "" {
			t.Errorf("Event %d missing context.Host", i)
		}
		if event.Context.User == "" {
			t.Errorf("Event %d missing context.User", i)
		}
		if event.Context.Transport == "" {
			t.Errorf("Event %d missing context.Transport", i)
		}
	}
}

// TestExecutorContextConsistency tests that context is consistent across events
func TestExecutorContextConsistency(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	// Get initial context
	initialContext := executor.GetContext()

	// Capture events
	var capturedEvents []ExecutionEvent
	executor.OnEvent = func(event ExecutionEvent) {
		capturedEvents = append(capturedEvents, event)
	}

	// Execute multiple steps
	steps := []InstallStep{
		{
			Name: "Step 1",
			Step: CommandStep{Command: "echo step1"},
		},
		{
			Name: "Step 2",
			Step: CommandStep{Command: "echo step2"},
		},
	}

	for _, step := range steps {
		executor.ExecuteStep(step, nil)
	}

	// Verify all events have the same context
	for i, event := range capturedEvents {
		if event.Context.Host != initialContext.Host {
			t.Errorf("Event %d has different Host: %s vs %s", i, event.Context.Host, initialContext.Host)
		}
		if event.Context.User != initialContext.User {
			t.Errorf("Event %d has different User: %s vs %s", i, event.Context.User, initialContext.User)
		}
		if event.Context.Transport != initialContext.Transport {
			t.Errorf("Event %d has different Transport: %s vs %s", i, event.Context.Transport, initialContext.Transport)
		}
	}
}

// TestExecutorContextWithDryRun tests context in dry-run mode
func TestExecutorContextWithDryRun(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)
	executor.DryRun = true

	ctx := executor.GetContext()

	// Context should still be discovered in dry-run mode
	if ctx.Host == "" {
		t.Error("Host should be populated even in dry-run mode")
	}

	if ctx.User == "" {
		t.Error("User should be populated even in dry-run mode")
	}

	// Verify events still have context
	var capturedEvents []ExecutionEvent
	executor.OnEvent = func(event ExecutionEvent) {
		capturedEvents = append(capturedEvents, event)
	}

	step := InstallStep{
		Name: "Dry Run Step",
		Step: CommandStep{Command: "echo test"},
	}

	executor.ExecuteStep(step, nil)

	if len(capturedEvents) == 0 {
		t.Fatal("Expected events in dry-run mode")
	}

	for _, event := range capturedEvents {
		if event.Context.Host == "" {
			t.Error("Event context missing Host in dry-run mode")
		}
	}
}

// TestExecutorContextJSON tests that context can be serialized to JSON
func TestExecutorContextJSON(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	ctx := executor.GetContext()

	// Test that we can marshal to JSON (via ExecutionEvent)
	event := ExecutionEvent{
		Timestamp: "2025-01-01T00:00:00Z",
		RunID:     "test-run",
		StepName:  "Test",
		Status:    "success",
		Context:   ctx,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event with context: %v", err)
	}

	// Verify JSON contains context fields
	jsonStr := string(data)
	if !strings.Contains(jsonStr, "\"host\"") {
		t.Error("JSON missing host field")
	}
	if !strings.Contains(jsonStr, "\"user\"") {
		t.Error("JSON missing user field")
	}
	if !strings.Contains(jsonStr, "\"transport\"") {
		t.Error("JSON missing transport field")
	}
}
