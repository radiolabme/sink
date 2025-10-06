# Data Directory

This directory contains reference configurations and the JSON schema.

## Schema Files

### sink.schema.json

**This file is a copy of the canonical schema from `src/sink.schema.json`.**

The schema is embedded in the Sink binary at compile time. To get the schema:

```bash
# Output embedded schema to stdout
sink schema > sink.schema.json

# Use in your editor
sink schema > ~/.config/sink/sink.schema.json
```

The canonical source is `src/sink.schema.json` which is embedded into the binary using Go's `//go:embed` directive.

## Reference Configurations

- `demo-config.json` - Simple demo configuration
- `install-config.json` - Full installation example
- `install-config-with-facts.json` - Example with facts system
- `vps-sizes.json` - VPS sizing reference data

## Using the Schema

In your configuration files, reference the schema for editor autocompletion and validation:

```json
{
  "$schema": "https://raw.githubusercontent.com/radiolabme/sink/main/src/sink.schema.json",
  "version": "1.0.0",
  "platforms": [...]
}
```

Or use a local copy:

```json
{
  "$schema": "./sink.schema.json",
  "version": "1.0.0",
  "platforms": [...]
}
```

Or use the emitted schema:

```bash
sink schema > sink.schema.json
```

## Schema Versioning

The schema `$id` URL points to the GitHub raw content URL:

```
https://raw.githubusercontent.com/radiolabme/sink/main/src/sink.schema.json
```

This ensures:
- The schema is always accessible
- Version pinning is possible via branch/tag: `.../v0.1.0/src/sink.schema.json`
- No external hosting required
- Automatic updates when main branch changes

For version-specific schemas, use git tags in the URL:

```
https://raw.githubusercontent.com/radiolabme/sink/v0.1.0/src/sink.schema.json
```
