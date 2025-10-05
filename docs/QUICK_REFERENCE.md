# Sink Quick Reference

## Project Structure
```
sink/
├── src/    - Source code + tests (*.go, *_test.go)
├── bin/    - Binary output (sink executable)
├── data/   - Config files (*.json)
├── docs/   - Documentation (*.md)
└── test/   - (empty)
```

## Common Commands

### Build & Test
```bash
make build              # Build binary to bin/sink
make test               # Run all 116 tests
make coverage           # Show coverage report
make clean              # Remove artifacts
```

### Run
```bash
# Dry-run (no changes)
./bin/sink execute data/demo-config.json --dry-run

# Real execution (requires "yes" confirmation)
./bin/sink execute data/install-config.json

# Auto-confirm with yes
echo "yes" | ./bin/sink execute data/install-config.json
```

### Development
```bash
# Run tests
go test ./src/... -v

# Run specific test
go test ./src/... -run TestExecutorContext -v

# Generate coverage
go test ./src/... -coverprofile=test/coverage.out
go tool cover -html=test/coverage.out -o test/coverage.html

# Build
go build -o bin/sink ./src/...
```

## File Locations

| Item | Location |
|------|----------|
| Source code | `src/*.go` |
| Tests | `src/*_test.go` |
| Binary | `bin/sink` |
| Configs | `data/*.json` |
| Schema | `data/install-config.schema.json` |
| Docs | `docs/*.md` |
| Coverage | `test/coverage.{out,html}` |
| Test artifacts | `test/` |

## Key Files

- `src/main.go` - CLI entry point
- `src/executor.go` - Core execution engine with context
- `src/config.go` - JSON config parsing
- `src/facts.go` - Template variable system
- `src/transport.go` - Command execution layer
- `src/types.go` - Type definitions
- `Makefile` - Build automation
- `README.md` - Full documentation

## Test Statistics

- **Total Tests**: 116
- **Coverage**: ~51% (50.8%)
- **Test Files**: 8 (*_test.go files)
- **Run Time**: ~2.6 seconds

## Execution Context

Every execution shows:
```
🔍 Execution Context:
   Host:      your-hostname
   User:      your-username
   Work Dir:  /path/to/dir
   OS/Arch:   Darwin/arm64
   Transport: local
```

## Safety Features

1. **Context Discovery** - Always shows WHERE commands run
2. **Confirmation Prompt** - Must type "yes" to proceed
3. **Dry-Run Mode** - Preview without execution
4. **Idempotent Checks** - Runs check before action

## Documentation

- `README.md` - Full project documentation
- `docs/REORGANIZATION.md` - Structure change details
- `docs/EXECUTION_CONTEXT_SAFETY.md` - Safety implementation
- `docs/REST_AND_SSH.md` - Future enhancements
- `docs/SCHEMA.md` - Config schema details
- `docs/TRANSPORT_COVERAGE.md` - Coverage analysis

## Troubleshooting

### Build fails
```bash
make clean
make build
```

### Tests fail
```bash
cd /Users/brian/Projects/radiolabme/sink
go test ./src/... -v
```

### Binary not found
```bash
# Build it first
make build
# Or check path
ls -la bin/sink
```

### Config not found
```bash
# Use correct path
./bin/sink execute data/install-config.json
# Not: ./bin/sink execute install-config.json
```

## Next Steps

See `docs/REST_AND_SSH.md` for:
- Phase 2: Execution Guards (hostname patterns, user restrictions)
- Phase 3: SSH Transport (~150-200 LOC)
- Phase 4: REST API (~300-400 LOC)
