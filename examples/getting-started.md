# Getting Started

This guide will walk you through creating and running your first Sink configuration in 5 minutes.

## Prerequisites

Make sure you have Sink installed. If not, see the [Installation Guide](installation.md).

```bash
sink version
```

## Your First Configuration

Sink uses JSON configuration files to describe what should be installed on your system. Let's create a simple one.

### Create a Configuration File

Create a file called `hello.json`:

```json
{
  "version": "1.0.0",
  "platforms": [{
    "os": "darwin",
    "name": "macOS",
    "install_steps": [
      {
        "name": "Say hello",
        "command": "echo 'Hello from Sink!'"
      }
    ]
  }, {
    "os": "linux",
    "name": "Linux",
    "install_steps": [
      {
        "name": "Say hello",
        "command": "echo 'Hello from Sink!'"
      }
    ]
  }]
}
```

### Validate the Configuration

Before running anything, let's make sure the configuration is valid:

```bash
sink validate hello.json
```

If everything is correct, you'll see:

```
Configuration is valid
```

### Preview Execution (Dry-Run)

Sink can show you what it *would* do without actually doing it:

```bash
sink execute hello.json --dry-run
```

You'll see output like:

```
=== Sink Installation Plan ===
Platform: macOS (darwin)

Install Steps:
  1. Say hello
     Command: echo 'Hello from Sink!'

Facts: (none)

This is a dry-run. No commands will be executed.
Continue? (y/n)
```

### Execute the Configuration

Now let's actually run it:

```bash
sink execute hello.json
```

Output:

```
=== Sink Installation Plan ===
Platform: macOS (darwin)

Install Steps:
  1. Say hello
     Command: echo 'Hello from Sink!'

Continue? (y/n) y

[1/1] Running: Say hello
Hello from Sink!

✓ Installation complete
```

Congratulations! You've just run your first Sink configuration.

## Understanding the Configuration

Let's break down what we just created:

```json
{
  "version": "1.0.0",           // Schema version (required)
  "platforms": [{               // Platform-specific configurations
    "os": "darwin",             // macOS identifier
    "name": "macOS",            // Human-readable name
    "install_steps": [          // Commands to run
      {
        "name": "Say hello",    // Step description
        "command": "echo '...'" // Shell command to execute
      }
    ]
  }]
}
```

**Key concepts:**
- **version**: Specifies the Sink schema version (currently `1.0.0`)
- **platforms**: Array of platform-specific configurations
- **os**: Platform identifier (`darwin`, `linux`, `windows`)
- **install_steps**: Sequential commands to execute

## Adding Idempotency

The "Hello World" example always runs. In real-world scenarios, you want commands to run *only if needed*. This is called **idempotency**.

Create `install-jq.json`:

```json
{
  "version": "1.0.0",
  "platforms": [{
    "os": "darwin",
    "name": "macOS",
    "install_steps": [
      {
        "name": "Install jq",
        "check": "command -v jq",
        "on_missing": [
          {
            "command": "brew install jq"
          }
        ]
      }
    ]
  }]
}
```

This uses the **check-remediate pattern**:
- **check**: Command that returns 0 if jq is installed
- **on_missing**: Commands to run if check fails

Now run it:

```bash
sink execute install-jq.json
```

**First run** (jq not installed):
```
[1/1] Running: Install jq
  → Check: command -v jq (NOT FOUND - will remediate)
  → Installing...
==> Installing jq...
✓ jq installed
```

**Second run** (jq already installed):
```
[1/1] Running: Install jq
  → Check: command -v jq (FOUND - skipping)
✓ Installation complete (nothing to do)
```

## Using Facts (Dynamic Values)

Facts let you query system state and use those values in commands.

Create `system-info.json`:

```json
{
  "version": "1.0.0",
  "facts": {
    "cpu_count": {
      "command": "sysctl -n hw.ncpu",
      "description": "Number of CPU cores"
    },
    "hostname": {
      "command": "hostname",
      "description": "System hostname"
    }
  },
  "platforms": [{
    "os": "darwin",
    "name": "macOS",
    "install_steps": [
      {
        "name": "Display system info",
        "command": "echo 'Host: {{facts.hostname}} | CPUs: {{facts.cpu_count}}'"
      }
    ]
  }]
}
```

View the facts:

```bash
sink facts system-info.json
```

Output:

```
Facts:
  cpu_count: 8
  hostname: macbook-pro.local
```

Execute it:

```bash
sink execute system-info.json
```

Output:

```
[1/1] Running: Display system info
Host: macbook-pro.local | CPUs: 8
```

## Real-World Example: Installing Homebrew

Let's combine what we've learned to install Homebrew if it's missing:

Create `install-brew.json`:

```json
{
  "version": "1.0.0",
  "platforms": [{
    "os": "darwin",
    "name": "macOS",
    "install_steps": [
      {
        "name": "Install Homebrew",
        "check": "command -v brew",
        "on_missing": [
          {
            "name": "Download and install Homebrew",
            "command": "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
          },
          {
            "name": "Add Homebrew to PATH",
            "command": "echo 'eval \"$(/opt/homebrew/bin/brew shellenv)\"' >> ~/.zprofile"
          }
        ]
      },
      {
        "name": "Verify Homebrew",
        "command": "brew --version"
      }
    ]
  }]
}
```

Run it:

```bash
sink execute install-brew.json --dry-run  # Preview first
sink execute install-brew.json            # Actually install
```

## Summary

You've learned:
- ✅ How to create a Sink configuration
- ✅ Validate configurations before running
- ✅ Use dry-run to preview changes
- ✅ Check-remediate pattern for idempotency
- ✅ Facts for dynamic system queries
- ✅ Multi-step installations

## Next Steps

- **[Usage Guide](usage-guide.md)** - Learn all Sink patterns
- **[Configuration Reference](configuration-reference.md)** - Complete schema documentation
- **[Examples](.)** - Browse real-world configurations

## Quick Reference

| Command | Purpose |
|---------|---------|
| `sink execute config.json` | Run a configuration |
| `sink execute config.json --dry-run` | Preview without executing |
| `sink validate config.json` | Check configuration syntax |
| `sink facts config.json` | Display gathered facts |
| `sink version` | Show Sink version |
| `sink help` | Display help |

---

[← Back: Installation](installation.md) | [Up: Docs](README.md) | [Next: Usage Guide →](usage-guide.md)
