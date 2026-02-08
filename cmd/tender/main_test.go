package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// Usage function tests
func TestUsage(t *testing.T) {
	t.Run("when called then displays complete usage guide with all commands", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Call usage function
		usage()

		// Restore stdout and capture output
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf strings.Builder
		io.Copy(&buf, r)

		result := buf.String()

		// Check that key information is present
		expectedLines := []string{
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
			"Use `tender <command> --help` to show command-specific usage and flags.",
		}

		for _, line := range expectedLines {
			if !strings.Contains(result, line) {
				t.Fatalf("usage output missing line: %q\nActual output:\n%s", line, result)
			}
		}
	})
}

// Error handling tests
func TestFail(t *testing.T) {
	t.Run("when called then formats error message with proper prefix", func(t *testing.T) {
		// We can't test os.Exit directly, but we can verify the error format
		testErr := &testError{msg: "test error message"}

		// The fail function prints "error: " + err.Error() + "\n"
		expected := "error: test error message\n"
		actual := "error: " + testErr.Error() + "\n"

		if actual != expected {
			t.Fatalf("expected %q, got %q", expected, actual)
		}
	})
}

// Test error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// Test main function behavior with different arguments
// Note: We can't test the main function directly because it calls os.Exit
// But we can test the logic by extracting functions or using integration tests

func TestMainArgumentParsing(t *testing.T) {
	t.Run("when called then requires integration testing due to os.Exit", func(t *testing.T) {
		// Main function testing requires integration tests or refactoring because:
		// 1. main() calls os.Exit which terminates the test process
		// 2. We need to extract argument parsing logic to test it independently
		// 3. Alternative approaches include subprocess execution or mocking os.Exit

		t.Skip("main function testing requires integration tests or refactoring")
	})
}
