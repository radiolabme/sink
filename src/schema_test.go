package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSchemaEmbed tests that the schema is embedded correctly
func TestSchemaEmbed(t *testing.T) {
	if embeddedSchema == "" {
		t.Fatal("embeddedSchema is empty - schema not embedded")
	}

	// Verify it's valid JSON
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(embeddedSchema), &schema); err != nil {
		t.Fatalf("embedded schema is not valid JSON: %v", err)
	}

	// Verify required fields
	requiredFields := []string{"$schema", "$id", "title", "description", "type", "properties"}
	for _, field := range requiredFields {
		if _, ok := schema[field]; !ok {
			t.Errorf("schema missing required field: %s", field)
		}
	}

	// Verify $id points to correct URL
	if id, ok := schema["$id"].(string); ok {
		expectedID := "https://raw.githubusercontent.com/radiolabme/sink/main/src/sink.schema.json"
		if id != expectedID {
			t.Errorf("schema $id = %q, want %q", id, expectedID)
		}
	} else {
		t.Error("schema $id is not a string")
	}

	// Verify title
	if title, ok := schema["title"].(string); ok {
		if !strings.Contains(title, "Sink") {
			t.Errorf("schema title doesn't contain 'Sink': %q", title)
		}
	}
}

// TestSchemaValidation tests schema validation with jq
// This test installs jq to a temp directory and uses it to validate the schema
func TestSchemaValidation(t *testing.T) {
	// Skip if we're in a CI environment without package manager access
	if os.Getenv("CI") == "true" && runtime.GOOS == "linux" {
		t.Skip("Skipping jq installation test in CI")
	}

	// Create temp directory for installation
	tempDir, err := os.MkdirTemp("", "sink-schema-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	jqBin := filepath.Join(tempDir, "bin", "jq")

	// Create config to install jq
	configContent := createJqInstallConfig(tempDir)

	// Write config to temp file
	configFile := filepath.Join(tempDir, "install-jq.json")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Build the binary if it doesn't exist
	binaryPath := filepath.Join("..", "bin", "sink")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Log("Building sink binary...")
		cmd := exec.Command("go", "build", "-o", binaryPath, ".")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to build binary: %v\n%s", err, output)
		}
	}

	// Execute the config to install jq (with confirmation bypass)
	t.Log("Installing jq to temp directory...")
	cmd := exec.Command(binaryPath, "execute", configFile)
	cmd.Stdin = strings.NewReader("yes\n") // Auto-confirm
	cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s/bin:%s", tempDir, os.Getenv("PATH")))

	output, err := cmd.CombinedOutput()
	t.Logf("Install output:\n%s", output)

	if err != nil {
		// If jq installation fails (e.g., no internet), skip the rest
		if strings.Contains(string(output), "curl") || strings.Contains(string(output), "brew") {
			t.Skipf("jq installation failed (possibly no internet): %v", err)
		}
		t.Fatalf("failed to install jq: %v", err)
	}

	// Verify jq was installed
	if _, err := os.Stat(jqBin); os.IsNotExist(err) {
		t.Fatalf("jq binary not found at %s after installation", jqBin)
	}

	// Make jq executable (in case it's not)
	if err := os.Chmod(jqBin, 0755); err != nil {
		t.Fatalf("failed to make jq executable: %v", err)
	}

	// Test 1: Emit schema and validate it's valid JSON with jq
	t.Log("Validating schema with jq...")
	schemaCmd := exec.Command(binaryPath, "schema")
	jqCmd := exec.Command(jqBin, ".")

	pipe, err := schemaCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	jqCmd.Stdin = pipe

	if err := schemaCmd.Start(); err != nil {
		t.Fatalf("failed to start schema command: %v", err)
	}

	jqOutput, err := jqCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("jq validation failed: %v\nOutput: %s", err, jqOutput)
	}

	if err := schemaCmd.Wait(); err != nil {
		t.Fatalf("schema command failed: %v", err)
	}

	// Parse jq output to verify it's valid JSON
	var schemaData map[string]interface{}
	if err := json.Unmarshal(jqOutput, &schemaData); err != nil {
		t.Fatalf("jq output is not valid JSON: %v", err)
	}

	t.Log("✓ Schema is valid JSON")

	// Test 2: Extract specific fields with jq
	tests := []struct {
		name     string
		jqFilter string
		validate func(string) error
	}{
		{
			name:     "schema $id",
			jqFilter: `.["$id"]`,
			validate: func(result string) error {
				result = strings.Trim(result, `"`+"\n")
				if !strings.Contains(result, "sink.schema.json") {
					return fmt.Errorf("$id doesn't contain 'sink.schema.json': %s", result)
				}
				return nil
			},
		},
		{
			name:     "schema title",
			jqFilter: ".title",
			validate: func(result string) error {
				if !strings.Contains(result, "Sink") {
					return fmt.Errorf("title doesn't contain 'Sink': %s", result)
				}
				return nil
			},
		},
		{
			name:     "required fields",
			jqFilter: ".required | length",
			validate: func(result string) error {
				result = strings.TrimSpace(result)
				if result != "2" {
					return fmt.Errorf("expected 2 required fields, got: %s", result)
				}
				return nil
			},
		},
		{
			name:     "properties exist",
			jqFilter: ".properties | keys | length",
			validate: func(result string) error {
				result = strings.TrimSpace(result)
				// Should have multiple properties (version, facts, platforms, etc.)
				if result == "0" {
					return fmt.Errorf("schema has no properties")
				}
				return nil
			},
		},
		{
			name:     "version property type",
			jqFilter: ".properties.version.type",
			validate: func(result string) error {
				result = strings.Trim(result, `"`+"\n")
				if result != "string" {
					return fmt.Errorf("version property type = %s, want 'string'", result)
				}
				return nil
			},
		},
		{
			name:     "platforms property type",
			jqFilter: ".properties.platforms.type",
			validate: func(result string) error {
				result = strings.Trim(result, `"`+"\n")
				if result != "array" {
					return fmt.Errorf("platforms property type = %s, want 'array'", result)
				}
				return nil
			},
		},
		{
			name:     "facts property type",
			jqFilter: ".properties.facts.type",
			validate: func(result string) error {
				result = strings.Trim(result, `"`+"\n")
				if result != "object" {
					return fmt.Errorf("facts property type = %s, want 'object'", result)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaCmd := exec.Command(binaryPath, "schema")
			jqCmd := exec.Command(jqBin, "-r", tt.jqFilter)

			pipe, err := schemaCmd.StdoutPipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}
			jqCmd.Stdin = pipe

			if err := schemaCmd.Start(); err != nil {
				t.Fatalf("failed to start schema command: %v", err)
			}

			output, err := jqCmd.CombinedOutput()
			if err != nil {
				t.Fatalf("jq command failed: %v\nOutput: %s", err, output)
			}

			if err := schemaCmd.Wait(); err != nil {
				t.Fatalf("schema command failed: %v", err)
			}

			result := string(output)
			if err := tt.validate(result); err != nil {
				t.Errorf("validation failed: %v", err)
			} else {
				t.Logf("✓ %s validated: %s", tt.name, strings.TrimSpace(result))
			}
		})
	}

	// Test 3: Validate that the schema can validate a config file
	t.Run("validate config with schema", func(t *testing.T) {
		// Create a simple valid config
		validConfig := `{
  "$schema": "../src/sink.schema.json",
  "version": "1.0.0",
  "platforms": [
    {
      "os": "darwin",
      "match": "darwin*",
      "name": "Test Platform",
      "install_steps": [
        {
          "name": "test",
          "command": "echo test"
        }
      ]
    }
  ]
}`
		validConfigFile := filepath.Join(tempDir, "valid-config.json")
		if err := os.WriteFile(validConfigFile, []byte(validConfig), 0644); err != nil {
			t.Fatalf("failed to write valid config: %v", err)
		}

		// Validate it with sink
		cmd := exec.Command(binaryPath, "validate", validConfigFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("validation failed: %v\nOutput: %s", err, output)
		}

		if !strings.Contains(string(output), "✅") {
			t.Errorf("validation didn't succeed: %s", output)
		}

		t.Log("✓ Config validated successfully with embedded schema")
	})
}

// createJqInstallConfig creates a configuration to install jq to a temp directory
func createJqInstallConfig(installDir string) string {
	binDir := filepath.Join(installDir, "bin")
	jqPath := filepath.Join(binDir, "jq")

	// Detect OS and arch for jq download
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// jq download URL (using GitHub releases)
	jqVersion := "1.7.1"
	var downloadURL string
	var jqBinaryName string

	switch goos {
	case "darwin":
		if goarch == "arm64" {
			jqBinaryName = "jq-macos-arm64"
		} else {
			jqBinaryName = "jq-macos-amd64"
		}
		downloadURL = fmt.Sprintf("https://github.com/jqlang/jq/releases/download/jq-%s/%s", jqVersion, jqBinaryName)
	case "linux":
		if goarch == "arm64" {
			jqBinaryName = "jq-linux-arm64"
		} else {
			jqBinaryName = "jq-linux-amd64"
		}
		downloadURL = fmt.Sprintf("https://github.com/jqlang/jq/releases/download/jq-%s/%s", jqVersion, jqBinaryName)
	default:
		// Fallback
		downloadURL = fmt.Sprintf("https://github.com/jqlang/jq/releases/download/jq-%s/jq-linux-amd64", jqVersion)
	}

	config := fmt.Sprintf(`{
  "$schema": "../src/sink.schema.json",
  "version": "1.0.0",
  "platforms": [
    {
      "os": "%s",
      "match": "%s*",
      "name": "Test Platform",
      "install_steps": [
        {
          "name": "Create bin directory",
          "command": "mkdir -p %s"
        },
        {
          "name": "Download jq",
          "check": "test -f %s",
          "command": "curl -L -o %s %s",
          "error": "Failed to download jq"
        },
        {
          "name": "Make jq executable",
          "command": "chmod +x %s"
        },
        {
          "name": "Verify jq installation",
          "command": "%s --version"
        }
      ]
    }
  ]
}`, goos, goos, binDir, jqPath, jqPath, downloadURL, jqPath, jqPath)

	return config
}

// TestSchemaCommand tests the schema command independently
func TestSchemaCommand(t *testing.T) {
	// Build the binary if needed
	binaryPath := filepath.Join("..", "bin", "sink")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Log("Building sink binary...")
		cmd := exec.Command("go", "build", "-o", binaryPath, ".")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to build binary: %v\n%s", err, output)
		}
	}

	// Run schema command
	cmd := exec.Command(binaryPath, "schema")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("schema command failed: %v\nOutput: %s", err, output)
	}

	// Verify output is valid JSON
	var schema map[string]interface{}
	if err := json.Unmarshal(output, &schema); err != nil {
		t.Fatalf("schema output is not valid JSON: %v", err)
	}

	// Verify key fields
	if id, ok := schema["$id"].(string); !ok || id == "" {
		t.Error("schema missing or empty $id field")
	}

	if title, ok := schema["title"].(string); !ok || title == "" {
		t.Error("schema missing or empty title field")
	}

	if properties, ok := schema["properties"].(map[string]interface{}); !ok || len(properties) == 0 {
		t.Error("schema missing or empty properties")
	}

	t.Logf("✓ Schema command emitted %d bytes of valid JSON", len(output))
}

// TestSchemaSynchronization verifies that the schema file and embedded schema are synchronized
// This test ensures that sink.schema.json and the embedded schema in schema.go match exactly
func TestSchemaSynchronization(t *testing.T) {
	// Read the schema file from disk
	schemaPath := filepath.Join("sink.schema.json")
	schemaFileBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema file %s: %v", schemaPath, err)
	}

	// Parse both schemas as JSON to normalize formatting
	var schemaFromFile map[string]interface{}
	if err := json.Unmarshal(schemaFileBytes, &schemaFromFile); err != nil {
		t.Fatalf("schema file is not valid JSON: %v", err)
	}

	var schemaEmbedded map[string]interface{}
	if err := json.Unmarshal([]byte(embeddedSchema), &schemaEmbedded); err != nil {
		t.Fatalf("embedded schema is not valid JSON: %v", err)
	}

	// Convert both to canonical JSON (sorted keys, no whitespace)
	fileCanonical, err := json.Marshal(schemaFromFile)
	if err != nil {
		t.Fatalf("failed to marshal schema from file: %v", err)
	}

	embeddedCanonical, err := json.Marshal(schemaEmbedded)
	if err != nil {
		t.Fatalf("failed to marshal embedded schema: %v", err)
	}

	// Compare the canonical forms
	if string(fileCanonical) != string(embeddedCanonical) {
		t.Error("❌ Schema file and embedded schema are OUT OF SYNC!")
		t.Error("")
		t.Error("The schema in sink.schema.json does not match the embedded schema in schema.go")
		t.Error("")
		t.Error("To fix this issue:")
		t.Error("  1. Ensure sink.schema.json contains the correct, up-to-date schema")
		t.Error("  2. Run: make build")
		t.Error("  3. This will regenerate schema.go with the embedded schema from sink.schema.json")
		t.Error("")
		t.Error("If the schema file was modified more recently than the binary was built,")
		t.Error("you need to rebuild the binary to embed the latest schema.")
		t.Error("")

		// Show which properties differ
		t.Error("Checking for specific differences...")
		compareSchemaProperties(t, schemaFromFile, schemaEmbedded, "")

		t.FailNow()
	}

	t.Log("✓ Schema file and embedded schema are synchronized")
	t.Logf("  Schema file: %s (%d bytes)", schemaPath, len(schemaFileBytes))
	t.Logf("  Embedded schema: %d bytes", len(embeddedSchema))
}

// compareSchemaProperties recursively compares two schema objects and reports differences
func compareSchemaProperties(t *testing.T, file, embedded interface{}, path string) {
	fileMap, fileIsMap := file.(map[string]interface{})
	embeddedMap, embeddedIsMap := embedded.(map[string]interface{})

	if fileIsMap && embeddedIsMap {
		// Check for missing keys in embedded
		for key := range fileMap {
			if _, ok := embeddedMap[key]; !ok {
				t.Errorf("  - Property %s.%s exists in file but missing in embedded schema", path, key)
			}
		}

		// Check for extra keys in embedded
		for key := range embeddedMap {
			if _, ok := fileMap[key]; !ok {
				t.Errorf("  - Property %s.%s exists in embedded but missing in file schema", path, key)
			}
		}

		// Recursively compare common keys (limit depth to avoid noise)
		if len(path) < 50 { // Prevent infinite recursion
			for key := range fileMap {
				if embeddedVal, ok := embeddedMap[key]; ok {
					newPath := key
					if path != "" {
						newPath = path + "." + key
					}
					compareSchemaProperties(t, fileMap[key], embeddedVal, newPath)
				}
			}
		}
	} else if fileIsMap != embeddedIsMap {
		t.Errorf("  - Property %s type mismatch: file is map=%v, embedded is map=%v", path, fileIsMap, embeddedIsMap)
	}
}

// TestSchemaHasRequiredNewProperties verifies that new properties (verbose, sleep, timeout) exist
// This test ensures features added to the code are also present in the schema
func TestSchemaHasRequiredNewProperties(t *testing.T) {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(embeddedSchema), &schema); err != nil {
		t.Fatalf("embedded schema is not valid JSON: %v", err)
	}

	// Get $defs section
	defs, ok := schema["$defs"].(map[string]interface{})
	if !ok {
		t.Fatal("schema missing $defs section")
	}

	// Test cases for each type that should have the new properties
	tests := []struct {
		defName    string
		properties []string
	}{
		{
			defName:    "fact",
			properties: []string{"verbose", "sleep", "timeout"},
		},
		{
			defName:    "remediation_step",
			properties: []string{"verbose", "sleep", "timeout"},
		},
		{
			defName:    "install_step",
			properties: []string{"verbose", "sleep", "timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.defName, func(t *testing.T) {
			def, ok := defs[tt.defName].(map[string]interface{})
			if !ok {
				t.Fatalf("$defs.%s not found or not an object", tt.defName)
			}

			// For install_step, it's wrapped in oneOf
			var props map[string]interface{}
			if tt.defName == "install_step" {
				oneOf, ok := def["oneOf"].([]interface{})
				if !ok || len(oneOf) == 0 {
					t.Fatal("install_step missing oneOf")
				}
				firstOption, ok := oneOf[0].(map[string]interface{})
				if !ok {
					t.Fatal("install_step oneOf[0] is not an object")
				}
				props, ok = firstOption["properties"].(map[string]interface{})
				if !ok {
					t.Fatal("install_step oneOf[0] missing properties")
				}
			} else {
				props, ok = def["properties"].(map[string]interface{})
				if !ok {
					t.Fatalf("$defs.%s missing properties section", tt.defName)
				}
			}

			// Check each required property
			for _, propName := range tt.properties {
				if _, exists := props[propName]; !exists {
					t.Errorf("❌ $defs.%s missing property: %s", tt.defName, propName)
					t.Errorf("   This indicates the schema is out of sync with the code")
				} else {
					t.Logf("✓ $defs.%s has property: %s", tt.defName, propName)
				}
			}

			// Verify timeout has oneOf structure for string|object
			if timeoutProp, ok := props["timeout"].(map[string]interface{}); ok {
				if oneOf, ok := timeoutProp["oneOf"].([]interface{}); ok {
					if len(oneOf) != 2 {
						t.Errorf("timeout oneOf should have 2 options (string and object), got %d", len(oneOf))
					} else {
						t.Logf("✓ $defs.%s.timeout has oneOf with %d options", tt.defName, len(oneOf))
					}
				} else {
					t.Errorf("timeout property missing oneOf structure")
				}
			}
		})
	}
}
