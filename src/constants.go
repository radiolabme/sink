package main

import "time"

// HTTP Configuration
const (
	// DefaultHTTPTimeout is the timeout for HTTP requests when downloading configs
	DefaultHTTPTimeout = 30 * time.Second

	// ChecksumHTTPTimeout is the timeout for fetching .sha256 checksum files
	ChecksumHTTPTimeout = 10 * time.Second

	// MaxHTTPRetries is the maximum number of retry attempts for HTTP requests
	MaxHTTPRetries = 3
)

// File Permissions
const (
	// TempFilePermission is the permission for temporary files (user read/write only)
	TempFilePermission = 0600

	// ConfigFilePermission is the permission for config files (user read/write, group/others read)
	ConfigFilePermission = 0644

	// ExecutablePermission is the permission for executable files
	ExecutablePermission = 0755
)

// Execution Configuration
const (
	// MaxConcurrentSteps is the maximum number of steps that can run concurrently
	MaxConcurrentSteps = 10

	// DefaultRetryAttempts is the default number of retry attempts for failed commands
	DefaultRetryAttempts = 3

	// RetryBackoffMultiplier is the multiplier for exponential backoff between retries
	RetryBackoffMultiplier = 2

	// MinRetryWait is the minimum wait time between retries
	MinRetryWait = 1 * time.Second

	// MaxRetryWait is the maximum wait time between retries
	MaxRetryWait = 30 * time.Second
)

// Command Execution
const (
	// DefaultCommandTimeout is the default timeout for command execution
	DefaultCommandTimeout = 5 * time.Minute

	// MaxCommandOutputSize is the maximum size of command output to capture
	MaxCommandOutputSize = 1024 * 1024 // 1MB
)

// Network Configuration
const (
	// MaxIdleHTTPConnections is the maximum number of idle HTTP connections
	MaxIdleHTTPConnections = 100

	// MaxIdleConnectionsPerHost is the maximum idle connections per host
	MaxIdleConnectionsPerHost = 10

	// IdleConnectionTimeout is how long idle connections are kept alive
	IdleConnectionTimeout = 90 * time.Second
)

// UI/Display
const (
	// ProgressUpdateInterval is how often to update progress displays
	ProgressUpdateInterval = 100 * time.Millisecond

	// SpinnerFrames are the characters used for the spinner animation
	SpinnerFrames = `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`
)

// Version Information
const (
	// Version is the current version of sink
	Version = "0.3.2"

	// SchemaVersion is the supported config schema version
	SchemaVersion = "0.3.2"
)
