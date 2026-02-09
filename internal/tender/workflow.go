package tender

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func EnsureWorkflowDir(root string) error {
	return os.MkdirAll(filepath.Join(root, WorkflowDir), 0o755)
}

func LoadTenders(root string) ([]Tender, error) {
	dir := filepath.Join(root, WorkflowDir)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return []Tender{}, nil
		}
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]Tender, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !(strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) {
			continue
		}
		abs := filepath.Join(dir, name)
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}
		t, ok := parseTenderWorkflow(string(data))
		if !ok {
			continue
		}
		t.WorkflowFile = name
		out = append(out, t)
	}

	SortTenders(out)
	return out, nil
}

func SaveTender(root string, t Tender) error {
	if err := ValidateTender(t); err != nil {
		return err
	}
	if err := EnsureWorkflowDir(root); err != nil {
		return err
	}

	file := t.WorkflowFile
	if strings.TrimSpace(file) == "" {
		file = Slugify(t.Name) + ".yml"
	}
	if !strings.HasSuffix(file, ".yml") && !strings.HasSuffix(file, ".yaml") {
		file += ".yml"
	}
	file = filepath.Base(file)
	path := filepath.Join(root, WorkflowDir, file)
	return os.WriteFile(path, []byte(RenderWorkflow(t)), 0o644)
}

func RemoveTender(root, name string) error {
	tenders, err := LoadTenders(root)
	if err != nil {
		return err
	}
	idx := findTenderIndex(tenders, name)
	if idx < 0 {
		return fmt.Errorf("tender %q not found", name)
	}
	path := filepath.Join(root, WorkflowDir, tenders[idx].WorkflowFile)
	return os.Remove(path)
}

func RenderWorkflow(t Tender) string {
	var b strings.Builder
	b.WriteString("name: ")
	b.WriteString(strconv.Quote("tender/" + strings.TrimSpace(t.Name)))
	b.WriteString("\n\n")
	b.WriteString("on:\n")
	if t.Manual {
		b.WriteString("  workflow_dispatch:\n")
		b.WriteString("    inputs:\n")
		b.WriteString("      prompt:\n")
		b.WriteString("        description: \"Optional prompt override\"\n")
		b.WriteString("        required: false\n")
		b.WriteString("        default: \"\"\n")
		b.WriteString("        type: string\n")
	}
	if t.Push {
		b.WriteString("  push:\n")
		b.WriteString("    branches:\n")
		b.WriteString("      - main\n")
	}
	if strings.TrimSpace(t.Cron) != "" {
		b.WriteString("  schedule:\n")
		b.WriteString("    - cron: ")
		b.WriteString(strconv.Quote(strings.TrimSpace(t.Cron)))
		b.WriteString("\n")
	}
	if !t.Manual && !t.Push && strings.TrimSpace(t.Cron) == "" {
		b.WriteString("  workflow_dispatch:\n")
	}

	b.WriteString("\npermissions:\n")
	b.WriteString("  contents: write\n\n")
	b.WriteString("concurrency:\n")
	b.WriteString("  group: tender-main\n")
	b.WriteString("  cancel-in-progress: false\n\n")
	b.WriteString("jobs:\n")
	b.WriteString("  tender:\n")
	if t.Push {
		// Prevent circular runs when this workflow pushes back to main.
		b.WriteString("    if: ${{ github.event_name != 'push' || github.actor != 'github-actions[bot]' }}\n")
	}
	b.WriteString("    runs-on: ubuntu-latest\n")
	b.WriteString("    timeout-minutes: ")
	b.WriteString(strconv.Itoa(normalizeTimeoutMinutes(t.TimeoutMinutes)))
	b.WriteString("\n")
	b.WriteString("    env:\n")
	b.WriteString("      TENDER_NAME: ")
	b.WriteString(strconv.Quote(strings.TrimSpace(t.Name)))
	b.WriteString("\n")
	b.WriteString("      TENDER_AGENT: ")
	b.WriteString(strconv.Quote(strings.TrimSpace(t.Agent)))
	b.WriteString("\n")
	b.WriteString("      TENDER_PROMPT: ")
	b.WriteString(strconv.Quote(strings.TrimSpace(t.Prompt)))
	b.WriteString("\n")
	b.WriteString("    steps:\n")
	b.WriteString("      - uses: actions/checkout@v4\n")
	b.WriteString("        with:\n")
	b.WriteString("          fetch-depth: 0\n\n")
	b.WriteString("      - name: Install OpenCode\n")
	b.WriteString("        shell: bash\n")
	b.WriteString("        run: |\n")
	b.WriteString("          set -euo pipefail\n")
	b.WriteString("          curl -fsSL https://opencode.ai/install | bash\n")
	b.WriteString("          echo \"$HOME/bin\" >> \"$GITHUB_PATH\"\n")
	b.WriteString("          echo \"$HOME/.local/bin\" >> \"$GITHUB_PATH\"\n")
	b.WriteString("          echo \"$HOME/.opencode/bin\" >> \"$GITHUB_PATH\"\n\n")
	b.WriteString("      - name: Prepare main\n")
	b.WriteString("        shell: bash\n")
	b.WriteString("        run: |\n")
	b.WriteString("          set -euo pipefail\n")
	b.WriteString("          git config user.name \"tender[bot]\"\n")
	b.WriteString("          git config user.email \"tender[bot]@users.noreply.github.com\"\n")
	b.WriteString("          git fetch origin main\n")
	b.WriteString("          git checkout -B main origin/main\n\n")
	b.WriteString("      - name: Run OpenCode\n")
	b.WriteString("        shell: bash\n")
	b.WriteString("        env:\n")
	b.WriteString("          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}\n")
	b.WriteString("          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}\n")
	b.WriteString("        run: |\n")
	b.WriteString("          set -euo pipefail\n")
	b.WriteString("          cd \"$GITHUB_WORKSPACE\"\n")
	b.WriteString("          if [ -f \"$GITHUB_WORKSPACE/opencode.json\" ]; then export OPENCODE_CONFIG=\"$GITHUB_WORKSPACE/opencode.json\"; fi\n")
	b.WriteString("          if [ -d \"$GITHUB_WORKSPACE/.opencode\" ]; then export OPENCODE_CONFIG_DIR=\"$GITHUB_WORKSPACE/.opencode\"; fi\n")
	b.WriteString("          DISPATCH_PROMPT=\"${{ github.event_name == 'workflow_dispatch' && inputs.prompt || '' }}\"\n")
	b.WriteString("          RUN_PROMPT=\"${DISPATCH_PROMPT:-}\"\n")
	b.WriteString("          if [ -z \"${RUN_PROMPT}\" ]; then\n")
	b.WriteString("            RUN_PROMPT=\"${TENDER_PROMPT:-}\"\n")
	b.WriteString("          fi\n")
	b.WriteString("          if [ -z \"${RUN_PROMPT}\" ]; then\n")
	b.WriteString("            RUN_PROMPT=\"Run the tender task '$TENDER_NAME' for this repository.\"\n")
	b.WriteString("          fi\n")
	b.WriteString("          opencode run --agent \"$TENDER_AGENT\" \"$RUN_PROMPT\"\n\n")
	b.WriteString("      - name: Commit and push main\n")
	b.WriteString("        shell: bash\n")
	b.WriteString("        run: |\n")
	b.WriteString("          set -euo pipefail\n")
	b.WriteString("          CURRENT_BRANCH=\"$(git rev-parse --abbrev-ref HEAD || echo detached)\"\n")
	b.WriteString("          AHEAD_COUNT=\"$(git rev-list --count origin/main..HEAD || echo 0)\"\n")
	b.WriteString("          if git diff --quiet --ignore-submodules -- && git diff --cached --quiet --ignore-submodules --; then\n")
	b.WriteString("            if [ \"$CURRENT_BRANCH\" != \"main\" ] || [ \"$AHEAD_COUNT\" -gt 0 ]; then\n")
	b.WriteString("              echo \"No working tree changes; pushing existing commits from $CURRENT_BRANCH to main\"\n")
	b.WriteString("              git pull --rebase origin main\n")
	b.WriteString("              git push origin HEAD:main\n")
	b.WriteString("              exit 0\n")
	b.WriteString("            fi\n")
	b.WriteString("            echo \"No changes to commit\"\n")
	b.WriteString("            exit 0\n")
	b.WriteString("          fi\n")
	b.WriteString("          git add -A\n")
	b.WriteString("          git commit -m \"tender($TENDER_NAME): autonomous update\"\n")
	b.WriteString("          git pull --rebase origin main\n")
	b.WriteString("          git push origin HEAD:main\n")
	return b.String()
}

func parseTenderWorkflow(content string) (Tender, bool) {
	var t Tender
	lines := strings.Split(content, "\n")
	hasRun := false
	hasAgent := false
	hasTopName := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "name:") && !strings.HasPrefix(trim, "- name:"):
			raw := strings.TrimSpace(strings.TrimPrefix(trim, "name:"))
			name := parseQuotedValue(raw)
			if strings.HasPrefix(name, "tender/") {
				t.Name = strings.TrimPrefix(name, "tender/")
				hasTopName = true
			}
		case trim == "workflow_dispatch:":
			t.Manual = true
		case trim == "push:":
			t.Push = true
		case strings.HasPrefix(trim, "- cron:"):
			t.Cron = parseQuotedValue(strings.TrimSpace(strings.TrimPrefix(trim, "- cron:")))
		case strings.HasPrefix(trim, "TENDER_AGENT:"):
			t.Agent = parseQuotedValue(strings.TrimSpace(strings.TrimPrefix(trim, "TENDER_AGENT:")))
			hasAgent = strings.TrimSpace(t.Agent) != ""
		case strings.HasPrefix(trim, "TENDER_PROMPT:"):
			t.Prompt = parseQuotedValue(strings.TrimSpace(strings.TrimPrefix(trim, "TENDER_PROMPT:")))
		case strings.HasPrefix(trim, "timeout-minutes:"):
			timeoutRaw := strings.TrimSpace(strings.TrimPrefix(trim, "timeout-minutes:"))
			timeout, err := strconv.Atoi(timeoutRaw)
			if err == nil && timeout > 0 {
				t.TimeoutMinutes = timeout
			}
		case strings.Contains(trim, "opencode run"):
			hasRun = true
		}
	}
	if !hasTopName || !hasAgent || !hasRun {
		return Tender{}, false
	}
	if strings.TrimSpace(t.Name) == "" {
		t.Name = strings.TrimSpace(t.Agent)
	}
	t.TimeoutMinutes = normalizeTimeoutMinutes(t.TimeoutMinutes)
	return t, true
}

func parseQuotedValue(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "\"") {
		if u, err := strconv.Unquote(raw); err == nil {
			return u
		}
	}
	return strings.Trim(raw, "\"'")
}

func ValidateTender(t Tender) error {
	name := strings.TrimSpace(t.Name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(t.Agent) == "" {
		return fmt.Errorf("agent is required")
	}
	if strings.ContainsAny(name, "\r\n") {
		return fmt.Errorf("name cannot contain newlines")
	}
	if strings.Contains(name, "/") {
		return fmt.Errorf("name cannot contain '/'")
	}
	if strings.TrimSpace(t.Cron) != "" {
		if len(strings.Fields(t.Cron)) != 5 {
			return fmt.Errorf("cron must have 5 fields")
		}
	}
	if t.TimeoutMinutes < 0 {
		return fmt.Errorf("timeout-minutes must be greater than 0")
	}
	if !t.Manual && !t.Push && strings.TrimSpace(t.Cron) == "" {
		return fmt.Errorf("enable manual or set a schedule")
	}
	return nil
}

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var out []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out = append(out, r)
			continue
		}
		if r == '-' || r == '_' || r == ' ' {
			if len(out) == 0 || out[len(out)-1] == '-' {
				continue
			}
			out = append(out, '-')
		}
	}
	res := strings.Trim(string(out), "-")
	if res == "" {
		return "tender"
	}
	return res
}

func findTenderIndex(tenders []Tender, name string) int {
	needle := strings.TrimSpace(strings.ToLower(name))
	for i, t := range tenders {
		if strings.ToLower(t.Name) == needle {
			return i
		}
	}
	return -1
}

func PrintList(root string, stdout io.Writer) error {
	tenders, err := LoadTenders(root)
	if err != nil {
		return err
	}
	if len(tenders) == 0 {
		_, _ = fmt.Fprintln(stdout, "No managed tender workflows found.")
		return nil
	}
	_, _ = fmt.Fprintln(stdout, "NAME\tAGENT\tTRIGGER\tWORKFLOW")
	for _, t := range tenders {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", t.Name, t.Agent, TriggerSummary(t.Cron, t.Manual, t.Push), t.WorkflowFile)
	}
	return nil
}

func TriggerSummary(cron string, manual bool, push bool) string {
	schedule := ""
	if strings.TrimSpace(cron) != "" {
		if d, ok := scheduleDefaultsFromCron(cron); ok {
			switch d.Mode {
			case "hourly":
				schedule = fmt.Sprintf("every hour at :%02d UTC", d.Minute)
			case "daily":
				schedule = fmt.Sprintf("daily at %02d:%02d UTC", d.Hour, d.Minute)
			case "weekly":
				dayNames := make([]string, 0, len(d.Days))
				for _, day := range d.Days {
					dayNames = append(dayNames, weekdayName(day))
				}
				schedule = fmt.Sprintf("weekly %s at %02d:%02d UTC", strings.Join(dayNames, ","), d.Hour, d.Minute)
			}
		} else {
			schedule = "scheduled"
		}
	}

	parts := make([]string, 0, 3)
	if schedule != "" {
		parts = append(parts, schedule)
	}
	if push {
		parts = append(parts, "on-push(main)")
	}
	if manual {
		parts = append(parts, "on-demand")
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " + ")
}

func weekdayName(day int) string {
	switch day {
	case 0:
		return "Sun"
	case 1:
		return "Mon"
	case 2:
		return "Tue"
	case 3:
		return "Wed"
	case 4:
		return "Thu"
	case 5:
		return "Fri"
	case 6:
		return "Sat"
	default:
		return strconv.Itoa(day)
	}
}

func findUnusedWorkflowName(root, base string) (string, error) {
	dir := filepath.Join(root, WorkflowDir)
	if err := EnsureWorkflowDir(root); err != nil {
		return "", err
	}
	base = Slugify(base)
	candidate := base + ".yml"
	if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
		return candidate, nil
	}
	for i := 2; i < 1000; i++ {
		candidate = fmt.Sprintf("%s-%d.yml", base, i)
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("unable to find available workflow filename for %q", base)
}

func SaveNewTender(root string, t Tender) (Tender, error) {
	current, err := LoadTenders(root)
	if err != nil {
		return Tender{}, err
	}
	if findTenderIndex(current, t.Name) >= 0 {
		return Tender{}, fmt.Errorf("tender %q already exists", t.Name)
	}
	wf, err := findUnusedWorkflowName(root, t.Name)
	if err != nil {
		return Tender{}, err
	}
	t.WorkflowFile = wf
	if err := SaveTender(root, t); err != nil {
		return Tender{}, err
	}
	return t, nil
}

func UpdateTender(root string, oldName string, updated Tender) error {
	tenders, err := LoadTenders(root)
	if err != nil {
		return err
	}
	idx := findTenderIndex(tenders, oldName)
	if idx < 0 {
		return fmt.Errorf("tender %q not found", oldName)
	}
	for i, t := range tenders {
		if i == idx {
			continue
		}
		if strings.EqualFold(t.Name, updated.Name) {
			return fmt.Errorf("tender %q already exists", updated.Name)
		}
	}
	updated.WorkflowFile = tenders[idx].WorkflowFile
	return SaveTender(root, updated)
}

func ManagedWorkflowPath(root, tenderName string) (string, error) {
	tenders, err := LoadTenders(root)
	if err != nil {
		return "", err
	}
	idx := findTenderIndex(tenders, tenderName)
	if idx < 0 {
		return "", fmt.Errorf("tender %q not found", tenderName)
	}
	return filepath.Join(root, WorkflowDir, tenders[idx].WorkflowFile), nil
}

func SortedCrons(tenders []Tender) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, t := range tenders {
		c := strings.TrimSpace(t.Cron)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func ScanWorkflowHasTender(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	has := false
	for s.Scan() {
		if strings.Contains(s.Text(), "TENDER_AGENT:") {
			has = true
			break
		}
	}
	if err := s.Err(); err != nil {
		return false, err
	}
	return has, nil
}
