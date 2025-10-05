package main

import (
	"runtime"
	"strings"
	"testing"
)

// TestIntegrationFullStack tests the complete stack end-to-end
func TestIntegrationFullStack(t *testing.T) {
	// Create a minimal config
	config := &Config{
		Version: "1.0.0",
		Facts: map[string]FactDef{
			"os": {
				Command: "uname -s | tr '[:upper:]' '[:lower:]'",
				Export:  "SINK_OS",
			},
		},
		Platforms: []Platform{
			{
				OS:    "darwin",
				Match: "darwin*",
				Name:  "macOS",
				InstallSteps: []InstallStep{
					{
						Name: "Check shell",
						Step: CheckErrorStep{
							Check: "command -v sh",
							Error: "sh not found",
						},
					},
					{
						Name: "Echo with fact",
						Step: CommandStep{
							Command: "echo Running on {{.os}}",
						},
					},
				},
			},
			{
				OS:    "linux",
				Match: "linux*",
				Name:  "Linux",
				InstallSteps: []InstallStep{
					{
						Name: "Check shell",
						Step: CheckErrorStep{
							Check: "command -v sh",
							Error: "sh not found",
						},
					},
				},
			},
		},
	}

	// 1. Validate config
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	// 2. Create transport
	transport := NewLocalTransport()

	// 3. Gather facts
	gatherer := NewFactGatherer(config.Facts, transport)
	facts, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("fact gathering failed: %v", err)
	}

	// Verify facts were gathered
	if len(facts) == 0 {
		t.Fatal("no facts gathered")
	}

	osValue, ok := facts["os"]
	if !ok {
		t.Fatal("os fact not gathered")
	}

	// 4. Select platform based on runtime.GOOS
	var selectedPlatform *Platform
	for i := range config.Platforms {
		if config.Platforms[i].OS == runtime.GOOS {
			selectedPlatform = &config.Platforms[i]
			break
		}
	}

	if selectedPlatform == nil {
		t.Skipf("no platform configured for %s", runtime.GOOS)
	}

	// 5. Execute platform steps
	executor := NewExecutor(transport)
	results := executor.ExecutePlatform(*selectedPlatform, facts)

	if len(results) == 0 {
		t.Fatal("no results from execution")
	}

	// Verify execution results
	for i, result := range results {
		if result.Error != "" {
			t.Errorf("step %d (%s) failed: %s", i, result.StepName, result.Error)
		}
		if result.Status != "success" {
			t.Errorf("step %d status = %s, want success", i, result.Status)
		}
	}

	// Verify fact interpolation worked in the second step
	if len(results) > 1 {
		if !strings.Contains(results[1].Output, osValue.(string)) {
			t.Errorf("expected fact interpolation in output, got %q", results[1].Output)
		}
	}
}

// TestIntegrationWithRemediation tests check-remediate flow
func TestIntegrationWithRemediation(t *testing.T) {
	transport := NewLocalTransport()

	// Create a platform with check-remediate step
	platform := Platform{
		OS:    runtime.GOOS,
		Match: runtime.GOOS + "*",
		Name:  "Test Platform",
		InstallSteps: []InstallStep{
			{
				Name: "Check and install marker",
				Step: CheckRemediateStep{
					Check: "test -f /tmp/sink-test-marker",
					OnMissing: []RemediationStep{
						{
							Name:    "Create marker",
							Command: "touch /tmp/sink-test-marker",
						},
					},
				},
			},
		},
	}

	executor := NewExecutor(transport)

	// First run - marker doesn't exist, remediation should run
	results1 := executor.ExecutePlatform(platform, nil)
	if len(results1) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results1))
	}

	if results1[0].Error != "" {
		t.Errorf("first run failed: %s", results1[0].Error)
	}

	if len(results1[0].RemediationSteps) != 1 {
		t.Errorf("expected 1 remediation step, got %d", len(results1[0].RemediationSteps))
	}

	// Second run - marker exists, no remediation needed
	results2 := executor.ExecutePlatform(platform, nil)
	if len(results2) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results2))
	}

	if results2[0].Error != "" {
		t.Errorf("second run failed: %s", results2[0].Error)
	}

	if len(results2[0].RemediationSteps) != 0 {
		t.Errorf("expected no remediation steps on second run, got %d", len(results2[0].RemediationSteps))
	}

	// Clean up
	transport.Run("rm -f /tmp/sink-test-marker")
}

// TestIntegrationFactExport tests fact export as environment variables
func TestIntegrationFactExport(t *testing.T) {
	transport := NewLocalTransport()

	facts := map[string]FactDef{
		"test_var": {
			Command: "echo test_value",
			Export:  "TEST_VAR",
		},
	}

	gatherer := NewFactGatherer(facts, transport)
	gatheredFacts, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("fact gathering failed: %v", err)
	}

	// Export facts
	exports := gatherer.Export(gatheredFacts)

	if len(exports) != 1 {
		t.Fatalf("expected 1 export, got %d", len(exports))
	}

	if exports[0] != "TEST_VAR=test_value" {
		t.Errorf("export = %q, want 'TEST_VAR=test_value'", exports[0])
	}

	// Verify we can use the export in a command
	transport.Env = append(transport.Env, exports...)
	stdout, _, exitCode, err := transport.Run("echo $TEST_VAR")
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout, "test_value") {
		t.Errorf("stdout = %q, want to contain 'test_value'", stdout)
	}
}

// TestIntegrationEventEmission tests event emission during execution
func TestIntegrationEventEmission(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)

	events := []ExecutionEvent{}
	executor.OnEvent = func(event ExecutionEvent) {
		events = append(events, event)
	}

	platform := Platform{
		OS:    runtime.GOOS,
		Match: runtime.GOOS + "*",
		Name:  "Test Platform",
		InstallSteps: []InstallStep{
			{
				Name: "Step 1",
				Step: CommandStep{Command: "echo step1"},
			},
			{
				Name: "Step 2",
				Step: CommandStep{Command: "echo step2"},
			},
		},
	}

	results := executor.ExecutePlatform(platform, nil)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Should have at least 4 events: 2 "running" + 2 "success"
	if len(events) < 4 {
		t.Errorf("expected at least 4 events, got %d", len(events))
	}

	// Verify event structure
	if events[0].Status != "running" {
		t.Errorf("first event status = %s, want running", events[0].Status)
	}

	if events[0].StepName != "Step 1" {
		t.Errorf("first event step = %s, want 'Step 1'", events[0].StepName)
	}

	// All events should have a run ID
	runID := events[0].RunID
	for i, event := range events {
		if event.RunID != runID {
			t.Errorf("event %d has different runID: %s vs %s", i, event.RunID, runID)
		}
	}
}

// TestIntegrationDryRun tests dry-run mode
func TestIntegrationDryRun(t *testing.T) {
	transport := NewLocalTransport()
	executor := NewExecutor(transport)
	executor.DryRun = true

	platform := Platform{
		OS:    runtime.GOOS,
		Match: runtime.GOOS + "*",
		Name:  "Test Platform",
		InstallSteps: []InstallStep{
			{
				Name: "Dangerous command",
				Step: CommandStep{Command: "rm -rf /"},
			},
		},
	}

	results := executor.ExecutePlatform(platform, nil)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != "skipped" {
		t.Errorf("status = %s, want skipped", results[0].Status)
	}

	if results[0].Error != "" {
		t.Errorf("unexpected error in dry-run: %s", results[0].Error)
	}
}
