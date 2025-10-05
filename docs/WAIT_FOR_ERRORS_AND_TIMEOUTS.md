# Wait-For Error Handling and Timeout Design

**Date:** October 4, 2025  
**Context:** How to handle errors and represent timeouts in wait_for

---

## The Questions

1. **What do we do if there is an error?**
2. **How do we represent a timeout?**

---

## Question 1: Error Handling During Polling

### **Philosophy: Errors During Polling Are Expected**

When waiting for something to become ready, **errors are normal**:

- Port not open yet → Connection refused (expected)
- HTTP endpoint not ready → Connection refused or 503 (expected)
- Command fails → Exit code != 0 (expected)
- File doesn't exist → test -f returns 1 (expected)

**Key insight: These aren't "errors" - they're "not ready yet"**

---

### **Design: Treat All Errors as "Not Ready Yet"**

```go
func (c *PortChecker) Check() bool {
    conn, err := net.DialTimeout("tcp", address, 1*time.Second)
    if err == nil {
        conn.Close()
        return true  // Ready!
    }
    return false  // Not ready yet (error = expected)
}
```

**During polling:**
- Error → Return false → Sleep 1s → Try again
- Success → Return true → Stop polling

**No special error handling needed. Errors are part of "not ready yet".**

---

### **What About Permanent Errors?**

**Problem:** Some errors might be permanent (config issue, not "not ready yet"):

```json
{"wait_for": "localhost:999999"}  // Invalid port number
{"wait_for": "http://localhost:8080/wrong-path"}  // Wrong endpoint (will never work)
{"wait_for": "command-that-doesnt-exist"}  // Command not found
```

**Question:** Should we detect these and fail fast?

**Answer:** No. Keep it simple. Timeout will catch them.

**Reasoning:**
1. **Hard to detect:** Is 404 permanent or will the endpoint appear? Is "command not found" permanent or will it be installed?
2. **Timeout catches all:** If it's truly broken, timeout will happen
3. **Simpler code:** No heuristics for "this error is permanent"

**Example:**
```json
{"wait_for": "http://localhost:8080/wrong-path", "timeout": "30s"}
```

**Behavior:**
```
t=0s:   GET /wrong-path → 404 → Keep polling
t=1s:   GET /wrong-path → 404 → Keep polling
...
t=30s:  Timeout → Fail with "Timeout waiting for: http://localhost:8080/wrong-path"
```

**User sees clear timeout message. They can debug from there.**

---

### **Should We Show Last Error?**

**Option A: Just show timeout**
```
❌ Timeout after 30s waiting for: http://localhost:8080/health
```

**Option B: Show last error**
```
❌ Timeout after 30s waiting for: http://localhost:8080/health
   Last error: Connection refused
```

**Option C: Show error history**
```
❌ Timeout after 30s waiting for: http://localhost:8080/health
   Attempts: 30
   Errors: Connection refused (25 times), 503 Service Unavailable (5 times)
```

**Recommendation: Option B (show last error)**

**Why:**
- Helps debugging (user knows what went wrong)
- Not too verbose (just last error, not all errors)
- Simple to implement (save last error in checker)

**Implementation:**
```go
type Checker interface {
    Check() bool
    LastError() string  // Returns last error message (or "" if success)
}

func (e *Executor) executeWaitFor(step WaitForStep) StepResult {
    checker := detectChecker(step.Target, e)
    
    // ... polling loop ...
    
    // On timeout:
    lastErr := checker.LastError()
    errMsg := fmt.Sprintf("Timeout after %s waiting for: %s", timeout, step.Target)
    if lastErr != "" {
        errMsg += fmt.Sprintf("\nLast error: %s", lastErr)
    }
    
    return StepResult{
        StepName: step.Name,
        Status:   "failed",
        Error:    errMsg,
    }
}
```

**Example output:**
```
❌ Timeout after 30s waiting for: :5432
   Last error: connection refused
```

```
❌ Timeout after 60s waiting for: http://localhost:8080/health
   Last error: HTTP 503 Service Unavailable
```

```
❌ Timeout after 30s waiting for: pg_isready
   Last error: command exited with code 2
```

---

## Question 2: How to Represent Timeout?

### **Option 1: Duration String (RECOMMENDED)**

```json
{
  "name": "Wait for database",
  "wait_for": ":5432",
  "timeout": "30s"
}
```

**Format:** Go duration format
- `"10s"` = 10 seconds
- `"2m"` = 2 minutes
- `"1h"` = 1 hour
- `"90s"` = 90 seconds (same as "1m30s")

**Pros:**
- ✅ Intuitive (looks like what it means)
- ✅ Go stdlib parses it (`time.ParseDuration`)
- ✅ Flexible (can express any duration)
- ✅ Readable (self-documenting)

**Cons:**
- ⚠️ Must validate format (but schema can help)

**Implementation:**
```go
timeout, err := time.ParseDuration(step.Timeout)
if err != nil {
    return StepResult{
        Status: "failed",
        Error:  fmt.Sprintf("Invalid timeout format: %s (use '30s', '2m', '1h')", step.Timeout),
    }
}
```

---

### **Option 2: Integer Seconds**

```json
{
  "wait_for": ":5432",
  "timeout": 30
}
```

**Pros:**
- ✅ Simple
- ✅ No parsing needed

**Cons:**
- ❌ Not intuitive for long timeouts (300 vs "5m")
- ❌ No support for subsecond (not needed but loses flexibility)
- ❌ Ambiguous unit (seconds? milliseconds?)

---

### **Option 3: Object with Unit**

```json
{
  "wait_for": ":5432",
  "timeout": {"value": 30, "unit": "seconds"}
}
```

**Pros:**
- ✅ Explicit

**Cons:**
- ❌ Verbose
- ❌ Over-engineered

---

### **Recommendation: Duration String ("30s", "2m", "1h")**

**Schema:**
```json
{
  "timeout": {
    "type": "string",
    "default": "60s",
    "pattern": "^[0-9]+(s|m|h|ms|us|ns)$",
    "description": "Timeout duration (e.g., '30s', '2m', '1h')",
    "examples": ["30s", "2m", "1h", "90s"]
  }
}
```

**Why:**
- Matches Go idioms (time.Duration)
- Readable in configs
- Flexible for all use cases

---

## Default Timeout

**Question:** What if timeout is omitted?

```json
{"wait_for": ":5432"}  // No timeout specified
```

**Options:**

**A) No default - require timeout**
- Pro: Explicit, no surprises
- Con: Verbose, annoying

**B) Default to 60s**
- Pro: Reasonable default
- Con: Might be too short for some, too long for others

**C) Default to infinity (no timeout)**
- Pro: Simple
- Con: Dangerous (could hang forever)

**Recommendation: Default to 60s**

**Reasoning:**
- Most waits are < 60s (databases, web servers start quickly)
- Long waits should be explicit (cloud-init needs "5m")
- Prevents hanging forever on mistakes
- Can always override: `"timeout": "5m"`

**Implementation:**
```go
timeout, err := time.ParseDuration(step.Timeout)
if err != nil || timeout == 0 {
    timeout = 60 * time.Second  // Default 60s
}
```

**Schema:**
```json
{
  "timeout": {
    "type": "string",
    "default": "60s",
    "pattern": "^[0-9]+(s|m|h)$"
  }
}
```

---

## Complete Error Handling Design

### **During Polling: Silent**

```go
for time.Now().Before(deadline) {
    if checker.Check() {
        return success()
    }
    // Don't print errors here - they're expected
    time.Sleep(1 * time.Second)
}
```

**Output:**
```
⏳ Waiting for :5432 (timeout: 30s)...
```

**No spam during polling.**

---

### **On Success: Show Duration**

```go
elapsed := time.Since(startTime)
return StepResult{
    Status:  "success",
    Output:  fmt.Sprintf("Ready: %s (took %s)", step.Target, elapsed.Round(time.Second)),
}
```

**Output:**
```
✅ Ready: :5432 (took 3s)
```

---

### **On Timeout: Show Last Error**

```go
lastErr := checker.LastError()
errMsg := fmt.Sprintf("Timeout after %s waiting for: %s", timeout, step.Target)
if lastErr != "" {
    errMsg += fmt.Sprintf("\nLast error: %s", lastErr)
}

return StepResult{
    Status: "failed",
    Error:  errMsg,
}
```

**Output:**
```
❌ Timeout after 30s waiting for: :5432
   Last error: connection refused
```

```
❌ Timeout after 60s waiting for: http://localhost:8080/health
   Last error: HTTP 503 Service Unavailable
```

---

## Implementation with Error Tracking

```go
// Checker interface with error tracking
type Checker interface {
    Check() bool
    LastError() string
}

// Port checker
type PortChecker struct {
    Host     string
    Port     string
    lastErr  string
}

func (c *PortChecker) Check() bool {
    conn, err := net.DialTimeout("tcp", 
        fmt.Sprintf("%s:%s", c.Host, c.Port), 
        1*time.Second)
    
    if err == nil {
        conn.Close()
        c.lastErr = ""
        return true
    }
    
    c.lastErr = err.Error()
    return false
}

func (c *PortChecker) LastError() string {
    return c.lastErr
}

// HTTP checker
type HTTPChecker struct {
    URL     string
    lastErr string
}

func (c *HTTPChecker) Check() bool {
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(c.URL)
    
    if err != nil {
        c.lastErr = err.Error()
        return false
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 200 && resp.StatusCode < 400 {
        c.lastErr = ""
        return true
    }
    
    c.lastErr = fmt.Sprintf("HTTP %d %s", resp.StatusCode, resp.Status)
    return false
}

func (c *HTTPChecker) LastError() string {
    return c.lastErr
}

// Command checker
type CommandChecker struct {
    Command  string
    Executor *Executor
    lastErr  string
}

func (c *CommandChecker) Check() bool {
    _, stderr, exitCode, err := c.Executor.transport.Run(c.Command)
    
    if err != nil {
        c.lastErr = err.Error()
        return false
    }
    
    if exitCode == 0 {
        c.lastErr = ""
        return true
    }
    
    if stderr != "" {
        c.lastErr = fmt.Sprintf("exit code %d: %s", exitCode, strings.TrimSpace(stderr))
    } else {
        c.lastErr = fmt.Sprintf("exit code %d", exitCode)
    }
    return false
}

func (c *CommandChecker) LastError() string {
    return c.lastErr
}

// Main execution with error handling
func (e *Executor) executeWaitFor(step WaitForStep) StepResult {
    // Parse timeout
    timeout, err := time.ParseDuration(step.Timeout)
    if err != nil {
        return StepResult{
            StepName: step.Name,
            Status:   "failed",
            Error:    fmt.Sprintf("Invalid timeout: %s (use '30s', '2m', '1h')", step.Timeout),
        }
    }
    if timeout == 0 {
        timeout = 60 * time.Second  // Default
    }
    
    // Detect checker
    checker := detectChecker(step.Target, e)
    
    // Polling loop
    startTime := time.Now()
    deadline := startTime.Add(timeout)
    
    for time.Now().Before(deadline) {
        if checker.Check() {
            elapsed := time.Since(startTime)
            return StepResult{
                StepName: step.Name,
                Status:   "success",
                Output:   fmt.Sprintf("Ready: %s (took %s)", step.Target, elapsed.Round(time.Second)),
            }
        }
        time.Sleep(1 * time.Second)
    }
    
    // Timeout
    lastErr := checker.LastError()
    errMsg := fmt.Sprintf("Timeout after %s waiting for: %s", timeout, step.Target)
    if lastErr != "" {
        errMsg += "\nLast error: " + lastErr
    }
    
    return StepResult{
        StepName: step.Name,
        Status:   "failed",
        Error:    errMsg,
    }
}
```

---

## Schema Updates

```json
{
  "install_step": {
    "oneOf": [
      {
        "description": "Wait for target to become ready",
        "required": ["name", "wait_for"],
        "properties": {
          "name": {
            "type": "string",
            "description": "Step name"
          },
          "wait_for": {
            "type": "string",
            "description": "Target to wait for. Patterns: ':PORT' or 'HOST:PORT' (TCP), 'http://...' (HTTP GET), anything else (command that exits 0)",
            "examples": [":5432", "localhost:8080", "http://localhost/health", "pg_isready", "test -f /tmp/ready"]
          },
          "timeout": {
            "type": "string",
            "default": "60s",
            "pattern": "^[0-9]+(s|m|h)$",
            "description": "Timeout duration (e.g., '30s', '2m', '1h'). Defaults to 60s if omitted.",
            "examples": ["30s", "2m", "5m", "1h"]
          }
        },
        "additionalProperties": false
      },
      // ... other step types ...
    ]
  }
}
```

---

## Example Outputs

### **Success (Fast)**
```
⏳ Step 2/5: Wait for database
   Target: :5432
   
✅ Ready: :5432 (took 3s)
```

### **Success (Slow)**
```
⏳ Step 3/5: Wait for cloud-init
   Target: test -f /var/lib/cloud/instance/boot-finished
   
✅ Ready: test -f /var/lib/cloud/instance/boot-finished (took 2m15s)
```

### **Timeout (Port)**
```
⏳ Step 2/5: Wait for database
   Target: :5432
   
❌ Timeout after 30s waiting for: :5432
   Last error: dial tcp [::1]:5432: connection refused

Error: Step failed: Wait for database
```

### **Timeout (HTTP)**
```
⏳ Step 3/5: Wait for web app
   Target: http://localhost:8080/health
   
❌ Timeout after 60s waiting for: http://localhost:8080/health
   Last error: HTTP 503 Service Unavailable

Error: Step failed: Wait for web app
```

### **Timeout (Command)**
```
⏳ Step 2/5: Wait for database
   Target: pg_isready -U postgres
   
❌ Timeout after 30s waiting for: pg_isready -U postgres
   Last error: exit code 2

Error: Step failed: Wait for database
```

### **Invalid Timeout**
```
❌ Step 2/5: Wait for database
   Error: Invalid timeout: invalid (use '30s', '2m', '1h')

Error: Step failed: Wait for database
```

---

## Edge Cases

### **Zero Timeout**
```json
{"wait_for": ":5432", "timeout": "0s"}
```

**Behavior:** Use default 60s
```go
if timeout == 0 {
    timeout = 60 * time.Second
}
```

### **Negative Timeout**
```json
{"wait_for": ":5432", "timeout": "-10s"}
```

**Behavior:** Parsing fails, error message
```
❌ Error: Invalid timeout: -10s (use '30s', '2m', '1h')
```

### **Very Short Timeout**
```json
{"wait_for": ":5432", "timeout": "1s"}
```

**Behavior:** Works, but likely to timeout (only 1 check)

### **Very Long Timeout**
```json
{"wait_for": ":5432", "timeout": "24h"}
```

**Behavior:** Works, but waits up to 24 hours (user's choice)

---

## Summary

### **Error Handling:**

1. **During polling:** Errors are "not ready yet" - silent, keep polling
2. **On timeout:** Show clear message with last error
3. **No early failure:** Timeout catches everything (simple)

### **Timeout Representation:**

1. **Format:** Duration string (`"30s"`, `"2m"`, `"1h"`)
2. **Default:** 60s if omitted
3. **Validation:** Schema pattern + Go's `time.ParseDuration`

### **Output:**

**Silent polling:**
```
⏳ Waiting for :5432 (timeout: 30s)...
```

**Success:**
```
✅ Ready: :5432 (took 3s)
```

**Timeout:**
```
❌ Timeout after 30s waiting for: :5432
   Last error: connection refused
```

**Simple, clear, debuggable.**

---

Does this answer both questions? Should I proceed with implementation?
