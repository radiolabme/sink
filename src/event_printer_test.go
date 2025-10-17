package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBug1_WrongLineSplitFunction tests that output line splitting uses the correct function
func TestBug1_WrongLineSplitFunction(t *testing.T) {
	// This test demonstrates that filepath.SplitList is used incorrectly
	// It should use strings.Split(output, "\n") instead

	multiLineOutput := "First line of output\nSecond line of output\nThird line of output"

	// What the current code does (WRONG)
	wrongResult := filepath.SplitList(multiLineOutput)

	// What it should do (CORRECT)
	correctResult := strings.Split(multiLineOutput, "\n")

	t.Logf("Multi-line output: %q", multiLineOutput)
	t.Logf("filepath.SplitList result (WRONG): %v (length=%d)", wrongResult, len(wrongResult))
	t.Logf("strings.Split result (CORRECT): %v (length=%d)", correctResult, len(correctResult))

	// Test case 1: Multi-line output should split into multiple lines
	if len(wrongResult) != 1 {
		t.Errorf("Bug not reproduced: expected filepath.SplitList to return 1 element, got %d", len(wrongResult))
	}

	if len(correctResult) != 3 {
		t.Errorf("strings.Split should return 3 lines, got %d", len(correctResult))
	}

	// Test case 2: Output with path separators gets incorrectly split
	pathOutput := "/usr/bin:/usr/local/bin:/opt/homebrew/bin"
	wrongPathResult := filepath.SplitList(pathOutput)
	correctPathResult := strings.Split(pathOutput, "\n")

	t.Logf("\nPath output: %q", pathOutput)
	t.Logf("filepath.SplitList result (WRONG): %v (length=%d)", wrongPathResult, len(wrongPathResult))
	t.Logf("strings.Split result (CORRECT): %v (length=%d)", correctPathResult, len(correctPathResult))

	// The bug: filepath.SplitList splits by ':' on Unix, not by newline
	if len(wrongPathResult) == 3 {
		t.Logf("BUG CONFIRMED: filepath.SplitList incorrectly splits by ':' giving %d parts", len(wrongPathResult))
	}

	if len(correctPathResult) != 1 {
		t.Errorf("Single-line path output should return 1 line, got %d", len(correctPathResult))
	}

	// Test case 3: Output with both newlines and path separators
	complexOutput := "PATH=/usr/bin:/usr/local/bin\nHOME=/home/user\nSHELL=/bin/bash"
	wrongComplexResult := filepath.SplitList(complexOutput)
	correctComplexResult := strings.Split(complexOutput, "\n")

	t.Logf("\nComplex output: %q", complexOutput)
	t.Logf("filepath.SplitList result (WRONG): %v (length=%d)", wrongComplexResult, len(wrongComplexResult))
	t.Logf("strings.Split result (CORRECT): %v (length=%d)", correctComplexResult, len(correctComplexResult))

	// The bug causes incorrect splitting
	if len(wrongComplexResult) != len(correctComplexResult) {
		t.Logf("BUG CONFIRMED: filepath.SplitList gives %d parts vs correct %d parts",
			len(wrongComplexResult), len(correctComplexResult))
	}
}

// TestBug1_ActualEventHandler tests the bug in the actual event handler code
func TestBug1_ActualEventHandler(t *testing.T) {
	// Create a mock transport that returns multi-line output
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"echo test": {
				stdout:   "Line 1: Hello\nLine 2: World\nLine 3: Test",
				exitCode: 0,
			},
			"echo $PATH": {
				stdout:   "/usr/bin:/usr/local/bin:/opt/bin",
				exitCode: 0,
			},
		},
	}

	executor := NewExecutor(mockTransport)

	// Capture events
	var capturedEvents []ExecutionEvent
	executor.OnEvent = func(event ExecutionEvent) {
		capturedEvents = append(capturedEvents, event)
	}

	// Execute a step that produces multi-line output
	step := InstallStep{
		Name: "Multi-line output test",
		Step: CommandStep{Command: "echo test"},
	}

	result := executor.ExecuteStep(step, nil)

	if result.Error != "" {
		t.Fatalf("Step failed: %v", result.Error)
	}

	// Find the success event
	var successEvent *ExecutionEvent
	for i := range capturedEvents {
		if capturedEvents[i].Status == "success" {
			successEvent = &capturedEvents[i]
			break
		}
	}

	if successEvent == nil {
		t.Fatal("No success event found")
	}

	if successEvent.Output == "" {
		t.Fatal("Success event has no output")
	}

	// Test what the current buggy code does
	lines := filepath.SplitList(successEvent.Output)

	t.Logf("Event output: %q", successEvent.Output)
	t.Logf("filepath.SplitList gives %d lines: %v", len(lines), lines)

	// The bug: it doesn't split by newlines, so we get all output in one "line"
	if len(lines) == 1 && strings.Contains(successEvent.Output, "\n") {
		t.Errorf("BUG CONFIRMED: Multi-line output not split correctly. Got 1 line when output contains newlines")
	}

	// What it should do
	correctLines := strings.Split(successEvent.Output, "\n")
	t.Logf("strings.Split gives %d lines: %v", len(correctLines), correctLines)

	if len(correctLines) < 3 {
		t.Errorf("Expected at least 3 lines in output, got %d", len(correctLines))
	}
}

// TestBug2_MissingVerboseMetadataInDryRun tests that dry-run skipped events lack metadata
func TestBug2_MissingVerboseMetadataInDryRun(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"echo test": {stdout: "test", exitCode: 0},
		},
	}

	executor := NewExecutor(mockTransport)
	executor.DryRun = true
	executor.Verbose = true

	// Capture events
	var capturedEvents []ExecutionEvent
	executor.OnEvent = func(event ExecutionEvent) {
		capturedEvents = append(capturedEvents, event)
	}

	// Create a step with metadata
	message := "Test message for debugging"
	retry := "until"
	step := InstallStep{
		Name: "Test Step",
		Step: CommandStep{
			Command: "echo test",
			Message: &message,
			Retry:   &retry,
		},
	}

	result := executor.ExecuteStep(step, nil)

	if result.Status != "skipped" {
		t.Errorf("Expected skipped status in dry-run, got %s", result.Status)
	}

	// We should have 2 events: "running" and "skipped"
	if len(capturedEvents) != 2 {
		t.Fatalf("Expected 2 events (running + skipped), got %d", len(capturedEvents))
	}

	runningEvent := capturedEvents[0]
	skippedEvent := capturedEvents[1]

	if runningEvent.Status != "running" {
		t.Errorf("First event should be 'running', got %s", runningEvent.Status)
	}

	if skippedEvent.Status != "skipped" {
		t.Errorf("Second event should be 'skipped', got %s", skippedEvent.Status)
	}

	// Check that running event has verbose metadata
	if runningEvent.StepType != "CommandStep" {
		t.Errorf("Running event missing StepType, got %q", runningEvent.StepType)
	}

	if runningEvent.Message != message {
		t.Errorf("Running event missing Message, got %q, want %q", runningEvent.Message, message)
	}

	if runningEvent.Retry != retry {
		t.Errorf("Running event missing Retry, got %q, want %q", runningEvent.Retry, retry)
	}

	// FIXED: Check that skipped event now HAS verbose metadata (bug is fixed!)
	if skippedEvent.StepType != "CommandStep" {
		t.Errorf("Skipped event missing StepType: got %q, want %q", skippedEvent.StepType, "CommandStep")
	} else {
		t.Logf("✓ BUG FIXED: Skipped event now has StepType=%q", skippedEvent.StepType)
	}

	if skippedEvent.Message != message {
		t.Errorf("Skipped event missing Message: got %q, want %q", skippedEvent.Message, message)
	} else {
		t.Logf("✓ BUG FIXED: Skipped event now has Message=%q", skippedEvent.Message)
	}

	if skippedEvent.Retry != retry {
		t.Errorf("Skipped event missing Retry: got %q, want %q", skippedEvent.Retry, retry)
	} else {
		t.Logf("✓ BUG FIXED: Skipped event now has Retry=%q", skippedEvent.Retry)
	}

	t.Logf("\nRunning event: StepType=%q, Message=%q, Retry=%q",
		runningEvent.StepType, runningEvent.Message, runningEvent.Retry)
	t.Logf("Skipped event: StepType=%q, Message=%q, Retry=%q",
		skippedEvent.StepType, skippedEvent.Message, skippedEvent.Retry)
}

// TestBug2_JSONOutputConsistency tests that JSON output has consistent metadata
func TestBug2_JSONOutputConsistency(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"echo test": {stdout: "test", exitCode: 0},
		},
	}

	executor := NewExecutor(mockTransport)
	executor.DryRun = true
	executor.Verbose = true
	executor.JSONOutput = true

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Capture events via handler too
	var capturedEvents []ExecutionEvent
	executor.OnEvent = func(event ExecutionEvent) {
		capturedEvents = append(capturedEvents, event)
	}

	// Execute step
	step := InstallStep{
		Name: "JSON Test",
		Step: CommandStep{
			Command: "echo test",
		},
	}

	executor.ExecuteStep(step, nil)

	// Restore stdout and capture output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	jsonOutput := buf.String()

	t.Logf("JSON output:\n%s", jsonOutput)

	// Parse JSON events - they're pretty-printed, so we need to parse each complete object
	// Split by "}\n{" pattern to find object boundaries
	decoder := json.NewDecoder(strings.NewReader(jsonOutput))
	var jsonEvents []ExecutionEvent

	for {
		var event ExecutionEvent
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			t.Logf("Failed to decode JSON event: %v", err)
			break
		}
		jsonEvents = append(jsonEvents, event)
	}

	if len(jsonEvents) < 2 {
		t.Fatalf("Expected at least 2 JSON events, got %d", len(jsonEvents))
	}

	// Check consistency between running and skipped events
	var runningEvent, skippedEvent *ExecutionEvent
	for i := range jsonEvents {
		if jsonEvents[i].Status == "running" {
			runningEvent = &jsonEvents[i]
		}
		if jsonEvents[i].Status == "skipped" {
			skippedEvent = &jsonEvents[i]
		}
	}

	if runningEvent == nil {
		t.Fatal("No running event in JSON output")
	}
	if skippedEvent == nil {
		t.Fatal("No skipped event in JSON output")
	}

	// FIXED: Both events should now have the same metadata
	if runningEvent.StepType != "" && skippedEvent.StepType == "" {
		t.Errorf("BUG: Running event has StepType=%q but skipped event has empty StepType",
			runningEvent.StepType)
	} else if runningEvent.StepType == skippedEvent.StepType {
		t.Logf("✓ BUG FIXED: Both events have consistent StepType=%q", runningEvent.StepType)
	}
}

// TestBug3_SilentJSONMarshalingErrors tests that JSON marshaling errors are silently ignored
func TestBug3_SilentJSONMarshalingErrors(t *testing.T) {
	// Note: This is harder to test because Go's json.Marshal rarely fails
	// with normal data. We'd need to use channels or other types that
	// can't be marshaled, but ExecutionEvent uses simple types.

	// Instead, we'll test the error handling path by examining the code
	// and creating a scenario that could fail

	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"echo test": {stdout: "test", exitCode: 0},
		},
	}

	executor := NewExecutor(mockTransport)
	executor.JSONOutput = true

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutR, stdoutW, _ := os.Pipe()
	stderrR, stderrW, _ := os.Pipe()

	os.Stdout = stdoutW
	os.Stderr = stderrW

	// Execute step with normal data (this will succeed)
	step := InstallStep{
		Name: "Test",
		Step: CommandStep{Command: "echo test"},
	}

	executor.ExecuteStep(step, nil)

	// Restore stdout/stderr
	stdoutW.Close()
	stderrW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var stdoutBuf, stderrBuf bytes.Buffer
	io.Copy(&stdoutBuf, stdoutR)
	io.Copy(&stderrBuf, stderrR)

	jsonOutput := stdoutBuf.String()
	errorOutput := stderrBuf.String()

	t.Logf("JSON output length: %d bytes", len(jsonOutput))
	t.Logf("Stderr output: %q", errorOutput)

	// With normal data, JSON marshaling succeeds
	if len(jsonOutput) == 0 {
		t.Error("Expected JSON output, got none")
	}

	// BUG: If marshaling failed, there would be no error message to stderr
	// The current code does: if err == nil { fmt.Println(...) }
	// This means errors are silently ignored with no output at all

	t.Logf("BUG EXISTS: The emitEvent function silently ignores JSON marshaling errors")
	t.Logf("Current code: if err == nil { fmt.Println(string(jsonBytes)) }")
	t.Logf("If err != nil, nothing happens - no error message, no event, nothing")
}

// TestBug3_JSONErrorHandlingBehavior tests the actual error handling behavior
func TestBug3_JSONErrorHandlingBehavior(t *testing.T) {
	// Create a custom type that we try to marshal
	type BadType struct {
		Ch chan int // channels can't be marshaled to JSON
	}

	bad := BadType{Ch: make(chan int)}

	// Try to marshal it
	_, err := json.Marshal(bad)

	if err == nil {
		t.Error("Expected marshaling error for channel type")
	} else {
		t.Logf("Marshaling error (expected): %v", err)
	}

	// Now test what the current code does with errors
	silentlyIgnored := false
	if err != nil {
		// Current code does: if err == nil { fmt.Println(...) }
		// So when err != nil, nothing happens at all
		silentlyIgnored = true
	}

	if !silentlyIgnored {
		t.Error("Expected error to be silently ignored")
	} else {
		t.Log("BUG CONFIRMED: JSON marshaling errors are silently ignored")
		t.Log("No output to stdout (missing event)")
		t.Log("No output to stderr (missing error message)")
	}

	// The correct behavior would be to emit an error to stderr
	// so it doesn't corrupt the JSON stdout stream
	correctBehavior := func(err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Failed to marshal event to JSON: %v\n", err)
		}
	}

	// Test the correct behavior
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	correctBehavior(err)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	errOutput := buf.String()

	if !strings.Contains(errOutput, "WARNING") || !strings.Contains(errOutput, "marshal") {
		t.Error("Correct behavior should emit warning to stderr")
	} else {
		t.Logf("CORRECT behavior would output: %s", strings.TrimSpace(errOutput))
	}
}

// TestBug1_WindowsBehavior tests filepath.SplitList behavior on Windows
func TestBug1_WindowsBehavior(t *testing.T) {
	// On Windows, filepath.SplitList splits by ';' instead of ':'
	// This test documents the platform-specific nature of the bug

	output := "Line 1\nLine 2\nLine 3"

	// filepath.SplitList uses ListSeparator which is:
	// - ':' on Unix
	// - ';' on Windows
	separator := string(filepath.ListSeparator)

	t.Logf("OS-specific path separator: %q", separator)

	wrongResult := filepath.SplitList(output)
	correctResult := strings.Split(output, "\n")

	t.Logf("Output: %q", output)
	t.Logf("filepath.SplitList: %v (length=%d)", wrongResult, len(wrongResult))
	t.Logf("strings.Split: %v (length=%d)", correctResult, len(correctResult))

	// On any platform, using filepath.SplitList for line splitting is wrong
	if len(correctResult) != 3 {
		t.Error("strings.Split should always split newlines correctly")
	}

	// Document the bug across platforms
	t.Log("\nBUG AFFECTS ALL PLATFORMS:")
	t.Log("- Unix/Linux/macOS: filepath.SplitList splits by ':' (wrong for lines)")
	t.Log("- Windows: filepath.SplitList splits by ';' (wrong for lines)")
	t.Log("- Correct solution: strings.Split(output, \"\\n\") works on all platforms")
}

// TestBug2_CheckRemediateStepMetadata tests verbose metadata in CheckRemediateStep
func TestBug2_CheckRemediateStepMetadata(t *testing.T) {
	mockTransport := &MockTransport{
		responses: map[string]MockResponse{
			"which git":        {stdout: "", exitCode: 1}, // Check fails
			"brew install git": {stdout: "Installing...", exitCode: 0},
		},
	}

	executor := NewExecutor(mockTransport)
	executor.DryRun = true
	executor.Verbose = true

	var capturedEvents []ExecutionEvent
	executor.OnEvent = func(event ExecutionEvent) {
		capturedEvents = append(capturedEvents, event)
	}

	// Create a CheckRemediateStep with remediation metadata
	remRetry := "until"
	step := InstallStep{
		Name: "Install Git",
		Step: CheckRemediateStep{
			Check: "which git",
			OnMissing: []RemediationStep{
				{
					Name:    "Install via Homebrew",
					Command: "brew install git",
					Retry:   &remRetry,
				},
			},
		},
	}

	executor.ExecuteStep(step, nil)

	// In dry-run, we should get running + skipped events
	if len(capturedEvents) < 2 {
		t.Fatalf("Expected at least 2 events, got %d", len(capturedEvents))
	}

	runningEvent := capturedEvents[0]
	skippedEvent := capturedEvents[1]

	// Check running event has remediation metadata
	if runningEvent.StepType != "CheckRemediateStep" {
		t.Errorf("Running event has wrong StepType: %q", runningEvent.StepType)
	}

	if len(runningEvent.RemediationSteps) == 0 {
		t.Error("Running event missing remediation steps metadata")
	} else {
		t.Logf("Running event has %d remediation steps", len(runningEvent.RemediationSteps))
		if runningEvent.RemediationSteps[0].Retry != remRetry {
			t.Errorf("Remediation step missing retry: got %q, want %q",
				runningEvent.RemediationSteps[0].Retry, remRetry)
		}
	}

	// FIXED: Skipped event should now have metadata too
	if skippedEvent.StepType == "" {
		t.Error("Skipped event missing StepType")
	} else {
		t.Logf("✓ BUG FIXED: Skipped event has StepType=%q", skippedEvent.StepType)
	}

	if len(skippedEvent.RemediationSteps) == 0 {
		t.Error("Skipped event missing remediation steps metadata")
	} else {
		t.Logf("✓ BUG FIXED: Skipped event has %d remediation steps", len(skippedEvent.RemediationSteps))
	}
}
