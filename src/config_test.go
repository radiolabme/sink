package main

import (
	"encoding/json"
	"os"
	"testing"
)

// TestConfigParsing tests parsing of the JSON config into our type-safe domain model
func TestConfigParsing(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "valid minimal config",
			json: `{
				"version": "1.0.0",
				"platforms": [{
					"os": "darwin",
					"match": "darwin*",
					"name": "macOS",
					"install_steps": [{
						"name": "Test step",
						"command": "echo test"
					}]
				}]
			}`,
			wantErr: false,
		},
		{
			name: "config with facts",
			json: `{
				"version": "1.0.0",
				"facts": {
					"os": {
						"command": "uname -s",
						"export": "SINK_OS"
					}
				},
				"platforms": [{
					"os": "darwin",
					"match": "darwin*",
					"name": "macOS",
					"install_steps": [{
						"name": "Test step",
						"command": "echo test"
					}]
				}]
			}`,
			wantErr: false,
		},
		{
			name: "config with check-error step",
			json: `{
				"version": "1.0.0",
				"platforms": [{
					"os": "darwin",
					"match": "darwin*",
					"name": "macOS",
					"install_steps": [{
						"name": "Check Homebrew",
						"check": "command -v brew",
						"error": "Homebrew required"
					}]
				}]
			}`,
			wantErr: false,
		},
		{
			name: "config with check-remediate step",
			json: `{
				"version": "1.0.0",
				"platforms": [{
					"os": "linux",
					"match": "linux*",
					"name": "Linux",
					"distributions": [{
						"ids": ["ubuntu"],
						"name": "Ubuntu",
						"install_steps": [{
							"name": "Check snapd",
							"check": "command -v snap",
							"on_missing": [{
								"name": "Install snapd",
								"command": "sudo apt-get install -y snapd"
							}]
						}]
					}]
				}]
			}`,
			wantErr: false,
		},
		{
			name: "missing required version",
			json: `{
				"platforms": []
			}`,
			wantErr: true,
		},
		{
			name: "empty platforms",
			json: `{
				"version": "1.0.0",
				"platforms": []
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := json.Unmarshal([]byte(tt.json), &config)

			if tt.wantErr {
				if err == nil {
					// Check if validation catches it
					if err = ValidateConfig(&config); err == nil {
						t.Errorf("expected error but got none")
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected parse error: %v", err)
					return
				}
				if err = ValidateConfig(&config); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

// TestConfigFromFile tests loading the actual config files
func TestConfigFromFile(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		wantErr  bool
		validate func(*testing.T, *Config)
	}{
		{
			name:    "original install-config.json",
			file:    "install-config.json",
			wantErr: false,
			validate: func(t *testing.T, c *Config) {
				if c.Version != "1.0.0" {
					t.Errorf("expected version 1.0.0, got %s", c.Version)
				}
				if len(c.Platforms) != 3 {
					t.Errorf("expected 3 platforms, got %d", len(c.Platforms))
				}
			},
		},
		{
			name:    "config with facts",
			file:    "install-config-with-facts.json",
			wantErr: false,
			validate: func(t *testing.T, c *Config) {
				if len(c.Facts) == 0 {
					t.Error("expected facts to be defined")
				}
				if _, ok := c.Facts["os"]; !ok {
					t.Error("expected 'os' fact to be defined")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if file exists
			if _, err := os.Stat(tt.file); os.IsNotExist(err) {
				t.Skipf("config file %s does not exist", tt.file)
			}

			config, err := LoadConfig(tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

// TestStepTypeEnforcement tests that step variants are properly enforced
func TestStepTypeEnforcement(t *testing.T) {
	// This test ensures that our type system prevents invalid step combinations
	// at compile time by using the StepVariant interface

	var _ StepVariant = CommandStep{}
	var _ StepVariant = CheckErrorStep{}
	var _ StepVariant = CheckRemediateStep{}
	var _ StepVariant = ErrorOnlyStep{}

	// The following should not compile (commented out):
	// var _ StepVariant = struct{}{}
	// var _ StepVariant = RemediationStep{}
}

// TestFactDefinitionValidation tests validation of fact definitions
func TestFactDefinitionValidation(t *testing.T) {
	tests := []struct {
		name     string
		factName string
		factDef  FactDef
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid fact with export",
			factName: "os_type",
			factDef: FactDef{
				Command: "uname -s",
				Export:  "SINK_OS",
			},
			wantErr: false,
		},
		{
			name:     "valid fact name with underscore start",
			factName: "_private_fact",
			factDef: FactDef{
				Command: "echo test",
			},
			wantErr: false,
		},
		{
			name:     "valid fact with all platforms",
			factName: "multi_platform",
			factDef: FactDef{
				Command:   "uname -s",
				Platforms: []string{"darwin", "linux", "windows"},
			},
			wantErr: false,
		},
		{
			name:     "valid fact with string type and transform",
			factName: "arch_type",
			factDef: FactDef{
				Command: "uname -m",
				Type:    "string",
				Transform: map[string]string{
					"x86_64": "amd64",
					"arm64":  "aarch64",
				},
			},
			wantErr: false,
		},
		{
			name:     "valid fact with transform and no explicit type",
			factName: "arch_type",
			factDef: FactDef{
				Command: "uname -m",
				Transform: map[string]string{
					"x86_64": "amd64",
				},
			},
			wantErr: false,
		},
		{
			name:     "valid fact with boolean type",
			factName: "is_macos",
			factDef: FactDef{
				Command: "test $(uname -s) = Darwin",
				Type:    "boolean",
			},
			wantErr: false,
		},
		{
			name:     "valid fact with integer type",
			factName: "cpu_count",
			factDef: FactDef{
				Command: "nproc",
				Type:    "integer",
			},
			wantErr: false,
		},
		{
			name:     "invalid fact name (starts with number)",
			factName: "1_fact",
			factDef: FactDef{
				Command: "echo test",
			},
			wantErr: true,
			errMsg:  "fact name must match pattern",
		},
		{
			name:     "invalid fact name (uppercase)",
			factName: "OS_TYPE",
			factDef: FactDef{
				Command: "uname -s",
			},
			wantErr: true,
			errMsg:  "fact name must match pattern",
		},
		{
			name:     "invalid fact name (has dash)",
			factName: "os-type",
			factDef: FactDef{
				Command: "uname -s",
			},
			wantErr: true,
			errMsg:  "fact name must match pattern",
		},
		{
			name:     "invalid export name (lowercase)",
			factName: "os_type",
			factDef: FactDef{
				Command: "uname -s",
				Export:  "sink_os",
			},
			wantErr: true,
			errMsg:  "export variable name must match",
		},
		{
			name:     "invalid export name (starts with number)",
			factName: "os_type",
			factDef: FactDef{
				Command: "uname -s",
				Export:  "1SINK_OS",
			},
			wantErr: true,
			errMsg:  "export variable name must match",
		},
		{
			name:     "invalid export name (has dash)",
			factName: "os_type",
			factDef: FactDef{
				Command: "uname -s",
				Export:  "SINK-OS",
			},
			wantErr: true,
			errMsg:  "export variable name must match",
		},
		{
			name:     "invalid platform",
			factName: "os_type",
			factDef: FactDef{
				Command:   "uname -s",
				Platforms: []string{"invalid-os"},
			},
			wantErr: true,
			errMsg:  "invalid platform",
		},
		{
			name:     "mixed valid and invalid platforms",
			factName: "os_type",
			factDef: FactDef{
				Command:   "uname -s",
				Platforms: []string{"darwin", "freebsd"},
			},
			wantErr: true,
			errMsg:  "invalid platform",
		},
		{
			name:     "empty command",
			factName: "test_fact",
			factDef: FactDef{
				Command: "",
			},
			wantErr: true,
			errMsg:  "command cannot be empty",
		},
		{
			name:     "whitespace-only command",
			factName: "test_fact",
			factDef: FactDef{
				Command: "   \t\n  ",
			},
			wantErr: true,
			errMsg:  "command cannot be empty",
		},
		{
			name:     "transform with boolean type",
			factName: "test_fact",
			factDef: FactDef{
				Command: "echo 1",
				Type:    "boolean",
				Transform: map[string]string{
					"1": "true",
				},
			},
			wantErr: true,
			errMsg:  "transform only allowed for string type",
		},
		{
			name:     "transform with integer type",
			factName: "test_fact",
			factDef: FactDef{
				Command: "echo 42",
				Type:    "integer",
				Transform: map[string]string{
					"42": "forty-two",
				},
			},
			wantErr: true,
			errMsg:  "transform only allowed for string type",
		},
		{
			name:     "invalid type",
			factName: "test_fact",
			factDef: FactDef{
				Command: "echo test",
				Type:    "float",
			},
			wantErr: true,
			errMsg:  "invalid type",
		},
		{
			name:     "invalid type (random string)",
			factName: "test_fact",
			factDef: FactDef{
				Command: "echo test",
				Type:    "whatever",
			},
			wantErr: true,
			errMsg:  "invalid type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFactDef(tt.factName, tt.factDef)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFactDef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

// TestValidatePlatform tests platform validation
func TestValidatePlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid platform with install steps",
			platform: Platform{
				OS:    "darwin",
				Match: "darwin*",
				Name:  "macOS",
				InstallSteps: []InstallStep{
					{
						Name: "Test",
						Step: CommandStep{Command: "echo test"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid platform with distributions",
			platform: Platform{
				OS:    "linux",
				Match: "linux*",
				Name:  "Linux",
				Distributions: []Distribution{
					{
						IDs:  []string{"ubuntu"},
						Name: "Ubuntu",
						InstallSteps: []InstallStep{
							{
								Name: "Test",
								Step: CommandStep{Command: "echo test"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing os",
			platform: Platform{
				Match: "darwin*",
				Name:  "macOS",
				InstallSteps: []InstallStep{
					{Name: "Test", Step: CommandStep{Command: "echo test"}},
				},
			},
			wantErr: true,
			errMsg:  "os is required",
		},
		{
			name: "missing match",
			platform: Platform{
				OS:   "darwin",
				Name: "macOS",
				InstallSteps: []InstallStep{
					{Name: "Test", Step: CommandStep{Command: "echo test"}},
				},
			},
			wantErr: true,
			errMsg:  "match pattern is required",
		},
		{
			name: "missing name",
			platform: Platform{
				OS:    "darwin",
				Match: "darwin*",
				InstallSteps: []InstallStep{
					{Name: "Test", Step: CommandStep{Command: "echo test"}},
				},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "no install steps and no distributions",
			platform: Platform{
				OS:    "darwin",
				Match: "darwin*",
				Name:  "macOS",
			},
			wantErr: true,
			errMsg:  "must have either install_steps or distributions",
		},
		{
			name: "both install steps and distributions",
			platform: Platform{
				OS:    "linux",
				Match: "linux*",
				Name:  "Linux",
				InstallSteps: []InstallStep{
					{Name: "Test", Step: CommandStep{Command: "echo test"}},
				},
				Distributions: []Distribution{
					{
						IDs:  []string{"ubuntu"},
						Name: "Ubuntu",
						InstallSteps: []InstallStep{
							{Name: "Test", Step: CommandStep{Command: "echo test"}},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "cannot have both install_steps and distributions",
		},
		{
			name: "distribution with no IDs",
			platform: Platform{
				OS:    "linux",
				Match: "linux*",
				Name:  "Linux",
				Distributions: []Distribution{
					{
						IDs:  []string{},
						Name: "Ubuntu",
						InstallSteps: []InstallStep{
							{Name: "Test", Step: CommandStep{Command: "echo test"}},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "at least one distribution ID is required",
		},
		{
			name: "distribution with no name",
			platform: Platform{
				OS:    "linux",
				Match: "linux*",
				Name:  "Linux",
				Distributions: []Distribution{
					{
						IDs: []string{"ubuntu"},
						InstallSteps: []InstallStep{
							{Name: "Test", Step: CommandStep{Command: "echo test"}},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "distribution with no install steps",
			platform: Platform{
				OS:    "linux",
				Match: "linux*",
				Name:  "Linux",
				Distributions: []Distribution{
					{
						IDs:          []string{"ubuntu"},
						Name:         "Ubuntu",
						InstallSteps: []InstallStep{},
					},
				},
			},
			wantErr: true,
			errMsg:  "at least one install step is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlatform(&tt.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePlatform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

// TestLoadConfig tests the LoadConfig function with file I/O
func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "test-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	validConfig := `{
		"version": "1.0.0",
		"platforms": [{
			"os": "darwin",
			"match": "darwin*",
			"name": "macOS",
			"install_steps": [{
				"name": "Test step",
				"command": "echo test"
			}]
		}]
	}`

	if _, err := tmpFile.Write([]byte(validConfig)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	tests := []struct {
		name     string
		filename string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid config file",
			filename: tmpFile.Name(),
			wantErr:  false,
		},
		{
			name:     "nonexistent file",
			filename: "/nonexistent/path/config.json",
			wantErr:  true,
			errMsg:   "failed to read config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadConfig(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("Expected config to be non-nil")
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

// TestLoadConfigInvalidJSON tests handling of malformed JSON
func TestLoadConfigInvalidJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	invalidJSON := `{
		"version": "1.0.0"
		"platforms": []
	}`

	if _, err := tmpFile.Write([]byte(invalidJSON)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	_, err = LoadConfig(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	if !contains(err.Error(), "failed to parse config") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

// TestLoadConfigValidationFailure tests that validation failures are caught
func TestLoadConfigValidationFailure(t *testing.T) {
	tests := []struct {
		name   string
		config string
		errMsg string
	}{
		{
			name: "no platforms",
			config: `{
				"version": "1.0.0",
				"platforms": []
			}`,
			errMsg: "at least one platform is required",
		},
		{
			name: "missing version",
			config: `{
				"platforms": [{
					"os": "darwin",
					"match": "darwin*",
					"name": "macOS",
					"install_steps": [{
						"name": "Test",
						"command": "echo test"
					}]
				}]
			}`,
			errMsg: "version is required",
		},
		{
			name: "invalid step structure",
			config: `{
				"version": "1.0.0",
				"platforms": [{
					"os": "darwin",
					"match": "darwin*",
					"name": "macOS",
					"install_steps": [{
						"name": "Invalid step"
					}]
				}]
			}`,
			errMsg: "unable to determine step variant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test-config-*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.Write([]byte(tt.config)); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			tmpFile.Close()

			_, err = LoadConfig(tmpFile.Name())
			if err == nil {
				t.Error("Expected validation error")
			} else if !contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

// TestParsePlatformStepsError tests error handling in parsePlatformSteps
func TestParsePlatformStepsError(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		wantErr  bool
		errMsg   string
	}{
		{
			name: "install step with nil Step variant",
			platform: Platform{
				OS:    "darwin",
				Match: "darwin*",
				Name:  "macOS",
				InstallSteps: []InstallStep{
					{
						Name: "Invalid step",
						Step: nil, // This simulates a parse error
					},
				},
			},
			wantErr: true,
			errMsg:  "step variant not parsed",
		},
		{
			name: "distribution install step with nil Step variant",
			platform: Platform{
				OS:    "linux",
				Match: "linux*",
				Name:  "Linux",
				Distributions: []Distribution{
					{
						IDs:  []string{"ubuntu"},
						Name: "Ubuntu",
						InstallSteps: []InstallStep{
							{
								Name: "Invalid step",
								Step: nil, // This simulates a parse error
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "distribution Ubuntu",
		},
		{
			name: "valid platform passes",
			platform: Platform{
				OS:    "darwin",
				Match: "darwin*",
				Name:  "macOS",
				InstallSteps: []InstallStep{
					{
						Name: "Valid step",
						Step: CommandStep{Command: "echo test"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parsePlatformSteps(&tt.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePlatformSteps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

// TestValidateConfigFactError tests that fact validation errors are caught
func TestValidateConfigFactError(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Facts: map[string]FactDef{
			"invalid_fact": {
				Command: "", // Empty command should fail
			},
		},
		Platforms: []Platform{
			{
				OS:    "darwin",
				Match: "darwin*",
				Name:  "macOS",
				InstallSteps: []InstallStep{
					{Name: "Test", Step: CommandStep{Command: "echo test"}},
				},
			},
		},
	}

	err := ValidateConfig(&config)
	if err == nil {
		t.Error("Expected validation error for invalid fact")
	}
	if !contains(err.Error(), "fact") && !contains(err.Error(), "command cannot be empty") {
		t.Errorf("Expected fact validation error, got: %v", err)
	}
}

// TestValidateConfigPlatformError tests that platform validation errors are caught
func TestValidateConfigPlatformError(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Platforms: []Platform{
			{
				OS:    "",
				Match: "darwin*",
				Name:  "macOS",
				InstallSteps: []InstallStep{
					{Name: "Test", Step: CommandStep{Command: "echo test"}},
				},
			},
		},
	}

	err := ValidateConfig(&config)
	if err == nil {
		t.Error("Expected validation error for invalid platform")
	}
	if !contains(err.Error(), "os is required") {
		t.Errorf("Expected platform validation error, got: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
