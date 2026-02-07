package tender

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func DispatchTenderNow(root string, tenderName string, prompt string, stdout io.Writer, stderr io.Writer) error {
	tenders, err := LoadTenders(root)
	if err != nil {
		return err
	}
	idx := findTenderIndex(tenders, tenderName)
	if idx < 0 {
		return fmt.Errorf("tender %q not found", tenderName)
	}
	t := tenders[idx]
	if !t.Manual {
		return fmt.Errorf("tender %q is schedule-only; enable on-demand runs to use 'tender run'", tenderName)
	}

	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI 'gh' is required to run a tender now")
	}

	args := buildGHWorkflowRunArgs(t, prompt)
	cmd := exec.Command("gh", args...)
	cmd.Dir = root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh workflow dispatch failed: %w", err)
	}
	return nil
}

func buildGHWorkflowRunArgs(t Tender, prompt string) []string {
	args := []string{"workflow", "run", t.WorkflowFile}
	if strings.TrimSpace(prompt) != "" {
		args = append(args, "-f", "prompt="+strings.TrimSpace(prompt))
	}
	return args
}
