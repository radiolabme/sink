# Schema Consolidation - October 4, 2025

## Overview

Consolidated two redundant schema files into a single, comprehensive schema that supports all configuration features.

## What Changed

### Before
- `install-config.schema.json` - Basic schema (no facts support, wrong metadata)
- `install-config-enhanced.schema.json` - Full schema with facts support

### After
- `install-config.schema.json` - Single consolidated schema with all features

## Schema Features

The consolidated schema (`install-config.schema.json`) supports:

### Core Features (Always Available)
- **Platform Detection**: Multi-platform configurations (macOS, Linux, Windows)
- **Distribution Support**: Linux distribution-specific steps
- **Check-and-Remediate**: Idempotent installation steps
- **Defaults**: Default values across platforms
- **Fallbacks**: Error messages for unsupported platforms

### Optional Features
- **Facts System**: Declarative fact gathering (optional)
  - Environment variables
  - Command outputs
  - File contents
  - Type transforms (string, boolean, integer)
  - Platform-specific facts
  - Required vs optional facts

## Backward Compatibility

The consolidated schema is **100% backward compatible**:

- Configs without facts still work (facts property is optional)
- `install-config.json` - No facts, works perfectly
- `demo-config.json` - Uses facts, works perfectly
- `install-config-with-facts.json` - Full facts example, works perfectly

## Schema Metadata

```json
{
  "$id": "https://github.com/brian/sink/install-config.schema.json",
  "title": "Sink Configuration Schema",
  "description": "Schema for defining multi-platform installation configurations with declarative fact gathering"
}
```

## Configuration Files

All configs now reference the single schema:

| File | Schema Reference | Uses Facts |
|------|-----------------|------------|
| `install-config.json` | `./install-config.schema.json` | No |
| `demo-config.json` | `./install-config.schema.json` | Yes (1 fact) |
| `install-config-with-facts.json` | `./install-config.schema.json` | Yes (6 facts) |

## Benefits

1. **Single Source of Truth**: One schema to maintain
2. **Correct Metadata**: Fixed "Colima/koto" references to "Sink"
3. **Feature Complete**: All features in one schema
4. **Backward Compatible**: Existing configs work without changes
5. **Simpler for Users**: No confusion about which schema to use
6. **Optional Facts**: Can use facts or not, same schema

## Schema Structure

```
install-config.schema.json
├── properties
│   ├── version (required)
│   ├── platforms (required)
│   ├── facts (optional) ← Key difference from old basic schema
│   ├── defaults (optional)
│   └── fallback (optional)
└── $defs
    ├── fact ← Comprehensive fact definition
    ├── platform
    ├── distribution
    ├── install_step
    ├── remediation_step
    └── fallback
```

## Validation

All configs validated successfully:

```bash
# Test basic config (no facts)
./bin/sink execute data/install-config.json --dry-run
# ✅ Works

# Test demo config (with facts)
./bin/sink execute data/demo-config.json --dry-run
# ✅ Works - gathered 1 fact

# Test full facts config
./bin/sink execute data/install-config-with-facts.json --dry-run
# ✅ Works - gathered 6 facts
```

## Documentation Updates

Updated references in:
- `README.md` - Project structure shows single schema
- `docs/QUICK_REFERENCE.md` - File locations table
- `docs/REORGANIZATION.md` - Directory structure
- `docs/SCHEMA_CONSOLIDATION.md` - This document

## For Developers

When creating new configs, always use:

```json
{
  "$schema": "./install-config.schema.json",
  "version": "1.0.0",
  "platforms": [...]
}
```

### With Facts (Optional)
```json
{
  "$schema": "./install-config.schema.json",
  "version": "1.0.0",
  "facts": {
    "my_fact": {
      "command": "echo hello"
    }
  },
  "platforms": [...]
}
```

### Without Facts (Also Valid)
```json
{
  "$schema": "./install-config.schema.json",
  "version": "1.0.0",
  "platforms": [...]
}
```

Both are valid against the same schema!

## Migration Guide

If you have external configs referencing the old schemas:

1. **Old basic schema**: Change `install-config.schema.json` references
   - No change needed! File still exists, just updated content

2. **Old enhanced schema**: Change `install-config-enhanced.schema.json` references
   - Find: `"$schema": "./install-config-enhanced.schema.json"`
   - Replace: `"$schema": "./install-config.schema.json"`

## Technical Details

The consolidation involved:

1. Deleted `install-config.schema.json` (outdated metadata)
2. Renamed `install-config-enhanced.schema.json` → `install-config.schema.json`
3. Updated all config file `$schema` references
4. Verified all configs still work
5. Updated documentation

## Conclusion

We now have a single, comprehensive schema that supports all features while maintaining backward compatibility with existing configurations. The schema is properly named and attributed to the Sink project.
