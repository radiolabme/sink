# Smart Wait Primitive: Pattern-Based Detection

**Date:** October 4, 2025  
**Context:** Simplify 3 wait primitives into 1 with smart pattern detection

---

## The Core Idea

Instead of:
```json
{"wait_for_port": {"port": 8080, "timeout": 60}}
{"wait_for_http": {"url": "http://localhost/health", "timeout": 60}}
{"wait_for_command": {"command": "pg_isready", "timeout": 60}}
```

**One primitive with smart defaults:**
```json
{"wait_for": ":8080", "timeout": "60s"}
{"wait_for": "http://localhost/health", "timeout": "60s"}
{"wait_for": "pg_isready", "timeout": "60s"}
```

**Pattern detection determines the wait strategy.**

---

## Pattern Detection Rules

### **1. Port Number (`:PORT`)**

**Pattern:** Starts with `:` followed by number
```json
{"wait_for": ":8080"}
{"wait_for": ":5432"}
{"wait_for": ":3000"}
```

**Detection:**
```go
if strings.HasPrefix(target, ":") {
    port := strings.TrimPrefix(target, ":")
    // Wait for localhost:PORT
}
```

**Implementation:** TCP dial to `localhost:PORT`

---

### **2. Host:Port (`HOST:PORT`)**

**Pattern:** Contains `:` and number at end
```json
{"wait_for": "localhost:8080"}
{"wait_for": "db:5432"}
{"wait_for": "192.168.1.10:3306"}
```

**Detection:**
```go
if strings.Contains(target, ":") && !strings.HasPrefix(target, "http") {
    // Parse as HOST:PORT
}
```

**Implementation:** TCP dial to `HOST:PORT`

---

### **3. HTTP/HTTPS URL**

**Pattern:** Starts with `http://` or `https://`
```json
{"wait_for": "http://localhost:8080/health"}
{"wait_for": "https://api.example.com/ready"}
{"wait_for": "http://192.168.1.10/ping"}
```

**Detection:**
```go
if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
    // HTTP GET request
}
```

**Implementation:** HTTP GET, expect 2xx status

---

### **4. IP Address (ping)**

**Pattern:** Matches IP address pattern
```json
{"wait_for": "192.168.1.10"}
{"wait_for": "10.0.0.5"}
{"wait_for": "8.8.8.8"}
```

**Detection:**
```go
if net.ParseIP(target) != nil {
    // Valid IP address
}
```

**Implementation:** ICMP ping or TCP dial to port 80/443

---

### **5. Hostname (DNS + ping)**

**Pattern:** Looks like hostname (contains `.` but not `/`)
```json
{"wait_for": "example.com"}
{"wait_for": "db.internal"}
{"wait_for": "api.service.local"}
```

**Detection:**
```go
if strings.Contains(target, ".") && !strings.Contains(target, "/") {
    // Hostname
}
```

**Implementation:** DNS lookup + TCP dial or ping

---

### **6. File Path**

**Pattern:** Starts with `/` or `./` or `../` or `~`
```json
{"wait_for": "/tmp/app-ready"}
{"wait_for": "/var/log/cloud-init-output.log"}
{"wait_for": "./config/done.txt"}
{"wait_for": "~/setup-complete"}
```

**Detection:**
```go
if strings.HasPrefix(target, "/") || 
   strings.HasPrefix(target, "./") || 
   strings.HasPrefix(target, "../") ||
   strings.HasPrefix(target, "~") {
    // File path
}
```

**Implementation:** Check if file exists with `os.Stat()`

---

### **7. File Content Pattern**

**Pattern:** File path with `@content` suffix
```json
{"wait_for": "/var/log/cloud-init.log@Cloud-init.*finished"}
{"wait_for": "/tmp/status.txt@READY"}
```

**Detection:**
```go
if strings.Contains(target, "@") {
    parts := strings.SplitN(target, "@", 2)
    filePath := parts[0]
    pattern := parts[1]
    // Wait for pattern in file
}
```

**Implementation:** Read file, check if regex matches

---

### **8. Command Success**

**Pattern:** Everything else (command to execute)
```json
{"wait_for": "pg_isready -U postgres"}
{"wait_for": "redis-cli ping"}
{"wait_for": "docker ps"}
{"wait_for": "systemctl is-active postgresql"}
```

**Detection:**
```go
// Default case - execute as command
```

**Implementation:** Run command, wait for exit code 0

---

## Complete Implementation

```go
type WaitForStep struct {
    Target  string  // What to wait for (pattern determines strategy)
    Timeout string  // Duration like "60s", "2m", "1h"
}

func (e *Executor) executeWaitFor(step WaitForStep) StepResult {
    timeout, err := time.ParseDuration(step.Timeout)
    if err != nil {
        timeout = 60 * time.Second  // Default 60 seconds
    }
    
    // Detect pattern and delegate
    strategy := detectWaitStrategy(step.Target)
    
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if strategy.Check(step.Target) {
            return StepResult{
                StepName: step.Name,
                Status:   "success",
                Output:   fmt.Sprintf("Target ready: %s", step.Target),
            }
        }
        time.Sleep(1 * time.Second)
    }
    
    return StepResult{
        StepName: step.Name,
        Status:   "failed",
        Error:    fmt.Sprintf("Target not ready after %s: %s", step.Timeout, step.Target),
    }
}

type WaitStrategy interface {
    Check(target string) bool
}

func detectWaitStrategy(target string) WaitStrategy {
    // 1. Port (starts with :)
    if strings.HasPrefix(target, ":") {
        port := strings.TrimPrefix(target, ":")
        return &PortStrategy{Host: "localhost", Port: port}
    }
    
    // 2. HTTP/HTTPS URL
    if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
        return &HTTPStrategy{URL: target}
    }
    
    // 3. File with content pattern
    if strings.Contains(target, "@") {
        parts := strings.SplitN(target, "@", 2)
        return &FileContentStrategy{Path: parts[0], Pattern: parts[1]}
    }
    
    // 4. File path
    if strings.HasPrefix(target, "/") || 
       strings.HasPrefix(target, "./") || 
       strings.HasPrefix(target, "../") ||
       strings.HasPrefix(target, "~") {
        return &FileExistsStrategy{Path: target}
    }
    
    // 5. Host:Port
    if strings.Contains(target, ":") && !strings.Contains(target, " ") {
        parts := strings.Split(target, ":")
        if len(parts) == 2 {
            return &PortStrategy{Host: parts[0], Port: parts[1]}
        }
    }
    
    // 6. IP address
    if net.ParseIP(target) != nil {
        return &PingStrategy{Host: target}
    }
    
    // 7. Hostname (contains . but not /)
    if strings.Contains(target, ".") && !strings.Contains(target, "/") {
        return &HostnameStrategy{Host: target}
    }
    
    // 8. Default: command
    return &CommandStrategy{Command: target}
}

// Strategy implementations
type PortStrategy struct {
    Host string
    Port string
}

func (s *PortStrategy) Check(target string) bool {
    conn, err := net.DialTimeout("tcp", 
        fmt.Sprintf("%s:%s", s.Host, s.Port), 
        1*time.Second)
    if err == nil {
        conn.Close()
        return true
    }
    return false
}

type HTTPStrategy struct {
    URL string
}

func (s *HTTPStrategy) Check(target string) bool {
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(s.URL)
    if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
        resp.Body.Close()
        return true
    }
    if resp != nil {
        resp.Body.Close()
    }
    return false
}

type FileExistsStrategy struct {
    Path string
}

func (s *FileExistsStrategy) Check(target string) bool {
    _, err := os.Stat(s.Path)
    return err == nil
}

type FileContentStrategy struct {
    Path    string
    Pattern string
}

func (s *FileContentStrategy) Check(target string) bool {
    content, err := os.ReadFile(s.Path)
    if err != nil {
        return false
    }
    matched, _ := regexp.MatchString(s.Pattern, string(content))
    return matched
}

type CommandStrategy struct {
    Command string
}

func (s *CommandStrategy) Check(target string) bool {
    _, _, exitCode, _ := e.transport.Run(s.Command)
    return exitCode == 0
}

type PingStrategy struct {
    Host string
}

func (s *PingStrategy) Check(target string) bool {
    // Try TCP dial to common ports (80, 443) as ICMP requires root
    for _, port := range []string{"80", "443"} {
        conn, err := net.DialTimeout("tcp", 
            fmt.Sprintf("%s:%s", s.Host, port), 
            2*time.Second)
        if err == nil {
            conn.Close()
            return true
        }
    }
    return false
}

type HostnameStrategy struct {
    Host string
}

func (s *HostnameStrategy) Check(target string) bool {
    // DNS lookup
    _, err := net.LookupHost(s.Host)
    if err != nil {
        return false
    }
    
    // Then try to connect
    for _, port := range []string{"80", "443"} {
        conn, err := net.DialTimeout("tcp", 
            fmt.Sprintf("%s:%s", s.Host, port), 
            2*time.Second)
        if err == nil {
            conn.Close()
            return true
        }
    }
    return false
}
```

**Total: ~150 LOC** (more than 95 LOC for 3 primitives, but MUCH better UX)

---

## Usage Examples

### **Example 1: Simple Cases**

```json
{
  "install_steps": [
    {"name": "Start PostgreSQL", "command": "brew services start postgresql@15"},
    {"name": "Wait for database", "wait_for": ":5432", "timeout": "30s"},
    
    {"name": "Start Redis", "command": "brew services start redis"},
    {"name": "Wait for Redis", "wait_for": "redis-cli ping", "timeout": "30s"},
    
    {"name": "Start app", "command": "python manage.py runserver &"},
    {"name": "Wait for app", "wait_for": "http://localhost:8000/health", "timeout": "60s"}
  ]
}
```

### **Example 2: Cloud-Init**

```json
{
  "install_steps": [
    {"name": "Create VM", "command": "limactl create --name=dev ubuntu"},
    {"name": "Start VM", "command": "limactl start dev"},
    {"name": "Wait for VM", "wait_for": "192.168.5.15", "timeout": "2m"},
    {"name": "Wait for cloud-init", "wait_for": "/var/log/cloud-init-output.log@Cloud-init.*finished", "timeout": "5m"}
  ]
}
```

### **Example 3: Microservices**

```json
{
  "install_steps": [
    {"name": "Start gateway", "command": "docker run -d -p 8080:8080 gateway"},
    {"name": "Wait", "wait_for": ":8080", "timeout": "30s"},
    
    {"name": "Start auth", "command": "docker run -d -p 9000:9000 auth"},
    {"name": "Wait", "wait_for": "http://localhost:9000/health", "timeout": "30s"},
    
    {"name": "Start users", "command": "docker run -d -p 9001:9001 users"},
    {"name": "Wait", "wait_for": ":9001", "timeout": "30s"}
  ]
}
```

### **Example 4: File-Based Coordination**

```json
{
  "install_steps": [
    {"name": "Start init script", "command": "./initialize.sh &"},
    {"name": "Wait for config", "wait_for": "/tmp/config-ready", "timeout": "2m"},
    
    {"name": "Start app", "command": "./app.sh &"},
    {"name": "Wait for app", "wait_for": "/tmp/app.pid", "timeout": "30s"}
  ]
}
```

---

## Advanced: Process Monitoring

> "The notion about strace or ps is that we could see if a command is in a death spiral or hung."

**Add optional monitoring:**

```json
{
  "name": "Start long-running task",
  "command": "npm install",
  "monitor": {
    "timeout": "10m",
    "check_interval": "10s",
    "hung_detection": {
      "no_output_for": "2m",
      "same_state_for": "5m"
    }
  }
}
```

**Implementation approach:**
```go
// Monitor command execution
// - Track stdout/stderr activity (last output time)
// - Track CPU usage (is it at 0%? hung)
// - Track process state (D state = uninterruptible sleep = hung)
// - If hung_detection triggers, kill with SIGKILL

// Simple version:
type Monitor struct {
    Timeout       time.Duration
    NoOutputFor   time.Duration  // Kill if no output for this long
}

func (e *Executor) executeWithMonitor(cmd string, mon Monitor) StepResult {
    // Start command
    // Watch for output
    // If no output for NoOutputFor duration, kill
}
```

**But this is complex (~200 LOC) and platform-specific.**

**Better: Keep it simple, add later if needed.**

---

## Comparison

### **Before (3 separate primitives):**

```json
{
  "wait_for_port": {"host": "localhost", "port": 5432, "timeout": 60}
}
{
  "wait_for_http": {"url": "http://localhost:8000/health", "status": 200, "timeout": 60}
}
{
  "wait_for_command": {"command": "pg_isready", "timeout": 60}
}
```

**Pros:** Explicit
**Cons:** Verbose, lots of JSON, harder to read

---

### **After (1 smart primitive):**

```json
{"wait_for": ":5432", "timeout": "60s"}
{"wait_for": "http://localhost:8000/health", "timeout": "60s"}
{"wait_for": "pg_isready", "timeout": "60s"}
```

**Pros:** 
- ✅ Concise and readable
- ✅ Pattern-based (intuitive)
- ✅ One primitive to learn
- ✅ Flexible (8 strategies from 1 primitive)

**Cons:**
- ⚠️ Pattern detection could be ambiguous (edge cases)
- ⚠️ More implementation complexity (150 LOC vs 95 LOC)

---

## Edge Cases to Handle

### **1. Ambiguous Patterns**

```json
// Is this a hostname or command?
{"wait_for": "postgres"}

// Resolution: Check if it looks like a command (has spaces or special chars)
// "postgres" → Try as hostname first, fall back to command
// "postgres -D /data" → Clearly a command
```

### **2. IPv6 Addresses**

```json
{"wait_for": "::1"}
{"wait_for": "[::1]:8080"}
```

### **3. Windows Paths**

```json
{"wait_for": "C:\\temp\\ready.txt"}
```

### **4. Commands with Colons**

```json
// Could be confused with host:port
{"wait_for": "docker ps --filter status=running"}
// Resolution: If contains spaces, it's a command
```

---

## Schema Addition

```json
{
  "install_step": {
    "oneOf": [
      {
        "description": "Wait for target to become ready",
        "required": ["name", "wait_for"],
        "properties": {
          "name": {"type": "string"},
          "wait_for": {
            "type": "string",
            "description": "Target to wait for. Pattern determines strategy: :PORT (port), http://... (HTTP), /path (file), command (command success)"
          },
          "timeout": {
            "type": "string",
            "default": "60s",
            "pattern": "^[0-9]+(s|m|h)$",
            "description": "Timeout duration (e.g., '30s', '2m', '1h')"
          },
          "error": {
            "type": "string",
            "description": "Custom error message if timeout"
          }
        }
      }
      // ... existing step types ...
    ]
  }
}
```

---

## The Trade-Off

### **Smart Pattern Detection:**

**Pros:**
- ✅ Extremely concise (`wait_for: ":8080"`)
- ✅ Intuitive (looks like what you're waiting for)
- ✅ One primitive for 8 strategies
- ✅ Extensible (add new patterns without breaking existing)

**Cons:**
- ⚠️ 150 LOC (vs 95 LOC for 3 separate primitives)
- ⚠️ Pattern ambiguity possible (edge cases)
- ⚠️ Magic (not explicit about strategy)

### **Separate Primitives:**

**Pros:**
- ✅ Explicit (clear what strategy is used)
- ✅ Slightly less code (95 LOC)

**Cons:**
- ❌ Verbose JSON
- ❌ 3 primitives to learn
- ❌ Repetitive configs

---

## Recommendation

**Build the smart `wait_for` primitive (150 LOC).**

**Why:**
1. **Better UX** - Configs are 50% shorter and more readable
2. **More flexible** - 8 strategies from 1 primitive
3. **Intuitive** - Pattern matches what you're actually waiting for
4. **Extensible** - Can add new patterns (e.g., Unix socket, gRPC)

**The 55 LOC difference is worth it for the UX improvement.**

---

## Process Monitoring (Future)

> "The notion about strace or ps is that we could see if a command is in a death spiral or hung."

**Don't build this now.** Here's why:

1. **Complex** - Need to track CPU, I/O, state, output
2. **Platform-specific** - `/proc` on Linux, `ps` variations, Windows different
3. **Edge cases** - What's "hung"? npm install is slow but not hung
4. **Better solutions exist** - `timeout` command, systemd watchdog

**If needed later, add as:**
```json
{
  "command": "npm install",
  "timeout": "10m",
  "monitor": {
    "kill_if_no_output_for": "5m"
  }
}
```

**But validate first - do people actually need this?**

---

## Next Steps

**Implement smart `wait_for` primitive (150 LOC):**

1. Pattern detection (8 strategies)
2. Polling with timeout
3. Good error messages
4. Schema update

**Then build 5 configs that use it:**
1. Django + PostgreSQL + Redis
2. Microservices (3 services)
3. Cloud-init VM setup
4. MCP server
5. Colima + Docker

**Ship and validate.**

**Should I implement the smart `wait_for` primitive?**
