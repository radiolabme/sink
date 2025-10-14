# Sink Examples

This directory contains production-ready examples demonstrating Sink patterns for real-world system setup tasks. Each example configuration builds on core idempotency principles to provide safe, repeatable installations across different platforms and environments.

## Fundamental Concepts

Before exploring the example configurations, understanding three core mechanisms explains how Sink creates idempotent, adaptive installations. These concepts appear throughout all examples and form the foundation for reliable system configuration.

### Check-Remediate Pattern

The check-remediate pattern forms the basis of idempotent operations. Every installation step first checks whether the desired state already exists. If the check succeeds (exits with code 0), the step is considered complete and remediation is skipped. If the check fails (exits non-zero), remediation commands execute to establish the desired state:

```json
{
  "name": "Install jq",
  "check": "command -v jq",
  "on_missing": [
    {"command": "brew install jq"}
  ]
}
```

In this example, `command -v jq` checks if the jq command exists. When jq is already installed, the check succeeds and Sink skips the installation command. When jq is missing, the check fails and Sink executes `brew install jq` to remedy the situation. Running this configuration multiple times is safe because the check prevents redundant installations.

The `on_missing` field contains an array of commands to execute when the check fails. Multiple commands execute in sequence, each with its own output and error handling. This allows complex remediation workflows broken into clear steps.

### Facts System

Facts gather system information before configuration execution begins. This enables dynamic configurations that adapt to their environment without hardcoded values. Facts can execute commands, read environment variables, or reference other facts:

```json
{
  "facts": {
    "cpu_count": {
      "command": "sysctl -n hw.ncpu",
      "description": "Number of CPU cores"
    },
    "ram_gb": {
      "command": "sysctl -n hw.memsize | awk '{print int($1/1024/1024/1024)}'",
      "description": "Total system RAM in GB"
    },
    "half_cpus": {
      "command": "echo $(( {{facts.cpu_count}} / 2 ))",
      "description": "Half of available CPUs"
    }
  },
  "install_steps": [
    {
      "command": "echo 'System has {{facts.cpu_count}} CPUs, {{facts.ram_gb}}GB RAM. Allocating {{facts.half_cpus}} CPUs.'"
    }
  ]
}
```

Facts execute once at the beginning, before any installation steps run. Their values remain constant throughout execution, ensuring consistency. Facts can reference each other using template syntax `{{facts.name}}`, enabling computed values that build on gathered information. In the example above, `half_cpus` calculates a value based on `cpu_count`, and all three facts are available for use in installation commands.

Facts also support default values when environment variables might not be set:

```json
{
  "facts": {
    "package_name": {
      "command": "echo ${PACKAGE_NAME:-jq}",
      "description": "Package to install, from env or default"
    }
  }
}
```

This pattern allows external parameterization while providing sensible defaults.

### Retry Mechanism

Services often require initialization time after starting. The retry mechanism polls a command repeatedly until it succeeds or a timeout expires. This prevents race conditions where subsequent steps attempt to use services before they're ready:

```json
{
  "name": "Start Docker and wait for ready",
  "command": "brew services start docker",
  "on_success": [
    {
      "name": "Wait for Docker daemon",
      "command": "docker info",
      "retry": "until",
      "timeout": "60s"
    }
  ]
}
```

The `retry: "until"` setting tells Sink to execute the command repeatedly until it succeeds. The command runs every second by default. When `docker info` succeeds (meaning Docker is ready), the step completes immediately. If the timeout of 60 seconds is reached before success, the step fails with a timeout error.

The `on_success` field contains commands that execute only if the parent step succeeds. This creates sequential dependency chains where later steps depend on earlier ones completing successfully. In this example, checking Docker readiness only makes sense if starting the service succeeded.

Retry also supports `retry: "times"` for fixed retry counts:

```json
{
  "command": "flaky-operation",
  "retry": "times",
  "retry_count": 3,
  "timeout": "30s"
}
```

This attempts the command up to 3 times, failing if all attempts fail within the 30-second timeout.

## Getting Started

New users should explore the numbered examples in the `examples/` directory, which demonstrate core Sink patterns progressively:

1. **[01-basic.json](../examples/01-basic.json)** - Simple validation and check-only steps
2. **[02-multi-platform.json](../examples/02-multi-platform.json)** - Cross-platform support
3. **[03-distributions.json](../examples/03-distributions.json)** - Linux distribution detection
4. **[04-facts.json](../examples/04-facts.json)** - System fact gathering
5. **[05-nested-steps.json](../examples/05-nested-steps.json)** - Conditional execution patterns
6. **[06-retry.json](../examples/06-retry.json)** - Retry logic for transient failures
7. **[07-defaults.json](../examples/07-defaults.json)** - Reusable configuration values
8. **[08-error-handling.json](../examples/08-error-handling.json)** - Error handling patterns

Each example is self-contained and demonstrates one specific Sink feature clearly. For comprehensive explanations, use cases, best practices, and a complete FAQ, see **[examples/FAQ.md](../examples/FAQ.md)**.

Quick validation of all examples:

```bash
for f in examples/0*.json; do ./bin/sink validate "$f"; done
```

## Example Configurations

### platform-dependencies.json

This configuration installs cross-platform packages using platform-specific package managers. The primary challenge is handling different operating systems and distributions, each requiring different commands to install the same software. Without proper detection and conditional logic, configurations would need separate files for each platform or contain fragile conditional scripts.

The configuration solves this through platform detection with distribution-specific command selection. It defines installation steps for macOS using Homebrew and multiple Linux distributions using their native package managers (apt, dnf, pacman). The structure separates concerns by platform:

```json
{
  "version": "1.0.0",
  "platforms": [
    {
      "os": "darwin",
      "name": "macOS",
      "match": ".*",
      "install_steps": [
        {
          "name": "Ensure Homebrew installed",
          "check": "command -v brew",
          "on_missing": [
            {"command": "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""}
          ]
        },
        {
          "name": "Install jq",
          "check": "command -v jq",
          "on_missing": [{"command": "brew install jq"}]
        }
      ]
    },
    {
      "os": "linux",
      "name": "Linux",
      "match": ".*",
      "distributions": [
        {
          "ids": ["ubuntu", "debian"],
          "name": "Debian-based",
          "install_steps": [
            {
              "name": "Install jq",
              "check": "command -v jq",
              "on_missing": [
                {"command": "sudo apt-get update"},
                {"command": "sudo apt-get install -y jq"}
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

Sink automatically detects the current operating system and, for Linux, identifies the distribution by reading `/etc/os-release`. The `match` field uses regex to filter which OS versions the platform applies to, with `.*` matching all versions. The `distributions` array within Linux platforms allows further refinement based on distribution identifiers.

The check-remediate pattern appears in two layers here. First, the macOS block ensures Homebrew exists before using it. Then each package installation checks whether the tool is already present. This two-tier approach bootstraps the package manager if needed, then uses it idempotently to install packages.

As a result, a single configuration file works reliably across macOS, Ubuntu, Fedora, and Arch Linux. The idempotent checks prevent duplicate installations, and the automatic package manager setup means the configuration works even on minimal systems. This pattern forms the foundation for portable system configuration.

### lima-setup.json

Installing Lima VM manager requires ensuring platform dependencies are satisfied first. The complication arises because Lima depends on packages like qemu and socket_vmnet that may not be present, and attempting installation without them leads to cryptic errors. Traditional shell scripts either skip dependency checking or implement complex prerequisite verification logic.

This configuration chains dependencies by referencing platform-dependencies.json for base package installation before proceeding with Lima-specific steps. Dependency chaining works by executing one configuration from within another:

```json
{
  "version": "1.0.0",
  "platforms": [
    {
      "os": "darwin",
      "name": "macOS",
      "match": ".*",
      "install_steps": [
        {
          "name": "Ensure platform dependencies",
          "command": "sink execute examples/platform-dependencies.json",
          "description": "Install base tools required by Lima"
        },
        {
          "name": "Install Lima",
          "check": "command -v lima",
          "on_missing": [
            {"command": "brew install lima"}
          ]
        },
        {
          "name": "Verify Lima installation",
          "command": "lima --version"
        }
      ]
    }
  ]
}
```

The first step invokes Sink recursively to execute platform-dependencies.json. Since that configuration uses check-remediate patterns throughout, it completes almost instantly if dependencies are already installed, or takes time to install them if missing. Either way, by the time the second step runs, all dependencies are guaranteed to be present.

Each step includes checks that verify prerequisites are met, only executing installation commands when needed. The platform-specific blocks handle differences between macOS (using Homebrew) and Linux (using distribution package managers). The verification step at the end confirms Lima is properly installed and executable, catching any installation issues before reporting success.

The outcome is reliable Lima installation that gracefully handles missing dependencies. Users can run the configuration repeatedly without issues, and it clearly reports which steps executed versus which were skipped due to existing installation. This dependency chaining pattern extends to any multi-component installation scenario. Configurations can form dependency graphs where complex setups are broken into focused, reusable components.

### colima-setup.json

Colima provides container runtimes for macOS and Linux but requires specific packages and services to be present. The complexity increases because Colima itself must be installed before it can be configured, and configuration attempts fail if the service isn't ready. Simple installation scripts often complete successfully but leave the system in an inconsistent state.

The configuration addresses this through multi-step verification. It first ensures Colima is installed using the check-remediate pattern, then verifies the service starts correctly. Service readiness checks prevent race conditions where subsequent operations attempt to use Colima before it's fully initialized. Platform-specific blocks handle OS differences in service management and configuration file locations.

This produces a complete, working Colima installation that can be verified immediately after the configuration runs. The idempotent nature means running it again after manual changes will restore the desired state. This pattern of install-verify-configure works for any service-based installation.

### colima-docker-runtime.json

Configuring Colima with Docker runtime requires resource allocation decisions that should reflect the host system's capacity. Hardcoding values creates problems: configurations that work well on developer machines may consume too many resources on CI servers, while conservative defaults waste capacity on powerful workstations. Manual configuration for each environment is error-prone and tedious.

The configuration employs the facts system to query CPU count and RAM size, then calculates appropriate resource allocations as a percentage of available capacity. This fact gathering happens before any installation steps execute:

```json
{
  "version": "1.0.0",
  "facts": {
    "cpu_count": {
      "command": "sysctl -n hw.ncpu",
      "description": "Total CPU cores available"
    },
    "total_ram_gb": {
      "command": "sysctl -n hw.memsize | awk '{print int($1/1024/1024/1024)}'",
      "description": "Total system RAM in gigabytes"
    },
    "allocated_cpus": {
      "command": "echo $(( {{facts.cpu_count}} / 2 ))",
      "description": "Allocate 50% of CPUs to Colima"
    },
    "allocated_ram": {
      "command": "echo $(( {{facts.total_ram_gb}} / 2 ))",
      "description": "Allocate 50% of RAM to Colima"
    }
  },
  "platforms": [
    {
      "os": "darwin",
      "name": "macOS",
      "match": ".*",
      "install_steps": [
        {
          "name": "Start Colima with Docker runtime",
          "check": "colima status -p docker >/dev/null 2>&1",
          "on_missing": [
            {
              "command": "colima start -p docker --cpu {{facts.allocated_cpus}} --memory {{facts.allocated_ram}} --disk 60",
              "description": "Start Docker runtime with calculated resources"
            }
          ]
        },
        {
          "name": "Wait for Docker socket",
          "command": "test -S ~/.colima/docker/docker.sock",
          "retry": "until",
          "timeout": "60s"
        }
      ]
    }
  ]
}
```

Notice how `allocated_cpus` and `allocated_ram` reference earlier facts using the `{{facts.name}}` template syntax. Facts are gathered in order, so later facts can depend on earlier ones. The shell arithmetic `$(( expression ))` performs integer division to calculate percentages.

The configuration also references vps-sizes.json to select standardized configurations that fit within the calculated budget. This reference data approach separates resource sizing policy from configuration logic. If the organization wants different allocation strategies, they can modify vps-sizes.json without changing this configuration.

The check-remediate pattern first verifies Colima's Docker runtime is running. The check command `colima status -p docker` returns success if the profile exists and is running. Only when this check fails does Sink execute the start command with the calculated resource allocations. The subsequent wait step uses the retry mechanism to poll for the Docker socket file, ensuring Docker is fully initialized before the configuration completes.

Users receive optimally configured Docker environments that scale with their hardware. The configuration adapts automatically whether deployed on a laptop with 8GB RAM or a server with 128GB. Resource allocation follows industry-standard VPS tiers, ensuring predictable performance characteristics. Re-running the configuration updates resources if the hardware changes.

### colima-incus-runtime.json

Setting up Incus containers with proper networking on macOS presents unique challenges. Incus expects direct host network access, but macOS networking differs significantly from Linux. On Linux, Incus may be available natively, making Colima unnecessary. Configurations must detect these differences and adapt accordingly.

The configuration detects native Incus on Linux and uses it directly, skipping Colima entirely. On macOS, it configures Colima with Incus runtime and establishes network plumbing so containers appear on the host network. Platform-specific blocks handle the different networking requirements, while checks verify each step completes before proceeding. Service readiness checks ensure networking is functional before reporting success.

The result is portable container infrastructure that adapts to the platform. Linux users get native Incus with full performance, while macOS users get properly configured Colima-based Incus with working networking. Both paths produce equivalent functionality from the user's perspective, demonstrating how complex platform adaptations can be hidden behind idempotent configuration.

### vps-sizes.json

This file provides reference data for standard VPS configurations across various hosting providers. Configurations like colima-docker-runtime.json use this data to select appropriate resource allocations that match industry-standard sizing tiers. The data structure maps tier names to CPU, RAM, and disk specifications.

Users can customize this file to match their organization's infrastructure standards or specific cloud provider offerings. The reference data approach separates resource sizing decisions from configuration logic, making both easier to maintain and understand.

## Advanced Usage Patterns

### Chaining Multiple Configurations

Complex environments often require running several configurations in sequence. For example, setting up a complete development environment might involve installing platform dependencies, configuring container runtimes, and then deploying application-specific tooling. Each configuration can focus on a specific concern while depending on previous steps being complete.

The fundamental principle behind chaining is idempotency. Because every step in every configuration checks existing state before acting, running configurations multiple times or in different orders remains safe. This enables both sequential execution scripts and embedded configuration references.

Sequential execution from shell scripts:

```bash
#!/bin/bash
set -e  # Exit on any error

sink execute examples/03-distributions.json
sink execute examples/05-nested-steps.json

echo "Development environment ready"
```

Each configuration performs its checks and only executes necessary operations. If 03-distributions.json already ran successfully, re-running it completes almost instantly because all checks pass. The `set -e` ensures that if any configuration fails, the script stops immediately rather than attempting subsequent steps that would fail due to missing prerequisites.

Alternatively, configurations can embed references to other configurations as installation steps:

```json
{
  "install_steps": [
    {
      "name": "Ensure dependencies",
      "command": "sink execute examples/03-distributions.json",
      "description": "Recursive execution of dependency configuration"
    },
    {
      "name": "Main installation",
      "check": "command -v my-tool",
      "on_missing": [{"command": "brew install my-tool"}]
    }
  ]
}
```

For more examples and patterns, see [examples/FAQ.md](../examples/FAQ.md).

This embedded approach makes dependencies explicit in the configuration itself. Users only need to run the top-level configuration, and it automatically ensures prerequisites are met. This pattern works for arbitrary depth: configurations can reference configurations that reference other configurations, with the idempotent checks preventing redundant work at each level.

### Dynamic Configuration Through Environment Variables

Facts can reference environment variables to customize behavior without modifying configuration files. This enables the same configuration to work in different contexts based on runtime parameters. For example, a package installation configuration could accept the package name as an environment variable.

The pattern uses shell parameter expansion within fact commands to read environment variables with fallback defaults:

```json
{
  "facts": {
    "package_name": {
      "command": "echo ${PACKAGE_NAME:-jq}",
      "description": "Package to install, from PACKAGE_NAME env var or default to jq"
    },
    "package_version": {
      "command": "echo ${PACKAGE_VERSION:-latest}",
      "description": "Package version, from PACKAGE_VERSION env var or latest"
    }
  },
  "install_steps": [
    {
      "name": "Install {{facts.package_name}}",
      "check": "command -v {{facts.package_name}}",
      "on_missing": [
        {"command": "brew install {{facts.package_name}}"}
      ]
    }
  ]
}
```

The syntax `${VAR:-default}` means "use $VAR if set, otherwise use default". This provides graceful fallback behavior when environment variables are absent.

Pass variables through the shell environment:

```bash
PACKAGE_NAME=wget sink execute examples/platform-dependencies.json
```

Or set multiple variables:

```bash
PACKAGE_NAME=postgresql PACKAGE_VERSION=14 sink execute config.json
```

This pattern supports deployment pipelines where configurations need slight variations per environment while maintaining a single source of truth. CI/CD systems can inject environment-specific parameters without maintaining separate configuration files for each environment. The configuration remains generic and testable, with variation introduced through external parameterization.

### Debugging and Validation

Understanding what a configuration will do before execution prevents surprises and catches errors early. Sink provides tools to inspect configurations at different levels of detail. Validation checks syntax and schema compliance, while dry-run mode shows the full execution plan.

Validate configuration syntax:

```bash
sink validate config.json
```

Preview execution plan:

```bash
sink execute config.json --dry-run
```

Inspect fact values:

```bash
sink facts config.json
```

These tools compose to support iterative development. Write a configuration, validate syntax, check fact gathering, review the execution plan in dry-run, then execute for real. Each step provides feedback to catch issues before they affect the system.

## Design Patterns

### Idempotent Installation

Every installation step should check current state before taking action. The check command verifies whether the desired state exists. If the check succeeds (exits 0), the step is skipped because the goal is already achieved. If the check fails (exits non-zero), remediation commands execute to establish the desired state. This pattern enables safe re-execution and makes configurations resilient to partial failures.

The simplest form checks for command existence:

```json
{
  "name": "Ensure tool installed",
  "check": "command -v tool",
  "on_missing": [
    {"command": "brew install tool"}
  ]
}
```

More sophisticated checks verify actual functionality rather than mere presence:

```json
{
  "name": "Ensure Docker functional",
  "check": "docker ps >/dev/null 2>&1",
  "on_missing": [
    {"command": "open -a Docker"},
    {
      "command": "docker ps",
      "retry": "until",
      "timeout": "60s"
    }
  ],
  "description": "Verify Docker daemon is running and responsive"
}
```

This check doesn't just verify the `docker` command exists - it ensures Docker is running and can list containers. If Docker is installed but not started, the check fails and remediation starts the daemon.

For configuration files, verify content matches expectations:

```json
{
  "name": "Ensure correct config",
  "check": "grep -q 'expected_setting=true' ~/.config/app/config",
  "on_missing": [
    {"command": "mkdir -p ~/.config/app"},
    {"command": "echo 'expected_setting=true' > ~/.config/app/config"}
  ]
}
```

The check should test for the actual desired state, not just the existence of files. For example, checking for a configuration file should verify it contains expected content, not merely that it exists. This ensures configurations detect and repair drift from the desired state. If someone modifies the configuration incorrectly, running the configuration again will detect the discrepancy and restore the correct state.

### Multi-Step Remediation

Complex installations often require multiple operations in sequence. Rather than combining commands with shell operators (which can hide failures), express each operation as a separate step. This provides clear feedback about progress and makes partial failures easier to diagnose and resolve.

Break installations into logical steps:

```json
{
  "name": "Ensure service running",
  "check": "service status check",
  "on_missing": [
    {"name": "Install service", "command": "install command"},
    {"name": "Start service", "command": "start command"},
    {
      "name": "Wait for ready",
      "command": "readiness check",
      "retry": "until",
      "timeout": "30s"
    }
  ]
}
```

Each step can include its own checks and error handling. This granularity helps users understand exactly what operations ran and where any failures occurred. The pattern scales from simple two-step installations to complex orchestrations with dozens of operations.

### Resource-Aware Configuration

System configurations should adapt to available resources rather than using hardcoded values. Query system specifications through facts, then use that information to calculate appropriate resource allocations. This ensures configurations work well across different hardware without manual tuning.

Gather system facts and compute allocations:

```json
{
  "facts": {
    "cpu_count": {"command": "nproc"},
    "ram_gb": {"command": "free -g | awk '/^Mem:/{print $2}'"},
    "allocated_cpus": {"command": "echo $(( {{facts.cpu_count}} / 2 ))"},
    "allocated_ram": {"command": "echo $(( {{facts.ram_gb}} / 2 ))"}
  }
}
```

Facts can reference other facts, enabling computed values that build on system information. Allocate percentages rather than absolute amounts so configurations scale from small VMs to large servers. Reference sizing data from files to select standardized configurations that match available resources.

### Service Readiness

Services need time to initialize after starting. Attempting to use a service immediately after launch often fails with connection errors or timeouts. The retry mechanism polls a readiness check until it succeeds or a timeout is reached, ensuring subsequent steps interact with fully initialized services.

The fundamental concept is polling: Sink executes the readiness check command repeatedly, waiting one second between attempts. When the command succeeds (exits 0), the retry completes immediately. If the timeout expires before success, the step fails with a clear error.

Basic service readiness pattern:

```json
{
  "name": "Start and wait for PostgreSQL",
  "command": "brew services start postgresql",
  "on_success": [
    {
      "name": "Wait for ready",
      "command": "pg_isready -q",
      "retry": "until",
      "timeout": "60s",
      "description": "Poll until PostgreSQL accepts connections"
    }
  ]
}
```

The `on_success` field creates a dependency: the readiness check only runs if starting the service succeeded. This prevents polling for readiness when the service didn't start at all, which would waste time waiting for a timeout.

For HTTP services, use health check endpoints:

```json
{
  "name": "Start API server",
  "command": "./start-api-server.sh",
  "on_success": [
    {
      "name": "Wait for API health check",
      "command": "curl -sf http://localhost:8080/health",
      "retry": "until",
      "timeout": "120s",
      "description": "Poll health endpoint until it returns 200 OK"
    }
  ]
}
```

The `-s` flag silences curl's progress output, and `-f` makes it fail (exit non-zero) on HTTP errors. This means curl only succeeds when the server responds with HTTP 200, not when it receives error responses like 404 or 500.

For services without dedicated readiness commands, test functional operations:

```json
{
  "name": "Wait for Redis",
  "command": "redis-cli ping | grep -q PONG",
  "retry": "until",
  "timeout": "30s",
  "description": "Verify Redis responds to PING command"
}
```

Choose readiness checks that verify actual service functionality, not just process existence. Testing `pgrep postgres` only confirms a process is running, not that the database accepts connections. Similarly, checking for a PID file doesn't guarantee the service initialized successfully. Always verify functional operations to ensure services are truly ready for use, not merely running but still initializing.

## Testing Configurations

Configurations should be tested before deployment to catch errors early. Start with syntax validation to ensure the JSON is well-formed and matches the schema. Preview execution in dry-run mode to verify the logic and command sequencing. Test on clean systems to confirm idempotency and completeness.

Validation workflow:

```bash
# Verify syntax
sink validate config.json

# Preview execution plan
sink execute config.json --dry-run

# Check fact gathering
sink facts config.json

# Test on fresh system
sink execute config.json
```

Create test environments that mirror production systems. Virtual machines or containers provide isolated environments for validation. Run configurations twice to verify idempotency: the second run should detect existing installation and skip nearly all operations. This confirms the configuration properly checks state before acting.

## Contributing Examples

New examples should demonstrate specific patterns or solve real-world problems. Each example needs clear documentation explaining the scenario, complications, and how the configuration resolves them. Include comments in the JSON explaining non-obvious logic.

Requirements for new examples:

Create a descriptive JSON filename that indicates the example's purpose. Write documentation following the scenario-complication-resolution structure used throughout this guide. Test the configuration on fresh systems to verify it works as described. Include expected dry-run output showing what the configuration does. Demonstrate idempotency by running the configuration multiple times.

Examples should build on patterns shown in existing configurations rather than introducing entirely new approaches. This maintains consistency and helps users transfer knowledge between examples. Focus on clarity over cleverness: configurations should be obvious even to users new to Sink.
