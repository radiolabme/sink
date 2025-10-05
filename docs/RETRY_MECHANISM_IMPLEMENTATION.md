# Retry Mechanism

**Feature:** Retry commands until success or timeout

**Added:** Implementation complete, tested, and documented

---

## Overview

The retry mechanism allows any command step to be retried until it succeeds (exit code 0) or a timeout is reached. This is useful for waiting for services to become ready, files to appear, or network operations to complete.

**Key Design:** No ambiguity. Only one retry mode: `"until"` (retry until success or timeout).

---

## Basic Usage

### Without Retry (Default Behavior)

```json
{
  "name": "Install package",
  "command": "brew install jq"
}
```

**Behavior:** Run once, fail immediately if error

---

### With Retry

```json
{
  "name": "Wait for database",
  "command": "nc -z localhost 5432",
  "retry": "until",
  "timeout": "30s"
}
```

**Behavior:** 
- Keep running command every 1 second
- Stop when command succeeds (exit code 0)
- Stop after 30 seconds (timeout)
- Report last error on timeout

---

## Configuration Fields

### `retry` (optional)

**Type:** string  
**Values:** `"until"` (only valid value)  
**Default:** Not set (no retry)

**Behavior:** When set to `"until"`, the command will be retried until it succeeds or timeout is reached.

### `timeout` (optional)

**Type:** string (duration format)  
**Examples:** `"30s"`, `"2m"`, `"5m"`, `"1h"`  
**Default:** `"60s"` (if retry is set but timeout is omitted)

**Format:** Go duration format:
- `s` = seconds (e.g., `"30s"` = 30 seconds)
- `m` = minutes (e.g., `"2m"` = 2 minutes)
- `h` = hours (e.g., `"1h"` = 1 hour)

**Notes:**
- Only used if `retry` is set
- Invalid formats will cause an error
- Negative durations are not allowed

---

## Common Use Cases

### 1. Wait for Port to Open

```json
{
  "name": "Wait for PostgreSQL",
  "command": "nc -z localhost 5432",
  "retry": "until",
  "timeout": "30s"
}
```

**When to use:** After starting a service, wait for it to listen on a port

---

### 2. Wait for HTTP Endpoint

```json
{
  "name": "Wait for web app",
  "command": "curl -f http://localhost:8080/health",
  "retry": "until",
  "timeout": "60s"
}
```

**When to use:** After starting a web server, wait for health check to pass

---

### 3. Wait for File to Exist

```json
{
  "name": "Wait for cloud-init",
  "command": "test -f /var/lib/cloud/instance/boot-finished",
  "retry": "until",
  "timeout": "5m"
}
```

**When to use:** Wait for a file to be created by another process

---

### 4. Wait for File Content

```json
{
  "name": "Wait for log message",
  "command": "grep -q 'Server started' /var/log/app.log",
  "retry": "until",
  "timeout": "2m"
}
```

**When to use:** Wait for specific content to appear in a log file

---

### 5. Wait for Command Success

```json
{
  "name": "Wait for database ready",
  "command": "pg_isready -U postgres",
  "retry": "until",
  "timeout": "30s"
}
```

**When to use:** Wait for a service-specific readiness check

---

### 6. Retry Network Operations

```json
{
  "name": "Download file",
  "command": "curl -fSL https://example.com/file.tar.gz -o /tmp/file.tar.gz",
  "retry": "until",
  "timeout": "2m"
}
```

**When to use:** Retry downloads on transient network errors

---

### 7. Wait for Multiple Conditions

```json
{
  "name": "Wait for system ready",
  "command": "test -f /tmp/ready && systemctl is-active postgresql",
  "retry": "until",
  "timeout": "2m"
}
```

**When to use:** Wait for complex conditions using shell operators

---

## Retry in Remediation Steps

Retry works in remediation steps too:

```json
{
  "name": "Ensure package installed",
  "check": "which jq",
  "on_missing": [
    {
      "name": "Install jq",
      "command": "brew install jq",
      "retry": "until",
      "timeout": "5m"
    }
  ]
}
```

**When to use:** Retry installation commands that might fail due to locks or transient errors

---

## Behavior Details

### Polling Interval

**Fixed at 1 second between attempts**

Example with 5s timeout:
```
t=0s:  Run command → Fail → Sleep 1s
t=1s:  Run command → Fail → Sleep 1s
t=2s:  Run command → Fail → Sleep 1s
t=3s:  Run command → Fail → Sleep 1s
t=4s:  Run command → Fail → Sleep 1s
t=5s:  Timeout reached → Report failure
```

### Success Criteria

**Exit code 0** is the only success condition.

Any non-zero exit code = not ready yet, keep retrying.

### Timeout Behavior

**On timeout:**
- Execution stops
- Last error is reported
- Step fails

**Example output:**
```
❌ Timeout after 30s
   Last error: exit code 1: connection refused
```

### Error Tracking

**During polling:**
- Errors are expected (not ready yet)
- No output/spam during retries
- Last error is saved

**On success:**
```
✅ Ready after 3s
```

**On timeout:**
```
❌ Timeout after 30s
   Last error: exit code 1: connection refused
```

---

## Examples: Full Configs

### Django + PostgreSQL

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
          "command": "createdb myapp || true"
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

### Microservices Startup

```json
{
  "version": "1.0.0",
  "platforms": [
    {
      "os": "linux",
      "match": "linux*",
      "name": "Linux",
      "distributions": [
        {
          "ids": ["ubuntu"],
          "name": "Ubuntu",
          "install_steps": [
            {
              "name": "Start Gateway",
              "command": "docker run -d -p 8080:8080 gateway"
            },
            {
              "name": "Wait for Gateway",
              "command": "curl -f http://localhost:8080/health",
              "retry": "until",
              "timeout": "30s"
            },
            {
              "name": "Start Auth Service",
              "command": "docker run -d -p 9000:9000 auth"
            },
            {
              "name": "Wait for Auth",
              "command": "curl -f http://localhost:9000/health",
              "retry": "until",
              "timeout": "30s"
            },
            {
              "name": "Start Users Service",
              "command": "docker run -d -p 9001:9001 users"
            },
            {
              "name": "Wait for Users",
              "command": "nc -z localhost 9001",
              "retry": "until",
              "timeout": "30s"
            }
          ]
        }
      ]
    }
  ]
}
```

---

## Error Handling

### Invalid Timeout Format

```json
{
  "command": "echo test",
  "retry": "until",
  "timeout": "invalid"
}
```

**Result:**
```
❌ Error: invalid timeout 'invalid': time: invalid duration "invalid" (use format like '30s', '2m', '1h')
```

### Negative Timeout

```json
{
  "timeout": "-10s"
}
```

**Result:** Parse error, step fails

### Missing Timeout

```json
{
  "command": "nc -z localhost 5432",
  "retry": "until"
}
```

**Result:** Uses default 60s timeout

---

## Design Rationale

### Why Only "until" Mode?

**Question:** Why not support retry count (e.g., `retry: 3`)?

**Answer:** Ambiguity and complexity.

- `retry: 5000` → 5000 attempts? 5000 seconds? 5000 milliseconds?
- Most use cases need "wait until ready" not "try N times"
- Timeout is clearer than attempt count

**Philosophy:** One unambiguous mode is better than multiple confusing modes.

### Why 1 Second Polling Interval?

**Tradeoffs:**
- Too fast (100ms): Wastes CPU, hammers services
- Too slow (5s): Adds unnecessary latency
- 1s: Good balance for most use cases

**Not configurable (yet):** Keep it simple. Can add `interval: "2s"` later if needed.

### Why Exit Code 0 Only?

**Exit code 0 = success** is universal shell convention.

Any other exit code = failure/not ready:
- 1 = general error
- 2 = misuse of shell builtins
- 127 = command not found
- etc.

**This aligns with the "doneness" principle:** Success = exit code 0.

---

## Limitations

1. **Fixed 1s polling interval** (not yet configurable)
2. **No exponential backoff** (constant 1s between attempts)
3. **No custom success criteria** (exit code 0 only)
4. **No progress indication** (silent during polling)

**These are intentional simplifications. Can be added later if needed based on real-world usage.**

---

## Future Considerations

**Possible enhancements (not implemented):**

1. **Configurable interval:**
   ```json
   {"retry": "until", "interval": "2s", "timeout": "1m"}
   ```

2. **Exponential backoff:**
   ```json
   {"retry": "until", "backoff": "exponential", "timeout": "5m"}
   ```

3. **Progress output:**
   ```
   ⏳ Waiting for :5432 (attempt 5, elapsed 5s)...
   ```

4. **Custom success exit codes:**
   ```json
   {"retry": "until", "success_codes": [0, 2], "timeout": "30s"}
   ```

**But:** Keep it simple until there's clear demand from real-world usage.

---

## Schema

```json
{
  "command": {
    "type": "string",
    "description": "Shell command to execute"
  },
  "retry": {
    "type": "string",
    "enum": ["until"],
    "description": "Retry behavior. 'until' = keep retrying until success or timeout"
  },
  "timeout": {
    "type": "string",
    "pattern": "^[0-9]+(s|m|h)$",
    "description": "Timeout for retry (e.g., '30s', '2m', '5m'). Defaults to 60s if retry is set.",
    "examples": ["30s", "2m", "5m", "1h"]
  }
}
```

---

## Testing

**Unit tests:** 3 test functions covering:
- Basic retry behavior
- Timeout handling
- Invalid timeout formats
- Retry in remediation steps

**Integration tests:** Test configs covering:
- Port waiting (nc)
- HTTP endpoint waiting (curl)
- File creation waiting (test -f)
- Command success waiting (pg_isready)

**All tests pass.**

---

## Code Size

**Implementation:**
- Types: +2 fields (Retry, Timeout)
- Executor: +120 LOC (executeCommandWithRetry, executeRemediationWithRetry)
- Schema: +10 LOC
- Tests: +170 LOC

**Total new code: ~130 LOC (excluding tests)**

**Well within budget!**
