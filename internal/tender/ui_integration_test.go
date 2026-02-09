package tender

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
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

	t.Run("allows q as name and keeps dashboard frame on create screen", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		binDir := t.TempDir()
		writeFakeOpenCode(t, binDir, `#!/bin/sh
cat <<'EOF'
NAME MODE
TendTests primary
EOF
`)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		// create tender named "q" to verify q is treated as data in name entry.
		stdin := strings.NewReader(strings.Join([]string{
			"1", // create
			"q", // name
			"",  // agent (default TendTests)
			"",  // push (default no)
			"",  // timeout (default 30)
			"2", // recurring schedule: no
			"q", // exit dashboard
		}, "\n") + "\n")
		var stdout bytes.Buffer

		err := RunInteractive(root, stdin, &stdout)
		if err != nil {
			t.Fatalf("RunInteractive returned error: %v", err)
		}

		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders returned error: %v", err)
		}
		if len(tenders) != 1 {
			t.Fatalf("expected one tender after create flow, got %d", len(tenders))
		}
		if tenders[0].Name != "q" {
			t.Fatalf("expected tender name 'q', got %q", tenders[0].Name)
		}

		clean := ansiRE.ReplaceAllString(stdout.String(), "")
		if strings.Contains(clean, "Back to dashboard") {
			t.Fatalf("did not expect q-back hint during name entry:\n%s", clean)
		}
		if strings.Count(clean, "STATE") < 2 {
			t.Fatalf("expected dashboard frame to persist between home and create screens:\n%s", clean)
		}
		if strings.Count(clean, "Select Tender") < 2 {
			t.Fatalf("expected return to dashboard after create flow:\n%s", clean)
		}
	})

	t.Run("returns to dashboard from continue screen", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		// open create flow, submit empty name (shows continue screen), then return and exit.
		stdin := strings.NewReader("1\n\nq\nq\n")
		var stdout bytes.Buffer

		err := RunInteractive(root, stdin, &stdout)
		if err != nil {
			t.Fatalf("RunInteractive returned error: %v", err)
		}

		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders returned error: %v", err)
		}
		if len(tenders) != 0 {
			t.Fatalf("expected no tenders after leaving continue screen, got %d", len(tenders))
		}
		clean := ansiRE.ReplaceAllString(stdout.String(), "")
		if strings.Count(clean, "Select Tender") < 2 {
			t.Fatalf("expected to return to dashboard after continue screen:\n%s", clean)
		}
	})

	t.Run("create flow shows default yes for recurring schedule", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		binDir := t.TempDir()
		writeFakeOpenCode(t, binDir, `#!/bin/sh
cat <<'EOF'
NAME MODE
TendTests primary
EOF
`)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		stdin := strings.NewReader(strings.Join([]string{
			"1",          // create
			"new-tender", // name
			"",           // agent (default: TendTests)
			"",           // push (default: no)
			"",           // timeout (default: 30)
			"",           // recurring schedule (default: yes)
			"",           // schedule mode (default: daily)
			"",           // daily time (default: 09:00 UTC)
			"q",          // exit
		}, "\n") + "\n")
		var stdout bytes.Buffer

		if err := RunInteractive(root, stdin, &stdout); err != nil {
			t.Fatalf("RunInteractive returned error: %v", err)
		}

		clean := ansiRE.ReplaceAllString(stdout.String(), "")
		if !strings.Contains(clean, "Timeout in minutes (default: 30):") {
			t.Fatalf("expected timeout prompt with create default:\n%s", clean)
		}
		assertQuestionDefaultChoice(t, clean, "Enable recurring schedule?", 1)
		if !strings.Contains(clean, "Yes (default)") {
			t.Fatalf("expected schedule prompt to mark Yes as default:\n%s", clean)
		}
	})

	t.Run("create flow does not preview agent before selection", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		binDir := t.TempDir()
		writeFakeOpenCode(t, binDir, `#!/bin/sh
cat <<'EOF'
NAME MODE
TendTests primary
EOF
`)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		stdin := strings.NewReader(strings.Join([]string{
			"1",          // create
			"new-tender", // name
			"",           // agent (default TendTests)
			"2",          // push: no
			"",           // timeout (default: 30)
			"2",          // recurring schedule: no
			"q",          // exit
		}, "\n") + "\n")
		var stdout bytes.Buffer

		if err := RunInteractive(root, stdin, &stdout); err != nil {
			t.Fatalf("RunInteractive returned error: %v", err)
		}

		clean := ansiRE.ReplaceAllString(stdout.String(), "")
		if !strings.Contains(clean, "Current: name=new-tender") {
			t.Fatalf("expected create flow to show selected name before agent selection:\n%s", clean)
		}

		agentHeading := regexp.MustCompile(`(?m)^\s*Agent\s*$`)
		loc := agentHeading.FindStringIndex(clean)
		if loc == nil {
			t.Fatalf("missing agent heading in create flow output:\n%s", clean)
		}
		beforeAgent := clean[:loc[0]]
		if strings.Contains(beforeAgent, "Current: name=new-tender | agent=") {
			t.Fatalf("agent value was shown before selection:\n%s", clean)
		}
	})

	t.Run("edit flow without schedule shows default no for recurring schedule", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}
		if _, err := SaveNewTender(root, Tender{
			Name:           "existing",
			Agent:          "Build",
			Manual:         true,
			TimeoutMinutes: 45,
		}); err != nil {
			t.Fatalf("failed to seed test tender: %v", err)
		}

		binDir := t.TempDir()
		writeFakeOpenCode(t, binDir, `#!/bin/sh
cat <<'EOF'
NAME MODE
TendTests primary
EOF
`)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		stdin := strings.NewReader(strings.Join([]string{
			"2", // open first tender
			"2", // edit
			"",  // name (default existing)
			"",  // agent (default TendTests)
			"",  // push (default no)
			"",  // timeout (default existing)
			"",  // recurring schedule (default no)
			"1", // back
			"q", // exit
		}, "\n") + "\n")
		var stdout bytes.Buffer

		if err := RunInteractive(root, stdin, &stdout); err != nil {
			t.Fatalf("RunInteractive returned error: %v", err)
		}

		clean := ansiRE.ReplaceAllString(stdout.String(), "")
		if !strings.Contains(clean, "Timeout in minutes (default: 45):") {
			t.Fatalf("expected timeout prompt with edit default:\n%s", clean)
		}
		assertQuestionDefaultChoice(t, clean, "Enable recurring schedule?", 2)
		if !strings.Contains(clean, "No (default)") {
			t.Fatalf("expected schedule prompt to mark No as default:\n%s", clean)
		}
	})

	t.Run("agent picker supports 9/0 paging for long lists", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		binDir := t.TempDir()
		writeFakeOpenCode(t, binDir, `#!/bin/sh
cat <<'EOF'
NAME MODE
Agent01 primary
Agent02 primary
Agent03 primary
Agent04 primary
Agent05 primary
Agent06 primary
Agent07 primary
Agent08 primary
Agent09 primary
Agent10 primary
EOF
`)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		stdin := strings.NewReader(strings.Join([]string{
			"1",            // create
			"paged-agents", // name
			"0",            // agent page down
			"2",            // choose Agent10 (2nd slot on page 2)
			"",             // push (default no)
			"",             // timeout (default 30)
			"2",            // recurring schedule: no
			"q",            // exit
		}, "\n") + "\n")
		var stdout bytes.Buffer

		if err := RunInteractive(root, stdin, &stdout); err != nil {
			t.Fatalf("RunInteractive returned error: %v", err)
		}

		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders returned error: %v", err)
		}
		if len(tenders) != 1 {
			t.Fatalf("expected one tender after create flow, got %d", len(tenders))
		}
		if tenders[0].Agent != "Agent10" {
			t.Fatalf("expected selected agent Agent10, got %q", tenders[0].Agent)
		}

		clean := ansiRE.ReplaceAllString(stdout.String(), "")
		if !strings.Contains(clean, "Choose 1-8, 9(up), 0(down) (Enter for default):") {
			t.Fatalf("expected single-key paged prompt in agent picker:\n%s", clean)
		}
		if !strings.Contains(clean, "Showing 1-8 of 10 (page 1/2)") || !strings.Contains(clean, "Showing 9-10 of 10 (page 2/2)") {
			t.Fatalf("expected paging summaries in agent picker:\n%s", clean)
		}
		if strings.Contains(clean, "Choose 1-10") {
			t.Fatalf("did not expect multi-digit selection prompt:\n%s", clean)
		}
	})

	t.Run("complete journey handles blank name then exercises create back edit delete and exit", func(t *testing.T) {
		root := t.TempDir()
		if err := EnsureWorkflowDir(root); err != nil {
			t.Fatalf("failed to create workflow dir: %v", err)
		}

		binDir := t.TempDir()
		writeFakeOpenCode(t, binDir, `#!/bin/sh
cat <<'EOF'
NAME MODE
TendTests primary
RefineDocs primary
EOF
`)
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		stdin := strings.NewReader(strings.Join([]string{
			"1",               // create
			"",                // name (blank; should fail validation)
			"1",               // continue/back to dashboard
			"1",               // create
			"journey",         // name
			"",                // agent (default TendTests)
			"1",               // push: yes
			"45",              // timeout
			"1",               // recurring schedule: yes
			"2",               // schedule mode: daily
			"4",               // daily time: 12:00 UTC
			"2",               // open tender
			"1",               // back
			"2",               // open tender again
			"2",               // edit
			"journey-renamed", // name
			"2",               // agent: RefineDocs
			"2",               // push: no
			"60",              // timeout
			"2",               // recurring schedule: no
			"3",               // delete
			"1",               // confirm delete: yes
			"q",               // exit
		}, "\n") + "\n")
		var stdout bytes.Buffer

		if err := RunInteractive(root, stdin, &stdout); err != nil {
			t.Fatalf("RunInteractive() error = %v", err)
		}

		tenders, err := LoadTenders(root)
		if err != nil {
			t.Fatalf("LoadTenders(%q) error = %v", root, err)
		}
		if len(tenders) != 0 {
			t.Fatalf("len(LoadTenders(%q)) = %d, want 0", root, len(tenders))
		}

		clean := ansiRE.ReplaceAllString(stdout.String(), "")
		requiredSnippets := []string{
			"ERROR: Name is required.",
			"OK: Saved journey.yml",
			"OK: Updated journey.yml",
			"OK: Deleted journey.yml",
			"Tender journey",
			"Tender journey-renamed",
			"daily at 12:00 UTC + on-push(main) + on-demand",
			"RefineDocs",
		}
		for _, snippet := range requiredSnippets {
			if !strings.Contains(clean, snippet) {
				t.Fatalf("RunInteractive() output missing %q\noutput:\n%s", snippet, clean)
			}
		}
	})
}

func assertQuestionDefaultChoice(t *testing.T, output, question string, defaultChoice int) {
	t.Helper()
	start := strings.Index(output, question)
	if start < 0 {
		t.Fatalf("missing question %q in output:\n%s", question, output)
	}
	segment := output[start:]
	expected := "Choose 1-2 (default: " + strconv.Itoa(defaultChoice) + "):"
	if !strings.Contains(segment, expected) {
		t.Fatalf("missing %q after %q:\n%s", expected, question, segment)
	}
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

		clean := ansiRE.ReplaceAllString(output, "")
		section := slotSection(clean)
		if len(section) != rootTenderSlots() {
			t.Fatalf("expected %d slot rows, got %d", rootTenderSlots(), len(section))
		}
		for i, line := range section {
			if strings.TrimSpace(line) != "" {
				t.Fatalf("expected blank slot row %d, got %q", i, line)
			}
		}
		if strings.Contains(clean, "(empty)") {
			t.Fatalf("expected no empty placeholder text:\n%s", clean)
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

		clean := ansiRE.ReplaceAllString(output, "")
		section := slotSection(clean)
		if len(section) != rootTenderSlots() {
			t.Fatalf("expected %d slot rows, got %d", rootTenderSlots(), len(section))
		}
		visible := 0
		blank := 0
		for _, line := range section {
			if strings.TrimSpace(line) == "" {
				blank++
				continue
			}
			visible++
		}
		if visible != 2 {
			t.Fatalf("expected 2 visible slot rows, got %d", visible)
		}
		if blank != rootTenderSlots()-2 {
			t.Fatalf("expected %d blank slot rows, got %d", rootTenderSlots()-2, blank)
		}
	})
}

func slotSection(cleanOutput string) []string {
	lines := strings.Split(cleanOutput, "\n")
	createIdx := -1
	scrollIdx := -1
	for i, line := range lines {
		if createIdx < 0 && strings.Contains(line, "Create tender") {
			createIdx = i
			continue
		}
		if createIdx >= 0 && strings.Contains(line, "Scroll up") {
			scrollIdx = i
			break
		}
	}
	if createIdx < 0 || scrollIdx < 0 || scrollIdx <= createIdx+1 {
		return nil
	}
	return lines[createIdx+1 : scrollIdx]
}
