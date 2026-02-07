package tender

import (
	"bytes"
	"strings"
	"testing"
)

// ui.go integration tests

func TestRunInteractive(t *testing.T) {
	t.Run("displays interface and exits cleanly", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Create a test tender to verify listing works
		tender := Tender{
			Name:   "test-tender",
			Agent:  "Build",
			Manual: true,
		}
		if _, err := SaveNewTender(root, tender); err != nil {
			t.Fatalf("failed to save test tender: %v", err)
		}

		// Simulate user input: choose option 4 (exit)
		stdin := strings.NewReader("4\n")
		var stdout bytes.Buffer

		err := RunInteractive(root, stdin, &stdout)
		if err != nil {
			t.Fatalf("RunInteractive returned error: %v", err)
		}

		output := stdout.String()
		// Verify basic interface elements are displayed
		if !strings.Contains(output, "TENDER COMMAND DECK") {
			t.Fatal("expected header to be displayed")
		}
		if !strings.Contains(output, "Action Deck") {
			t.Fatal("expected actions to be displayed")
		}
		if !strings.Contains(output, "test-tender") {
			t.Fatal("expected tender name to be displayed")
		}
	})

	t.Run("handles directory with no tenders", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		stdin := strings.NewReader("4\n")
		var stdout bytes.Buffer

		err := RunInteractive(root, stdin, &stdout)
		if err != nil {
			t.Fatalf("RunInteractive returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "No managed tender workflows found") {
			t.Fatal("expected no tenders message")
		}
	})

	t.Run("handles existing workflow directory", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create initial workflow dir: %v", err)
		}

		// EnsureWorkflowDir should handle existing directories gracefully
		stdin := strings.NewReader("4\n")
		var stdout bytes.Buffer

		err := RunInteractive(root, stdin, &stdout)
		if err != nil {
			t.Fatalf("RunInteractive should handle existing workflow dir: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "No managed tender workflows found") {
			t.Fatal("expected empty tenders message")
		}
	})
}

func TestDrawHome(t *testing.T) {
	t.Run("when no tenders exist", func(t *testing.T) {
		var stdout bytes.Buffer
		tenders := []Tender{}

		drawHome(&stdout, tenders)

		output := stdout.String()
		if !strings.Contains(output, "No managed tender workflows found") {
			t.Fatal("expected no tenders message")
		}
		if !strings.Contains(output, "TENDER COMMAND DECK") {
			t.Fatal("expected header")
		}
	})

	t.Run("when tenders exist", func(t *testing.T) {
		var stdout bytes.Buffer
		tenders := []Tender{
			{Name: "test1", Agent: "Build", Manual: true, WorkflowFile: "test1.yml"},
			{Name: "test2", Agent: "Deploy", Cron: "0 9 * * *", WorkflowFile: "test2.yml"},
		}

		drawHome(&stdout, tenders)

		output := stdout.String()
		if !strings.Contains(output, "Name") {
			t.Fatal("expected table header")
		}
		if !strings.Contains(output, "test1") {
			t.Fatal("expected test1 tender")
		}
		if !strings.Contains(output, "test2") {
			t.Fatal("expected test2 tender")
		}
		if !strings.Contains(output, "Build") {
			t.Fatal("expected Build agent")
		}
		if !strings.Contains(output, "Deploy") {
			t.Fatal("expected Deploy agent")
		}
	})
}

func TestDrawActions(t *testing.T) {
	var stdout bytes.Buffer
	drawActions(&stdout)

	output := stdout.String()
	expectedActions := []string{
		"1",
		"2",
		"3",
		"4",
		"Forge new tender",
		"Tune existing tender",
		"Retire tender",
		"Exit",
	}

	for _, action := range expectedActions {
		if !strings.Contains(output, action) {
			t.Fatalf("expected action %q to be displayed. Actual output:\n%s", action, output)
		}
	}
}
