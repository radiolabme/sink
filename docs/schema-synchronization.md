# Schema Synchronization

## Overview

Sink maintains a JSON Schema (`sink.schema.json`) that validates configuration files. This schema exists in three places:

1. **Source**: `src/sink.schema.json` - The canonical source of truth
2. **Embedded**: `src/schema.go` - The schema embedded in the compiled binary
3. **Reference**: `data/sink.schema.json` - A copy for external tools and editors

## Why Synchronization Matters

When the schema file is modified but the binary isn't rebuilt, or when code features are added without updating the schema, several issues can arise:

- **Configuration validation failures**: Users can't validate configs against an outdated schema
- **Missing feature documentation**: New properties aren't discoverable through schema introspection
- **CI/CD failures**: Automated validation may fail due to schema mismatches
- **Editor support breaks**: IDE autocomplete and validation use the schema file

## Automated Safeguards

### 1. Test Suite Protection

The test suite includes comprehensive schema validation:

```bash
# Run schema synchronization tests
make verify-schema

# Or run specific tests
go test ./src/... -run TestSchemaSynchronization -v
go test ./src/... -run TestSchemaHasRequiredNewProperties -v
```

**TestSchemaSynchronization** - Verifies that `src/sink.schema.json` and the embedded schema in `src/schema.go` are identical. If they differ, the test fails with clear instructions on how to fix it.

**TestSchemaHasRequiredNewProperties** - Ensures that new code features (like `verbose`, `sleep`, `timeout`) have corresponding schema definitions in all applicable types (`fact`, `remediation_step`, `install_step`).

### 2. CI Pipeline Checks

The GitHub Actions CI pipeline includes a dedicated `schema-validation` job that:

1. Builds the binary
2. Runs `make verify-schema` to test synchronization
3. Compares the embedded schema with the source file using `diff`
4. Verifies `data/sink.schema.json` matches `src/sink.schema.json`
5. Validates the schema is valid JSON

If any check fails, the entire CI pipeline fails, preventing merges of out-of-sync code.

### 3. Manual Verification

You can manually verify schema synchronization:

```bash
# Compare schemas (should produce no output if synchronized)
diff <(jq -S . src/sink.schema.json) <(./bin/sink schema | jq -S .)

# Check file timestamps
ls -lh src/sink.schema.json bin/sink
# The binary should be newer than the schema file

# Verify all three copies match
diff src/sink.schema.json data/sink.schema.json
diff src/sink.schema.json <(./bin/sink schema)
```

## How to Maintain Synchronization

### When Modifying the Schema

If you edit `src/sink.schema.json`:

```bash
# 1. Make your schema changes
vim src/sink.schema.json

# 2. Rebuild the binary to embed the new schema
make build

# 3. Copy to data directory for external tools
cp src/sink.schema.json data/sink.schema.json

# 4. Run tests to verify synchronization
make verify-schema

# 5. Run all tests to ensure nothing broke
make test
```

### When Adding Code Features

If you add new properties to Go types (`FactDef`, `CommandStep`, `RemediationStep`):

```bash
# 1. Add the property to types.go
# 2. Add the property to sink.schema.json
# 3. Update the embedded schema and data copy
make build
cp src/sink.schema.json data/sink.schema.json

# 4. Verify the feature is in the schema
go test ./src/... -run TestSchemaHasRequiredNewProperties -v

# 5. Verify synchronization
make verify-schema
```

### Pre-commit Checklist

Before committing schema-related changes:

- [ ] Schema file updated: `src/sink.schema.json`
- [ ] Binary rebuilt: `make build`
- [ ] Data copy updated: `cp src/sink.schema.json data/sink.schema.json`
- [ ] Tests pass: `make verify-schema`
- [ ] All tests pass: `make test`
- [ ] Manual diff clean: `diff <(jq -S . src/sink.schema.json) <(./bin/sink schema | jq -S .)`

## Build Process

The build process automatically embeds the schema from `src/sink.schema.json` into `src/schema.go`:

```bash
# Standard build (embeds current schema)
make build

# The build generates src/schema.go with:
# var embeddedSchema = `<contents of sink.schema.json>`
```

**Important**: The schema is embedded at build time. If you modify `sink.schema.json` without rebuilding, the binary will still have the old schema.

## Troubleshooting

### "Schema out of sync" Test Failure

```
❌ Schema file and embedded schema are OUT OF SYNC!

To fix this issue:
  1. Ensure sink.schema.json contains the correct, up-to-date schema
  2. Run: make build
  3. This will regenerate schema.go with the embedded schema from sink.schema.json
```

**Solution**:
```bash
make build
make verify-schema  # Should pass now
```

### CI Pipeline Fails on Schema Validation

**Symptom**: CI job `schema-validation` fails with "Embedded schema does not match source"

**Cause**: The schema file was modified but not committed with an updated binary, or `schema.go` wasn't regenerated.

**Solution**:
```bash
# Rebuild and update
make build
cp src/sink.schema.json data/sink.schema.json
git add bin/sink src/schema.go src/sink.schema.json data/sink.schema.json
git commit -m "Rebuild binary with updated schema"
```

### Missing Properties in Schema

**Symptom**: Test `TestSchemaHasRequiredNewProperties` fails

**Cause**: New properties added to Go types but not to the JSON schema

**Solution**:
1. Open `src/sink.schema.json`
2. Add the missing properties to the appropriate `$defs` sections
3. Rebuild and test:
   ```bash
   make build
   cp src/sink.schema.json data/sink.schema.json
   make verify-schema
   ```

## Best Practices

1. **Always rebuild after schema changes**: Run `make build` immediately after editing `sink.schema.json`

2. **Test before committing**: Run `make verify-schema` as part of your pre-commit workflow

3. **Keep copies in sync**: Remember to update `data/sink.schema.json` when changing the source schema

4. **Add schema tests for new features**: When adding new properties to Go types, add corresponding test cases to `TestSchemaHasRequiredNewProperties`

5. **Use the Makefile**: Prefer `make build` over direct `go build` commands to ensure consistency

6. **Check CI before merging**: Always verify the `schema-validation` CI job passes before merging PRs

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Schema Synchronization                    │
└─────────────────────────────────────────────────────────────┘

Source of Truth:
  src/sink.schema.json
       │
       ├─── (embedded at build time) ──→ src/schema.go ──→ bin/sink
       │
       └─── (manual copy) ──────────────→ data/sink.schema.json

Verification:
  ┌─── make verify-schema
  │     ├─ TestSchemaSynchronization (file vs embedded)
  │     └─ TestSchemaHasRequiredNewProperties (features)
  │
  ├─── CI: schema-validation job
  │     ├─ make verify-schema
  │     ├─ diff src vs embedded
  │     └─ diff src vs data
  │
  └─── Manual: diff commands
```

## Related Files

- `src/sink.schema.json` - Source schema file
- `src/schema.go` - Generated file with embedded schema
- `src/schema_test.go` - Schema validation tests
- `data/sink.schema.json` - Reference copy for external tools
- `Makefile` - Build automation (includes `verify-schema` target)
- `.github/workflows/ci.yml` - CI pipeline with schema validation

## Further Reading

- [JSON Schema Documentation](https://json-schema.org/)
- [Go embed directive](https://pkg.go.dev/embed)
- [Configuration Reference](./configuration-reference.md)
- [Development Workflow](../README.md#development-workflow)
