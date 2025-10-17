package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed sink.schema.json
var embeddedSchema string

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Handle global --help flag
	if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
		printCommandHelp(command)
		os.Exit(0)
	}

	switch command {
	case "version", "-v", "--version":
		fmt.Printf("sink version %s\n", Version)
	case "execute", "exec":
		executeCommand()
	case "bootstrap":
		bootstrapCommand()
	case "remote":
		remoteCommand()
	case "facts":
		factsCommand()
	case "validate":
		validateCommand()
	case "schema":
		schemaCommand()
	case "help", "-h", "--help":
		// Handle "sink help <command>"
		if len(os.Args) > 2 {
			printCommandHelp(os.Args[2])
		} else {
			printUsage()
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`sink - Nano-scale dependency management engine

Usage:
  sink <command> [options]

Commands:
  execute <config>    Execute installation steps from config file
  bootstrap <source>  Bootstrap from URL or file (HTTP/HTTPS/GitHub)
  remote deploy       Deploy to remote hosts via SSH
  facts <config>      Gather and display facts from config file
  validate <config>   Validate config file structure
  schema              Output JSON schema to stdout
  version             Show version information
  help [command]      Show help for a specific command

Global Options:
  -h, --help         Show help for command
  -v, --version      Show version information

Get detailed help for a command:
  sink help execute
  sink execute --help

Examples:
  sink execute install-config.json
  sink execute --dry-run install-config.json
  sink facts install-config.json
  sink validate install-config.json
  sink schema > schema.json

Documentation:
  See docs/ directory for complete documentation
  Configuration reference: examples/configuration-reference.md
  GitHub pinning guide: docs/GITHUB_URL_PINNING.md

Version: %s
`, Version)
}

// printCommandHelp displays detailed help information for a specific Sink command.
// This function serves as the central help dispatcher, routing help requests to
// command-specific help functions.
//
// Parameters:
//   - command: The command name to display help for (e.g., "execute", "bootstrap")
//
// Supported commands:
//   - execute/exec: Installation step execution
//   - bootstrap: Remote configuration loading
//   - remote: SSH deployment to remote hosts
//   - facts: System fact gathering
//   - validate: Configuration validation
//   - schema: JSON schema output
//   - version: Version information
//
// For unknown commands, displays an error message and shows general usage.
// This function is called when users request help via:
//   - sink help <command>
//   - sink <command> --help
//   - sink <command> -h
func printCommandHelp(command string) {
	switch command {
	case "execute", "exec":
		printExecuteHelp()
	case "bootstrap":
		printBootstrapHelp()
	case "remote":
		printRemoteHelp()
	case "facts":
		printFactsHelp()
	case "validate":
		printValidateHelp()
	case "schema":
		printSchemaHelp()
	case "version":
		printVersionHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
	}
}

func printExecuteHelp() {
	fmt.Printf(`sink execute - Execute installation steps from configuration

Usage:
  sink execute <config> [options]
  sink exec <config> [options]

Description:
  Executes the installation steps defined in the configuration file for the
  current platform (or overridden platform). Steps are executed sequentially
  with real-time progress output.

  The execute command:
  1. Loads and validates the configuration file
  2. Gathers facts (system information queries)
  3. Detects the current platform (OS and distribution)
  4. Executes installation steps for the matched platform
  5. Reports success or failure with detailed output

Options:
  --dry-run              Preview steps without executing them
                         Shows what would be executed
                         
  --platform <os>        Override platform detection
                         Values: darwin, linux, windows
                         Useful for testing configs on different platforms
  
  -v, --verbose          Enable verbose output for debugging
                         Shows detailed command execution, exit codes,
                         and output for all steps
  
  --json                 Output execution events as JSON to stdout
                         Enables machine-readable structured output
                         Compatible with --verbose for detailed metadata
  
  -h, --help             Show this help message

Arguments:
  <config>               Path to configuration file (JSON format)
                         Must be a valid Sink configuration

Exit Codes:
  0                      All steps executed successfully
  1                      One or more steps failed or config invalid

Examples:
  # Execute configuration
  sink execute install-config.json

  # Dry run to preview steps
  sink execute --dry-run install-config.json

  # Enable verbose output for debugging
  sink execute --verbose install-config.json

  # JSON output for machine consumption
  sink execute --json install-config.json
  
  # Combine flags for detailed JSON output
  sink execute --json --verbose --dry-run install-config.json

  # Combine dry-run with verbose for detailed preview
  sink execute --dry-run --verbose install-config.json

  # Override platform for testing
  sink execute --platform linux install-config.json

  # Execute with short command alias
  sink exec config.json

Output:
  The command shows:
  • Facts gathered from the system
  • Platform and distribution detected
  • Each step's progress (running, success, failed)
  • Final summary with counts
  • Confirmation prompt before execution (unless --dry-run)

Configuration:
  See: examples/configuration-reference.md
  Schema: sink schema > schema.json

Related Commands:
  sink validate <config>     Validate config before execution
  sink facts <config>        View facts that would be gathered
  sink help facts            Help for facts command
`)
}

func printFactsHelp() {
	fmt.Printf(`sink facts - Gather and display system facts

Usage:
  sink facts <config>

Description:
  Gathers facts (system information) defined in the configuration file
  and displays them in a detailed format. Facts are queries that run
  before installation steps to gather information about the system.

  Facts can be:
  • String values (hostnames, versions, paths)
  • Boolean values (feature flags, existence checks)
  • Integer values (CPU count, memory)
  • Transformed values (mapped to canonical names)

  This command is useful for:
  • Debugging fact definitions
  • Understanding what information will be available
  • Testing fact commands before execution
  • Viewing environment variable exports

Options:
  -h, --help             Show this help message

Arguments:
  <config>               Path to configuration file with facts section

Output:
  For each fact, displays:
  • Name - The fact identifier
  • Value - The gathered value
  • Type - The Go type (string, bool, int, etc.)
  • Export - Environment variable name (if defined)
  • Description - Human-readable description (if defined)

  Also shows export statements that can be eval'd in shell.

Examples:
  # Gather and display all facts
  sink facts install-config.json

  # Use with eval to export to shell
  eval $(sink facts config.json | grep "export")

  # View facts for a specific platform
  sink facts --platform linux config.json

Fact Definition:
  Facts are defined in the config's "facts" section:

  {
    "facts": {
      "hostname": {
        "command": "hostname",
        "description": "System hostname",
        "export": "SINK_HOSTNAME"
      }
    }
  }

Related Commands:
  sink execute <config>      Execute with facts
  sink validate <config>     Validate fact definitions
`)
}

func printValidateHelp() {
	fmt.Printf(`sink validate - Validate configuration file

Usage:
  sink validate <config>

Description:
  Validates the configuration file against the Sink schema. Checks for:
  • Valid JSON syntax
  • Required fields present (version, platforms)
  • Correct data types for all fields
  • Valid platform patterns and OS names
  • Valid fact definitions
  • Valid install step structures
  • Bootstrap configuration (if present)

  This command is useful for:
  • Testing configurations before deployment
  • CI/CD validation pipelines
  • Debugging configuration syntax errors
  • Understanding configuration structure

Options:
  -h, --help             Show this help message

Arguments:
  <config>               Path to configuration file to validate

Output:
  On success:
  • ✅ Config is valid
  • Summary of configuration contents
    - Version number
    - Number of facts
    - Number of platforms
    - Platform details (install steps, distributions)
    - Default values (if present)

  On failure:
  • ❌ Validation failed with detailed error message
  • Error indicates the problem location and type

Exit Codes:
  0                      Configuration is valid
  1                      Configuration is invalid

Examples:
  # Validate a configuration
  sink validate install-config.json

  # Validate in CI/CD pipeline
  for config in configs/*.json; do
    sink validate "$config" || exit 1
  done

Schema:
  Configurations are validated against the embedded JSON schema.
  View the schema:
    sink schema
    sink schema > schema.json

  Schema location in source:
    src/sink.schema.json (embedded at build time)

Related Commands:
  sink schema                Output the JSON schema
  sink execute <config>      Execute validated config
`)
}

func printSchemaHelp() {
	fmt.Printf(`sink schema - Output JSON schema

Usage:
  sink schema

Description:
  Outputs the Sink JSON schema to stdout. The schema is embedded in
  the binary at build time, ensuring it always matches the version
  of Sink you're running.

  The schema defines:
  • Configuration structure
  • Required and optional fields
  • Field types and formats
  • Valid values (enums)
  • Platform definitions
  • Fact definitions
  • Install step types

  Use the schema for:
  • IDE autocompletion (VS Code, IntelliJ, etc.)
  • Configuration validation
  • Documentation generation
  • Understanding config structure

Options:
  -h, --help             Show this help message

Output:
  Complete JSON Schema Draft 2020-12 schema

Examples:
  # View schema
  sink schema

  # Save schema to file
  sink schema > sink.schema.json

  # Use with jq to explore
  sink schema | jq '.properties'
  sink schema | jq '."$defs"'

  # Validate config with external tool
  sink schema > schema.json
  jsonschema -i config.json schema.json

Schema in Configs:
  Reference the schema in your configuration files:

  {
    "$schema": "../src/sink.schema.json",
    "version": "1.0.0",
    "platforms": [...]
  }

  With the $schema property, editors provide:
  • Autocompletion
  • Inline validation
  • Hover documentation
  • Error highlighting

Schema Location:
  Source: src/sink.schema.json
  Online: https://raw.githubusercontent.com/radiolabme/sink/main/src/sink.schema.json
  Versioned: .../v0.1.0/src/sink.schema.json (replace with git tag)

Related Commands:
  sink validate <config>     Validate against schema
`)
}

func printVersionHelp() {
	fmt.Printf(`sink version - Show version information

Usage:
  sink version
  sink -v
  sink --version

Description:
  Displays the version number of Sink.

Examples:
  sink version
  sink -v

Output:
  sink version 0.1.0
`)
}

func executeCommand() {
	var configFile string
	var dryRun bool
	var verbose bool
	var jsonOutput bool
	var platformOverride string

	// Parse flags
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			printExecuteHelp()
			os.Exit(0)
		case "--dry-run":
			dryRun = true
		case "-v", "--verbose":
			verbose = true
		case "--json":
			jsonOutput = true
		case "--platform":
			if i+1 < len(args) {
				platformOverride = args[i+1]
				i++
			} else {
				fmt.Fprintf(os.Stderr, "Error: --platform requires a value\n")
				os.Exit(1)
			}
		default:
			if configFile == "" {
				configFile = arg
			} else {
				fmt.Fprintf(os.Stderr, "Error: unexpected argument: %s\n", arg)
				os.Exit(1)
			}
		}
	}

	if configFile == "" {
		fmt.Fprintf(os.Stderr, "Error: config file required\n")
		os.Exit(1)
	}

	// Load config
	config, err := LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Execute using shared function
	executeConfigWithOptions(config, dryRun, verbose, jsonOutput, platformOverride)
}

// executeConfigWithOptions executes a loaded configuration with the specified options.
// This function is the core execution engine shared by both the execute and bootstrap commands.
//
// Parameters:
//   - config: A validated Sink configuration loaded from JSON
//   - dryRun: If true, preview steps without executing them
//   - verbose: If true, enable detailed logging for debugging
//   - platformOverride: Optional platform override (e.g., "linux", "darwin")
//
// The function performs the following operations:
//  1. Creates a local transport for command execution
//  2. Gathers facts defined in the configuration
//  3. Determines the target platform (detected or overridden)
//  4. Selects the appropriate platform configuration
//  5. Creates and configures an executor
//  6. Displays execution context and prompts for confirmation (unless dry-run)
//  7. Executes all installation steps for the platform
//  8. Reports success/failure summary and exits with appropriate code
//
// Exit codes:
//   - 0: All steps executed successfully
//   - 1: Configuration errors, platform not found, or step failures
//
// The function handles user interaction for confirmation in non-dry-run mode
// and provides real-time progress feedback during execution.
func executeConfigWithOptions(config *Config, dryRun bool, verbose bool, jsonOutput bool, platformOverride string) {
	// Create transport
	transport := NewLocalTransport()

	// Gather facts
	if !jsonOutput {
		fmt.Println("📊 Gathering facts...")
	}
	gatherer := NewFactGatherer(config.Facts, transport)
	gatherer.Verbose = verbose
	facts, err := gatherer.Gather()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error gathering facts: %v\n", err)
		os.Exit(1)
	}

	// Display gathered facts (only in non-JSON mode)
	if !jsonOutput && len(facts) > 0 {
		fmt.Printf("   Gathered %d facts:\n", len(facts))
		for name, value := range facts {
			fmt.Printf("   • %s = %v\n", name, value)
		}
	}

	// Determine platform
	targetOS := runtime.GOOS
	if platformOverride != "" && !jsonOutput {
		targetOS = platformOverride
		fmt.Printf("🎯 Platform override: %s\n", targetOS)
	} else if platformOverride != "" {
		targetOS = platformOverride
	}

	// Select platform
	if verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Looking for platform matching OS: %s\n", targetOS)
		fmt.Fprintf(os.Stderr, "[VERBOSE] Available platforms in config:\n")
		for _, p := range config.Platforms {
			fmt.Fprintf(os.Stderr, "[VERBOSE]   - %s (os=%s)\n", p.Name, p.OS)
		}
	}

	var selectedPlatform *Platform
	for i := range config.Platforms {
		if config.Platforms[i].OS == targetOS {
			selectedPlatform = &config.Platforms[i]
			if verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] ✓ Matched platform: %s\n", selectedPlatform.Name)
			}
			break
		}
	}

	if selectedPlatform == nil {
		fmt.Fprintf(os.Stderr, "Error: no platform configuration found for %s\n", targetOS)
		if verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] No platform matched target OS '%s'\n", targetOS)
			fmt.Fprintf(os.Stderr, "[VERBOSE] Available platforms were: ")
			for i, p := range config.Platforms {
				if i > 0 {
					fmt.Fprintf(os.Stderr, ", ")
				}
				fmt.Fprintf(os.Stderr, "%s", p.OS)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
		os.Exit(1)
	}

	if !jsonOutput {
		fmt.Printf("🖥️  Platform: %s (%s)\n", selectedPlatform.Name, selectedPlatform.OS)
		fmt.Printf("📝 Steps: %d\n\n", len(selectedPlatform.InstallSteps))
	}

	// Create executor
	executor := NewExecutor(transport)
	executor.DryRun = dryRun
	executor.Verbose = verbose
	executor.JSONOutput = jsonOutput

	// Display execution context (only in non-JSON mode)
	ctx := executor.GetContext()
	if !jsonOutput {
		fmt.Println("🔍 Execution Context:")
		fmt.Printf("   Host:      %s\n", ctx.Host)
		fmt.Printf("   User:      %s\n", ctx.User)
		fmt.Printf("   Work Dir:  %s\n", ctx.WorkDir)
		fmt.Printf("   OS/Arch:   %s/%s\n", ctx.OS, ctx.Arch)
		fmt.Printf("   Transport: %s\n", ctx.Transport)
		fmt.Println()
	}

	if dryRun {
		if !jsonOutput {
			fmt.Println("🔍 DRY RUN MODE - No commands will be executed")
			fmt.Println()
		}
	} else {
		// Confirmation prompt for real execution (skip in JSON mode)
		if !jsonOutput {
			fmt.Printf("⚠️  You are about to execute %d steps on %s as %s\n",
				len(selectedPlatform.InstallSteps),
				ctx.Host,
				ctx.User)
			fmt.Print("   Continue? [yes/no]: ")

			var response string
			fmt.Scanln(&response)

			if response != "yes" {
				fmt.Println("\n❌ Execution cancelled by user")
				os.Exit(0)
			}
			fmt.Println()
		}
	}

	// Set up event handler for progress (only in non-JSON mode)
	stepNum := 0
	if !jsonOutput {
		executor.OnEvent = func(event ExecutionEvent) {
			switch event.Status {
			case "running":
				stepNum++
				fmt.Printf("[%d/%d] %s...\n", stepNum, len(selectedPlatform.InstallSteps), event.StepName)
			case "success":
				fmt.Printf("      ✓ Success\n")
				if event.Output != "" && !dryRun {
					// Show first line of output
					lines := filepath.SplitList(event.Output)
					if len(lines) > 0 && lines[0] != "" {
						fmt.Printf("      Output: %s\n", lines[0])
					}
				}
			case "failed":
				fmt.Printf("      ✗ Failed: %s\n", event.Error)
			case "skipped":
				fmt.Printf("      ⊘ Skipped\n")
			}
		}
	}

	// Execute
	results := executor.ExecutePlatform(*selectedPlatform, facts)

	// Summary (only in non-JSON mode)
	if !jsonOutput {
		fmt.Println()
		successCount := 0
		failCount := 0
		for _, result := range results {
			if result.Error == "" {
				successCount++
			} else {
				failCount++
			}
		}

		if failCount > 0 {
			fmt.Printf("❌ Execution failed: %d succeeded, %d failed\n", successCount, failCount)
			os.Exit(1)
		} else {
			if dryRun {
				fmt.Printf("✅ Dry run complete: %d steps validated\n", successCount)
			} else {
				fmt.Printf("✅ Execution complete: %d steps succeeded\n", successCount)
			}
		}
	}
}

func factsCommand() {
	// Check for --help first
	for _, arg := range os.Args[2:] {
		if arg == "-h" || arg == "--help" {
			printFactsHelp()
			os.Exit(0)
		}
	}

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: config file required\n\n")
		printFactsHelp()
		os.Exit(1)
	}

	configFile := os.Args[2]

	// Load config
	config, err := LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if len(config.Facts) == 0 {
		fmt.Println("No facts defined in config")
		return
	}

	// Create transport
	transport := NewLocalTransport()

	// Gather facts
	fmt.Println("📊 Gathering facts...")
	fmt.Println()
	gatherer := NewFactGatherer(config.Facts, transport)
	facts, err := gatherer.Gather()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error gathering facts: %v\n", err)
		os.Exit(1)
	}

	// Display facts
	fmt.Printf("Gathered %d facts:\n\n", len(facts))
	for name, value := range facts {
		def := config.Facts[name]
		fmt.Printf("  %s\n", name)
		fmt.Printf("    Value: %v\n", value)
		fmt.Printf("    Type: %T\n", value)
		if def.Export != "" {
			fmt.Printf("    Export: %s=%v\n", def.Export, value)
		}
		if def.Description != "" {
			fmt.Printf("    Description: %s\n", def.Description)
		}
		fmt.Println()
	}

	// Show export statements
	exports := gatherer.Export(facts)
	if len(exports) > 0 {
		fmt.Println("Environment variables:")
		for _, exp := range exports {
			fmt.Printf("  export %s\n", exp)
		}
	}
}

func schemaCommand() {
	// Check for --help
	for _, arg := range os.Args[2:] {
		if arg == "-h" || arg == "--help" {
			printSchemaHelp()
			os.Exit(0)
		}
	}

	// Output the embedded schema to stdout
	fmt.Print(embeddedSchema)
}

func validateCommand() {
	// Check for --help first
	for _, arg := range os.Args[2:] {
		if arg == "-h" || arg == "--help" {
			printValidateHelp()
			os.Exit(0)
		}
	}

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: config file required\n\n")
		printValidateHelp()
		os.Exit(1)
	}

	configFile := os.Args[2]

	// Load and validate config
	config, err := LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("✅ Config is valid\n\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("  Version: %s\n", config.Version)
	fmt.Printf("  Facts: %d\n", len(config.Facts))
	fmt.Printf("  Platforms: %d\n", len(config.Platforms))

	for _, platform := range config.Platforms {
		fmt.Printf("\n  Platform: %s (%s)\n", platform.Name, platform.OS)
		fmt.Printf("    Install steps: %d\n", len(platform.InstallSteps))
		if len(platform.Distributions) > 0 {
			fmt.Printf("    Distributions: %d\n", len(platform.Distributions))
			for _, dist := range platform.Distributions {
				fmt.Printf("      • %s (%d steps)\n", dist.Name, len(dist.InstallSteps))
			}
		}
	}

	// Show default values if present
	if config.Defaults != nil {
		fmt.Printf("\n  Defaults:\n")
		defaultsJSON, _ := json.MarshalIndent(config.Defaults, "    ", "  ")
		fmt.Printf("    %s\n", string(defaultsJSON))
	}
}
