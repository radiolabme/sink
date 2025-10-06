# Sink Usage Guide

This guide provides comprehensive examples demonstrating Sink patterns and capabilities for real-world system setup tasks. Each example builds on core concepts to show how declarative configuration enables reproducible, idempotent system installations.

## Quick Start

Begin by previewing a configuration in dry-run mode to understand what operations would be performed. This displays the execution plan including which commands would run, in what order, and with what context. The dry-run completes without making any system changes, allowing safe exploration of configuration behavior.

```bash
sink execute examples/platform-dependencies.json --dry-run
```

After reviewing the plan and understanding what will happen, proceed with actual execution. The system displays the execution context including hostname, current user, and working directory, then requests explicit confirmation before making changes:

```bash
sink execute examples/platform-dependencies.json
```

This two-step workflow of preview-then-execute prevents surprises and catches configuration errors before they affect the system.

## Core Concepts

### Install Steps

Every configuration defines platform-specific installation steps that describe the desired system state. The simplest configuration specifies a platform and one or more commands to execute:

```json
{
  "version": "1.0.0",
  "platforms": [{
    "os": "darwin",
    "name": "macOS",
    "match": ".*",
    "install_steps": [
      {"name": "Step 1", "command": "echo hello"}
    ]
  }]
}
```

This structure allows a single configuration to target multiple platforms with platform-appropriate commands. The framework automatically selects the matching platform based on the current operating system.

### Check-Remediate Pattern (Idempotency)

The check-remediate pattern forms the foundation of idempotent configuration. Before executing remediation steps, the system verifies whether the desired state already exists:

```json
{
  "name": "Install tool",
  "check": "command -v tool",
  "on_missing": [
    {"command": "brew install tool"}
  ]
}
```

When this step executes, Sink first runs the check command. If the command succeeds with exit code 0, the tool is already installed and remediation is skipped. If the check fails with a non-zero exit code, the commands in `on_missing` execute to establish the desired state. This pattern enables safe re-execution of configurations without unwanted side effects.

### Facts System (Dynamic Values)

Facts query system state before configuration execution, enabling dynamic and context-aware installations. Facts can retrieve environment variables, execute commands, or read files:

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
    }
  },
  "install_steps": [
    {
      "command": "echo 'System has {{facts.cpu_count}} CPUs and {{facts.total_ram_gb}}GB RAM'"
    }
  ]
}
```

Facts are gathered once at the start of execution and remain constant throughout. They enable patterns like resource-based sizing, conditional logic, and environment-specific configuration. Facts can reference each other, allowing computed values to build on previously gathered information.

### Retry Mechanism (Service Readiness)

Services often require time to initialize after starting. The retry mechanism polls a command until it succeeds or a timeout is reached:

```json
{
  "name": "Wait for Docker",
  "command": "docker info",
  "retry": "until",
  "timeout": "30s"
}
```

The command executes every second. If it succeeds (returns exit code 0), the step completes immediately. If the timeout is reached before success, the step fails. This pattern is essential for services that must be ready before subsequent steps can proceed.

### Platform Detection

Different operating systems and distributions require different commands. Platform detection allows a single configuration to work across diverse environments:

```json
{
  "platforms": [
    {
      "os": "darwin",
      "name": "macOS",
      "match": ".*",
      "install_steps": [{"command": "brew install jq"}]
    },
    {
      "os": "linux",
      "name": "Linux",
      "match": ".*",
      "distributions": [
        {
          "ids": ["ubuntu", "debian"],
          "name": "Debian-based",
          "install_steps": [{"command": "apt-get install jq"}]
        },
        {
          "ids": ["fedora"],
          "name": "Fedora",
          "install_steps": [{"command": "dnf install jq"}]
        }
      ]
    }
  ]
}
```

The framework detects the current operating system and, for Linux, identifies the distribution. It then selects and executes only the relevant installation steps. This enables truly portable configurations that adapt to their environment.

## Pattern Reference

### Simple Package Installation

Installing software requires verifying it isn't already present to avoid redundant operations. Traditional shell scripts either skip this check (wasting time and risking errors) or implement ad-hoc detection logic that varies between scripts. Without standardization, each script handles existence checking differently, making maintenance difficult.

The check-remediate pattern standardizes this approach. Define a check command that succeeds when the desired state exists. If the check passes, remediation is skipped. If it fails, remediation commands execute. This creates idempotent operations that safely re-run:

```json
{
  "name": "Install package",
  "check": "command -v jq",
  "on_missing": [
    {"command": "brew install jq"}
  ]
}
```

As a result, installations complete quickly when software already exists, execute fully when needed, and provide clear feedback about what actions were taken versus skipped.

### Package Manager Bootstrap

Installing packages assumes a package manager exists, but minimal systems may lack one. Attempting package installation without the manager produces cryptic errors. Requiring users to manually install package managers before running configurations adds friction and error-prone manual steps.

This pattern applies the same check-remediate approach to the package manager itself. Check for the manager's existence, and install it if missing. The package manager installation becomes another idempotent step that runs only when needed:

```json
{
  "name": "Ensure Homebrew",
  "check": "command -v brew",
  "on_missing": [
    {
      "command": "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
    }
  ]
}
```

Configurations now work on fresh systems without manual preparation. Users run a single command, and the configuration handles all prerequisites automatically.

### Multi-Step Installation

Complex software requires multiple operations: installation, configuration, and service startup. Combining these into single commands makes failure diagnosis difficult. If any step fails, users must manually determine which operations completed and which need retrying. Traditional scripts either succeed completely or fail opaquely.

Multi-step remediation breaks installations into discrete operations, each with its own verification. Install the software, start the service, then wait for readiness. Each step can include checks and provide feedback about progress:

```json
{
  "name": "Ensure Docker",
  "check": "command -v docker",
  "on_missing": [
    {"name": "Install Docker", "command": "brew install docker"},
    {"name": "Start Docker", "command": "open -a Docker"},
    {
      "name": "Wait for Docker",
      "command": "docker info",
      "retry": "until",
      "timeout": "60s"
    }
  ]
}
```

Users receive detailed feedback about installation progress. Failures clearly indicate which step encountered problems. Partial completions can be resumed without repeating successful steps.

### Conditional Configuration

Applications require configuration files that shouldn't be overwritten if users have customized them. Creating configurations only when absent preserves user modifications while ensuring defaults exist for fresh installations. Traditional approaches either always overwrite (losing customizations) or never create (requiring manual setup).

Check for the configuration file's existence before creating it. Use file system tests as checks, with remediation creating necessary directories and files. This respects existing configurations while providing sensible defaults:

```json
{
  "name": "Configure if missing",
  "check": "test -f ~/.config/app/config.json",
  "on_missing": [
    {"command": "mkdir -p ~/.config/app"},
    {"command": "echo '{}' > ~/.config/app/config.json"}
  ]
}
```

Fresh installations receive working default configurations immediately. Existing installations preserve user customizations. The pattern extends to any file-based configuration scenario.

### Resource-Based Sizing

System configurations need resource allocations that adapt to hardware capacity. Hardcoded values work poorly: they waste resources on powerful systems and overwhelm constrained ones. Manual configuration for each environment is error-prone and doesn't scale across diverse hardware.

Use facts to query system specifications, then calculate allocations as percentages of available resources. This creates configurations that automatically adapt to deployment targets:

```json
{
  "facts": {
    "cpu_count": {"command": "sysctl -n hw.ncpu"},
    "ram_gb": {"command": "sysctl -n hw.memsize | awk '{print int($1/1024/1024/1024)}'"},
    "allocated_cpus": {"command": "echo $(( {{facts.cpu_count}} / 5 ))"},
    "allocated_ram": {"command": "echo $(( {{facts.ram_gb}} / 5 ))"}
  },
  "install_steps": [
    {
      "command": "colima start --cpu {{facts.allocated_cpus}} --memory {{facts.allocated_ram}}"
    }
  ]
}
```

Services receive appropriate resources automatically. The same configuration works on development laptops and production servers. Resource utilization scales predictably with hardware capacity.

### Service Readiness

Services need initialization time after starting before they accept connections. Attempting immediate use produces connection failures and cryptic errors. Traditional scripts either add arbitrary sleep delays (wasting time or being insufficient) or skip readiness checking (causing race conditions).

Use retry with a readiness check command. Poll the service until it responds correctly or a timeout is reached. This waits exactly as long as necessary, no more and no less:

```json
{
  "name": "Start and wait for PostgreSQL",
  "command": "brew services start postgresql",
  "on_success": [
    {
      "name": "Wait for ready",
      "command": "pg_isready",
      "retry": "until",
      "timeout": "30s"
    }
  ]
}
```

Subsequent operations can safely assume the service is ready. Installations complete as quickly as the service allows. Timeout failures clearly indicate service initialization problems rather than configuration errors.

### Precondition Validation

Some operations require specific conditions like root privileges or particular environments. Running without required conditions produces confusing errors deep into execution. Users waste time before discovering the fundamental requirement wasn't met.

Check preconditions explicitly with descriptive error messages. Fail fast with clear guidance about what's required:

```json
{
  "name": "Require root",
  "check": "test $(id -u) -eq 0",
  "error": "Error: This installation requires root privileges. Run with sudo."
}
```

Users receive immediate, actionable feedback about requirement failures. Error messages guide correction rather than leaving users to diagnose problems. Configurations fail cleanly rather than producing partially applied changes.

## Configuration Examples

### platform-dependencies.json

Installing tools across different operating systems requires handling platform-specific package managers and commands. Without proper abstraction, configurations either target single platforms or contain fragile conditional logic that breaks as platforms evolve. Maintaining separate configurations for each platform creates duplication and synchronization problems.

This configuration demonstrates platform detection with distribution-specific command selection. It defines installation steps for macOS using Homebrew and multiple Linux distributions using their native package managers. The check-remediate pattern ensures package managers themselves are installed before attempting package installation. Execute with:

```bash
sink execute examples/platform-dependencies.json --dry-run
```

The result is portable configuration that works on macOS, Ubuntu, Debian, Fedora, and Arch without modification. The automatic package manager setup means it succeeds even on minimal systems. Idempotent checks prevent duplicate installations across repeated runs.

### lima-setup.json

Lima VM manager requires platform dependencies that may not be present on fresh systems. Attempting Lima installation without prerequisites produces cryptic dependency errors. Users must manually research and install requirements before running installation scripts, adding friction to the setup process.

This configuration chains dependencies by ensuring platform-dependencies.json requirements are met before proceeding with Lima-specific installation. Platform-specific blocks handle differences between macOS and Linux package management. Test the configuration with:

```bash
sink execute examples/lima-setup.json
```

Users can run this single command on fresh systems and receive complete Lima installation. Missing dependencies are automatically detected and installed. The idempotent nature allows safe re-execution if problems occur.

### colima-setup.json

Colima installation requires specific packages and proper service initialization. Simple installation scripts often complete successfully but leave services in inconsistent states. Users discover problems only when attempting to use Colima, requiring manual diagnosis and repair.

Multi-step verification ensures both installation and service initialization complete successfully. The configuration installs Colima, starts the service, and verifies readiness before completing. Execute with:

```bash
sink execute examples/colima-setup.json
```

Installations complete in a fully functional state. The configuration provides clear feedback about progress through each step. Failures clearly indicate which operation encountered problems, simplifying diagnosis.

### colima-docker-runtime.json

Configuring Colima with Docker requires resource allocation decisions that should reflect host capacity. Hardcoded values create problems: they waste capacity on powerful systems and overwhelm resource-constrained ones. Manual tuning for each environment doesn't scale across diverse infrastructure.

The facts system queries system specifications and calculates resource allocations as percentages of available capacity. The configuration references vps-sizes.json for standardized sizing that matches allocated resources. Preview with:

```bash
sink execute examples/colima-docker-runtime.json --dry-run
```

Docker environments automatically receive appropriate resource allocations based on host hardware. The same configuration works from developer laptops to high-capacity servers. Resource limits follow industry-standard VPS tiers for predictable performance.

### colima-incus-runtime.json

Incus containers require different setup approaches on macOS versus Linux. macOS needs Colima with special networking configuration, while Linux can use native Incus. Without platform detection, configurations either fail on incompatible platforms or require manual selection, adding complexity.

Platform-specific blocks detect native Incus availability on Linux and use it directly, while macOS gets Colima-based Incus with proper network plumbing. Service readiness checks ensure networking functions before completion. Test with:

```bash
sink execute examples/colima-incus-runtime.json
```

Both platforms receive working Incus container infrastructure optimized for their capabilities. Linux users get native performance, macOS users get properly configured Colima-based containers. Platform differences are transparent to end users.

### vps-sizes.json

Resource sizing decisions benefit from standardized tiers that match industry VPS offerings. Creating ad-hoc sizing in each configuration duplicates logic and makes capacity planning difficult. This reference data provides standard configurations used by multiple setup scripts.

The file defines CPU, RAM, and disk specifications for common VPS tiers. Configurations query system capacity, then select the largest tier that fits within resource budgets. This approach separates sizing policy from configuration logic, making both easier to maintain and customize for organizational needs.

## Advanced Topics

### Configuration Chaining

Complex environments require multiple installation phases with dependencies between them. Running everything in a single massive configuration makes failure diagnosis difficult and prevents reusing common components. Breaking installations into focused configurations improves maintainability but requires coordinating execution order.

Execute related configurations in sequence, relying on idempotency for safety. Each configuration performs its checks and only executes necessary operations. If earlier configurations already ran successfully, they complete almost instantly:

```bash
sink execute examples/platform-dependencies.json
sink execute examples/colima-setup.json
sink execute examples/colima-docker-runtime.json
```

This approach allows mixing and matching components. Install only the configurations needed for specific environments. The idempotent design means interrupted installations can resume safely by re-running from the beginning.

### Environment-Based Configuration

Single configurations need to adapt to different contexts without modification. Hardcoding environment-specific values requires maintaining multiple configuration variants. Branch-based conditional logic becomes complex and error-prone as environments proliferate.

Use environment variables with facts to parameterize configurations. Pass values through the shell environment:

```bash
PACKAGE_NAME=wget sink execute examples/platform-dependencies.json
```

Configure facts to reference environment variables with sensible defaults:

```json
{
  "facts": {
    "package": {"command": "echo ${PACKAGE_NAME:-jq}"}
  }
}
```

Configurations remain generic while supporting environment-specific customization. Development and production can use different values from the same configuration file. Deployment pipelines pass parameters through environment variables without modifying configurations.

### Debugging and Validation

Configurations should be thoroughly validated before execution to catch errors early. Running unvalidated configurations risks system changes from buggy logic. Understanding what will happen prevents surprises and builds confidence in configuration correctness.

Sink provides tools for progressive validation. Start with syntax checking:

```bash
sink validate config.json
```

Preview fact gathering to verify data sources work correctly:

```bash
sink facts config.json
```

Review the full execution plan without making changes:

```bash
sink execute config.json --dry-run
```

This workflow catches errors progressively. Syntax problems appear immediately. Fact gathering issues surface before execution planning. The dry-run reveals logical problems in step sequencing or conditionals. Only after all validation passes should real execution proceed.

### Testing on Fresh Systems

Configurations should work reliably on clean systems without manual preparation. Testing on developer machines with accumulated state doesn't validate the complete installation process. Configurations that work on configured systems may fail on fresh ones due to missing dependencies.

Create isolated test environments using virtual machines or containers. These provide clean slates that mirror production deployment targets. Run configurations twice to verify idempotency: the second run should complete almost instantly because all checks pass.

Testing workflow:

```bash
# First run - full installation
sink execute config.json

# Second run - should skip everything
sink execute config.json
```

The second run validates that checks properly detect existing state. If operations repeat unnecessarily, checks aren't working correctly. This testing approach catches both installation completeness and idempotency problems.

## Common Operational Patterns

### System Dependency Installation

Many applications require build tools, compilers, or system libraries. These dependencies vary by platform and may require different installation approaches. Without proper handling, application installations fail with cryptic errors about missing components.

Use platform-specific check-remediate patterns for system dependencies:

```json
{
  "name": "Install build tools",
  "check": "command -v gcc",
  "on_missing": [
    {"command": "xcode-select --install"}
  ]
}
```

Verify the actual tool exists rather than checking for meta-packages. Different platforms provide build tools through different mechanisms. The check-remediate pattern adapts to platform differences while maintaining a consistent interface.

### Directory Structure Creation

Applications need configuration and data directories in specific locations. Creating these during installation is straightforward, but handling existing directories requires care. Blindly creating directories can override permissions or fail if paths partially exist.

Check for directory existence before creation:

```json
{
  "name": "Create config directory",
  "check": "test -d ~/.config/myapp",
  "on_missing": [
    {"command": "mkdir -p ~/.config/myapp"}
  ]
}
```

The `-p` flag creates parent directories as needed and succeeds if the directory already exists. This combines with the existence check to provide truly idempotent directory creation.

### Remote File Retrieval

Configurations often need to download files from remote sources. Downloads can fail due to network issues, so retry logic improves reliability. Checking for existing files prevents re-downloading unchanged content.

Verify files exist before downloading:

```json
{
  "name": "Download configuration",
  "check": "test -f ~/.config/app/config.json",
  "on_missing": [
    {"command": "curl -fsSL https://example.com/config.json -o ~/.config/app/config.json"}
  ]
}
```

For files that may change, add integrity checking through checksums. Download to temporary locations first, verify integrity, then move to final locations. This prevents partially downloaded or corrupted files from causing application failures.

### Permission Management

Files and directories need appropriate permissions for security and functionality. Setting permissions during installation ensures correct access controls. Checking current permissions before modification prevents unnecessary filesystem operations.

Verify and set permissions:

```json
{
  "name": "Fix permissions",
  "check": "test -x ~/bin/script.sh",
  "on_missing": [
    {"command": "chmod +x ~/bin/script.sh"}
  ]
}
```

The check verifies the specific permission bit rather than blindly setting permissions. This makes operations idempotent while providing clear feedback about what changed.

## Getting Started

New users should begin with platform-dependencies.json to understand the fundamental check-remediate pattern. After mastering basic idempotent installation, progress to lima-setup.json and colima-setup.json for multi-step configurations. Advanced users can explore colima-docker-runtime.json and colima-incus-runtime.json to see facts-based dynamic configuration.

For additional patterns and detailed documentation, consult the examples directory README.md which provides comprehensive scenario-complication-resolution explanations for each example configuration.

## Best Practices

### ✅ DO

- **Use check-remediate pattern** for idempotency
- **Gather facts** for dynamic configuration
- **Use retry** for service readiness
- **Validate configs** before execution (`--dry-run`)
- **Check preconditions** with check+error pattern
- **Document each step** with descriptive names

### ❌ DON'T

- **Don't repeat yourself** - use facts for common values
- **Don't hardcode paths** - use `$HOME`, `$USER`, etc.
- **Don't assume state** - always check before acting
- **Don't skip dry-run** - preview changes first
- **Don't ignore errors** - provide clear error messages

---

## Testing Your Configs

### 1. Validate Syntax
```bash
sink validate your-config.json
```

### 2. Preview Execution
```bash
sink execute your-config.json --dry-run
```

### 3. Check Facts
```bash
sink facts your-config.json
```

### 4. Run on Fresh System
Test on a clean VM or container to verify idempotency.

---

## Common Patterns

### Installing System Dependencies

```json
{
  "name": "Install build tools",
  "check": "command -v gcc",
  "on_missing": [
    {"command": "xcode-select --install"}
  ]
}
```

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

### Downloading Files

```json
{
  "name": "Download configuration",
  "check": "test -f ~/.config/app/config.json",
  "on_missing": [
    {"command": "curl -fsSL https://example.com/config.json -o ~/.config/app/config.json"}
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

## Getting Help

- **Documentation:** See `docs/` directory
- **Schema:** Use `sink schema` to output the JSON schema, or reference `src/sink.schema.json`
- **Issues:** Check existing patterns in these examples
- **Command help:** `sink help`

---

## Contributing Examples

To add a new example:

1. Create a descriptive `.json` file
2. Add entry to this README
3. Document the pattern it demonstrates
4. Include dry-run output example
5. Test on fresh system

---

## Quick Reference Card

| Task | Command |
|------|---------|
| Preview changes | `sink execute config.json --dry-run` |
| Run installation | `sink execute config.json` |
| View facts | `sink facts config.json` |
| Validate syntax | `sink validate config.json` |
| Show version | `sink version` |
| Get help | `sink help` |

---

**Next:** Start with [platform-dependencies.json](platform-dependencies.json) to see the basic pattern.
