# Best Practices

Idiomatic patterns and style guide for writing Sink configurations.

## Table of Contents

- [Configuration Style](#configuration-style)
- [Naming Conventions](#naming-conventions)
- [Idempotency](#idempotency)
- [Error Handling](#error-handling)
- [Platform Support](#platform-support)
- [Facts Usage](#facts-usage)
- [Security](#security)
- [Testing](#testing)
- [Organization](#organization)

---

## Configuration Style

### ✅ DO: Use Descriptive Names

```json
{
  \"name\": \"Install Homebrew package manager\",
  \"command\": \"brew install jq\"
}
```

### ❌ DON'T: Use Vague Names

```json
{
  \"name\": \"Install\",
  \"command\": \"brew install jq\"
}
```

### ✅ DO: Add Descriptions to Facts

```json
{
  \"facts\": {
    \"cpu_count\": {
      \"command\": \"sysctl -n hw.ncpu\",
      \"description\": \"Number of CPU cores available\"
    }
  }
}
```

### ✅ DO: Include Schema Reference

```json
{
  "$schema": "https://raw.githubusercontent.com/radiolabme/sink/main/src/sink.schema.json",
  "version": "1.0.0"
}
```

This enables autocompletion and validation in editors.

### ✅ DO: Add Configuration Description

```json
{
  \"description\": \"Install development tools and configure shell environment\",
  \"version\": \"1.0.0\"
}
```

---

## Naming Conventions

### Configuration Files

Use descriptive kebab-case names:

✅ **Good:**
- `platform-dependencies.json`
- `development-environment.json`
- `docker-runtime-setup.json`

❌ **Bad:**
- `config.json`
- `setup.json`
- `install.json`

### Facts

Use snake_case (lowercase with underscores):

✅ **Good:**
- `cpu_count`
- `total_ram_gb`
- `hostname`
- `has_docker`

❌ **Bad:**
- `CPUCount` (camelCase)
- `cpu-count` (kebab-case)
- `1cpu` (starts with number)

### Step Names

Use action-oriented descriptions:

✅ **Good:**
- `\"Install jq JSON processor\"`
- `\"Download Docker Desktop installer\"`
- `\"Wait for PostgreSQL to become ready\"`

❌ **Bad:**
- `\"jq\"`
- `\"Step 1\"`
- `\"Docker\"`

---

## Idempotency

### ✅ DO: Use Check-Remediate Pattern

Always check before acting:

```json
{
  \"name\": \"Install jq\",
  \"check\": \"command -v jq\",
  \"on_missing\": [
    {\"command\": \"brew install jq\"}
  ]
}
```

### ❌ DON'T: Run Commands Unconditionally

```json
{
  \"name\": \"Install jq\",
  \"command\": \"brew install jq\"  // Runs every time!
}
```

### ✅ DO: Check File Existence

```json
{
  \"name\": \"Create config file\",
  \"check\": \"test -f ~/.config/app/config.json\",
  \"on_missing\": [
    {\"command\": \"mkdir -p ~/.config/app\"},
    {\"command\": \"echo '{}' > ~/.config/app/config.json\"}
  ]
}
```

### ✅ DO: Check Service State

```json
{
  \"name\": \"Ensure Docker is running\",
  \"check\": \"docker info 2>/dev/null\",
  \"on_missing\": [
    {\"command\": \"open -a Docker\"},
    {\"command\": \"docker info\", \"retry\": \"until\", \"timeout\": \"60s\"}
  ]
}
```

---

## Error Handling

### ✅ DO: Provide Helpful Error Messages

```json
{
  \"name\": \"Require Homebrew\",
  \"check\": \"command -v brew\",
  \"error\": \"Error: Homebrew is required but not installed. Install from https://brew.sh\"
}
```

### ❌ DON'T: Use Generic Errors

```json
{
  \"name\": \"Check Homebrew\",
  \"check\": \"command -v brew\",
  \"error\": \"Error\"
}
```

### ✅ DO: Check Preconditions Early

```json
{
  \"install_steps\": [
    {
      \"name\": \"Require root privileges\",
      \"check\": \"test $(id -u) -eq 0\",
      \"error\": \"Error: This installation requires root. Run with sudo.\"
    },
    {
      \"name\": \"Install system package\",
      \"command\": \"apt-get install -y package\"
    }
  ]
}
```

### ✅ DO: Add Context to Errors

```json
{
  \"name\": \"Download application\",
  \"command\": \"curl -fsSL https://example.com/app -o ~/bin/app\",
  \"error\": \"Failed to download application. Check your internet connection and try again.\"
}
```

---

## Platform Support

### ✅ DO: Support Multiple Platforms

```json
{
  \"platforms\": [
    {
      \"os\": \"darwin\",
      \"match\": \"darwin*\",
      \"name\": \"macOS\",
      \"install_steps\": [...]
    },
    {
      \"os\": \"linux\",
      \"match\": \"linux*\",
      \"name\": \"Linux\",
      \"distributions\": [...]
    }
  ]
}
```

### ✅ DO: Use Distribution Detection

```json
{
  \"os\": \"linux\",
  \"distributions\": [
    {
      \"ids\": [\"ubuntu\", \"debian\"],
      \"name\": \"Debian-based\",
      \"install_steps\": [...]
    },
    {
      \"ids\": [\"fedora\", \"rhel\", \"centos\"],
      \"name\": \"Red Hat-based\",
      \"install_steps\": [...]
    }
  ]
}
```

### ✅ DO: Provide Fallback Messages

```json
{
  \"os\": \"linux\",
  \"distributions\": [...],
  \"fallback\": {
    \"error\": \"Unsupported Linux distribution: {distro}. Please install manually.\"
  }
}
```

### ❌ DON'T: Hardcode Platform Assumptions

```json
{
  \"command\": \"brew install jq\"  // Only works on macOS/Linux with Homebrew
}
```

---

## Facts Usage

### ✅ DO: Use Facts for Dynamic Values

```json
{
  \"facts\": {
    \"cpu_count\": {\"command\": \"sysctl -n hw.ncpu\"}
  },
  \"install_steps\": [{
    \"command\": \"colima start --cpu {{facts.cpu_count}}\"
  }]
}
```

### ✅ DO: Add Type Annotations

```json
{
  \"facts\": {
    \"cpu_count\": {
      \"command\": \"sysctl -n hw.ncpu\",
      \"type\": \"integer\",
      \"description\": \"Number of CPU cores\"
    }
  }
}
```

### ✅ DO: Use Platform-Specific Facts

```json
{
  \"platforms\": [{
    \"os\": \"darwin\",
    \"facts\": {
      \"cpus\": {\"command\": \"sysctl -n hw.ncpu\"}
    }
  }, {
    \"os\": \"linux\",
    \"facts\": {
      \"cpus\": {\"command\": \"nproc\"}
    }
  }]
}
```

### ❌ DON'T: Try to Reference Facts in Facts

```json
{
  \"facts\": {
    \"total\": {\"command\": \"echo 100\"},
    \"half\": {\"command\": \"echo $(( {{facts.total}} / 2 ))\"}  // ❌ Won't work
  }
}
```

---

## Security

### ✅ DO: Use HTTPS for Downloads

```json
{
  \"command\": \"curl -fsSL https://example.com/installer.sh | sh\"
}
```

### ❌ DON'T: Use Insecure HTTP

```json
{
  \"command\": \"curl -fsSL http://example.com/installer.sh | sh\"
}
```

### ✅ DO: Verify Checksums

```json
{
  \"install_steps\": [
    {\"command\": \"curl -LO https://example.com/app\"},
    {\"command\": \"echo 'abc123 app' | sha256sum -c\"},
    {\"command\": \"mv app /usr/local/bin/\"}
  ]
}
```

### ✅ DO: Avoid Storing Secrets

❌ **Don't do this:**
```json
{
  \"command\": \"export API_KEY=secret123 && deploy.sh\"
}
```

✅ **Do this:**
```json
{
  \"facts\": {
    \"api_key\": {\"command\": \"echo $API_KEY\"}
  },
  \"install_steps\": [{
    \"command\": \"deploy.sh --key {{facts.api_key}}\"
  }]
}
```

Then set environment variable:
```bash\nAPI_KEY=secret123 sink execute config.json\n```\n\n### ✅ DO: Check Permissions\n\n```json\n{\n  \"name\": \"Create sensitive file\",\n  \"check\": \"test -f ~/.ssh/config\",\n  \"on_missing\": [\n    {\"command\": \"touch ~/.ssh/config\"},\n    {\"command\": \"chmod 600 ~/.ssh/config\"}\n  ]\n}\n```\n\n---\n\n## Testing\n\n### ✅ DO: Always Validate First\n\n```bash\nsink validate config.json\n```\n\n### ✅ DO: Use Dry-Run\n\n```bash\nsink execute config.json --dry-run\n```\n\n### ✅ DO: Test on Clean Systems\n\nUse Docker for testing:\n\n```dockerfile\nFROM ubuntu:22.04\nCOPY sink /usr/local/bin/\nCOPY config.json /tmp/\nRUN sink execute /tmp/config.json --yes\n```\n\n### ✅ DO: Test All Platforms\n\n```bash\n# Test macOS\nsink execute config.json --platform darwin\n\n# Test Linux\ndocker run -v $(pwd):/work ubuntu bash -c \"cd /work && sink execute config.json --yes\"\n```\n\n### ✅ DO: Verify Facts\n\n```bash\nsink facts config.json\n```\n\n---\n\n## Organization\n\n### ✅ DO: Separate Concerns\n\nUse multiple configuration files:\n\n```bash\nconfigs/\n  01-system-dependencies.json\n  02-development-tools.json\n  03-application-setup.json\n  04-configuration.json\n```\n\n### ✅ DO: Use Logical Naming\n\nPrefix with numbers for execution order:\n\n```bash\n01-dependencies.json\n02-applications.json\n03-configuration.json\n```\n\n### ✅ DO: Document Complex Logic\n\nAdd comments in surrounding documentation:\n\n```markdown\n# Configuration Notes\n\n## Resource Sizing\n\nThe `colima-docker-runtime.json` configuration calculates\nresource allocation as 20% of system resources using shell\narithmetic since facts cannot reference other facts.\n```\n\n### ✅ DO: Keep Configurations in Version Control\n\n```bash\ngit add configs/\ngit commit -m \"Add development environment setup\"\n```\n\n---\n\n## Common Patterns\n\n### Bootstrap Pattern\n\nInstall dependencies before main application:\n\n```json\n{\n  \"install_steps\": [\n    {\"name\": \"Install Homebrew\", \"check\": \"command -v brew\", \"on_missing\": [...]},\n    {\"name\": \"Install Git\", \"check\": \"command -v git\", \"on_missing\": [{\"command\": \"brew install git\"}]},\n    {\"name\": \"Install application\", \"command\": \"...\"}\n  ]\n}\n```\n\n### Service Readiness Pattern\n\nStart service and wait for ready:\n\n```json\n{\n  \"install_steps\": [\n    {\"name\": \"Start PostgreSQL\", \"command\": \"brew services start postgresql\"},\n    {\"name\": \"Wait for ready\", \"command\": \"pg_isready\", \"retry\": \"until\", \"timeout\": \"30s\"}\n  ]\n}\n```\n\n### Backup Pattern\n\nBackup before making changes:\n\n```json\n{\n  \"install_steps\": [\n    {\n      \"name\": \"Backup existing config\",\n      \"check\": \"test ! -f ~/.config/app/config.json\",\n      \"on_missing\": [{\"command\": \"cp ~/.config/app/config.json ~/.config/app/config.json.bak\"}]\n    },\n    {\"name\": \"Install new config\", \"command\": \"...\"}\n  ]\n}\n```\n\n---\n\n## Anti-Patterns\n\n### ❌ DON'T: Assume State\n\n```json\n{\"command\": \"rm /tmp/app\"}  // What if it doesn't exist?\n```\n\n✅ **Do this:**\n```json\n{\"command\": \"rm -f /tmp/app\"}  // -f ignores missing files\n```\n\n### ❌ DON'T: Hardcode Paths\n\n```json\n{\"command\": \"mkdir /Users/brian/.config\"}  // Won't work for other users!\n```\n\n✅ **Do this:**\n```json\n{\"command\": \"mkdir -p $HOME/.config\"}\n```\n\n### ❌ DON'T: Ignore Errors Silently\n\n```json\n{\"command\": \"some-command || true\"}  // Hides real errors\n```\n\n✅ **Do this:**\n```json\n{\n  \"command\": \"some-command\",\n  \"error\": \"Failed to run some-command. Check that X is installed.\"\n}\n```\n\n### ❌ DON'T: Use Complex Shell Logic\n\n```json\n{\n  \"command\": \"if [ -f file ]; then cmd1; else cmd2; fi\"  // Hard to maintain\n}\n```\n\n✅ **Do this:**\n```json\n{\n  \"install_steps\": [\n    {\"check\": \"test -f file\", \"on_missing\": [{\"command\": \"cmd2\"}]},\n    {\"check\": \"test ! -f file\", \"on_missing\": [{\"command\": \"cmd1\"}]}\n  ]\n}\n```\n\n---\n\n## Summary Checklist\n\nBefore committing a configuration:\n\n- [ ] Added `$schema` reference\n- [ ] Used descriptive names for all steps\n- [ ] Added descriptions to facts\n- [ ] Used check-remediate for idempotency\n- [ ] Provided helpful error messages\n- [ ] Tested with `--dry-run`\n- [ ] Validated with `sink validate`\n- [ ] Supported multiple platforms (if applicable)\n- [ ] No hardcoded paths or secrets\n- [ ] Tested on clean system\n- [ ] Documented complex logic\n\n---\n\n[← Back: FAQ](faq.md) | [Up: Docs](README.md)\n