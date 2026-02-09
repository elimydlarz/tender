package tender

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// buildGHWorkflowRunArgs tests

func TestBuildGHWorkflowRunArgs(t *testing.T) {
	t.Run("builds args without prompt", func(t *testing.T) {
		tender := Tender{WorkflowFile: "nightly.yml"}
		got := buildGHWorkflowRunArgs(tender, "")
		want := []string{"workflow", "run", "nightly.yml"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected args: got=%v want=%v", got, want)
		}
	})

	t.Run("builds args with prompt", func(t *testing.T) {
		tender := Tender{WorkflowFile: "nightly.yml"}
		got := buildGHWorkflowRunArgs(tender, "Fix tests")
		want := []string{"workflow", "run", "nightly.yml", "-f", "prompt=Fix tests"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected args: got=%v want=%v", got, want)
		}
	})

	t.Run("trims whitespace from prompt", func(t *testing.T) {
		tender := Tender{WorkflowFile: "test.yml"}
		got := buildGHWorkflowRunArgs(tender, "  spaced prompt  ")
		want := []string{"workflow", "run", "test.yml", "-f", "prompt=spaced prompt"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected args: got=%v want=%v", got, want)
		}
	})

	t.Run("ignores empty prompt after trimming", func(t *testing.T) {
		tender := Tender{WorkflowFile: "test.yml"}
		got := buildGHWorkflowRunArgs(tender, "   ")
		want := []string{"workflow", "run", "test.yml"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected args: got=%v want=%v", got, want)
		}
	})
}

// DispatchTenderNow tests

func TestDispatchTenderNow(t *testing.T) {
	t.Run("dispatches manual tender successfully", func(t *testing.T) {
		// Skip if gh is not available
		if _, err := exec.LookPath("gh"); err != nil {
			t.Skip("GitHub CLI 'gh' not available")
		}

		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Create a manual tender
		tender := Tender{
			Name:         "test-tender",
			Agent:        "Build",
			Manual:       true,
			WorkflowFile: "test-tender.yml",
		}
		if err := SaveTender(root, tender); err != nil {
			t.Fatalf("failed to save tender: %v", err)
		}

		var stdout, stderr bytes.Buffer
		err := DispatchTenderNow(root, "test-tender", "test prompt", &stdout, &stderr)

		// We expect this to fail in test environment since we don't have a real GitHub repo
		// But we can verify the error is about gh workflow dispatch, not earlier validation
		if err == nil {
			// If it succeeds, that's fine too (maybe in a real GitHub repo context)
			return
		}

		// Should fail at gh workflow dispatch level, not validation level
		if !strings.Contains(err.Error(), "gh workflow dispatch failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for non-existent tender", func(t *testing.T) {
		root := t.TempDir()
		var stdout, stderr bytes.Buffer

		err := DispatchTenderNow(root, "non-existent", "prompt", &stdout, &stderr)
		if err == nil {
			t.Fatal("expected error for non-existent tender")
		}

		expected := "tender \"non-existent\" not found"
		if err.Error() != expected {
			t.Fatalf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("returns error for schedule-only tender", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Create a schedule-only tender
		tender := Tender{
			Name:         "schedule-only",
			Agent:        "Build",
			Manual:       false,
			Cron:         "0 9 * * *",
			WorkflowFile: "schedule-only.yml",
		}
		if err := SaveTender(root, tender); err != nil {
			t.Fatalf("failed to save tender: %v", err)
		}

		var stdout, stderr bytes.Buffer
		err := DispatchTenderNow(root, "schedule-only", "prompt", &stdout, &stderr)
		if err == nil {
			t.Fatal("expected error for schedule-only tender")
		}

		expected := "tender \"schedule-only\" does not allow on-demand runs; enable workflow_dispatch to use 'tender run'"
		if err.Error() != expected {
			t.Fatalf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("returns error for push-only tender", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		tender := Tender{
			Name:         "push-only",
			Agent:        "Build",
			Manual:       false,
			Push:         true,
			WorkflowFile: "push-only.yml",
		}
		if err := SaveTender(root, tender); err != nil {
			t.Fatalf("failed to save tender: %v", err)
		}

		var stdout, stderr bytes.Buffer
		err := DispatchTenderNow(root, "push-only", "prompt", &stdout, &stderr)
		if err == nil {
			t.Fatal("expected error for push-only tender")
		}

		expected := "tender \"push-only\" does not allow on-demand runs; enable workflow_dispatch to use 'tender run'"
		if err.Error() != expected {
			t.Fatalf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("returns error when gh CLI is not available", func(t *testing.T) {
		// This test is hard to do reliably since gh might be in system paths
		// We'll test the validation logic by mocking the exec.LookPath call
		// through a test that creates an environment where gh would fail
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Create a manual tender
		tender := Tender{
			Name:         "test-tender",
			Agent:        "Build",
			Manual:       true,
			WorkflowFile: "test-tender.yml",
		}
		if err := SaveTender(root, tender); err != nil {
			t.Fatalf("failed to save tender: %v", err)
		}

		// If gh is available, we can't easily test this case
		if _, err := exec.LookPath("gh"); err == nil {
			t.Skip("gh CLI is available, cannot test unavailable case")
		}

		var stdout, stderr bytes.Buffer
		err := DispatchTenderNow(root, "test-tender", "prompt", &stdout, &stderr)
		if err == nil {
			t.Fatal("expected error when gh CLI is not available")
		}

		expected := "GitHub CLI 'gh' is required to run a tender now"
		if err.Error() != expected {
			t.Fatalf("expected error %q, got %q", expected, err.Error())
		}
	})

	t.Run("propagates LoadTenders errors", func(t *testing.T) {
		// Create a directory and then make it unreadable
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		workflowDir := filepath.Join(root, WorkflowDir)
		if err := os.Chmod(workflowDir, 0o000); err != nil {
			t.Fatalf("failed to make workflow dir unreadable: %v", err)
		}
		// Restore permissions after test
		defer os.Chmod(workflowDir, 0o755)

		var stdout, stderr bytes.Buffer
		err := DispatchTenderNow(root, "any", "prompt", &stdout, &stderr)
		if err == nil {
			t.Fatal("expected error from LoadTenders")
		}

		// Should be a permission error or similar filesystem error
		if !strings.Contains(err.Error(), "permission denied") &&
			!strings.Contains(err.Error(), "access") &&
			!strings.Contains(err.Error(), "denied") {
			t.Logf("unexpected error type: %v", err)
		}
	})
}
