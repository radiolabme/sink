# CLI Reference

Complete reference for all Sink command-line interface commands and flags.

## Getting Help

Sink provides comprehensive built-in help for all commands:

```bash
# General help
sink --help
sink -h
sink help

# Command-specific help (two methods)
sink help <command>
sink <command> --help

# Examples
sink help execute
sink execute --help
sink facts --help
```

For detailed help system documentation, see: `docs/CLI_HELP_SYSTEM.md`

## Table of Contents

- [Getting Help](#getting-help)
- [Commands](#commands)
  - [execute](#execute)
  - [facts](#facts)
  - [validate](#validate)
  - [schema](#schema)
  - [version](#version)
  - [help](#help)
- [Global Flags](#global-flags)
- [Exit Codes](#exit-codes)
- [Environment Variables](#environment-variables)

---

## Commands

### execute

Execute a Sink configuration.

**Usage:**
```bash
sink execute <config-file> [flags]
sink exec <config-file> [flags]  # Short alias
```

**Arguments:**
- `<config-file>` - Path to JSON configuration file

**Flags:**
- `--dry-run` - Preview execution without running commands
- `--platform <os>` - Override platform detection (`darwin`, `linux`, `windows`)
- `-h, --help` - Show detailed help for execute command

**Examples:**
```bash
# Execute configuration
sink execute install.json

# Dry-run mode (preview only)
sink execute install.json --dry-run

# Override platform detection
sink execute install.json --platform linux

# Short alias
sink exec install.json

# Get help
sink execute --help
```

**Behavior:**
1. Validates configuration
2. Gathers facts from system
3. Detects platform (or uses `--platform` override)
4. Shows execution plan
5. Prompts for confirmation
6. Executes install steps sequentially
7. Shows real-time progress
8. Reports success or failure

**See also:** `sink help execute` for comprehensive help

---

### facts

Gather and display facts from a configuration.

**Usage:**
```bash
sink facts <config-file>
```

**Arguments:**
- `<config-file>` - Path to JSON configuration file with facts section

**Flags:**
- `-h, --help` - Show detailed help for facts command

**Examples:**
```bash
# Display facts
sink facts install.json

# Export to shell environment
eval $(sink facts install.json | grep "export")

# Get help
sink facts --help
```

**Output:**
For each fact displays:
- Name and value
- Type (string, bool, int)
- Export variable (if defined)
- Description (if defined)

**See also:** `sink help facts` for comprehensive help

---

### validate

Validate a configuration file against the JSON schema.

**Usage:**
```bash
sink validate <config-file>
```

**Arguments:**
- `<config-file>` - Path to JSON configuration file

**Flags:**
- `-h, --help` - Show detailed help for validate command

**Examples:**
```bash
# Validate a configuration
sink validate install.json

# Validate in CI/CD pipeline
for config in configs/*.json; do
  sink validate "$config" || exit 1
done

# Get help
sink validate --help
```

**Output:**
- ✅ Validation success with configuration summary
- ❌ Validation errors with details

**Exit Codes:**
- 0 on success
- 1 on failure

**See also:** `sink help validate` for comprehensive help

---

### schema

Output the JSON schema to stdout.

**Usage:**
```bash
sink schema
```

**Flags:**
- `-h, --help` - Show detailed help for schema command

**Examples:**
```bash
# View schema
sink schema

# Save to file
sink schema > sink.schema.json

# Use with jq
sink schema | jq '.properties'
sink schema | jq '."$defs"'
```

**Description:**
The schema is embedded in the Sink binary at compile time. This command outputs the complete JSON Schema that can be used for:
- Editor autocompletion and validation
- External validation tools
- Documentation generation
- Understanding configuration structure

**Schema Location:**
- Source: `src/sink.schema.json`
- Online: `https://raw.githubusercontent.com/radiolabme/sink/main/src/sink.schema.json`
- Versioned: `https://raw.githubusercontent.com/radiolabme/sink/v0.1.0/src/sink.schema.json`

**See also:** `sink help schema` for comprehensive help

---

### version

Show version information.

**Usage:**
```bash
sink version
sink -v
sink --version
```

**Output:**
```
sink version 0.1.0
```

---

### help

Show help for commands.

**Usage:**
```bash
sink help [command]
sink --help
sink -h
```

**Examples:**
```bash
# General help
sink help

# Command-specific help
sink help execute
sink help facts
sink help validate
sink help schema
```

**Description:**
Displays comprehensive help for commands including:
- Usage syntax
- Available options
- Detailed examples
- Related commands

**Alternative:**
Each command also supports the `--help` flag:
```bash
sink execute --help
sink facts --help
```

---

## Global Flags

The following flags work with any command:

- `-h, --help` - Show help for the command
- `-v, --version` - Show version (only at top level: `sink -v`)

---
```bash
sink facts <config-file> [flags]
sink facts - [flags]  # Read from stdin
```

**Arguments:**
- `<config-file>` - Path to JSON configuration file, or `-` for stdin

**Flags:**
- `--platform <os>` - Override platform detection

**Examples:**
```bash
# Show facts
sink facts install.json

# Show facts for specific platform
sink facts install.json --platform linux
```

**Output:**
```bash
$ sink facts install.json
Facts:
  cpu_count: 8
  total_ram_gb: 16
  hostname: macbook-pro.local
```

---

### version

Display Sink version information.

**Usage:**
```bash
sink version
```

**Output:**
```bash
$ sink version
sink version 0.1.0
```

---

### help

Display help information.

**Usage:**
```bash
sink help
sink --help
sink -h
sink <command> --help
```

**Examples:**
```bash
# General help
sink help

# Command-specific help
sink execute --help
sink validate --help
```

---

## Global Flags

Flags that work with any command.

### --help, -h

Display help for a command.

```bash
sink --help
sink execute --help
```

---

## Exit Codes

Sink uses standard exit codes:

| Code | Meaning | When It Happens |
|------|---------|----------------|
| `0` | Success | Command completed successfully |
| `1` | General error | Configuration invalid, command failed, platform unsupported |
| `2` | Usage error | Invalid command-line arguments |
| `130` | User interrupt | User pressed Ctrl+C |

**Examples:**

```bash
# Success
$ sink execute install.json
$ echo $?
0

# Validation error
$ sink validate bad.json
Validation errors: ...
$ echo $?
1

# User canceled
$ sink execute install.json
Continue? (y/n) n
Canceled by user
$ echo $?
130
```

---

## Environment Variables

### SINK_CACHE_DIR

Customize the cache directory location.

**Default:** `~/.cache/sink/`

**Usage:**
```bash
export SINK_CACHE_DIR=/tmp/sink-cache
sink execute install.json
```

### NO_COLOR

Disable colored output.

**Usage:**
```bash
export NO_COLOR=1
sink execute install.json
```

### Configuration Variables

You can reference environment variables in your configurations:

```json
{
  "facts": {
    "home": {"command": "echo $HOME"},
    "user": {"command": "echo $USER"}
  }
}
```

---

## Shell Integration

### Bash Completion

```bash
# Add to ~/.bashrc
eval "$(sink completion bash)"
```

### Zsh Completion

```zsh
# Add to ~/.zshrc
eval "$(sink completion zsh)"
```

---

## Common Patterns

### Validate Before Execute

```bash
sink validate config.json && sink execute config.json
```

### Dry-Run First

```bash
sink execute config.json --dry-run
# Review output, then:
sink execute config.json
```

### Silent Execution

```bash
sink execute config.json --yes > /dev/null 2>&1
```

### Check Facts

```bash
sink facts config.json
```

---

## Debugging

### Verbose Output

Sink shows all command output by default. To reduce noise:

```bash
sink execute config.json 2>&1 | grep -v "Warning"
```

### Check Configuration

```bash
# Validate syntax
sink validate config.json

# Check facts
sink facts config.json

# Dry-run
sink execute config.json --dry-run
```

---

[← Back: Configuration Reference](configuration-reference.md) | [Up: Docs](README.md) | [Next: Best Practices →](best-practices.md)