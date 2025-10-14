# Sink Usage Guide

Comprehensive examples demonstrating Sink patterns for real-world system setup tasks.

## Quick Start

Preview a configuration in dry-run mode:

```bash
sink execute examples/01-basic.json --dry-run
```

Execute after reviewing:

```bash
sink execute examples/01-basic.json
```

## Examples Overview

Sink provides focused examples demonstrating individual concepts. See `examples/FAQ.md` for detailed documentation of each pattern.

### Core Patterns

| Example | Purpose | Key Concepts |
|---------|---------|--------------|
| `01-basic.json` | Simplest configuration | Single platform, check steps, error messages |
| `02-multi-platform.json` | Cross-platform support | Multiple OS blocks, platform selection, fallbacks |
| `03-distributions.json` | Linux distributions | Distribution detection, package manager abstraction |
| `04-facts.json` | System information | Fact gathering, type coercion, template substitution |
| `05-nested-steps.json` | Conditional execution | check/on_missing pattern, idempotency |
| `06-retry.json` | Service readiness | Retry logic, timeouts, polling |
| `07-defaults.json` | Reusable values | Default values, DRY principle, templates |
| `08-error-handling.json` | Error patterns | Check-only, error-only, remediation |

### Bootstrap (Remote Configs)

| Example | Purpose | Key Concepts |
|---------|---------|--------------|
| `bootstrap-https-url.json` | HTTPS loading | Remote configs, TLS validation, checksums |
| `bootstrap-github-pinned.json` | GitHub pinning | Version pinning, immutable refs, security |

For detailed documentation of each example, see `examples/FAQ.md`.

## Bootstrap Command

The `bootstrap` command loads configurations from remote URLs:

```bash
# GitHub with version pinning (recommended)
sink bootstrap https://raw.githubusercontent.com/org/repo/v1.0.0/config.json

# HTTPS URL with auto-checksum
sink bootstrap https://example.com/config.json

# HTTP with explicit checksum
sink bootstrap http://example.com/config.json --sha256 abc123...
```

See `examples/FAQ.md` for complete bootstrap documentation and security best practices.

## Getting Help

- **Examples & FAQ**: `examples/FAQ.md` - Complete guide with examples
- **Schema**: `sink schema` or `src/sink.schema.json`
- **Command Help**: `sink help <command>`
- **Architecture**: `docs/ARCHITECTURE.md`

---

**For comprehensive usage patterns and examples, see `examples/FAQ.md`**
