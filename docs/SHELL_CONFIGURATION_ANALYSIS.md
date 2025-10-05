# Shell Configuration Analysis

**Date:** October 4, 2025  
**Issues:** Hardcoded shell paths, low timeout coverage  
**Status:** ANALYSIS & RECOMMENDATIONS

---

## Problems Identified

### 1. Hardcoded Shell in transport.go

**Current code:**
```go
switch runtime.GOOS {
case "windows":
    shell = "cmd.exe"
    shellFlag = "/C"
default:
    // Unix-like systems (darwin, linux, etc.)
    shell = "/bin/sh"
    shellFlag = "-c"
}
```

**Issues:**

❌ **Assumes `/bin/sh` exists** - Not true for:
- NixOS (shell at `/run/current-system/sw/bin/sh`)
- Custom Linux builds
- Containers with minimal filesystems
- BSD variants with different paths

❌ **Assumes `/bin/sh` is POSIX** - Not always true:
- Some systems link `/bin/sh` to `dash` (limited features)
- Some link to `bash` (extended features)
- Behavior varies across systems

❌ **No way to use other shells:**
- Cannot use `bash` for bash-specific features
- Cannot use `zsh` for zsh-specific features
- Cannot use `fish` for fish-specific syntax
- Cannot use custom shells (`nushell`, `xonsh`, etc.)

❌ **Windows assumptions:**
- Assumes `cmd.exe` on Windows
- No PowerShell support
- No WSL bash support

---

### 2. Low Timeout Test Coverage

**Current coverage:**
```
executeCommandWithRetry      90.3%
executeRemediationWithRetry  77.4%
```

**Missing test scenarios:**
- ❌ Timeout actually triggering (commands that take > timeout)
- ❌ Commands that succeed just before timeout
- ❌ Invalid timeout formats (already tested in retry_test.go)
- ❌ Very short timeouts (<1s)
- ❌ Edge case: timeout = poll interval

---

### 3. Low Executor Coverage Overall

**Current coverage:**
```
ExecuteStep                  93.8%
executeCommand               92.9%
executeCheckError            85.7%
executeCheckRemediate        92.3%
executeRemediation           85.7%
interpolate                  88.9%
transport.Run                88.9%
```

**Missing scenarios:**
- ❌ Template interpolation errors
- ❌ Retry with timeout edge cases
- ❌ Remediation with errors
- ❌ Certain error paths

---

## Solution 1: Shell Configuration in Schema

### Approach: Per-Command Shell Override

Add optional `shell` field to command steps:

**Schema changes:**
```json
{
  "name": "Run bash script",
  "command": "echo ${BASH_VERSION}",
  "shell": "/bin/bash"
}

{
  "name": "Run zsh script", 
  "command": "echo ${ZSH_VERSION}",
  "shell": "/usr/bin/zsh"
}

{
  "name": "Use default shell",
  "command": "echo hello"
  // No shell field = use platform default
}
```

**Pros:**
- ✅ Maximum flexibility
- ✅ Backward compatible (shell is optional)
- ✅ Can use different shells for different commands
- ✅ Clear and explicit

**Cons:**
- ⚠️ Adds complexity to every command step
- ⚠️ Users must know shell paths
- ⚠️ Repetitive if many commands need same shell

**LOC Impact:** ~30 LOC (schema + types + executor changes)

---

### Approach: Platform-Level Shell Configuration

Add optional `shell` field to platform definition:

**Schema changes:**
```json
{
  "platforms": [{
    "os": "darwin",
    "match": "darwin*",
    "name": "macOS",
    "shell": "/bin/bash",
    "install_steps": [
      {"name": "Step 1", "command": "echo ${BASH_VERSION}"},
      {"name": "Step 2", "command": "echo test"}
    ]
  }]
}
```

**Pros:**
- ✅ Set once, applies to all steps in platform
- ✅ Less repetitive
- ✅ Still backward compatible
- ✅ Can override per-distribution

**Cons:**
- ⚠️ Cannot mix shells within same platform
- ⚠️ Less flexible than per-command

**LOC Impact:** ~25 LOC

---

### Approach: Environment Variable Discovery

Let shell path be discovered via environment:

**Schema changes:**
```json
{
  "facts": {
    "shell": {
      "command": "echo ${SHELL:-/bin/sh}",
      "description": "User's preferred shell"
    }
  },
  "platforms": [{
    "os": "darwin",
    "name": "macOS",
    "shell": "{{ .shell }}",
    "install_steps": [...]
  }]
}
```

**Pros:**
- ✅ Respects user's $SHELL preference
- ✅ Works across systems automatically
- ✅ Uses existing fact system

**Cons:**
- ⚠️ $SHELL might not be suitable for scripts
- ⚠️ Requires fact gathering
- ⚠️ Complex template interpolation

**LOC Impact:** ~10 LOC (just template support)

---

### Approach: Shell Auto-Discovery

Search for shell in common locations:

**Code changes:**
```go
func findShell() string {
    // Try in order of preference
    candidates := []string{
        "/bin/sh",
        "/usr/bin/sh", 
        "/run/current-system/sw/bin/sh", // NixOS
        "/system/bin/sh", // Android
        "/bin/bash",
        "/usr/bin/bash",
    }
    
    for _, shell := range candidates {
        if _, err := os.Stat(shell); err == nil {
            return shell
        }
    }
    
    return "sh" // Hope it's in PATH
}
```

**Pros:**
- ✅ Works automatically across systems
- ✅ No config changes needed
- ✅ Handles NixOS, containers, etc.

**Cons:**
- ⚠️ Discovery happens at runtime
- ⚠️ Order of preference might not match user intent
- ⚠️ Still hardcoded list of paths

**LOC Impact:** ~15 LOC

---

## Recommended Solution

### **Hybrid Approach: Auto-Discovery + Optional Override**

**Implementation:**

1. **Auto-discover shell** (fallback to common paths)
2. **Allow platform-level override** (for platforms that need specific shell)
3. **Allow per-command override** (for commands that need specific shell)

**Schema:**
```json
{
  "platforms": [{
    "os": "darwin",
    "match": "darwin*",
    "name": "macOS",
    "shell": "/bin/bash",  // Optional: Override platform default
    "install_steps": [
      {
        "name": "Bash-specific command",
        "command": "echo ${BASH_VERSION}",
        "shell": "/usr/local/bin/bash"  // Optional: Override platform shell
      },
      {
        "name": "Default shell command",
        "command": "echo hello"
        // Uses platform shell or auto-discovered shell
      }
    ]
  }]
}
```

**Precedence:**
1. Per-command `shell` field (highest priority)
2. Platform `shell` field
3. Auto-discovered shell (lowest priority)

**Auto-discovery logic:**
```go
func (lt *LocalTransport) getShell() (string, string) {
    // Check if custom shell set
    if lt.Shell != "" {
        return lt.Shell, lt.ShellFlag
    }
    
    // Platform-specific defaults
    switch runtime.GOOS {
    case "windows":
        return "cmd.exe", "/C"
    default:
        // Auto-discover Unix shell
        candidates := []string{
            "/bin/sh",
            "/usr/bin/sh",
            "/run/current-system/sw/bin/sh", // NixOS
            "/bin/bash",
        }
        
        for _, shell := range candidates {
            if _, err := os.Stat(shell); err == nil {
                return shell, "-c"
            }
        }
        
        // Fallback: hope it's in PATH
        return "sh", "-c"
    }
}
```

**Why This Works:**

✅ **Default behavior unchanged** - Still uses `/bin/sh` on most systems  
✅ **Works on NixOS** - Auto-discovers `/run/current-system/sw/bin/sh`  
✅ **Works in containers** - Falls back to `sh` in PATH  
✅ **Allows customization** - Users can override when needed  
✅ **Backward compatible** - No schema changes required for existing configs  
✅ **LOC budget friendly** - ~50 LOC total

---

## Solution 2: Improve Timeout Test Coverage

### Missing Test Scenarios

**1. Actual Timeout (Command Runs Longer Than Timeout)**

```go
func TestRetryActualTimeout(t *testing.T) {
    mockTransport := &MockTransport{
        responses: map[string]MockResponse{
            "sleep 10": {
                stdout:   "",
                exitCode: 0,
                delay:    5 * time.Second, // Simulates slow command
            },
        },
    }
    
    cmd := CommandStep{
        Command: "sleep 10",
        Retry:   stringPtr("until"),
        Timeout: stringPtr("2s"),
    }
    
    executor := NewExecutor(mockTransport)
    result := executor.executeCommandWithRetry("test", cmd, nil)
    
    if result.Status != "failed" {
        t.Error("Expected timeout failure")
    }
    if !strings.Contains(result.Error, "Timeout") {
        t.Errorf("Expected timeout error, got: %s", result.Error)
    }
}
```

**2. Success Just Before Timeout**

```go
func TestRetrySuccessNearTimeout(t *testing.T) {
    attempts := 0
    mockTransport := &MockTransportWithTracking{
        onRun: func(cmd string) {
            attempts++
        },
        responses: map[string]MockResponse{
            "nc -z localhost 5432": {
                // Succeeds on 4th attempt (at ~4 seconds)
                dynamic: func() (string, string, int, error) {
                    if attempts >= 4 {
                        return "ok", "", 0, nil
                    }
                    return "", "connection refused", 1, nil
                },
            },
        },
    }
    
    cmd := CommandStep{
        Command: "nc -z localhost 5432",
        Retry:   stringPtr("until"),
        Timeout: stringPtr("5s"),
    }
    
    executor := NewExecutor(mockTransport)
    result := executor.executeCommandWithRetry("test", cmd, nil)
    
    if result.Status != "success" {
        t.Error("Expected success just before timeout")
    }
    if attempts != 4 {
        t.Errorf("Expected 4 attempts, got %d", attempts)
    }
}
```

**3. Template Interpolation Errors**

```go
func TestRetryTemplateError(t *testing.T) {
    cmd := CommandStep{
        Command: "echo {{ .nonexistent }}",
        Retry:   stringPtr("until"),
        Timeout: stringPtr("5s"),
    }
    
    executor := NewExecutor(&MockTransport{})
    result := executor.executeCommandWithRetry("test", cmd, Facts{})
    
    if result.Status != "failed" {
        t.Error("Expected template error")
    }
    if !strings.Contains(result.Error, "template") {
        t.Errorf("Expected template error, got: %s", result.Error)
    }
}
```

**4. Remediation With Retry and Error**

```go
func TestRemediationRetryWithError(t *testing.T) {
    mockTransport := &MockTransport{
        responses: map[string]MockResponse{
            "test -f /tmp/file": {
                exitCode: 1, // Always fails
            },
            "create-file": {
                exitCode: 1, // Remediation fails
            },
        },
    }
    
    step := CheckRemediateStep{
        Check: "test -f /tmp/file",
        OnMissing: []RemediationStep{
            {
                Name:    "Create file",
                Command: "create-file",
                Retry:   stringPtr("until"),
                Timeout: stringPtr("2s"),
            },
        },
    }
    
    executor := NewExecutor(mockTransport)
    result := executor.executeCheckRemediate("test", step, nil)
    
    if result.Status != "failed" {
        t.Error("Expected remediation failure")
    }
}
```

**LOC Impact:** ~150 LOC for comprehensive timeout tests

---

## Solution 3: Improve Transport Test Coverage

### Missing Scenarios

**1. Different Exit Codes**

Already covered in `TestLocalTransportExitCodes` ✅

**2. Environment Variable Handling**

```go
func TestTransportEnvironment(t *testing.T) {
    transport := &LocalTransport{
        Env: []string{"TEST_VAR=hello"},
    }
    
    stdout, _, exitCode, err := transport.Run("echo $TEST_VAR")
    
    if err != nil || exitCode != 0 {
        t.Fatalf("Command failed: %v", err)
    }
    
    if !strings.Contains(stdout, "hello") {
        t.Errorf("Expected 'hello' in output, got: %s", stdout)
    }
}
```

**3. Working Directory**

```go
func TestTransportWorkDir(t *testing.T) {
    tmpDir := t.TempDir()
    
    transport := &LocalTransport{
        WorkDir: tmpDir,
    }
    
    stdout, _, exitCode, err := transport.Run("pwd")
    
    if err != nil || exitCode != 0 {
        t.Fatalf("Command failed: %v", err)
    }
    
    if !strings.Contains(stdout, tmpDir) {
        t.Errorf("Expected %s in output, got: %s", tmpDir, stdout)
    }
}
```

**4. Stderr Capture**

```go
func TestTransportStderr(t *testing.T) {
    transport := NewLocalTransport()
    
    stdout, stderr, exitCode, _ := transport.Run("echo error >&2")
    
    if exitCode != 0 {
        t.Error("Command should succeed")
    }
    
    if stdout != "" {
        t.Errorf("Expected empty stdout, got: %s", stdout)
    }
    
    if !strings.Contains(stderr, "error") {
        t.Errorf("Expected 'error' in stderr, got: %s", stderr)
    }
}
```

**5. Command Not Found (Exit 127)**

```go
func TestTransportCommandNotFound(t *testing.T) {
    transport := NewLocalTransport()
    
    _, _, exitCode, _ := transport.Run("nonexistent-command-12345")
    
    if exitCode != 127 {
        t.Errorf("Expected exit code 127, got %d", exitCode)
    }
}
```

**LOC Impact:** ~100 LOC for transport tests

---

## Implementation Plan

### Phase 1: Shell Configuration (50 LOC)

**Priority: HIGH** - Fixes real-world issues (NixOS, containers)

1. Add `Shell` and `ShellFlag` fields to `LocalTransport`
2. Add `shell` field to Platform and CommandStep in schema
3. Implement auto-discovery with fallback
4. Add tests for shell override

**Files to modify:**
- `src/transport.go` (~25 LOC)
- `src/types.go` (~10 LOC)
- `data/install-config.schema.json` (~15 LOC)
- `src/transport_test.go` (~50 LOC tests)

**Total: ~100 LOC (50 implementation + 50 tests)**

---

### Phase 2: Timeout Test Coverage (150 LOC)

**Priority: MEDIUM** - Improves confidence in retry mechanism

1. Add timeout tests to `executor_test.go`
2. Add remediation retry tests
3. Add template error tests
4. Enhance MockTransport to support delays

**Files to modify:**
- `src/executor_test.go` (~150 LOC tests)
- `src/mock_transport.go` (~20 LOC enhancements)

**Total: ~170 LOC**

---

### Phase 3: Transport Test Coverage (100 LOC)

**Priority: LOW** - Nice to have, but transport is simple

1. Add environment variable tests
2. Add working directory tests
3. Add stderr tests
4. Add command not found tests

**Files to modify:**
- `src/transport_test.go` (~100 LOC tests)

**Total: ~100 LOC**

---

## LOC Budget Analysis

**Current state:** ~1,650 LOC  
**Budget remaining:** ~350 LOC

**Phase 1 (Shell config):** 100 LOC  
**Phase 2 (Timeout tests):** 170 LOC  
**Phase 3 (Transport tests):** 100 LOC  

**Total needed:** 370 LOC ⚠️ **OVER BUDGET by 20 LOC**

**Recommendation:**
1. ✅ **Do Phase 1** (Shell config) - Essential for NixOS/containers - 100 LOC
2. ✅ **Do Phase 2** (Timeout tests) - Important for confidence - 170 LOC
3. ❌ **Skip Phase 3** (Transport tests) - Nice to have, not critical - 100 LOC

**Final budget:** 100 + 170 = 270 LOC ✅ **Within budget** (80 LOC remaining)

---

## Alternative: Minimal Shell Fix

If we want to minimize LOC impact, we can do **auto-discovery only** without schema changes:

**Code change (transport.go only):**
```go
func (lt *LocalTransport) getShell() (string, string) {
    switch runtime.GOOS {
    case "windows":
        return "cmd.exe", "/C"
    default:
        // Try common Unix shell locations
        for _, shell := range []string{
            "/bin/sh",
            "/usr/bin/sh",
            "/run/current-system/sw/bin/sh", // NixOS
        } {
            if _, err := os.Stat(shell); err == nil {
                return shell, "-c"
            }
        }
        return "sh", "-c" // Hope it's in PATH
    }
}
```

**LOC Impact:** ~15 LOC (no schema changes, no tests)

**Pros:**
- ✅ Fixes NixOS/container issues immediately
- ✅ Minimal LOC budget impact
- ✅ No breaking changes

**Cons:**
- ❌ No way to override shell
- ❌ Fixed order of preference
- ❌ Cannot use bash/zsh for specific commands

**Recommendation:** Start with this minimal fix, add override capability later if users request it.

---

## Decision Matrix

| Solution | LOC | Fixes NixOS | Allows Override | Breaking Changes |
|----------|-----|-------------|-----------------|------------------|
| **Minimal auto-discovery** | 15 | ✅ | ❌ | ❌ |
| **Per-command override** | 30 | ✅ | ✅ | ❌ |
| **Platform override** | 25 | ✅ | ✅ | ❌ |
| **Hybrid (recommended)** | 50 | ✅ | ✅ | ❌ |
| **Environment variable** | 10 | ⚠️ | ✅ | ❌ |

**Recommendation:**
1. Start with **minimal auto-discovery** (15 LOC) to fix immediate issues
2. Add **hybrid override** later (35 more LOC) if users request it
3. Keep under budget and validate with real-world usage first

---

## Conclusion

### Immediate Action (Within Budget)

1. ✅ **Add shell auto-discovery** - 15 LOC - Fixes NixOS/containers
2. ✅ **Add timeout tests** - 170 LOC - Improves confidence
3. ✅ **Add basic transport tests** - 50 LOC - Essential coverage

**Total: 235 LOC** ✅ **Well within budget** (115 LOC remaining)

### Future Enhancements (If Needed)

4. ⚠️ **Add shell override to schema** - 35 LOC - If users request it
5. ⚠️ **Add comprehensive transport tests** - 50 LOC - If needed

### Philosophy

**"Fix the real problem first (NixOS), add configurability later (if needed)."**

- ✅ Most users: Auto-discovery works fine
- ✅ NixOS users: Auto-discovery finds their shell
- ✅ Power users: Can request override feature if needed
- ✅ LOC budget: Preserved for other features

**YAGNI principle:** You Aren't Gonna Need It (yet)

Don't build override mechanism until users actually need it. Start simple, evolve based on real usage.
