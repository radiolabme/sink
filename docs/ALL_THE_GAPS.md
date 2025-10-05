# All the Gaps: What's Actually Missing

**Date:** October 4, 2025  
**Context:** Comprehensive gap analysis for real-world use cases

---

## The Real-World Use Cases (From THE_REAL_PROBLEM.md)

1. **Setting up AI coding tools** (Claude, Copilot, Cursor, Windsurf, Aider)
2. **MCP server deployment** (Security, sandboxing, configuration)
3. **Linux desktop in VM** (Ubuntu/Debian desktop, not just server)
4. **Container runtime setup** (Docker/Colima/Lima/Podman complexity)
5. **Development environments** (Language toolchains, dependencies)

Let me analyze each use case and identify **every gap**.

---

## Use Case 1: Setting Up AI Coding Tools

### **What Users Need to Do:**

```
1. Install VS Code (or Cursor, or Windsurf)
2. Install extension (from marketplace or .vsix)
3. Get API key (from provider website)
4. Configure extension with API key
5. Test that it works
```

### **What Sink Can Do Today:**

```json
{
  "platforms": [{
    "os": "darwin",
    "install_steps": [
      {
        "name": "Install VS Code",
        "check": "command -v code",
        "on_missing": [
          {"name": "Install", "command": "brew install --cask visual-studio-code"}
        ]
      },
      {
        "name": "Install extension",
        "command": "code --install-extension anthropics.claude-vscode"
      }
    ]
  }]
}
```

### **Gaps:**

| Gap | Description | Workaround Available? | Impact |
|-----|-------------|----------------------|--------|
| **No interactive prompts** | Can't ask for API key | ❌ Have to edit config manually | HIGH |
| **No secret handling** | API keys visible in logs | ✅ Can use env vars | MEDIUM |
| **No config file editing** | Can't set VS Code settings.json | ✅ Can use jq/sed commands | LOW |
| **No verification** | Can't test extension works | ✅ Can add test command | LOW |
| **No error recovery** | If extension install fails, no retry | ✅ Can re-run | LOW |

**PRIORITY GAP: Interactive prompts**

---

## Use Case 2: MCP Server Deployment

### **What Users Need to Do:**

```
1. Choose runtime (Node.js, Python, Go binary, Docker)
2. Install runtime if missing
3. Install MCP server package
4. Configure server (ports, API keys, permissions)
5. Set up security/sandboxing
6. Start server as background service
7. Configure client to connect
```

### **What Sink Can Do Today:**

```json
{
  "install_steps": [
    {
      "name": "Install Node.js",
      "check": "command -v node",
      "on_missing": [
        {"name": "Install", "command": "brew install node"}
      ]
    },
    {
      "name": "Install MCP server",
      "command": "npm install -g @modelcontextprotocol/server-filesystem"
    },
    {
      "name": "Start server",
      "command": "mcp-server-filesystem &"
    }
  ]
}
```

### **Gaps:**

| Gap | Description | Workaround Available? | Impact |
|-----|-------------|----------------------|--------|
| **No background process management** | Can't reliably start/stop servers | ⚠️ Can use `&` but brittle | HIGH |
| **No service installation** | Can't create systemd/launchd services | ✅ Can use commands, but complex | HIGH |
| **No port checking** | Can't detect if port already in use | ✅ Can use `lsof` command | MEDIUM |
| **No health checking** | Can't verify server actually started | ✅ Can use curl command | MEDIUM |
| **No log management** | Server logs go to stdout/stderr | ❌ No way to capture/redirect | MEDIUM |
| **No sandboxing** | Server runs with full user permissions | ✅ Can use sudo/docker commands | LOW |

**PRIORITY GAPS: Background process management, service installation**

---

## Use Case 3: Linux Desktop in VM

### **What Users Need to Do:**

```
1. Choose VM tool (Lima, Colima, Multipass, VirtualBox)
2. Install VM tool
3. Find/create Ubuntu desktop image (not just server!)
4. Configure VM (RAM, CPU, disk, networking)
5. Start VM
6. Install desktop environment
7. Configure display (resolution, HiDPI)
8. Set up shared folders
9. Configure networking (works with VPN)
10. Open desktop GUI
```

### **What Sink Can Do Today:**

```json
{
  "install_steps": [
    {
      "name": "Install Lima",
      "check": "command -v limactl",
      "on_missing": [
        {"name": "Install", "command": "brew install lima"}
      ]
    },
    {
      "name": "Create VM",
      "command": "limactl create --name=dev template://ubuntu-lts"
    },
    {
      "name": "Start VM",
      "command": "limactl start dev"
    }
  ]
}
```

### **Gaps:**

| Gap | Description | Workaround Available? | Impact |
|-----|-------------|----------------------|--------|
| **No image customization** | Lima templates are server-only | ❌ Need custom cloud-init | CRITICAL |
| **No desktop environment** | No GNOME/KDE/Xfce in cloud images | ❌ Need multi-step install | CRITICAL |
| **No resource detection** | Can't auto-size VM for laptop | ✅ Can gather facts about RAM/CPU | HIGH |
| **No network configuration** | VPN compatibility issues | ⚠️ Can set Lima network mode, but complex | HIGH |
| **No display setup** | No X11/Wayland configuration | ❌ Need custom scripts | HIGH |
| **No shared folders** | Can't mount host directories | ✅ Lima supports this via config | MEDIUM |
| **No GUI launch** | Can't open desktop window | ⚠️ Can SSH with X forwarding, but slow | HIGH |

**PRIORITY GAPS: Image customization, desktop environment install**

---

## Use Case 4: Container Runtime Setup

### **What Users Need to Do:**

```
1. Detect what's already installed (Docker Desktop, Colima, Lima, Podman)
2. Choose best option for platform (macOS vs Linux)
3. Handle license issues (Docker Desktop requires license for companies)
4. Install chosen runtime
5. Configure resources (RAM, CPU, storage)
6. Start runtime
7. Verify containers work
8. Configure registries (Docker Hub, GitHub, private)
```

### **What Sink Can Do Today:**

```json
{
  "platforms": [{
    "os": "darwin",
    "install_steps": [
      {
        "name": "Check for existing runtime",
        "check": "docker info 2>/dev/null || colima status 2>/dev/null"
      },
      {
        "name": "Install Colima if nothing exists",
        "command": "brew install colima"
      },
      {
        "name": "Start Colima",
        "command": "colima start --cpu 2 --memory 4 --disk 60"
      }
    ]
  }]
}
```

### **Gaps:**

| Gap | Description | Workaround Available? | Impact |
|-----|-------------|----------------------|--------|
| **No multi-condition checks** | Can't check "Docker OR Colima OR Lima" elegantly | ⚠️ Can use shell OR, but messy | MEDIUM |
| **No resource auto-sizing** | Can't detect laptop RAM and set appropriate limits | ✅ Can use facts for RAM/CPU | HIGH |
| **No conflict detection** | Can't detect if multiple runtimes installed | ✅ Can check each one | LOW |
| **No preference handling** | Can't let user choose runtime | ❌ No interactive selection | MEDIUM |
| **No state persistence** | Colima config not saved for next run | ✅ Can write config file | LOW |

**PRIORITY GAP: Resource auto-sizing**

---

## Use Case 5: Development Environments

### **What Users Need to Do:**

```
1. Install language runtime (Python, Node, Go, Rust, Ruby)
2. Install version manager (pyenv, nvm, gvm, rustup, rbenv)
3. Install specific version
4. Set default version
5. Install package manager (pip, npm, cargo, gem)
6. Install global tools (black, eslint, golangci-lint, rustfmt)
7. Configure editor integration
8. Set up shell completions
```

### **What Sink Can Do Today:**

```json
{
  "install_steps": [
    {
      "name": "Install Python 3.11",
      "check": "python3.11 --version",
      "on_missing": [
        {"name": "Install", "command": "brew install python@3.11"}
      ]
    },
    {
      "name": "Install pip packages",
      "command": "pip3.11 install black ruff pytest"
    }
  ]
}
```

### **Gaps:**

| Gap | Description | Workaround Available? | Impact |
|-----|-------------|----------------------|--------|
| **No version management** | Installing multiple Python versions is complex | ⚠️ Can install pyenv, but many steps | MEDIUM |
| **No PATH management** | Can't modify .zshrc/.bashrc/.profile safely | ✅ Can append, but risky | MEDIUM |
| **No shell detection** | Don't know which shell user uses | ✅ Can gather as fact | LOW |
| **No virtual environment** | Can't create/activate venv | ✅ Can use commands | LOW |
| **No dependency conflicts** | Can't detect incompatible versions | ❌ No dependency resolution | LOW |

**PRIORITY GAP: PATH management**

---

## Cross-Cutting Gaps (Affect Multiple Use Cases)

### **1. Interactive Input**

**Problem:** Need to ask user for information  
**Examples:**
- API keys (Claude, OpenAI, GitHub)
- Usernames/emails
- Configuration choices (which runtime? which desktop?)
- Confirmation prompts (beyond yes/no)

**Current Workaround:** ❌ None (have to edit config files manually)

**Impact:** CRITICAL - Blocks all AI tool setups

---

### **2. Background Process Management**

**Problem:** Need to start long-running services  
**Examples:**
- MCP servers
- Development databases (PostgreSQL, Redis)
- Docker daemon
- VM managers

**Current Workaround:** ⚠️ Can use `&` but no monitoring, no restart, no logs

**Impact:** HIGH - Services don't survive reboot, can't tell if crashed

---

### **3. Service Installation (systemd/launchd)**

**Problem:** Need to install as system service  
**Examples:**
- MCP servers should auto-start
- Container runtimes should auto-start
- Development services should persist

**Current Workaround:** ✅ Can generate unit files with commands, but very platform-specific

**Impact:** HIGH - Services don't start on boot, not production-ready

---

### **4. File/Template Generation**

**Problem:** Need to create config files from templates  
**Examples:**
- VS Code settings.json with user's API key
- systemd unit files
- Docker Compose files
- Shell rc files with PATH additions

**Current Workaround:** ✅ Can use here-docs or echo, but ugly

**Impact:** MEDIUM - Configs work but are messy

---

### **5. State/Config Persistence**

**Problem:** Need to remember choices for next run  
**Examples:**
- Colima resource limits
- User's preferred runtime (Docker vs Colima)
- API keys (without committing to git)
- VM configuration

**Current Workaround:** ❌ None - configs are static JSON

**Impact:** MEDIUM - Users re-enter same info repeatedly

---

### **6. Image/Asset Management**

**Problem:** Need custom OS images or large downloads  
**Examples:**
- Ubuntu desktop cloud-init image (doesn't exist!)
- Pre-configured VM templates
- Extension .vsix files
- Binary installers

**Current Workaround:** ⚠️ Can curl download, but no caching, no verification

**Impact:** HIGH - Desktop VMs impossible without custom images

---

### **7. Multi-Step Orchestration**

**Problem:** Need to coordinate between multiple tools  
**Examples:**
- Install Lima → Create VM → Install desktop → Configure X11 → Open GUI
- Install VS Code → Install extension → Configure → Test
- Start Colima → Wait for ready → Pull image → Start container

**Current Workaround:** ✅ Can chain commands, but no wait/retry logic

**Impact:** MEDIUM - Brittle, fails if timing is off

---

### **8. Resource Auto-Detection**

**Problem:** Need to size VMs/containers appropriately  
**Examples:**
- 2015 laptop with 8GB RAM → Colima gets 2GB
- 2024 laptop with 192GB RAM → Colima gets 32GB
- Detect available disk space
- Detect CPU cores

**Current Workaround:** ✅ Facts system can do this!

**Impact:** LOW - Can solve with existing features

---

### **9. Error Recovery/Retry**

**Problem:** Network failures, transient errors  
**Examples:**
- Homebrew download fails → retry
- npm install timeout → retry
- VM start fails → try different settings

**Current Workaround:** ❌ None - fails permanently

**Impact:** MEDIUM - Users have to manually retry

---

### **10. Secrets Management**

**Problem:** API keys shouldn't be in plain text  
**Examples:**
- Claude API key
- GitHub tokens
- Database passwords
- SSH keys

**Current Workaround:** ✅ Can use environment variables, but not ideal

**Impact:** MEDIUM - Security concern for shared configs

---

## Gap Priority Matrix

| Priority | Gap | Use Cases Affected | Can Workaround? | LOC to Fix |
|----------|-----|-------------------|-----------------|------------|
| 🔴 **CRITICAL** | Interactive prompts | AI tools, MCP, Dev envs | ❌ No | ~30 |
| 🔴 **CRITICAL** | Custom VM images | Linux desktop | ❌ No (external) | N/A |
| 🟠 **HIGH** | Background processes | MCP servers | ⚠️ Brittle | ~100 |
| 🟠 **HIGH** | Service installation | MCP, Container runtimes | ✅ Yes, complex | ~150 |
| 🟠 **HIGH** | Resource auto-sizing | VMs, Containers | ✅ Yes (facts) | ~20 |
| 🟠 **HIGH** | Desktop environment | Linux desktop | ❌ No (complex) | ~200 |
| 🟡 **MEDIUM** | File templates | All | ✅ Yes (here-docs) | ~50 |
| 🟡 **MEDIUM** | State persistence | All | ❌ No | ~80 |
| 🟡 **MEDIUM** | Error retry | All | ❌ No | ~60 |
| 🟡 **MEDIUM** | Secrets management | AI tools | ✅ Yes (env vars) | ~100 |
| 🟢 **LOW** | Multi-condition checks | Container setup | ⚠️ Yes (messy) | ~40 |
| 🟢 **LOW** | PATH management | Dev envs | ✅ Yes (append) | ~30 |

---

## What Can We Ship in 7 Days?

### **Tier 1: Must Have (Ship Blocker)**

1. ✅ **Interactive prompts** (30 LOC)
   - Without this, AI tool configs require manual editing
   - Pure Go, no platform-specific code

### **Tier 2: Should Have (Quality of Life)**

2. ✅ **Resource auto-sizing** (20 LOC)
   - Use facts to detect RAM/CPU
   - Calculate reasonable VM sizes
   - Example already in docs

3. ✅ **File templates** (50 LOC)
   - Write config files from templates
   - Much cleaner than here-docs
   - Reuses existing interpolation

### **Tier 3: Nice to Have (Polish)**

4. ⚠️ **Multi-condition checks** (40 LOC)
   - Check "Docker OR Colima OR Lima"
   - Makes configs more robust

5. ⚠️ **PATH management** (30 LOC)
   - Safe append to .zshrc/.bashrc
   - Detect which shell

### **Tier 4: Defer to Plugins**

6. ❌ **Background process management** → Plugin
7. ❌ **Service installation** → Plugin (systemd-plugin, launchd-plugin)
8. ❌ **State persistence** → Plugin
9. ❌ **Error retry** → Plugin
10. ❌ **Secrets management** → Use env vars for now, plugin later

### **Tier 5: External Solutions**

11. ❌ **Custom VM images** → Point users to image builders
12. ❌ **Desktop environment** → Multi-step commands, document well

---

## Recommendation: The 7-Day MVP

### **Add 3 Features (100 LOC total):**

1. **Interactive prompts** (30 LOC) - CRITICAL
2. **Resource auto-sizing facts** (20 LOC) - HIGH VALUE
3. **File templates** (50 LOC) - QUALITY OF LIFE

### **Then build 5 configs:**

1. **claude-desktop.json** (uses prompts for API key)
2. **github-copilot.json** (uses prompts for token)
3. **cursor.json** (uses prompts)
4. **colima-setup.json** (uses resource auto-sizing)
5. **python-dev.json** (uses file templates for .zshrc)

### **Ship and validate:**

- Post in AI Discord servers
- Measure: Do they work? Do people use them again?
- Learn: Which gaps hurt most?

### **After validation:**

- If successful: Add Tier 3 features, build plugin system
- If not: Pivot to different use case

---

## The Gaps We're NOT Solving (And Why)

### **Desktop in VM** - TOO COMPLEX
- Requires custom images (external problem)
- Desktop environments have 100+ packages
- Display configuration is OS/DE specific
- Better to document than automate

### **Background Processes** - NEEDS PLUGIN
- Platform-specific (systemd vs launchd vs nothing)
- State management complex (PID files, monitoring)
- Should be systemd-plugin, not core

### **Service Installation** - NEEDS PLUGIN
- Too platform-specific for core
- systemd vs launchd vs OpenRC vs nothing
- Should be separate plugins

### **Secrets Management** - NEEDS PLUGIN
- Platform-specific keychains
- Environment variables work for 80% case
- Can add keychain-plugin later

---

## Bottom Line

### **Sink CAN solve these use cases, but needs 3 features:**

1. ✅ Interactive prompts (30 LOC) - **DO THIS**
2. ✅ Resource facts (20 LOC) - **DO THIS**
3. ✅ File templates (50 LOC) - **DO THIS**

**Total: 100 LOC to make it work.**

### **The other gaps:**

- 50% solvable with workarounds (good enough for validation)
- 30% need plugins (defer to post-validation)
- 20% need external solutions (document, don't build)

**Want me to implement the 100 LOC to close the critical gaps?**
