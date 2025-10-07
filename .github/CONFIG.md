# CI/CD Configuration

This directory contains GitHub Actions workflows and configuration for automated testing, building, and validation.

## Workflows

### ci.yml - Continuous Integration
Runs on every push to `main`/`develop` branches and all pull requests.

**Jobs:**

1. **Test** (Matrix: Ubuntu/macOS × Go 1.22/1.23)
   - Runs all unit tests
   - Generates coverage reports
   - Uploads coverage to Codecov (Ubuntu + Go 1.23 only)

2. **Build** (Matrix: Ubuntu/macOS)
   - Builds binary for each platform
   - Verifies binary works (`version`, `help`, `schema` commands)
   - Uploads artifacts for 7 days

3. **Validate Configurations**
   - Validates all JSON configs in `data/`, `examples/`, `test/`
   - Runs dry-run tests on example configs
   - Ensures configs match schema

4. **Lint**
   - Runs `go vet` for correctness
   - Checks `go fmt` formatting
   - Verifies `go.mod`/`go.sum` are up to date

5. **Schema Validation**
   - Verifies embedded schema matches source
   - Checks `data/sink.schema.json` is in sync with `src/sink.schema.json`
   - Validates schema is valid JSON

6. **Integration Tests**
   - Tests `facts`, `validate`, `execute` commands
   - Verifies help commands work
   - End-to-end command testing

7. **Security Scan** (Non-blocking)
   - Runs `govulncheck` for known vulnerabilities
   - Runs `gosec` for security issues
   - Continues even if issues found

8. **Documentation Check** (Non-blocking)
   - Checks for broken markdown links
   - Verifies all README files exist
   - Looks for TODO comments in code
   - Continues even if issues found

9. **CI Summary**
   - Aggregates results from all jobs
   - Fails if any critical job failed
   - Security and Documentation are informational only

**Runtime:** ~8-12 minutes total with caching

### release.yml - Release Automation
Triggered by version tags (e.g., `v1.0.0`).

**Jobs:**

1. **Build Release Binaries**
   - Cross-compiles for all platforms:
     - `darwin/amd64` (macOS Intel)
     - `darwin/arm64` (macOS Apple Silicon)
     - `linux/amd64` (static)
     - `linux/arm64` (static)
   - Generates SHA256 checksums
   - Creates GitHub release with changelog
   - Attaches binaries and checksums

**Trigger:**
```bash
git tag v1.0.0
git push origin v1.0.0
```

## Configuration Files

### dependabot.yml
Automated dependency updates:
- Go modules (weekly, Mondays)
- GitHub Actions (weekly, Mondays)
- Max 5 PRs per ecosystem
- Auto-labeled with "dependencies"

### markdown-link-check-config.json
Configuration for markdown link validation:
- Ignores localhost, example.com
- 20s timeout, 3 retries
- Retry on 429 (rate limit)

### Issue Templates
Three structured issue templates:
- `bug_report.yml` - Bug reports with environment info
- `feature_request.yml` - Feature requests with use cases
- `documentation.yml` - Documentation improvements

### Pull Request Template
`PULL_REQUEST_TEMPLATE.md` - Comprehensive PR checklist covering:
- Type of change
- Testing requirements
- Documentation updates
- Self-review checklist

## Local Testing

Reproduce CI checks locally before pushing:

### Run All Tests
```bash
make test
make coverage
```

### Lint Checks
```bash
cd src
go vet ./...
go fmt ./...
go mod tidy
```

### Validate Configurations
```bash
./bin/sink validate data/*.json
./bin/sink validate examples/*.json
./bin/sink validate test/*.json
```

### Security Scan
```bash
cd src
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...
```

### Integration Tests
```bash
./bin/sink facts data/install-config-with-facts.json
./bin/sink validate data/demo-config.json
./bin/sink execute data/demo-config.json --dry-run
```

### Schema Validation
```bash
diff src/sink.schema.json <(./bin/sink schema)
diff src/sink.schema.json data/sink.schema.json
```

## Troubleshooting

### "Lint job failed"
**Common causes:**
- Code not formatted: Run `cd src && go fmt ./...`
- `go.mod` out of date: Run `cd src && go mod tidy`
- `go vet` errors: Run `cd src && go vet ./...` to see issues

**Fix:**
```bash
cd src
go fmt ./...
go mod tidy
go vet ./...
git add go.mod go.sum **/*.go
git commit -m "Fix lint issues"
```

### "Security Scan failed"
Security job is **non-blocking** (marked `continue-on-error: true`). It provides informational warnings but won't block PRs.

**Common warnings:**
- `G104: Errors unhandled` - Consider wrapping errors
- `G204: Subprocess launched` - By design for this tool
- `G302: File permissions` - May be intentional

### "Documentation Check failed"
Documentation job is **non-blocking**. Common issues:
- Broken external links (may be temporary)
- Rate limiting from link checkers
- TODO comments in code (informational only)

### "Validate Configurations failed"
**Common causes:**
- JSON syntax error in config file
- Config doesn't match schema
- Missing required fields

**Fix:**
```bash
# Validate specific file
./bin/sink validate path/to/config.json

# Check schema sync
diff src/sink.schema.json data/sink.schema.json
```

### "Integration Tests failed"
**Common causes:**
- Binary not built or not in PATH
- Test config file missing
- Command syntax changed

**Debug:**
```bash
# Rebuild binary
make clean build

# Test commands manually
./bin/sink version
./bin/sink facts data/install-config-with-facts.json
./bin/sink execute data/demo-config.json --dry-run
```

### "Test failed"
**Common causes:**
- Unit test regression
- Platform-specific test failure
- Race condition in concurrent tests

**Debug:**
```bash
# Run tests with verbose output
cd src
go test -v ./...

# Run specific test
go test -v -run TestFunctionName ./...

# Check for race conditions
go test -race ./...
```

## CI Performance

**Caching:**
- Go modules cached per OS + Go version
- Cache key based on `go.sum` hash
- Significantly speeds up subsequent runs

**Parallelization:**
- Test job runs 4 matrix combinations in parallel
- Build job runs 2 platforms in parallel
- Other jobs run sequentially due to dependencies

**Optimization tips:**
- Keep dependencies minimal (currently zero!)
- Use `make` targets for consistency
- Leverage build caching via `actions/cache`

## Badge Status

Add to README.md:
```markdown
[![CI](https://github.com/radiolabme/sink/actions/workflows/ci.yml/badge.svg)](https://github.com/radiolabme/sink/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/radiolabme/sink/branch/main/graph/badge.svg)](https://codecov.io/gh/radiolabme/sink)
```

## Manual Workflow Dispatch

Trigger CI manually from GitHub Actions UI:
1. Go to Actions tab
2. Select "CI" workflow
3. Click "Run workflow"
4. Select branch
5. Click "Run workflow" button

## Future Improvements

**Potential enhancements:**
- [ ] Add benchmark tracking
- [ ] Add performance regression detection
- [ ] Add cross-platform integration tests (Docker)
- [ ] Add release notes generation
- [ ] Add automatic PR labeling
- [ ] Add code coverage thresholds
- [ ] Add mutation testing
