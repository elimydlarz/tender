package tender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// workflow.go tests
//
// This file contains comprehensive tests for the workflow management functionality.
// Tests are organized into logical sections:
// 1. Directory and file management tests
// 2. Tender lifecycle tests (create, read, update, delete)
// 3. Workflow rendering and parsing tests
// 4. Validation and error handling tests
// 5. Utility function tests
// 6. Edge case and error recovery tests

// === Directory and File Management Tests ===

func TestEnsureWorkflowDir(t *testing.T) {
	t.Run("creates new directory", func(t *testing.T) {
		root := t.TempDir()
		err := EnsureWorkflowDir(root)
		if err != nil {
			t.Fatalf("EnsureWorkflowDir returned error: %v", err)
		}

		dir := filepath.Join(root, WorkflowDir)
		if info, err := os.Stat(dir); err != nil {
			t.Fatalf("workflow directory not created: %v", err)
		} else if !info.IsDir() {
			t.Fatal("workflow path is not a directory")
		}
	})

	t.Run("handles existing directory", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, WorkflowDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}

		err := EnsureWorkflowDir(root)
		if err != nil {
			t.Fatalf("EnsureWorkflowDir failed on existing directory: %v", err)
		}
	})
}

// === Tender Lifecycle Tests ===

func TestLoadTenders(t *testing.T) {
	t.Run("empty directory returns empty list", func(t *testing.T) {
		root := t.TempDir()
		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders returned error: %v", err)
		}
		if len(tenders) != 0 {
			t.Fatalf("expected empty list, got %d tenders", len(tenders))
		}
	})

	t.Run("non-existent directory returns empty list", func(t *testing.T) {
		root := t.TempDir()
		nonExistent := filepath.Join(root, "non-existent")
		tenders, err := LoadTenders(nonExistent)
		if err != nil {
			t.Fatalf("LoadTenders returned error for non-existent dir: %v", err)
		}
		if len(tenders) != 0 {
			t.Fatalf("expected empty list for non-existent dir, got %d", len(tenders))
		}
	})

	t.Run("loads valid tender workflows", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Create a valid workflow file
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

		workflowPath := filepath.Join(root, WorkflowDir, "test.yml")
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0o644); err != nil {
			t.Fatalf("failed to write workflow file: %v", err)
		}

		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders returned error: %v", err)
		}
		if len(tenders) != 1 {
			t.Fatalf("expected 1 tender, got %d", len(tenders))
		}

		tender := tenders[0]
		if tender.Name != "test-workflow" {
			t.Fatalf("unexpected name: %q", tender.Name)
		}
		if tender.Agent != "Build" {
			t.Fatalf("unexpected agent: %q", tender.Agent)
		}
		if tender.WorkflowFile != "test.yml" {
			t.Fatalf("unexpected workflow file: %q", tender.WorkflowFile)
		}
	})

	t.Run("ignores non-yaml files", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Create non-yaml files
		files := []string{"readme.txt", "config.json", "script.sh"}
		for _, file := range files {
			path := filepath.Join(root, WorkflowDir, file)
			if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
				t.Fatalf("failed to write %s: %v", file, err)
			}
		}

		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders returned error: %v", err)
		}
		if len(tenders) != 0 {
			t.Fatalf("expected empty list, got %d tenders", len(tenders))
		}
	})

	t.Run("ignores directories", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Create a subdirectory
		subdir := filepath.Join(root, WorkflowDir, "subdir")
		if err := os.MkdirAll(subdir, 0o755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders returned error: %v", err)
		}
		if len(tenders) != 0 {
			t.Fatalf("expected empty list, got %d tenders", len(tenders))
		}
	})
}

// === Tender Creation and Persistence Tests ===

func TestSaveTender(t *testing.T) {
	t.Run("saves new tender with generated filename", func(t *testing.T) {
		root := t.TempDir()
		tender := Tender{
			Name:   "Test Workflow",
			Agent:  "Build",
			Prompt: "test prompt",
			Manual: true,
		}

		err := SaveTender(root, tender)
		if err != nil {
			t.Fatalf("SaveTender returned error: %v", err)
		}

		// Check file was created
		workflowPath := filepath.Join(root, WorkflowDir, "test-workflow.yml")
		if _, err := os.Stat(workflowPath); err != nil {
			t.Fatalf("workflow file not created: %v", err)
		}

		// Check content
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("failed to read workflow file: %v", err)
		}

		text := string(content)
		requiredSnippets := []string{
			`name: "tender/Test Workflow"`,
			`TENDER_AGENT: "Build"`,
			`TENDER_PROMPT: "test prompt"`,
			`workflow_dispatch:`,
		}

		for _, snippet := range requiredSnippets {
			if !strings.Contains(text, snippet) {
				t.Fatalf("workflow missing snippet %q:\n%s", snippet, text)
			}
		}
	})

	t.Run("uses provided workflow file", func(t *testing.T) {
		root := t.TempDir()
		tender := Tender{
			Name:         "Test Workflow",
			Agent:        "Build",
			WorkflowFile: "custom-name.yml",
			Manual:       true,
		}

		err := SaveTender(root, tender)
		if err != nil {
			t.Fatalf("SaveTender returned error: %v", err)
		}

		workflowPath := filepath.Join(root, WorkflowDir, "custom-name.yml")
		if _, err := os.Stat(workflowPath); err != nil {
			t.Fatalf("workflow file not created with custom name: %v", err)
		}
	})

	t.Run("adds yml extension if missing", func(t *testing.T) {
		root := t.TempDir()
		tender := Tender{
			Name:         "Test Workflow",
			Agent:        "Build",
			WorkflowFile: "custom-name",
			Manual:       true,
		}

		err := SaveTender(root, tender)
		if err != nil {
			t.Fatalf("SaveTender returned error: %v", err)
		}

		workflowPath := filepath.Join(root, WorkflowDir, "custom-name.yml")
		if _, err := os.Stat(workflowPath); err != nil {
			t.Fatalf("workflow file not created with .yml extension: %v", err)
		}
	})

	t.Run("validates tender before saving", func(t *testing.T) {
		root := t.TempDir()
		invalidTender := Tender{
			Name:  "", // Empty name should fail validation
			Agent: "Build",
		}

		err := SaveTender(root, invalidTender)
		if err == nil {
			t.Fatal("expected validation error")
		}
		if !strings.Contains(err.Error(), "name is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// === Tender Management Tests ===

func TestSaveNewTender(t *testing.T) {
	t.Run("saves new tender successfully", func(t *testing.T) {
		root := t.TempDir()
		input := Tender{
			Name:   "Build Review",
			Agent:  "Build",
			Manual: true,
		}

		saved, err := SaveNewTender(root, input)
		if err != nil {
			t.Fatalf("SaveNewTender returned error: %v", err)
		}
		if saved.WorkflowFile == "" {
			t.Fatal("expected workflow filename")
		}

		// Verify it was saved correctly
		list, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders returned error: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 tender, got %d", len(list))
		}
		if list[0].Name != "Build Review" {
			t.Fatalf("unexpected name: %q", list[0].Name)
		}
	})

	t.Run("prevents duplicate names", func(t *testing.T) {
		root := t.TempDir()

		// Save first tender
		first := Tender{Name: "duplicate", Agent: "Build", Manual: true}
		_, err := SaveNewTender(root, first)
		if err != nil {
			t.Fatalf("failed to save first tender: %v", err)
		}

		// Try to save duplicate
		second := Tender{Name: "duplicate", Agent: "Other", Manual: true}
		_, err = SaveNewTender(root, second)
		if err == nil {
			t.Fatal("expected error for duplicate name")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("finds unused filename when name conflicts", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Create existing file
		existingPath := filepath.Join(root, WorkflowDir, "test.yml")
		if err := os.WriteFile(existingPath, []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to create existing file: %v", err)
		}

		input := Tender{Name: "test", Agent: "Build", Manual: true}
		saved, err := SaveNewTender(root, input)
		if err != nil {
			t.Fatalf("SaveNewTender returned error: %v", err)
		}

		// Should use test-2.yml since test.yml exists
		if saved.WorkflowFile != "test-2.yml" {
			t.Fatalf("expected test-2.yml, got %q", saved.WorkflowFile)
		}
	})
}

func TestUpdateTender(t *testing.T) {
	t.Run("updates existing tender", func(t *testing.T) {
		root := t.TempDir()

		// Save initial tender
		original := Tender{Name: "test", Agent: "Build", Manual: true}
		_, err := SaveNewTender(root, original)
		if err != nil {
			t.Fatalf("failed to save original tender: %v", err)
		}

		// Update it
		updated := Tender{
			Name:   "renamed",
			Agent:  "Updated",
			Manual: false,
			Cron:   "0 9 * * *",
		}

		err = UpdateTender(root, "test", updated)
		if err != nil {
			t.Fatalf("UpdateTender returned error: %v", err)
		}

		// Verify update
		list, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders returned error: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 tender, got %d", len(list))
		}

		tender := list[0]
		if tender.Name != "renamed" {
			t.Fatalf("expected renamed name, got %q", tender.Name)
		}
		if tender.Agent != "Updated" {
			t.Fatalf("expected updated agent, got %q", tender.Agent)
		}
		if tender.Manual {
			t.Fatal("expected manual to be false")
		}
		if tender.Cron != "0 9 * * *" {
			t.Fatalf("expected cron, got %q", tender.Cron)
		}
	})

	t.Run("prevents name conflicts during update", func(t *testing.T) {
		root := t.TempDir()

		// Save two tenders
		first := Tender{Name: "first", Agent: "Build", Manual: true}
		second := Tender{Name: "second", Agent: "Build", Manual: true}
		_, err := SaveNewTender(root, first)
		if err != nil {
			t.Fatalf("failed to save first tender: %v", err)
		}
		_, err = SaveNewTender(root, second)
		if err != nil {
			t.Fatalf("failed to save second tender: %v", err)
		}

		// Try to rename first to second (should fail)
		updated := Tender{Name: "second", Agent: "Updated", Manual: true}
		err = UpdateTender(root, "first", updated)
		if err == nil {
			t.Fatal("expected error for name conflict")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("returns error for non-existent tender", func(t *testing.T) {
		root := t.TempDir()

		updated := Tender{Name: "new", Agent: "Build", Manual: true}
		err := UpdateTender(root, "non-existent", updated)
		if err == nil {
			t.Fatal("expected error for non-existent tender")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func TestRemoveTender(t *testing.T) {
	t.Run("removes existing tender", func(t *testing.T) {
		root := t.TempDir()

		// Save a tender
		original := Tender{Name: "test", Agent: "Build", Manual: true}
		saved, err := SaveNewTender(root, original)
		if err != nil {
			t.Fatalf("failed to save tender: %v", err)
		}

		// Verify it exists
		workflowPath := filepath.Join(root, WorkflowDir, saved.WorkflowFile)
		if _, err := os.Stat(workflowPath); err != nil {
			t.Fatalf("workflow file not found before removal: %v", err)
		}

		// Remove it
		err = RemoveTender(root, "test")
		if err != nil {
			t.Fatalf("RemoveTender returned error: %v", err)
		}

		// Verify file is gone
		if _, err := os.Stat(workflowPath); err == nil {
			t.Fatal("workflow file still exists after removal")
		}

		// Verify it's not in list
		list, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders returned error: %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("expected empty list, got %d tenders", len(list))
		}
	})

	t.Run("returns error for non-existent tender", func(t *testing.T) {
		root := t.TempDir()

		err := RemoveTender(root, "non-existent")
		if err == nil {
			t.Fatal("expected error for non-existent tender")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func TestRenderWorkflow(t *testing.T) {
	t.Run("renders manual workflow", func(t *testing.T) {
		tender := Tender{
			Name:   "test-workflow",
			Agent:  "Build",
			Prompt: "test prompt",
			Manual: true,
		}

		result := RenderWorkflow(tender)

		requiredSnippets := []string{
			`name: "tender/test-workflow"`,
			`workflow_dispatch:`,
			`inputs:`,
			`prompt:`,
			`TENDER_NAME: "test-workflow"`,
			`TENDER_AGENT: "Build"`,
			`TENDER_PROMPT: "test prompt"`,
			`opencode run --agent "$TENDER_AGENT"`,
			`group: tender-main`,
		}

		for _, snippet := range requiredSnippets {
			if !strings.Contains(result, snippet) {
				t.Fatalf("workflow missing snippet %q:\n%s", snippet, result)
			}
		}

		// Should not contain schedule
		if strings.Contains(result, "schedule:") {
			t.Fatal("manual workflow should not contain schedule")
		}
	})

	t.Run("renders scheduled workflow", func(t *testing.T) {
		tender := Tender{
			Name:   "scheduled-workflow",
			Agent:  "Build",
			Prompt: "",
			Cron:   "0 9 * * 1",
			Manual: false,
		}

		result := RenderWorkflow(tender)

		requiredSnippets := []string{
			`name: "tender/scheduled-workflow"`,
			`schedule:`,
			`- cron: "0 9 * * 1"`,
			`TENDER_AGENT: "Build"`,
			`TENDER_PROMPT: ""`,
		}

		for _, snippet := range requiredSnippets {
			if !strings.Contains(result, snippet) {
				t.Fatalf("workflow missing snippet %q:\n%s", snippet, result)
			}
		}

		// Should not contain manual inputs
		if strings.Contains(result, "inputs:") {
			t.Fatal("scheduled workflow should not contain manual inputs")
		}
	})

	t.Run("renders workflow with both manual and schedule", func(t *testing.T) {
		tender := Tender{
			Name:   "hybrid-workflow",
			Agent:  "Build",
			Prompt: "hybrid prompt",
			Cron:   "30 14 * * *",
			Manual: true,
		}

		result := RenderWorkflow(tender)

		// Should contain both
		if !strings.Contains(result, "workflow_dispatch:") {
			t.Fatal("hybrid workflow should contain workflow_dispatch")
		}
		if !strings.Contains(result, "schedule:") {
			t.Fatal("hybrid workflow should contain schedule")
		}
		if !strings.Contains(result, "inputs:") {
			t.Fatal("hybrid workflow should contain manual inputs")
		}
	})

	t.Run("adds workflow_dispatch when neither manual nor schedule", func(t *testing.T) {
		tender := Tender{
			Name:   "minimal-workflow",
			Agent:  "Build",
			Prompt: "",
			Manual: false,
			Cron:   "",
		}

		result := RenderWorkflow(tender)

		// Should still have workflow_dispatch for basic functionality
		if !strings.Contains(result, "workflow_dispatch:") {
			t.Fatal("minimal workflow should contain workflow_dispatch")
		}
		// But no inputs
		if strings.Contains(result, "inputs:") {
			t.Fatal("minimal workflow should not contain inputs")
		}
	})
}

func TestParseTenderWorkflow(t *testing.T) {
	t.Run("when workflow is valid", func(t *testing.T) {
		t.Run("parses manual workflow with all fields", func(t *testing.T) {
			content := `name: "tender/test-workflow"
on:
  workflow_dispatch:
    inputs:
      prompt:
        description: "Optional prompt override"
        required: false
        default: ""
        type: string
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_NAME: "test-workflow"
      TENDER_AGENT: "Build"
      TENDER_PROMPT: "test prompt"
    steps:
      - name: Run OpenCode
        run: opencode run --agent "$TENDER_AGENT" "$RUN_PROMPT"
`

			tender, ok := parseTenderWorkflow(content)
			if !ok {
				t.Fatal("failed to parse valid workflow")
			}

			if tender.Name != "test-workflow" {
				t.Fatalf("unexpected name: %q", tender.Name)
			}
			if tender.Agent != "Build" {
				t.Fatalf("unexpected agent: %q", tender.Agent)
			}
			if tender.Prompt != "test prompt" {
				t.Fatalf("unexpected prompt: %q", tender.Prompt)
			}
			if !tender.Manual {
				t.Fatal("expected manual to be true")
			}
		})

		t.Run("parses scheduled workflow", func(t *testing.T) {
			content := `name: "tender/scheduled-workflow"
on:
  schedule:
    - cron: "0 9 * * 1"
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_NAME: "scheduled-workflow"
      TENDER_AGENT: "Build"
      TENDER_PROMPT: ""
    steps:
      - name: Run OpenCode
        run: opencode run --agent "$TENDER_AGENT" "$RUN_PROMPT"
`

			tender, ok := parseTenderWorkflow(content)
			if !ok {
				t.Fatal("failed to parse valid scheduled workflow")
			}

			if tender.Name != "scheduled-workflow" {
				t.Fatalf("unexpected name: %q", tender.Name)
			}
			if tender.Agent != "Build" {
				t.Fatalf("unexpected agent: %q", tender.Agent)
			}
			if tender.Cron != "0 9 * * 1" {
				t.Fatalf("unexpected cron: %q", tender.Cron)
			}
			if tender.Manual {
				t.Fatal("expected manual to be false")
			}
		})

		t.Run("parses workflow with both manual and schedule", func(t *testing.T) {
			content := `name: "tender/hybrid-workflow"
on:
  workflow_dispatch:
    inputs:
      prompt:
        description: "Optional prompt override"
        required: false
        default: ""
        type: string
  schedule:
    - cron: "0 9 * * *"
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_NAME: "hybrid-workflow"
      TENDER_AGENT: "Build"
      TENDER_PROMPT: "test prompt"
    steps:
      - name: Run OpenCode
        run: opencode run --agent "$TENDER_AGENT" "$RUN_PROMPT"
`

			tender, ok := parseTenderWorkflow(content)
			if !ok {
				t.Fatal("failed to parse hybrid workflow")
			}

			if tender.Name != "hybrid-workflow" {
				t.Fatalf("unexpected name: %q", tender.Name)
			}
			if !tender.Manual {
				t.Fatal("expected manual to be true for hybrid workflow")
			}
			if tender.Cron != "0 9 * * *" {
				t.Fatalf("unexpected cron: %q", tender.Cron)
			}
		})

		t.Run("infers name from agent when name is empty", func(t *testing.T) {
			content := `name: "tender/"
on:
  workflow_dispatch:
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_AGENT: "Build"
    steps:
      - name: Run OpenCode
        run: opencode run --agent "$TENDER_AGENT"
`

			tender, ok := parseTenderWorkflow(content)
			if !ok {
				t.Fatal("failed to parse workflow with empty name")
			}

			if tender.Name != "Build" {
				t.Fatalf("expected inferred name 'Build', got %q", tender.Name)
			}
		})
	})

	t.Run("when workflow is invalid", func(t *testing.T) {
		t.Run("rejects workflow without tender name prefix", func(t *testing.T) {
			content := `name: "other-workflow"
on:
  workflow_dispatch:
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_AGENT: "Build"
    steps:
      - name: Run OpenCode
        run: opencode run --agent "$TENDER_AGENT"
`

			_, ok := parseTenderWorkflow(content)
			if ok {
				t.Fatal("should reject workflow without tender name")
			}
		})

		t.Run("rejects workflow without agent", func(t *testing.T) {
			content := `name: "tender/test-workflow"
on:
  workflow_dispatch:
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_NAME: "test-workflow"
    steps:
      - name: Run OpenCode
        run: echo "no agent"
`

			_, ok := parseTenderWorkflow(content)
			if ok {
				t.Fatal("should reject workflow without agent")
			}
		})

		t.Run("rejects workflow without opencode run", func(t *testing.T) {
			content := `name: "tender/test-workflow"
on:
  workflow_dispatch:
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_NAME: "test-workflow"
      TENDER_AGENT: "Build"
    steps:
      - name: Do something else
        run: echo "not opencode"
`

			_, ok := parseTenderWorkflow(content)
			if ok {
				t.Fatal("should reject workflow without opencode run")
			}
		})
	})

	t.Run("when workflow has edge cases", func(t *testing.T) {
		t.Run("handles workflow with no TENDER_PROMPT env var", func(t *testing.T) {
			content := `name: "tender/missing-prompt"
on:
  workflow_dispatch:
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_NAME: "missing-prompt"
      TENDER_AGENT: "Build"
    steps:
      - name: Run OpenCode
        run: opencode run --agent "$TENDER_AGENT"
`

			tender, ok := parseTenderWorkflow(content)
			if !ok {
				t.Fatal("failed to parse workflow without prompt")
			}

			if tender.Prompt != "" {
				t.Fatalf("expected empty prompt, got %q", tender.Prompt)
			}
		})

		t.Run("parses workflow regardless of job name (current implementation)", func(t *testing.T) {
			content := `name: "tender/wrong-job"
on:
  workflow_dispatch:
jobs:
  not-tender:
    runs-on: ubuntu-latest
    env:
      TENDER_AGENT: "Build"
    steps:
      - name: Run OpenCode
        run: opencode run --agent "$TENDER_AGENT"
`

			tender, ok := parseTenderWorkflow(content)
			if !ok {
				t.Fatal("should parse workflow with opencode run regardless of job name")
			}

			if tender.Name != "wrong-job" {
				t.Fatalf("expected name 'wrong-job', got %q", tender.Name)
			}
		})
	})

	t.Run("when workflow is malformed", func(t *testing.T) {
		t.Run("rejects workflow with no jobs", func(t *testing.T) {
			content := `name: "tender/no-jobs"
on:
  workflow_dispatch:
`

			_, ok := parseTenderWorkflow(content)
			if ok {
				t.Fatal("should reject workflow with no jobs")
			}
		})
	})
}

func TestValidateTender(t *testing.T) {
	t.Run("when tender is valid", func(t *testing.T) {
		t.Run("accepts manual tender", func(t *testing.T) {
			tender := Tender{
				Name:   "valid-name",
				Agent:  "Build",
				Manual: true,
			}

			err := ValidateTender(tender)
			if err != nil {
				t.Fatalf("valid tender rejected: %v", err)
			}
		})

		t.Run("accepts scheduled tender", func(t *testing.T) {
			tender := Tender{
				Name:   "valid-name",
				Agent:  "Build",
				Cron:   "0 9 * * *",
				Manual: false,
			}

			err := ValidateTender(tender)
			if err != nil {
				t.Fatalf("valid scheduled tender rejected: %v", err)
			}
		})

		t.Run("accepts hybrid tender with both manual and schedule", func(t *testing.T) {
			tender := Tender{
				Name:   "hybrid-tender",
				Agent:  "Build",
				Cron:   "0 9 * * *",
				Manual: true,
			}

			err := ValidateTender(tender)
			if err != nil {
				t.Fatalf("valid hybrid tender rejected: %v", err)
			}
		})

		t.Run("accepts name with edge cases (current implementation)", func(t *testing.T) {
			cases := []struct {
				name   string
				reason string
			}{
				{"bad\tname", "tabs are not checked"},
				{"bad\\name", "backslashes are not checked"},
				{"  spaced-name  ", "whitespace is trimmed"},
			}

			for _, tc := range cases {
				t.Run(tc.reason, func(t *testing.T) {
					tender := Tender{
						Name:   tc.name,
						Agent:  "Build",
						Manual: true,
					}

					err := ValidateTender(tender)
					if err != nil {
						t.Fatalf("unexpected validation error for %q: %v", tc.name, err)
					}
				})
			}
		})
	})

	t.Run("when tender has invalid name", func(t *testing.T) {
		t.Run("rejects empty name", func(t *testing.T) {
			tender := Tender{
				Name:   "",
				Agent:  "Build",
				Manual: true,
			}

			err := ValidateTender(tender)
			if err == nil {
				t.Fatal("expected validation error for empty name")
			}
			if !strings.Contains(err.Error(), "name is required") {
				t.Fatalf("unexpected error message: %v", err)
			}
		})

		t.Run("rejects name with newlines", func(t *testing.T) {
			cases := []struct {
				name string
				desc string
			}{
				{"bad\nname", "newline character"},
				{"bad\rname", "carriage return character"},
			}

			for _, tc := range cases {
				t.Run(tc.desc, func(t *testing.T) {
					tender := Tender{
						Name:   tc.name,
						Agent:  "Build",
						Manual: true,
					}

					err := ValidateTender(tender)
					if err == nil {
						t.Fatal("expected validation error for name with newlines")
					}
					if !strings.Contains(err.Error(), "name cannot contain newlines") {
						t.Fatalf("unexpected error message: %v", err)
					}
				})
			}
		})

		t.Run("rejects name with forward slash", func(t *testing.T) {
			tender := Tender{
				Name:   "bad/name",
				Agent:  "Build",
				Manual: true,
			}

			err := ValidateTender(tender)
			if err == nil {
				t.Fatal("expected validation error for name with slash")
			}
			if !strings.Contains(err.Error(), "name cannot contain '/'") {
				t.Fatalf("unexpected error message: %v", err)
			}
		})
	})

	t.Run("when tender has invalid configuration", func(t *testing.T) {
		t.Run("rejects empty agent", func(t *testing.T) {
			tender := Tender{
				Name:   "test",
				Agent:  "",
				Manual: true,
			}

			err := ValidateTender(tender)
			if err == nil {
				t.Fatal("expected validation error for empty agent")
			}
			if !strings.Contains(err.Error(), "agent is required") {
				t.Fatalf("unexpected error message: %v", err)
			}
		})

		t.Run("rejects invalid cron format", func(t *testing.T) {
			tender := Tender{
				Name:   "test",
				Agent:  "Build",
				Cron:   "invalid",
				Manual: false,
			}

			err := ValidateTender(tender)
			if err == nil {
				t.Fatal("expected validation error for invalid cron")
			}
			if !strings.Contains(err.Error(), "cron must have 5 fields") {
				t.Fatalf("unexpected error message: %v", err)
			}
		})

		t.Run("rejects tender without manual or schedule", func(t *testing.T) {
			tender := Tender{
				Name:   "test",
				Agent:  "Build",
				Manual: false,
				Cron:   "",
			}

			err := ValidateTender(tender)
			if err == nil {
				t.Fatal("expected validation error for tender without trigger")
			}
			if !strings.Contains(err.Error(), "enable manual or set a schedule") {
				t.Fatalf("unexpected error message: %v", err)
			}
		})

		t.Run("rejects cron with too few fields", func(t *testing.T) {
			tender := Tender{
				Name:   "test",
				Agent:  "Build",
				Cron:   "0 9 * *", // Only 4 fields
				Manual: false,
			}

			err := ValidateTender(tender)
			if err == nil {
				t.Fatal("expected validation error for incomplete cron")
			}
			if !strings.Contains(err.Error(), "cron must have 5 fields") {
				t.Fatalf("unexpected error message: %v", err)
			}
		})

		t.Run("rejects cron with too many fields", func(t *testing.T) {
			tender := Tender{
				Name:   "test",
				Agent:  "Build",
				Cron:   "0 9 * * * extra", // 6 fields
				Manual: false,
			}

			err := ValidateTender(tender)
			if err == nil {
				t.Fatal("expected validation error for extra cron fields")
			}
			if !strings.Contains(err.Error(), "cron must have 5 fields") {
				t.Fatalf("unexpected error message: %v", err)
			}
		})

		t.Run("accepts cron with non-numeric fields (only validates field count)", func(t *testing.T) {
			tender := Tender{
				Name:   "test",
				Agent:  "Build",
				Cron:   "abc 9 * * *",
				Manual: false,
			}

			err := ValidateTender(tender)
			// Current implementation only validates field count, not content
			if err != nil {
				t.Fatalf("unexpected validation error for cron with non-numeric fields: %v", err)
			}
		})

		t.Run("accepts agent with whitespace", func(t *testing.T) {
			tender := Tender{
				Name:   "test",
				Agent:  "  Build Agent  ",
				Manual: true,
			}

			err := ValidateTender(tender)
			if err != nil {
				t.Fatalf("unexpected validation error for agent with whitespace: %v", err)
			}
		})

		t.Run("accepts cron with valid step values", func(t *testing.T) {
			tender := Tender{
				Name:   "test",
				Agent:  "Build",
				Cron:   "*/15 * * * *", // Valid step notation
				Manual: false,
			}

			err := ValidateTender(tender)
			if err != nil {
				t.Fatalf("unexpected validation error for valid step notation: %v", err)
			}
		})

		t.Run("accepts cron with range values", func(t *testing.T) {
			tender := Tender{
				Name:   "test",
				Agent:  "Build",
				Cron:   "0 9-17 * * *", // Valid hour range
				Manual: false,
			}

			err := ValidateTender(tender)
			if err != nil {
				t.Fatalf("unexpected validation error for valid range: %v", err)
			}
		})
	})
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"simple name", "simple-name"},
		{"Simple Name", "simple-name"},
		{"Name with spaces", "name-with-spaces"},
		{"name_with_underscores", "name-with-underscores"},
		{"name-with-dashes", "name-with-dashes"},
		{"Name   with   multiple   spaces", "name-with-multiple-spaces"},
		{"--leading--and--trailing--dashes--", "leading-and-trailing-dashes"},
		{"__leading__and__trailing__underscores__", "leading-and-trailing-underscores"},
		{"MixedCASE_with-Different SEPARATORS", "mixedcase-with-different-separators"},
		{"name123with456numbers", "name123with456numbers"},
		{"", "tender"},    // Empty string defaults to "tender"
		{"---", "tender"}, // Only separators defaults to "tender"
		{"   ", "tender"}, // Only whitespace defaults to "tender"
		{"Special!@#$%^&*()Characters", "specialcharacters"},
		{"Name-with-1-2-3-numbers", "name-with-1-2-3-numbers"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := Slugify(tc.input)
			if got != tc.want {
				t.Fatalf("Slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFindTenderIndex(t *testing.T) {
	tenders := []Tender{
		{Name: "first", Agent: "Build"},
		{Name: "second", Agent: "Test"},
		{Name: "third", Agent: "Deploy"},
	}

	t.Run("finds existing tender by name", func(t *testing.T) {
		idx := findTenderIndex(tenders, "second")
		if idx != 1 {
			t.Fatalf("expected index 1, got %d", idx)
		}
	})

	t.Run("finds existing tender case-insensitive", func(t *testing.T) {
		idx := findTenderIndex(tenders, "SECOND")
		if idx != 1 {
			t.Fatalf("expected index 1 for case-insensitive search, got %d", idx)
		}
	})

	t.Run("returns -1 for non-existent tender", func(t *testing.T) {
		idx := findTenderIndex(tenders, "non-existent")
		if idx != -1 {
			t.Fatalf("expected -1 for non-existent tender, got %d", idx)
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		idx := findTenderIndex([]Tender{}, "anything")
		if idx != -1 {
			t.Fatalf("expected -1 for empty list, got %d", idx)
		}
	})

	t.Run("handles whitespace in search", func(t *testing.T) {
		idx := findTenderIndex(tenders, "  second  ")
		if idx != 1 {
			t.Fatalf("expected index 1 for whitespace search, got %d", idx)
		}
	})
}

func TestParseQuotedValue(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{`"quoted value"`, "quoted value"},
		{`'single quoted'`, "single quoted"},
		{`"quoted with \"escape\""`, "quoted with \"escape\""},
		{`"quoted with \n newline"`, "quoted with \n newline"},
		{`unquoted value`, "unquoted value"},
		{`"`, ""},                // Malformed quote returns empty
		{``, ""},                 // Empty string
		{`"trimmed"`, "trimmed"}, // Quoted value
		{`'trimmed'`, "trimmed"}, // Single quotes
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := parseQuotedValue(tc.input)
			if got != tc.want {
				t.Fatalf("parseQuotedValue(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestPrintList(t *testing.T) {
	t.Run("prints empty list message", func(t *testing.T) {
		root := t.TempDir()
		var buf strings.Builder

		err := PrintList(root, &buf)
		if err != nil {
			t.Fatalf("PrintList returned error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "No managed tender workflows found") {
			t.Fatalf("expected empty list message, got: %s", output)
		}
	})

	t.Run("prints list of tenders", func(t *testing.T) {
		root := t.TempDir()

		// Create some tenders
		tenders := []Tender{
			{Name: "first", Agent: "Build", Manual: true, WorkflowFile: "first.yml"},
			{Name: "second", Agent: "Test", Cron: "0 9 * * *", WorkflowFile: "second.yml"},
		}

		for _, tender := range tenders {
			if err := SaveTender(root, tender); err != nil {
				t.Fatalf("failed to save tender: %v", err)
			}
		}

		var buf strings.Builder
		err := PrintList(root, &buf)
		if err != nil {
			t.Fatalf("PrintList returned error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "NAME\tAGENT\tTRIGGER\tWORKFLOW") {
			t.Fatal("expected header row")
		}
		if !strings.Contains(output, "first\tBuild\ton-demand\tfirst.yml") {
			t.Fatal("expected first tender row")
		}
		if !strings.Contains(output, "second\tTest\tdaily at 09:00 UTC\tsecond.yml") {
			t.Fatal("expected second tender row")
		}
	})
}

func TestManagedWorkflowPath(t *testing.T) {
	t.Run("returns path for existing tender", func(t *testing.T) {
		root := t.TempDir()

		tender := Tender{Name: "test", Agent: "Build", Manual: true}
		saved, err := SaveNewTender(root, tender)
		if err != nil {
			t.Fatalf("failed to save tender: %v", err)
		}

		path, err := ManagedWorkflowPath(root, "test")
		if err != nil {
			t.Fatalf("ManagedWorkflowPath returned error: %v", err)
		}

		expected := filepath.Join(root, WorkflowDir, saved.WorkflowFile)
		if path != expected {
			t.Fatalf("expected path %q, got %q", expected, path)
		}
	})

	t.Run("returns error for non-existent tender", func(t *testing.T) {
		root := t.TempDir()

		_, err := ManagedWorkflowPath(root, "non-existent")
		if err == nil {
			t.Fatal("expected error for non-existent tender")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func TestSortedCrons(t *testing.T) {
	t.Run("returns unique sorted crons", func(t *testing.T) {
		tenders := []Tender{
			{Name: "a", Agent: "Build", Cron: "0 9 * * *"},
			{Name: "b", Agent: "Test", Cron: "30 14 * * *"},
			{Name: "c", Agent: "Deploy", Cron: "0 9 * * *"}, // Duplicate
			{Name: "d", Agent: "Build", Cron: ""},           // Empty
			{Name: "e", Agent: "Test", Cron: "15 * * * *"},
		}

		crons := SortedCrons(tenders)
		expected := []string{"0 9 * * *", "15 * * * *", "30 14 * * *"}

		if len(crons) != len(expected) {
			t.Fatalf("expected %d crons, got %d", len(expected), len(crons))
		}
		for i, cron := range crons {
			if cron != expected[i] {
				t.Fatalf("cron %d: expected %q, got %q", i, expected[i], cron)
			}
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		crons := SortedCrons([]Tender{})
		if len(crons) != 0 {
			t.Fatalf("expected empty list, got %d crons", len(crons))
		}
	})

	t.Run("handles list with no crons", func(t *testing.T) {
		tenders := []Tender{
			{Name: "a", Agent: "Build", Manual: true},
			{Name: "b", Agent: "Test", Manual: true},
		}

		crons := SortedCrons(tenders)
		if len(crons) != 0 {
			t.Fatalf("expected empty list for manual tenders, got %d crons", len(crons))
		}
	})
}

func TestScanWorkflowHasTender(t *testing.T) {
	t.Run("detects tender workflow", func(t *testing.T) {
		content := `name: "test"
on:
  workflow_dispatch:
jobs:
  tender:
    runs-on: ubuntu-latest
    env:
      TENDER_AGENT: "Build"
    steps:
      - name: Run OpenCode
        run: opencode run --agent "$TENDER_AGENT"
`

		root := t.TempDir()
		workflowPath := filepath.Join(root, "workflow.yml")
		if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write workflow: %v", err)
		}

		has, err := ScanWorkflowHasTender(workflowPath)
		if err != nil {
			t.Fatalf("ScanWorkflowHasTender returned error: %v", err)
		}
		if !has {
			t.Fatal("expected to detect tender workflow")
		}
	})

	t.Run("rejects non-tender workflow", func(t *testing.T) {
		content := `name: "test"
on:
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Do something
        run: echo "not a tender"
`

		root := t.TempDir()
		workflowPath := filepath.Join(root, "workflow.yml")
		if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write workflow: %v", err)
		}

		has, err := ScanWorkflowHasTender(workflowPath)
		if err != nil {
			t.Fatalf("ScanWorkflowHasTender returned error: %v", err)
		}
		if has {
			t.Fatal("expected to reject non-tender workflow")
		}
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		_, err := ScanWorkflowHasTender("/non/existent/file.yml")
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}
	})
}

func TestFindUnusedWorkflowName(t *testing.T) {
	t.Run("returns base name when available", func(t *testing.T) {
		root := t.TempDir()

		name, err := findUnusedWorkflowName(root, "test")
		if err != nil {
			t.Fatalf("findUnusedWorkflowName returned error: %v", err)
		}
		if name != "test.yml" {
			t.Fatalf("expected 'test.yml', got %q", name)
		}
	})

	t.Run("finds unused name with suffix", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Create existing files
		existing := []string{"test.yml", "test-2.yml"}
		for _, file := range existing {
			path := filepath.Join(root, WorkflowDir, file)
			if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
				t.Fatalf("failed to create %s: %v", file, err)
			}
		}

		name, err := findUnusedWorkflowName(root, "test")
		if err != nil {
			t.Fatalf("findUnusedWorkflowName returned error: %v", err)
		}
		if name != "test-3.yml" {
			t.Fatalf("expected 'test-3.yml', got %q", name)
		}
	})

	t.Run("slugifies base name", func(t *testing.T) {
		root := t.TempDir()

		name, err := findUnusedWorkflowName(root, "Test With Spaces")
		if err != nil {
			t.Fatalf("findUnusedWorkflowName returned error: %v", err)
		}
		if name != "test-with-spaces.yml" {
			t.Fatalf("expected 'test-with-spaces.yml', got %q", name)
		}
	})
}

// Error handling tests

func TestWorkflowErrorHandling(t *testing.T) {
	t.Run("when saving to read-only directory", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Make workflow directory read-only
		workflowDir := filepath.Join(root, WorkflowDir)
		if err := os.Chmod(workflowDir, 0o444); err != nil {
			t.Fatalf("failed to make directory read-only: %v", err)
		}
		defer func() {
			_ = os.Chmod(workflowDir, 0o755) // Restore permissions for cleanup
		}()

		tender := Tender{
			Name:   "test-tender",
			Agent:  "Build",
			Manual: true,
		}

		err := SaveTender(root, tender)
		if err == nil {
			t.Fatal("expected error when saving to read-only directory")
		}
	})

	t.Run("when loading from corrupted directory", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Create an unreadable file
		workflowPath := filepath.Join(root, WorkflowDir, "unreadable.yml")
		if err := os.WriteFile(workflowPath, []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to create workflow file: %v", err)
		}

		// Make the file unreadable
		if err := os.Chmod(workflowPath, 0o000); err != nil {
			t.Fatalf("failed to make file unreadable: %v", err)
		}
		defer func() {
			_ = os.Chmod(workflowPath, 0o644) // Restore permissions for cleanup
		}()

		// LoadTenders returns an error when it can't read files
		tenders, err := LoadTenders(root)
		if err == nil {
			t.Fatal("expected error when loading unreadable files")
		}
		// Should still return empty list on error
		if len(tenders) != 0 {
			t.Fatalf("expected empty list when files are unreadable, got %d tenders", len(tenders))
		}
	})

	t.Run("when removing non-existent workflow file", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// Try to remove tender that doesn't exist
		err := RemoveTender(root, "ghost-tender")
		// This should return an error
		if err == nil {
			t.Fatal("expected error when removing non-existent tender")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected 'not found' error, got: %v", err)
		}
	})

	t.Run("when scanning malformed YAML file", func(t *testing.T) {
		root := t.TempDir()

		// Create a malformed YAML file
		workflowPath := filepath.Join(root, "malformed.yml")
		malformedContent := `name: "tender/test"
on:
  workflow_dispatch:
invalid yaml: [unclosed bracket
jobs:`
		if err := os.WriteFile(workflowPath, []byte(malformedContent), 0o644); err != nil {
			t.Fatalf("failed to create malformed workflow: %v", err)
		}

		has, err := ScanWorkflowHasTender(workflowPath)
		// Should handle malformed YAML gracefully
		// The function might return (false, nil) if it can't parse, or an error
		if err == nil && has {
			t.Fatal("should not detect tender in malformed file")
		}
		// Either error or false is acceptable
	})
}

func TestEdgeCaseHandling(t *testing.T) {
	t.Run("when workflow filename contains special characters", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		tender := Tender{
			Name:         "test@#$%",
			Agent:        "Build",
			Manual:       true,
			WorkflowFile: "special-chars.yml",
		}

		// Save should handle the name by slugifying it
		err := SaveTender(root, tender)
		if err != nil {
			t.Fatalf("failed to save tender with special chars: %v", err)
		}

		// Load and verify it was saved correctly
		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("failed to load tenders: %v", err)
		}
		if len(tenders) != 1 {
			t.Fatalf("expected 1 tender, got %d", len(tenders))
		}
	})

	t.Run("when agent name contains spaces", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		tender := Tender{
			Name:   "test-tender",
			Agent:  "Build Agent",
			Manual: true,
		}

		// Should accept agent names with spaces
		err := SaveTender(root, tender)
		if err != nil {
			t.Fatalf("failed to save tender with spaced agent: %v", err)
		}

		// Load and verify
		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("failed to load tenders: %v", err)
		}
		if len(tenders) != 1 {
			t.Fatalf("expected 1 tender, got %d", len(tenders))
		}
		if tenders[0].Agent != "Build Agent" {
			t.Fatalf("expected agent 'Build Agent', got %q", tenders[0].Agent)
		}
	})

	t.Run("when cron expression has unusual values", func(t *testing.T) {
		tender := Tender{
			Name:   "unusual-cron",
			Agent:  "Build",
			Cron:   "*/15 * * * *", // Every 15 minutes
			Manual: false,
		}

		// Should accept valid cron expressions
		err := ValidateTender(tender)
		if err != nil {
			t.Fatalf("valid cron expression rejected: %v", err)
		}
	})

	t.Run("when name has leading/trailing whitespace", func(t *testing.T) {
		root := t.TempDir()
		tender := Tender{
			Name:   "  spaced name  ",
			Agent:  "Build",
			Manual: true,
		}

		err := SaveTender(root, tender)
		if err != nil {
			t.Fatalf("failed to save tender with whitespace: %v", err)
		}

		// Load and verify whitespace was trimmed
		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("failed to load tenders: %v", err)
		}
		if len(tenders) != 1 {
			t.Fatalf("expected 1 tender, got %d", len(tenders))
		}
		if tenders[0].Name != "spaced name" {
			t.Fatalf("expected trimmed name 'spaced name', got %q", tenders[0].Name)
		}
	})

	t.Run("when prompt contains special characters", func(t *testing.T) {
		root := t.TempDir()
		tender := Tender{
			Name:   "test",
			Agent:  "Build",
			Prompt: "Test with quotes: \"hello\" and 'world'",
			Manual: true,
		}

		err := SaveTender(root, tender)
		if err != nil {
			t.Fatalf("failed to save tender with special prompt: %v", err)
		}

		// Verify prompt is preserved in workflow (check for quoted version)
		workflowPath := filepath.Join(root, WorkflowDir, "test.yml")
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("failed to read workflow: %v", err)
		}
		workflowText := string(content)
		// The prompt gets quoted in YAML, so check for the quoted version
		if !strings.Contains(workflowText, `TENDER_PROMPT: "Test with quotes: \"hello\" and 'world'"`) {
			t.Fatalf("prompt with special characters not preserved. Got:\n%s", workflowText)
		}
	})

	t.Run("when workflow file has no extension", func(t *testing.T) {
		root := t.TempDir()
		tender := Tender{
			Name:         "test",
			Agent:        "Build",
			Manual:       true,
			WorkflowFile: "no-ext",
		}

		err := SaveTender(root, tender)
		if err != nil {
			t.Fatalf("failed to save tender with no extension: %v", err)
		}

		// Should add .yml extension
		workflowPath := filepath.Join(root, WorkflowDir, "no-ext.yml")
		if _, err := os.Stat(workflowPath); err != nil {
			t.Fatalf("expected file with .yml extension: %v", err)
		}
	})

	t.Run("when workflow file has .yaml extension", func(t *testing.T) {
		root := t.TempDir()
		tender := Tender{
			Name:         "test",
			Agent:        "Build",
			Manual:       true,
			WorkflowFile: "test.yaml",
		}

		err := SaveTender(root, tender)
		if err != nil {
			t.Fatalf("failed to save tender with .yaml extension: %v", err)
		}

		// Should preserve .yaml extension
		workflowPath := filepath.Join(root, WorkflowDir, "test.yaml")
		if _, err := os.Stat(workflowPath); err != nil {
			t.Fatalf("expected file with .yaml extension: %v", err)
		}
	})
}

// Helper functions

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
