# The Hard Problems: Time, State, and Process Management

**Date:** October 4, 2025  
**Context:** Deep analysis of error retry, timeouts, state, background processes

---

## The Core Insight

> "Time is difficult to work with because there's no real baseline to work from. Knowing if something is hung, slow, nearly done, stuck, is hard."

**This is the distributed systems problem.** These aren't simple features - they're:
- Non-deterministic (network speed varies)
- Context-dependent (what's "slow" on laptop vs datacenter?)
- State-dependent (is it stuck or just slow?)
- Platform-dependent (signals work differently)

**These are the problems that killed Boxen, Otto, Chef Solo.**

---

## Problem 1: Error Retry

### **What Seems Simple**

```json
{
  "name": "Download file",
  "command": "curl -o file.tar.gz https://example.com/file.tar.gz",
  "retry": {
    "max_attempts": 3,
    "backoff": "exponential"
  }
}
```

### **What's Actually Hard**

**Question 1: Which errors should retry?**
```
Exit code 1 - Generic error (retry?)
Exit code 2 - Misuse of shell (don't retry!)
Exit code 6 - curl: couldn't resolve host (retry? DNS might be down temporarily)
Exit code 7 - curl: failed to connect (retry? Network might recover)
Exit code 22 - curl: HTTP 404 (DON'T retry - file doesn't exist)
Exit code 28 - curl: timeout (retry!)
Exit code 56 - curl: connection reset (retry!)
Exit code 130 - User hit Ctrl-C (DON'T retry!)
```

**You need to know what every tool's exit codes mean.** This is tool-specific knowledge.

**Question 2: What about partial downloads?**
```bash
# First attempt downloads 500MB, fails
# Second attempt: restart from 0 or resume?
curl -C - ... # Resume from where it failed
# But not all servers support this
# And not all tools have this flag
```

**Question 3: Is the operation idempotent?**
```bash
# Safe to retry (idempotent)
curl -o file.tar.gz ...
brew install colima

# NOT safe to retry (side effects)
mkdir /tmp/workdir  # Fails second time: already exists
echo "data" >> file.txt  # Appends twice!
sudo systemctl enable service  # Maybe safe? Depends on service
```

**Question 4: How long to wait between retries?**
```
Linear backoff: 1s, 2s, 3s (predictable, but may hit rate limit)
Exponential: 1s, 2s, 4s, 8s (standard, but gets slow fast)
Exponential with jitter: 1s, 1.8s, 3.7s, 7.2s (prevents thundering herd)

But what's the right initial delay?
- Network hiccup: 1 second is fine
- DNS propagation: might need 60 seconds
- GitHub Actions queue: might need 5 minutes
- How do you know?
```

**Question 5: What about cascading failures?**
```bash
# Step 1: Download fails (retries 3x, succeeds)
# Step 2: Extract fails (retries 3x, fails)
# Step 3: Install fails (never runs)

# Do you retry the whole sequence?
# Or just the failed step?
# What if Step 1 is expensive (10 minute download)?
```

### **Real-World Example: npm install**

```bash
$ npm install
# Fails with ETIMEDOUT

# Retry #1
$ npm install
# Fails with ECONNRESET

# Retry #2
$ npm install
# Fails with E404 (package doesn't exist!)
# Should we have retried this?

# Retry #3
$ npm install
# Succeeds! (network was flaky)
```

**How do you distinguish transient errors from permanent errors?**

### **The Complexity Trap**

To do retry "right", you need:

1. **Exit code database** for every tool
2. **Idempotency detection** (static analysis? annotations?)
3. **Backoff strategies** (linear, exponential, jitter)
4. **Retry budgets** (total time limit)
5. **Error classification** (transient vs permanent)
6. **Partial operation handling** (resume support)
7. **State tracking** (what was already done?)

**This is 500+ LOC and endless edge cases.**

### **What Actually Works**

**Let users handle it:**

```json
{
  "name": "Download with retry",
  "command": "for i in 1 2 3; do curl -fsSL -o file.tar.gz https://... && break || sleep 2; done"
}
```

Or:

```json
{
  "name": "Download",
  "command": "curl --retry 3 --retry-delay 2 --retry-all-errors -o file.tar.gz https://..."
}
```

**curl already has retry logic.** So does wget. So does pip. So does npm (with `--retry`).

**Don't reimplement what tools already have.**

---

## Problem 2: Sleep/Wait

### **What Seems Simple**

```json
{
  "name": "Wait for service",
  "sleep": 5
}
```

### **What's Actually Hard**

**Question 1: How long to wait?**
```
VM startup: 2 seconds? 5 seconds? 30 seconds?
- Depends on: CPU speed, disk speed, RAM, workload
- 2015 laptop: 30 seconds
- 2024 laptop: 2 seconds
- How do you know?

Service startup: 1 second? 10 seconds? 60 seconds?
- PostgreSQL: usually 2-3 seconds
- Elasticsearch: 10-30 seconds
- Kafka: 30-60 seconds
- ML model loading: 5 minutes
- How do you know?
```

**Question 2: What if it takes longer?**
```
Wait 5 seconds → Service not ready yet → Proceed anyway → Fail

Better: Poll until ready
But polling has its own problems (see below)
```

**Question 3: What about spurious readiness?**
```bash
# Service says "ready" but isn't actually ready
systemctl status myservice  # Shows "active"
curl localhost:8080  # Connection refused! (still starting)

# Service port is open but not serving yet
nc -z localhost 8080  # Success!
curl localhost:8080/health  # 500 Internal Server Error (still initializing)
```

### **Real-World Example: Docker/Colima**

```bash
$ colima start
# How long until ready?
# - macOS 2015: ~45 seconds
# - macOS 2024: ~5 seconds
# - Linux with KVM: ~2 seconds
# - Linux without KVM: ~30 seconds

$ colima status
# Status: Running
# But is Docker ready?

$ docker ps
# Error: Cannot connect to daemon
# Colima is running but Docker socket not ready yet!

# Wait how long?
sleep 5  # Maybe enough? Maybe not?

# Better: Poll
while ! docker ps &>/dev/null; do sleep 1; done
# But how long to poll before giving up?
```

### **The Complexity Trap**

To do waiting "right", you need:

1. **Platform-specific baselines** (how long on this hardware?)
2. **Service-specific knowledge** (PostgreSQL vs Elasticsearch)
3. **Health checking** (port open ≠ service ready)
4. **Timeout handling** (give up after X seconds)
5. **Progress indication** (still waiting... 5s, 10s, 15s...)
6. **Readiness protocols** (HTTP /health, TCP connect, file exists, process running)

**This is 300+ LOC per service type.**

### **What Actually Works**

**Poll with explicit timeout:**

```json
{
  "name": "Wait for Docker",
  "command": "timeout 60 sh -c 'until docker ps &>/dev/null; do sleep 1; done'",
  "error": "Docker daemon failed to start within 60 seconds"
}
```

**Or use service's built-in wait:**

```bash
# Docker Compose has built-in waits
docker-compose up --wait

# systemd has built-in waits
systemctl start myservice --wait

# Many tools have --wait flags
# Use them instead of building your own
```

---

## Problem 3: Timeout

### **What Seems Simple**

```json
{
  "name": "Download",
  "command": "curl ...",
  "timeout": 300
}
```

### **What's Actually Hard**

**Question 1: Timeout for what?**
```
Total runtime: 5 minutes max
Connection timeout: 10 seconds to connect
Read timeout: 30 seconds between bytes
Write timeout: 30 seconds to send data

Which one do you mean?
curl has different flags for each!
```

**Question 2: What happens on timeout?**
```bash
# Kill the process (SIGTERM)
# Wait 5 seconds
# Still running? SIGKILL

# But what about child processes?
curl spawns subprocesses → Do you kill them too?
Need to kill the whole process group

# And what about cleanup?
curl was downloading to temp file → Delete the partial file?
curl was writing to database → Roll back the transaction?
```

**Question 3: How to implement across platforms?**
```bash
# macOS/Linux
timeout 300 curl ...

# But timeout command not on all systems
# And behavior differs (GNU timeout vs BSD timeout)

# Go implementation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
cmd := exec.CommandContext(ctx, "curl", "...")
# This is the "right" way but requires Go code
```

**Question 4: Is timeout per-step or total?**
```json
{
  "install_steps": [
    {"name": "Download", "command": "curl ...", "timeout": 300},
    {"name": "Extract", "command": "tar xzf ...", "timeout": 60},
    {"name": "Install", "command": "make install", "timeout": 600}
  ]
}

// Total max time: 960 seconds
// But what if you want total budget of 600 seconds across all steps?
```

### **Real-World Example: Homebrew**

```bash
$ brew install llvm
# Takes 30 minutes (compiling from source)
# What timeout do you set?

$ brew install wget
# Takes 10 seconds (binary download)
# Same timeout as llvm?

# Timeout needs to be per-package
# But how do you know before running?
```

### **The Complexity Trap**

To do timeout "right", you need:

1. **Different timeout types** (connect, read, total)
2. **Platform-specific killing** (SIGTERM vs SIGKILL, process groups)
3. **Cleanup handling** (partial files, database state)
4. **Adaptive timeouts** (slow network? increase timeout)
5. **Per-step vs total budgets**

**This is 200+ LOC and tricky edge cases (zombie processes, signal handling).**

### **What Actually Works**

**Use tool's built-in timeouts:**

```bash
# curl has timeouts
curl --connect-timeout 10 --max-time 300 ...

# wget has timeouts
wget --timeout=300 ...

# ssh has timeouts
ssh -o ConnectTimeout=10 ...
```

**Or wrap with timeout command:**

```json
{
  "name": "Download",
  "command": "timeout 300 curl ..."
}
```

---

## Problem 4: State Persistence

### **What Seems Simple**

```json
{
  "state": {
    "colima_ram": "8GB",
    "user_api_key": "sk-..."
  }
}
```

### **What's Actually Hard**

**Question 1: Where to store state?**
```
~/.sink/state.json  # Per-user
./.sink/state.json  # Per-project
/var/lib/sink/state.json  # System-wide

Which one? All three?
How do they interact?
What if they conflict?
```

**Question 2: When to persist?**
```
After each step? (slow, but safe)
At end of execution? (fast, but lose state if crash)
On-demand when state changes? (complex to detect)
```

**Question 3: How to merge state?**
```json
// Existing state
{"colima_ram": "4GB", "user_name": "brian"}

// New execution sets
{"colima_ram": "8GB", "api_key": "sk-..."}

// Result should be?
{"colima_ram": "8GB", "user_name": "brian", "api_key": "sk-..."}  // Merge?
{"colima_ram": "8GB", "api_key": "sk-..."}  // Replace?
```

**Question 4: What about secrets?**
```json
// DON'T store this in plain text!
{"api_key": "sk-proj-abc123..."}

// Encrypt it? (need encryption key - where to store that?)
// Use system keychain? (platform-specific)
// Environment variables only? (lost on restart)
```

**Question 5: State invalidation**
```bash
# State says: colima_ram = 8GB
# User manually runs: colima delete && colima start --memory 4

# State is now wrong!
# How do you detect this?
# Re-query every time? (slow)
# Trust cache? (gets stale)
```

**Question 6: Concurrent access**
```bash
# Two sink processes running at once
# Process 1: Read state → Modify → Write state
# Process 2: Read state → Modify → Write state
# Result: Process 2 overwrites Process 1's changes

# Need file locking (platform-specific)
# Or atomic operations (complex)
```

### **Real-World Example: Terraform**

Terraform has **terraform.tfstate** and it's a constant source of problems:

```bash
# State gets out of sync
$ terraform plan
Error: state file is locked by another process

# State is stale
$ terraform apply
Error: resource doesn't exist (was deleted outside terraform)

# State is corrupt
$ terraform apply
Error: checksum mismatch, state file corrupt

# Multiple people working
$ terraform apply
Error: state file conflict, run `terraform refresh` first
```

**Terraform spent YEARS getting state right.** And it's still hard.

### **The Complexity Trap**

To do state "right", you need:

1. **Storage location strategy** (user, project, system)
2. **Merge semantics** (replace, merge, conflict resolution)
3. **Encryption/secrets** (secure storage)
4. **Staleness detection** (cache invalidation)
5. **Concurrency control** (locking, atomic ops)
6. **Migration/versioning** (state format changes)
7. **Backup/recovery** (state corruption)

**This is 500+ LOC and distributed systems problems (locks, consistency, durability).**

### **What Actually Works**

**Don't persist state. Use facts instead:**

```json
{
  "facts": {
    "colima_memory": {
      "command": "colima list -j | jq -r '.memory'",
      "type": "string"
    }
  }
}
```

**Query current state every time.** It's slower but correct.

**Or use environment variables for user input:**

```bash
# User sets once
export CLAUDE_API_KEY=sk-...

# Sink uses it
{
  "facts": {
    "api_key": {
      "command": "echo $CLAUDE_API_KEY",
      "required": true
    }
  }
}
```

---

## Problem 5: Background Processes

### **What Seems Simple**

```json
{
  "name": "Start server",
  "command": "mcp-server-filesystem",
  "background": true
}
```

### **What's Actually Hard**

**Question 1: How to track the process?**
```bash
# Start in background
mcp-server-filesystem &

# How do you track it?
# PID? (store where? what if parent dies?)
# PID file? (who writes it? what if crashes before writing?)
# Process name? (multiple instances?)
```

**Question 2: How to know if it's running?**
```bash
# Check PID
kill -0 $PID  # Process exists
# But is it the right process? PID could be reused!

# Check PID file
cat /var/run/myservice.pid
# But file might be stale (process crashed, didn't clean up)

# Check port
lsof -i :8080  # Port in use
# But is it our service or something else?

# Check process name
pgrep -f mcp-server  # Found!
# But which instance? And is it actually our instance?
```

**Question 3: How to stop it later?**
```bash
# Kill by PID
kill $PID
# But PID might be wrong (reused, or child processes)

# Kill by name
pkill -f mcp-server
# But might kill wrong processes!

# Kill by process group
kill -- -$PGID
# Platform-specific, doesn't always work
```

**Question 4: What about logs?**
```bash
# Background process stdout/stderr go where?
mcp-server &  # Output to terminal (but we don't want that)
mcp-server >/dev/null 2>&1 &  # Lost forever
mcp-server >server.log 2>&1 &  # Saved, but how to rotate logs?

# systemd handles this (journald)
# launchd handles this
# But you're reimplementing them
```

**Question 5: What about crashes?**
```bash
# Start background process
mcp-server &

# It crashes 5 seconds later
# How do you know?
# Do you restart it? (respawn)
# How many times? (avoid restart loops)
# With backoff? (exponential, with max attempts)

# This is what systemd's Restart= does
# You're reimplementing systemd
```

**Question 6: What about dependencies?**
```bash
# Start PostgreSQL
postgres &

# Start app server (depends on PostgreSQL)
app-server &

# But PostgreSQL takes 3 seconds to start
# app-server tries to connect immediately → fails

# Need to wait for PostgreSQL
# But how? (polling, signals, readiness checks)

# This is what systemd's After= and Requires= do
# You're reimplementing systemd
```

**Question 7: What about cleanup?**
```bash
# User hits Ctrl-C
# Sink exits
# Background processes keep running (orphaned)

# Need to track children
# Need to handle signals (SIGTERM, SIGINT)
# Need to forward signals to children
# Need to wait for graceful shutdown
# Need to force-kill if timeout

# This is what process managers do
# You're reimplementing supervisord/systemd/pm2
```

### **Real-World Example: Docker Daemon**

```bash
# Start Docker daemon in background
dockerd &

# Problems:
# - Where are logs? (/var/log/docker.log? who rotates?)
# - How to stop? (kill? pkill? docker system stop?)
# - Crashes? (who restarts? how many times?)
# - Depends on containerd (need to start that first)
# - Cleanup? (do you stop containerd too?)

# Solution: Don't do this yourself
# Use systemd: systemctl start docker
# Use launchd: launchctl start com.docker.dockerd
# Use Docker Desktop: open -a Docker
```

### **The Complexity Trap**

To do background processes "right", you need:

1. **Process tracking** (PIDs, PID files, process groups)
2. **Liveness checking** (is it still running? is it the right process?)
3. **Log management** (capture, rotate, search)
4. **Crash recovery** (detect crashes, respawn with backoff)
5. **Dependency management** (wait for dependencies, start order)
6. **Signal handling** (forward signals, graceful shutdown, force-kill)
7. **Cleanup** (orphan prevention, resource cleanup)
8. **Platform differences** (systemd vs launchd vs Windows Services vs nothing)

**This is 1000+ LOC and you're rebuilding systemd/supervisord.**

### **What Actually Works**

**Use the platform's service manager:**

```json
{
  "platforms": [{
    "os": "darwin",
    "install_steps": [
      {
        "name": "Install plist",
        "command": "cat > ~/Library/LaunchAgents/com.example.service.plist << EOF\n...\nEOF"
      },
      {
        "name": "Load service",
        "command": "launchctl load ~/Library/LaunchAgents/com.example.service.plist"
      }
    ]
  }, {
    "os": "linux",
    "install_steps": [
      {
        "name": "Install unit file",
        "command": "sudo cat > /etc/systemd/system/myservice.service << EOF\n...\nEOF"
      },
      {
        "name": "Enable and start",
        "command": "sudo systemctl enable --now myservice"
      }
    ]
  }]
}
```

**Or use a process manager:**

```json
{
  "name": "Start with PM2",
  "command": "pm2 start mcp-server --name mcp"
}
```

**Or use Docker:**

```json
{
  "name": "Run as container",
  "command": "docker run -d --name mcp --restart unless-stopped mcp-server"
}
```

**Don't reimplement process management.**

---

## The Pattern: Distributed Systems Problems

All of these are **distributed systems problems**:

| Problem | Why It's Hard | Distributed Systems Concept |
|---------|---------------|----------------------------|
| **Error retry** | Don't know if error is transient | Retry semantics, idempotency |
| **Sleep/wait** | Don't know how long is needed | Timeouts, deadlines |
| **Timeout** | Don't know what's "too long" | Failure detection, timeouts |
| **State** | Multiple writers, staleness | Consistency, CAP theorem |
| **Background processes** | Failures, crashes, orphans | Process supervision, health checks |

These problems **don't have simple solutions.** They have **complex, context-dependent trade-offs.**

---

## What This Means for Sink

### **DON'T BUILD INTO CORE:**

1. ❌ **Retry logic** - Let tools handle it (curl --retry, npm --retry)
2. ❌ **Sleep/wait** - Use timeout + polling in shell
3. ❌ **Timeout** - Use tool timeouts (curl --max-time) or timeout command
4. ❌ **State persistence** - Use facts (query current state) or env vars
5. ❌ **Background processes** - Use systemd/launchd/pm2/docker

### **BECAUSE:**

- Each is 200-1000 LOC
- Each has platform-specific quirks
- Each has complex edge cases
- Each requires domain knowledge (tool exit codes, service startup times, etc.)
- **Total: 2000-4000 LOC just for these features**

**This violates the core constraint: < 2000 LOC total.**

### **INSTEAD:**

**1. For retry:** Use tool's built-in retry or shell loops
```bash
curl --retry 3 --retry-delay 2 ...
# or
for i in 1 2 3; do curl ... && break || sleep 2; done
```

**2. For waiting:** Use timeout + polling
```bash
timeout 60 sh -c 'until docker ps &>/dev/null; do sleep 1; done'
```

**3. For timeout:** Use tool timeouts or timeout command
```bash
curl --max-time 300 ...
# or
timeout 300 curl ...
```

**4. For state:** Use facts (query current state every time)
```json
{
  "facts": {
    "colima_memory": {"command": "colima list -j | jq -r '.memory'"}
  }
}
```

**5. For background processes:** Use platform services
```bash
# macOS
launchctl load ~/Library/LaunchAgents/com.example.plist

# Linux
systemctl enable --now myservice

# Or PM2
pm2 start app.js

# Or Docker
docker run -d --restart unless-stopped ...
```

---

## The Brutal Truth

> "Process management is hard. Flaky network or system restarts might be hard to deal with."

**YES. It's very hard.**

These are the problems that:
- systemd solves (1M+ LOC, 15 years development)
- supervisord solves (10K+ LOC)
- Docker solves (100K+ LOC)
- Kubernetes solves (1M+ LOC)

**You cannot solve them in < 2000 LOC.**

And you shouldn't try, because **these tools already exist.**

---

## Recommendation

### **Core Sink Strategy:**

**1. Stay thin (< 2000 LOC)**
- Command orchestration
- Platform detection
- Facts gathering
- Interactive prompts (30 LOC)

**2. Delegate complexity:**
- Retry → curl --retry, npm --retry
- Timeout → tool timeouts, timeout command
- State → facts (query each time)
- Background → systemd, launchd, pm2, docker
- Waiting → timeout + shell polling

**3. Document patterns:**
Create docs showing how to handle these with existing tools:
- `docs/PATTERNS_RETRY.md` - How to retry with curl, npm, etc.
- `docs/PATTERNS_SERVICES.md` - How to use systemd/launchd
- `docs/PATTERNS_WAITING.md` - How to poll with timeout

**4. Plugin for advanced cases:**
If users really need it, build plugins:
- `sink-plugin-systemd` - Generate systemd units
- `sink-plugin-retry` - Advanced retry with exit code detection
- `sink-plugin-state` - State persistence (if really needed)

But validate first. Most users probably don't need these.

---

## The Answer

> "These seem like we should think about them pretty hard."

**YES. And the answer is: DON'T BUILD THEM.**

They're too complex for core. They violate the < 2000 LOC constraint. They trap you into distributed systems problems.

**Instead:**
- Document how to use existing tools
- Show patterns and examples
- Let plugins handle advanced cases if needed

**This keeps Sink simple, maintainable, and human-scale.**
