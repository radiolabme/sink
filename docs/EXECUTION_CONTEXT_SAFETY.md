# Execution Context Safety and Observability

## The Problem

When running commands over SSH (or even locally), there's a critical safety gap:

1. **Environment blindness** - We don't know the actual environment state before running commands
2. **Wrong host risk** - Commands could run on wrong machine (dev vs prod)
3. **State drift** - Remote environment may have changed since last run
4. **No pre-flight checks** - We start executing before verifying we're in the right place
5. **Limited visibility** - Events show command output, but not execution context

### Example Dangerous Scenarios

```bash
# Developer thinks they're on staging
./sink execute --ssh user@staging.example.com install-colima.json

# But DNS changed, now pointing to production!
# Or SSH config has wrong host alias
# Commands run on PROD instead of staging ⚠️

# Result: Accidentally install dev tools on production server
```

## Proposed Solution: Execution Context Guards

### 1. Pre-Flight Environment Discovery

Before executing ANY steps, gather and display the execution context:

```go
type ExecutionContext struct {
    Host        string            `json:"host"`         // Hostname where running
    User        string            `json:"user"`         // User running as
    WorkDir     string            `json:"work_dir"`     // Current working directory
    OS          string            `json:"os"`           // Operating system
    Arch        string            `json:"arch"`         // Architecture
    Environment map[string]string `json:"environment"`  // Key env vars
    Timestamp   string            `json:"timestamp"`    // When context captured
    Transport   string            `json:"transport"`    // "local" or "ssh:host"
}
```

### 2. Execution Guards in Config

Add optional guards to config to prevent wrong-environment execution:

```json
{
  "version": "1.0.0",
  "guards": {
    "required_hostname_pattern": "^staging-.*$",
    "forbidden_hostname_pattern": "^prod-.*$",
    "required_user": "deploy",
    "forbidden_users": ["root"],
    "required_env_vars": {
      "DEPLOYMENT_ENV": "staging"
    },
    "min_disk_space_gb": 10,
    "confirm_prompt": "You are about to install on {hostname}. Continue? [yes/no]"
  },
  "platforms": [...]
}
```

### 3. Context Visibility in Events

Every event should include where it's running:

```go
type ExecutionEvent struct {
    Timestamp string           `json:"timestamp"`
    RunID     string           `json:"run_id"`
    StepName  string           `json:"step_name"`
    Status    string           `json:"status"`
    Output    string           `json:"output,omitempty"`
    Error     string           `json:"error,omitempty"`
    Context   ExecutionContext `json:"context"`  // NEW: Always know WHERE
}
```

## Implementation Plan

### Phase 1: Execution Context Discovery (Simple)

Add context discovery to executor initialization:

```go
// executor.go
type Executor struct {
    transport Transport
    DryRun    bool
    OnEvent   func(ExecutionEvent)
    runID     string
    context   ExecutionContext  // NEW
}

func NewExecutor(transport Transport) *Executor {
    executor := &Executor{
        transport: transport,
        runID:     generateRunID(),
    }
    
    // Discover execution context immediately
    executor.context = executor.discoverContext()
    
    return executor
}

func (e *Executor) discoverContext() ExecutionContext {
    ctx := ExecutionContext{
        Timestamp: time.Now().Format(time.RFC3339),
    }
    
    // Discover hostname
    stdout, _, exitCode, _ := e.transport.Run("hostname")
    if exitCode == 0 {
        ctx.Host = strings.TrimSpace(stdout)
    }
    
    // Discover user
    stdout, _, exitCode, _ = e.transport.Run("whoami")
    if exitCode == 0 {
        ctx.User = strings.TrimSpace(stdout)
    }
    
    // Discover working directory
    stdout, _, exitCode, _ = e.transport.Run("pwd")
    if exitCode == 0 {
        ctx.WorkDir = strings.TrimSpace(stdout)
    }
    
    // Discover OS
    stdout, _, exitCode, _ = e.transport.Run("uname -s")
    if exitCode == 0 {
        ctx.OS = strings.TrimSpace(stdout)
    }
    
    // Discover architecture
    stdout, _, exitCode, _ = e.transport.Run("uname -m")
    if exitCode == 0 {
        ctx.Arch = strings.TrimSpace(stdout)
    }
    
    // Determine transport type
    switch e.transport.(type) {
    case *LocalTransport:
        ctx.Transport = "local"
    case *SSHTransport:
        sshT := e.transport.(*SSHTransport)
        ctx.Transport = fmt.Sprintf("ssh:%s@%s", sshT.User, sshT.Host)
    default:
        ctx.Transport = "unknown"
    }
    
    return ctx
}
```

### Phase 2: Display Context Before Execution

Update CLI to show context prominently:

```go
// main.go executeCommand()
func executeCommand() {
    // ... existing setup ...
    
    // Create executor
    executor := NewExecutor(transport)
    executor.DryRun = dryRun
    
    // Display execution context
    fmt.Println("🔍 Execution Context:")
    fmt.Printf("   Host:      %s\n", executor.context.Host)
    fmt.Printf("   User:      %s\n", executor.context.User)
    fmt.Printf("   Work Dir:  %s\n", executor.context.WorkDir)
    fmt.Printf("   OS/Arch:   %s/%s\n", executor.context.OS, executor.context.Arch)
    fmt.Printf("   Transport: %s\n", executor.context.Transport)
    fmt.Println()
    
    // Confirmation prompt for non-dry-run
    if !dryRun {
        fmt.Printf("⚠️  You are about to execute %d steps on %s as %s.\n", 
            len(selectedPlatform.InstallSteps), 
            executor.context.Host, 
            executor.context.User)
        fmt.Print("   Continue? [yes/no]: ")
        
        var response string
        fmt.Scanln(&response)
        
        if response != "yes" {
            fmt.Println("Aborted.")
            os.Exit(0)
        }
        fmt.Println()
    }
    
    // ... execute ...
}
```

### Phase 3: Execution Guards (Config-Based)

Add guards to config schema and validation:

```go
// types.go
type Config struct {
    Version     string             `json:"version"`
    Description string             `json:"description,omitempty"`
    Guards      *ExecutionGuards   `json:"guards,omitempty"`  // NEW
    Facts       map[string]FactDef `json:"facts,omitempty"`
    // ...
}

type ExecutionGuards struct {
    RequiredHostnamePattern  string            `json:"required_hostname_pattern,omitempty"`
    ForbiddenHostnamePattern string            `json:"forbidden_hostname_pattern,omitempty"`
    RequiredUser             string            `json:"required_user,omitempty"`
    ForbiddenUsers           []string          `json:"forbidden_users,omitempty"`
    RequiredEnvVars          map[string]string `json:"required_env_vars,omitempty"`
    MinDiskSpaceGB           int               `json:"min_disk_space_gb,omitempty"`
    ConfirmPrompt            string            `json:"confirm_prompt,omitempty"`
    AllowRoot                bool              `json:"allow_root"`
}
```

Implement guard checking:

```go
// executor.go
func (e *Executor) checkGuards(guards *ExecutionGuards) error {
    if guards == nil {
        return nil // No guards defined
    }
    
    ctx := e.context
    
    // Check required hostname pattern
    if guards.RequiredHostnamePattern != "" {
        matched, err := regexp.MatchString(guards.RequiredHostnamePattern, ctx.Host)
        if err != nil {
            return fmt.Errorf("invalid hostname pattern: %v", err)
        }
        if !matched {
            return fmt.Errorf("hostname '%s' does not match required pattern '%s'", 
                ctx.Host, guards.RequiredHostnamePattern)
        }
    }
    
    // Check forbidden hostname pattern
    if guards.ForbiddenHostnamePattern != "" {
        matched, err := regexp.MatchString(guards.ForbiddenHostnamePattern, ctx.Host)
        if err != nil {
            return fmt.Errorf("invalid forbidden hostname pattern: %v", err)
        }
        if matched {
            return fmt.Errorf("hostname '%s' matches forbidden pattern '%s' - EXECUTION BLOCKED", 
                ctx.Host, guards.ForbiddenHostnamePattern)
        }
    }
    
    // Check required user
    if guards.RequiredUser != "" && ctx.User != guards.RequiredUser {
        return fmt.Errorf("must run as user '%s', but running as '%s'", 
            guards.RequiredUser, ctx.User)
    }
    
    // Check forbidden users
    for _, forbiddenUser := range guards.ForbiddenUsers {
        if ctx.User == forbiddenUser {
            return fmt.Errorf("cannot run as forbidden user '%s'", forbiddenUser)
        }
    }
    
    // Check root user if not allowed
    if !guards.AllowRoot && ctx.User == "root" {
        return fmt.Errorf("running as root is not allowed (set allow_root: true to override)")
    }
    
    // Check required environment variables
    for envVar, expectedValue := range guards.RequiredEnvVars {
        stdout, _, exitCode, _ := e.transport.Run(fmt.Sprintf("echo $%s", envVar))
        actualValue := strings.TrimSpace(stdout)
        
        if exitCode != 0 || actualValue != expectedValue {
            return fmt.Errorf("environment variable %s must be '%s', but is '%s'", 
                envVar, expectedValue, actualValue)
        }
    }
    
    // Check minimum disk space
    if guards.MinDiskSpaceGB > 0 {
        stdout, _, exitCode, _ := e.transport.Run("df -BG . | tail -1 | awk '{print $4}' | sed 's/G//'")
        if exitCode == 0 {
            availableGB, _ := strconv.Atoi(strings.TrimSpace(stdout))
            if availableGB < guards.MinDiskSpaceGB {
                return fmt.Errorf("insufficient disk space: %dGB available, %dGB required", 
                    availableGB, guards.MinDiskSpaceGB)
            }
        }
    }
    
    return nil
}

func (e *Executor) ExecutePlatform(platform Platform, facts Facts) []StepResult {
    // Check guards BEFORE executing any steps
    if e.config != nil && e.config.Guards != nil {
        if err := e.checkGuards(e.config.Guards); err != nil {
            // Emit failure event
            e.emitEvent(ExecutionEvent{
                Timestamp: time.Now().Format(time.RFC3339),
                RunID:     e.runID,
                StepName:  "Guard Check",
                Status:    "failed",
                Error:     fmt.Sprintf("Execution guards failed: %v", err),
                Context:   e.context,
            })
            
            return []StepResult{{
                StepName: "Guard Check",
                Status:   "failed",
                Error:    err.Error(),
            }}
        }
    }
    
    // Guards passed, proceed with execution
    // ...
}
```

### Phase 4: Enhanced Event Logging

Include context in every event:

```go
func (e *Executor) emitEvent(event ExecutionEvent) {
    // Always include context
    event.Context = e.context
    
    if e.OnEvent != nil {
        e.OnEvent(event)
    }
}
```

Update CLI display to show context when needed:

```go
// main.go
executor.OnEvent = func(event ExecutionEvent) {
    if event.Status == "running" {
        stepNum++
        fmt.Printf("[%d/%d] %s... (%s)\n", 
            stepNum, 
            len(selectedPlatform.InstallSteps), 
            event.StepName,
            event.Context.Host)  // Show host in each step
    }
    // ...
}
```

## Example Config with Guards

```json
{
  "version": "1.0.0",
  "description": "Install Colima on staging environment",
  
  "guards": {
    "required_hostname_pattern": "^staging-.*\\.example\\.com$",
    "forbidden_hostname_pattern": "^prod-.*",
    "required_user": "deploy",
    "forbidden_users": ["root"],
    "required_env_vars": {
      "DEPLOYMENT_ENV": "staging",
      "AWS_REGION": "us-west-2"
    },
    "min_disk_space_gb": 20,
    "confirm_prompt": "Install Colima on {hostname} as {user}?",
    "allow_root": false
  },
  
  "facts": {
    "os": {
      "command": "uname -s",
      "type": "string"
    }
  },
  
  "platforms": [
    {
      "os": "darwin",
      "name": "macOS",
      "install_steps": [
        {
          "name": "Check Homebrew",
          "check": "which brew",
          "error": "Homebrew required"
        }
      ]
    }
  ]
}
```

## Example Execution Output

### Safe Execution (Guards Pass)

```bash
$ ./sink execute install-colima.json --ssh deploy@staging-app1.example.com

🔍 Execution Context:
   Host:      staging-app1.example.com
   User:      deploy
   Work Dir:  /home/deploy
   OS/Arch:   Linux/x86_64
   Transport: ssh:deploy@staging-app1.example.com

✅ Execution guards passed:
   ✓ Hostname matches required pattern: ^staging-.*\.example\.com$
   ✓ User is required user: deploy
   ✓ Environment variable DEPLOYMENT_ENV=staging ✓
   ✓ Environment variable AWS_REGION=us-west-2 ✓
   ✓ Disk space: 45GB available (20GB required) ✓

⚠️  You are about to execute 5 steps on staging-app1.example.com as deploy.
   Continue? [yes/no]: yes

📊 Gathering facts...
   Gathered 3 facts:
   • os = Linux
   • arch = x86_64
   • hostname = staging-app1.example.com

🖥️  Platform: Linux (linux)
📝 Steps: 5

[1/5] Check Docker... (staging-app1.example.com)
      ✓ Success
[2/5] Install Colima... (staging-app1.example.com)
      ✓ Success
...
```

### Blocked Execution (Guards Fail)

```bash
$ ./sink execute install-colima.json --ssh root@prod-db1.example.com

🔍 Execution Context:
   Host:      prod-db1.example.com
   User:      root
   Work Dir:  /root
   OS/Arch:   Linux/x86_64
   Transport: ssh:root@prod-db1.example.com

❌ Execution guards FAILED:
   ✗ Hostname 'prod-db1.example.com' matches forbidden pattern: ^prod-.*
   ✗ User 'root' is in forbidden users list
   ✗ User 'root' not allowed (allow_root is false)

🛑 EXECUTION BLOCKED - Safety guards prevented execution

Error: hostname 'prod-db1.example.com' matches forbidden pattern '^prod-.*' - EXECUTION BLOCKED
```

## Benefits

### 1. Safety
- ✅ Never accidentally run on production
- ✅ Enforce user requirements (no root)
- ✅ Verify environment variables before execution
- ✅ Check prerequisites (disk space, etc.)

### 2. Observability
- ✅ Always know WHERE commands are running
- ✅ Context in every event (host, user, transport)
- ✅ Audit trail of execution location
- ✅ Easy to trace which host had issues

### 3. Confidence
- ✅ Pre-flight checks before any changes
- ✅ Confirmation prompt with full context
- ✅ Guards prevent mistakes
- ✅ Clear feedback on why guards failed

### 4. Debugging
- ✅ Context captured at start of run
- ✅ Know exact environment state
- ✅ Compare context between runs
- ✅ Track environment drift

## Implementation Effort

| Feature | LOC | Effort | Priority |
|---------|-----|--------|----------|
| Context Discovery | ~80 | 1-2h | High |
| Context Display | ~20 | 30min | High |
| Confirmation Prompt | ~15 | 30min | High |
| Guards Schema | ~30 | 1h | Medium |
| Guards Checking | ~120 | 2-3h | Medium |
| Context in Events | ~10 | 30min | Medium |
| **Total** | **~275 LOC** | **5-7h** | - |

## Recommended Phased Approach

### Phase 1 (High Priority, 2-3 hours)
1. ✅ Add ExecutionContext to Executor
2. ✅ Implement discoverContext()
3. ✅ Display context before execution
4. ✅ Add confirmation prompt

**Result:** Immediate visibility and safety

### Phase 2 (Medium Priority, 3-4 hours)
5. ✅ Add ExecutionGuards to schema
6. ✅ Implement checkGuards()
7. ✅ Update config validation

**Result:** Config-based safety enforcement

### Phase 3 (Low Priority, 1 hour)
8. ✅ Include context in all events
9. ✅ Update CLI display to show context

**Result:** Full audit trail

## Conclusion

This addresses your concern about **"not knowing precisely where commands are being executed"** by:

1. **Discovering context first** - Before any execution
2. **Displaying prominently** - User sees exactly where they are
3. **Requiring confirmation** - No accidental execution
4. **Enforcing guards** - Config-based safety rules
5. **Tracking in events** - Every event knows its context

The implementation is clean, adds ~275 LOC, and provides critical safety for remote execution scenarios.

Would you like me to implement Phase 1 (context discovery and display) first?
