package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Integration tests for CLI commands
// These tests build the actual tender binary and run it as a subprocess

func TestCLICommands(t *testing.T) {
	// Build the tender binary for testing
	binPath := buildTenderForTesting(t)
	defer os.Remove(binPath)

	t.Run("tender help", func(t *testing.T) {
		cmd := exec.Command(binPath, "help")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Fatalf("tender help failed: %v", err)
		}

		output := stdout.String()
		expected := []string{
			"tender - interactive CLI for autonomous OpenCode schedules",
			"Usage:",
			"tender             Launch interactive TUI",
			"tender init        Ensure .github/workflows exists",
			"tender ls          List managed tender workflows",
			"tender run <name>  Trigger an on-demand tender now via GitHub CLI",
			"tender rm <name>   Remove a tender workflow",
		}

		for _, line := range expected {
			if !strings.Contains(output, line) {
				t.Fatalf("help output missing: %q\nActual output:\n%s", line, output)
			}
		}
	})

	t.Run("tender --help", func(t *testing.T) {
		cmd := exec.Command(binPath, "--help")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		err := cmd.Run()
		if err != nil {
			t.Fatalf("tender --help failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tender - interactive CLI") {
			t.Fatal("expected help output")
		}
	})

	t.Run("tender init", func(t *testing.T) {
		tmpDir := t.TempDir()

		cmd := exec.Command(binPath, "init")
		cmd.Dir = tmpDir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Fatalf("tender init failed: %v\nStderr: %s", err, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "initialized") {
			t.Fatalf("expected initialization message, got: %s", output)
		}

		// Check that .github/workflows directory was created
		workflowDir := filepath.Join(tmpDir, ".github", "workflows")
		if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
			t.Fatal("workflow directory was not created")
		}
	})

	t.Run("tender ls with no tenders", func(t *testing.T) {
		tmpDir := t.TempDir()

		// First initialize
		initCmd := exec.Command(binPath, "init")
		initCmd.Dir = tmpDir
		if err := initCmd.Run(); err != nil {
			t.Fatalf("init failed: %v", err)
		}

		// Then list
		cmd := exec.Command(binPath, "ls")
		cmd.Dir = tmpDir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Fatalf("tender ls failed: %v\nStderr: %s", err, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "No managed tender workflows found") {
			t.Fatalf("expected no tenders message, got: %s", output)
		}
	})

	t.Run("tender rm requires name argument", func(t *testing.T) {
		tmpDir := t.TempDir()

		cmd := exec.Command(binPath, "rm")
		cmd.Dir = tmpDir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error when no name provided")
		}

		output := stderr.String()
		if !strings.Contains(output, "usage: tender rm") {
			t.Fatalf("expected usage error, got: %s", output)
		}
	})

	t.Run("tender run requires name argument", func(t *testing.T) {
		tmpDir := t.TempDir()

		cmd := exec.Command(binPath, "run")
		cmd.Dir = tmpDir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error when no name provided")
		}

		output := stderr.String()
		if !strings.Contains(output, "usage: tender run") {
			t.Fatalf("expected usage error, got: %s", output)
		}
	})

	t.Run("tender run with non-existent tender", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize first
		initCmd := exec.Command(binPath, "init")
		initCmd.Dir = tmpDir
		if err := initCmd.Run(); err != nil {
			t.Fatalf("init failed: %v", err)
		}

		// Try to run non-existent tender
		cmd := exec.Command(binPath, "run", "non-existent")
		cmd.Dir = tmpDir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for non-existent tender")
		}

		output := stderr.String()
		if !strings.Contains(output, "not found") {
			t.Fatalf("expected not found error, got: %s", output)
		}
	})

	t.Run("tender invalid command", func(t *testing.T) {
		cmd := exec.Command(binPath, "invalid-command")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for invalid command")
		}

		// Check exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 2 {
				t.Fatalf("expected exit code 2, got %d", exitErr.ExitCode())
			}
		}

		// Output should contain usage information (goes to stdout)
		output := stdout.String()
		if !strings.Contains(output, "tender - interactive CLI") {
			t.Fatalf("expected usage information for invalid command, got: %s", output)
		}
	})
}

func TestCLIWorkflowManagement(t *testing.T) {
	binPath := buildTenderForTesting(t)
	defer os.Remove(binPath)

	tmpDir := t.TempDir()

	// Initialize the repository
	initCmd := exec.Command(binPath, "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	t.Run("can create workflow manually for testing", func(t *testing.T) {
		// Create a simple workflow file manually to test other commands
		workflowContent := `name: "tender/test-workflow"
on:
  workflow_dispatch:
    inputs:
      prompt:
        description: "Optional prompt override"
        required: false
        default: ""
        type: string
permissions:
  contents: write
concurrency:
  group: tender-main
  cancel-in-progress: false
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_NAME: "test-workflow"
      TENDER_AGENT: "Build"
      TENDER_PROMPT: "test prompt"
    steps:
      - uses: actions/checkout@v4
      - name: Run OpenCode
        run: opencode run --agent "$TENDER_AGENT" "$RUN_PROMPT"
`

		workflowPath := filepath.Join(tmpDir, ".github", "workflows", "test.yml")
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0o644); err != nil {
			t.Fatalf("failed to write workflow: %v", err)
		}

		// Test ls command finds the workflow
		cmd := exec.Command(binPath, "ls")
		cmd.Dir = tmpDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		err := cmd.Run()
		if err != nil {
			t.Fatalf("ls failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "test-workflow") {
			t.Fatalf("expected workflow in list, got: %s", output)
		}
		if !strings.Contains(output, "Build") {
			t.Fatalf("expected agent in list, got: %s", output)
		}
	})
}

// Helper function to build the tender binary for testing
func buildTenderForTesting(t *testing.T) string {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "tender")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = "."
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build tender: %v\nStderr: %s", err, stderr.String())
	}

	return binPath
}

// Test that we can capture both stdout and stderr from CLI commands
func TestCLIOutputCapture(t *testing.T) {
	binPath := buildTenderForTesting(t)
	defer os.Remove(binPath)

	t.Run("captures stdout and stderr separately", func(t *testing.T) {
		// Test with a command that should produce output
		cmd := exec.Command(binPath, "help")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Fatalf("help command failed: %v", err)
		}

		// Should have output on stdout, stderr should be empty for help
		if stdout.Len() == 0 {
			t.Fatal("expected output on stdout")
		}
		if stderr.Len() > 0 {
			t.Fatalf("unexpected stderr output: %s", stderr.String())
		}
	})
}
