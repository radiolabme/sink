# What NOT to Build Into Sink Core

**Date:** October 4, 2025  
**Context:** Analyzing feature requests through the lens of "OS/platform-specific invariants trap"

---

## The Core Question

> **"Which features trap Sink into ingesting OS or platform-specific invariants?"**

If a feature requires knowing about:
- Package manager differences (apt vs brew vs dnf vs apk vs snap vs...)
- File path conventions (/usr/local vs /opt vs C:\Program Files...)
- Desktop environments (GNOME vs KDE vs Xfce vs...)
- Init systems (systemd vs launchd vs OpenRC vs...)
- Shell differences (bash vs zsh vs fish vs pwsh...)

**→ DON'T BUILD IT INTO CORE. Use plugin pattern.**

---

## Feature Analysis: What's Already Solved?

### ❌ **ANTI-PATTERN: Package Manager Abstraction**

**The trap:**
```json
// This seems nice...
{"install_package": "visual-studio-code"}

// But requires Sink to know:
- macOS: brew install --cask visual-studio-code
- Ubuntu: snap install code --classic
- Debian: wget .deb && dpkg -i
- Fedora: dnf install code
- Arch: yay -S visual-studio-code-bin
- Alpine: Not available, use tarball
- Windows: winget install Microsoft.VisualStudioCode
- FreeBSD: pkg install vscode
- Nix: nix-env -iA nixpkgs.vscode

// And this changes CONSTANTLY:
- Snap removed from some distros
- Flatpak vs Snap wars
- New package managers (brew alternatives)
- Different package names per repo
- Version differences
```

**Why it's a trap:**
1. **Endless maintenance** - Package ecosystems change monthly
2. **Incomplete coverage** - Always missing some distro/version
3. **Wrong layer** - Package managers already exist, use them
4. **Opinionated** - Choosing snap over flatpak is political
5. **Breaking changes** - Distros change defaults constantly

**What to do instead:**
```json
// Just use commands - let users pick their package manager
{
  "os": "darwin",
  "install_steps": [
    {"name": "Install VS Code", "command": "brew install --cask visual-studio-code"}
  ]
}

{
  "os": "linux",
  "distributions": [
    {
      "ids": ["ubuntu"],
      "install_steps": [
        {"name": "Install VS Code", "command": "snap install code --classic"}
      ]
    }
  ]
}
```

**You said it perfectly:** 
> "There are already so many [package managers]. Isn't the 'new standard' an anti-pattern?"

**YES.** Don't build the 15th standard. Use the existing ones via commands.

---

### ✅ **NOT A TRAP: Interactive Prompts**

**The question:**
> "Interactivity does seem like an issue... can't I just readline or is that too OS specific?"

**Analysis:**

**Go's `bufio.ReadString()` works everywhere:**
```go
import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

// This works on macOS, Linux, Windows, FreeBSD, everywhere Go runs
reader := bufio.NewReader(os.Stdin)
fmt.Print("Enter API key: ")
apiKey, _ := reader.ReadString('\n')
apiKey = strings.TrimSpace(apiKey)
```

**For secrets (no echo):**
```go
import "golang.org/x/term"

// This is cross-platform (uses termios on Unix, Console API on Windows)
fmt.Print("Enter password: ")
password, _ := term.ReadPassword(int(os.Stdin.Fd()))
```

**Is this a trap?**
- ✅ **NOT OS-specific** - Go stdlib handles platform differences
- ✅ **NOT platform-specific** - Works in any terminal
- ✅ **Minimal code** - ~20 LOC in core
- ✅ **No external dependencies** - Pure Go (term is golang.org/x/term, semi-official)

**Verdict: SAFE TO ADD TO CORE**

---

### ❌ **TRAP: VS Code Extension Management**

**The trap:**
```json
{"install_vscode_extension": "github.copilot"}

// Requires knowing:
- Where is VS Code installed? (platform-specific paths)
- Which binary? (code vs code-insiders vs codium)
- Extension marketplace API (Microsoft-specific)
- Authentication (GitHub auth, marketplace tokens)
- Extension dependencies (recursive installs)
- Extension conflicts (incompatible versions)
- Offline installs (.vsix files)
```

**Why it's a trap:**
1. **VS Code specific** - What about Cursor? Zed? Windsurf?
2. **Marketplace coupling** - Tied to Microsoft's infrastructure
3. **Installation paths vary** - Snap vs Homebrew vs manual install
4. **Multiple VS Code variants** - Code, Insiders, Codium, forks
5. **Auth complexity** - Requires GitHub tokens for some extensions

**What to do instead:**
```json
// Just call the VS Code CLI (it exists!)
{
  "name": "Install Copilot extension",
  "command": "code --install-extension github.copilot",
  "check": "code --list-extensions | grep -q github.copilot"
}
```

**VS Code already has a CLI.** Use it. Don't reimplement it.

**Verdict: DON'T BUILD. Use `code` CLI via commands.**

---

### ❌ **TRAP: JSON/Config File Editing**

**The trap:**
```json
{"edit_json": {"file": "~/.config/app/settings.json", "path": "$.apiKey", "value": "..."}}

// Requires knowing:
- JSON parsing/editing (use jq? implement parser?)
- YAML editing (different parser)
- TOML editing (different parser)
- INI files (different parser)
- Path conventions (~/.config vs ~/Library vs %APPDATA%)
- File permissions (chmod on Unix, ACLs on Windows)
- Atomic writes (don't corrupt on failure)
- Backup/rollback (in case of errors)
```

**Why it's a trap:**
1. **Every config format is different** - JSON, YAML, TOML, INI, XML, HCL, etc.
2. **Path conventions differ** - macOS vs Linux vs Windows
3. **Parsing complexity** - Maintaining parsers for N formats
4. **Tool proliferation** - jq for JSON, yq for YAML, tomlq for TOML...
5. **Edge cases** - Comments, formatting preservation, merge conflicts

**What to do instead:**
```json
// Use existing tools via commands
{
  "name": "Set API key in config",
  "command": "jq '.apiKey = \"$API_KEY\"' ~/.config/app/settings.json > /tmp/temp.json && mv /tmp/temp.json ~/.config/app/settings.json"
}

// Or use sed for simple cases
{
  "name": "Set API key",
  "command": "sed -i.bak 's/\"apiKey\": \".*\"/\"apiKey\": \"'$API_KEY'\"/' ~/.config/app/settings.json"
}
```

**Or better: Use facts + templates:**
```json
{
  "facts": {
    "api_key": {
      "command": "read -p 'Enter API key: ' key && echo $key",
      "export": "API_KEY"
    }
  },
  "install_steps": [
    {
      "name": "Write config file",
      "command": "cat > ~/.config/app/settings.json << EOF\n{\"apiKey\": \"{{ .api_key }}\"}\nEOF"
    }
  ]
}
```

**Verdict: DON'T BUILD. Use jq/sed via commands, or templates.**

---

### ❌ **TRAP: File Downloads**

**The trap:**
```json
{"download": {"url": "https://...", "dest": "/tmp/file", "checksum": "sha256:..."}}

// Seems simple, but:
- Which HTTP client? (curl vs wget vs native Go)
- SSL/TLS certificates (different per OS)
- Proxy support (corporate networks)
- Progress bars (terminal detection)
- Resumable downloads (range requests)
- Checksum verification (sha256, sha512, md5, gpg?)
- Retries on failure (exponential backoff)
- Timeouts (network-dependent)
```

**Why it's a trap:**
1. **HTTP clients exist** - curl/wget already on every system
2. **Certificate management** - Let OS handle it
3. **Proxy configuration** - Already in environment
4. **Progress indication** - Curl/wget have better UX
5. **Feature creep** - Soon you're rebuilding curl

**What to do instead:**
```json
// Use curl/wget (already installed everywhere)
{
  "name": "Download extension",
  "command": "curl -fsSL https://github.com/.../extension.vsix -o /tmp/extension.vsix"
}

// With checksum verification
{
  "name": "Verify checksum",
  "command": "echo 'sha256-hash /tmp/extension.vsix' | shasum -a 256 -c -"
}
```

**Verdict: DON'T BUILD. Use curl/wget via commands.**

---

### ❌ **TRAP: Secrets/Keychain Management**

**The trap:**
```json
{"store_secret": {"name": "api_key", "value": "...", "service": "myapp"}}

// Requires platform-specific backends:
- macOS: Keychain Services API (Security.framework)
- Linux: Secret Service API (libsecret, gnome-keyring, kwallet)
- Windows: Credential Manager (wincred.dll)
- Headless Linux: Where to store? (gpg? pass? vault?)
```

**Why it's a trap:**
1. **Platform-specific APIs** - Completely different per OS
2. **Desktop environment dependency** - GNOME vs KDE use different backends
3. **Headless scenarios** - Servers have no keychain
4. **CGO required** - Linking to system libraries (breaks cross-compile)
5. **Security audit** - Getting this wrong = vulnerability

**What to do instead:**

**Option 1: Environment variables**
```json
{
  "facts": {
    "api_key": {
      "command": "echo $CLAUDE_API_KEY",
      "export": "CLAUDE_API_KEY",
      "required": true
    }
  }
}
```

**Option 2: User's existing keychain CLI**
```json
{
  "name": "Store in macOS keychain",
  "command": "security add-generic-password -a $(whoami) -s 'Claude API' -w '$API_KEY'"
}

{
  "name": "Store in Linux keychain",
  "command": "secret-tool store --label='Claude API' service claude key api_key"
}
```

**Option 3: Let user choose tool (pass, 1password CLI, vault, etc.)**

**Verdict: DON'T BUILD. Use environment variables or existing keychain CLIs.**

---

### ❌ **TRAP: Container/VM Orchestration**

**The trap:**
```json
{"create_vm": {"os": "ubuntu-24.04", "ram": "4GB", "cpus": 2}}

// Requires knowing:
- Lima vs Colima vs Docker Desktop vs Multipass vs VirtualBox
- Image formats (qcow2, vmdk, vdi)
- Networking modes (bridge, NAT, host)
- Storage drivers (overlay2, btrfs, zfs)
- Init systems (cloud-init, ignition)
- Platform differences (Apple Virtualization.framework vs QEMU vs Hyper-V)
```

**Why it's a trap:**
1. **Massive scope** - Each VM tool is thousands of LOC
2. **Rapidly changing** - Lima, Colima, Rancher Desktop all evolve fast
3. **Platform-specific** - Apple Virtualization.framework vs QEMU
4. **Opinionated** - Choosing Lima over Colima is contentious
5. **Complex state** - VM lifecycle management is hard

**What to do instead:**
```json
// Just call the VM tool's CLI
{
  "name": "Create Ubuntu VM with Lima",
  "command": "limactl create --name=dev --cpus=2 --memory=4 template://ubuntu-lts"
}

{
  "name": "Start VM",
  "command": "limactl start dev"
}
```

**Verdict: DON'T BUILD. Use Lima/Colima/Multipass CLIs via commands.**

---

## The Pattern: What SHOULD Be In Core?

### ✅ **Core Capabilities (Platform-Agnostic)**

1. **Command execution** ✅ (already have)
2. **Platform detection** ✅ (already have)
3. **Facts gathering** ✅ (already have)
4. **Idempotent checks** ✅ (already have)
5. **Template interpolation** ✅ (already have)
6. **Dry-run mode** ✅ (already have)
7. **Interactive prompts** 🟡 (SHOULD ADD - see below)
8. **Exit on error** ✅ (already have)

### 🟡 **Safe to Add: Interactive Prompts**

**Why it's safe:**
- Pure Go stdlib (`bufio` + `golang.org/x/term`)
- Works on all platforms Go supports
- No CGO, no system libraries
- Minimal code (~30 LOC)
- No platform-specific invariants

**Add as new step type:**
```json
{
  "name": "Get API key",
  "prompt": {
    "message": "Enter your Claude API key: ",
    "store_as": "api_key",
    "secret": true
  }
}
```

**Implementation (30 LOC):**
```go
type PromptStep struct {
    Message  string
    StoreAs  string  // Store in facts
    Secret   bool    // Use term.ReadPassword if true
}

func (e *Executor) executePrompt(step PromptStep) (string, error) {
    fmt.Print(step.Message)
    
    if step.Secret {
        password, err := term.ReadPassword(int(os.Stdin.Fd()))
        fmt.Println() // newline after password
        return string(password), err
    }
    
    reader := bufio.NewReader(os.Stdin)
    line, err := reader.ReadString('\n')
    return strings.TrimSpace(line), err
}
```

**This doesn't trap Sink into platform specifics.** ✅

---

## The Caddy Pattern: Plugins for Platform-Specific Stuff

### **What Caddy Does Right**

**Core Caddy:**
- HTTP server (platform-agnostic)
- Config parsing (platform-agnostic)
- Plugin system (platform-agnostic)
- ~10K LOC core

**Plugins add:**
- Cloudflare DNS (Cloudflare-specific API)
- Route53 DNS (AWS-specific API)
- PostgreSQL storage (database-specific)
- Redis cache (Redis-specific)

**Sink should follow this:**

**Core Sink:** (~1,500 LOC today)
- Command execution ✅
- Platform detection ✅
- Facts system ✅
- Idempotent checks ✅
- Interactive prompts (add 30 LOC)
- **Plugin registry** (add 100 LOC)

**Plugins handle platform-specific stuff:**
- `sink-plugin-brew` - Homebrew abstraction
- `sink-plugin-apt` - Debian/Ubuntu packages
- `sink-plugin-vscode` - VS Code extension management
- `sink-plugin-lima` - VM orchestration
- `sink-plugin-keychain` - Secret storage
- `sink-plugin-systemd` - Unit file generation

**User picks plugins they need:**
```bash
# Install only what you need
sink plugin install brew vscode

# Now configs can use plugin steps
{
  "name": "Install VS Code",
  "plugin": "vscode.install"
}
```

---

## What This Means: The 7-Day Validation Plan

### **DON'T BUILD ANYTHING NEW (except prompts)**

**Week 1: Validate with RAW commands**

Create 5 AI tool configs using ONLY existing features:

1. **claude-desktop.json**
```json
{
  "version": "1.0.0",
  "facts": {
    "api_key": {
      "command": "read -p 'Enter Claude API key: ' key && echo $key",
      "export": "CLAUDE_API_KEY"
    }
  },
  "platforms": [
    {
      "os": "darwin",
      "install_steps": [
        {
          "name": "Check Homebrew",
          "check": "command -v brew",
          "error": "Homebrew required. Install from https://brew.sh"
        },
        {
          "name": "Install VS Code",
          "check": "command -v code",
          "on_missing": [
            {"name": "Install", "command": "brew install --cask visual-studio-code"}
          ]
        },
        {
          "name": "Install Claude extension",
          "command": "code --install-extension anthropics.claude-vscode",
          "check": "code --list-extensions | grep -q anthropics.claude-vscode"
        },
        {
          "name": "Configure API key",
          "command": "code --user-data-dir ~/.config/Code/User --install-extension ...",
          "message": "Setting up Claude API key..."
        }
      ]
    }
  ]
}
```

2. **github-copilot.json**
3. **cursor.json**
4. **windsurf.json**
5. **aider.json**

**These work TODAY.** Ship them. See if anyone uses them.

### **Add ONE Feature: Interactive Prompts**

**Only if feedback says:** "Typing `read -p` is annoying"

Then add proper prompt support (30 LOC).

### **Plugins come LATER**

**Only after validation succeeds:**
1. People use the configs (repeat usage)
2. Common patterns emerge (package installs, extension management)
3. Community asks for abstractions

**Then build plugin system** (100 LOC) and let community build plugins.

---

## The Key Insight: You Already Have a Registry

> "With commands I can still access a registry using the existing registry providers can't i?"

**EXACTLY.** You don't need to build a package manager abstraction because:

```json
// Want Homebrew? Use Homebrew.
{"command": "brew install colima"}

// Want apt? Use apt.
{"command": "apt-get install colima"}

// Want snap? Use snap.
{"command": "snap install colima --classic"}

// Want cargo? Use cargo.
{"command": "cargo install colima"}

// Want go install? Use go install.
{"command": "go install github.com/abiosoft/colima@latest"}

// Want curl + install? Use that.
{"command": "curl -L ... | tar xz && mv colima /usr/local/bin/"}
```

**The registry IS THE PACKAGE MANAGER.** You're just orchestrating it.

---

## Decision Matrix

| Feature | Platform-Specific? | In Core? | Alternative |
|---------|-------------------|----------|-------------|
| **Interactive prompts** | ❌ No (pure Go) | ✅ YES (30 LOC) | `read -p` hack |
| Package manager abstraction | ✅ YES (endless) | ❌ NO | Use commands |
| VS Code extensions | ✅ YES (paths vary) | ❌ NO | Use `code` CLI |
| JSON editing | ✅ YES (many formats) | ❌ NO | Use jq/sed |
| File downloads | ✅ YES (SSL/proxy) | ❌ NO | Use curl/wget |
| Secrets storage | ✅ YES (per-OS APIs) | ❌ NO | Use env vars |
| VM orchestration | ✅ YES (huge scope) | ❌ NO | Use Lima/Colima CLI |

---

## The Plan

### **Today (Add Prompts - 30 LOC)**
```go
// Add PromptStep to types.go
type PromptStep struct {
    Message  string
    StoreAs  string
    Secret   bool
}

// Add to executor.go
func (e *Executor) executePrompt(step PromptStep) StepResult {
    // Implementation here
}
```

### **Week 1 (Validate Use Case)**
- Build 5 AI tool configs with raw commands
- Post in Discord/X/GitHub
- Measure success rate
- Collect feedback

### **Week 2 (Iterate or Pivot)**
- If successful: Polish UX, add to registry
- If not: Try MCP servers or VM desktops
- Learn what's actually painful

### **Month 2+ (Plugins if needed)**
- Only if common patterns emerge
- Build plugin system (Caddy-style)
- Let community build platform-specific plugins

---

## The Answer to Your Question

> "Which of these are going to trap sink into ingesting OS or platform specific invariants?"

**ALL OF THEM except interactive prompts.**

- ❌ Package managers → Trap
- ❌ VS Code extensions → Trap  
- ❌ Config file editing → Trap
- ❌ File downloads → Already solved (curl/wget)
- ❌ Secrets management → Trap
- ❌ VM orchestration → Trap
- ✅ Interactive prompts → Safe (pure Go)

**Your instinct is correct:**
> "That would seem to call more for a plugin pattern (caddy doesn't ship with support for cloudflare but it can be added, etc)."

**YES.** Platform-specific stuff = plugins. Core stays lean.

---

## The Brutal Truth

You don't need to build ANY of this to validate.

**Just make 5 configs with raw commands and see if anyone uses them.**

If they do → You found product-market fit → Then add polish.

If they don't → You avoided building features nobody wants.

**Ship the configs. Validate first. Build second.**
