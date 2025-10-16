# Makefile for sink project

.PHONY: all build build-static build-linux build-all test coverage clean install help verify-schema

# Default target
all: build test

# Build binary for current platform (dynamic linking)
build:
	@echo "Building sink for current platform..."
	@go build -o bin/sink ./src/...
	@cp src/sink.schema.json data/sink.schema.json
	@echo "✅ Binary built: bin/sink"
	@echo "   Platform: $$(go env GOOS)/$$(go env GOARCH)"
	@echo "   Schema synced: data/sink.schema.json"

# Verify schema synchronization (run before commits/CI)
verify-schema:
	@echo "Verifying schema synchronization..."
	@go test ./src/... -run TestSchemaSynchronization -v
	@echo "✅ Schema verification complete"

# Build static binary for Linux AMD64 (fully portable, no dependencies)
build-static:
	@echo "Building static binary for linux/amd64..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags="-s -w -extldflags=-static" \
		-tags netgo,osusergo \
		-o bin/sink-linux-amd64-static \
		./src/...
	@echo "✅ Static binary built: bin/sink-linux-amd64-static"
	@echo "   Platform: linux/amd64 (static)"
	@file bin/sink-linux-amd64-static || true

# Build dynamic binary for Linux AMD64
build-linux:
	@echo "Building dynamic binary for linux/amd64..."
	@GOOS=linux GOARCH=amd64 go build -o bin/sink-linux-amd64 ./src/...
	@echo "✅ Dynamic binary built: bin/sink-linux-amd64"
	@echo "   Platform: linux/amd64 (dynamic)"

# Build all platform binaries
build-all:
	@echo "Cross-compiling for all platforms and architectures..."
	@mkdir -p bin
	@echo "  darwin/amd64 (dynamic)..."
	@GOOS=darwin GOARCH=amd64 go build -o bin/sink-darwin-amd64 ./src/...
	@echo "  darwin/arm64 (dynamic)..."
	@GOOS=darwin GOARCH=arm64 go build -o bin/sink-darwin-arm64 ./src/...
	@echo "  linux/amd64 (dynamic)..."
	@GOOS=linux GOARCH=amd64 go build -o bin/sink-linux-amd64 ./src/...
	@echo "  linux/arm64 (dynamic)..."
	@GOOS=linux GOARCH=arm64 go build -o bin/sink-linux-arm64 ./src/...
	@echo "  linux/amd64 (static)..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags="-s -w -extldflags=-static" \
		-tags netgo,osusergo \
		-o bin/sink-linux-amd64-static \
		./src/...
	@echo ""
	@echo "✅ Cross-compilation complete - 5 binaries built:"
	@ls -lh bin/sink-* 2>/dev/null || ls -l bin/sink-*

# Run all tests
test:
	@echo "Running tests..."
	@go test ./src/... -v

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@mkdir -p test
	@go test ./src/... -coverprofile=test/coverage.out
	@go tool cover -func=test/coverage.out | tail -1
	@echo ""
	@echo "📊 To view detailed coverage:"
	@echo "   make coverage-html"
	@echo "   open test/coverage.html"

# Generate coverage HTML report
coverage-html: coverage
	@go tool cover -html=test/coverage.out -o test/coverage.html
	@echo "✅ Coverage report: test/coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f bin/sink bin/sink-*
	@rm -f test/coverage.out test/coverage.html test/coverage-new.out test/*.log test/*.tmp
	@echo "✅ Clean complete"

# Install to /usr/local/bin (requires sudo)
install: build
	@echo "Installing to /usr/local/bin..."
	@sudo cp bin/sink /usr/local/bin/
	@echo "✅ Installed: /usr/local/bin/sink"

# Run with demo config (dry-run)
demo:
	@echo "Running demo configuration (dry-run)..."
	@./bin/sink execute data/demo-config.json --dry-run

# Run with install config (dry-run)
demo-install:
	@echo "Running install configuration (dry-run)..."
	@./bin/sink execute data/install-config.json --dry-run

# Show help
help:
	@echo "Sink - Shell Installation Kit"
	@echo ""
	@echo "Build Targets:"
	@echo "  make build           - Build for current platform (auto-detects GOOS/GOARCH)"
	@echo "  make build-static    - Cross-compile static binary for linux/amd64"
	@echo "  make build-linux     - Cross-compile dynamic binary for linux/amd64"
	@echo "  make build-all       - Cross-compile for all platforms (darwin+linux, amd64+arm64)"
	@echo ""
	@echo "Test Targets:"
	@echo "  make test            - Run all tests"
	@echo "  make coverage        - Run tests with coverage report"
	@echo "  make coverage-html   - Generate HTML coverage report"
	@echo "  make verify-schema   - Verify schema file and embedded schema are synchronized"
	@echo ""
	@echo "Other Targets:"
	@echo "  make clean           - Remove build artifacts and coverage files"
	@echo "  make install         - Install to /usr/local/bin (requires sudo)"
	@echo "  make demo            - Run demo config (dry-run)"
	@echo "  make demo-install    - Run install config (dry-run)"
	@echo "  make help            - Show this help"
	@echo ""
	@echo "Project Structure:"
	@echo "  src/       - Go source files and tests"
	@echo "  bin/       - Built binaries"
	@echo "  data/      - Configuration files and schemas"
	@echo "  docs/      - Documentation"
	@echo "  examples/  - Example configurations"
	@echo "  test/      - Test configurations and coverage reports"
	@echo "  scripts/   - Utility scripts (bootstrap-remote.sh)"
	@echo ""
	@echo "Build Notes:"
	@echo "  - All builds are cross-platform: set GOOS/GOARCH to target any platform"
	@echo "  - Static builds (linux only): CGO_ENABLED=0, no external dependencies"
	@echo "  - Dynamic builds: smaller, use system libraries (libc, etc.)"
	@echo "  - Current platform: $$(go env GOOS)/$$(go env GOARCH)"
	@echo ""
	@echo "Common Platforms (64-bit):"
	@echo "  darwin/amd64   darwin/arm64   linux/amd64   linux/arm64"
	@echo ""
	@echo "Other Supported Architectures:"
	@echo "  linux/386 (32-bit x86)    linux/arm (32-bit ARM)"
	@echo "  linux/riscv64             linux/ppc64le"
	@echo "  Run 'go tool dist list' to see all platforms"
