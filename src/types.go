package main

import (
	"encoding/json"
	"fmt"
)

// Config represents the top-level configuration
type Config struct {
	Schema      string             `json:"$schema,omitempty"`
	Name        string             `json:"name,omitempty"`
	Version     string             `json:"version"`
	Description string             `json:"description,omitempty"`
	Facts       map[string]FactDef `json:"facts,omitempty"`
	Defaults    map[string]string  `json:"defaults,omitempty"`
	Platforms   []Platform         `json:"platforms"`
	Fallback    *Fallback          `json:"fallback,omitempty"`
}

// FactDef defines how to gather a single fact
type FactDef struct {
	Command     string            `json:"command"`
	Description string            `json:"description,omitempty"`
	Export      string            `json:"export,omitempty"`
	Platforms   []string          `json:"platforms,omitempty"`
	Type        string            `json:"type,omitempty"` // "string", "boolean", "integer"
	Transform   map[string]string `json:"transform,omitempty"`
	Strict      bool              `json:"strict,omitempty"`
	Required    bool              `json:"required,omitempty"`
	Timeout     json.RawMessage   `json:"timeout,omitempty"` // Can be string or TimeoutConfig object
	Sleep       *string           `json:"sleep,omitempty"`   // Duration string like "1s", "500ms"
	Verbose     bool              `json:"verbose,omitempty"`
}

// Platform represents a platform configuration
// Uses json.RawMessage to defer parsing of variant fields
type Platform struct {
	OS            string         `json:"os"`
	Match         string         `json:"match"`
	Name          string         `json:"name"`
	RequiredTools []string       `json:"required_tools,omitempty"`
	InstallSteps  []InstallStep  `json:"install_steps,omitempty"`
	Distributions []Distribution `json:"distributions,omitempty"`
	Fallback      *Fallback      `json:"fallback,omitempty"`
}

// Distribution represents a Linux distribution configuration
type Distribution struct {
	IDs          []string      `json:"ids"`
	Name         string        `json:"name"`
	InstallSteps []InstallStep `json:"install_steps"`
}

// InstallStep represents a single installation step
// The Step field contains the variant (one of the Step* types)
type InstallStep struct {
	Name string
	Step StepVariant
}

// UnmarshalJSON implements custom JSON unmarshaling for InstallStep
func (is *InstallStep) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map to inspect fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract name
	name, _ := raw["name"].(string)
	is.Name = name

	// Determine which variant based on fields present
	_, hasCommand := raw["command"]
	_, hasCheck := raw["check"]
	_, hasOnMissing := raw["on_missing"]
	errorVal, hasError := raw["error"]

	if hasCommand {
		// CommandStep
		var cmd CommandStep
		if err := json.Unmarshal(data, &cmd); err != nil {
			return err
		}
		is.Step = cmd
	} else if hasCheck && hasOnMissing {
		// CheckRemediateStep
		var cr CheckRemediateStep
		if err := json.Unmarshal(data, &cr); err != nil {
			return err
		}
		is.Step = cr
	} else if hasCheck && hasError {
		// CheckErrorStep
		var ce CheckErrorStep
		if err := json.Unmarshal(data, &ce); err != nil {
			return err
		}
		is.Step = ce
	} else if hasError && errorVal != nil {
		// ErrorOnlyStep
		var eo ErrorOnlyStep
		if err := json.Unmarshal(data, &eo); err != nil {
			return err
		}
		is.Step = eo
	} else {
		return fmt.Errorf("unable to determine step variant for step '%s'", name)
	}

	return nil
}

// StepVariant is a sealed union of possible step types
type StepVariant interface {
	isStep()
}

// CommandStep executes a command
type CommandStep struct {
	Command string
	Message *string
	Error   *string
	Retry   *string         // "until" = retry until success or timeout
	Timeout json.RawMessage // Can be string or TimeoutConfig object
	Sleep   *string         // Duration string like "1s", "500ms"
	Verbose bool            // Enable verbose output
}

func (CommandStep) isStep() {}

// CheckErrorStep checks a condition and fails with error if not met
type CheckErrorStep struct {
	Check string
	Error string
}

func (CheckErrorStep) isStep() {}

// CheckRemediateStep checks a condition and runs remediation if not met
type CheckRemediateStep struct {
	Check     string            `json:"check"`
	OnMissing []RemediationStep `json:"on_missing"`
}

func (CheckRemediateStep) isStep() {}

// ErrorOnlyStep always fails with an error message
type ErrorOnlyStep struct {
	Error string
}

func (ErrorOnlyStep) isStep() {}

// RemediationStep is a step that runs during remediation
type RemediationStep struct {
	Name    string
	Command string
	Error   *string
	Retry   *string         // "until" = retry until success or timeout
	Timeout json.RawMessage // Can be string or TimeoutConfig object
	Sleep   *string         // Duration string like "1s", "500ms"
	Verbose bool            // Enable verbose output
}

// TimeoutConfig represents advanced timeout configuration
type TimeoutConfig struct {
	Interval  string `json:"interval"`             // Duration string like "30s", "5m"
	ErrorCode *int   `json:"error_code,omitempty"` // Custom exit code on timeout
}

// ParseTimeout parses a timeout field that can be either a string or TimeoutConfig object
func ParseTimeout(raw json.RawMessage) (interval string, errorCode *int, err error) {
	if len(raw) == 0 {
		return "", nil, nil
	}

	// Try to unmarshal as string first
	var timeoutStr string
	if err := json.Unmarshal(raw, &timeoutStr); err == nil {
		return timeoutStr, nil, nil
	}

	// Try to unmarshal as TimeoutConfig object
	var timeoutCfg TimeoutConfig
	if err := json.Unmarshal(raw, &timeoutCfg); err != nil {
		return "", nil, fmt.Errorf("timeout must be a string or object with interval: %w", err)
	}

	return timeoutCfg.Interval, timeoutCfg.ErrorCode, nil
}

// Fallback represents a fallback error message
type Fallback struct {
	Error string `json:"error"`
}

// Facts represents gathered system facts
type Facts map[string]interface{}

// ExecutionContext represents the environment where commands are executed
type ExecutionContext struct {
	Host      string `json:"host"`      // Hostname where commands run
	User      string `json:"user"`      // User running commands
	WorkDir   string `json:"work_dir"`  // Current working directory
	OS        string `json:"os"`        // Operating system (uname -s)
	Arch      string `json:"arch"`      // Architecture (uname -m)
	Transport string `json:"transport"` // "local" or "ssh:user@host"
	Timestamp string `json:"timestamp"` // When context was captured
}

// ExecutionEvent represents an event during execution
type ExecutionEvent struct {
	Timestamp string           `json:"timestamp"`
	RunID     string           `json:"run_id"`
	StepName  string           `json:"step_name"`
	Status    string           `json:"status"` // "running", "success", "failed", "skipped"
	Output    string           `json:"output,omitempty"`
	Error     string           `json:"error,omitempty"`
	Context   ExecutionContext `json:"context"` // Execution context for this event

	// Verbose metadata (populated when verbose mode is enabled)
	StepType         string                `json:"step_type,omitempty"`         // Type of step (CommandStep, CheckRemediateStep, etc.)
	Command          string                `json:"command,omitempty"`           // Command being executed
	ExitCode         *int                  `json:"exit_code,omitempty"`         // Command exit code
	Stdout           string                `json:"stdout,omitempty"`            // Standard output
	Stderr           string                `json:"stderr,omitempty"`            // Standard error
	Message          string                `json:"message,omitempty"`           // Step message
	CustomError      string                `json:"custom_error,omitempty"`      // Custom error message
	Retry            string                `json:"retry,omitempty"`             // Retry configuration
	Timeout          string                `json:"timeout,omitempty"`           // Timeout configuration
	Sleep            string                `json:"sleep,omitempty"`             // Sleep duration
	RemediationSteps []RemediationStepInfo `json:"remediation_steps,omitempty"` // Remediation step metadata
}

// RemediationStepInfo represents metadata about a remediation step
type RemediationStepInfo struct {
	Name        string `json:"name"`
	CustomError string `json:"custom_error,omitempty"`
	Retry       string `json:"retry,omitempty"`
	Timeout     string `json:"timeout,omitempty"`
	Sleep       string `json:"sleep,omitempty"`
	Verbose     bool   `json:"verbose"`
}

// ExecutionResult represents the result of a full execution
type ExecutionResult struct {
	RunID     string           `json:"run_id"`
	Success   bool             `json:"success"`
	Events    []ExecutionEvent `json:"events"`
	Facts     Facts            `json:"facts,omitempty"`
	StartTime string           `json:"start_time"`
	EndTime   string           `json:"end_time"`
	Error     string           `json:"error,omitempty"`
}
