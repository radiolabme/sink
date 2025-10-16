# Configuration Reference

Complete reference for the Sink configuration schema (version 1.0.0).

## Table of Contents

- [Schema Overview](#schema-overview)
- [Root Schema](#root-schema)
- [Facts](#facts)
- [Platforms](#platforms)
- [Install Steps](#install-steps)
- [Remediation Steps](#remediation-steps)
- [Complete Examples](#complete-examples)

---

## Schema Overview

A Sink configuration describes **what to install** and **how to install it** across different platforms. The schema has a hierarchical structure:

```
Config (root)
├── version: "1.0.0"              # Required: Schema version
├── description: "..."            # Optional: Human description
├── facts: {}                     # Optional: System queries (evaluated ONCE at start)
│   └── fact_name:
│       ├── command: "..."        # Command to run
│       ├── platforms: [...]      # Optional: Only gather on these OSes
│       ├── type: "string"        # Optional: Value type
│       └── ...
├── defaults: {}                  # Optional: Default values
├── platforms: []                 # Required: Platform-specific configs
│   └── Platform:
│       ├── os: "darwin"          # Platform identifier
│       ├── match: "darwin*"      # Shell pattern for uname
│       ├── name: "macOS"         # Human name
│       ├── install_steps: []    # Direct steps OR...
│       └── distributions: []    # Linux distribution-specific
│           └── Distribution:
│               ├── ids: [...]   # Distribution IDs
│               └── install_steps: []
└── fallback: {}                  # Optional: Global error for unsupported platforms
```

### Key Concepts

**Facts are Global with Platform Filtering:**
- Facts are defined at the **root level** (`config.facts`)
- All facts are evaluated **once** before any install steps run
- Facts can be **filtered by platform** using the `platforms` field in the fact definition
- Facts are **available to all platforms** that match the filter (or all if no filter)

**Platform Detection:**
1. Sink detects your OS using `runtime.GOOS` (or `--platform` override)
2. Finds matching platform configuration using `match` pattern
3. For Linux, further matches against distribution IDs from `/etc/os-release`

**Execution Flow:**
1. **Load & Validate** configuration
2. **Detect Platform** (OS + distribution if Linux)
3. **Gather Facts** (filtered by platform, all evaluated once)
4. **Execute Install Steps** sequentially (facts available via `{{facts.name}}`)

**Example showing facts inheritance:**

```json
{
  "version": "1.0.0",
  "facts": {
    "hostname": {
      "command": "hostname",
      "description": "Available to ALL platforms"
    },
    "cpu_count_mac": {
      "command": "sysctl -n hw.ncpu",
      "platforms": ["darwin"],
      "description": "Only gathered on macOS"
    },
    "cpu_count_linux": {
      "command": "nproc",
      "platforms": ["linux"],
      "description": "Only gathered on Linux"
    }
  },
  "platforms": [
    {
      "os": "darwin",
      "match": "darwin*",
      "name": "macOS",
      "install_steps": [
        {
          "name": "Show facts",
          "command": "echo 'Host: {{facts.hostname}}, CPUs: {{facts.cpu_count_mac}}'"
        }
      ]
    },
    {
      "os": "linux",
      "match": "linux*",
      "name": "Linux",
      "install_steps": [
        {
          "name": "Show facts",
          "command": "echo 'Host: {{facts.hostname}}, CPUs: {{facts.cpu_count_linux}}'"
        }
      ]
    }
  ]
}
```

When running on macOS:
- `hostname` is gathered (no platform filter)
- `cpu_count_mac` is gathered (platform filter matches)
- `cpu_count_linux` is **skipped** (platform filter doesn't match)
- Install steps can use `{{facts.hostname}}` and `{{facts.cpu_count_mac}}`

When running on Linux:
- `hostname` is gathered
- `cpu_count_linux` is gathered
- `cpu_count_mac` is **skipped**
- Install steps can use `{{facts.hostname}}` and `{{facts.cpu_count_linux}}`

---

## Root Schema

Top-level configuration structure.

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Semantic version of configuration format (e.g., `"1.0.0"`) |
| `platforms` | array | List of platform-specific configurations |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `$schema` | string | Reference to JSON schema for validation |
| `description` | string | Human-readable description of this configuration |
| `facts` | object | Declarative fact gathering definitions |
| `defaults` | object | Default values across all platforms |
| `fallback` | object | Global fallback error for unsupported platforms |
| `bootstrap` | object | Remote deployment configuration (see [Bootstrap](#bootstrap)) |

### Example

```json
{
  "$schema": "../src/sink.schema.json",
  "version": "1.0.0",
  "description": "Install jq JSON processor",
  "facts": {},
  "bootstrap": {},
  "platforms": []
}
```

---

## Bootstrap

Remote deployment configuration with GitHub URL pinning support.

### Bootstrap Object

The `bootstrap` section enables declarative remote deployment with security policies.

**See also:** 
- Schema definition: `src/sink.schema.json` (automatically embedded in binary)
- Go tests: `src/github_test.go` (GitHub URL pin detection tests)
- Examples:
  - `examples/bootstrap-github-pinned.json` - Release tag pinning
  - `examples/bootstrap-github-commit.json` - Commit SHA pinning
  - `examples/bootstrap-github-release.json` - GitHub Releases with verification
- Documentation:
  - `docs/BOOTSTRAP_CONFIG_SCHEMA.md` - Complete schema reference
  - `docs/GITHUB_URL_PINNING.md` - GitHub pinning guide
  - `docs/GITHUB_PINNING_QUICKSTART.md` - Quick start guide

### Security Object

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `allowed_sources` | array[string] | - | Allowlist of URL patterns (supports glob) |
| `require_signatures` | boolean | `false` | Require GPG signatures for all remote configs |
| `require_https` | boolean | `true` | Require HTTPS for all remote configs |
| `require_pinning` | boolean | `false` | Require GitHub URLs to be pinned (no mutable branches) |
| `trusted_keys` | array[string] | - | List of trusted GPG key IDs |

### Remote Config Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ❌ | Human-readable name for this config source |
| `url` | string | ✅ | URL to fetch config from (supports GitHub pinning) |
| `checksum_url` | string | ❌ | URL to SHA256 checksum (auto: `{url}.sha256`) |
| `signature_url` | string | ❌ | URL to GPG signature (auto: `{url}.asc`) |
| `pin` | object | ❌ | GitHub URL pinning configuration |
| `verification` | object | ❌ | Verification settings |

### GitHub Pin Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | enum | ✅ | `"tag"`, `"commit"`, or `"branch"` |
| `value` | string | ✅ | Tag name, commit SHA, or branch name |
| `repository` | string | ✅ | GitHub repository (`owner/repo`) |
| `require_immutable` | boolean | `true` | Fail if pin is mutable (branch) |

### Verification Object

| Field | Type | Description |
|-------|------|-------------|
| `checksums` | object | Expected checksums: `sha256`, `sha512`, `blake2b` |
| `gpg_key` | string | GPG key ID for signature verification |
| `auto_fetch_checksum` | boolean | Auto-fetch `.sha256` file (default: `true`) |
| `max_age_seconds` | integer | Maximum age for freshness check |

### SSH Object

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `targets` | array[string] | - | SSH targets (`user@host` or `user@host:port`) |
| `parallel` | boolean | `false` | Execute on targets in parallel |
| `timeout` | string | `"5m"` | SSH connection timeout |
| `retry` | integer | `0` | Number of retry attempts |

### Bootstrap Examples

**GitHub Release Tag (Recommended):**
```json
{
  "bootstrap": {
    "security": {
      "require_pinning": true,
      "require_https": true
    },
    "remote_configs": [{
      "name": "Production Config",
      "url": "https://raw.githubusercontent.com/myorg/configs/v1.0.0/prod.json",
      "pin": {
        "type": "tag",
        "value": "v1.0.0",
        "repository": "myorg/configs",
        "require_immutable": true
      },
      "verification": {
        "auto_fetch_checksum": true
      }
    }],
    "ssh": {
      "targets": ["deploy@prod-1.example.com"],
      "timeout": "5m"
    }
  }
}
```

**Commit SHA (Maximum Security):**
```json
{
  "bootstrap": {
    "remote_configs": [{
      "url": "https://raw.githubusercontent.com/myorg/configs/abc123def456/prod.json",
      "pin": {
        "type": "commit",
        "value": "abc123def456...",
        "repository": "myorg/configs"
      }
    }]
  }
}
```

See `examples/bootstrap-*.json` for complete working examples.

---

## Facts

Facts query system state before installation steps execute.

### How Facts Work

1. **Defined at root level** - Facts are part of the `Config` object, not individual platforms
2. **Evaluated once** - All facts (that match platform filters) run before any install steps
3. **Platform-filtered** - Use `platforms` field to only gather facts on specific OSes
4. **Globally available** - Once gathered, facts are available to all install steps via `{{facts.name}}`

### Fact Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | ✅ | Shell command to gather this fact |
| `description` | string | ❌ | Human-readable description |
| `export` | string | ❌ | Environment variable name to export (must match `^[A-Z_][A-Z0-9_]*$`) |
| `type` | enum | ❌ | Value type: `"string"`, `"boolean"`, `"integer"` (default: `"string"`) |
| `transform` | object | ❌ | Map input values to output values (string type only) |
| `strict` | boolean | ❌ | Fail if output not in transform map (default: `false`) |
| `platforms` | array | ❌ | Only gather on specified platforms: `["darwin", "linux", "windows"]` |
| `required` | boolean | ❌ | Fail if fact cannot be gathered (default: `false`) |
| `timeout` | object | ❌ | Timeout configuration with `interval` (duration string) and `error_code` (int) |
| `sleep` | string | ❌ | Duration to sleep after gathering fact (e.g., `"1s"`, `"500ms"`) |
| `verbose` | boolean | ❌ | Enable verbose output for this fact's execution (default: `false`) |

### String Facts

```json
{
  "facts": {
    "hostname": {
      "command": "hostname",
      "description": "System hostname",
      "type": "string"
    }
  }
}
```

### Integer Facts

```json
{
  "facts": {
    "cpu_count": {
      "command": "nproc",
      "description": "Number of CPU cores",
      "type": "integer"
    }
  }
}
```

### Boolean Facts

```json
{
  "facts": {
    "has_docker": {
      "command": "command -v docker",
      "description": "Docker is installed",
      "type": "boolean"
    }
  }
}
```

### Transformed Facts

Map command output to standardized values:

```json
{
  "facts": {
    "arch": {
      "command": "uname -m",
      "description": "CPU architecture",
      "transform": {
        "x86_64": "amd64",
        "aarch64": "arm64",
        "arm64": "arm64"
      },
      "strict": false
    }
  }
}
```

### Platform-Specific Facts

```json
{
  "facts": {
    "cpu_count": {
      "command": "sysctl -n hw.ncpu",
      "platforms": ["darwin"],
      "description": "CPU count (macOS only)"
    }
  }
}
```

### Facts with Timeout and Sleep

```json
{
  "facts": {
    "slow_query": {
      "command": "curl -s https://api.example.com/status",
      "description": "API status check",
      "timeout": {
        "interval": "10s",
        "error_code": 124
      },
      "sleep": "2s",
      "verbose": true
    },
    "database_ready": {
      "command": "pg_isready -q",
      "type": "boolean",
      "timeout": {
        "interval": "5s",
        "error_code": 1
      }
    }
  }
}
```

**Timeout Configuration:**
- `interval`: Duration string (e.g., `"30s"`, `"2m"`, `"1h"`)
- `error_code`: Exit code to return on timeout (default: platform-specific)

**Sleep Configuration:**
- Pauses execution after the fact is gathered
- Useful for rate limiting or waiting for system stabilization
- Duration string format: `"500ms"`, `"1s"`, `"30s"`, `"2m"`

**Verbose Output:**
- When `verbose: true`, displays detailed execution information
- Shows command being executed, stdout/stderr, exit codes
- Useful for debugging complex fact gathering

### Using Facts in Commands

Reference facts with `{{facts.name}}`:

```json
{
  "install_steps": [
    {
      "name": "Show CPU count",
      "command": "echo 'CPUs: {{facts.cpu_count}}'"
    }
  ]
}
```

### Fact Name Rules

- Must match pattern: `^[a-z_][a-z0-9_]*$`
- Lowercase letters, numbers, underscores only
- Must start with lowercase letter or underscore

**Valid:** `cpu_count`, `total_ram`, `my_fact_1`  
**Invalid:** `CPUCount`, `1fact`, `my-fact`

---

## Platforms

Platform-specific configurations.

### Platform Object (Simple)

For non-Linux platforms or when distribution detection isn't needed:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `os` | string | ✅ | OS identifier: `"darwin"`, `"linux"`, `"windows"` |
| `match` | string | ✅ | Shell pattern to match `uname -s` output |
| `name` | string | ✅ | Human-readable platform name |
| `install_steps` | array | ✅ | Array of install step objects |
| `required_tools` | array | ❌ | List of required command-line tools |
| `fallback` | object | ❌ | Fallback error for unsupported variants |

### Platform Object (Linux with Distributions)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `os` | string | ✅ | Must be `"linux"` |
| `match` | string | ✅ | Shell pattern (typically `"linux*"`) |
| `name` | string | ✅ | Human-readable name |
| `distributions` | array | ✅ | Array of distribution objects |
| `fallback` | object | ❌ | Fallback error for unsupported distributions |

### Distribution Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ids` | array | ✅ | Distribution IDs from `/etc/os-release` |
| `name` | string | ✅ | Human-readable distribution name |
| `install_steps` | array | ✅ | Array of install step objects |

### Platform Examples

**macOS:**
```json
{
  "os": "darwin",
  "match": "darwin*",
  "name": "macOS",
  "install_steps": [...]
}
```

**Linux with Distributions:**
```json
{
  "os": "linux",
  "match": "linux*",
  "name": "Linux",
  "distributions": [
    {
      "ids": ["ubuntu", "debian"],
      "name": "Debian-based",
      "install_steps": [...]
    },
    {
      "ids": ["fedora", "rhel", "centos"],
      "name": "Red Hat-based",
      "install_steps": [...]
    }
  ]
}
```

### Distribution IDs

Common IDs from `/etc/os-release`:

| Distribution | ID |
|--------------|-----|
| Ubuntu | `ubuntu` |
| Debian | `debian` |
| Fedora | `fedora` |
| RHEL | `rhel` |
| CentOS | `centos` |
| Alpine | `alpine` |
| Arch Linux | `arch` |

---

## Install Steps

Commands that execute during installation.

### Step Types

1. **Command Execution** - Run a command
2. **Check with Error** - Check condition, fail with error if check fails
3. **Check with Remediation** - Check condition, run remediation if check fails
4. **Error Only** - Always fail with error message

### Common Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Human-readable step name |

### Command Execution Step

Run a shell command.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Step name |
| `command` | string | ✅ | Shell command to execute |
| `message` | string | ❌ | Message to display before executing |
| `error` | string | ❌ | Custom error message if command fails |
| `retry` | enum | ❌ | Retry behavior: `"until"` (retry until success or timeout) |
| `timeout` | string or object | ❌ | Simple: duration string (e.g., `"30s"`). Advanced: object with `interval` and `error_code` |
| `sleep` | string | ❌ | Duration to sleep after command execution (e.g., `"1s"`, `"500ms"`) |
| `verbose` | boolean | ❌ | Enable verbose output for this command (default: `false`) |

**Example:**
```json
{
  "name": "Install package",
  "command": "brew install jq",
  "error": "Failed to install jq. Ensure Homebrew is installed."
}
```

**With Retry:**
```json
{
  "name": "Wait for Docker",
  "command": "docker info",
  "retry": "until",
  "timeout": "60s"
}
```

**With Advanced Timeout:**
```json
{
  "name": "Long running process",
  "command": "./build-script.sh",
  "timeout": {
    "interval": "30m",
    "error_code": 124
  },
  "verbose": true
}
```

**With Sleep (Rate Limiting):**
```json
{
  "name": "API call with rate limiting",
  "command": "curl -X POST https://api.example.com/deploy",
  "sleep": "2s"
}
```

**Verbose Debugging:**
```json
{
  "name": "Debug complex command",
  "command": "./setup.sh --config production",
  "verbose": true,
  "timeout": {
    "interval": "5m",
    "error_code": 143
  }
}
```

### Check with Error Step

Check a condition and fail if check fails.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Step name |
| `check` | string | ✅ | Shell command to check condition |
| `error` | string | ✅ | Error message if check fails |

**Example:**
```json
{
  "name": "Require root",
  "check": "test $(id -u) -eq 0",
  "error": "Error: This installation requires root privileges. Run with sudo."
}
```

### Check with Remediation Step

Check a condition and run remediation steps if check fails.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Step name |
| `check` | string | ✅ | Shell command to check condition |
| `on_missing` | array | ✅ | Remediation steps to run if check fails |

**Example:**
```json
{
  "name": "Install jq",
  "check": "command -v jq",
  "on_missing": [
    {
      "name": "Install via Homebrew",
      "command": "brew install jq"
    }
  ]
}
```

### Error Only Step

Always fail with an error message. Useful for unsupported scenarios.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Step name |
| `error` | string | ✅ | Error message to display |

**Example:**
```json
{
  "name": "Unsupported platform",
  "error": "This configuration does not support Windows. Please use WSL2."
}
```

---

## Remediation Steps

Steps that run when a check fails. Simpler than install steps.

### Remediation Step Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Human-readable step name |
| `command` | string | ✅ | Shell command to execute |
| `error` | string | ❌ | Custom error message if command fails |
| `retry` | enum | ❌ | Retry behavior: `"until"` |
| `timeout` | string or object | ❌ | Simple: duration string (e.g., `"30s"`). Advanced: object with `interval` and `error_code` |
| `sleep` | string | ❌ | Duration to sleep after command execution (e.g., `"1s"`) |
| `verbose` | boolean | ❌ | Enable verbose output for this step (default: `false`) |

### Example

```json
{
  "name": "Install Docker",
  "check": "command -v docker",
  "on_missing": [
    {
      "name": "Install Docker Desktop",
      "command": "brew install --cask docker",
      "timeout": {
        "interval": "10m",
        "error_code": 124
      },
      "verbose": true
    },
    {
      "name": "Start Docker",
      "command": "open -a Docker",
      "sleep": "5s"
    },
    {
      "name": "Wait for Docker daemon",
      "command": "docker info",
      "retry": "until",
      "timeout": "60s",
      "verbose": true
    }
  ]
}
```

### Limitations

⚠️ **Remediation steps cannot contain:**
- `check` field (no nested checks)
- `on_missing` field (no nested remediation)

For complex logic, use separate install steps.

---

## Advanced Features

### Verbose Output

Enable detailed execution information for debugging:

```json
{
  "name": "Debug step",
  "command": "./complex-script.sh",
  "verbose": true
}
```

**Verbose output includes:**
- Full command being executed
- Real-time stdout and stderr
- Exit codes and timing information
- Environment variable interpolation details

**Global verbose mode:**
```bash
# Command line flag (future feature)
sink execute config.json --verbose
```

### Timeout Configuration

**Simple timeout (duration string):**
```json
{
  "command": "long-running-process",
  "timeout": "5m"
}
```

**Advanced timeout (with custom error code):**
```json
{
  "command": "critical-process",
  "timeout": {
    "interval": "30m",
    "error_code": 124
  }
}
```

**Timeout object fields:**
- `interval` (string, required): Duration before timeout (e.g., `"30s"`, `"5m"`, `"2h"`)
- `error_code` (integer, optional): Exit code to return on timeout

**Common timeout error codes:**
- `124`: Standard `timeout` command exit code
- `143`: SIGTERM (graceful termination)
- `137`: SIGKILL (force kill)
- `1`: Generic error

### Sleep Intervals

Pause execution after a command or fact gathering:

```json
{
  "facts": {
    "api_status": {
      "command": "curl -s https://api.example.com/health",
      "sleep": "1s"
    }
  },
  "platforms": [{
    "os": "darwin",
    "match": "darwin*",
    "name": "macOS",
    "install_steps": [
      {
        "name": "Rate-limited API call",
        "command": "curl -X POST https://api.example.com/deploy",
        "sleep": "2s"
      },
      {
        "name": "Another API call",
        "command": "curl -X POST https://api.example.com/notify",
        "sleep": "2s"
      }
    ]
  }]
}
```

**Sleep duration format:**
- Nanoseconds: `"100ns"`
- Microseconds: `"100us"` or `"100µs"`
- Milliseconds: `"100ms"`
- Seconds: `"30s"`
- Minutes: `"5m"`
- Hours: `"2h"`

**Use cases:**
- **Rate limiting**: Prevent API throttling
- **System stabilization**: Allow services time to initialize
- **Resource contention**: Space out resource-intensive operations
- **Network delays**: Account for eventual consistency

### Combining Features

All features can be combined:

```json
{
  "name": "Production deployment",
  "command": "./deploy.sh --environment production",
  "verbose": true,
  "timeout": {
    "interval": "30m",
    "error_code": 124
  },
  "sleep": "10s",
  "error": "Production deployment failed. Check logs at /var/log/deploy.log"
}
```

---

## Fallback

Provide error messages for unsupported platforms or distributions.

### Fallback Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `error` | string | ✅ | Error message (supports `{os}` and `{distro}` placeholders) |

### Global Fallback

```json
{
  "version": "1.0.0",
  "platforms": [...],
  "fallback": {
    "error": "Unsupported platform: {os}"
  }
}
```

### Platform Fallback

```json
{
  "os": "linux",
  "distributions": [...],
  "fallback": {
    "error": "Unsupported Linux distribution: {distro}"
  }
}
```

---

## Complete Examples

### Simple Command

```json
{
  "version": "1.0.0",
  "platforms": [{
    "os": "darwin",
    "match": "darwin*",
    "name": "macOS",
    "install_steps": [
      {
        "name": "Say hello",
        "command": "echo 'Hello, World!'"
      }
    ]
  }]
}
```

### With Facts

```json
{
  "version": "1.0.0",
  "facts": {
    "cpu_count": {
      "command": "sysctl -n hw.ncpu",
      "description": "Number of CPU cores",
      "type": "integer"
    }
  },
  "platforms": [{
    "os": "darwin",
    "match": "darwin*",
    "name": "macOS",
    "install_steps": [
      {
        "name": "Show CPU count",
        "command": "echo 'This system has {{facts.cpu_count}} CPUs'"
      }
    ]
  }]
}
```

### Cross-Platform Package Install

```json
{
  "version": "1.0.0",
  "platforms": [
    {
      "os": "darwin",
      "match": "darwin*",
      "name": "macOS",
      "install_steps": [
        {
          "name": "Install jq",
          "check": "command -v jq",
          "on_missing": [
            {
              "name": "Install via Homebrew",
              "command": "brew install jq"
            }
          ]
        }
      ]
    },
    {
      "os": "linux",
      "match": "linux*",
      "name": "Linux",
      "distributions": [
        {
          "ids": ["ubuntu", "debian"],
          "name": "Debian-based",
          "install_steps": [
            {
              "name": "Install jq",
              "check": "command -v jq",
              "on_missing": [
                {
                  "name": "Update apt cache",
                  "command": "sudo apt-get update"
                },
                {
                  "name": "Install jq",
                  "command": "sudo apt-get install -y jq"
                }
              ]
            }
          ]
        },
        {
          "ids": ["fedora", "rhel", "centos"],
          "name": "Red Hat-based",
          "install_steps": [
            {
              "name": "Install jq",
              "check": "command -v jq",
              "on_missing": [
                {
                  "name": "Install via dnf",
                  "command": "sudo dnf install -y jq"
                }
              ]
            }
          ]
        }
      ],
      "fallback": {
        "error": "Unsupported Linux distribution: {distro}. Please install jq manually."
      }
    }
  ]
}
```

### With Resource Sizing

```json
{
  "version": "1.0.0",
  "facts": {
    "cpu_count": {
      "command": "sysctl -n hw.ncpu",
      "type": "integer"
    },
    "total_ram_gb": {
      "command": "sysctl -n hw.memsize | awk '{print int($1/1024/1024/1024)}'",
      "type": "integer"
    }
  },
  "platforms": [{
    "os": "darwin",
    "match": "darwin*",
    "name": "macOS",
    "install_steps": [
      {
        "name": "Start Colima with resource limits",
        "command": "CPUS=$(( {{facts.cpu_count}} / 5 )); RAM=$(( {{facts.total_ram_gb}} / 5 )); colima start --cpu $CPUS --memory $RAM"
      }
    ]
  }]
}
```

---

## Validation

Validate your configuration against the schema:

```bash
sink validate config.json
```

### Getting the Schema

The schema is embedded in the Sink binary. To output it:

```bash
# Output to stdout
sink schema

# Save to file
sink schema > sink.schema.json

# Use in your editor's schema store
mkdir -p ~/.config/sink
sink schema > ~/.config/sink/sink.schema.json
```

### Schema $id

The schema `$id` is:
```
https://raw.githubusercontent.com/radiolabme/sink/main/src/sink.schema.json
```

For version-specific schemas, use git tags:
```
https://raw.githubusercontent.com/radiolabme/sink/v0.1.0/src/sink.schema.json
```

### Using the Schema

In your configuration files, reference the schema:

```json
{
  "$schema": "https://raw.githubusercontent.com/radiolabme/sink/main/src/sink.schema.json",
  "version": "1.0.0",
  "platforms": [...]
}
```

Or use a relative path for local development:

```json
{
  "$schema": "../src/sink.schema.json",
  "version": "1.0.0",
  "platforms": [...]
}
```

---

[← Back: Usage Guide](usage-guide.md) | [Up: Docs](README.md) | [Next: CLI Reference →](cli-reference.md)
