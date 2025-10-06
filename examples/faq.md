# FAQ

Frequently asked questions about Sink.

## General Questions

### What is Sink?

Sink is a declarative shell installation tool. It lets you define system setup steps in JSON configuration files that are idempotent, cross-platform, and reproducible.

### How is Sink different from Ansible/Chef/Puppet?

Sink is intentionally minimal (< 2,000 lines of code) and has **zero dependencies**. It's designed for:
- Personal machine setup
- Development environment bootstrapping
- Simple automation tasks

Use Ansible/Chef/Puppet for:
- Enterprise infrastructure
- Complex orchestration
- Server fleet management

### How is Sink different from Homebrew Bundle?

- **Homebrew Bundle**: Package management only (via Brewfile)
- **Sink**: Arbitrary shell commands + multi-platform support + idempotency patterns

They're complementary! Use Homebrew Bundle for packages, Sink for custom automation.

### What platforms does Sink support?

- ✅ macOS (Intel and Apple Silicon)
- ✅ Linux (all distributions)
- ✅ FreeBSD
- ⚠️ Windows (WSL2 only)

## Configuration Questions

### Can I use environment variables in configurations?

Yes! Use shell expansion:

```json
{
  "command": "echo $HOME"
}
```

Or capture them as facts:

```json
{
  "facts": {
    "home": {"command": "echo $HOME"}
  }
}
```

### Can facts reference other facts?

No. Facts are evaluated independently. Use shell arithmetic instead:

❌ **Won't work:**
```json
{
  "facts": {
    "total": {"command": "echo 100"},
    "half": {"command": "echo $(( {{facts.total}} / 2 ))"}
  }
}
```

✅ **Use this:**
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

### Can I nest check-remediate patterns?

No. Sink only supports one level of check-remediate. Use separate install steps for complex logic:

❌ **Won't work:**
```json
{
  \"check\": \"...\",
  \"on_missing\": [{
    \"check\": \"...\",       // ❌ Not supported
    \"on_missing\": [...]
  }]
}
```

✅ **Use this:**
```json
{
  \"install_steps\": [
    {\"check\": \"...\", \"on_missing\": [...]},
    {\"check\": \"...\", \"on_missing\": [...]}
  ]
}
```

### How do I handle platform-specific commands?

Use the `platforms` array with OS-specific configurations:

```json
{
  \"platforms\": [{
    \"os\": \"darwin\",
    \"install_steps\": [{\"command\": \"brew install jq\"}]
  }, {
    \"os\": \"linux\",
    \"distributions\": [{
      \"ids\": [\"ubuntu\"],
      \"install_steps\": [{\"command\": \"apt-get install jq\"}]
    }]
  }]
}
```

## Execution Questions

### How do I preview changes without running them?

Use `--dry-run`:

```bash
sink execute config.json --dry-run
```

### How do I skip confirmation prompts?

Use `--yes`:

```bash
sink execute config.json --yes
```

### Why did my command fail?

Check these common issues:

1. **Command not in PATH**: Ensure tools are installed
2. **Permission denied**: May need `sudo` or different user
3. **Syntax errors**: Validate configuration first
4. **Platform mismatch**: Check `--platform` override

Debug steps:
```bash
# Validate configuration
sink validate config.json

# Check facts
sink facts config.json

# Dry-run
sink execute config.json --dry-run
```

### Can I run Sink in CI/CD?

Yes! Use `--yes` to skip prompts:

```bash
sink execute config.json --yes
```

**GitHub Actions:**
```yaml
- name: Setup environment
  run: sink execute .github/setup.json --yes
```

**GitLab CI:**
```yaml
setup:
  script:
    - sink execute ci/setup.json --yes
```

## Troubleshooting

### \"Configuration is invalid\" error

Run validation to see specific errors:

```bash
sink validate config.json
```

Common issues:
- Missing required fields (`version`, `platforms`)
- Invalid JSON syntax
- Wrong field types
- Invalid platform identifiers

### \"Unsupported platform\" error

Your OS isn't covered by the configuration. Add a platform entry:

```json
{
  \"platforms\": [{
    \"os\": \"linux\",
    \"match\": \"linux*\",
    \"name\": \"Linux\",
    \"install_steps\": [...]
  }]
}
```

### \"Command not found\" in check

The check command doesn't exist or isn't in PATH. Common fixes:

✅ **Check if command exists:**
```json
{\"check\": \"command -v jq\"}
```

✅ **Check if file exists:**
```json
{\"check\": \"test -f ~/.config/app/config.json\"}
```

### Facts not appearing

Check these:
- Fact name follows pattern: `^[a-z_][a-z0-9_]*$`
- Command succeeds (test it manually)
- Platform-specific facts match your OS

View gathered facts:
```bash
sink facts config.json
```

### Retry timeout too short

Increase timeout:

```json
{
  \"command\": \"docker info\",
  \"retry\": \"until\",
  \"timeout\": \"120s\"  // Was 30s
}
```

## Best Practices Questions

### Should I commit configurations to git?

Yes! Configurations are portable and reproducible. Add them to your repository:

```bash
.github/
  setup.json
.sink/
  dependencies.json
  development.json
```

### How do I organize multiple configurations?

Use separate files for different concerns:

```bash
sink execute 01-dependencies.json
sink execute 02-applications.json
sink execute 03-configuration.json
```

### Should I use Sink for production servers?

Sink is designed for:
- ✅ Development environments
- ✅ Personal machine setup
- ✅ CI/CD bootstrapping

For production:
- ❌ Use Ansible/Chef/Puppet for complexity
- ❌ Use Docker/Kubernetes for containers
- ⚠️ Sink can work for simple deployments

### How do I test configurations?

1. **Validate syntax:** `sink validate config.json`
2. **Check facts:** `sink facts config.json`
3. **Dry-run:** `sink execute config.json --dry-run`
4. **Test on clean system:** Use Docker or VMs

## Advanced Questions

### Can I generate configurations dynamically?

Yes! Sink reads from stdin:

```bash\n# Generate and execute\ncat << 'EOF' | sink execute -\n{\n  \"version\": \"1.0.0\",\n  \"platforms\": [...]\n}\nEOF\n\n# Use jq to transform\njq '.platforms[0].install_steps += [{\"name\": \"Extra\", \"command\": \"...\"}]' base.json | sink execute -\n```\n\n### Can I use Sink with Docker?\n\nYes! Add Sink to your Dockerfile:\n\n```dockerfile\nFROM ubuntu:22.04\nRUN curl -L https://github.com/your-org/sink/releases/latest/download/sink-linux-amd64 -o /usr/local/bin/sink && chmod +x /usr/local/bin/sink\nCOPY setup.json /tmp/\nRUN sink execute /tmp/setup.json --yes\n```\n\n### How do I extend Sink?\n\nSink is intentionally minimal. For complex needs:\n\n1. **Use shell scripts** in commands\n2. **Chain configurations** sequentially\n3. **Combine with other tools** (Make, Just, Task)\n\n### Where is the cache directory?\n\n**Default:** `~/.cache/sink/`\n\n**Custom:**\n```bash\nexport SINK_CACHE_DIR=/tmp/sink\n```\n\n## Contributing\n\n### How do I report bugs?\n\nOpen an issue on GitHub with:\n- Sink version (`sink version`)\n- OS and architecture\n- Configuration file\n- Full error output\n\n### How do I contribute examples?\n\nContributions welcome! See `examples/` directory and CONTRIBUTING.md.\n\n### Can I add new features?\n\nSink is intentionally minimal (< 2,000 LOC). New features must:\n- Fit the core mission (declarative shell installation)\n- Not add dependencies\n- Not break existing configurations\n- Stay under LOC budget\n\nDiscuss in GitHub Discussions first.\n\n---\n\n[← Back: CLI Reference](cli-reference.md) | [Up: Docs](README.md)\n