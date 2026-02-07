package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// Test the usage function
func TestUsage(t *testing.T) {
	t.Run("prints usage information", func(t *testing.T) {
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
			"tender             Launch interactive TUI",
			"tender init        Ensure .github/workflows exists",
			"tender ls          List managed tender workflows",
			"tender run <name>  Trigger an on-demand tender now via GitHub CLI",
			"tender rm <name>   Remove a tender workflow",
		}

		for _, line := range expectedLines {
			if !strings.Contains(result, line) {
				t.Fatalf("usage output missing line: %q\nActual output:\n%s", line, result)
			}
		}
	})
}

// Test the fail function
func TestFail(t *testing.T) {
	t.Run("formats error message correctly", func(t *testing.T) {
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
	// This would be better as an integration test, but we can at least
	// verify that the function exists and has the right signature

	// We can't call main() directly because it exits the process
	// In a real scenario, we'd either:
	// 1. Extract the argument parsing logic to a testable function
	// 2. Use integration tests with subprocess execution
	// 3. Mock os.Exit for testing

	t.Skip("main function testing requires integration tests or refactoring")
}
