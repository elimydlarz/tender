package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
			"tender <command> [args]",
			"Commands:",
			"init            Ensure .github/workflows exists",
			"add             Add a tender non-interactively (agent-friendly)",
			"update          Update a tender non-interactively (agent-friendly)",
			"ls              List managed tender workflows",
			"run             Trigger an on-demand tender now via GitHub CLI",
			"rm              Remove a tender workflow",
			"help [command]  Show command help",
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
		if !strings.Contains(output, "Commands:") {
			t.Fatal("expected command list in help output")
		}
	})

	t.Run("tender add --help includes command list", func(t *testing.T) {
		cmd := exec.Command(binPath, "add", "--help")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		err := cmd.Run()
		if err != nil {
			t.Fatalf("tender add --help failed: %v", err)
		}

		output := stdout.String()
		expected := []string{
			"Commands:",
			"add             Add a tender non-interactively (agent-friendly)",
			"Command: add",
			"usage: tender add [--name <name>] --agent <agent>",
		}
		for _, line := range expected {
			if !strings.Contains(output, line) {
				t.Fatalf("add --help output missing: %q\nActual output:\n%s", line, output)
			}
		}
	})

	t.Run("tender help add", func(t *testing.T) {
		cmd := exec.Command(binPath, "help", "add")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		err := cmd.Run()
		if err != nil {
			t.Fatalf("tender help add failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Command: add") {
			t.Fatalf("expected add command help, got: %s", output)
		}
		if !strings.Contains(output, "Commands:") {
			t.Fatalf("expected command list in help add output, got: %s", output)
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

func TestCLIAgentCommands(t *testing.T) {
	binPath := buildTenderForTesting(t)
	defer os.Remove(binPath)

	t.Run("tender add requires a name", func(t *testing.T) {
		tmpDir := t.TempDir()
		cmd := exec.Command(binPath, "add", "--agent", "TendTests")
		cmd.Dir = tmpDir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error when name is missing")
		}
		if !strings.Contains(stderr.String(), "usage: tender add") {
			t.Fatalf("expected add usage error, got: %s", stderr.String())
		}
	})

	t.Run("tender add requires agent", func(t *testing.T) {
		tmpDir := t.TempDir()
		cmd := exec.Command(binPath, "add", "--name", "nightly")
		cmd.Dir = tmpDir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error when agent is missing")
		}
		if !strings.Contains(stderr.String(), "agent is required") {
			t.Fatalf("expected agent validation error, got: %s", stderr.String())
		}
	})

	t.Run("tender add and update work non-interactively", func(t *testing.T) {
		tmpDir := t.TempDir()
		fakeBin := installFakeOpenCodeForCLI(t, tmpDir, []string{"TendTests", "TestReviewer"})

		addCmd := exec.Command(
			binPath,
			"add",
			"--name", "nightly",
			"--agent", "TendTests",
			"--prompt", "ship the updates",
			"--cron", "0 9 * * 1",
			"--manual", "true",
			"--push", "false",
		)
		addCmd.Dir = tmpDir
		addCmd.Env = withPrependedPATH(fakeBin)
		var addStdout, addStderr bytes.Buffer
		addCmd.Stdout = &addStdout
		addCmd.Stderr = &addStderr
		if err := addCmd.Run(); err != nil {
			t.Fatalf("add failed: %v\nstderr: %s", err, addStderr.String())
		}
		if !strings.Contains(addStdout.String(), "saved nightly.yml") {
			t.Fatalf("expected saved output, got: %s", addStdout.String())
		}

		listCmd := exec.Command(binPath, "ls")
		listCmd.Dir = tmpDir
		var listStdout bytes.Buffer
		listCmd.Stdout = &listStdout
		if err := listCmd.Run(); err != nil {
			t.Fatalf("ls failed: %v", err)
		}
		if !strings.Contains(listStdout.String(), "nightly\tTendTests\tweekly Mon at 09:00 UTC + on-demand\tnightly.yml") {
			t.Fatalf("expected created tender in list output, got: %s", listStdout.String())
		}

		updateCmd := exec.Command(
			binPath,
			"update",
			"nightly",
			"--agent", "TendTests",
			"--manual", "false",
			"--push", "true",
			"--clear-cron",
		)
		updateCmd.Dir = tmpDir
		updateCmd.Env = withPrependedPATH(fakeBin)
		var updateStdout, updateStderr bytes.Buffer
		updateCmd.Stdout = &updateStdout
		updateCmd.Stderr = &updateStderr
		if err := updateCmd.Run(); err != nil {
			t.Fatalf("update failed: %v\nstderr: %s", err, updateStderr.String())
		}
		if !strings.Contains(updateStdout.String(), "updated nightly.yml") {
			t.Fatalf("expected updated output, got: %s", updateStdout.String())
		}

		workflowPath := filepath.Join(tmpDir, ".github", "workflows", "nightly.yml")
		workflowBytes, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("read updated workflow: %v", err)
		}
		workflow := string(workflowBytes)
		if !strings.Contains(workflow, `TENDER_AGENT: "TendTests"`) {
			t.Fatalf("expected updated agent in workflow, got:\n%s", workflow)
		}
		triggerBlock := workflow
		if idx := strings.Index(triggerBlock, "\npermissions:\n"); idx >= 0 {
			triggerBlock = triggerBlock[:idx]
		}
		if !strings.Contains(triggerBlock, "\n  push:\n") {
			t.Fatalf("expected push trigger block, got:\n%s", triggerBlock)
		}
		if strings.Contains(triggerBlock, "\n  workflow_dispatch:\n") {
			t.Fatalf("did not expect workflow_dispatch trigger block, got:\n%s", triggerBlock)
		}
		if strings.Contains(triggerBlock, "\n  schedule:\n") {
			t.Fatalf("did not expect schedule trigger block, got:\n%s", triggerBlock)
		}

		listAfterUpdateCmd := exec.Command(binPath, "ls")
		listAfterUpdateCmd.Dir = tmpDir
		var listAfterUpdateStdout bytes.Buffer
		listAfterUpdateCmd.Stdout = &listAfterUpdateStdout
		if err := listAfterUpdateCmd.Run(); err != nil {
			t.Fatalf("ls after update failed: %v", err)
		}
		if !strings.Contains(listAfterUpdateStdout.String(), "nightly\tTendTests\ton-push(main)\tnightly.yml") {
			t.Fatalf("expected updated tender in list output, got: %s", listAfterUpdateStdout.String())
		}
	})

	t.Run("tender add rejects reserved system agent", func(t *testing.T) {
		tmpDir := t.TempDir()
		cmd := exec.Command(binPath, "add", "--name", "nightly", "--agent", "Build")
		cmd.Dir = tmpDir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for reserved system agent")
		}
		if !strings.Contains(stderr.String(), "reserved; choose a custom agent") {
			t.Fatalf("expected reserved agent validation error, got: %s", stderr.String())
		}
	})

	t.Run("tender add rejects unknown custom agent", func(t *testing.T) {
		tmpDir := t.TempDir()
		fakeBin := installFakeOpenCodeForCLI(t, tmpDir, []string{"TendTests", "TestReviewer"})
		cmd := exec.Command(binPath, "add", "--name", "nightly", "--agent", "UnknownAgent")
		cmd.Dir = tmpDir
		cmd.Env = withPrependedPATH(fakeBin)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err == nil {
			t.Fatal("expected error for unknown custom agent")
		}
		if !strings.Contains(stderr.String(), "is not a discovered custom primary agent") {
			t.Fatalf("expected unknown custom agent validation error, got: %s", stderr.String())
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

func installFakeOpenCodeForCLI(t *testing.T, root string, agents []string) string {
	t.Helper()
	binDir := filepath.Join(root, ".tender", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir fake opencode dir: %v", err)
	}

	name := "opencode"
	if runtime.GOOS == "windows" {
		name = "opencode.bat"
	}
	path := filepath.Join(binDir, name)

	var script string
	if runtime.GOOS == "windows" {
		lines := []string{
			"@echo off",
			"if \"%1\"==\"agent\" if \"%2\"==\"list\" goto list",
			"echo unsupported fake opencode command: %* 1>&2",
			"exit /b 1",
			":list",
		}
		for _, agent := range agents {
			lines = append(lines, "echo "+agent+" primary")
		}
		lines = append(lines, "exit /b 0")
		script = strings.Join(lines, "\r\n") + "\r\n"
	} else {
		var b strings.Builder
		b.WriteString("#!/bin/sh\n")
		b.WriteString("if [ \"$1\" = \"agent\" ] && [ \"$2\" = \"list\" ]; then\n")
		b.WriteString("cat <<'EOF'\n")
		for _, agent := range agents {
			b.WriteString(agent)
			b.WriteString(" primary\n")
		}
		b.WriteString("EOF\n")
		b.WriteString("exit 0\n")
		b.WriteString("fi\n")
		b.WriteString("echo \"unsupported fake opencode command: $*\" >&2\n")
		b.WriteString("exit 1\n")
		script = b.String()
	}

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake opencode: %v", err)
	}
	return binDir
}

func withPrependedPATH(prefix string) []string {
	env := os.Environ()
	for i, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			env[i] = "PATH=" + prefix + string(os.PathListSeparator) + strings.TrimPrefix(kv, "PATH=")
			return env
		}
	}
	return append(env, "PATH="+prefix)
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
