package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// Executor executes installation steps
type Executor struct {
	transport Transport
	DryRun    bool
	OnEvent   func(ExecutionEvent)
	runID     string
	context   ExecutionContext // Execution context (where commands run)
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

	return ctx
}

// GetContext returns the execution context
func (e *Executor) GetContext() ExecutionContext {
	return e.context
}

// ExecuteStep executes a single installation step
func (e *Executor) ExecuteStep(step InstallStep, facts Facts) StepResult {
	e.emitEvent(ExecutionEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		RunID:     e.runID,
		StepName:  step.Name,
		Status:    "running",
	})

	// Handle dry-run mode
	if e.DryRun {
		e.emitEvent(ExecutionEvent{
			Timestamp: time.Now().Format(time.RFC3339),
			RunID:     e.runID,
			StepName:  step.Name,
			Status:    "skipped",
			Output:    "(dry-run mode)",
		})
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
	e.emitEvent(ExecutionEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		RunID:     e.runID,
		StepName:  step.Name,
		Status:    status,
		Output:    result.Output,
		Error:     result.Error,
	})

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

	if cmd.Verbose {
		verboseLog("Executing command: %s", command)
	}

	// Run the command
	stdout, stderr, exitCode, err := e.transport.Run(command)

	if cmd.Verbose {
		verboseLog("Command exit code: %d", exitCode)
		if stdout != "" {
			verboseLog("stdout: %s", stdout)
		}
		if stderr != "" {
			verboseLog("stderr: %s", stderr)
		}
	}

	// Apply sleep if specified
	if err := applySleep(cmd.Sleep, cmd.Verbose); err != nil {
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

	if cmd.Verbose {
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

	if cmd.Verbose {
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

	for time.Now().Before(deadline) {
		stdout, stderr, exitCode, err := e.transport.Run(command)

		if cmd.Verbose {
			verboseLog("Retry attempt - exit code: %d", exitCode)
		}

		// Success!
		if err == nil && exitCode == 0 {
			elapsed := time.Since(startTime).Round(time.Second)

			// Apply sleep after successful retry
			if sleepErr := applySleep(cmd.Sleep, cmd.Verbose); sleepErr != nil {
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

	// Run the check
	stdout, _, exitCode, _ := e.transport.Run(checkCmd)

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

	// Run the check
	_, _, exitCode, _ := e.transport.Run(checkCmd)

	if exitCode == 0 {
		// Check passed, no remediation needed
		return StepResult{
			StepName: stepName,
			Status:   "success",
			Output:   "check passed, no remediation needed",
		}
	}

	// Check failed, run remediation steps
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
	_, _, recheckExitCode, _ := e.transport.Run(checkCmd)

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

	if remStep.Verbose {
		verboseLog("Executing remediation command: %s", command)
	}

	// Run the command
	stdout, stderr, exitCode, err := e.transport.Run(command)

	if remStep.Verbose {
		verboseLog("Remediation exit code: %d", exitCode)
		if stdout != "" {
			verboseLog("stdout: %s", stdout)
		}
		if stderr != "" {
			verboseLog("stderr: %s", stderr)
		}
	}

	// Apply sleep if specified
	if sleepErr := applySleep(remStep.Sleep, remStep.Verbose); sleepErr != nil {
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

	for time.Now().Before(deadline) {
		stdout, stderr, exitCode, err := e.transport.Run(command)

		if remStep.Verbose {
			verboseLog("Remediation retry attempt - exit code: %d", exitCode)
		}

		// Success!
		if err == nil && exitCode == 0 {
			elapsed := time.Since(startTime).Round(time.Second)

			// Apply sleep after successful retry
			if sleepErr := applySleep(remStep.Sleep, remStep.Verbose); sleepErr != nil {
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

	tmpl, err := template.New("command").Parse(command)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, facts); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// emitEvent emits an execution event if a handler is configured
func (e *Executor) emitEvent(event ExecutionEvent) {
	// Always include execution context in events
	event.Context = e.context

	if e.OnEvent != nil {
		e.OnEvent(event)
	}
}

// generateRunID generates a unique run ID
func generateRunID() string {
	return fmt.Sprintf("run-%d", time.Now().UnixNano())
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
