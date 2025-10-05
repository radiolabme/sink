package main

import (
	"bytes"
	"os"
	"os/exec"
	"runtime"
)

// LocalTransport executes commands on the local machine
type LocalTransport struct {
	Env     []string // Environment variables (if nil, inherits from parent)
	WorkDir string   // Working directory (if empty, uses current directory)
}

// NewLocalTransport creates a new local transport
func NewLocalTransport() *LocalTransport {
	return &LocalTransport{}
}

// Run executes a command locally and returns stdout, stderr, exit code, and error
func (lt *LocalTransport) Run(command string) (stdout, stderr string, exitCode int, err error) {
	// Determine the shell to use based on OS
	shell, shellFlag := lt.getShell()

	// Create the command
	cmd := exec.Command(shell, shellFlag, command)

	// Set up stdout and stderr capture
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	// Set environment if specified, otherwise inherit
	if lt.Env != nil {
		cmd.Env = lt.Env
	} else {
		cmd.Env = os.Environ()
	}

	// Set working directory if specified
	if lt.WorkDir != "" {
		cmd.Dir = lt.WorkDir
	}

	// Run the command
	err = cmd.Run()

	// Get the exit code
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			// Don't return error for non-zero exits, just the exit code
			err = nil
		}
		// For other errors (like command not found), exitCode will be set
		// by the shell (typically 127)
		if exitCode == 0 && err != nil {
			// If we still have an error but no exit code, it's likely
			// a system error (command not found, etc.)
			exitCode = 127
		}
	}

	stdout = outBuf.String()
	stderr = errBuf.String()

	return stdout, stderr, exitCode, err
}

// getShell determines the best shell to use for command execution
// Prefers bash (handles more syntax) but falls back to sh (POSIX compatible)
func (lt *LocalTransport) getShell() (string, string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", "/C"
	}

	// Prefer bash (most common, handles bash-specific syntax)
	// ~60% of install scripts use #!/bin/bash, ~30% use bash-specific syntax
	bashPaths := []string{"/bin/bash", "/usr/bin/bash", "/usr/local/bin/bash"}
	for _, path := range bashPaths {
		if _, err := os.Stat(path); err == nil {
			return path, "-c"
		}
	}

	// Fall back to sh (POSIX compatible, always available)
	// Handles minimal containers, NixOS, and edge cases
	shPaths := []string{
		"/bin/sh",
		"/usr/bin/sh",
		"/run/current-system/sw/bin/sh", // NixOS
	}
	for _, path := range shPaths {
		if _, err := os.Stat(path); err == nil {
			return path, "-c"
		}
	}

	// Last resort: hope it's in PATH
	return "sh", "-c"
}
