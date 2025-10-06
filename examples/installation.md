# Installation

Sink is distributed as a single binary with no dependencies. Choose your preferred installation method below.

## Official Installation Methods

### Binary Download

Download the latest binary from the [releases page](https://github.com/your-org/sink/releases) and add to your `$PATH`.

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/your-org/sink/releases/latest/download/sink-darwin-arm64 -o sink
chmod +x sink
sudo mv sink /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -L https://github.com/your-org/sink/releases/latest/download/sink-darwin-amd64 -o sink
chmod +x sink
sudo mv sink /usr/local/bin/
```

**Linux (x86_64):**
```bash
curl -L https://github.com/your-org/sink/releases/latest/download/sink-linux-amd64 -o sink
chmod +x sink
sudo mv sink /usr/local/bin/
```

**Linux (ARM64):**
```bash
curl -L https://github.com/your-org/sink/releases/latest/download/sink-linux-arm64 -o sink
chmod +x sink
sudo mv sink /usr/local/bin/
```

### Install Script

Use our install script for automated installation:

```bash
curl -sSL https://raw.githubusercontent.com/your-org/sink/main/install.sh | sh
```

By default, it installs to `./bin`. To install system-wide:

```bash
curl -sSL https://raw.githubusercontent.com/your-org/sink/main/install.sh | sudo sh -s -- --prefix=/usr/local
```

### Homebrew (macOS/Linux)

```bash
brew tap your-org/sink
brew install sink
```

Or install from the official Homebrew repository:

```bash
brew install sink
```

## Build From Source

### Requirements

- Go 1.21 or later
- Git

### Using Go Install

```bash
go install github.com/your-org/sink/cmd/sink@latest
```

### From Source

```bash
git clone https://github.com/your-org/sink.git
cd sink
make build
sudo make install
```

This will install `sink` to `/usr/local/bin` by default.

## Verify Installation

After installation, verify Sink is working:

```bash
sink version
```

You should see output like:

```
sink version 0.1.0
```

## System Requirements

- **Minimum**: Any OS with `/bin/sh` (POSIX shell)
- **Recommended**: Bash or Zsh for advanced features
- **Platforms**: macOS, Linux, FreeBSD, WSL2

### Supported Platforms

| OS | Architecture | Support |
|----|--------------|---------|
| macOS | x86_64, ARM64 | ✅ Full |
| Linux | x86_64, ARM64, ARM | ✅ Full |
| FreeBSD | x86_64 | ✅ Full |
| Windows | WSL2 | ⚠️ Via WSL2 only |

### Supported Linux Distributions

Sink automatically detects:
- Ubuntu / Debian
- Fedora / RHEL / CentOS
- Alpine Linux
- Arch Linux
- Any distribution with `/etc/os-release`

## Shell Compatibility

Sink uses `/bin/sh` (POSIX shell) by default, ensuring maximum compatibility:

- ✅ Bash
- ✅ Zsh
- ✅ Dash
- ✅ Ash (Alpine)
- ✅ POSIX-compliant shells

Commands in your configuration files should use POSIX shell syntax for maximum portability.

## Configuration Directory

Sink stores runtime data in:
- **macOS/Linux**: `~/.cache/sink/`
- **Custom**: Set `SINK_CACHE_DIR` environment variable

## Uninstallation

### Binary Installation

```bash
sudo rm /usr/local/bin/sink
rm -rf ~/.cache/sink
```

### Homebrew

```bash
brew uninstall sink
rm -rf ~/.cache/sink
```

### Go Install

```bash
rm $(which sink)
rm -rf ~/.cache/sink
```

## Troubleshooting

### Binary not found after installation

Add `/usr/local/bin` to your `$PATH`:

**Bash** (`~/.bashrc` or `~/.bash_profile`):
```bash
export PATH="/usr/local/bin:$PATH"
```

**Zsh** (`~/.zshrc`):
```zsh
export PATH="/usr/local/bin:$PATH"
```

### Permission denied

If you see "permission denied" errors, ensure the binary is executable:

```bash
chmod +x /usr/local/bin/sink
```

### WSL2 Installation

On Windows, install Sink inside WSL2, not in Windows directly:

```bash
# Inside WSL2
curl -L https://github.com/your-org/sink/releases/latest/download/sink-linux-amd64 -o sink
chmod +x sink
sudo mv sink /usr/local/bin/
```

## Next Steps

- **[Getting Started](getting-started.md)** - Create your first configuration
- **[Usage Guide](usage-guide.md)** - Learn all Sink patterns
- **[Examples](.)** - Browse real-world configurations

---

[← Back to Docs](README.md) | [Next: Getting Started →](getting-started.md)
