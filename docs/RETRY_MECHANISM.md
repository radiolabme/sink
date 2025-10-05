# General Doneness Mechanism: Retry Until Complete

**Date:** October 4, 2025  
**Context:** Unify wait_for into a general retry mechanism

---

## The Insight

> "Maybe we should just have a general 'doneness' mechanism. In general the doneness of a step is a zero error level from the shell, right? But sometimes there's no shell involved. Maybe an implied done on no error, retry zero times is default and then if you want a retry # or retry until ... maybe a retry conditional is what we need?"

**KEY REALIZATION:**

Every step already has an implicit "done" condition: **exit code 0**

- Command succeeds → Done
- Command fails → Not done

**Current behavior:** Retry 0 times (fail immediately)

**What if:** Make retry configurable?

---

## The Unified Model

### **Every step has doneness:**

```go
type Step interface {
    IsDone() bool  // Has this step completed successfully?
}
```

**For commands:** Exit code 0 = done

**For checks:** Check passes = done

**For wait_for:** Would be: Port open / HTTP 2xx / Command succeeds = done

---

## Proposal: Retry Mechanism on Steps

### **Default: No retry (current behavior)**

```json
{
  "name": "Install package",
  "command": "brew install jq"
}
```

**Behavior:** Run once, fail if error

---

### **With retry count:**

```json
{
  "name": "Install package",
  "command": "brew install jq",
  "retry": 3
}
```

**Behavior:** Try up to 3 times, succeed on first success

---

### **With retry until timeout:**

```json
{
  "name": "Wait for database",
  "command": "nc -z localhost 5432",
  "retry": "until",
  "timeout": "30s"
}
```

**Behavior:** Keep retrying until success or 30s timeout

---

### **With retry conditional:**

```json
{
  "name": "Network operation",
  "command": "curl https://example.com/api",
  "retry": {
    "until": "success",
    "timeout": "30s",
    "interval": "2s"
  }
}
```

---

## This Replaces wait_for!

### **Before (separate wait_for):**

```json
{"wait_for": ":5432", "timeout": "30s"}
{"wait_for": "http://localhost:8080/health", "timeout": "60s"}
{"wait_for": "pg_isready", "timeout": "30s"}
```

---

### **After (retry on command):**

```json
{
  "name": "Wait for database",
  "command": "nc -z localhost 5432",
  "retry": "until",
  "timeout": "30s"
}

{
  "name": "Wait for web app",
  "command": "curl -f http://localhost:8080/health",
  "retry": "until",
  "timeout": "60s"
}

{
  "name": "Wait for database ready",
  "command": "pg_isready",
  "retry": "until",
  "timeout": "30s"
}
```

**Same behavior, more general!**

---

## Retry Semantics

### **Option 1: Simple Integer (Retry N Times)**

```json
{
  "command": "curl https://api.example.com",
  "retry": 3
}
```

**Behavior:**
- Try 1: Fail → Wait 1s
- Try 2: Fail → Wait 1s
- Try 3: Fail → Wait 1s
- Try 4: Fail → Give up

**Total attempts: 4** (initial + 3 retries)

---

### **Option 2: String "until" (Retry Until Success)**

```json
{
  "command": "nc -z localhost 5432",
  "retry": "until",
  "timeout": "30s"
}
```

**Behavior:**
- Keep trying until success OR timeout
- Interval: 1s between attempts (configurable?)

---

### **Option 3: Object (Full Control)**

```json
{
  "command": "curl https://api.example.com",
  "retry": {
    "max_attempts": 5,
    "timeout": "30s",
    "interval": "2s",
    "on": ["connection_refused", "timeout"]
  }
}
```

**But this is getting complex...**

---

## Recommendation: Keep It Simple

### **Two retry modes:**

**1. No retry (default):**
```json
{"command": "brew install jq"}
```
Run once, fail immediately

**2. Retry until success (with timeout):**
```json
{
  "command": "nc -z localhost 5432",
  "retry": "until",
  "timeout": "30s"
}
```
Keep trying until success or timeout

---

## Implementation

### **Add to step types:**

```go
type InstallStep struct {
    Name    string `json:"name"`
    Command string `json:"command"`
    
    // NEW: Retry configuration
    Retry   string `json:"retry,omitempty"`   // "until" or "" (no retry)
    Timeout string `json:"timeout,omitempty"` // "30s", "2m", etc.
}
```

### **Execution logic:**

```go
func (e *Executor) executeCommand(step InstallStep) StepResult {
    if step.Retry == "until" {
        return e.executeWithRetry(step)
    }
    
    // Default: no retry (current behavior)
    return e.executeOnce(step)
}

func (e *Executor) executeOnce(step InstallStep) StepResult {
    stdout, stderr, exitCode, err := e.transport.Run(step.Command)
    
    if err != nil || exitCode != 0 {
        return failure(step.Name, stderr)
    }
    
    return success(step.Name, stdout)
}

func (e *Executor) executeWithRetry(step InstallStep) StepResult {
    // Parse timeout (default 60s)
    timeout, _ := time.ParseDuration(step.Timeout)
    if timeout == 0 {
        timeout = 60 * time.Second
    }
    
    deadline := time.Now().Add(timeout)
    interval := 1 * time.Second
    
    var lastErr string
    
    for time.Now().Before(deadline) {
        stdout, stderr, exitCode, err := e.transport.Run(step.Command)
        
        // Success!
        if err == nil && exitCode == 0 {
            return success(step.Name, stdout)
        }
        
        // Track last error
        if stderr != "" {
            lastErr = stderr
        } else if err != nil {
            lastErr = err.Error()
        } else {
            lastErr = fmt.Sprintf("exit code %d", exitCode)
        }
        
        // Wait and retry
        time.Sleep(interval)
    }
    
    // Timeout
    return StepResult{
        StepName: step.Name,
        Status:   "failed",
        Error:    fmt.Sprintf("Timeout after %s\nLast error: %s", timeout, lastErr),
    }
}
```

**Total: ~50 LOC**

---

## Comparison with wait_for

### **wait_for approach (100 LOC):**

- New `wait_for` field
- Pattern detection (port, HTTP, command)
- 3 checker types
- Special handling for ports/HTTP

**Pros:**
- Concise: `{"wait_for": ":5432"}`
- Smart pattern detection

**Cons:**
- New primitive to learn
- Pattern detection complexity
- Port/HTTP handling built-in (increases LOC)

---

### **Retry approach (50 LOC):**

- Add `retry` field to existing steps
- No pattern detection needed
- Reuse existing command execution

**Pros:**
- ✅ Fewer LOC (50 vs 100)
- ✅ More general (works for ANY command)
- ✅ No new primitives
- ✅ Leverages existing commands (nc, curl, pg_isready)

**Cons:**
- ⚠️ Slightly more verbose: `{"command": "nc -z localhost 5432", "retry": "until"}` vs `{"wait_for": ":5432"}`
- ⚠️ User needs to know shell commands (but that's the design philosophy)

---

## Real-World Examples

### **Example 1: Wait for port**

**wait_for approach:**
```json
{"wait_for": ":5432", "timeout": "30s"}
```

**retry approach:**
```json
{
  "name": "Wait for database",
  "command": "nc -z localhost 5432",
  "retry": "until",
  "timeout": "30s"
}
```

**Difference:** User must know `nc -z` command

---

### **Example 2: Wait for HTTP**

**wait_for approach:**
```json
{"wait_for": "http://localhost:8080/health", "timeout": "60s"}
```

**retry approach:**
```json
{
  "name": "Wait for web app",
  "command": "curl -f http://localhost:8080/health",
  "retry": "until",
  "timeout": "60s"
}
```

**Difference:** User must know `curl -f` command

---

### **Example 3: Database-specific check**

**wait_for approach:**
```json
{"wait_for": "pg_isready -U postgres", "timeout": "30s"}
```

**retry approach:**
```json
{
  "name": "Wait for PostgreSQL",
  "command": "pg_isready -U postgres",
  "retry": "until",
  "timeout": "30s"
}
```

**Identical!** (wait_for would delegate to command anyway)

---

### **Example 4: File exists**

**wait_for approach:**
```json
{"wait_for": "test -f /tmp/ready", "timeout": "5m"}
```

**retry approach:**
```json
{
  "name": "Wait for file",
  "command": "test -f /tmp/ready",
  "retry": "until",
  "timeout": "5m"
}
```

**Identical!**

---

## New Use Cases Unlocked

### **1. Retry network operations:**

```json
{
  "name": "Download file",
  "command": "curl -fSL https://example.com/file.tar.gz -o /tmp/file.tar.gz",
  "retry": "until",
  "timeout": "2m"
}
```

Retries on network errors until success or timeout.

---

### **2. Retry idempotent installs:**

```json
{
  "name": "Install package",
  "command": "apt-get install -y postgresql",
  "retry": "until",
  "timeout": "5m"
}
```

Retries if apt is locked (common in cloud-init).

---

### **3. Wait for file content:**

```json
{
  "name": "Wait for cloud-init",
  "command": "grep -q 'Cloud-init.*finished' /var/log/cloud-init.log",
  "retry": "until",
  "timeout": "5m"
}
```

---

### **4. Wait for multiple conditions:**

```json
{
  "name": "Wait for system ready",
  "command": "test -f /tmp/ready && systemctl is-active postgresql",
  "retry": "until",
  "timeout": "2m"
}
```

Shell composition for complex conditions!

---

## Schema

### **Add retry to install_step:**

```json
{
  "install_step": {
    "properties": {
      "name": {"type": "string"},
      "command": {"type": "string"},
      "retry": {
        "type": "string",
        "enum": ["until"],
        "description": "Retry behavior. 'until' = keep retrying until success or timeout"
      },
      "timeout": {
        "type": "string",
        "pattern": "^[0-9]+(s|m|h)$",
        "default": "60s",
        "description": "Timeout for retry (e.g., '30s', '2m', '5m'). Only used if retry is set."
      }
    }
  }
}
```

---

## The Trade-Off

### **wait_for (100 LOC):**

**Config:**
```json
{"wait_for": ":5432", "timeout": "30s"}
```

**Pros:**
- Very concise
- Pattern detection (smart)
- Beginner-friendly

**Cons:**
- 100 LOC
- New primitive
- Pattern detection complexity
- Limited to port/HTTP/command patterns

---

### **retry (50 LOC):**

**Config:**
```json
{
  "command": "nc -z localhost 5432",
  "retry": "until",
  "timeout": "30s"
}
```

**Pros:**
- 50 LOC (half the code)
- No new primitive
- More general (works for ANY command)
- Unlocks new use cases (network retries, apt retry, etc.)
- Unix philosophy (compose existing tools)

**Cons:**
- Slightly more verbose
- User must know shell commands (nc, curl, test, grep)

---

## The Philosophy Alignment

**Sink's design philosophy:**

> "There's always an escape route for people to drop to the shell if they want to do something truly wild."

**The retry approach doubles down on this:**

- Not trying to abstract away shell commands
- Not building special-case handlers
- Instead: Make ANY command retryable

**This is MORE aligned with the philosophy!**

---

## Recommendation

**Use retry mechanism (50 LOC) instead of wait_for (100 LOC)**

**Why:**

1. **Fewer LOC** - 50 vs 100 (saves budget)
2. **More general** - Works for ANY command, not just port/HTTP/command
3. **No new primitive** - Extends existing command steps
4. **Unix philosophy** - Compose existing tools (nc, curl, test, grep)
5. **More powerful** - Enables retry for network ops, apt locks, etc.

**Cost:**

- Slightly more verbose configs
- User must know shell commands (but that's the design)

**The shell IS the abstraction. We just make it retryable.**

---

## Examples: Full Config

### **Django + PostgreSQL:**

```json
{
  "version": "1.0.0",
  "platforms": [
    {
      "os": "darwin",
      "match": "darwin*",
      "name": "macOS",
      "install_steps": [
        {
          "name": "Install PostgreSQL",
          "check": "brew list postgresql@15",
          "on_missing": [
            {
              "name": "Install via Homebrew",
              "command": "brew install postgresql@15"
            }
          ]
        },
        {
          "name": "Start PostgreSQL",
          "command": "brew services start postgresql@15"
        },
        {
          "name": "Wait for database",
          "command": "nc -z localhost 5432",
          "retry": "until",
          "timeout": "30s"
        },
        {
          "name": "Create database",
          "command": "createdb myapp",
          "retry": "until",
          "timeout": "10s"
        },
        {
          "name": "Run migrations",
          "command": "python manage.py migrate"
        }
      ]
    }
  ]
}
```

---

## Implementation Size

**Add to types.go:**
```go
type InstallStep struct {
    // ... existing fields ...
    Retry   string `json:"retry,omitempty"`   // 1 line
    Timeout string `json:"timeout,omitempty"` // 1 line
}
```

**Add to executor.go:**
```go
func executeWithRetry(step) StepResult {
    // Parse timeout
    // Polling loop
    // Track last error
    // Return success or timeout
}
```
**~40 LOC**

**Modify executeCommand:**
```go
func executeCommand(step) {
    if step.Retry == "until" {
        return executeWithRetry(step)
    }
    return executeOnce(step)
}
```
**~5 LOC**

**Update schema:** ~5 LOC

**Total: ~50 LOC**

---

## The Answer

> "Maybe we should just have a general 'doneness' mechanism."

**YES.** And doneness = exit code 0 (already the case).

> "Retry zero times is default and then if you want a retry # or retry until..."

**YES.** Add `retry: "until"` to make any command retryable.

> "Maybe a retry conditional is what we need?"

**YES, BUT keep it simple:** Just `retry: "until"` + timeout.

**This is better than wait_for because:**
- Fewer LOC (50 vs 100)
- More general (works for ALL commands)
- More Unix-like (compose tools, don't replace them)
- More powerful (retry network ops, apt locks, etc.)

---

## Should I implement retry instead of wait_for?

**This is the better design.**
