package tender

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// workflow_integration_test.go - Integration tests for complete workflow scenarios

func TestWorkflowIntegration_CompleteLifecycle(t *testing.T) {
	t.Run("creates, lists, updates, and removes tender workflow", func(t *testing.T) {
		root := t.TempDir()

		// Step 1: Create a new tender
		original := Tender{
			Name:   "integration-test",
			Agent:  "Build",
			Prompt: "Initial test prompt",
			Manual: true,
			Cron:   "0 9 * * *", // Daily at 9:00 UTC
		}

		saved, err := SaveNewTender(root, original)
		if err != nil {
			t.Fatalf("failed to create tender: %v", err)
		}

		// Verify workflow file was created
		workflowPath := filepath.Join(root, WorkflowDir, saved.WorkflowFile)
		if _, err := os.Stat(workflowPath); err != nil {
			t.Fatalf("workflow file not created: %v", err)
		}

		// Step 2: List tenders and verify our tender is there
		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("failed to load tenders: %v", err)
		}
		if len(tenders) != 1 {
			t.Fatalf("expected 1 tender, got %d", len(tenders))
		}
		if tenders[0].Name != "integration-test" {
			t.Fatalf("expected tender name 'integration-test', got %q", tenders[0].Name)
		}

		// Step 3: Update the tender
		updated := Tender{
			Name:   "renamed-integration-test",
			Agent:  "Test",
			Prompt: "Updated test prompt",
			Manual: false,             // Change to schedule-only
			Cron:   "30 14 * * 1,3,5", // Mon,Wed,Fri at 14:30 UTC
		}

		err = UpdateTender(root, "integration-test", updated)
		if err != nil {
			t.Fatalf("failed to update tender: %v", err)
		}

		// Step 4: Verify update
		tenders, err = LoadTenders(root)
		if err != nil {
			t.Fatalf("failed to load tenders after update: %v", err)
		}
		if len(tenders) != 1 {
			t.Fatalf("expected 1 tender after update, got %d", len(tenders))
		}
		if tenders[0].Name != "renamed-integration-test" {
			t.Fatalf("expected updated name 'renamed-integration-test', got %q", tenders[0].Name)
		}
		if tenders[0].Agent != "Test" {
			t.Fatalf("expected updated agent 'Test', got %q", tenders[0].Agent)
		}
		if tenders[0].Manual {
			t.Fatal("expected manual to be false after update")
		}

		// Step 5: Verify workflow content was updated
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("failed to read updated workflow: %v", err)
		}
		workflowText := string(content)
		if !strings.Contains(workflowText, "renamed-integration-test") {
			t.Fatal("workflow not updated with new name")
		}
		if !strings.Contains(workflowText, "30 14 * * 1,3,5") {
			t.Fatal("workflow not updated with new cron")
		}
		if strings.Contains(workflowText, "workflow_dispatch:") {
			t.Fatal("workflow should not have manual trigger after update")
		}

		// Step 6: Remove the tender
		err = RemoveTender(root, "renamed-integration-test")
		if err != nil {
			t.Fatalf("failed to remove tender: %v", err)
		}

		// Step 7: Verify removal
		if _, err := os.Stat(workflowPath); err == nil {
			t.Fatal("workflow file still exists after removal")
		}
		tenders, err = LoadTenders(root)
		if err != nil {
			t.Fatalf("failed to load tenders after removal: %v", err)
		}
		if len(tenders) != 0 {
			t.Fatalf("expected 0 tenders after removal, got %d", len(tenders))
		}
	})
}

func TestWorkflowIntegration_MultipleTenders(t *testing.T) {
	t.Run("manages multiple tenders with different configurations", func(t *testing.T) {
		root := t.TempDir()

		// Create multiple tenders with different configurations
		tenders := []Tender{
			{
				Name:   "daily-build",
				Agent:  "Build",
				Manual: true,
				Cron:   "0 8 * * *", // Daily at 8:00 UTC + manual
			},
			{
				Name:   "weekly-test",
				Agent:  "Test",
				Manual: false,
				Cron:   "0 12 * * 6", // Saturday at noon UTC only
			},
			{
				Name:   "manual-deploy",
				Agent:  "Deploy",
				Manual: true,
				Cron:   "", // Manual only
			},
		}

		// Save all tenders
		for i, tender := range tenders {
			_, err := SaveNewTender(root, tender)
			if err != nil {
				t.Fatalf("failed to save tender %d: %v", i, err)
			}
		}

		// Verify all tenders are loaded and sorted
		loaded, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("failed to load tenders: %v", err)
		}
		if len(loaded) != 3 {
			t.Fatalf("expected 3 tenders, got %d", len(loaded))
		}

		// Check sorting (should be alphabetical)
		expectedOrder := []string{"daily-build", "manual-deploy", "weekly-test"}
		for i, expected := range expectedOrder {
			if loaded[i].Name != expected {
				t.Fatalf("tender %d: expected name %q, got %q", i, expected, loaded[i].Name)
			}
		}

		// Verify each tender has correct workflow file
		for _, tender := range loaded {
			workflowPath := filepath.Join(root, WorkflowDir, tender.WorkflowFile)
			if _, err := os.Stat(workflowPath); err != nil {
				t.Fatalf("workflow file missing for %q: %v", tender.Name, err)
			}
		}

		// Test SortedCrons function
		crons := SortedCrons(loaded)
		expectedCrons := []string{"0 12 * * 6", "0 8 * * *"} // Should be sorted alphabetically
		if len(crons) != 2 {
			t.Fatalf("expected 2 unique crons, got %d", len(crons))
		}
		for i, expected := range expectedCrons {
			if crons[i] != expected {
				t.Fatalf("cron %d: expected %q, got %q", i, expected, crons[i])
			}
		}
	})
}

func TestWorkflowIntegration_ErrorRecovery(t *testing.T) {
	t.Run("handles errors gracefully during operations", func(t *testing.T) {
		root := t.TempDir()

		// Test 1: Try to update non-existent tender
		nonExistent := Tender{Name: "non-existent", Agent: "Build"}
		err := UpdateTender(root, "non-existent", nonExistent)
		if err == nil {
			t.Fatal("expected error when updating non-existent tender")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected 'not found' error, got: %v", err)
		}

		// Test 2: Try to remove non-existent tender
		err = RemoveTender(root, "non-existent")
		if err == nil {
			t.Fatal("expected error when removing non-existent tender")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected 'not found' error, got: %v", err)
		}

		// Test 3: Create tender with invalid data
		invalid := Tender{Name: "", Agent: ""} // Empty name and agent
		_, err = SaveNewTender(root, invalid)
		if err == nil {
			t.Fatal("expected error when saving invalid tender")
		}

		// Test 4: Create valid tender, then try to create duplicate
		valid := Tender{Name: "test", Agent: "Build", Manual: true}
		_, err = SaveNewTender(root, valid)
		if err != nil {
			t.Fatalf("failed to save valid tender: %v", err)
		}

		duplicate := Tender{Name: "test", Agent: "Test", Manual: true}
		_, err = SaveNewTender(root, duplicate)
		if err == nil {
			t.Fatal("expected error when creating duplicate tender")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("expected 'already exists' error, got: %v", err)
		}

		// Test 5: Try to rename to existing name
		err = UpdateTender(root, "test", Tender{Name: "test", Agent: "Updated"})
		if err == nil {
			t.Fatal("expected error when updating to same name")
		}
	})
}

func TestWorkflowIntegration_WorkflowContentValidation(t *testing.T) {
	t.Run("generates valid GitHub Actions workflow content", func(t *testing.T) {
		root := t.TempDir()

		// Create a comprehensive tender
		tender := Tender{
			Name:   "comprehensive-test",
			Agent:  "Build",
			Prompt: "Test with special chars: \"quotes\" and 'apostrophes'",
			Manual: true,
			Cron:   "30 14 * * 1,3,5", // Mon,Wed,Fri at 14:30 UTC
		}

		saved, err := SaveNewTender(root, tender)
		if err != nil {
			t.Fatalf("failed to save tender: %v", err)
		}

		// Read and validate workflow content
		workflowPath := filepath.Join(root, WorkflowDir, saved.WorkflowFile)
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("failed to read workflow: %v", err)
		}

		workflowText := string(content)

		// Required workflow elements
		requiredElements := map[string]string{
			"name:":                                  `"tender/comprehensive-test"`,
			"workflow_dispatch:":                     "should have manual trigger",
			"schedule:":                              "should have schedule trigger",
			"- cron:":                                `"30 14 * * 1,3,5"`,
			"permissions:":                           "contents: write",
			"concurrency:":                           "group: tender-main",
			"TENDER_NAME:":                           `"comprehensive-test"`,
			"TENDER_AGENT:":                          `"Build"`,
			"TENDER_PROMPT:":                         `Test with special chars: \"quotes\" and 'apostrophes'`,
			"timeout-minutes: 30":                    "should default timeout to 30 minutes",
			"actions/checkout@v4":                    "should checkout code",
			"curl -fsSL https://opencode.ai/install": "should install OpenCode",
			"opencode run":                           "should run OpenCode",
			"git push origin HEAD:main":              "should push to main",
		}

		for element, description := range requiredElements {
			if !strings.Contains(workflowText, element) {
				t.Fatalf("workflow missing required element %s (%s)", element, description)
			}
		}

		// Verify it can be parsed back
		parsed, ok := parseTenderWorkflow(workflowText)
		if !ok {
			t.Fatal("generated workflow cannot be parsed back")
		}
		if parsed.Name != tender.Name {
			t.Fatalf("parsed name mismatch: expected %q, got %q", tender.Name, parsed.Name)
		}
		if parsed.Agent != tender.Agent {
			t.Fatalf("parsed agent mismatch: expected %q, got %q", tender.Agent, parsed.Agent)
		}
		if !parsed.Manual {
			t.Fatal("parsed manual should be true")
		}
		if parsed.Cron != tender.Cron {
			t.Fatalf("parsed cron mismatch: expected %q, got %q", tender.Cron, parsed.Cron)
		}
		if parsed.TimeoutMinutes != DefaultTimeoutMinutes {
			t.Fatalf("parsed timeout mismatch: expected %d, got %d", DefaultTimeoutMinutes, parsed.TimeoutMinutes)
		}
	})
}

func TestWorkflowIntegration_PerformanceAndScalability(t *testing.T) {
	t.Run("handles large numbers of tenders efficiently", func(t *testing.T) {
		root := t.TempDir()

		// Create many tenders
		const numTenders = 50
		for i := 0; i < numTenders; i++ {
			tender := Tender{
				Name:   fmt.Sprintf("tender-%03d", i),
				Agent:  "Build",
				Manual: i%2 == 0, // Alternate between manual and schedule-only
				Cron:   fmt.Sprintf("%d %d * * *", i%60, i%24),
			}
			_, err := SaveNewTender(root, tender)
			if err != nil {
				t.Fatalf("failed to save tender %d: %v", i, err)
			}
		}

		// Test loading performance
		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("failed to load tenders: %v", err)
		}
		if len(tenders) != numTenders {
			t.Fatalf("expected %d tenders, got %d", numTenders, len(tenders))
		}

		// Verify sorting
		for i := 1; i < len(tenders); i++ {
			if tenders[i].Name < tenders[i-1].Name {
				t.Fatalf("tenders not properly sorted: %q should come after %q",
					tenders[i].Name, tenders[i-1].Name)
			}
		}

		// Test finding tenders by name
		for i := 0; i < 10; i++ { // Test a subset
			name := fmt.Sprintf("tender-%03d", i*5)
			idx := findTenderIndex(tenders, name)
			if idx < 0 {
				t.Fatalf("tender %q not found", name)
			}
			if tenders[idx].Name != name {
				t.Fatalf("found wrong tender at index %d: expected %q, got %q",
					idx, name, tenders[idx].Name)
			}
		}

		// Test updating a tender in the middle
		middleIndex := numTenders / 2
		originalName := tenders[middleIndex].Name
		updated := Tender{
			Name:   "updated-" + originalName,
			Agent:  "Test",
			Manual: true,
			Cron:   "0 0 * * *",
		}

		err = UpdateTender(root, originalName, updated)
		if err != nil {
			t.Fatalf("failed to update tender: %v", err)
		}

		// Verify update and re-sorting
		tenders, err = LoadTenders(root)
		if err != nil {
			t.Fatalf("failed to reload tenders: %v", err)
		}
		if len(tenders) != numTenders {
			t.Fatalf("expected %d tenders after update, got %d", numTenders, len(tenders))
		}

		// Find the updated tender
		found := false
		for _, tender := range tenders {
			if tender.Name == updated.Name {
				found = true
				if tender.Agent != "Test" {
					t.Fatalf("updated tender agent not preserved")
				}
				break
			}
		}
		if !found {
			t.Fatal("updated tender not found in list")
		}
	})
}
