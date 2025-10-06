# Usage Guide

Complete guide to all Sink patterns, features, and best practices.

## Table of Contents

- [Configuration Files](#configuration-files)
- [Platform Detection](#platform-detection)
- [Install Steps](#install-steps)
- [Check-Remediate Pattern](#check-remediate-pattern)
- [Facts System](#facts-system)
- [Retry Mechanism](#retry-mechanism)
- [Error Handling](#error-handling)
- [Multi-Step Installations](#multi-step-installations)
- [Platform-Specific Commands](#platform-specific-commands)
- [Environment Variables](#environment-variables)
- [Working with Files](#working-with-files)
- [Service Management](#service-management)
- [Resource Sizing](#resource-sizing)
- [Advanced Patterns](#advanced-patterns)

---

## Configuration Files

### Supported File Formats

Sink uses JSON configuration files with the `.json` extension.

```bash
sink execute config.json
```

### Reading from stdin

You can also pipe configurations to Sink:

```bash
cat config.json | sink execute -
# or
sink execute - < config.json
```

This is useful for dynamically generated configurations.

### Configuration Structure

Every Sink configuration has this basic structure:

```json
{
  "version": "1.0.0",
  "facts": {},
  "platforms": []
}
```

**Required fields:**
- `version`: Schema version (currently `1.0.0`)
- `platforms`: Array of platform-specific configurations

**Optional fields:**
- `facts`: System queries executed before installation

---

## Platform Detection

Sink automatically detects your operating system and selects the appropriate platform configuration.

### OS Identifiers

| OS | Identifier | Examples |
|----|------------|----------|
| macOS | `darwin` | All macOS versions |
| Linux | `linux` | All Linux distributions |
| Windows | `windows` | Windows (WSL2 only) |

### Basic Platform Configuration

```json
{
  "platforms": [
    {
      "os": "darwin",
      "name": "macOS",
      "install_steps": [...]
    },
    {
      "os": "linux",
      "name": "Linux",
      "install_steps": [...]
    }
  ]
}
```

### Linux Distribution Detection

For Linux, you can specify distribution-specific commands:

```json
{
  "os": "linux",
  "name": "Linux",
  "distributions": [
    {
      "ids": ["ubuntu", "debian"],
      "match": "apt",
      "install_steps": [
        {"command": "apt-get install -y jq"}
      ]
    },
    {
      "ids": ["fedora", "rhel", "centos"],
      "match": "dnf",
      "install_steps": [
        {"command": "dnf install -y jq"}
      ]
    },
    {
      "ids": ["alpine"],
      "match": "apk",
      "install_steps": [
        {"command": "apk add jq"}
      ]
    }
  ]
}
```

**Distribution IDs** (from `/etc/os-release`):
- Ubuntu: `ubuntu`
- Debian: `debian`
- Fedora: `fedora`
- RHEL: `rhel`
- CentOS: `centos`
- Alpine: `alpine`
- Arch: `arch`

### Platform Override

Force execution for a specific platform:

```bash
sink execute config.json --platform linux
```

---

## Install Steps

Install steps are the commands that Sink executes. They run sequentially.

### Simple Command

```json
{
  "install_steps": [
    {
      "name": "Create directory",
      "command": "mkdir -p ~/myapp"
    }
  ]
}
```

### Command with Description

The `name` field is optional but recommended for clarity:

```json
{
  "name": "Install dependencies",
  "command": "npm install"
}
```

Without a name, Sink shows the command itself.

### Multiple Steps

Steps execute in order:

```json
{
  "install_steps": [
    {
      "name": "Step 1",
      "command": "echo 'First'"
    },
    {
      "name": "Step 2",
      "command": "echo 'Second'"
    },
    {
      "name": "Step 3",
      "command": "echo 'Third'"
    }
  ]
}
```

---

## Check-Remediate Pattern

The core pattern for idempotency: **check if something exists, install only if missing**.

### Basic Check-Remediate

```json
{
  "name": "Install jq",
  "check": "command -v jq",
  "on_missing": [
    {"command": "brew install jq"}
  ]
}
```

**How it works:**
1. Run `check` command
2. If exit code is 0 → skip (already installed)
3. If exit code is non-0 → run `on_missing` steps

### Common Check Commands

**Check if command exists:**
```json
{"check": "command -v docker"}
```

**Check if file exists:**
```json
{"check": "test -f ~/.config/app/config.json"}
```

**Check if directory exists:**
```json
{"check": "test -d /opt/myapp"}
```

**Check if service is running:**
```json
{"check": "systemctl is-active docker"}
```

**Check if port is listening:**
```json
{"check": "lsof -i :8080"}
```

**Check if package is installed:**
```json
{"check": "brew list jq"}
```

### Multiple Remediation Steps

If the check fails, all `on_missing` steps run in order:

```json
{
  "name": "Setup application",
  "check": "test -f ~/app/binary",
  "on_missing": [
    {"name": "Create directory", "command": "mkdir -p ~/app"},
    {"name": "Download binary", "command": "curl -o ~/app/binary https://example.com/binary"},
    {"name": "Make executable", "command": "chmod +x ~/app/binary"}
  ]
}
```

### Nested Checks (Not Supported)

❌ Sink does NOT support nested check-remediate patterns:

```json
{
  "check": "...",
  "on_missing": [
    {
      "check": "...",           // ❌ NOT SUPPORTED
      "on_missing": [...]       // ❌ NOT SUPPORTED
    }
  ]
}
```

Instead, use separate install steps or shell logic.

### Precondition Checks

Use `error` to fail if a precondition isn't met:

```json
{
  "name": "Require Homebrew",
  "check": "command -v brew",
  "error": "Error: Homebrew is required. Install from https://brew.sh"
}
```

If the check fails, execution stops with the error message.

---

## Facts System

Facts query system state before running install steps. Use them for dynamic configuration.

### Defining Facts

```json
{
  "facts": {
    "cpu_count": {
      "command": "sysctl -n hw.ncpu",
      "description": "Number of CPU cores"
    },
    "total_ram_gb": {
      "command": "sysctl -n hw.memsize | awk '{print int($1/1024/1024/1024)}'",
      "description": "Total RAM in GB"
    },
    "username": {
      "command": "whoami",
      "description": "Current user"
    }
  }
}
```

### Using Facts in Commands

Reference facts with `{{facts.name}}`:

```json
{
  "install_steps": [
    {
      "command": "echo 'CPUs: {{facts.cpu_count}}'"
    },
    {
      "command": "echo 'RAM: {{facts.total_ram_gb}}GB'"
    },
    {
      "command": "echo 'User: {{facts.username}}'"
    }
  ]
}
```

### Viewing Facts

See what facts will be gathered:

```bash
sink facts config.json
```

Output:
```
Facts:
  cpu_count: 8
  total_ram_gb: 16
  username: brian
```

### Cross-Platform Facts

Use platform-specific facts:

```json
{
  "platforms": [{
    "os": "darwin",
    "facts": {
      "cpus": {"command": "sysctl -n hw.ncpu"}
    }
  }, {
    "os": "linux",
    "facts": {
      "cpus": {"command": "nproc"}
    }
  }]
}
```

### Facts Cannot Reference Other Facts

❌ This does NOT work:

```json
{
  "facts": {
    "total": {"command": "echo 100"},
    "half": {"command": "echo $(( {{facts.total}} / 2 ))"}  // ❌ NOT SUPPORTED
  }
}
```

Instead, do arithmetic in shell:

```json
{
  "facts": {
    "total": {"command": "echo 100"}
  },
  "install_steps": [{
    "command": "HALF=$(( {{facts.total}} / 2 )); echo $HALF"
  }]
}
```

---

## Retry Mechanism

Wait for services to become ready.

### Until Pattern

Retry a command until it succeeds (exit 0):

```json
{
  "name": "Wait for Docker",
  "command": "docker info",
  "retry": "until",
  "timeout": "30s"
}
```

**Behavior:**
- Runs `docker info` every 1 second
- Stops when command succeeds (exit 0)
- Fails if timeout (30s) is reached

### Timeout Format

Specify durations with units:
- `30s` - 30 seconds
- `5m` - 5 minutes
- `1h` - 1 hour

### Common Retry Patterns

**Wait for port to open:**
```json
{
  "command": "nc -z localhost 8080",
  "retry": "until",
  "timeout": "60s"
}
```

**Wait for service:**
```json
{
  "command": "systemctl is-active postgresql",
  "retry": "until",
  "timeout": "30s"
}
```

**Wait for file:**
```json
{
  "command": "test -f /var/run/app.pid",
  "retry": "until",
  "timeout": "10s"
}
```

**Wait for HTTP endpoint:**
```json
{
  "command": "curl -f http://localhost:8080/health",
  "retry": "until",
  "timeout": "60s"
}
```

---

## Error Handling

Control how Sink handles command failures.

### Default Behavior

By default, if any command fails (non-zero exit), Sink stops execution.

```json
{
  "install_steps": [
    {"command": "echo 'Step 1'"},
    {"command": "false"},        // Fails here
    {"command": "echo 'Step 3'"} // Never runs
  ]
}
```

### Custom Error Messages

Provide helpful error messages:

```json
{
  "name": "Check root privileges",
  "check": "test $(id -u) -eq 0",
  "error": "Error: This installation requires root privileges. Run with sudo."
}
```

### Ignore Errors (Not Recommended)

⚠️ Sink does not support `ignore_error`. If you need to continue despite errors, use shell logic:

```bash
command || true
```

---

## Multi-Step Installations

### Sequential Steps

Steps always run in order:

```json
{
  "install_steps": [
    {"name": "Download", "command": "curl -O https://example.com/app.tar.gz"},
    {"name": "Extract", "command": "tar -xzf app.tar.gz"},
    {"name": "Install", "command": "sudo cp app /usr/local/bin/"},
    {"name": "Cleanup", "command": "rm app.tar.gz"}
  ]
}
```

### Conditional Steps

Use check-remediate for conditional execution:

```json
{
  "install_steps": [
    {
      "name": "Install if missing",
      "check": "command -v app",
      "on_missing": [
        {"command": "install app"}
      ]
    },
    {
      "name": "Configure if missing",
      "check": "test -f ~/.config/app/config.json",
      "on_missing": [
        {"command": "mkdir -p ~/.config/app"},
        {"command": "app --init-config"}
      ]
    }
  ]
}
```

### Chaining Configurations

Run multiple configurations in sequence:

```bash
sink execute 01-dependencies.json
sink execute 02-application.json
sink execute 03-configuration.json
```

---

## Platform-Specific Commands

### Distribution-Specific Package Managers

```json
{
  "os": "linux",
  "distributions": [
    {
      "ids": ["ubuntu", "debian"],
      "install_steps": [
        {"command": "apt-get update"},
        {"command": "apt-get install -y jq"}
      ]
    },
    {
      "ids": ["fedora", "rhel"],
      "install_steps": [
        {"command": "dnf install -y jq"}
      ]
    },
    {
      "ids": ["alpine"],
      "install_steps": [
        {"command": "apk add jq"}
      ]
    },
    {
      "ids": ["arch"],
      "install_steps": [
        {"command": "pacman -S --noconfirm jq"}
      ]
    }
  ]
}
```

### macOS-Specific Commands

```json
{
  "os": "darwin",
  "install_steps": [
    {"command": "brew install jq"},
    {"command": "defaults write com.apple.dock autohide -bool true"},
    {"command": "killall Dock"}
  ]
}
```

### Linux-Specific Commands

```json
{
  "os": "linux",
  "install_steps": [
    {"command": "systemctl enable docker"},
    {"command": "systemctl start docker"},
    {"command": "usermod -aG docker $USER"}
  ]
}
```

---

## Environment Variables

### Using Environment Variables

Reference environment variables in commands:

```json
{
  "install_steps": [
    {"command": "echo $HOME"},
    {"command": "echo $USER"},
    {"command": "echo $PATH"}
  ]
}
```

### Setting Environment Variables

Set variables for subsequent commands:

```json
{
  "install_steps": [
    {"command": "export APP_ENV=production"},
    {"command": "echo $APP_ENV"}  // Won't work - each command is isolated
  ]
}
```

⚠️ **Note**: Each command runs in a separate shell. To share variables, combine commands:

```json
{
  "command": "export APP_ENV=production && echo $APP_ENV"
}
```

### Facts from Environment

Use facts to capture environment variables:

```json
{
  "facts": {
    "home": {"command": "echo $HOME"},
    "user": {"command": "echo $USER"}
  },
  "install_steps": [
    {"command": "mkdir -p {{facts.home}}/.config/myapp"}
  ]
}
```

---

## Working with Files

### Creating Directories

```json
{
  "name": "Create config directory",
  "check": "test -d ~/.config/myapp",
  "on_missing": [
    {"command": "mkdir -p ~/.config/myapp"}
  ]
}
```

### Creating Files

```json
{
  "name": "Create config file",
  "check": "test -f ~/.config/myapp/config.json",
  "on_missing": [
    {"command": "mkdir -p ~/.config/myapp"},
    {"command": "echo '{}' > ~/.config/myapp/config.json"}
  ]
}
```

### Downloading Files

```json
{
  "name": "Download binary",
  "check": "test -f ~/bin/app",
  "on_missing": [
    {"command": "mkdir -p ~/bin"},
    {"command": "curl -L https://example.com/app -o ~/bin/app"},
    {"command": "chmod +x ~/bin/app"}
  ]
}
```

### Setting Permissions

```json
{
  "name": "Fix permissions",
  "check": "test -x ~/bin/script.sh",
  "on_missing": [
    {"command": "chmod +x ~/bin/script.sh"}
  ]
}
```

---

## Service Management

### Starting Services

```json
{
  "name": "Start Docker",
  "command": "brew services start docker"
}
```

### Waiting for Service Readiness

```json
{
  "install_steps": [
    {"name": "Start PostgreSQL", "command": "brew services start postgresql"},
    {
      "name": "Wait for ready",
      "command": "pg_isready",
      "retry": "until",
      "timeout": "30s"
    }
  ]
}
```

### Service with Retry

```json
{
  "name": "Start and wait for Docker",
  "command": "open -a Docker && until docker info; do sleep 1; done"
}
```

---

## Resource Sizing

### Detecting System Resources

```json
{
  "facts": {
    "cpu_count": {
      "command": "sysctl -n hw.ncpu",
      "description": "CPU cores"
    },
    "total_ram_gb": {
      "command": "sysctl -n hw.memsize | awk '{print int($1/1024/1024/1024)}'",
      "description": "RAM in GB"
    }
  }
}
```

### Using Resource Facts

```json
{
  "install_steps": [
    {
      "command": "colima start --cpu {{facts.cpu_count}} --memory {{facts.total_ram_gb}}"
    }
  ]
}
```

### Calculated Resources

Since facts can't reference other facts, use shell arithmetic:

```json
{
  "install_steps": [
    {
      "command": "CPUS=$(( {{facts.cpu_count}} / 5 )); RAM=$(( {{facts.total_ram_gb}} / 5 )); colima start --cpu $CPUS --memory $RAM"
    }
  ]
}
```

---

## Advanced Patterns

### Homebrew Auto-Install

Install Homebrew if missing, then use it:

```json
{
  "install_steps": [
    {
      "name": "Ensure Homebrew",
      "check": "command -v brew",
      "on_missing": [
        {"command": "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""}
      ]
    },
    {
      "name": "Install package",
      "check": "command -v jq",
      "on_missing": [
        {"command": "brew install jq"}
      ]
    }
  ]
}
```

### Dependency Chains

Install dependencies before main application:

```json
{
  "install_steps": [
    {"name": "Dependency 1", "check": "command -v git", "on_missing": [{"command": "brew install git"}]},
    {"name": "Dependency 2", "check": "command -v make", "on_missing": [{"command": "brew install make"}]},
    {"name": "Main app", "command": "git clone ... && cd ... && make install"}
  ]
}
```

### Configuration Backup

Backup existing configuration before changes:

```json
{
  "install_steps": [
    {
      "name": "Backup existing config",
      "check": "test ! -f ~/.config/app/config.json",
      "on_missing": [
        {"command": "cp ~/.config/app/config.json ~/.config/app/config.json.bak"}
      ]
    },
    {
      "name": "Install new config",
      "command": "curl -o ~/.config/app/config.json https://example.com/config.json"
    }
  ]
}
```

### Post-Installation Verification

```json
{
  "install_steps": [
    {"name": "Install", "command": "brew install docker"},
    {"name": "Start", "command": "open -a Docker"},
    {
      "name": "Wait for ready",
      "command": "docker info",
      "retry": "until",
      "timeout": "60s"
    },
    {
      "name": "Verify",
      "command": "docker run hello-world"
    }
  ]
}
```

---

## Debugging

### Dry-Run Mode

Preview what Sink will do:

```bash
sink execute config.json --dry-run
```

### View Facts

See gathered facts:

```bash
sink facts config.json
```

### Validate Configuration

Check syntax:

```bash
sink validate config.json
```

### Verbose Output

Sink shows all commands and their output by default. To reduce noise, use shell redirections:

```json
{
  "command": "brew install jq 2>&1 | grep -v 'Warning'"
}
```

---

## Best Practices

### ✅ DO

- **Use check-remediate** for idempotency
- **Add descriptive names** to all steps
- **Test with --dry-run** before real execution
- **Validate configurations** before committing
- **Use facts** for system-specific values
- **Document complex logic** with comments (in surrounding docs)

### ❌ DON'T

- **Don't hardcode paths** - use `$HOME`, facts, or shell expansions
- **Don't assume state** - always check before acting
- **Don't skip validation** - catch errors early
- **Don't nest check-remediate** - use separate steps instead
- **Don't ignore errors** - fail fast and fix issues

---

## Next Steps

- **[Configuration Reference](configuration-reference.md)** - Complete schema documentation
- **[CLI Reference](cli-reference.md)** - All commands and flags
- **[Best Practices](best-practices.md)** - Idiomatic patterns
- **[Examples](.)** - Real-world configurations

---

[← Back: Getting Started](getting-started.md) | [Up: Docs](README.md) | [Next: Configuration Reference →](configuration-reference.md)
