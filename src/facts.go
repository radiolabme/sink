package main

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// Transport abstracts command execution (local or SSH)
type Transport interface {
	Run(cmd string) (stdout, stderr string, exitCode int, err error)
}

// FactGatherer gathers facts by running commands
type FactGatherer struct {
	definitions map[string]FactDef
	transport   Transport
	currentOS   string // Platform to use for filtering (defaults to runtime.GOOS)
}

// NewFactGatherer creates a new fact gatherer
func NewFactGatherer(definitions map[string]FactDef, transport Transport) *FactGatherer {
	return &FactGatherer{
		definitions: definitions,
		transport:   transport,
		currentOS:   runtime.GOOS,
	}
}

// Gather runs all fact-gathering commands and returns the facts
func (fg *FactGatherer) Gather() (Facts, error) {
	facts := make(Facts)

	for name, def := range fg.definitions {
		// Skip if platform-specific and doesn't match current OS
		if len(def.Platforms) > 0 {
			matched := false
			for _, platform := range def.Platforms {
				if platform == fg.currentOS {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Run the command
		stdout, _, exitCode, err := fg.transport.Run(def.Command)

		// Handle failures based on Required flag
		if err != nil || exitCode != 0 {
			if def.Required {
				return nil, fmt.Errorf("required fact '%s' failed: %w", name, err)
			}
			// Skip optional failed facts
			continue
		}

		// Trim output
		value := strings.TrimSpace(stdout)

		// Apply transform if specified
		if def.Transform != nil {
			transformed, err := applyTransform(value, def.Transform, def.Strict)
			if err != nil {
				if def.Required {
					return nil, fmt.Errorf("fact '%s' transform failed: %w", name, err)
				}
				continue
			}
			value = transformed
		}

		// Coerce to the specified type
		typedValue, err := coerceType(value, def.Type)
		if err != nil {
			if def.Required {
				return nil, fmt.Errorf("fact '%s' type coercion failed: %w", name, err)
			}
			continue
		}

		facts[name] = typedValue
	}

	return facts, nil
}

// Export converts facts to environment variable format
func (fg *FactGatherer) Export(facts Facts) []string {
	var exports []string

	for name, value := range facts {
		def, ok := fg.definitions[name]
		if !ok || def.Export == "" {
			continue
		}

		// Convert value to string
		var strValue string
		switch v := value.(type) {
		case string:
			strValue = v
		case bool:
			strValue = strconv.FormatBool(v)
		case int64:
			strValue = strconv.FormatInt(v, 10)
		default:
			strValue = fmt.Sprintf("%v", v)
		}

		exports = append(exports, fmt.Sprintf("%s=%s", def.Export, strValue))
	}

	return exports
}

// applyTransform applies value transformation using the transform map
func applyTransform(value string, transform map[string]string, strict bool) (string, error) {
	if transform == nil {
		return value, nil
	}

	if transformed, ok := transform[value]; ok {
		return transformed, nil
	}

	if strict {
		return "", fmt.Errorf("no transform mapping for value '%s'", value)
	}

	return value, nil
}

// coerceType converts a string value to the specified type
func coerceType(value string, typ string) (interface{}, error) {
	switch typ {
	case "", "string":
		return value, nil
	case "boolean":
		if value == "true" {
			return true, nil
		}
		if value == "false" {
			return false, nil
		}
		return nil, fmt.Errorf("cannot convert '%s' to boolean", value)
	case "integer":
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert '%s' to integer: %w", value, err)
		}
		return i, nil
	default:
		return nil, fmt.Errorf("unknown type '%s'", typ)
	}
}
