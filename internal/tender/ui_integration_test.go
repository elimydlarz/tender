package tender

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunInteractive(t *testing.T) {
	t.Run("displays interface and exits cleanly", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		tender := Tender{
			Name:   "test-tender",
			Agent:  "Build",
			Manual: true,
		}
		if _, err := SaveNewTender(root, tender); err != nil {
			t.Fatalf("failed to save test tender: %v", err)
		}

		stdin := strings.NewReader("q\n")
		var stdout bytes.Buffer

		err := RunInteractive(root, stdin, &stdout)
		if err != nil {
			t.Fatalf("RunInteractive returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Select Tender") {
			t.Fatal("expected selection menu to be displayed")
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

		stdin := strings.NewReader("q\n")
		var stdout bytes.Buffer

		err := RunInteractive(root, stdin, &stdout)
		if err != nil {
			t.Fatalf("RunInteractive returned error: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Showing 0 tenders") {
			t.Fatal("expected empty state summary")
		}
	})
}

func TestDrawHome(t *testing.T) {
	t.Run("when no tenders exist", func(t *testing.T) {
		var stdout bytes.Buffer
		tenders := []Tender{}

		drawHome(&stdout, tenders, 0, nil)

		output := stdout.String()
		if !strings.Contains(output, "Select Tender") {
			t.Fatal("expected menu title")
		}
		if !strings.Contains(output, "Showing 0 tenders") {
			t.Fatal("expected empty summary")
		}
	})

	t.Run("when tenders exist", func(t *testing.T) {
		var stdout bytes.Buffer
		tenders := []Tender{
			{Name: "test1", Agent: "Build", Manual: true, WorkflowFile: "test1.yml"},
			{Name: "test2", Agent: "Deploy", Cron: "0 9 * * *", WorkflowFile: "test2.yml"},
		}

		drawHome(&stdout, tenders, 0, nil)

		output := stdout.String()
		if !strings.Contains(output, "test1") {
			t.Fatal("expected test1 tender")
		}
		if !strings.Contains(output, "test2") {
			t.Fatal("expected test2 tender")
		}
		if !strings.Contains(output, "Showing 1-2 of 2 (page 1/1)") {
			t.Fatal("expected paging summary")
		}
	})
}
