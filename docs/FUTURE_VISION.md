# Sink: Future Vision & Strategic Architecture

**Date:** October 4, 2025  
**Status:** Vision Document

## Philosophy: Constraints Create Opportunities

> "Perfection is achieved not when there is nothing more to add, but when there is nothing more to take away." - Antoine de Saint-Exupéry

### Core Principle: Stay Simple, Stay Understandable

Sink must remain **understandable at a glance**. Every feature must justify its complexity. The path forward is not to add features, but to create **extension points** that enable unlimited capability without core complexity.

---

## The Caddy Inspiration

### What Makes Caddy Brilliant

**Caddy's genius:**
- Simple core: Web server that "just works"
- Zero-config HTTPS (radical simplicity)
- **Plugin architecture** via Go modules
- **Caddyfile DSL** (declarative, human-readable)
- Extensible without bloat

**Key insight:** Caddy didn't try to be everything. It provided:
1. A solid core (HTTP/TLS)
2. Clean interfaces (HTTP handlers, middleware)
3. Plugin discovery (Caddy module ecosystem)
4. Simple configuration (Caddyfile)

**Result:** Thousands of plugins, zero core bloat.

---

## Sink's Architecture: Lessons from Caddy

### Current Strengths (Keep These)

✅ **Zero dependencies** (stdlib only)  
✅ **Declarative configs** (JSON schema)  
✅ **Platform abstraction** (darwin/linux/windows)  
✅ **Facts system** (runtime discovery)  
✅ **Transport interface** (local today, SSH tomorrow)  
✅ **Event system** (callbacks for extensibility)

### Architectural Constraints (The Foundation)

#### 1. **Core = Execution Engine Only**

```
┌─────────────────────────────────────┐
│         SINK CORE (~1500 LOC)       │
│                                     │
│  • Config parser                    │
│  • Executor                         │
│  • Transport interface              │
│  • Event system                     │
│  • Facts engine                     │
│                                     │
│  NO business logic                  │
│  NO external APIs                   │
│  NO tool-specific code              │
└─────────────────────────────────────┘
           ↓ Extension Points
┌─────────────────────────────────────┐
│           PLUGINS                   │
└─────────────────────────────────────┘
```

#### 2. **Everything Else = Plugins**

**Rule:** If it's not about executing commands, it's a plugin.

---

## Plugin Architecture

### Design: Go Plugins via Interfaces

**Not like Caddy** (Go plugins are fragile)  
**Instead:** Embedded plugins via interface registration

```go
// Core interface
type Plugin interface {
    Name() string
    Version() string
    Init(config map[string]interface{}) error
}

// Plugin types
type TransportPlugin interface {
    Plugin
    Run(command string) (stdout, stderr string, exitCode int, err error)
}

type OutputPlugin interface {
    Plugin
    OnEvent(event ExecutionEvent) error
    Finalize() error
}

type ValidatorPlugin interface {
    Plugin
    ValidateConfig(config Config) []ValidationError
}

type GeneratorPlugin interface {
    Plugin
    Generate(config Config, format string) (string, error)
}
```

### Plugin Registration

```go
// plugins/registry.go
var (
    transports = make(map[string]TransportPlugin)
    outputs    = make(map[string]OutputPlugin)
    validators = make(map[string]ValidatorPlugin)
    generators = make(map[string]GeneratorPlugin)
)

func RegisterTransport(name string, plugin TransportPlugin) {
    transports[name] = plugin
}

// In plugin code
func init() {
    plugins.RegisterTransport("ssh", &SSHTransport{})
}
```

---

## Killer Plugins (High Impact, Low Core Complexity)

### 1. **Transport Plugins** 🚀

**Purpose:** Where commands execute

```go
type TransportPlugin interface {
    Run(command string) (stdout, stderr string, exitCode int, err error)
    Connect(target string, opts map[string]interface{}) error
    Disconnect() error
}
```

**Built-in:**
- `local` - Current implementation

**Plugin ideas:**
- `ssh` - Remote execution via SSH
- `docker` - Execute in container
- `kubernetes` - Execute in pod
- `systemd` - Execute via systemd-run (isolation)
- `wsl` - Execute in WSL from Windows
- `vagrant` - Execute in Vagrant VM

**Config:**
```json
{
  "transport": "ssh",
  "transport_config": {
    "host": "app-server.example.com",
    "user": "deploy",
    "key": "~/.ssh/id_rsa"
  }
}
```

---

### 2. **Output Plugins** 🎯

**Purpose:** What happens with execution results

```go
type OutputPlugin interface {
    OnEvent(event ExecutionEvent) error
    Finalize() error
}
```

**Built-in:**
- `console` - Current stdout display

**Plugin ideas:**
- `json` - Structured JSON output
- `junit` - JUnit XML for CI
- `prometheus` - Metrics export
- `syslog` - System logging
- `webhook` - POST to URL
- `s3` - Upload logs to S3
- `database` - Store in Postgres/SQLite
- `replay` - Generate shell script

**Config:**
```json
{
  "outputs": [
    {"type": "console", "verbose": true},
    {"type": "json", "file": "execution.json"},
    {"type": "webhook", "url": "https://api.example.com/events"}
  ]
}
```

---

### 3. **Generator Plugins** 💎

**Purpose:** Transform sink configs to other formats

```go
type GeneratorPlugin interface {
    Generate(config Config, opts map[string]interface{}) (string, error)
}
```

**Plugin ideas:**
- `systemd` - Generate .service files
- `docker` - Generate Dockerfile
- `ansible` - Generate playbook.yml
- `terraform` - Generate provisioner blocks
- `github-actions` - Generate workflow YAML
- `makefile` - Generate Makefile
- `justfile` - Generate Justfile
- `bash` - Generate standalone script

**CLI:**
```bash
sink generate systemd install-config.json > myapp.service
sink generate ansible install-config.json > playbook.yml
sink generate bash install-config.json > install.sh
```

---

### 4. **Validator Plugins** ✅

**Purpose:** Extend config validation

```go
type ValidatorPlugin interface {
    Validate(config Config) []ValidationError
}
```

**Plugin ideas:**
- `security` - Check for unsafe commands (curl | sh)
- `best-practices` - Idempotency checks
- `performance` - Detect slow operations
- `compliance` - Ensure org policies
- `cost` - Estimate cloud costs
- `dependencies` - Check for circular deps

**Config:**
```json
{
  "validators": [
    {"type": "security", "level": "strict"},
    {"type": "best-practices"},
    {"type": "compliance", "policy": "company-policy.json"}
  ]
}
```

---

### 5. **Facts Plugins** 🔍

**Purpose:** Extend fact gathering

```go
type FactsPlugin interface {
    GatherFacts() (Facts, error)
}
```

**Plugin ideas:**
- `cloud` - AWS/GCP/Azure metadata
- `kubernetes` - Cluster info
- `docker` - Container info
- `systemd` - Service states
- `hardware` - CPU/RAM/Disk
- `network` - Network topology
- `packages` - Installed packages
- `git` - Repo information

**Config:**
```json
{
  "facts": {
    "aws_region": {"plugin": "cloud.aws", "key": "region"},
    "k8s_namespace": {"plugin": "kubernetes", "key": "namespace"},
    "installed_packages": {"plugin": "packages.list"}
  }
}
```

---

## Killer Webhooks 🪝

### Webhook Architecture

**Concept:** Fire webhooks at execution lifecycle events

```go
type WebhookConfig struct {
    URL     string            `json:"url"`
    Events  []string          `json:"events"`
    Headers map[string]string `json:"headers"`
    Retry   RetryConfig       `json:"retry"`
}

// Events
const (
    EventExecutionStart = "execution.start"
    EventExecutionEnd   = "execution.end"
    EventStepStart      = "step.start"
    EventStepSuccess    = "step.success"
    EventStepFailure    = "step.failure"
    EventFactsGathered  = "facts.gathered"
    EventContextDiscovered = "context.discovered"
)
```

### Use Cases

#### 1. **CI/CD Integration**

```json
{
  "webhooks": [{
    "url": "https://ci.example.com/sink-events",
    "events": ["execution.start", "execution.end"],
    "headers": {
      "Authorization": "Bearer ${CI_TOKEN}"
    }
  }]
}
```

**Payload:**
```json
{
  "event": "execution.end",
  "run_id": "20251004-123456-abc",
  "config": "install-config.json",
  "context": {
    "host": "build-agent-3",
    "user": "ci",
    "os": "linux"
  },
  "result": {
    "success": true,
    "duration_ms": 12345,
    "steps_total": 5,
    "steps_success": 5
  }
}
```

#### 2. **Slack Notifications**

```json
{
  "webhooks": [{
    "url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    "events": ["step.failure", "execution.end"],
    "format": "slack"
  }]
}
```

#### 3. **Audit Logging**

```json
{
  "webhooks": [{
    "url": "https://logs.example.com/audit",
    "events": ["*"],
    "headers": {
      "X-Audit-Source": "sink",
      "X-Compliance-Level": "strict"
    }
  }]
}
```

#### 4. **Metrics Collection**

```json
{
  "webhooks": [{
    "url": "https://metrics.example.com/sink",
    "events": ["execution.end"],
    "format": "prometheus"
  }]
}
```

#### 5. **Approval Workflows**

```json
{
  "webhooks": [{
    "url": "https://approval.example.com/request",
    "events": ["execution.start"],
    "wait_for_approval": true,
    "timeout": "5m"
  }]
}
```

**Flow:**
1. Sink sends `execution.start` webhook
2. Waits for approval API response
3. If approved: continues
4. If denied: aborts

---

## Integration Strategies

### 1. **Backstage Integration**

**Plugin:** `output.backstage`

```go
type BackstageOutput struct {
    APIUrl string
    Token  string
}

func (b *BackstageOutput) OnEvent(event ExecutionEvent) error {
    // POST event to Backstage TechDocs API
    // Update service status in catalog
    // Record deployment in timeline
}
```

**Config:**
```json
{
  "outputs": [{
    "type": "backstage",
    "api_url": "https://backstage.example.com/api",
    "token": "${BACKSTAGE_TOKEN}",
    "catalog_entity": "component:default/payment-api"
  }]
}
```

**Use case:** Service setup from Backstage templates

---

### 2. **systemd Integration**

**Generator:** `generate.systemd`

```go
func (s *SystemdGenerator) Generate(config Config) string {
    unit := "[Unit]\n"
    unit += fmt.Sprintf("Description=%s\n", config.Description)
    
    // Extract from config
    for _, step := range config.Platforms[0].InstallSteps {
        if step.Check != "" {
            unit += fmt.Sprintf("ExecStartPre=%s\n", step.Check)
        }
        // Convert command steps to ExecStart
        // Convert on_missing to ExecStartPost
    }
    
    unit += "\n[Service]\n"
    unit += "Type=simple\n"
    unit += "Restart=on-failure\n"
    
    return unit
}
```

**CLI:**
```bash
sink generate systemd app-config.json > /etc/systemd/system/myapp.service
systemctl daemon-reload
systemctl enable --now myapp
```

---

### 3. **Terraform Integration**

**Provisioner:**
```hcl
resource "aws_instance" "app" {
  # ... instance config

  provisioner "remote-exec" {
    inline = [
      "curl -sL https://get.sink.sh | sh",
      "sink execute https://configs.example.com/app-setup.json"
    ]
  }
}
```

**Or via webhook:**
```hcl
resource "null_resource" "sink_execution" {
  triggers = {
    config_hash = filesha256("app-setup.json")
  }

  provisioner "local-exec" {
    command = "sink execute app-setup.json --webhook ${var.terraform_cloud_webhook}"
  }
}
```

---

## Code Organization for Plugins

### Directory Structure

```
sink/
├── src/
│   ├── main.go              # CLI entry point
│   ├── executor.go          # Core execution engine
│   ├── config.go            # Config parsing
│   ├── facts.go             # Facts system
│   ├── transport.go         # Transport interface + local impl
│   ├── types.go             # Core types
│   └── events.go            # Event system
│
├── plugins/
│   ├── registry.go          # Plugin registration
│   ├── interfaces.go        # Plugin interfaces
│   │
│   ├── transports/
│   │   ├── ssh/
│   │   │   ├── ssh.go
│   │   │   └── ssh_test.go
│   │   ├── docker/
│   │   └── kubernetes/
│   │
│   ├── outputs/
│   │   ├── json/
│   │   ├── webhook/
│   │   ├── prometheus/
│   │   └── backstage/
│   │
│   ├── generators/
│   │   ├── systemd/
│   │   ├── ansible/
│   │   ├── terraform/
│   │   └── bash/
│   │
│   └── validators/
│       ├── security/
│       └── compliance/
│
├── data/                    # Config files
├── docs/                    # Documentation
└── test/                    # Test artifacts
```

### Plugin Interface Examples

```go
// plugins/interfaces.go
package plugins

type Plugin interface {
    Name() string
    Version() string
    Init(config map[string]interface{}) error
}

type TransportPlugin interface {
    Plugin
    Run(command string) (TransportResult, error)
}

type TransportResult struct {
    Stdout   string
    Stderr   string
    ExitCode int
    Duration time.Duration
}

type OutputPlugin interface {
    Plugin
    OnEvent(event ExecutionEvent) error
    Finalize(result ExecutionResult) error
}

type GeneratorPlugin interface {
    Plugin
    Generate(config Config) ([]byte, error)
    Format() string // "systemd", "ansible", etc.
}

type ValidatorPlugin interface {
    Plugin
    Validate(config Config) []ValidationError
}
```

---

## Plugin Configuration

### In Config File

```json
{
  "version": "1.0.0",
  "description": "Multi-output execution",
  
  "plugins": {
    "transports": {
      "ssh": {
        "host": "app-server.example.com",
        "user": "deploy",
        "key": "~/.ssh/id_rsa"
      }
    },
    "outputs": {
      "console": {"verbose": true},
      "json": {"file": "execution.json"},
      "webhook": {
        "url": "https://api.example.com/events",
        "events": ["execution.end", "step.failure"]
      },
      "backstage": {
        "api_url": "https://backstage.example.com",
        "entity": "component:default/myapp"
      }
    },
    "validators": {
      "security": {"level": "strict"},
      "compliance": {"policy": "company-policy.json"}
    }
  },
  
  "platforms": [...]
}
```

### Via CLI Flags

```bash
# Use SSH transport
sink execute --transport ssh://deploy@app-server config.json

# Multiple outputs
sink execute \
  --output console:verbose=true \
  --output json:file=result.json \
  --output webhook:url=https://... \
  config.json

# Enable validators
sink validate \
  --validator security \
  --validator compliance:policy=policy.json \
  config.json

# Generate systemd unit
sink generate systemd config.json > myapp.service
```

---

## Constraints That Create Opportunities

### 1. **Constraint: No External Dependencies**
**Opportunity:** Pure Go plugins can be embedded  
**Result:** Fast, portable, no dependency hell

### 2. **Constraint: Simple Config Schema**
**Opportunity:** Easy to parse, validate, transform  
**Result:** Generators can create configs programmatically

### 3. **Constraint: Event-Driven Architecture**
**Opportunity:** Plugins react to events  
**Result:** Composable behaviors without coupling

### 4. **Constraint: Interface-Based Design**
**Opportunity:** Plugins implement standard interfaces  
**Result:** Test with mocks, swap implementations

### 5. **Constraint: Idempotent Execution Model**
**Opportunity:** Safe to retry, replay, recover  
**Result:** Reliable automation, audit trails

---

## Killer Feature Combinations

### 1. **Sink + Backstage + Webhook = Self-Service Infrastructure**

```
Developer creates service in Backstage
  ↓
Backstage calls sink webhook
  ↓
Sink executes setup on target host
  ↓
Sink sends status back to Backstage
  ↓
Service appears as "Running" in catalog
```

### 2. **Sink + systemd + Generator = Service Management**

```
Write sink config once
  ↓
Generate systemd unit: sink generate systemd
  ↓
Install and enable: systemctl enable myapp
  ↓
Auto-restart, logging, dependencies handled by systemd
```

### 3. **Sink + Terraform + SSH Transport = Complete IaC**

```
Terraform provisions VM
  ↓
Sink (SSH transport) configures VM
  ↓
Sink (webhook) notifies Backstage
  ↓
Backstage shows service in catalog
```

### 4. **Sink + CI/CD + JSON Output = Test Automation**

```
CI runs: sink execute tests.json --output json
  ↓
Parse JSON for results
  ↓
Convert to JUnit XML
  ↓
Publish to test reporting
```

---

## Implementation Roadmap

### Phase 1: Plugin Foundation (2-3 weeks)
- Define plugin interfaces
- Implement registry system
- Add plugin config loading
- Create plugin loader

### Phase 2: Core Plugins (2-3 weeks)
- SSH transport plugin
- JSON output plugin
- Webhook output plugin
- systemd generator plugin

### Phase 3: Integrations (2-3 weeks)
- Backstage output plugin
- Terraform examples
- CI/CD templates
- Documentation

### Phase 4: Ecosystem (Ongoing)
- Community plugins
- Plugin marketplace
- Best practices guide
- Example configs

---

## Success Metrics

**Core stays simple:**
- Core LOC < 2000 (currently ~1300)
- Zero external dependencies
- Test coverage > 50%
- Build time < 5s

**Plugins enable power:**
- 5+ official plugins in 6 months
- 10+ community plugins in 1 year
- 100+ configs shared in registry

**Integration adoption:**
- 3+ major tool integrations (Backstage, Terraform, etc.)
- 10+ companies using in production
- 1000+ GitHub stars

---

## Conclusion

**The Vision:**

Sink becomes the **universal execution layer** for infrastructure automation:

- **Simple core** (like Caddy)
- **Powerful plugins** (extend without bloat)
- **Standard interfaces** (compose freely)
- **Rich ecosystem** (community-driven)

**The Constraint:**

Stay understandable. Every feature must justify its existence. Complexity lives in plugins, not core.

**The Opportunity:**

By constraining the core and opening extension points, we create a platform that can integrate with anything while remaining simple enough to understand in an afternoon.

**Next Steps:**

1. Implement plugin interfaces
2. Build 3-5 killer plugins (SSH, webhook, systemd)
3. Demonstrate Backstage + systemd integration
4. Document plugin development guide
5. Launch plugin ecosystem

---

*"Simplicity is the ultimate sophistication."* - Leonardo da Vinci
