package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	// Valid export variable name: starts with A-Z or _, followed by A-Z, 0-9, or _
	exportVarRegex = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

	// Valid fact name: starts with a-z or _, followed by a-z, 0-9, or _
	factNameRegex = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

	// Valid platforms
	validPlatforms = map[string]bool{
		"darwin":  true,
		"linux":   true,
		"windows": true,
	}
)

// LoadConfig loads and validates a configuration from a JSON file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Parse install steps into type-safe variants
	for i := range config.Platforms {
		if err := parsePlatformSteps(&config.Platforms[i]); err != nil {
			return nil, fmt.Errorf("platform %s: %w", config.Platforms[i].Name, err)
		}
	}

	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// parsePlatformSteps converts raw JSON install steps into typed StepVariant
func parsePlatformSteps(platform *Platform) error {
	// Parse direct install steps
	if len(platform.InstallSteps) > 0 {
		for i := range platform.InstallSteps {
			if err := parseInstallStep(&platform.InstallSteps[i]); err != nil {
				return fmt.Errorf("install_step[%d]: %w", i, err)
			}
		}
	}

	// Parse distribution install steps
	for di := range platform.Distributions {
		dist := &platform.Distributions[di]
		for si := range dist.InstallSteps {
			if err := parseInstallStep(&dist.InstallSteps[si]); err != nil {
				return fmt.Errorf("distribution %s, install_step[%d]: %w", dist.Name, si, err)
			}
		}
	}

	return nil
}

// parseInstallStep determines the step variant from raw JSON
// This is where we enforce the "impossible states" at parse time
func parseInstallStep(step *InstallStep) error {
	// Step variant should already be parsed by UnmarshalJSON
	// This is just for additional validation if needed
	if step.Step == nil {
		return fmt.Errorf("step variant not parsed for step '%s'", step.Name)
	}
	return nil
}

// ValidateConfig validates the entire configuration
func ValidateConfig(config *Config) error {
	// Validate version is present
	if config.Version == "" {
		return fmt.Errorf("version is required")
	}

	// Validate platforms
	if len(config.Platforms) == 0 {
		return fmt.Errorf("at least one platform is required")
	}

	// Validate facts
	for name, factDef := range config.Facts {
		if err := ValidateFactDef(name, factDef); err != nil {
			return fmt.Errorf("fact '%s': %w", name, err)
		}
	}

	// Validate each platform
	for i, platform := range config.Platforms {
		if err := validatePlatform(&platform); err != nil {
			return fmt.Errorf("platform[%d] %s: %w", i, platform.Name, err)
		}
	}

	// TODO: Validate template references
	// TODO: Detect circular dependencies in facts

	return nil
}

// ValidateFactDef validates a single fact definition
func ValidateFactDef(name string, factDef FactDef) error {
	// Validate fact name
	if !factNameRegex.MatchString(name) {
		return fmt.Errorf("fact name must match pattern ^[a-z_][a-z0-9_]*$")
	}

	// Validate command is not empty
	if strings.TrimSpace(factDef.Command) == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Validate export variable name if specified
	if factDef.Export != "" && !exportVarRegex.MatchString(factDef.Export) {
		return fmt.Errorf("export variable name must match pattern ^[A-Z_][A-Z0-9_]*$")
	}

	// Validate platforms
	for _, platform := range factDef.Platforms {
		if !validPlatforms[platform] {
			return fmt.Errorf("invalid platform '%s', must be one of: darwin, linux, windows", platform)
		}
	}

	// Validate transform only works with string or unspecified type
	if len(factDef.Transform) > 0 {
		if factDef.Type != "" && factDef.Type != "string" {
			return fmt.Errorf("transform only allowed for string type, got type '%s'", factDef.Type)
		}
	}

	// Validate type if specified
	if factDef.Type != "" {
		validTypes := map[string]bool{
			"string":  true,
			"boolean": true,
			"integer": true,
		}
		if !validTypes[factDef.Type] {
			return fmt.Errorf("invalid type '%s', must be one of: string, boolean, integer", factDef.Type)
		}
	}

	return nil
}

// validatePlatform validates a single platform
func validatePlatform(platform *Platform) error {
	if platform.OS == "" {
		return fmt.Errorf("os is required")
	}
	if platform.Match == "" {
		return fmt.Errorf("match pattern is required")
	}
	if platform.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Platform must have either install_steps or distributions, not both
	hasSteps := len(platform.InstallSteps) > 0
	hasDists := len(platform.Distributions) > 0

	if !hasSteps && !hasDists {
		return fmt.Errorf("platform must have either install_steps or distributions")
	}
	if hasSteps && hasDists {
		return fmt.Errorf("platform cannot have both install_steps and distributions")
	}

	// Validate distributions if present
	for i, dist := range platform.Distributions {
		if err := validateDistribution(&dist); err != nil {
			return fmt.Errorf("distribution[%d] %s: %w", i, dist.Name, err)
		}
	}

	return nil
}

// validateDistribution validates a single distribution
func validateDistribution(dist *Distribution) error {
	if len(dist.IDs) == 0 {
		return fmt.Errorf("at least one distribution ID is required")
	}
	if dist.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(dist.InstallSteps) == 0 {
		return fmt.Errorf("at least one install step is required")
	}
	return nil
}
