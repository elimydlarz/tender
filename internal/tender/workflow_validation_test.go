package tender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Workflow content validation tests

func TestWorkflowContentValidation(t *testing.T) {
	t.Run("validates rendered workflow structure", func(t *testing.T) {
		tender := Tender{
			Name:   "test-workflow",
			Agent:  "Build",
			Prompt: "test prompt",
			Manual: true,
		}

		workflow := RenderWorkflow(tender)

		// Check for required workflow sections
		requiredSections := []string{
			"name:",
			"on:",
			"permissions:",
			"concurrency:",
			"jobs:",
		}

		for _, section := range requiredSections {
			if !strings.Contains(workflow, section) {
				t.Fatalf("workflow missing required section: %s\nWorkflow content:\n%s", section, workflow)
			}
		}

		// Check for tender-specific content
		tenderSpecific := []string{
			`name: "tender/test-workflow"`,
			"TENDER_NAME: \"test-workflow\"",
			"TENDER_AGENT: \"Build\"",
			"TENDER_PROMPT: \"test prompt\"",
			"timeout-minutes: 30",
			"opencode run",
			"group: tender-main",
		}

		for _, content := range tenderSpecific {
			if !strings.Contains(workflow, content) {
				t.Fatalf("workflow missing tender-specific content: %s\nWorkflow content:\n%s", content, workflow)
			}
		}
	})

	t.Run("validates scheduled workflow structure", func(t *testing.T) {
		tender := Tender{
			Name:   "scheduled-workflow",
			Agent:  "Deploy",
			Cron:   "0 9 * * *",
			Manual: false,
		}

		workflow := RenderWorkflow(tender)

		// Should contain schedule section
		if !strings.Contains(workflow, "schedule:") {
			t.Fatal("scheduled workflow should contain schedule section")
		}
		if !strings.Contains(workflow, "- cron: \"0 9 * * *\"") {
			t.Fatal("scheduled workflow should contain cron expression")
		}

		// Should not contain manual inputs
		if strings.Contains(workflow, "inputs:") {
			t.Fatal("scheduled workflow should not contain manual inputs")
		}
	})

	t.Run("validates hybrid workflow structure", func(t *testing.T) {
		tender := Tender{
			Name:   "hybrid-workflow",
			Agent:  "Test",
			Cron:   "30 14 * * 1,3,5",
			Manual: true,
		}

		workflow := RenderWorkflow(tender)

		// Should contain both manual and schedule
		if !strings.Contains(workflow, "workflow_dispatch:") {
			t.Fatal("hybrid workflow should contain workflow_dispatch")
		}
		if !strings.Contains(workflow, "schedule:") {
			t.Fatal("hybrid workflow should contain schedule")
		}
		if !strings.Contains(workflow, "inputs:") {
			t.Fatal("hybrid workflow should contain manual inputs")
		}
	})

	t.Run("validates workflow file creation and content", func(t *testing.T) {
		root := t.TempDir()
		tender := Tender{
			Name:   "validation-test",
			Agent:  "Build",
			Prompt: "validation prompt",
			Manual: true,
		}

		// Save the workflow
		err := SaveTender(root, tender)
		if err != nil {
			t.Fatalf("failed to save tender: %v", err)
		}

		// Verify file exists
		workflowPath := filepath.Join(root, WorkflowDir, "validation-test.yml")
		if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
			t.Fatal("workflow file was not created")
		}

		// Read and validate content
		content, err := os.ReadFile(workflowPath)
		if err != nil {
			t.Fatalf("failed to read workflow file: %v", err)
		}

		workflowText := string(content)
		if !strings.Contains(workflowText, "tender/validation-test") {
			t.Fatal("workflow file should contain tender name")
		}
		if !strings.Contains(workflowText, "TENDER_AGENT: \"Build\"") {
			t.Fatal("workflow file should contain agent")
		}
	})

	t.Run("validates workflow round-trip consistency", func(t *testing.T) {
		root := t.TempDir()
		original := Tender{
			Name:   "round-trip-test",
			Agent:  "Test Agent",
			Prompt: "Round trip test prompt",
			Cron:   "15 */2 * * *",
			Manual: true,
		}

		// Save and reload
		_, err := SaveNewTender(root, original)
		if err != nil {
			t.Fatalf("failed to save tender: %v", err)
		}

		loaded, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("failed to load tenders: %v", err)
		}
		if len(loaded) != 1 {
			t.Fatalf("expected 1 tender, got %d", len(loaded))
		}

		reloaded := loaded[0]

		// Verify key fields are preserved
		if reloaded.Name != original.Name {
			t.Fatalf("name not preserved: expected %q, got %q", original.Name, reloaded.Name)
		}
		if reloaded.Agent != original.Agent {
			t.Fatalf("agent not preserved: expected %q, got %q", original.Agent, reloaded.Agent)
		}
		if reloaded.Prompt != original.Prompt {
			t.Fatalf("prompt not preserved: expected %q, got %q", original.Prompt, reloaded.Prompt)
		}
		if reloaded.Cron != original.Cron {
			t.Fatalf("cron not preserved: expected %q, got %q", original.Cron, reloaded.Cron)
		}
		if reloaded.Manual != original.Manual {
			t.Fatalf("manual flag not preserved: expected %v, got %v", original.Manual, reloaded.Manual)
		}
		if reloaded.TimeoutMinutes != DefaultTimeoutMinutes {
			t.Fatalf("timeout-minutes not preserved: expected %d, got %d", DefaultTimeoutMinutes, reloaded.TimeoutMinutes)
		}
	})

	t.Run("validates workflow permissions and security", func(t *testing.T) {
		tender := Tender{
			Name:   "security-test",
			Agent:  "Build",
			Manual: true,
		}

		workflow := RenderWorkflow(tender)

		// Check for proper permissions
		if !strings.Contains(workflow, "contents: write") {
			t.Fatal("workflow should have contents: write permission")
		}

		// Check for concurrency control
		if !strings.Contains(workflow, "group: tender-main") {
			t.Fatal("workflow should use tender-main concurrency group")
		}

		// Check for proper checkout
		if !strings.Contains(workflow, "actions/checkout@v4") {
			t.Fatal("workflow should checkout code")
		}
		if !strings.Contains(workflow, "fetch-depth: 0") {
			t.Fatal("workflow should fetch full history")
		}
		if !strings.Contains(workflow, "timeout-minutes: 30") {
			t.Fatal("workflow should set default timeout-minutes")
		}
	})

	t.Run("validates workflow environment variables", func(t *testing.T) {
		tender := Tender{
			Name:   "env-test",
			Agent:  "Test Agent",
			Prompt: "Test prompt with special chars: \"quotes\" and 'apostrophes'",
			Manual: true,
		}

		workflow := RenderWorkflow(tender)

		// Check that environment variables are properly quoted
		if !strings.Contains(workflow, `TENDER_NAME: "env-test"`) {
			t.Fatal("workflow should have quoted TENDER_NAME")
		}
		if !strings.Contains(workflow, `TENDER_AGENT: "Test Agent"`) {
			t.Fatal("workflow should have quoted TENDER_AGENT")
		}
		if !strings.Contains(workflow, `TENDER_PROMPT: "Test prompt with special chars: \"quotes\" and 'apostrophes'"`) {
			t.Fatal("workflow should have quoted TENDER_PROMPT with escaped quotes")
		}
	})

	t.Run("validates workflow step structure", func(t *testing.T) {
		tender := Tender{
			Name:   "steps-test",
			Agent:  "Build",
			Manual: true,
		}

		workflow := RenderWorkflow(tender)

		// Check for required steps
		requiredSteps := []string{
			"- uses: actions/checkout@v4",
			"- name: Install OpenCode",
			"- name: Prepare main",
			"- name: Run OpenCode",
			"- name: Commit and push main",
		}

		for _, step := range requiredSteps {
			if !strings.Contains(workflow, step) {
				t.Fatalf("workflow missing required step: %s", step)
			}
		}

		// Check for OpenCode installation
		if !strings.Contains(workflow, "curl -fsSL https://opencode.ai/install | bash") {
			t.Fatal("workflow should install OpenCode")
		}

		// Check for git configuration
		if !strings.Contains(workflow, "git config user.name \"tender[bot]\"") {
			t.Fatal("workflow should configure git user")
		}
		if !strings.Contains(workflow, "git config user.email \"tender[bot]@users.noreply.github.com\"") {
			t.Fatal("workflow should configure git email")
		}
	})
}

func TestWorkflowYAMLStructure(t *testing.T) {
	t.Run("produces valid YAML syntax", func(t *testing.T) {
		tender := Tender{
			Name:   "yaml-test",
			Agent:  "Build",
			Prompt: "YAML validation test",
			Manual: true,
		}

		workflow := RenderWorkflow(tender)

		// Basic YAML structure checks
		lines := strings.Split(workflow, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue // Skip empty lines
			}

			// Check for proper indentation (basic check)
			if strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "  ") {
				t.Fatalf("line %d has inconsistent indentation: %q", i+1, line)
			}

			// Check that key-value pairs are properly formatted
			if strings.Contains(trimmed, ":") && !strings.HasSuffix(trimmed, ":") {
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) != 2 {
					t.Fatalf("line %d has malformed key-value pair: %q", i+1, line)
				}
			}
		}
	})

	t.Run("properly escapes special characters in YAML", func(t *testing.T) {
		tender := Tender{
			Name:   "escape-test",
			Agent:  "Build",
			Prompt: "Test with special chars: \n\t\"quotes\"",
			Manual: true,
		}

		workflow := RenderWorkflow(tender)

		// Check that the prompt is properly quoted and escaped
		expectedPrompt := `TENDER_PROMPT: "Test with special chars: \n\t\"quotes\""`
		if !strings.Contains(workflow, expectedPrompt) {
			t.Fatal("workflow should properly escape special characters in prompt")
		}

		// Check that name is properly quoted
		if !strings.Contains(workflow, `name: "tender/escape-test"`) {
			t.Fatal("workflow should properly quote name")
		}
	})

	t.Run("handles edge cases in content", func(t *testing.T) {
		testCases := []struct {
			name   string
			tender Tender
		}{
			{
				name: "empty prompt",
				tender: Tender{
					Name:   "empty-prompt",
					Agent:  "Build",
					Prompt: "",
					Manual: true,
				},
			},
			{
				name: "prompt with newlines",
				tender: Tender{
					Name:   "newline-prompt",
					Agent:  "Build",
					Prompt: "Line 1\nLine 2\nLine 3",
					Manual: true,
				},
			},
			{
				name: "agent with spaces",
				tender: Tender{
					Name:   "spaced-agent",
					Agent:  "Build Agent",
					Prompt: "test",
					Manual: true,
				},
			},
			{
				name: "complex cron",
				tender: Tender{
					Name:   "complex-cron",
					Agent:  "Deploy",
					Cron:   "*/15 2,14 * * 1-5",
					Manual: false,
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				workflow := RenderWorkflow(tc.tender)

				// Should not contain any obvious YAML syntax errors
				if strings.Contains(workflow, "  :") {
					t.Fatalf("workflow contains empty key: %s", workflow)
				}

				// Should contain the tender name
				if !strings.Contains(workflow, "tender/"+tc.tender.Name) {
					t.Fatalf("workflow missing tender name: %s", workflow)
				}

				// Should contain the agent
				expectedAgent := `TENDER_AGENT: "` + tc.tender.Agent + `"`
				if !strings.Contains(workflow, expectedAgent) {
					t.Fatalf("workflow missing agent: %s", workflow)
				}
			})
		}
	})
}
