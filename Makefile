# Makefile for sink project

.PHONY: all build test coverage clean install help

# Default target
all: build test

# Build the binary
build:
	@echo "Building sink..."
	@go build -o bin/sink ./src/...
	@echo "✅ Binary built: bin/sink"

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
	@rm -f bin/sink
	@rm -rf test/coverage.out test/coverage.html test/*.log test/*.tmp
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
	@echo "Targets:"
	@echo "  make build           - Build the binary to bin/sink"
	@echo "  make test            - Run all tests"
	@echo "  make coverage        - Run tests with coverage report"
	@echo "  make coverage-html   - Generate HTML coverage report"
	@echo "  make clean           - Remove build artifacts"
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
	@echo "  test/      - (empty - tests are in src/)"
