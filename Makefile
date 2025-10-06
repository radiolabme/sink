# Makefile for sink project

.PHONY: all build build-static build-linux build-all test coverage clean install help

# Default target
all: build test

# Build the binary (dynamic linking)
build:
	@echo "Building sink..."
	@go build -o bin/sink ./src/...
	@echo "✅ Binary built: bin/sink"

# Build static binary (Linux only - fully static)
build-static:
	@echo "Building static binary (Linux)..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags="-s -w -extldflags=-static" \
		-tags netgo,osusergo \
		-o bin/sink-linux-amd64-static \
		./src/...
	@echo "✅ Static binary built: bin/sink-linux-amd64-static"
	@file bin/sink-linux-amd64-static || true

# Build Linux binary (dynamic linking)
build-linux:
	@echo "Building Linux binary..."
	@GOOS=linux GOARCH=amd64 go build -o bin/sink-linux-amd64 ./src/...
	@echo "✅ Linux binary built: bin/sink-linux-amd64"

# Build all platform binaries
build-all:
	@echo "Building all platform binaries..."
	@mkdir -p bin
	@echo "  Building macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build -o bin/sink-darwin-amd64 ./src/...
	@echo "  Building macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build -o bin/sink-darwin-arm64 ./src/...
	@echo "  Building Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -o bin/sink-linux-amd64 ./src/...
	@echo "  Building Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build -o bin/sink-linux-arm64 ./src/...
	@echo "  Building Linux AMD64 (static)..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags="-s -w -extldflags=-static" \
		-tags netgo,osusergo \
		-o bin/sink-linux-amd64-static \
		./src/...
	@echo "✅ All binaries built in bin/"
	@ls -lh bin/sink-*

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
	@echo "  make build           - Build the binary for current platform"
	@echo "  make build-static    - Build static binary for Linux (portable)"
	@echo "  make build-linux     - Build Linux binary (dynamic linking)"
	@echo "  make build-all       - Build binaries for all platforms"
	@echo ""
	@echo "Test Targets:"
	@echo "  make test            - Run all tests"
	@echo "  make coverage        - Run tests with coverage report"
	@echo "  make coverage-html   - Generate HTML coverage report"
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
	@echo "  - Default 'build' uses dynamic linking (normal)"
	@echo "  - 'build-static' creates fully static Linux binary (portable)"
	@echo "  - 'build-all' creates binaries for macOS/Linux AMD64/ARM64"
	@echo "  - Static builds use CGO_ENABLED=0 for maximum portability"
