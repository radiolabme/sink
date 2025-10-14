# Sink Examples & FAQ# FAQ



A curated collection of working examples demonstrating Sink's capabilities, from basic configurations to advanced patterns.Frequently asked questions about Sink.



## Quick Start## General Questions



**New to Sink?** Start here:### What is Sink?



1. [`01-basic.json`](#01-basicjson) - Your first Sink configurationSink is a declarative shell installation tool. It lets you define system setup steps in JSON configuration files that are idempotent, cross-platform, and reproducible.

2. [`04-facts.json`](#04-factsjson) - Learn fact gathering  

3. [`05-nested-steps.json`](#05-nested-stepsjson) - Conditional execution### How is Sink different from Ansible/Chef/Puppet?

4. [`03-distributions.json`](#03-distributionsjson) - Linux distribution support

Sink is intentionally minimal (< 2,000 lines of code) and has **zero dependencies**. It's designed for:

---- Personal machine setup

- Development environment bootstrapping

## Core Examples- Simple automation tasks



### 01-basic.json - Simplest ConfigurationUse Ansible/Chef/Puppet for:

- Enterprise infrastructure

**Purpose:** The simplest possible Sink configuration- Complex orchestration

- Server fleet management

**What it demonstrates:**

- Single-platform setup (macOS only)### How is Sink different from Homebrew Bundle?

- Check-only steps (validation without execution)

- Command execution steps- **Homebrew Bundle**: Package management only (via Brewfile)

- Error messages for missing dependencies- **Sink**: Arbitrary shell commands + multi-platform support + idempotency patterns



**Key concepts:**They're complementary! Use Homebrew Bundle for packages, Sink for custom automation.

```json

{### What platforms does Sink support?

  "name": "Validation step",

  "check": "command -v jq",- ✅ macOS (Intel and Apple Silicon)

  "error": "jq is not installed"- ✅ Linux (all distributions)

}- ✅ FreeBSD

```- ⚠️ Windows (WSL2 only)



The `check` field runs a command and expects exit code 0. If it fails, the `error` message is displayed.## Configuration Questions



**Try it:**### Can I use environment variables in configurations?

```bash

sink validate examples/01-basic.jsonYes! Use shell expansion:

sink execute examples/01-basic.json --dry-run

``````json

{

---  "command": "echo $HOME"

}

### 02-multi-platform.json - Cross-Platform Support```



**Purpose:** Same configuration across different operating systemsOr capture them as facts:



**What it demonstrates:**```json

- Multiple platform blocks (darwin, linux, windows){

- Platform-specific commands  "facts": {

- OS detection and automatic platform selection    "home": {"command": "echo $HOME"}

- Fallback error handling  }

}

**Key concepts:**```

```json

{### Can facts reference other facts?

  "platforms": [

    {"os": "darwin", "install_steps": [...]},No. Facts are evaluated independently. Use shell arithmetic instead:

    {"os": "linux", "install_steps": [...]}

  ],❌ **Won't work:**

  "fallback": {"error": "Unsupported OS"}```json

}{

```  "facts": {

    "total": {"command": "echo 100"},

---    "half": {"command": "echo $(( {{facts.total}} / 2 ))"}

  }

### 03-distributions.json - Linux Distribution Handling}

```

**Purpose:** Handle different Linux distributions with one configuration

✅ **Use this:**

**What it demonstrates:**```json

- Linux distribution detection (Ubuntu, Debian, Fedora, Alpine, Arch){

- Distribution-specific package managers  "facts": {

- Distribution fallback handling    "total": {"command": "echo 100"}

  },

**Key concepts:**  "install_steps": [{

```json    "command": "HALF=$(( {{facts.total}} / 2 )); echo $HALF"

{  }]

  "os": "linux",}

  "distributions": [```

    {"ids": ["ubuntu", "debian"], "install_steps": [...]},

    {"ids": ["fedora", "rhel"], "install_steps": [...]}### Can I nest check-remediate patterns?

  ]

}No. Sink only supports one level of check-remediate. Use separate install steps for complex logic:

```

❌ **Won't work:**

---```json

{

### 04-facts.json - System Information Gathering  \"check\": \"...\",

  \"on_missing\": [{

**Purpose:** Gather system information and use it in installation steps    \"check\": \"...\",       // ❌ Not supported

    \"on_missing\": [...]

**What it demonstrates:**  }]

- Fact definitions with commands}

- Type coercion (string, integer, boolean)```

- Fact substitution via `{{facts.name}}`

- Environment variable export✅ **Use this:**

```json

**Key concepts:**{

```json  \"install_steps\": [

{    {\"check\": \"...\", \"on_missing\": [...]},

  "facts": {    {\"check\": \"...\", \"on_missing\": [...]}

    "hostname": {  ]

      "type": "string",}

      "command": "hostname",```

      "export": "SINK_HOSTNAME"

    },### How do I handle platform-specific commands?

    "cpu_count": {

      "type": "integer",Use the `platforms` array with OS-specific configurations:

      "command": "nproc"

    }```json

  },{

  "install_steps": [  \"platforms\": [{

    {    \"os\": \"darwin\",

      "command": "echo 'Host: {{facts.hostname}}, CPUs: {{facts.cpu_count}}'"    \"install_steps\": [{\"command\": \"brew install jq\"}]

    }  }, {

  ]    \"os\": \"linux\",

}    \"distributions\": [{

```      \"ids\": [\"ubuntu\"],

      \"install_steps\": [{\"command\": \"apt-get install jq\"}]

**Try it:**    }]

```bash  }]

sink facts examples/04-facts.json}

sink execute examples/04-facts.json```

```

## Execution Questions

---

### How do I preview changes without running them?

### 05-nested-steps.json - Conditional Execution

Use `--dry-run`:

**Purpose:** Conditional execution with check/on_missing pattern

```bash

**What it demonstrates:**sink execute config.json --dry-run

- Idempotent installations```

- Check-before-act pattern

- Multi-step remediation### How do I skip confirmation prompts?

- Automatic prerequisite handling

Use `--yes`:

**Key concepts:**

```json```bash

{sink execute config.json --yes

  "name": "Check if Homebrew exists",```

  "check": "command -v brew",

  "on_missing": [### Why did my command fail?

    {"name": "Install Homebrew", "command": "..."},

    {"name": "Configure PATH", "command": "..."}Check these common issues:

  ]

}1. **Command not in PATH**: Ensure tools are installed

```2. **Permission denied**: May need `sudo` or different user

3. **Syntax errors**: Validate configuration first

If the check succeeds, `on_missing` steps are skipped. This makes configurations idempotent.4. **Platform mismatch**: Check `--platform` override



---Debug steps:

```bash

### 06-retry.json - Service Readiness# Validate configuration

sink validate config.json

**Purpose:** Handle transient failures and wait for services

# Check facts

**What it demonstrates:**sink facts config.json

- Retry logic with `retry: "until"`

- Timeout specifications# Dry-run

- Service readiness checkssink execute config.json --dry-run

```

**Key concepts:**

```json### Can I run Sink in CI/CD?

{

  "name": "Wait for network",Yes! Use `--yes` to skip prompts:

  "command": "ping -c 1 8.8.8.8",

  "retry": "until",```bash

  "timeout": "30s"sink execute config.json --yes

}```

```

**GitHub Actions:**

The command runs repeatedly until it succeeds or the timeout is reached.```yaml

- name: Setup environment

---  run: sink execute .github/setup.json --yes

```

### 07-defaults.json - Reusable Values

**GitLab CI:**

**Purpose:** Reusable configurations with substitutable values```yaml

setup:

**What it demonstrates:**  script:

- Default value definitions    - sink execute ci/setup.json --yes

- Template substitution via `{{defaults.name}}````

- DRY principle (Don't Repeat Yourself)

## Troubleshooting

**Key concepts:**

```json### \"Configuration is invalid\" error

{

  "defaults": {Run validation to see specific errors:

    "package": "jq",

    "check_command": "command -v jq"```bash

  },sink validate config.json

  "install_steps": [```

    {

      "check": "{{defaults.check_command}}",Common issues:

      "command": "brew install {{defaults.package}}"- Missing required fields (`version`, `platforms`)

    }- Invalid JSON syntax

  ]- Wrong field types

}- Invalid platform identifiers

```

### \"Unsupported platform\" error

---

Your OS isn't covered by the configuration. Add a platform entry:

### 08-error-handling.json - Error Patterns

```json

**Purpose:** Different error handling patterns{

  \"platforms\": [{

**What it demonstrates:**    \"os\": \"linux\",

- Check-only steps (validation)    \"match\": \"linux*\",

- Error-only steps (command must succeed)    \"name\": \"Linux\",

- Check with remediation    \"install_steps\": [...]

- Custom error messages  }]

}

**Three patterns:**```

1. **Check-only**: Validate, don't fix (`check` + `error`)

2. **Error-only**: Command must succeed (`command` + `error`)### \"Command not found\" in check

3. **Check-remediate**: Fix if needed (`check` + `on_missing`)

The check command doesn't exist or isn't in PATH. Common fixes:

---

✅ **Check if command exists:**

## Bootstrap Examples```json

{\"check\": \"command -v jq\"}

### Bootstrap Overview```



The `bootstrap` command loads configurations from remote URLs, enabling centralized configuration management.✅ **Check if file exists:**

```json

**Key features:**{\"check\": \"test -f ~/.config/app/config.json\"}

- Load configs from HTTP/HTTPS URLs```

- GitHub URL pinning validation

- Automatic checksum verification (.sha256 files)### Facts not appearing

- Required SHA256 for insecure HTTP

- TLS certificate validation for HTTPSCheck these:

- Fact name follows pattern: `^[a-z_][a-z0-9_]*$`

### bootstrap-https-url.json- Command succeeds (test it manually)

- Platform-specific facts match your OS

**Purpose:** Example configuration for remote loading via HTTPS

View gathered facts:

**How to use:**```bash

```bashsink facts config.json

# Load from web server```

sink bootstrap https://example.com/config.json

### Retry timeout too short

# With auto-checksum (if .sha256 file exists)

sink bootstrap https://example.com/config.jsonIncrease timeout:



# Dry run```json

sink bootstrap https://example.com/config.json --dry-run{

```  \"command\": \"docker info\",

  \"retry\": \"until\",

**Security:**  \"timeout\": \"120s\"  // Was 30s

- HTTPS URLs validated via TLS certificates}

- Optional .sha256 file for integrity verification```

- Auto-fetched if exists alongside config

## Best Practices Questions

---

### Should I commit configurations to git?

### bootstrap-github-pinned.json

Yes! Configurations are portable and reproducible. Add them to your repository:

**Purpose:** Demonstrate GitHub URL pinning best practices

```bash

**Pinning strategies:**.github/

  setup.json

✅ **Recommended (Immutable):**.sink/

```bash  dependencies.json

# Semantic version tag  development.json

sink bootstrap https://raw.githubusercontent.com/org/repo/v1.0.0/config.json```



# Commit SHA### How do I organize multiple configurations?

sink bootstrap https://raw.githubusercontent.com/org/repo/abc123def/config.json

Use separate files for different concerns:

# GitHub Release

sink bootstrap https://github.com/org/repo/releases/download/v1.0.0/config.json```bash

```sink execute 01-dependencies.json

sink execute 02-applications.json

⚠️ **Not Recommended (Mutable):**sink execute 03-configuration.json

```bash```

# Branch names - content can change

sink bootstrap https://raw.githubusercontent.com/org/repo/main/config.json### Should I use Sink for production servers?

```

Sink is designed for:

**Why pinning matters:**- ✅ Development environments

- Immutable configs ensure reproducible deployments- ✅ Personal machine setup

- Version tags can't be changed once created- ✅ CI/CD bootstrapping

- Commit SHAs always point to the same content

For production:

---- ❌ Use Ansible/Chef/Puppet for complexity

- ❌ Use Docker/Kubernetes for containers

## Frequently Asked Questions- ⚠️ Sink can work for simple deployments



### General### How do I test configurations?



**Q: What is Sink?**1. **Validate syntax:** `sink validate config.json`

2. **Check facts:** `sink facts config.json`

A: Sink is a declarative, zero-dependency shell installation tool for idempotent, cross-platform system setup.3. **Dry-run:** `sink execute config.json --dry-run`

4. **Test on clean system:** Use Docker or VMs

**Q: What platforms does Sink support?**

## Advanced Questions

A: macOS, Linux (all distributions), FreeBSD, and Windows (WSL2 only).

### Can I generate configurations dynamically?

**Q: How is Sink different from Ansible/Chef/Puppet?**

Yes! Sink reads from stdin:

A: Sink is minimal (< 2,000 lines), has zero dependencies, and is designed for personal machine setup and development environments. Use Ansible/Chef/Puppet for enterprise infrastructure.

```bash\n# Generate and execute\ncat << 'EOF' | sink execute -\n{\n  \"version\": \"1.0.0\",\n  \"platforms\": [...]\n}\nEOF\n\n# Use jq to transform\njq '.platforms[0].install_steps += [{\"name\": \"Extra\", \"command\": \"...\"}]' base.json | sink execute -\n```\n\n### Can I use Sink with Docker?\n\nYes! Add Sink to your Dockerfile:\n\n```dockerfile\nFROM ubuntu:22.04\nRUN curl -L https://github.com/your-org/sink/releases/latest/download/sink-linux-amd64 -o /usr/local/bin/sink && chmod +x /usr/local/bin/sink\nCOPY setup.json /tmp/\nRUN sink execute /tmp/setup.json --yes\n```\n\n### How do I extend Sink?\n\nSink is intentionally minimal. For complex needs:\n\n1. **Use shell scripts** in commands\n2. **Chain configurations** sequentially\n3. **Combine with other tools** (Make, Just, Task)\n\n### Where is the cache directory?\n\n**Default:** `~/.cache/sink/`\n\n**Custom:**\n```bash\nexport SINK_CACHE_DIR=/tmp/sink\n```\n\n## Contributing\n\n### How do I report bugs?\n\nOpen an issue on GitHub with:\n- Sink version (`sink version`)\n- OS and architecture\n- Configuration file\n- Full error output\n\n### How do I contribute examples?\n\nContributions welcome! See `examples/` directory and CONTRIBUTING.md.\n\n### Can I add new features?\n\nSink is intentionally minimal (< 2,000 LOC). New features must:\n- Fit the core mission (declarative shell installation)\n- Not add dependencies\n- Not break existing configurations\n- Stay under LOC budget\n\nDiscuss in GitHub Discussions first.\n\n---\n\n[← Back: CLI Reference](cli-reference.md) | [Up: Docs](README.md)\n
---

### Configuration

**Q: Can I use environment variables?**

A: Yes, two ways:
1. Shell expansion: `{"command": "echo $HOME"}`
2. Facts: `{"facts": {"home": {"command": "echo $HOME"}}}`

**Q: How do I make configurations idempotent?**

A: Use the check/on_missing pattern:
```json
{
  "check": "command -v tool",
  "on_missing": [{"command": "install tool"}]
}
```

**Q: Can facts reference other facts?**

A: Yes! Facts are gathered in order:
```json
{
  "facts": {
    "cpu_count": {"command": "nproc"},
    "half_cpus": {"command": "echo $(( {{facts.cpu_count}} / 2 ))"}
  }
}
```

**Q: How do I test without running?**

A: Use `--dry-run`:
```bash
sink execute config.json --dry-run
```

---

### Bootstrap & Remote Configs

**Q: What's the difference between `execute` and `bootstrap`?**

A:
- `execute`: Runs local configuration files
- `bootstrap`: Loads configs from URLs, then executes them

**Q: Is bootstrap secure?**

A: Yes:
- ✅ HTTPS validated via TLS certificates
- ✅ GitHub pinning prevents mutable references
- ✅ SHA256 checksums verify integrity
- ✅ HTTP requires explicit SHA256 hash

**Q: How do I host bootstrap configs?**

A: Three options:
1. **GitHub** (recommended): Tag releases, use raw URLs with version tags
2. **Web server**: Host JSON + SHA256 files on HTTPS
3. **S3/Cloud Storage**: Upload with public read or signed URLs

**Q: What's GitHub URL pinning?**

A: Security feature validating GitHub URLs are immutable:
- ✅ Pinned: v1.0.0 tags, commit SHAs, releases
- ⚠️ Mutable: branch names (main, develop)

**Q: Can I bootstrap from private repositories?**

A: Yes, with authentication:
```bash
# GitHub Personal Access Token
sink bootstrap https://TOKEN@raw.githubusercontent.com/private/repo/v1.0.0/config.json

# Or clone locally
git clone git@github.com:org/repo.git
sink execute repo/config.json
```

---

### Debugging

**Q: How do I validate a configuration?**

A: Three-step validation:
1. Syntax: `sink validate config.json`
2. Facts: `sink facts config.json`  
3. Execution plan: `sink execute config.json --dry-run`

**Q: What happens when a step fails?**

A: Execution stops immediately with:
- Which step failed
- Error message (if provided)
- Command output/error
- Exit code

**Q: How do I see what commands will run?**

A: Use `--dry-run`:
```bash
sink execute config.json --dry-run
```

---

### Best Practices

**Q: Should I put everything in one config?**

A: Split by concern:
- ✅ One config per tool/service
- ✅ Chain configs: `sink execute base.json && sink execute app.json`
- ❌ Don't create monolithic configs

**Q: How do I handle secrets?**

A: Don't put secrets in configs. Instead:
1. Use environment variables: `$SECRET_KEY`
2. Read from secure files: `cat ~/.secrets/key`
3. Use system keychains
4. Prompt user: `read -s -p "Password: " pass`

**Q: Can I use Sink in CI/CD?**

A: Yes! Common pattern:
```yaml
- name: Setup environment
  run: |
    curl -fsSL https://sink-install-url | bash
    sink bootstrap https://configs/v1.0.0/ci.json
```

---

## Getting Help

- **Documentation**: See `docs/` directory
- **Schema**: `sink schema` or `src/sink.schema.json`  
- **Command help**: `sink help <command>`
- **GitHub**: https://github.com/radiolabme/sink

---

## Contributing Examples

To add a new example:

1. Create a focused `.json` file demonstrating one pattern
2. Validate it: `sink validate examples/your-example.json`
3. Test on a fresh system
4. Add entry to this FAQ
5. Submit a PR

Keep examples simple, focused, and well-documented!
