package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version", "-v", "--version":
		fmt.Printf("sink version %s\n", version)
	case "execute", "exec":
		executeCommand()
	case "facts":
		factsCommand()
	case "validate":
		validateCommand()
	case "help", "-h", "--help":
		printUsage()
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
  facts <config>      Gather and display facts from config file
  validate <config>   Validate config file structure
  version             Show version information
  help                Show this help message

Options for execute:
  --dry-run          Preview steps without executing them
  --platform <os>    Override platform detection (darwin, linux, windows)

Examples:
  sink execute install-config.json
  sink execute --dry-run install-config.json
  sink facts install-config.json
  sink validate install-config.json

Version: %s
`, version)
}

func executeCommand() {
	var configFile string
	var dryRun bool
	var platformOverride string

	// Parse flags
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--dry-run":
			dryRun = true
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

	// Create transport
	transport := NewLocalTransport()

	// Gather facts
	fmt.Println("📊 Gathering facts...")
	gatherer := NewFactGatherer(config.Facts, transport)
	facts, err := gatherer.Gather()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error gathering facts: %v\n", err)
		os.Exit(1)
	}

	// Display gathered facts
	if len(facts) > 0 {
		fmt.Printf("   Gathered %d facts:\n", len(facts))
		for name, value := range facts {
			fmt.Printf("   • %s = %v\n", name, value)
		}
	}

	// Determine platform
	targetOS := runtime.GOOS
	if platformOverride != "" {
		targetOS = platformOverride
		fmt.Printf("🎯 Platform override: %s\n", targetOS)
	}

	// Select platform
	var selectedPlatform *Platform
	for i := range config.Platforms {
		if config.Platforms[i].OS == targetOS {
			selectedPlatform = &config.Platforms[i]
			break
		}
	}

	if selectedPlatform == nil {
		fmt.Fprintf(os.Stderr, "Error: no platform configuration found for %s\n", targetOS)
		os.Exit(1)
	}

	fmt.Printf("🖥️  Platform: %s (%s)\n", selectedPlatform.Name, selectedPlatform.OS)
	fmt.Printf("📝 Steps: %d\n\n", len(selectedPlatform.InstallSteps))

	// Create executor
	executor := NewExecutor(transport)
	executor.DryRun = dryRun

	// Display execution context
	ctx := executor.GetContext()
	fmt.Println("🔍 Execution Context:")
	fmt.Printf("   Host:      %s\n", ctx.Host)
	fmt.Printf("   User:      %s\n", ctx.User)
	fmt.Printf("   Work Dir:  %s\n", ctx.WorkDir)
	fmt.Printf("   OS/Arch:   %s/%s\n", ctx.OS, ctx.Arch)
	fmt.Printf("   Transport: %s\n", ctx.Transport)
	fmt.Println()

	if dryRun {
		fmt.Println("🔍 DRY RUN MODE - No commands will be executed")
		fmt.Println()
	} else {
		// Confirmation prompt for real execution
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

	// Set up event handler for progress
	stepNum := 0
	executor.OnEvent = func(event ExecutionEvent) {
		if event.Status == "running" {
			stepNum++
			fmt.Printf("[%d/%d] %s...\n", stepNum, len(selectedPlatform.InstallSteps), event.StepName)
		} else if event.Status == "success" {
			fmt.Printf("      ✓ Success\n")
			if event.Output != "" && !dryRun {
				// Show first line of output
				lines := filepath.SplitList(event.Output)
				if len(lines) > 0 && lines[0] != "" {
					fmt.Printf("      Output: %s\n", lines[0])
				}
			}
		} else if event.Status == "failed" {
			fmt.Printf("      ✗ Failed: %s\n", event.Error)
		} else if event.Status == "skipped" {
			fmt.Printf("      ⊘ Skipped\n")
		}
	}

	// Execute
	results := executor.ExecutePlatform(*selectedPlatform, facts)

	// Summary
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

func factsCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: config file required\n")
		fmt.Fprintf(os.Stderr, "Usage: sink facts <config>\n")
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

func validateCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Error: config file required\n")
		fmt.Fprintf(os.Stderr, "Usage: sink validate <config>\n")
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
