package main

import (
	"os"
	"strings"
	"testing"
)

// TestLocalTransportCommandExecution tests basic command execution
func TestLocalTransportCommandExecution(t *testing.T) {
	transport := NewLocalTransport()

	tests := []struct {
		name          string
		command       string
		wantExitCode  int
		wantStdoutHas string
		wantStderrHas string
		wantErr       bool
	}{
		{
			name:          "successful echo",
			command:       "echo hello",
			wantExitCode:  0,
			wantStdoutHas: "hello",
		},
		{
			name:         "failing command",
			command:      "exit 1",
			wantExitCode: 1,
		},
		{
			name:         "command not found",
			command:      "nonexistent-command-xyz",
			wantExitCode: 127,
		},
		{
			name:          "stderr output",
			command:       "echo error >&2",
			wantExitCode:  0,
			wantStderrHas: "error",
		},
		{
			name:          "multiline output",
			command:       "echo line1; echo line2",
			wantExitCode:  0,
			wantStdoutHas: "line1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitCode, err := transport.Run(tt.command)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if exitCode != tt.wantExitCode {
				t.Errorf("exitCode = %d, want %d", exitCode, tt.wantExitCode)
			}

			if tt.wantStdoutHas != "" && !strings.Contains(stdout, tt.wantStdoutHas) {
				t.Errorf("stdout = %q, want to contain %q", stdout, tt.wantStdoutHas)
			}

			if tt.wantStderrHas != "" && !strings.Contains(stderr, tt.wantStderrHas) {
				t.Errorf("stderr = %q, want to contain %q", stderr, tt.wantStderrHas)
			}
		})
	}
}

// TestLocalTransportWithEnvironment tests environment variable passing
func TestLocalTransportWithEnvironment(t *testing.T) {
	transport := NewLocalTransport()
	transport.Env = []string{"TEST_VAR=test_value"}

	stdout, _, exitCode, err := transport.Run("echo $TEST_VAR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout, "test_value") {
		t.Errorf("stdout = %q, want to contain 'test_value'", stdout)
	}
}

// TestLocalTransportWithWorkingDirectory tests working directory setting
func TestLocalTransportWithWorkingDirectory(t *testing.T) {
	// Create a temp directory for testing
	tmpDir := t.TempDir()

	transport := NewLocalTransport()
	transport.WorkDir = tmpDir

	stdout, _, exitCode, err := transport.Run("pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout, tmpDir) {
		t.Errorf("stdout = %q, want to contain %q", stdout, tmpDir)
	}
}

// TestLocalTransportShellExecution tests that commands run through a shell
func TestLocalTransportShellExecution(t *testing.T) {
	transport := NewLocalTransport()

	// Test shell features like pipes
	stdout, _, exitCode, err := transport.Run("echo hello | tr 'h' 'H'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout, "Hello") {
		t.Errorf("stdout = %q, want to contain 'Hello' (pipe should work)", stdout)
	}
}

// TestLocalTransportOutputCapture tests that both stdout and stderr are captured
func TestLocalTransportOutputCapture(t *testing.T) {
	transport := NewLocalTransport()

	// Command that writes to both stdout and stderr
	stdout, stderr, exitCode, err := transport.Run("echo out; echo err >&2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout, "out") {
		t.Errorf("stdout = %q, want to contain 'out'", stdout)
	}

	if !strings.Contains(stderr, "err") {
		t.Errorf("stderr = %q, want to contain 'err'", stderr)
	}
}

// TestLocalTransportLongRunningCommand tests commands that take time
func TestLocalTransportLongRunningCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	transport := NewLocalTransport()

	stdout, _, exitCode, err := transport.Run("sleep 0.1 && echo done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout, "done") {
		t.Errorf("stdout = %q, want to contain 'done'", stdout)
	}
}

// TestLocalTransportSpecialCharacters tests handling of special characters
func TestLocalTransportSpecialCharacters(t *testing.T) {
	transport := NewLocalTransport()

	tests := []struct {
		name    string
		command string
		wantOut string
	}{
		{
			name:    "quotes",
			command: `echo "hello world"`,
			wantOut: "hello world",
		},
		{
			name:    "single quotes",
			command: `echo 'test'`,
			wantOut: "test",
		},
		{
			name:    "dollar sign",
			command: `echo '$USER'`,
			wantOut: "$USER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, exitCode, err := transport.Run(tt.command)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if exitCode != 0 {
				t.Errorf("exitCode = %d, want 0", exitCode)
			}

			if !strings.Contains(stdout, tt.wantOut) {
				t.Errorf("stdout = %q, want to contain %q", stdout, tt.wantOut)
			}
		})
	}
}

// TestLocalTransportWithExistingEnvironment tests that existing env vars are preserved
func TestLocalTransportWithExistingEnvironment(t *testing.T) {
	transport := NewLocalTransport()

	// Don't set Env, should inherit from parent process
	stdout, _, exitCode, err := transport.Run("echo $HOME")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	home := os.Getenv("HOME")
	if home != "" && !strings.Contains(stdout, home) {
		t.Errorf("stdout = %q, want to contain HOME value %q", stdout, home)
	}
}

// TestLocalTransportExitCodes tests various exit codes
func TestLocalTransportExitCodes(t *testing.T) {
	transport := NewLocalTransport()

	tests := []struct {
		name         string
		command      string
		expectedCode int
	}{
		{"exit 0", "exit 0", 0},
		{"exit 1", "exit 1", 1},
		{"exit 2", "exit 2", 2},
		{"exit 42", "exit 42", 42},
		{"exit 127", "exit 127", 127},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, exitCode, _ := transport.Run(tt.command)
			if exitCode != tt.expectedCode {
				t.Errorf("expected exit code %d, got %d", tt.expectedCode, exitCode)
			}
		})
	}
}
