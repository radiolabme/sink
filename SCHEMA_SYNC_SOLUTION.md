# Schema Synchronization Solution

## Problem

The CI test detected that the schema file (`src/sink.schema.json`) and the embedded schema in the binary (`bin/sink`) were out of sync. This happened because the schema file was modified at 11:35, but the binary was built at 11:31 (4 minutes earlier).

## Immediate Solution

```bash
# Rebuild the binary to embed the latest schema
make build

# Verify synchronization
make verify-schema
```

## Long-term Prevention

We've implemented multiple layers of protection to prevent this from happening again:

### 1. Automated Test Suite

**New Test: `TestSchemaSynchronization`** (`src/schema_test.go`)
- Compares the schema file with the embedded schema byte-by-byte
- Fails with clear instructions if they differ
- Runs automatically as part of `make test`

**New Test: `TestSchemaHasRequiredNewProperties`** (`src/schema_test.go`)
- Verifies new code features have schema definitions
- Checks `fact`, `remediation_step`, and `install_step` types
- Ensures properties like `verbose`, `sleep`, `timeout` are present

### 2. Make Target

**New Target: `make verify-schema`**
- Runs schema synchronization tests
- Can be used as a pre-commit hook
- Provides quick verification during development

### 3. CI Pipeline Enhancement

**Updated: `.github/workflows/ci.yml`**
- Added `make verify-schema` as the first step in `schema-validation` job
- Runs on every push and pull request
- Prevents merging code with out-of-sync schemas

### 4. Documentation

**New Guide: `docs/schema-synchronization.md`**
- Comprehensive explanation of schema synchronization
- Troubleshooting guide for common issues
- Best practices for schema maintenance
- Pre-commit checklist

**Updated: `README.md`**
- Added "Schema Synchronization" section
- Links to detailed documentation
- Explains the automated safeguards

## How It Works

```
┌─────────────────────────────────────────────┐
│          Schema Change Workflow              │
└─────────────────────────────────────────────┘

1. Developer edits src/sink.schema.json
2. Developer runs: make build
3. Build embeds schema into src/schema.go
4. Developer runs: make verify-schema
5. Tests confirm synchronization ✅
6. CI pipeline re-verifies on push
7. Merge blocked if out of sync ❌
```

## Developer Workflow

### When Editing the Schema

```bash
# 1. Edit the schema
vim src/sink.schema.json

# 2. Rebuild (embeds the schema)
make build

# 3. Verify synchronization
make verify-schema

# 4. Update reference copy
cp src/sink.schema.json data/sink.schema.json

# 5. Test everything
make test
```

### When Adding Code Features

```bash
# 1. Add property to types.go
# 2. Add property to sink.schema.json
# 3. Rebuild and verify
make build
make verify-schema

# 4. Update reference copy
cp src/sink.schema.json data/sink.schema.json
```

## Verification Commands

```bash
# Quick verification
make verify-schema

# Manual diff (should show no differences)
diff <(jq -S . src/sink.schema.json) <(./bin/sink schema | jq -S .)

# Check timestamps
ls -lh src/sink.schema.json bin/sink
# Binary should be newer than schema file

# Run specific tests
go test ./src/... -run TestSchemaSynchronization -v
go test ./src/... -run TestSchemaHasRequiredNewProperties -v
```

## Files Modified

1. **src/schema_test.go**
   - Added `TestSchemaSynchronization()` - verifies file vs embedded schema
   - Added `TestSchemaHasRequiredNewProperties()` - verifies feature properties
   - Added helper function `compareSchemaProperties()` for detailed diff reporting

2. **Makefile**
   - Added `verify-schema` target
   - Updated `.PHONY` list
   - Updated help documentation

3. **.github/workflows/ci.yml**
   - Added `make verify-schema` step to `schema-validation` job
   - Ensures tests run before other schema checks

4. **docs/schema-synchronization.md**
   - New comprehensive guide (100+ lines)
   - Covers architecture, troubleshooting, best practices
   - Includes pre-commit checklist

5. **README.md**
   - Added "Schema Synchronization" subsection
   - Links to detailed documentation

## Test Results

All new tests passing:

```
✅ TestSchemaSynchronization - Verifies file/embedded sync
✅ TestSchemaHasRequiredNewProperties - Verifies feature coverage
✅ TestSchemaEmbed - Verifies basic embedding
✅ TestSchemaCommand - Verifies schema output
✅ All 270+ tests passing
```

## CI Protection

The CI pipeline now includes:

1. **Build** → Binary with embedded schema
2. **Test** → `make verify-schema` runs synchronization tests
3. **Diff** → Compares file vs embedded schema
4. **Validate** → Checks all three schema copies match

Any mismatch will fail the build and prevent merging.

## Benefits

✅ **Prevents out-of-sync issues** - Tests catch problems immediately
✅ **Clear error messages** - Tells developers exactly how to fix issues
✅ **Automated enforcement** - CI blocks merges with schema problems
✅ **Self-documenting** - Tests serve as executable documentation
✅ **Fast feedback** - `make verify-schema` runs in <1 second
✅ **Feature coverage** - Ensures new properties have schema definitions

## Future Enhancements

Potential improvements for even better synchronization:

1. **Git pre-commit hook** - Automatically run `make verify-schema`
2. **Schema generation** - Generate schema from Go types (or vice versa)
3. **Watch mode** - Auto-rebuild when schema changes during development
4. **Schema versioning** - Track schema versions and migrations
5. **Property validation** - Ensure Go struct tags match schema definitions

## Summary

The immediate issue (schema out of sync) is fixed with `make build`. Going forward, multiple automated safeguards prevent this from happening again:

- Test suite catches sync issues
- Make target provides quick verification
- CI pipeline blocks bad merges
- Documentation guides proper workflow

Run `make verify-schema` anytime to verify everything is synchronized!
