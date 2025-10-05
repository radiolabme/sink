# Sink - Shell Installation Kit

A declarative, idempotent shell command execution framework for system configuration and installation tasks.

## Features

- ✅ **Declarative Configuration**: Define steps in JSON with platform-specific commands
- ✅ **Idempotent Execution**: Checks before running actions to avoid redundant operations
- ✅ **Cross-Platform**: Supports macOS, Linux, and Windows with platform detection
- ✅ **Facts System**: Query system state (environment, files, commands) before execution
- ✅ **Dry-Run Mode**: Preview what would execute without making changes
- ✅ **Execution Context**: Always shows WHERE commands will run (host, user, directory)
- ✅ **Safety Confirmation**: Requires explicit "yes" before real execution
- ✅ **Event System**: Real-time progress updates with structured events
- ✅ **Zero Dependencies**: Pure Go stdlib implementation

## Project Structure

```
sink/
├── src/              # Go source files
│   ├── main.go       # CLI entry point
│   ├── executor.go   # Core execution engine
│   ├── config.go     # Configuration parsing
│   ├── facts.go      # Facts system
│   ├── transport.go  # Command execution layer
│   ├── types.go      # Type definitions
│   └── *_test.go     # Test files
├── bin/              # Built binaries
│   └── sink          # Compiled binary
├── data/             # Configuration files
│   ├── install-config.json           # Example config
│   ├── install-config-with-facts.json # Config with facts
│   ├── demo-config.json              # Demo configuration
│   └── install-config.schema.json    # JSON schema
├── docs/             # Documentation
│   ├── EXECUTION_CONTEXT_SAFETY.md   # Safety features
│   ├── REST_AND_SSH.md               # Future features guide
│   └── *.md                          # Additional docs
└── test/             # Test artifacts (coverage, logs)
    ├── coverage.out  # Coverage data
    └── coverage.html # Coverage report
```

## Quick Start

### Build

```bash
make build
# or
go build -o bin/sink ./src/...
```

### Run (Dry-Run)

```bash
./bin/sink execute data/install-config.json --dry-run
```

### Run (Real Execution)

```bash
./bin/sink execute data/install-config.json
# You'll see:
# 🔍 Execution Context:
#    Host:      your-hostname
#    User:      your-username
#    ...
# ⚠️  You are about to execute N steps on your-hostname as your-username
#    Continue? [yes/no]: yes
```

### Test

```bash
make test
# or
go test ./src/... -v
```

### Coverage

```bash
make coverage
# or
make coverage-html  # Opens HTML report
```

## Configuration Format

```json
{
  "version": "1.0",
  "name": "Example Installation",
  "steps": [
    {
      "name": "Check Homebrew",
      "check": {
        "command": "which brew"
      },
      "action": {
        "macos": "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
      }
    }
  ]
}
```

### With Facts

```json
{
  "version": "1.0",
  "name": "Conditional Installation",
  "steps": [
    {
      "name": "Install package {{package}}",
      "facts": {
        "package": {
          "env": "PACKAGE_NAME",
          "default": "vim"
        }
      },
      "check": {
        "command": "which {{package}}"
      },
      "action": {
        "macos": "brew install {{package}}",
        "linux": "apt-get install -y {{package}}"
      }
    }
  ]
}
```

## Makefile Targets

- `make build` - Build the binary
- `make test` - Run all tests
- `make coverage` - Run tests with coverage
- `make coverage-html` - Generate HTML coverage report
- `make clean` - Remove build artifacts
- `make install` - Install to /usr/local/bin
- `make demo` - Run demo config (dry-run)
- `make help` - Show all targets

## Execution Context Safety

Sink always discovers and displays the execution context before running commands:

- **Host**: Hostname where commands execute
- **User**: Current user
- **Work Dir**: Current working directory
- **OS/Arch**: Operating system and architecture
- **Transport**: Execution method (local, ssh, etc.)

This prevents accidentally running commands on the wrong host or as the wrong user.

## Testing

- **116 tests** covering all major functionality
- **~51% code coverage** (50.8% after reorganization)
- Tests include: unit tests, integration tests, edge cases, context tests
- Run with: `make test` or `go test ./src/... -v`

## Development

### Run Tests in Watch Mode

```bash
# Using watchexec (install with: brew install watchexec)
watchexec -e go -c clear -- go test ./src/... -v
```

### Debug with Verbose Output

```bash
./bin/sink execute data/install-config.json --dry-run -v
```

### View Coverage Report

```bash
make coverage-html
open test/coverage.html
```

## Architecture

### Core Components

1. **Executor**: Orchestrates step execution with context discovery
2. **Transport**: Abstracts command execution (local, future: SSH)
3. **Facts**: Template variable resolution from environment/files/commands
4. **Config**: JSON-based declarative configuration
5. **Events**: Real-time execution progress callbacks

### Key Design Principles

- **Zero Dependencies**: Only Go stdlib
- **Idempotency**: Check before action
- **Safety First**: Context display + confirmation
- **Testability**: Comprehensive test coverage
- **Extensibility**: Interface-based transport system

## Future Enhancements

See `docs/REST_AND_SSH.md` for planned features:

- **SSH Transport**: Remote execution over SSH
- **REST API**: HTTP server with SSE streaming
- **Execution Guards**: Config-based safety rules (hostname patterns, user restrictions)
- **Enhanced Logging**: Full audit trails

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
