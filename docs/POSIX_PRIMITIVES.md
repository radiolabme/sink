# POSIX Primitives: A Different Approach

**Date:** October 4, 2025  
**Context:** What if Sink exposed job control and process inspection instead of managing processes?

---

## The Core Idea

Instead of building process management INTO Sink:

```json
// DON'T DO THIS (reimplementing systemd)
{
  "name": "Start server",
  "background": true,
  "restart_on_crash": true,
  "healthcheck": "curl localhost:8080"
}
```

**Expose POSIX primitives for users to compose:**

```json
// Use shell job control
{
  "name": "Start server in background",
  "command": "mcp-server >server.log 2>&1 & echo $! > server.pid"
}

// Check if it's running
{
  "name": "Verify server started",
  "command": "kill -0 $(cat server.pid)",
  "error": "Server failed to start"
}
```

**The question: Can we make POSIX job control EASIER without managing it ourselves?**

---

## POSIX Primitives Analysis

### **1. Job Control: `&`, `fg`, `bg`, `jobs`**

**What they do:**
```bash
# Start in background
command &

# List background jobs
jobs

# Bring to foreground
fg %1

# Send to background
bg %1

# Disown (detach from shell)
disown %1
```

**How Sink could expose this:**

**Option A: Just use shell job control (current approach)**
```json
{
  "name": "Start server",
  "command": "mcp-server & echo $! > /tmp/mcp.pid"
}
```

**Option B: Job control step type**
```json
{
  "name": "Start server",
  "job": {
    "command": "mcp-server",
    "background": true,
    "save_pid": "/tmp/mcp.pid",
    "disown": true
  }
}
```

**Option C: Job control facts (query job state)**
```json
{
  "facts": {
    "mcp_server_running": {
      "command": "jobs -l | grep -q mcp-server",
      "type": "boolean"
    }
  }
}
```

**Pros of exposing job control:**
- ✅ POSIX standard (works everywhere)
- ✅ Shell already handles it
- ✅ No Sink code needed (Option A)
- ✅ ~50 LOC for Option B (wrapper)

**Cons:**
- ❌ Jobs only live as long as shell session
- ❌ No persistence across reboots
- ❌ No crash recovery
- ❌ Logs still go to stdout/stderr

**Verdict: Useful for temporary background tasks, not services.**

---

### **2. Process Inspection: `ps`, `pgrep`, `pidof`**

**What they do:**
```bash
# Find process by name
pgrep mcp-server

# Find process by PID file
cat /var/run/myservice.pid

# Check if process exists
kill -0 $PID

# Get process info
ps -p $PID -o pid,ppid,comm,state
```

**How Sink could expose this:**

**Option A: Just use commands (current approach)**
```json
{
  "name": "Check if running",
  "check": "pgrep -q mcp-server",
  "error": "MCP server not running"
}
```

**Option B: Process check step type**
```json
{
  "name": "Check if running",
  "check_process": {
    "name": "mcp-server",
    "pid_file": "/tmp/mcp.pid",
    "error": "MCP server not running"
  }
}
```

**Option C: Process facts**
```json
{
  "facts": {
    "mcp_pid": {
      "command": "pgrep mcp-server | head -1",
      "type": "integer",
      "required": false
    },
    "mcp_running": {
      "command": "pgrep -q mcp-server && echo true || echo false",
      "type": "boolean"
    }
  }
}
```

**Pros:**
- ✅ POSIX standard
- ✅ Works everywhere
- ✅ No Sink code needed (Option A)
- ✅ ~30 LOC for Option B

**Cons:**
- ❌ `pgrep` name matching is imprecise (multiple matches)
- ❌ PID files can be stale
- ❌ No information about WHY process died

**Verdict: Useful for checking if something is running, not WHY it's not.**

---

### **3. Process Signals: `kill`, `killall`, `pkill`**

**What they do:**
```bash
# Send signal to PID
kill -TERM $PID
kill -KILL $PID
kill -0 $PID  # Check if exists (no signal sent)

# Send signal by name
pkill -TERM mcp-server

# Send signal to all processes
killall mcp-server
```

**How Sink could expose this:**

**Option A: Just use commands (current approach)**
```json
{
  "name": "Stop server gracefully",
  "command": "kill -TERM $(cat /tmp/mcp.pid) && sleep 5 && kill -0 $(cat /tmp/mcp.pid) && kill -KILL $(cat /tmp/mcp.pid) || true"
}
```

**Option B: Signal step type**
```json
{
  "name": "Stop server gracefully",
  "signal": {
    "pid_file": "/tmp/mcp.pid",
    "signal": "TERM",
    "wait": 5,
    "force_signal": "KILL"
  }
}
```

**Option C: Signal facts (check if alive)**
```json
{
  "facts": {
    "process_alive": {
      "command": "kill -0 $(cat /tmp/mcp.pid) 2>/dev/null && echo true || echo false",
      "type": "boolean"
    }
  }
}
```

**Pros:**
- ✅ POSIX standard
- ✅ Works everywhere
- ✅ Option B makes graceful shutdown easier (~40 LOC)

**Cons:**
- ❌ PID files can be stale (kill wrong process!)
- ❌ No guarantee process actually stops
- ❌ No cleanup of child processes

**Verdict: Useful for stopping processes, but dangerous if PID is wrong.**

---

### **4. System Calls: `strace`, `dtrace`, `dtruss`**

**What they do:**
```bash
# Linux
strace -e trace=open,read,write command

# macOS
dtruss -t open command

# See what files a process touches
strace -e trace=file command

# See network calls
strace -e trace=network command

# See what process is waiting on
strace -p $PID
```

**How Sink could expose this:**

**Option A: Debugging helper**
```json
{
  "name": "Run with trace",
  "command": "strace -e trace=open,read mycommand",
  "debug": true
}
```

**Option B: Trace facts (what files does install touch?)**
```json
{
  "facts": {
    "files_written": {
      "command": "strace -e trace=open -o /tmp/trace.log brew install colima 2>&1 && grep -o '/[^\"]*' /tmp/trace.log",
      "type": "string"
    }
  }
}
```

**Pros:**
- ✅ Powerful debugging
- ✅ Can see what commands actually do
- ✅ Useful for understanding failures

**Cons:**
- ❌ Platform-specific (strace vs dtruss vs ktrace)
- ❌ Requires root on macOS (SIP restrictions)
- ❌ Verbose output (hard to parse)
- ❌ Slows down execution
- ❌ Not useful for end users (developers only)

**Verdict: Useful for debugging Sink configs, not for normal operation.**

---

### **5. File Descriptors: `lsof`**

**What they do:**
```bash
# Check if port is in use
lsof -i :8080

# What files is process using?
lsof -p $PID

# What processes are using file?
lsof /var/log/myapp.log

# Wait for port to be ready
until lsof -i :8080 >/dev/null 2>&1; do sleep 1; done
```

**How Sink could expose this:**

**Option A: Just use commands (current approach)**
```json
{
  "name": "Wait for server",
  "command": "timeout 60 sh -c 'until lsof -i :8080 >/dev/null 2>&1; do sleep 1; done'",
  "error": "Server did not start within 60 seconds"
}
```

**Option B: Port check step type**
```json
{
  "name": "Wait for server",
  "wait_for_port": {
    "port": 8080,
    "timeout": 60,
    "error": "Server did not start"
  }
}
```

**Option C: Port facts**
```json
{
  "facts": {
    "port_8080_open": {
      "command": "lsof -i :8080 >/dev/null 2>&1 && echo true || echo false",
      "type": "boolean"
    }
  }
}
```

**Pros:**
- ✅ `lsof` is standard on Unix
- ✅ Reliable way to check port status
- ✅ Option B makes waiting cleaner (~30 LOC)

**Cons:**
- ❌ `lsof` not on minimal containers (Alpine)
- ❌ Requires root on some systems
- ❌ Port open ≠ service ready (see earlier discussion)

**Verdict: Useful for port checking, but not always available.**

---

## The POSIX Primitives Pattern

### **What's Actually Useful?**

| Primitive | Use Case | Add to Sink? | LOC | Benefit |
|-----------|----------|-------------|-----|---------|
| **`&` (job control)** | Background tasks | ❌ No | 0 | Just use shell |
| **`pgrep`/`ps`** | Check if running | ✅ Maybe | 30 | Cleaner than shell |
| **`kill` signals** | Graceful shutdown | ✅ Maybe | 40 | Handles TERM → KILL |
| **`lsof` port check** | Wait for port | ✅ Maybe | 30 | Better than sleep |
| **`strace`/`dtruss`** | Debugging | ❌ No | 0 | Too platform-specific |

**Total if we add primitives: ~100 LOC**

---

## Concrete Proposal: "Wait For" Primitives

Instead of full process management, add **waiting primitives**:

### **1. Wait for Port**

```json
{
  "name": "Wait for database",
  "wait_for": {
    "port": 5432,
    "host": "localhost",
    "timeout": 60
  }
}
```

**Implementation:**
```go
type WaitForStep struct {
    Port    int
    Host    string
    Timeout int  // seconds
}

func (e *Executor) executeWaitFor(step WaitForStep) StepResult {
    deadline := time.Now().Add(time.Duration(step.Timeout) * time.Second)
    for time.Now().Before(deadline) {
        conn, err := net.DialTimeout("tcp", 
            fmt.Sprintf("%s:%d", step.Host, step.Port), 
            1*time.Second)
        if err == nil {
            conn.Close()
            return StepResult{Status: "success"}
        }
        time.Sleep(1 * time.Second)
    }
    return StepResult{
        Status: "failed",
        Error: fmt.Sprintf("Port %d not ready after %d seconds", 
            step.Port, step.Timeout),
    }
}
```

**Cost: ~30 LOC**

---

### **2. Wait for Process**

```json
{
  "name": "Wait for server to stop",
  "wait_for_exit": {
    "pid_file": "/tmp/server.pid",
    "timeout": 30
  }
}
```

**Implementation:**
```go
type WaitForExitStep struct {
    PIDFile string
    Timeout int
}

func (e *Executor) executeWaitForExit(step WaitForExitStep) StepResult {
    deadline := time.Now().Add(time.Duration(step.Timeout) * time.Second)
    for time.Now().Before(deadline) {
        pidBytes, err := os.ReadFile(step.PIDFile)
        if err != nil {
            // PID file gone = process exited
            return StepResult{Status: "success"}
        }
        
        pid, _ := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
        _, err = os.FindProcess(pid)
        if err != nil {
            return StepResult{Status: "success"}
        }
        
        time.Sleep(1 * time.Second)
    }
    return StepResult{
        Status: "failed",
        Error: fmt.Sprintf("Process did not exit within %d seconds", step.Timeout),
    }
}
```

**Cost: ~30 LOC**

---

### **3. Graceful Stop**

```json
{
  "name": "Stop server gracefully",
  "stop_process": {
    "pid_file": "/tmp/server.pid",
    "grace_period": 10
  }
}
```

**Implementation:**
```go
type StopProcessStep struct {
    PIDFile     string
    GracePeriod int  // seconds
}

func (e *Executor) executeStopProcess(step StopProcessStep) StepResult {
    // Read PID
    pidBytes, err := os.ReadFile(step.PIDFile)
    if err != nil {
        return StepResult{Status: "success"}  // Already stopped
    }
    
    pid, _ := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
    proc, err := os.FindProcess(pid)
    if err != nil {
        return StepResult{Status: "success"}  // Already stopped
    }
    
    // Send SIGTERM
    proc.Signal(syscall.SIGTERM)
    
    // Wait for grace period
    deadline := time.Now().Add(time.Duration(step.GracePeriod) * time.Second)
    for time.Now().Before(deadline) {
        // Check if still running
        err := proc.Signal(syscall.Signal(0))
        if err != nil {
            return StepResult{Status: "success"}  // Exited
        }
        time.Sleep(1 * time.Second)
    }
    
    // Still running, force kill
    proc.Kill()
    time.Sleep(1 * time.Second)
    
    return StepResult{Status: "success"}
}
```

**Cost: ~40 LOC**

---

## The Trade-Offs

### **Option 1: Do Nothing (Current Approach)**

**What users do:**
```json
{
  "name": "Start server",
  "command": "mcp-server >server.log 2>&1 & echo $! > /tmp/mcp.pid"
},
{
  "name": "Wait for server",
  "command": "timeout 60 sh -c 'until lsof -i :8080; do sleep 1; done'"
},
{
  "name": "Stop server",
  "command": "kill -TERM $(cat /tmp/mcp.pid) && sleep 10 && kill -9 $(cat /tmp/mcp.pid) 2>/dev/null || true"
}
```

**Pros:**
- ✅ 0 LOC in Sink
- ✅ Maximum flexibility
- ✅ Works today

**Cons:**
- ❌ Verbose and error-prone
- ❌ Platform-specific (`lsof` not everywhere)
- ❌ Easy to get wrong (PID file bugs)

---

### **Option 2: Add 3 Primitives (100 LOC)**

**What users do:**
```json
{
  "name": "Start server",
  "command": "mcp-server >server.log 2>&1 & echo $! > /tmp/mcp.pid"
},
{
  "name": "Wait for server",
  "wait_for": {"port": 8080, "timeout": 60}
},
{
  "name": "Stop server",
  "stop_process": {"pid_file": "/tmp/mcp.pid", "grace_period": 10}
}
```

**Pros:**
- ✅ Cleaner configs
- ✅ Less error-prone
- ✅ Still composable (just start with `&`, Sink handles rest)

**Cons:**
- ❌ 100 LOC in Sink
- ❌ Still no persistence/restart/logs
- ❌ PID file bugs still possible

---

### **Option 3: Plugin for Process Management**

**Core Sink:** 0 LOC

**Plugin:** `sink-plugin-processes`

```json
{
  "name": "Start server",
  "plugin": "processes.start",
  "config": {
    "command": "mcp-server",
    "log_file": "server.log",
    "pid_file": "/tmp/mcp.pid"
  }
},
{
  "name": "Wait for server",
  "plugin": "processes.wait_for_port",
  "config": {"port": 8080, "timeout": 60}
},
{
  "name": "Stop server",
  "plugin": "processes.stop",
  "config": {"pid_file": "/tmp/mcp.pid"}
}
```

**Pros:**
- ✅ 0 LOC in core
- ✅ Advanced features in plugin
- ✅ Optional (use only if needed)

**Cons:**
- ❌ Need plugin system first (~100 LOC)
- ❌ More complex for users (install plugin)

---

## Recommendation

### **Short Term (7-Day Validation):**

**Option 1: Do Nothing**

- Use shell commands for job control
- Document patterns in `docs/PATTERNS.md`
- Validate use cases first
- See if primitive needs emerge from usage

**Why:**
- 0 LOC
- Works today
- Don't build features nobody needs
- Users can compose with shell

---

### **Medium Term (After Validation):**

**IF** users complain that wait/stop patterns are too error-prone:

**Add 2 primitives (70 LOC):**

1. **`wait_for_port`** (30 LOC) - Most common need
2. **`stop_process`** (40 LOC) - Graceful shutdown

**Skip:**
- `wait_for_exit` (can use `stop_process`)
- Job control wrappers (shell `&` is fine)
- Process inspection (shell `pgrep` is fine)

**Total: 70 LOC**

---

### **Long Term (If Really Needed):**

**Build plugin system (100 LOC)** + **process management plugin (500 LOC)**

Plugin handles:
- Background process tracking
- Log redirection
- Crash recovery
- PID file management

Core stays < 2000 LOC.

---

## The Answer to Your Question

> "What if we were to do something with lsof, strace, ps, jobs, fg, bg (POSIX)?"

**Good primitives to expose:**
- ✅ **`wait_for_port`** (using `lsof` or Go's `net.Dial`) - 30 LOC
- ✅ **`stop_process`** (using `kill` signals) - 40 LOC

**Not worth exposing:**
- ❌ **Job control** (`&`, `fg`, `bg`) - shell already does this
- ❌ **Process inspection** (`ps`, `pgrep`) - shell already does this
- ❌ **Tracing** (`strace`, `dtruss`) - platform-specific, debugging only

**The key insight:**

You're not trying to MANAGE processes (that's systemd's job).

You're trying to make WAITING and STOPPING cleaner.

**Add 2 small primitives (70 LOC) to make configs less error-prone.**

But only AFTER validating that users actually need this.

---

## Next Steps

1. **Validate use cases first** (7 days) with shell commands
2. **Collect feedback** - Do users struggle with wait/stop patterns?
3. **If yes:** Add `wait_for_port` and `stop_process` (70 LOC)
4. **If no:** Keep it simple, document patterns

**Don't build before validating.**
