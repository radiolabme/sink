# Sink - Shell Installation Kit

Sink is a declarative, idempotent shell command execution framework designed for system configuration and installation tasks. The tool provides a structured approach to managing system dependencies, environment setup, and deployment workflows through JSON-based configuration files.

## Overview

Modern system administration requires repeatable, reliable installation procedures that work across different platforms and environments. Sink addresses this need by providing a framework where operations are defined declaratively, checked for completion before execution, and safely previewed before making any system changes.

The framework is built entirely on the Go standard library with zero external dependencies, making it easy to distribute and deploy. Configuration files describe the desired system state using platform-specific commands, while the execution engine ensures operations are idempotent and provides real-time feedback through a structured event system.

## Core Features

Sink configurations are declarative JSON documents that define installation steps for specific platforms. The framework automatically detects the host platform and executes only the relevant steps. Before any action is taken, Sink checks whether the target state already exists, preventing redundant operations and ensuring safe re-execution.

The facts system allows configurations to query system state, such as available CPU cores, installed software versions, or environment variables. These facts can be referenced throughout the configuration using template syntax, enabling dynamic and context-aware installations.

Dry-run mode provides a complete preview of planned operations without making any changes. This includes showing which commands would execute, in what order, and with what context. Every execution displays the host, user, working directory, and platform details before proceeding.

Safety is enforced through explicit user confirmation. The system always requires an affirmative "yes" response before executing commands in non-dry-run mode. The execution context is displayed prominently to prevent accidental operations on the wrong host or with incorrect permissions.

Cross-platform support is built into the configuration format. A single configuration file can define different installation procedures for macOS, Linux distributions, and Windows, with automatic platform detection selecting the appropriate steps at runtime.

## Project Organization

The project follows a conventional Go structure with clear separation between source code, documentation, configuration files, and build artifacts.

Source code resides in the `src/` directory and includes the main entry point, configuration parser, execution engine, facts system, transport layer for command execution, and comprehensive test files. The JSON schema is embedded directly in the binary, eliminating external file dependencies.

Built binaries are placed in `bin/` after compilation. The `data/` directory contains example configurations demonstrating various patterns, from simple installations to complex multi-step setups with facts and conditionals. The schema file is maintained both embedded in the binary and as a reference copy for development tools.

Documentation is organized in `docs/` with focused guides on safety features, execution context, and planned future enhancements. Test artifacts including coverage reports and logs are stored in `test/` for analysis and debugging.

## Quick Start

Building Sink requires Go 1.23 or later and Make. If you don't have these tools installed, run the setup script to install them locally:

```bash
./setup.sh
```

The setup script checks your system for Go and Make. If both are already installed, it displays the available make targets and exits. If either tool is missing, it downloads and installs them to the local `.tools/` directory. After local installation, run `source .envrc` to add the tools to your PATH for the current terminal session.

Once the development tools are available, build Sink using Make:

```bash
make build
```

Alternatively, use Go commands directly:

```bash
go build -o bin/sink ./src/...
```

Once built, configurations can be validated, previewed, and executed. Start by previewing an example configuration in dry-run mode:

```bash
./bin/sink execute data/install-config.json --dry-run
```

The dry-run output shows the execution plan without making changes. This includes the platform detected, steps that would run, and any facts that would be gathered. To proceed with actual execution:

```bash
./bin/sink execute data/install-config.json
```

The system will display the execution context including hostname, current user, working directory, operating system, and architecture. A confirmation prompt requires an explicit "yes" response before proceeding.

Running tests verifies the installation and provides confidence in the build:

```bash
make test
```

For detailed coverage analysis:

```bash
make coverage
make coverage-html
open test/coverage.html
```

## Configuration Basics

Configurations are JSON documents with a version field and platform-specific installation definitions. For detailed examples and patterns, see **[examples/FAQ.md](examples/FAQ.md)**, which provides a comprehensive guide to Sink's features with focused, working examples.

The simplest configuration specifies steps for a single platform:

```json
{
  "version": "1.0",
  "name": "Example Installation",
  "platforms": [{
    "name": "macOS",
    "os": "darwin",
    "match": ".*",
    "install_steps": [
      {
        "name": "Check Homebrew",
        "check": "command -v brew",
        "on_missing": [
          {
            "command": "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
          }
        ]
      }
    ]
  }]
}
```

This configuration demonstrates the check-remediate pattern. The system first runs the check command. If it succeeds (exit code 0), the step is skipped since Homebrew is already installed. If it fails, the remediation commands in `on_missing` are executed.

The facts system enables dynamic configuration based on system state. Facts are gathered before step execution and can be referenced using template syntax:

```json
{
  "version": "1.0",
  "name": "Conditional Installation",
  "facts": {
    "package_name": {
      "command": "echo ${PACKAGE_NAME:-vim}"
    }
  },
  "platforms": [{
    "name": "macOS",
    "os": "darwin",
    "match": ".*",
    "install_steps": [
      {
        "name": "Install {{facts.package_name}}",
        "check": "command -v {{facts.package_name}}",
        "on_missing": [
          {"command": "brew install {{facts.package_name}}"}
        ]
      }
    ]
  }]
}
```

Facts can query environment variables, execute commands, or read files. The results are available throughout the configuration, enabling patterns like conditional installation, resource-aware configuration, and template-based command generation.

## Command Line Interface

Sink provides several commands for working with configurations. General help is available through:

```bash
sink --help
sink help
```

Command-specific help can be accessed in two ways:

```bash
sink help execute
sink execute --help
sink help bootstrap
sink bootstrap --help
```

The execute command runs a configuration file with optional platform override and dry-run mode:

```bash
sink execute config.json
sink execute config.json --dry-run
sink execute config.json --platform linux
```

The bootstrap command loads and executes configurations from remote URLs or local files, supporting HTTP, HTTPS, and GitHub URLs with optional checksum verification:

```bash
sink bootstrap https://example.com/config.json
sink bootstrap https://example.com/config.json --sha256 <hash>
sink bootstrap https://example.com/config.json --dry-run
```

The remote command deploys configurations to remote hosts via SSH (currently in development):

```bash
sink remote deploy config.json
```

Validation checks configuration syntax against the JSON schema:

```bash
sink validate config.json
```

The facts command shows what facts would be gathered without executing any steps:

```bash
sink facts config.json
```

The schema can be output for use with editors and validation tools:

```bash
sink schema > sink.schema.json
```

Version information is available through:

```bash
sink version
```

Each command provides detailed help accessible through `sink help <command>` or `sink <command> --help`.

## Testing and Quality

The project maintains comprehensive test coverage with over 270 test cases covering unit tests, integration scenarios, edge cases, and execution context validation. Tests can be run through the Makefile or directly with Go:

```bash
make test
```

Or:

```bash
go test ./src/... -v
```

Coverage reports help identify untested code paths:

```bash
make coverage
```

The HTML coverage report provides detailed visualization:

```bash
make coverage-html
open test/coverage.html
```

Current test coverage stands at approximately 53% of statements, with critical validation functions at 100% coverage. The test suite includes thorough validation of configuration parsing, platform detection, fact gathering, step execution, and error handling.

## Development Workflow

During development, automated test execution on file changes streamlines the feedback loop. Using watchexec:

```bash
watchexec -e go -c clear -- go test ./src/... -v
```

Verbose execution output aids in debugging configuration issues:

```bash
./bin/sink execute data/install-config.json --dry-run -v
```

The Makefile provides additional targets for common operations:

```bash
make build        # Compile the binary
make test         # Run all tests
make coverage     # Generate coverage report
make coverage-html # Create HTML coverage visualization
make clean        # Remove build artifacts
make install      # Install to /usr/local/bin
make demo         # Run demo configuration in dry-run
make help         # Show all available targets
```

## System Architecture

Sink is designed around five core components that work together to provide safe, repeatable system configuration.

The executor orchestrates step execution with full context discovery. Before running any commands, it gathers information about the host, user, working directory, and platform. This context is displayed and confirmed before proceeding.

The transport layer abstracts command execution, currently supporting local execution with plans for SSH support. This abstraction allows the same configuration to target local or remote systems without modification.

The facts system resolves template variables from various sources including environment variables, command output, and file contents. Facts are gathered once at the beginning of execution and remain constant throughout.

Configuration management handles JSON parsing, validation against the embedded schema, and platform selection. The platform detection logic automatically selects the appropriate installation steps based on the current operating system and distribution.

The event system provides real-time progress updates through structured callbacks. This enables rich output formatting, progress indicators, and integration with external monitoring systems.

## Design Principles

The framework follows several key principles that guide its implementation and use.

Zero external dependencies ensure easy distribution. The entire system is built on the Go standard library, requiring no package management or dependency resolution at runtime.

Idempotency is enforced through the check-before-action pattern. Every operation should verify whether the desired state exists before attempting changes. This allows configurations to be run multiple times safely.

Safety takes precedence through context display and explicit confirmation. The system shows exactly where and how commands will execute, requiring affirmative user action before proceeding.

Comprehensive testing provides confidence in reliability. The test suite covers core functionality, edge cases, and integration scenarios, with continuous focus on improving coverage.

Interface-based design enables extension. The transport system uses interfaces allowing new execution backends to be added without modifying core logic.

## Examples

The `examples/` directory contains focused, production-ready examples demonstrating each Sink feature clearly:

- **[01-basic.json](examples/01-basic.json)** - Your first Sink configuration with simple validation
- **[02-multi-platform.json](examples/02-multi-platform.json)** - Cross-platform support (macOS, Linux, Windows)
- **[03-distributions.json](examples/03-distributions.json)** - Linux distribution detection and package management
- **[04-facts.json](examples/04-facts.json)** - System fact gathering and template substitution
- **[05-nested-steps.json](examples/05-nested-steps.json)** - Conditional execution with check/on_missing patterns
- **[06-retry.json](examples/06-retry.json)** - Retry logic for handling transient failures
- **[07-defaults.json](examples/07-defaults.json)** - Reusable configurations with default values
- **[08-error-handling.json](examples/08-error-handling.json)** - Different error handling patterns
- **[09-verbose-debugging.json](examples/09-verbose-debugging.json)** - Verbose output for debugging command execution
- **[10-sleep-rate-limiting.json](examples/10-sleep-rate-limiting.json)** - Sleep delays for rate limiting and service startup
- **[11-advanced-timeout.json](examples/11-advanced-timeout.json)** - Advanced timeout with custom error codes

### Advanced Execution Control

Sink supports several advanced features for fine-grained control over command execution:

**Verbose Mode** - Enable verbose output logging to stderr for debugging. Set `"verbose": true` on any command, fact, or remediation step to see detailed execution information including exit codes and output.

**Sleep Delays** - Add delays after command execution using the `"sleep"` property. Useful for rate limiting API calls, waiting for services to start, or spacing operations. Supports any Go duration format (e.g., `"1s"`, `"500ms"`, `"2m"`).

**Advanced Timeouts** - Configure retry timeouts with custom error codes. Use a simple string (`"timeout": "30s"`) or an object (`"timeout": {"interval": "30s", "error_code": 124}`) to specify both duration and exit code. Custom error codes help distinguish timeout failures from other errors.

Each example is self-contained and can be run independently. For detailed explanations, use cases, and best practices, see **[examples/FAQ.md](examples/FAQ.md)** and **[docs/configuration-reference.md](docs/configuration-reference.md)**, which provide comprehensive guides to all Sink features.

Quick example validation:

```bash
# Validate all examples
for f in examples/0*.json; do ./bin/sink validate "$f"; done

# Try the basic example
./bin/sink execute examples/01-basic.json --dry-run
./bin/sink execute examples/01-basic.json
```

## Future Development

Several enhancements are planned to extend Sink's capabilities beyond local execution.

Remote bootstrap functionality is already implemented, allowing configurations to be loaded from HTTP/HTTPS URLs with optional SHA256 checksum verification. GitHub URL pinning is supported to ensure configurations are loaded from specific, immutable versions.

SSH transport is under active development to enable remote system configuration through secure channels. The same configurations that work locally will be deployable to remote hosts over SSH connections.

A REST API is planned to provide HTTP access to Sink functionality with Server-Sent Events for real-time progress streaming. This will enable web-based interfaces and integration with deployment orchestration systems.

Execution guards will add configuration-based safety rules. These rules can restrict execution based on hostname patterns, user requirements, or custom validation logic, preventing accidental deployment to production systems.

Enhanced logging capabilities will provide full audit trails of all operations, including command execution, output capture, and timing information. This supports compliance requirements and troubleshooting.

Comprehensive architectural documentation is available in `src/ARCHITECTURE.md`, covering system design, security model, and future roadmap.

## Makefile Reference

The Makefile automates common development tasks. Building the project requires running `make build`, which compiles the source code into the `bin/sink` binary for the current platform. Cross-compilation targets include `make build-static` for static Linux binaries with no external dependencies, `make build-linux` for dynamic Linux binaries, and `make build-all` to compile for all supported platforms and architectures.

During development, `make test` executes the full test suite to verify functionality, while `make coverage` and `make coverage-html` generate coverage statistics and visual reports respectively.

Maintenance operations include `make clean` to remove build artifacts and test output, and `make install` to copy the binary to `/usr/local/bin` for system-wide access. Demonstration targets include `make demo` and `make demo-install` which run example configurations in dry-run mode for safe exploration. The `make help` target displays all available targets with detailed descriptions.

## Contributing

Contributions follow standard Go project conventions. All changes must include appropriate tests to maintain or improve code coverage. Code style should match existing patterns throughout the project. Documentation updates must accompany functional changes to keep the project accessible and maintainable.
