package main

import (
	"fmt"
	"os"
)

// remoteCommand handles remote deployment
func remoteCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Error: Missing subcommand")
		fmt.Fprintln(os.Stderr, "")
		printRemoteHelp()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	// Check for help
	for _, arg := range os.Args[2:] {
		if arg == "-h" || arg == "--help" {
			printRemoteHelp()
			os.Exit(0)
		}
	}

	switch subcommand {
	case "deploy":
		remoteDeployCommand()
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", subcommand)
		printRemoteHelp()
		os.Exit(1)
	}
}

// remoteDeployCommand deploys sink and config to remote hosts
func remoteDeployCommand() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "Error: Missing required arguments")
		fmt.Fprintln(os.Stderr, "")
		printRemoteHelp()
		os.Exit(1)
	}

	target := os.Args[3]
	configSource := os.Args[4]

	// Parse optional flags
	dryRun := false
	noCleanup := false

	for i := 5; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--no-cleanup":
			noCleanup = true
		case "--yes", "-y":
			// Skip confirmation (for future use)
			continue
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\n", arg)
			os.Exit(1)
		}
	}

	fmt.Println("🚀 Sink Remote Deployment")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("   Target: %s\n", target)
	fmt.Printf("   Config: %s\n", configSource)
	if dryRun {
		fmt.Println("   Mode:   DRY RUN")
	}
	fmt.Println()

	// TODO: Implement SSH deployment
	// For now, show what would happen
	fmt.Println("⚠️  SSH remote deployment not yet implemented")
	fmt.Println()
	fmt.Println("Planned steps:")
	fmt.Println("  1. Connect to", target, "via SSH")
	fmt.Println("  2. Transfer sink binary")
	fmt.Println("  3. Transfer or download config from", configSource)
	fmt.Println("  4. Execute: sink bootstrap <config>")
	if !noCleanup {
		fmt.Println("  5. Clean up temporary files")
	}
	fmt.Println()
	fmt.Println("For now, use the bash script: ./scripts/bootstrap-remote.sh")
	fmt.Println("Or manually SSH and run: sink bootstrap <url>")

	os.Exit(1)
}

// printRemoteHelp prints help for the remote command
func printRemoteHelp() {
	fmt.Print(`
sink remote - Deploy sink to remote hosts

Usage:
  sink remote deploy <target> <config-source> [options]

Subcommands:
  deploy              Deploy sink and config to remote host(s)

Arguments:
  target              SSH target (user@host or user@host:port)
                      Multiple targets: user@host1,user@host2
  config-source       Config file path or URL

Options:
  --dry-run          Show what would be executed without running
  --no-cleanup       Don't remove temporary files on remote
  --yes, -y          Skip confirmation prompt
  -h, --help         Show this help message

Description:
  The remote command deploys the sink binary and configuration to remote
  hosts via SSH, then executes the installation. This automates the full
  bootstrap process for new machines.

Deployment Process:
  1. Connect to target host via SSH
  2. Transfer sink binary to remote host
  3. Transfer config file or pass URL for download
  4. Execute sink bootstrap on remote host
  5. Clean up temporary files (unless --no-cleanup)

Security:
  - Uses SSH key-based authentication
  - Transfers over encrypted SSH connection
  - Supports GitHub URL pinning validation
  - Auto-checksum verification for remote downloads

Examples:
  # Deploy with local config file
  sink remote deploy user@host setup.json

  # Deploy with GitHub URL (pinned version)
  sink remote deploy user@host \
    https://raw.githubusercontent.com/org/configs/v1.0.0/prod.json

  # Deploy to multiple hosts
  sink remote deploy user@host1,user@host2 setup.json

  # Dry run to preview
  sink remote deploy user@host setup.json --dry-run

  # Skip confirmation
  sink remote deploy user@host setup.json --yes

  # Keep files for debugging
  sink remote deploy user@host setup.json --no-cleanup

Exit Codes:
  0    Success
  1    Error (connection failed, transfer failed, execution failed)

Current Status:
  ⚠️  SSH remote deployment is not yet fully implemented.
  
  For now, use the provided bash script:
    ./scripts/bootstrap-remote.sh user@host config.json

  Or manually SSH to the host and run:
    sink bootstrap <url-or-file>

Related Commands:
  sink bootstrap  - Bootstrap from URL on current host
  sink execute    - Execute local config file
  sink help       - Show general help

See Also:
  docs/REMOTE_BOOTSTRAP.md    - Remote deployment guide
  scripts/bootstrap-remote.sh - Bash implementation
`)
}
