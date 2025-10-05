# Sink Configuration Schema

## Current Schema Overview

Our schema has evolved to support **declarative fact gathering** while maintaining type safety for installation steps.

## Key Components

### 1. **Facts System** (NEW)

Declarative system for gathering system information that can be:
- Exported as environment variables
- Used in templates throughout the config
- Platform-filtered (only gather on specific OSes)
- Type-safe (string, boolean, integer)
- Transformed (map values, e.g., x86_64 → amd64)

#### Fact Definition Structure:
```json
{
  "facts": {
    "fact_name": {
      "command": "shell command to run",
      "description": "Human description",
      "export": "ENV_VAR_NAME",
      "platforms": ["darwin", "linux", "windows"],
      "type": "string|boolean|integer",
      "transform": {
        "input_value": "output_value"
      },
      "strict": true,
      "required": false
    }
  }
}
```

#### Example Facts:
```json
{
  "facts": {
    "os": {
      "command": "uname -s | tr '[:upper:]' '[:lower:]'",
      "export": "SINK_OS"
    },
    "arch": {
      "command": "uname -m",
      "transform": {
        "x86_64": "amd64",
        "aarch64": "arm64"
      },
      "export": "SINK_ARCH"
    },
    "has_brew": {
      "command": "command -v brew >/dev/null && echo true || echo false",
      "type": "boolean",
      "platforms": ["darwin"],
      "export": "SINK_HAS_BREW"
    }
  }
}
```

### 2. **Platform Definitions**

Two variants supported via `oneOf`:

#### Variant A: Simple Platform (e.g., macOS, Windows)
```json
{
  "os": "darwin",
  "match": "darwin*",
  "name": "macOS",
  "required_tools": ["brew"],
  "install_steps": [...]
}
```

#### Variant B: Distribution-Based Platform (Linux)
```json
{
  "os": "linux",
  "match": "linux*",
  "name": "Linux",
  "distributions": [
    {
      "ids": ["ubuntu", "debian"],
      "name": "Ubuntu/Debian",
      "install_steps": [...]
    }
  ],
  "fallback": {
    "error": "Unsupported distro: {distro}"
  }
}
```

### 3. **Installation Steps**

Four step types enforced via `oneOf`:

#### Type 1: Command Step
```json
{
  "name": "Install package",
  "command": "brew install colima",
  "message": "Installing...",  // optional
  "error": "Failed to install"  // optional
}
```

#### Type 2: Check-with-Error Step
```json
{
  "name": "Check Homebrew",
  "check": "command -v brew",
  "error": "Homebrew required"
}
```

#### Type 3: Check-with-Remediation Step
```json
{
  "name": "Check or install snapd",
  "check": "command -v snap",
  "on_missing": [
    {
      "name": "Update apt",
      "command": "sudo apt-get update",
      "error": "Failed to update"
    },
    {
      "name": "Install snapd",
      "command": "sudo apt-get install -y snapd",
      "error": "Failed to install snapd"
    }
  ]
}
```

#### Type 4: Error-Only Step
```json
{
  "name": "Windows not supported",
  "error": "Colima requires WSL2 or Linux VM"
}
```

## Type Safety Features

### Impossible States Made Unrepresentable:

1. ✅ **Step cannot have both `command` and `check`**
   - Enforced via `oneOf` with mutually exclusive `required` fields

2. ✅ **Step cannot have `on_missing` without `check`**
   - Enforced via specific variant requiring both

3. ✅ **Platform must have either `install_steps` OR `distributions`**
   - Enforced via `oneOf` at platform level

4. ✅ **Export var names must be valid shell variables**
   - Enforced via regex: `^[A-Z_][A-Z0-9_]*$`

5. ✅ **Fact names must be valid identifiers**
   - Enforced via regex: `^[a-z_][a-z0-9_]*$`

6. ✅ **Platform filters must be valid OSes**
   - Enforced via enum: `["darwin", "linux", "windows"]`

7. ✅ **Transform only allowed for string types**
   - Enforced via nested `oneOf` in fact definition

### Runtime Validations Needed:

These cannot be enforced by JSON Schema alone:

1. **Template variable references** must exist
2. **Circular dependencies** in facts must be detected
3. **Undefined transform values** (with `strict: false`) are warnings

## File Structure

```
sink/
├── install-config.schema.json          # Original schema (237 lines)
├── install-config-enhanced.schema.json # New schema with facts (~350 lines)
├── install-config.json                 # Original config (116 lines)
└── install-config-with-facts.json      # Enhanced config (~200 lines)
```

## Schema Statistics

### Original Schema:
- **237 lines** - Installation steps only
- **4 step types** enforced via `oneOf`
- **2 platform types** enforced via `oneOf`
- No fact gathering

### Enhanced Schema:
- **~350 lines** - Adds facts system (+~110 lines)
- **11 fact definitions** in example
- **4 fact type variants** enforced via `oneOf`
- **Backward compatible** - `facts` is optional

## Usage Examples

### Gather Facts:
```bash
# Get facts as JSON
sink facts

# Export as environment variables
eval $(sink facts --export)

# Export to dotenv file
sink facts --format dotenv > .env

# Export to KV store
sink facts --kv-store etcd://localhost:2379/prefix
```

### Execute with Facts:
```bash
# Local execution (gathers facts automatically)
sink execute install-config-with-facts.json

# Remote execution (gathers facts on remote host)
sink execute install-config-with-facts.json --host user@server

# Use pre-gathered facts
sink execute install-config.json --facts facts.json
```

### REST API:
```bash
# Get facts
curl http://localhost:8080/facts

# Get facts from remote host
curl http://localhost:8080/facts?host=user@server

# Execute with fact gathering
curl -X POST http://localhost:8080/execute \
  -d '{"config": "install-config-with-facts.json"}'
```

## Next Steps

The schema is now ready. Implementation will be ~1000 LOC Go:

```
sink/
├── main.go       # CLI (~100 LOC)
├── config.go     # JSON parsing (~150 LOC)
├── facts.go      # Fact gathering (~150 LOC)
├── executor.go   # Step execution (~200 LOC)
├── transport.go  # Local/SSH (~150 LOC)
├── server.go     # REST API (~200 LOC)
└── types.go      # Domain types (~50 LOC)
```

Total: ~1000 LOC of type-safe, dependency-free Go code.
