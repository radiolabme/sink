# Sink Architecture

## Overview

Sink is a declarative, idempotent shell command execution framework designed for system configuration and installation tasks. The architecture emphasizes safety, reliability, and cross-platform compatibility while maintaining simplicity and zero external dependencies.

## Design Principles

### Core Tenets

1. **Declarative Configuration**: Users describe desired state, not imperative steps
2. **Idempotency**: Safe to run multiple times with same results  
3. **Platform Awareness**: Automatic platform detection with override capabilities
4. **Safety First**: Dry-run mode, user confirmation, and validation
5. **Zero Dependencies**: Pure Go standard library implementation
6. **Fail Fast**: Early validation and clear error reporting

### Security Model

- **Least Privilege**: Commands run with user permissions by default
- **Explicit Escalation**: Sudo/admin requirements must be declared
- **Transport Security**: HTTPS enforced for remote configs, checksum verification
- **Supply Chain Protection**: GitHub URL pinning validation and warnings
- **Command Validation**: Pattern matching for dangerous operations

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Command Line Interface                  │
├─────────────────────────────────────────────────────────────┤
│ execute │ bootstrap │ remote │ facts │ validate │ schema    │
└─────────┬───────────┬────────┬───────┬──────────┬───────────┘
          │           │        │       │          │
          ▼           ▼        ▼       ▼          ▼
┌─────────────────────────────────────────────────────────────┐
│                    Core Execution Engine                    │
├─────────────────┬───────────────────┬───────────────────────┤
│   Config        │    Fact           │     Platform          │
│   Loader        │    Gatherer       │     Detector          │
├─────────────────┼───────────────────┼───────────────────────┤
│   Validator     │    Template       │     Executor          │
│                 │    Engine         │                       │
├─────────────────┴───────────────────┴───────────────────────┤
│                    Transport Layer                          │
├─────────────────────────────────────────────────────────────┤
│              Local │      SSH       │     Container         │
│             Command│    Remote      │      Execution        │
└─────────────────────────────────────────────────────────────┘
```

## Component Architecture

### 1. Configuration System

#### Config Loader (`config.go`)
- **Purpose**: Parse and validate JSON configurations
- **Responsibilities**:
  - JSON schema validation against embedded schema
  - Configuration file loading from local/remote sources
  - Bootstrap URL handling with security validation
  - Error reporting with location context

**Key Types:**
```go
type Config struct {
    Version     string               `json:"version"`
    Facts       map[string]string    `json:"facts"`
    Platforms   []Platform           `json:"platforms"`
}

type Platform struct {
    OS           string    `json:"os"`
    Name         string    `json:"name"`
    InstallSteps []Step    `json:"install_steps"`
}

type Step struct {
    Name    string `json:"name"`
    Command string `json:"command"`
    Check   string `json:"check"`
}
```

#### Schema System
- **Embedded Schema**: JSON schema bundled in binary (`sink.schema.json`)
- **Validation**: Runtime validation against schema
- **Evolution**: Version-aware schema with compatibility checks

### 2. Fact System

#### Fact Gatherer (`facts.go`)
- **Purpose**: Collect system information for template interpolation
- **Architecture**: Pluggable fact providers with caching
- **Execution**: Commands run through transport layer
- **Template Integration**: Facts available as `{{.facts.name}}` in configs

**Current Facts:**
- System: CPU cores, memory, hostname, OS details
- Environment: User, working directory, PATH
- Platform: Package managers, installed tools
- Custom: User-defined commands for dynamic facts

**Future Enhancement - Multi-Phase Facts:**
```go
type FactPhase int
const (
    PhaseBootstrap FactPhase = iota  // Basic system info
    PhasePlatform                    // Platform-specific facts  
    PhaseRuntime                     // Dependency-conditional facts
)
```

### 3. Execution Engine

#### Executor (`executor.go`)
- **Purpose**: Execute installation steps with safety guarantees
- **Features**:
  - Dry-run mode for safe preview
  - Real-time progress reporting
  - Error handling and recovery
  - Event-driven execution model

**Execution Flow:**
1. **Pre-execution**: Check validation, user confirmation
2. **Step Execution**: Run commands through transport
3. **Check Validation**: Verify step completion
4. **Event Emission**: Progress updates to UI
5. **Error Handling**: Collect context, suggest fixes

#### Transport Layer
- **Local Transport**: Direct command execution via `os/exec`
- **SSH Transport**: Remote execution with connection pooling  
- **Container Transport**: Docker/Podman execution (future)

### 4. Platform System

#### Platform Detection
- **OS Detection**: `runtime.GOOS` with manual override
- **Distribution Detection**: Parsing `/etc/os-release`, system commands
- **Architecture**: `runtime.GOARCH` for CPU architecture
- **Environment**: Container detection, CI/CD environment awareness

#### Platform Matching
```go
type PlatformMatcher struct {
    OS           string   `json:"os"`           // darwin, linux, windows
    Distribution string   `json:"distribution"` // ubuntu, fedora, macos
    Version      string   `json:"version"`      // 20.04, 11, etc.
    Architecture string   `json:"architecture"` // amd64, arm64
}
```

### 5. Security Architecture

#### Bootstrap Security
- **URL Validation**: GitHub pinning detection and warnings
- **Checksum Verification**: SHA256 hash validation
- **Transport Security**: HTTPS enforcement for remote configs
- **Source Validation**: Allowlisted domains and repositories

#### Command Security
- **Pattern Detection**: Scanning for dangerous command patterns
- **Privilege Escalation**: Explicit sudo/admin requirement declarations
- **Sandbox Mode**: Container-based execution for testing
- **Audit Trail**: Command logging and execution context

#### GitHub URL Pinning
```go
type GitHubPinType int
const (
    GitHubPinBranch  GitHubPinType = iota // Mutable - warns user
    GitHubPinTag                          // Immutable - recommended
    GitHubPinCommit                       // Immutable - highest security
    GitHubPinRelease                      // Immutable - release assets
)
```

### 6. Error Handling & Observability

#### Error System
- **Structured Errors**: Rich error context with suggestions
- **Error Recovery**: Rollback capabilities and cleanup
- **User Guidance**: Actionable error messages with next steps

#### Event System
```go
type ExecutionEvent struct {
    StepName  string    `json:"step_name"`
    Status    string    `json:"status"`    // running, success, failed, skipped
    Output    string    `json:"output"`
    Error     string    `json:"error"`
    Duration  time.Duration `json:"duration"`
    Context   ExecutionContext `json:"context"`
}
```

#### Logging & Tracing
- **Structured Logging**: JSON format with contextual information
- **Execution Tracing**: Step-by-step execution timeline
- **Performance Metrics**: Timing, resource usage, success rates

## Data Flow

### 1. Configuration Loading Flow
```
User Input → File/URL → JSON Parser → Schema Validator → Config Object
                                            ↓
Bootstrap URL → Security Check → Download → Checksum Verify → Parse
```

### 2. Execution Flow  
```
Config → Facts Gathering → Platform Selection → Step Execution → Results
   ↓         ↓                     ↓                ↓             ↓
Validate → Template → Match OS → Transport → Events → Summary
```

### 3. Bootstrap Flow
```
URL → GitHub Pin Check → Download → Checksum → Parse → Execute
  ↓         ↓             ↓         ↓         ↓       ↓
Security → Immutable → HTTP/HTTPS → SHA256 → JSON → Steps
Warning    Validation   Transport   Verify   Parse   Execute
```

## Extension Points

### 1. Transport Providers
```go
type Transport interface {
    Execute(command string, env map[string]string) (*Result, error)
    GetContext() ExecutionContext
    Close() error
}

// Implementations:
// - LocalTransport: Direct execution
// - SSHTransport: Remote execution  
// - ContainerTransport: Docker/Podman
// - MockTransport: Testing
```

### 2. Fact Providers
```go
type FactProvider interface {
    Name() string
    Gather(transport Transport) (map[string]interface{}, error)
    Dependencies() []string
}

// Built-in providers:
// - SystemFactProvider: OS, CPU, memory
// - EnvironmentFactProvider: User, PATH, working directory
// - PackageFactProvider: Installed packages, versions
// - CustomFactProvider: User-defined commands
```

### 3. Platform Detectors
```go
type PlatformDetector interface {
    Detect() (*PlatformInfo, error)
    Name() string
    Priority() int
}

// Detectors:
// - LinuxDetector: /etc/os-release parsing
// - MacOSDetector: system_profiler, sw_vers
// - WindowsDetector: registry, WMI queries
// - ContainerDetector: Docker/Kubernetes environment
```

### 4. Validators
```go
type ConfigValidator interface {
    Validate(config *Config) []ValidationError
    Name() string
}

// Validators:
// - SchemaValidator: JSON schema compliance
// - SecurityValidator: Command pattern analysis
// - PlatformValidator: Platform configuration consistency
// - DependencyValidator: Step dependency validation
```

## Performance Characteristics

### Memory Usage
- **Configuration**: ~1-10KB per config (JSON parsing)
- **Facts**: ~1-5KB per fact set (system information)
- **Execution**: ~100KB-1MB (command output buffering)
- **Transport**: ~10-100KB per connection (SSH state)

### Execution Time
- **Startup**: <50ms (binary loading, schema parsing)
- **Fact Gathering**: 100-500ms (system command execution)
- **Validation**: <10ms (JSON schema validation)
- **Step Execution**: Variable (depends on commands)

### Scalability
- **Concurrent Steps**: Limited by system resources
- **Remote Hosts**: Connection pooling, parallel execution
- **Large Configs**: Streaming JSON parsing (future)
- **Enterprise Scale**: Plugin architecture, distributed execution

## Testing Architecture

### Unit Testing
- **Component Tests**: Individual function validation
- **Mock Transports**: Isolated execution testing
- **Property Tests**: Configuration validation edge cases
- **Performance Tests**: Benchmarking critical paths

### Integration Testing
- **End-to-End**: Complete workflow validation
- **Platform Tests**: Multi-OS execution validation
- **Network Tests**: HTTP/HTTPS download scenarios
- **Security Tests**: Attack scenario validation

### Test Infrastructure
```go
// Test harness
type TestEnvironment struct {
    Transport    MockTransport
    TempDir      string
    ConfigFiles  map[string]string
    Facts        map[string]interface{}
}

// Test scenarios
type TestScenario struct {
    Name          string
    Config        string
    ExpectedSteps int
    ShouldFail    bool
    Platform      string
}
```

## Security Model

### Threat Model
1. **Malicious Configurations**: Remote configs with harmful commands
2. **Supply Chain Attacks**: Compromised configuration repositories
3. **Privilege Escalation**: Unauthorized system access
4. **Data Exfiltration**: Sensitive information exposure
5. **System Compromise**: Destructive command execution

### Security Controls
1. **Input Validation**: Schema validation, command pattern analysis
2. **Transport Security**: HTTPS, checksum verification, pinned sources
3. **Execution Security**: User permissions, explicit escalation
4. **Audit Logging**: Command execution logs, security events
5. **Sandboxing**: Container-based execution for testing

### Security Boundaries
- **User Boundary**: Commands run as invoking user
- **Network Boundary**: HTTPS required for remote configs
- **System Boundary**: No automatic privilege escalation
- **Process Boundary**: Isolated command execution

## Future Roadmap

### Phase 1: Core Stability (Current)
- ✅ Basic execution engine
- ✅ Configuration validation
- ✅ Bootstrap functionality
- ✅ Platform detection
- 🚧 Comprehensive testing

### Phase 2: Enhanced Features
- 🔄 Multi-phase fact system
- 🔄 Rollback capabilities
- 🔄 Enhanced error handling
- 🔄 Performance optimization
- 🔄 Security hardening

### Phase 3: Advanced Capabilities
- ⏳ Plugin architecture
- ⏳ Distributed execution
- ⏳ GUI interface
- ⏳ Enterprise integration
- ⏳ Cloud-native features

### Phase 4: Ecosystem
- ⏳ VS Code extension
- ⏳ CI/CD integrations
- ⏳ Package repository
- ⏳ Community plugins
- ⏳ Enterprise features

## Contributing Guidelines

### Code Organization
- **Single Responsibility**: Each component has one clear purpose
- **Interface Segregation**: Small, focused interfaces
- **Dependency Injection**: Testable, mockable dependencies
- **Error Handling**: Comprehensive error context and recovery

### Development Workflow
1. **Design**: Architecture discussion and documentation
2. **Implementation**: TDD with comprehensive test coverage
3. **Review**: Code review focusing on security and reliability
4. **Testing**: Multi-platform validation before merge
5. **Documentation**: Update architecture docs and examples

### Quality Standards
- **Test Coverage**: >80% for critical paths
- **Documentation**: All public APIs documented
- **Performance**: No regressions, benchmark validation
- **Security**: Threat model validation, security review
- **Compatibility**: Backwards compatibility preservation

---

*This architecture document is living documentation that evolves with the Sink project. Last updated: October 2025*