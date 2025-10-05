# How Wait-For Actually Works: The Polling Loop

**Date:** October 4, 2025  
**Context:** Explaining the wait mechanism

---

## The Core Question

> "How does it wait for something? How does sink wait for a command to be satisfied?"

**Answer: Poll in a loop until success or timeout.**

---

## The Polling Pattern

```go
func (e *Executor) executeWaitFor(step WaitForStep) StepResult {
    // Parse timeout (default 60s)
    timeout, _ := time.ParseDuration(step.Timeout)
    if timeout == 0 {
        timeout = 60 * time.Second
    }
    
    // Detect what we're waiting for
    checker := detectChecker(step.Target)
    
    // Calculate deadline
    deadline := time.Now().Add(timeout)
    interval := 1 * time.Second  // Check every second
    
    // POLL LOOP
    for time.Now().Before(deadline) {
        if checker.Check() {
            return success("Ready: " + step.Target)
        }
        time.Sleep(interval)  // Wait 1 second, then try again
    }
    
    // Timeout - failed to become ready
    return failure("Timeout waiting for: " + step.Target)
}
```

---

## Example: Waiting for Port

### **Config:**
```json
{
  "name": "Wait for PostgreSQL",
  "wait_for": ":5432",
  "timeout": "30s"
}
```

### **What Happens:**

```
t=0s:  Try connecting to localhost:5432 → Connection refused → Sleep 1s
t=1s:  Try connecting to localhost:5432 → Connection refused → Sleep 1s
t=2s:  Try connecting to localhost:5432 → Connection refused → Sleep 1s
t=3s:  Try connecting to localhost:5432 → Connection refused → Sleep 1s
t=4s:  Try connecting to localhost:5432 → Success! → Return success
```

### **Code:**
```go
type PortChecker struct {
    Host string
    Port string
}

func (c *PortChecker) Check() bool {
    conn, err := net.DialTimeout("tcp", 
        fmt.Sprintf("%s:%s", c.Host, c.Port), 
        1*time.Second)
    
    if err == nil {
        conn.Close()
        return true  // Port is open!
    }
    return false  // Port not ready yet
}
```

**Each check:**
1. Try to open TCP connection
2. If success → return true (stop polling)
3. If failure → return false (keep polling)

---

## Example: Waiting for HTTP

### **Config:**
```json
{
  "name": "Wait for web app",
  "wait_for": "http://localhost:8000/health",
  "timeout": "60s"
}
```

### **What Happens:**

```
t=0s:  GET http://localhost:8000/health → Connection refused → Sleep 1s
t=1s:  GET http://localhost:8000/health → Connection refused → Sleep 1s
t=2s:  GET http://localhost:8000/health → 503 Service Unavailable → Sleep 1s
t=3s:  GET http://localhost:8000/health → 503 Service Unavailable → Sleep 1s
t=4s:  GET http://localhost:8000/health → 200 OK → Return success
```

### **Code:**
```go
type HTTPChecker struct {
    URL string
}

func (c *HTTPChecker) Check() bool {
    client := &http.Client{Timeout: 5 * time.Second}
    
    resp, err := client.Get(c.URL)
    if err != nil {
        return false  // Connection failed, keep polling
    }
    defer resp.Body.Close()
    
    // Accept 2xx and 3xx
    return resp.StatusCode >= 200 && resp.StatusCode < 400
}
```

**Each check:**
1. Try HTTP GET request
2. If 2xx/3xx → return true (stop polling)
3. If error or 4xx/5xx → return false (keep polling)

---

## Example: Waiting for Command

### **Config:**
```json
{
  "name": "Wait for database ready",
  "wait_for": "pg_isready -U postgres",
  "timeout": "30s"
}
```

### **What Happens:**

```
t=0s:  Run: pg_isready -U postgres → Exit code 2 (not ready) → Sleep 1s
t=1s:  Run: pg_isready -U postgres → Exit code 2 (not ready) → Sleep 1s
t=2s:  Run: pg_isready -U postgres → Exit code 2 (not ready) → Sleep 1s
t=3s:  Run: pg_isready -U postgres → Exit code 0 (ready!) → Return success
```

### **Code:**
```go
type CommandChecker struct {
    Command string
}

func (c *CommandChecker) Check() bool {
    _, _, exitCode, err := e.transport.Run(c.Command)
    return err == nil && exitCode == 0
}
```

**Each check:**
1. Execute shell command
2. If exit code 0 → return true (stop polling)
3. If non-zero exit code → return false (keep polling)

---

## Example: File Exists (via Command)

### **Config:**
```json
{
  "name": "Wait for cloud-init done",
  "wait_for": "test -f /var/lib/cloud/instance/boot-finished",
  "timeout": "5m"
}
```

### **What Happens:**

```
t=0s:   Run: test -f /var/lib/cloud/instance/boot-finished → Exit 1 (false) → Sleep 1s
t=1s:   Run: test -f /var/lib/cloud/instance/boot-finished → Exit 1 (false) → Sleep 1s
...
t=180s: Run: test -f /var/lib/cloud/instance/boot-finished → Exit 0 (true!) → Return success
```

**The `test -f` command:**
- Exit 0 if file exists
- Exit 1 if file doesn't exist

So polling keeps running it until the file appears.

---

## Example: File Contains Pattern (via Command)

### **Config:**
```json
{
  "name": "Wait for cloud-init finished",
  "wait_for": "grep -q 'Cloud-init.*finished' /var/log/cloud-init.log",
  "timeout": "5m"
}
```

### **What Happens:**

```
t=0s:   Run: grep -q 'Cloud-init.*finished' /var/log/cloud-init.log → Exit 1 (not found) → Sleep 1s
t=1s:   Run: grep -q 'Cloud-init.*finished' /var/log/cloud-init.log → Exit 1 (not found) → Sleep 1s
...
t=240s: Run: grep -q 'Cloud-init.*finished' /var/log/cloud-init.log → Exit 0 (found!) → Return success
```

**The `grep -q` command:**
- Exit 0 if pattern found in file
- Exit 1 if pattern not found

So polling keeps reading the log file until the pattern appears.

---

## The Full Implementation

```go
// Step type
type WaitForStep struct {
    Name    string `json:"name"`
    Target  string `json:"wait_for"`
    Timeout string `json:"timeout,omitempty"`
}

// Main execution
func (e *Executor) executeWaitFor(step WaitForStep) StepResult {
    // Parse timeout
    timeout, err := time.ParseDuration(step.Timeout)
    if err != nil || timeout == 0 {
        timeout = 60 * time.Second  // Default 60s
    }
    
    // Detect checker strategy
    checker := detectChecker(step.Target, e)
    
    // Polling loop
    deadline := time.Now().Add(timeout)
    pollInterval := 1 * time.Second
    
    for time.Now().Before(deadline) {
        if checker.Check() {
            return StepResult{
                StepName: step.Name,
                Status:   "success",
                Output:   "Ready: " + step.Target,
            }
        }
        
        time.Sleep(pollInterval)
    }
    
    // Timeout
    return StepResult{
        StepName: step.Name,
        Status:   "failed",
        Error:    fmt.Sprintf("Timeout after %s waiting for: %s", step.Timeout, step.Target),
    }
}

// Checker interface
type Checker interface {
    Check() bool  // Returns true if ready, false if not ready yet
}

// Pattern detection
func detectChecker(target string, e *Executor) Checker {
    // HTTP/HTTPS?
    if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
        return &HTTPChecker{URL: target}
    }
    
    // Port?
    if strings.Contains(target, ":") {
        host, port := parseHostPort(target)
        return &PortChecker{Host: host, Port: port}
    }
    
    // Command (default)
    return &CommandChecker{Command: target, Executor: e}
}

// Port checker
type PortChecker struct {
    Host string
    Port string
}

func (c *PortChecker) Check() bool {
    conn, err := net.DialTimeout("tcp", 
        fmt.Sprintf("%s:%s", c.Host, c.Port), 
        1*time.Second)
    if err == nil {
        conn.Close()
        return true
    }
    return false
}

// HTTP checker
type HTTPChecker struct {
    URL string
}

func (c *HTTPChecker) Check() bool {
    client := &http.Client{
        Timeout: 5 * time.Second,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse
        },
    }
    
    resp, err := client.Get(c.URL)
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    
    return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// Command checker
type CommandChecker struct {
    Command  string
    Executor *Executor
}

func (c *CommandChecker) Check() bool {
    _, _, exitCode, err := c.Executor.transport.Run(c.Command)
    return err == nil && exitCode == 0
}
```

---

## Key Design Points

### **1. Polling Interval: 1 Second**

```go
pollInterval := 1 * time.Second
```

**Why 1 second?**
- Fast enough for most use cases
- Not so fast that it hammers the system
- Predictable timeout behavior (60s timeout ≈ 60 checks)

**Could make it configurable later:**
```json
{
  "wait_for": ":5432",
  "timeout": "60s",
  "interval": "2s"  // Check every 2 seconds instead
}
```

But start simple with 1s hardcoded.

---

### **2. Check Timeout: Different Per Strategy**

**Port check:** 1 second TCP timeout
```go
net.DialTimeout("tcp", address, 1*time.Second)
```

**HTTP check:** 5 second request timeout
```go
client := &http.Client{Timeout: 5 * time.Second}
```

**Command check:** Uses existing command runner (no specific timeout)
```go
c.Executor.transport.Run(c.Command)
```

**Why different?**
- Port check: Should be instant (open or closed)
- HTTP check: Might take time to process request
- Command: Depends on command (pg_isready is fast, complex scripts might be slow)

---

### **3. Silent Polling**

**Don't print every failed check:**

```
❌ BAD:
Waiting for :5432... not ready
Waiting for :5432... not ready
Waiting for :5432... not ready
Waiting for :5432... not ready
Waiting for :5432... ready!

✅ GOOD:
Waiting for :5432... (this stays on screen, doesn't spam)
Ready: :5432
```

**Implementation:**
```go
// At start of wait
fmt.Printf("⏳ Waiting for %s (timeout: %s)...\n", step.Target, step.Timeout)

// During polling - no output

// On success
fmt.Printf("✅ Ready: %s\n", step.Target)

// On timeout
fmt.Printf("❌ Timeout waiting for: %s\n", step.Target)
```

---

### **4. Graceful Degradation**

**Port check fails?** 
- Connection refused → Keep polling (expected)
- DNS failure → Keep polling (might resolve later)
- Network unreachable → Keep polling (network might come up)

**HTTP check fails?**
- Connection refused → Keep polling
- 503 Service Unavailable → Keep polling (service starting)
- 404 Not Found → Keep polling (might appear)
- Timeout → Keep polling

**Command check fails?**
- Exit code 1 → Keep polling (not ready yet)
- Exit code 127 (command not found) → Keep polling (might be installed)
- Command crashes → Keep polling

**Only stop on:**
1. Success (checker returns true)
2. Timeout (deadline reached)

**Never:**
- Panic on error
- Stop polling early
- Print scary errors during polling

---

## Visual Example: Complete Flow

### **Config:**
```json
{
  "install_steps": [
    {
      "name": "Start PostgreSQL",
      "command": "brew services start postgresql@15"
    },
    {
      "name": "Wait for database",
      "wait_for": ":5432",
      "timeout": "30s"
    },
    {
      "name": "Run migrations",
      "command": "python manage.py migrate"
    }
  ]
}
```

### **Execution:**

```
$ sink execute django.json

Execution Context:
  Host:         macbook.local
  User:         brian
  Directory:    /Users/brian/project
  OS:           macOS 14.5 (darwin/arm64)
  Platform:     macOS (Homebrew)

Continue? (yes/no): yes

✅ Step 1/3: Start PostgreSQL
   $ brew services start postgresql@15
   ==> Successfully started postgresql@15

⏳ Step 2/3: Wait for database (timeout: 30s)
   Target: :5432
   
   [Internally polling every 1s:]
   t=0s:  TCP dial localhost:5432 → refused
   t=1s:  TCP dial localhost:5432 → refused
   t=2s:  TCP dial localhost:5432 → refused
   t=3s:  TCP dial localhost:5432 → SUCCESS!
   
✅ Ready: :5432 (took 3s)

✅ Step 3/3: Run migrations
   $ python manage.py migrate
   Operations to perform: ...
   Running migrations: ...
```

**User sees:**
- Clean output
- Clear progress
- Time taken (3s in this case)

**User doesn't see:**
- Individual failed checks (spam)
- Connection errors (expected during startup)
- Internal polling details

---

## Error Cases

### **Timeout:**

```
⏳ Step 2/3: Wait for database (timeout: 30s)
   Target: :5432
   
   [30 seconds pass, still not ready]
   
❌ Timeout after 30s waiting for: :5432

Error: Step failed: Wait for database
```

### **Invalid timeout:**

```json
{"wait_for": ":5432", "timeout": "invalid"}
```

**Behavior:** Use default 60s
```go
timeout, err := time.ParseDuration(step.Timeout)
if err != nil || timeout == 0 {
    timeout = 60 * time.Second  // Fallback to default
}
```

### **Empty target:**

```json
{"wait_for": "", "timeout": "30s"}
```

**Behavior:** Command checker will fail immediately (empty command → exit 127)

---

## Comparison with Shell Approaches

### **Shell: Manual polling loop**

```bash
# Wait for port
timeout 30 sh -c 'until nc -z localhost 5432; do sleep 1; done'

# Wait for HTTP
timeout 60 sh -c 'until curl -f http://localhost:8000/health; do sleep 2; done'

# Wait for file
timeout 300 sh -c 'until test -f /tmp/ready; do sleep 1; done'
```

**Problems:**
- Verbose
- Error-prone (timeout vs until syntax)
- Prints errors by default (need `2>/dev/null`)
- Inconsistent (different tools, different syntax)

### **Sink: Built-in polling**

```json
{"wait_for": ":5432", "timeout": "30s"}
{"wait_for": "http://localhost:8000/health", "timeout": "60s"}
{"wait_for": "test -f /tmp/ready", "timeout": "5m"}
```

**Benefits:**
- Concise
- Consistent
- Silent polling (no spam)
- Clear errors

**But still allows shell escape:**
```json
{
  "command": "timeout 30 sh -c 'until nc -z localhost 5432; do sleep 1; done'"
}
```

---

## The 100 LOC Breakdown

**Polling loop + timeout:** 25 LOC
```go
func executeWaitFor(...)
```

**Pattern detection:** 15 LOC
```go
func detectChecker(...)
func parseHostPort(...)
```

**Port checker:** 20 LOC
```go
type PortChecker struct { ... }
func (c *PortChecker) Check() bool { ... }
```

**HTTP checker:** 25 LOC
```go
type HTTPChecker struct { ... }
func (c *HTTPChecker) Check() bool { ... }
```

**Command checker:** 15 LOC
```go
type CommandChecker struct { ... }
func (c *CommandChecker) Check() bool { ... }
```

**Total: ~100 LOC**

---

## Summary

**How does it wait?**

1. **Parse timeout** from config (default 60s)
2. **Detect pattern** (HTTP, Port, or Command)
3. **Poll in loop:**
   - Call `Check()` on the checker
   - If true → success, return
   - If false → sleep 1s, try again
4. **Timeout** if deadline reached before success

**Why polling?**
- Simple (easy to understand, easy to implement)
- Reliable (doesn't depend on events/notifications)
- Universal (works for any check)
- Predictable (consistent behavior)

**Alternative approaches considered:**
- Event-driven (too complex, not all systems support events)
- Exponential backoff (unnecessary complexity for typical waits)
- Adaptive polling (over-engineering)

**Simple polling wins: Check every second until ready or timeout.**

---

Does this answer your question about the wait mechanism?
