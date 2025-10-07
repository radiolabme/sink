# Development Environment Setup

This document explains how to set up your development environment for building and testing Sink.

## Prerequisites

Sink requires two tools for development:

1. **Go** (version 1.19 or later) - For building the project
2. **Make** - For running build tasks and tests

## Quick Setup

Run the setup script from the project root:

```bash
./setup.sh
```

### What the Script Does

The setup script performs the following checks and actions:

1. **Detects your platform** - Linux or macOS (Darwin), AMD64 or ARM64
2. **Checks for Go** - Looks in system PATH and local `.tools/go/`
3. **Checks for Make** - Looks in system PATH and local `.tools/bin/`
4. **Runs make help** - If both tools are found, shows available targets and exits
5. **Installs locally** - If either tool is missing, downloads and installs to `.tools/`

### If Tools Are Already Installed

When both Go and Make are already available in your system PATH, the script simply displays the available make targets and exits. No downloads or installations occur.

### If Tools Are Missing

When Go or Make are not found, the script installs them locally:

- **Go** is downloaded from golang.org and installed to `.tools/go/`
- **Make** on macOS triggers Xcode Command Line Tools installation
- **Make** on Linux displays package manager installation commands

After local installation, you must add the tools to your PATH:

```bash
source .envrc
```

This command adds the local Go and Make to your PATH for the current terminal session. You need to run this each time you open a new terminal, or add the PATH export to your shell profile (`~/.zshrc` or `~/.bashrc`).

The `.envrc` file is created automatically by the setup script and contains:

```bash
export PATH="${SCRIPT_DIR}/.tools/go/bin:${SCRIPT_DIR}/.tools/bin:${PATH}"
export GOPATH="${SCRIPT_DIR}/.go"
```

## Building Sink

Once your development environment is set up, build the project:

```bash
make build
```

This creates the `bin/sink` binary for your current platform.

For other build options:

```bash
make build-static    # Static Linux binary (portable)
make build-linux     # Dynamic Linux binary
make build-all       # All platforms (macOS/Linux, AMD64/ARM64)
```

See `docs/BUILD.md` for detailed information about build options and static vs dynamic linking.

## Running Tests

Run the test suite:

```bash
make test
```

Generate coverage reports:

```bash
make coverage        # Terminal coverage report
make coverage-html   # HTML coverage report
```

## Verifying Your Setup

To verify everything is working correctly:

1. Run the setup script: `./setup.sh`
2. If tools were installed locally: `source .envrc`
3. Build the project: `make build`
4. Run tests: `make test`
5. Try a demo: `make demo`

All commands should complete without errors.

## Troubleshooting

### "command not found: go" or "command not found: make"

After local installation, you must run `source .envrc` to add the tools to your PATH. This command updates your current terminal session only.

To avoid running `source .envrc` every time, add this to your `~/.zshrc` or `~/.bashrc`:

```bash
# Add Sink development tools to PATH
export PATH="/path/to/sink/.tools/go/bin:/path/to/sink/.tools/bin:${PATH}"
```

Replace `/path/to/sink` with the actual path to your Sink project directory.

### macOS: "xcode-select: error: command line tools are already installed"

This is normal. macOS provides Make through Xcode Command Line Tools. If they're already installed, the setup script will detect Make and proceed.

### Linux: Make not found

On Linux, Make is typically installed via the system package manager. The setup script displays the appropriate commands for your distribution:

```bash
# Debian/Ubuntu
sudo apt-get install build-essential

# RHEL/CentOS/Fedora
sudo yum groupinstall 'Development Tools'

# Alpine
sudo apk add make gcc musl-dev
```

### Go version too old

Sink requires Go 1.23 or later. If your system has an older version, the setup script will install Go 1.23.4 locally to `.tools/go/`. Run `source .envrc` to use the local version.

## Clean Installation

To remove all local tools and start fresh:

```bash
rm -rf .tools/ .go/ .envrc
./setup.sh
```

This removes all locally installed tools and regenerates the environment setup.
