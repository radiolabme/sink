package main

import (
	"os"
	"strings"
	"testing"
)

// TestValidateConfig tests the ValidateConfig function
func TestValidateConfig_ValidConfig(t *testing.T) {
	config := &Config{
		Version:     "1.0",
		Description: "Test Config",
		Facts:       map[string]FactDef{},
		Platforms: []Platform{
			{
				Name:  "Test Platform",
				OS:    "darwin",
				Match: ".*",
				InstallSteps: []InstallStep{
					{
						Name: "Test Step",
						Step: CommandStep{
							Command: "echo test",
						},
					},
				},
			},
		},
	}

	err := ValidateConfig(config)
	if err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}
}

func TestValidateConfig_MissingVersion(t *testing.T) {
	config := &Config{
		Version: "",
		Platforms: []Platform{
			{Name: "Test", OS: "darwin", Match: ".*", InstallSteps: []InstallStep{{Name: "Test", Step: CommandStep{Command: "test"}}}},
		},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for missing version, got nil")
	}
	if !strings.Contains(err.Error(), "version is required") {
		t.Errorf("Expected 'version is required' error, got: %v", err)
	}
}

func TestValidateConfig_NoPlatforms(t *testing.T) {
	config := &Config{
		Version:   "1.0",
		Platforms: []Platform{},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for no platforms, got nil")
	}
	if !strings.Contains(err.Error(), "at least one platform is required") {
		t.Errorf("Expected 'at least one platform is required' error, got: %v", err)
	}
}

func TestValidateConfig_InvalidFact(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Facts: map[string]FactDef{
			"InvalidName!": {
				Command: "echo test",
			},
		},
		Platforms: []Platform{
			{Name: "Test", OS: "darwin", Match: ".*", InstallSteps: []InstallStep{{Name: "Test", Step: CommandStep{Command: "test"}}}},
		},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid fact name, got nil")
	}
	if !strings.Contains(err.Error(), "fact 'InvalidName!'") {
		t.Errorf("Expected fact name error, got: %v", err)
	}
}

func TestValidateConfig_InvalidPlatform(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Platforms: []Platform{
			{
				Name:         "",
				OS:           "darwin",
				InstallSteps: []InstallStep{},
			},
		},
	}

	err := ValidateConfig(config)
	if err == nil {
		t.Error("Expected error for invalid platform, got nil")
	}
}

// TestValidateFactDef tests the ValidateFactDef function
func TestValidateFactDef_ValidFact(t *testing.T) {
	tests := []struct {
		name     string
		factName string
		factDef  FactDef
	}{
		{
			name:     "Simple command fact",
			factName: "os_name",
			factDef: FactDef{
				Command: "uname -s",
			},
		},
		{
			name:     "Fact with export",
			factName: "home_dir",
			factDef: FactDef{
				Command: "echo $HOME",
				Export:  "HOME_DIR",
			},
		},

		{
			name:     "Fact with platform filter",
			factName: "package_manager",
			factDef: FactDef{
				Command:   "which apt",
				Platforms: []string{"linux"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFactDef(tt.factName, tt.factDef)
			if err != nil {
				t.Errorf("Expected valid fact to pass, got error: %v", err)
			}
		})
	}
}

func TestValidateFactDef_InvalidFactName(t *testing.T) {
	tests := []struct {
		name     string
		factName string
	}{
		{"Uppercase", "OSName"},
		{"Hyphen", "os-name"},
		{"Space", "os name"},
		{"Special char", "os$name"},
		{"Starts with number", "1os"},
		{"Empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factDef := FactDef{Command: "echo test"}
			err := ValidateFactDef(tt.factName, factDef)
			if err == nil {
				t.Errorf("Expected error for invalid fact name '%s', got nil", tt.factName)
			}
			if !strings.Contains(err.Error(), "fact name must match pattern") {
				t.Errorf("Expected pattern error, got: %v", err)
			}
		})
	}
}

func TestValidateFactDef_EmptyCommand(t *testing.T) {
	factDef := FactDef{
		Command: "",
	}

	err := ValidateFactDef("test_fact", factDef)
	if err == nil {
		t.Error("Expected error for empty command, got nil")
	}
	if !strings.Contains(err.Error(), "command cannot be empty") {
		t.Errorf("Expected 'command cannot be empty' error, got: %v", err)
	}
}

func TestValidateFactDef_WhitespaceCommand(t *testing.T) {
	factDef := FactDef{
		Command: "   \n\t  ",
	}

	err := ValidateFactDef("test_fact", factDef)
	if err == nil {
		t.Error("Expected error for whitespace-only command, got nil")
	}
}

func TestValidateFactDef_InvalidExportName(t *testing.T) {
	tests := []struct {
		name       string
		exportName string
	}{
		{"Lowercase", "home_dir"},
		{"Hyphen", "HOME-DIR"},
		{"Starts with number", "1HOME"},
		{"Special char", "HOME$DIR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factDef := FactDef{
				Command: "echo test",
				Export:  tt.exportName,
			}
			err := ValidateFactDef("test_fact", factDef)
			if err == nil {
				t.Errorf("Expected error for invalid export name '%s', got nil", tt.exportName)
			}
			if !strings.Contains(err.Error(), "export variable name must match pattern") {
				t.Errorf("Expected export pattern error, got: %v", err)
			}
		})
	}
}

func TestValidateFactDef_InvalidPlatform(t *testing.T) {
	factDef := FactDef{
		Command:   "echo test",
		Platforms: []string{"invalid_os"},
	}

	err := ValidateFactDef("test_fact", factDef)
	if err == nil {
		t.Error("Expected error for invalid platform, got nil")
	}
	if !strings.Contains(err.Error(), "invalid platform") {
		t.Errorf("Expected invalid platform error, got: %v", err)
	}
}

func TestValidateFactDef_InvalidTransform(t *testing.T) {
	// Transform only allowed for string or unspecified type
	factDef := FactDef{
		Command: "echo test",
		Type:    "boolean",
		Transform: map[string]string{
			"yes": "true",
		},
	}

	err := ValidateFactDef("test_fact", factDef)
	if err == nil {
		t.Error("Expected error for transform with non-string type, got nil")
	}
	if !strings.Contains(err.Error(), "transform only allowed for string type") {
		t.Errorf("Expected transform type error, got: %v", err)
	}
}

// TestValidatePlatform tests the validatePlatform function
func TestValidatePlatform_ValidPlatform(t *testing.T) {
	platform := &Platform{
		Name:  "macOS",
		OS:    "darwin",
		Match: ".*",
		InstallSteps: []InstallStep{
			{
				Name: "Test Step",
				Step: CommandStep{
					Command: "which brew && brew install foo",
				},
			},
		},
	}

	err := validatePlatform(platform)
	if err != nil {
		t.Errorf("Expected valid platform to pass, got error: %v", err)
	}
}

func TestValidatePlatform_MissingName(t *testing.T) {
	platform := &Platform{
		Name:         "",
		OS:           "darwin",
		Match:        ".*",
		InstallSteps: []InstallStep{{Name: "Test", Step: CommandStep{Command: "test"}}},
	}

	err := validatePlatform(platform)
	if err == nil {
		t.Error("Expected error for missing platform name, got nil")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("Expected 'name is required' error, got: %v", err)
	}
}

func TestValidatePlatform_MissingOS(t *testing.T) {
	platform := &Platform{
		Name:         "Test Platform",
		OS:           "",
		Match:        ".*",
		InstallSteps: []InstallStep{{Name: "Test", Step: CommandStep{Command: "test"}}},
	}

	err := validatePlatform(platform)
	if err == nil {
		t.Error("Expected error for missing OS, got nil")
	}
	if !strings.Contains(err.Error(), "os is required") {
		t.Errorf("Expected 'os is required' error, got: %v", err)
	}
}

func TestValidatePlatform_MissingMatch(t *testing.T) {
	platform := &Platform{
		Name:         "Test Platform",
		OS:           "darwin",
		Match:        "",
		InstallSteps: []InstallStep{{Name: "Test", Step: CommandStep{Command: "test"}}},
	}

	err := validatePlatform(platform)
	if err == nil {
		t.Error("Expected error for missing Match, got nil")
	}
	if !strings.Contains(err.Error(), "match pattern is required") {
		t.Errorf("Expected 'match pattern is required' error, got: %v", err)
	}
}

func TestValidatePlatform_NoSteps(t *testing.T) {
	platform := &Platform{
		Name:         "Test Platform",
		OS:           "darwin",
		Match:        ".*",
		InstallSteps: []InstallStep{},
	}

	err := validatePlatform(platform)
	if err == nil {
		t.Error("Expected error for no steps, got nil")
	}
	if !strings.Contains(err.Error(), "must have either install_steps or distributions") {
		t.Errorf("Expected 'must have either install_steps or distributions' error, got: %v", err)
	}
}

func TestValidatePlatform_WithDistributions(t *testing.T) {
	platform := &Platform{
		Name:  "Linux",
		OS:    "linux",
		Match: ".*",
		Distributions: []Distribution{
			{
				Name:         "Ubuntu",
				IDs:          []string{"ubuntu"},
				InstallSteps: []InstallStep{{Name: "Test", Step: CommandStep{Command: "apt install foo"}}},
			},
		},
	}

	err := validatePlatform(platform)
	if err != nil {
		t.Errorf("Expected valid platform with distributions to pass, got error: %v", err)
	}
}

func TestValidatePlatform_InvalidDistribution(t *testing.T) {
	platform := &Platform{
		Name:  "Linux",
		OS:    "linux",
		Match: ".*",
		Distributions: []Distribution{
			{
				Name:         "",
				IDs:          []string{},
				InstallSteps: []InstallStep{{Name: "Test", Step: CommandStep{Command: "test"}}},
			},
		},
	}

	err := validatePlatform(platform)
	if err == nil {
		t.Error("Expected error for invalid distribution, got nil")
	}
}

// TestValidateDistribution tests the validateDistribution function
func TestValidateDistribution_ValidDistribution(t *testing.T) {
	dist := &Distribution{
		Name: "Ubuntu",
		IDs:  []string{"ubuntu"},
		InstallSteps: []InstallStep{
			{
				Name: "Install package",
				Step: CommandStep{Command: "apt install foo"},
			},
		},
	}

	err := validateDistribution(dist)
	if err != nil {
		t.Errorf("Expected valid distribution to pass, got error: %v", err)
	}
}

func TestValidateDistribution_MissingName(t *testing.T) {
	dist := &Distribution{
		Name:         "",
		IDs:          []string{"ubuntu"},
		InstallSteps: []InstallStep{{Name: "Test", Step: CommandStep{Command: "test"}}},
	}

	err := validateDistribution(dist)
	if err == nil {
		t.Error("Expected error for missing distribution name, got nil")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("Expected 'name is required' error, got: %v", err)
	}
}

func TestValidateDistribution_NoSteps(t *testing.T) {
	dist := &Distribution{
		Name:         "Ubuntu",
		IDs:          []string{"ubuntu"},
		InstallSteps: []InstallStep{},
	}

	err := validateDistribution(dist)
	if err == nil {
		t.Error("Expected error for no steps, got nil")
	}
	if !strings.Contains(err.Error(), "at least one install step is required") {
		t.Errorf("Expected 'at least one install step is required' error, got: %v", err)
	}
}

func TestValidateDistribution_NoIDs(t *testing.T) {
	dist := &Distribution{
		Name:         "Ubuntu",
		IDs:          []string{},
		InstallSteps: []InstallStep{{Name: "Test", Step: CommandStep{Command: "test"}}},
	}

	err := validateDistribution(dist)
	if err == nil {
		t.Error("Expected error for no IDs, got nil")
	}
	if !strings.Contains(err.Error(), "at least one distribution ID is required") {
		t.Errorf("Expected 'at least one distribution ID is required' error, got: %v", err)
	}
}

// TestLoadConfig_CompleteValidation tests loading a valid configuration
func TestLoadConfig_CompleteValidation(t *testing.T) {
	// Create a temporary valid config file
	validConfig := `{
		"version": "1.0",
		"name": "Test Config",
		"facts": {
			"os_type": {
				"command": "uname -s"
			}
		},
		"platforms": [{
			"name": "macOS",
			"os": "darwin",
			"match": ".*",
			"install_steps": [{
				"name": "Check Homebrew",
				"command": "which brew || echo 'Install brew'"
			}]
		}]
	}`

	tmpFile, err := os.CreateTemp("", "test-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(validConfig); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Errorf("Expected valid config to load, got error: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", config.Version)
	}

	if len(config.Platforms) != 1 {
		t.Errorf("Expected 1 platform, got %d", len(config.Platforms))
	}
}
