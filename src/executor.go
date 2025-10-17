package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"
)

// Executor executes installation steps
type Executor struct {
	transport  Transport
	DryRun     bool
	Verbose    bool // Global verbose flag for debugging
	JSONOutput bool // Output events as JSON to stdout
	OnEvent    func(ExecutionEvent)
	runID      string
	context    ExecutionContext // Execution context (where commands run)
}

// NewExecutor creates a new executor
func NewExecutor(transport Transport) *Executor {
	executor := &Executor{
		transport: transport,
		runID:     generateRunID(),
	}

	// Discover execution context immediately
	executor.context = executor.discoverContext()

	return executor
}

// discoverContext discovers the execution environment
func (e *Executor) discoverContext() ExecutionContext {
	if e.Verbose {
		verboseLog("Discovering execution context...")
	}

	ctx := ExecutionContext{
		Timestamp: time.Now().Format(time.RFC3339),
		Transport: "unknown",
	}

	// Discover hostname
	stdout, _, exitCode, _ := e.transport.Run("hostname")
	if exitCode == 0 {
		ctx.Host = strings.TrimSpace(stdout)
	}

	// Discover user
	stdout, _, exitCode, _ = e.transport.Run("whoami")
	if exitCode == 0 {
		ctx.User = strings.TrimSpace(stdout)
	}

	// Discover working directory
	stdout, _, exitCode, _ = e.transport.Run("pwd")
	if exitCode == 0 {
		ctx.WorkDir = strings.TrimSpace(stdout)
	}

	// Discover OS
	stdout, _, exitCode, _ = e.transport.Run("uname -s")
	if exitCode == 0 {
		ctx.OS = strings.TrimSpace(stdout)
	}

	// Discover architecture
	stdout, _, exitCode, _ = e.transport.Run("uname -m")
	if exitCode == 0 {
		ctx.Arch = strings.TrimSpace(stdout)
	}

	// Determine transport type
	if _, ok := e.transport.(*LocalTransport); ok {
		ctx.Transport = "local"
	}
	// SSH transport detection will be added when SSH is implemented

	if e.Verbose {
		verboseLog("Context discovered: Host=%s, User=%s, OS=%s, Arch=%s", ctx.Host, ctx.User, ctx.OS, ctx.Arch)
	}

	return ctx
}

// GetContext returns the execution context
func (e *Executor) GetContext() ExecutionContext {
	return e.context
}

// ExecuteStep executes a single installation step
func (e *Executor) ExecuteStep(step InstallStep, facts Facts) StepResult {
	if e.Verbose {
		verboseLog("Executing step: %s", step.Name)
		e.logStepMetadata(step)
	}

	event := ExecutionEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		RunID:     e.runID,
		StepName:  step.Name,
		Status:    "running",
	}
	e.populateVerboseMetadata(&event, step)
	e.emitEvent(event)

	// Handle dry-run mode
	if e.DryRun {
		skippedEvent := ExecutionEvent{
			Timestamp: time.Now().Format(time.RFC3339),
			RunID:     e.runID,
			StepName:  step.Name,
			Status:    "skipped",
			Output:    "(dry-run mode)",
		}
		e.populateVerboseMetadata(&skippedEvent, step)
		e.emitEvent(skippedEvent)
		return StepResult{
			StepName: step.Name,
			Status:   "skipped",
			Output:   "(dry-run mode)",
		}
	}

	// Execute based on step variant
	var result StepResult
	switch v := step.Step.(type) {
	case CommandStep:
		result = e.executeCommand(step.Name, v, facts)
	case CheckErrorStep:
		result = e.executeCheckError(step.Name, v, facts)
	case CheckRemediateStep:
		result = e.executeCheckRemediate(step.Name, v, facts)
	case ErrorOnlyStep:
		result = e.executeErrorOnly(step.Name, v)
	default:
		result = StepResult{
			StepName: step.Name,
			Status:   "failed",
			Error:    fmt.Sprintf("unknown step variant: %T", v),
		}
	}

	// Emit completion event
	status := "success"
	if result.Error != "" {
		status = "failed"
	}
	completionEvent := ExecutionEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		RunID:     e.runID,
		StepName:  step.Name,
		Status:    status,
		Output:    result.Output,
		Error:     result.Error,
	}
	if result.ExitCode != 0 {
		completionEvent.ExitCode = &result.ExitCode
	}
	e.populateVerboseMetadata(&completionEvent, step)
	e.emitEvent(completionEvent)

	return result
}

// ExecutePlatform executes all steps for a platform
func (e *Executor) ExecutePlatform(platform Platform, facts Facts) []StepResult {
	results := []StepResult{}

	for _, step := range platform.InstallSteps {
		result := e.ExecuteStep(step, facts)
		results = append(results, result)

		// Stop on first error
		if result.Error != "" {
			break
		}
	}

	return results
}

// executeCommand executes a CommandStep
func (e *Executor) executeCommand(stepName string, cmd CommandStep, facts Facts) StepResult {
	// Check if retry is enabled
	if cmd.Retry != nil && *cmd.Retry == "until" {
		return e.executeCommandWithRetry(stepName, cmd, facts)
	}

	// Interpolate command with facts
	command, err := e.interpolate(cmd.Command, facts)
	if err != nil {
		return StepResult{
			StepName: stepName,
			Status:   "failed",
			Error:    fmt.Sprintf("template error: %v", err),
		}
	}

	// Log command execution in verbose mode (use global verbose or step-specific)
	verbose := e.Verbose || cmd.Verbose
	if verbose {
		verboseLog("Executing command: %s", command)
	}

	// Execute command
	stdout, stderr, exitCode, err := e.transport.Run(command)

	if verbose {
		verboseLog("Command exit code: %d", exitCode)
		if stdout != "" {
			verboseLog("stdout: %s", stdout)
		}
		if stderr != "" {
			verboseLog("stderr: %s", stderr)
		}
	}

	// Apply sleep if specified
	if err := applySleep(cmd.Sleep, verbose); err != nil {
		return StepResult{
			StepName: stepName,
			Status:   "failed",
			Error:    fmt.Sprintf("sleep error: %v", err),
		}
	}

	if err != nil || exitCode != 0 {
		errorMsg := fmt.Sprintf("command failed (exit %d)", exitCode)
		if err != nil {
			errorMsg = fmt.Sprintf("%s: %v", errorMsg, err)
		}
		if stderr != "" {
			errorMsg = fmt.Sprintf("%s\nstderr: %s", errorMsg, stderr)
		}

		return StepResult{
			StepName: stepName,
			Status:   "failed",
			Output:   stdout,
			Error:    errorMsg,
			ExitCode: exitCode,
		}
	}

	return StepResult{
		StepName: stepName,
		Status:   "success",
		Output:   stdout,
		ExitCode: exitCode,
	}
}

// executeCommandWithRetry executes a command with retry-until-success
func (e *Executor) executeCommandWithRetry(stepName string, cmd CommandStep, facts Facts) StepResult {
	// Interpolate command with facts
	command, err := e.interpolate(cmd.Command, facts)
	if err != nil {
		return StepResult{
			StepName: stepName,
			Status:   "failed",
			Error:    fmt.Sprintf("template error: %v", err),
		}
	}

	// Log command execution in verbose mode (use global verbose or step-specific)
	verbose := e.Verbose || cmd.Verbose
	if verbose {
		verboseLog("Executing command with retry: %s", command)
	}

	// Parse timeout configuration (default 60s if not specified)
	timeout := 60 * time.Second
	var customErrorCode *int

	if len(cmd.Timeout) > 0 {
		parsedTimeout, errCode, err := parseTimeoutConfig(cmd.Timeout)
		if err != nil {
			return StepResult{
				StepName: stepName,
				Status:   "failed",
				Error:    err.Error(),
			}
		}
		if parsedTimeout > 0 {
			timeout = parsedTimeout
		}
		customErrorCode = errCode
	}

	if verbose {
		verboseLog("Retry timeout: %s", timeout)
		if customErrorCode != nil {
			verboseLog("Custom timeout error code: %d", *customErrorCode)
		}
	}

	// Polling loop
	startTime := time.Now()
	deadline := startTime.Add(timeout)
	pollInterval := 1 * time.Second

	var lastStdout, lastErrorMsg string
	var lastExitCode int
	attemptNum := 0

	if verbose {
		verboseLog("Starting retry loop: polling every %s, timeout at %s", pollInterval, deadline.Format("15:04:05"))
	}

	for time.Now().Before(deadline) {
		attemptNum++
		stdout, stderr, exitCode, err := e.transport.Run(command)

		if verbose {
			remaining := time.Until(deadline).Round(time.Second)
			verboseLog("Retry attempt #%d - exit code: %d (timeout in %s)", attemptNum, exitCode, remaining)
		}

		// Success!
		if err == nil && exitCode == 0 {
			elapsed := time.Since(startTime).Round(time.Second)
			if verbose {
				verboseLog("✓ Retry succeeded after %d attempt(s) in %s", attemptNum, elapsed)
			}

			// Apply sleep after successful retry
			if sleepErr := applySleep(cmd.Sleep, verbose); sleepErr != nil {
				return StepResult{
					StepName: stepName,
					Status:   "failed",
					Error:    fmt.Sprintf("sleep error: %v", sleepErr),
				}
			}

			return StepResult{
				StepName: stepName,
				Status:   "success",
				Output:   fmt.Sprintf("Ready after %s\n%s", elapsed, stdout),
				ExitCode: exitCode,
			}
		}

		// Save last error for reporting
		lastStdout = stdout
		lastExitCode = exitCode

		if stderr != "" {
			lastErrorMsg = fmt.Sprintf("exit code %d: %s", exitCode, strings.TrimSpace(stderr))
		} else if err != nil {
			lastErrorMsg = fmt.Sprintf("exit code %d: %v", exitCode, err)
		} else {
			lastErrorMsg = fmt.Sprintf("exit code %d", exitCode)
		}

		// Wait before retrying
		time.Sleep(pollInterval)
	}

	// Timeout reached
	elapsed := time.Since(startTime).Round(time.Second)
	errorMsg := fmt.Sprintf("Timeout after %s\nLast error: %s", elapsed, lastErrorMsg)

	// Use custom error code if specified, otherwise use last exit code
	finalExitCode := lastExitCode
	if customErrorCode != nil {
		finalExitCode = *customErrorCode
	}

	return StepResult{
		StepName: stepName,
		Status:   "failed",
		Output:   lastStdout,
		Error:    errorMsg,
		ExitCode: finalExitCode,
	}
}

// executeCheckError executes a CheckErrorStep
func (e *Executor) executeCheckError(stepName string, check CheckErrorStep, facts Facts) StepResult {
	// Interpolate check command
	checkCmd, err := e.interpolate(check.Check, facts)
	if err != nil {
		return StepResult{
			StepName: stepName,
			Status:   "failed",
			Error:    fmt.Sprintf("template error: %v", err),
		}
	}

	if e.Verbose {
		verboseLog("Running check command: %s", checkCmd)
	}

	// Run the check
	stdout, _, exitCode, _ := e.transport.Run(checkCmd)

	if e.Verbose {
		verboseLog("Check command exit code: %d", exitCode)
	}

	if exitCode != 0 {
		// Check failed, return the error message
		return StepResult{
			StepName: stepName,
			Status:   "failed",
			Error:    check.Error,
		}
	}

	return StepResult{
		StepName: stepName,
		Status:   "success",
		Output:   stdout,
	}
}

// executeCheckRemediate executes a CheckRemediateStep
func (e *Executor) executeCheckRemediate(stepName string, checkRem CheckRemediateStep, facts Facts) StepResult {
	// Interpolate check command
	checkCmd, err := e.interpolate(checkRem.Check, facts)
	if err != nil {
		return StepResult{
			StepName: stepName,
			Status:   "failed",
			Error:    fmt.Sprintf("template error: %v", err),
		}
	}

	if e.Verbose {
		verboseLog("Running check command: %s", checkCmd)
	}

	// Run the check
	_, _, exitCode, _ := e.transport.Run(checkCmd)

	if e.Verbose {
		verboseLog("Check command exit code: %d", exitCode)
	}

	if exitCode == 0 {
		// Check passed, no remediation needed
		return StepResult{
			StepName: stepName,
			Status:   "success",
			Output:   "check passed, no remediation needed",
		}
	}

	// Check failed, run remediation steps
	if e.Verbose {
		verboseLog("Check failed, running %d remediation step(s)", len(checkRem.OnMissing))
	}

	remediationResults := []StepResult{}
	for _, remStep := range checkRem.OnMissing {
		remResult := e.executeRemediation(remStep, facts)
		remediationResults = append(remediationResults, remResult)

		// Stop on first remediation failure
		if remResult.Error != "" {
			return StepResult{
				StepName:         stepName,
				Status:           "failed",
				Error:            fmt.Sprintf("remediation failed: %s", remResult.Error),
				RemediationSteps: remediationResults,
			}
		}
	}

	// Re-run the check to verify remediation actually fixed the issue
	if e.Verbose {
		verboseLog("Re-running check to verify remediation: %s", checkCmd)
	}

	_, _, recheckExitCode, _ := e.transport.Run(checkCmd)

	if e.Verbose {
		verboseLog("Recheck exit code: %d", recheckExitCode)
	}

	if recheckExitCode != 0 {
		return StepResult{
			StepName:         stepName,
			Status:           "failed",
			Error:            "remediation completed but check still fails",
			RemediationSteps: remediationResults,
		}
	}

	return StepResult{
		StepName:         stepName,
		Status:           "success",
		Output:           "check failed, remediation completed and verified",
		RemediationSteps: remediationResults,
	}
}

// executeRemediation executes a RemediationStep
func (e *Executor) executeRemediation(remStep RemediationStep, facts Facts) StepResult {
	// Check if retry is enabled
	if remStep.Retry != nil && *remStep.Retry == "until" {
		return e.executeRemediationWithRetry(remStep, facts)
	}

	// Interpolate command
	command, err := e.interpolate(remStep.Command, facts)
	if err != nil {
		return StepResult{
			StepName: remStep.Name,
			Status:   "failed",
			Error:    fmt.Sprintf("template error: %v", err),
		}
	}

	// Log remediation execution in verbose mode (use global verbose or step-specific)
	verbose := e.Verbose || remStep.Verbose
	if verbose {
		verboseLog("Executing remediation command: %s", command)
	}

	// Run the command
	stdout, stderr, exitCode, err := e.transport.Run(command)

	if verbose {
		verboseLog("Remediation exit code: %d", exitCode)
		if stdout != "" {
			verboseLog("stdout: %s", stdout)
		}
		if stderr != "" {
			verboseLog("stderr: %s", stderr)
		}
	}

	// Apply sleep if specified
	if sleepErr := applySleep(remStep.Sleep, verbose); sleepErr != nil {
		return StepResult{
			StepName: remStep.Name,
			Status:   "failed",
			Error:    fmt.Sprintf("sleep error: %v", sleepErr),
		}
	}

	if err != nil || exitCode != 0 {
		errorMsg := fmt.Sprintf("remediation command failed (exit %d)", exitCode)
		if err != nil {
			errorMsg = fmt.Sprintf("%s: %v", errorMsg, err)
		}
		if stderr != "" {
			errorMsg = fmt.Sprintf("%s\nstderr: %s", errorMsg, stderr)
		}

		return StepResult{
			StepName: remStep.Name,
			Status:   "failed",
			Output:   stdout,
			Error:    errorMsg,
			ExitCode: exitCode,
		}
	}

	return StepResult{
		StepName: remStep.Name,
		Status:   "success",
		Output:   stdout,
		ExitCode: exitCode,
	}
}

// executeRemediationWithRetry executes a remediation step with retry-until-success
func (e *Executor) executeRemediationWithRetry(remStep RemediationStep, facts Facts) StepResult {
	// Interpolate command
	command, err := e.interpolate(remStep.Command, facts)
	if err != nil {
		return StepResult{
			StepName: remStep.Name,
			Status:   "failed",
			Error:    fmt.Sprintf("template error: %v", err),
		}
	}

	if remStep.Verbose {
		verboseLog("Executing remediation with retry: %s", command)
	}

	// Parse timeout configuration (default 60s if not specified)
	timeout := 60 * time.Second
	var customErrorCode *int

	if len(remStep.Timeout) > 0 {
		parsedTimeout, errCode, err := parseTimeoutConfig(remStep.Timeout)
		if err != nil {
			return StepResult{
				StepName: remStep.Name,
				Status:   "failed",
				Error:    err.Error(),
			}
		}
		if parsedTimeout > 0 {
			timeout = parsedTimeout
		}
		customErrorCode = errCode
	}

	if remStep.Verbose {
		verboseLog("Retry timeout: %s", timeout)
		if customErrorCode != nil {
			verboseLog("Custom timeout error code: %d", *customErrorCode)
		}
	}

	// Polling loop
	startTime := time.Now()
	deadline := startTime.Add(timeout)
	pollInterval := 1 * time.Second

	var lastStdout, lastErrorMsg string
	var lastExitCode int
	attemptNum := 0

	verbose := e.Verbose || remStep.Verbose
	if verbose {
		verboseLog("Starting remediation retry loop: polling every %s, timeout at %s", pollInterval, deadline.Format("15:04:05"))
	}

	for time.Now().Before(deadline) {
		attemptNum++
		stdout, stderr, exitCode, err := e.transport.Run(command)

		if verbose {
			remaining := time.Until(deadline).Round(time.Second)
			verboseLog("Remediation retry attempt #%d - exit code: %d (timeout in %s)", attemptNum, exitCode, remaining)
		}

		// Success!
		if err == nil && exitCode == 0 {
			elapsed := time.Since(startTime).Round(time.Second)
			if verbose {
				verboseLog("✓ Remediation retry succeeded after %d attempt(s) in %s", attemptNum, elapsed)
			}

			// Apply sleep after successful retry
			if sleepErr := applySleep(remStep.Sleep, verbose); sleepErr != nil {
				return StepResult{
					StepName: remStep.Name,
					Status:   "failed",
					Error:    fmt.Sprintf("sleep error: %v", sleepErr),
				}
			}

			return StepResult{
				StepName: remStep.Name,
				Status:   "success",
				Output:   fmt.Sprintf("Ready after %s\n%s", elapsed, stdout),
				ExitCode: exitCode,
			}
		}

		// Save last error for reporting
		lastStdout = stdout
		lastExitCode = exitCode

		if stderr != "" {
			lastErrorMsg = fmt.Sprintf("exit code %d: %s", exitCode, strings.TrimSpace(stderr))
		} else if err != nil {
			lastErrorMsg = fmt.Sprintf("exit code %d: %v", exitCode, err)
		} else {
			lastErrorMsg = fmt.Sprintf("exit code %d", exitCode)
		}

		// Wait before retrying
		time.Sleep(pollInterval)
	}

	// Timeout reached
	elapsed := time.Since(startTime).Round(time.Second)
	errorMsg := fmt.Sprintf("Timeout after %s\nLast error: %s", elapsed, lastErrorMsg)

	// Use custom error code if specified, otherwise use last exit code
	finalExitCode := lastExitCode
	if customErrorCode != nil {
		finalExitCode = *customErrorCode
	}

	return StepResult{
		StepName: remStep.Name,
		Status:   "failed",
		Output:   lastStdout,
		Error:    errorMsg,
		ExitCode: finalExitCode,
	}
}

// executeErrorOnly executes an ErrorOnlyStep
func (e *Executor) executeErrorOnly(stepName string, errStep ErrorOnlyStep) StepResult {
	return StepResult{
		StepName: stepName,
		Status:   "failed",
		Error:    errStep.Error,
	}
}

// interpolate applies fact values to a command template
func (e *Executor) interpolate(command string, facts Facts) (string, error) {
	if facts == nil {
		return command, nil
	}

	// Log template before interpolation in verbose mode
	if e.Verbose && strings.Contains(command, "{{") {
		verboseLog("Template before interpolation: %s", command)
		verboseLog("Available facts: %v", facts)
	}

	tmpl, err := template.New("command").Parse(command)
	if err != nil {
		if e.Verbose {
			verboseLog("Template parse error: %v", err)
		}
		return "", fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, facts); err != nil {
		if e.Verbose {
			verboseLog("Template execution error: %v", err)
			verboseLog("Available facts were: %v", facts)
		}
		return "", fmt.Errorf("template execution error (check fact names): %w", err)
	}

	result := buf.String()
	if e.Verbose && command != result {
		verboseLog("Template after interpolation: %s", result)
	}

	return result, nil
}

// emitEvent emits an execution event if a handler is configured
func (e *Executor) emitEvent(event ExecutionEvent) {
	// Always include execution context in events
	event.Context = e.context

	// Output as JSON if JSON mode is enabled
	if e.JSONOutput {
		jsonBytes, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			// Emit warning to stderr so it doesn't corrupt JSON stdout stream
			fmt.Fprintf(os.Stderr, "WARNING: Failed to marshal event to JSON: %v\n", err)
			fmt.Fprintf(os.Stderr, "Event details: step=%s, status=%s\n", event.StepName, event.Status)
		} else {
			fmt.Println(string(jsonBytes))
		}
	}

	if e.OnEvent != nil {
		e.OnEvent(event)
	}
}

// generateRunID generates a unique run ID
func generateRunID() string {
	return fmt.Sprintf("run-%d", time.Now().UnixNano())
}

// logStepMetadata logs all metadata fields from the step configuration
func (e *Executor) logStepMetadata(step InstallStep) {
	switch v := step.Step.(type) {
	case CommandStep:
		verboseLog("  Step type: CommandStep")
		if v.Message != nil {
			verboseLog("  Message: %s", *v.Message)
		}
		if v.Error != nil {
			verboseLog("  Custom error: %s", *v.Error)
		}
		if v.Retry != nil {
			verboseLog("  Retry: %s", *v.Retry)
		}
		if len(v.Timeout) > 0 {
			verboseLog("  Timeout: %s", string(v.Timeout))
		}
		if v.Sleep != nil {
			verboseLog("  Sleep: %s", *v.Sleep)
		}
		verboseLog("  Verbose: %v", v.Verbose)

	case CheckErrorStep:
		verboseLog("  Step type: CheckErrorStep")
		verboseLog("  Custom error: %s", v.Error)

	case CheckRemediateStep:
		verboseLog("  Step type: CheckRemediateStep")
		verboseLog("  Remediation steps: %d", len(v.OnMissing))
		if len(v.OnMissing) > 0 {
			for i, rem := range v.OnMissing {
				verboseLog("    [%d] %s", i+1, rem.Name)
				if rem.Error != nil {
					verboseLog("      Custom error: %s", *rem.Error)
				}
				if rem.Retry != nil {
					verboseLog("      Retry: %s", *rem.Retry)
				}
				if len(rem.Timeout) > 0 {
					verboseLog("      Timeout: %s", string(rem.Timeout))
				}
				if rem.Sleep != nil {
					verboseLog("      Sleep: %s", *rem.Sleep)
				}
				verboseLog("      Verbose: %v", rem.Verbose)
			}
		}

	case ErrorOnlyStep:
		verboseLog("  Step type: ErrorOnlyStep")
		verboseLog("  Error: %s", v.Error)
	}
}

// populateVerboseMetadata populates verbose metadata fields in an event
func (e *Executor) populateVerboseMetadata(event *ExecutionEvent, step InstallStep) {
	if !e.Verbose && !e.JSONOutput {
		return
	}

	switch v := step.Step.(type) {
	case CommandStep:
		event.StepType = "CommandStep"
		if v.Message != nil {
			event.Message = *v.Message
		}
		if v.Error != nil {
			event.CustomError = *v.Error
		}
		if v.Retry != nil {
			event.Retry = *v.Retry
		}
		if len(v.Timeout) > 0 {
			event.Timeout = string(v.Timeout)
		}
		if v.Sleep != nil {
			event.Sleep = *v.Sleep
		}

	case CheckErrorStep:
		event.StepType = "CheckErrorStep"
		event.CustomError = v.Error

	case CheckRemediateStep:
		event.StepType = "CheckRemediateStep"
		if len(v.OnMissing) > 0 {
			event.RemediationSteps = make([]RemediationStepInfo, len(v.OnMissing))
			for i, rem := range v.OnMissing {
				info := RemediationStepInfo{
					Name:    rem.Name,
					Verbose: rem.Verbose,
				}
				if rem.Error != nil {
					info.CustomError = *rem.Error
				}
				if rem.Retry != nil {
					info.Retry = *rem.Retry
				}
				if len(rem.Timeout) > 0 {
					info.Timeout = string(rem.Timeout)
				}
				if rem.Sleep != nil {
					info.Sleep = *rem.Sleep
				}
				event.RemediationSteps[i] = info
			}
		}

	case ErrorOnlyStep:
		event.StepType = "ErrorOnlyStep"
		event.CustomError = v.Error
	}
}

// StepResult represents the result of executing a step
type StepResult struct {
	StepName         string
	Status           string // "success", "failed", "skipped"
	Output           string
	Error            string
	ExitCode         int
	RemediationSteps []StepResult
}
