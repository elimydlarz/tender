//go:build acceptance

package tender

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAcceptanceWorkflowDispatchAndScheduleVisibleInAct(t *testing.T) {
	ensureBinary(t, "act")
	ensureBinary(t, "git")
	ensureBinary(t, "go")

	fixture := newFixtureRepo(t, "manual-and-schedule")
	cli := buildTenderCLI(t)

	runInteractiveAdd(
		t,
		fixture,
		cli,
		"weekly-audit",
	)

	commitFixture(t, fixture)

	manualOut := runCmd(t, fixture, "act", "-l", "workflow_dispatch", "-W", ".github/workflows")
	if !strings.Contains(manualOut, "tender/weekly-audit") {
		t.Fatalf("act workflow_dispatch listing missing workflow:\n%s", manualOut)
	}

	scheduleOut := runCmd(t, fixture, "act", "-l", "schedule", "-W", ".github/workflows")
	if !strings.Contains(scheduleOut, "tender/weekly-audit") {
		t.Fatalf("act schedule listing missing workflow:\n%s", scheduleOut)
	}

	workflowPath := filepath.Join(fixture, WorkflowDir, "weekly-audit.yml")
	workflowContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", workflowPath, err)
	}
	text := string(workflowContent)

	requiredSnippets := []string{
		"name: \"tender/weekly-audit\"",
		"TENDER_AGENT: \"Build\"",
		"TENDER_PROMPT: \"\"",
		"- cron: \"0 9 * * 1\"",
		"opencode run --agent \"$TENDER_AGENT\"",
		"git push origin HEAD:main",
		"group: tender-main",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("workflow missing snippet %q\n--- workflow ---\n%s", snippet, text)
		}
	}
}

func TestAcceptanceScheduleOnlyExcludedFromManualEventInAct(t *testing.T) {
	ensureBinary(t, "act")
	ensureBinary(t, "git")

	fixture := newFixtureRepo(t, "schedule-only")

	_, err := SaveNewTender(fixture, Tender{
		Name:   "nightly-refactor",
		Agent:  "refactor_bot",
		Prompt: "",
		Cron:   "0 2 * * 1,2,3,4,5",
		Manual: false,
	})
	if err != nil {
		t.Fatalf("SaveNewTender: %v", err)
	}

	commitFixture(t, fixture)

	manualOut := runCmd(t, fixture, "act", "-l", "workflow_dispatch", "-W", ".github/workflows")
	if strings.Contains(manualOut, "tender/nightly-refactor") {
		t.Fatalf("schedule-only tender should not appear for workflow_dispatch:\n%s", manualOut)
	}

	scheduleOut := runCmd(t, fixture, "act", "-l", "schedule", "-W", ".github/workflows")
	if !strings.Contains(scheduleOut, "tender/nightly-refactor") {
		t.Fatalf("schedule-only tender missing for schedule event:\n%s", scheduleOut)
	}
}

func TestAcceptanceManualOnlyExcludedFromScheduleEventInAct(t *testing.T) {
	ensureBinary(t, "act")
	ensureBinary(t, "git")

	fixture := newFixtureRepo(t, "manual-only")

	_, err := SaveNewTender(fixture, Tender{
		Name:   "adhoc-fixer",
		Agent:  "fixer",
		Prompt: "Fix obvious issues",
		Cron:   "",
		Manual: true,
	})
	if err != nil {
		t.Fatalf("SaveNewTender: %v", err)
	}

	commitFixture(t, fixture)

	manualOut := runCmd(t, fixture, "act", "-l", "workflow_dispatch", "-W", ".github/workflows")
	if !strings.Contains(manualOut, "tender/adhoc-fixer") {
		t.Fatalf("manual-only tender missing for workflow_dispatch:\n%s", manualOut)
	}

	scheduleOut := runCmd(t, fixture, "act", "-l", "schedule", "-W", ".github/workflows")
	if strings.Contains(scheduleOut, "tender/adhoc-fixer") {
		t.Fatalf("manual-only tender should not appear for schedule event:\n%s", scheduleOut)
	}
}

func TestAcceptanceTTY_NumberThenEnter_DoesNotSkipName(t *testing.T) {
	ensureBinary(t, "git")
	ensureBinary(t, "go")
	ensureBinary(t, "expect")

	fixture := newFixtureRepo(t, "tty-enter-regression")
	cli := buildTenderCLI(t)

	// Reproduces real terminal usage where users press Enter after each menu number.
	// This must still keep the interaction aligned and preserve the typed name.
	script := strings.Join([]string{
		"set timeout 20",
		"spawn " + cli,
		"expect \"Choose 1/2/3/4:\"",
		"send \"1\\r\"",
		"expect \"Name:\"",
		"send \"My Tender\\r\"",
		"expect \"Agent\"",
		"expect -re {Choose .*:}",
		"send \"1\\r\"",
		"expect \"Allow on-demand run from Actions UI?\"",
		"expect -re {Choose .*:}",
		"send \"1\\r\"",
		"expect \"Enable recurring schedule?\"",
		"expect -re {Choose .*:}",
		"send \"2\\r\"",
		"expect \"OK:\"",
		"expect \"Choose 1/2/3/4:\"",
		"send \"4\\r\"",
		"expect eof",
	}, "\n")

	_ = runCmdWithStdin(t, fixture, script, "expect", "-f", "-")

	workflowPath := filepath.Join(fixture, WorkflowDir, "my-tender.yml")
	workflowContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", workflowPath, err)
	}
	text := string(workflowContent)
	if !strings.Contains(text, `name: "tender/My Tender"`) {
		t.Fatalf("workflow name was not preserved after tty menu selection:\n%s", text)
	}
}

func newFixtureRepo(t *testing.T, name string) string {
	t.Helper()
	root := projectRoot(t)
	fixture := filepath.Join(root, ".tender", "test-work", name)
	_ = os.RemoveAll(fixture)
	if err := os.MkdirAll(fixture, 0o755); err != nil {
		t.Fatalf("MkdirAll fixture: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(fixture)
	})
	return fixture
}

func commitFixture(t *testing.T, dir string) {
	t.Helper()
	runCmd(t, dir, "git", "init", "-q")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "-c", "user.name=tender-test", "-c", "user.email=tender-test@example.com", "commit", "-qm", "fixture")
}

func ensureBinary(t *testing.T, bin string) {
	t.Helper()
	if _, err := exec.LookPath(bin); err != nil {
		t.Fatalf("required binary %q not found in PATH", bin)
	}
}

func runCmd(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s %s\nerror: %v\noutput:\n%s", name, strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func runCmdWithStdin(t *testing.T, dir string, stdin string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s %s\nerror: %v\noutput:\n%s", name, strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("project root detection failed: %v", err)
	}
	return root
}

func buildTenderCLI(t *testing.T) string {
	t.Helper()
	root := projectRoot(t)
	binDir := filepath.Join(root, ".tender", "test-work", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll bin dir: %v", err)
	}
	binPath := filepath.Join(binDir, "tender")
	runCmd(t, root, "go", "build", "-o", binPath, "./cmd/tender")
	return binPath
}

func runInteractiveAdd(t *testing.T, fixture string, cli string, name string) {
	t.Helper()
	// Scripted input for interactive flow:
	// action(add) -> name -> agent(default) -> on-demand(default yes) -> enable schedule -> weekly ->
	// monday -> 09:00 -> continue -> quit.
	input := strings.Join([]string{
		"1",
		name,
		"",
		"",
		"1",
		"3",
		"4",
		"3",
		"4",
	}, "\n")
	_ = runCmdWithStdin(t, fixture, input, cli)
}
