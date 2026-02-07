package tender

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	cReset   = "\033[0m"
	cDim     = "\033[2m"
	cBold    = "\033[1m"
	cBlue    = "\033[34m"
	cCyan    = "\033[36m"
	cGreen   = "\033[32m"
	cYellow  = "\033[33m"
	cRed     = "\033[31m"
	cMagenta = "\033[35m"
)

func RunInteractive(root string, stdin io.Reader, stdout io.Writer) error {
	r := bufio.NewReader(stdin)
	tty := ttyFile(stdin)

	for {
		tenders, err := LoadTenders(root)
		if err != nil {
			return err
		}
		drawHome(stdout, tenders)
		drawActions(stdout)

		action, err := promptMenuChoice(r, stdout, tty, "Choose 1/2/3/4: ")
		if err != nil {
			return err
		}

		switch strings.TrimSpace(action) {
		case "1":
			base := Tender{Agent: "Build", Manual: true}
			t, ok, err := inputTender(r, stdout, root, base, true, tty)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			saved, err := SaveNewTender(root, t)
			if err != nil {
				printErr(stdout, err.Error())
				acknowledge(r, stdout, tty)
				continue
			}
			printOK(stdout, "Saved "+saved.WorkflowFile)

		case "2":
			selected, ok, err := selectTender(r, stdout, tty, tenders, "Edit")
			if err != nil {
				return err
			}
			if !ok {
				continue
			}

			updated, ok, err := inputTender(r, stdout, root, selected, false, tty)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}

			if err := UpdateTender(root, selected.Name, updated); err != nil {
				printErr(stdout, err.Error())
				acknowledge(r, stdout, tty)
				continue
			}
			printOK(stdout, "Updated "+selected.WorkflowFile)

		case "3":
			selected, ok, err := selectTender(r, stdout, tty, tenders, "Delete")
			if err != nil {
				return err
			}
			if !ok {
				continue
			}

			confirm, err := promptBinaryChoice(r, stdout, tty, fmt.Sprintf("Delete %q?", selected.Name), false, false)
			if err != nil {
				return err
			}
			if !confirm {
				printErr(stdout, "Delete cancelled")
				acknowledge(r, stdout, tty)
				continue
			}

			if err := RemoveTender(root, selected.Name); err != nil {
				return err
			}
			printOK(stdout, "Deleted "+selected.WorkflowFile)

		case "4":
			return nil

		default:
			printErr(stdout, "Unknown selection")
			acknowledge(r, stdout, tty)
		}
	}
}

func drawHome(w io.Writer, tenders []Tender) {
	clearScreen(w)
	rule(w, '=')
	fmt.Fprintf(w, "%s%sT E N D E R%s\n", cBold, cBlue, cReset)
	fmt.Fprintf(w, "%sVisual scheduler for autonomous OpenCode runs%s\n", cDim, cReset)
	rule(w, '=')
	fmt.Fprintf(w, "%sState%s  GitHub Actions workflows (.github/workflows)\n", colorLabel(cMagenta), cReset)
	fmt.Fprintf(w, "%sMode%s   Autonomous commits to main\n", colorLabel(cMagenta), cReset)
	fmt.Fprintf(w, "%sCount%s  %d managed tender(s)\n", colorLabel(cMagenta), cReset, len(tenders))
	fmt.Fprintln(w)

	if len(tenders) == 0 {
		fmt.Fprintf(w, "%sNo managed tender workflows found.%s\n\n", cDim, cReset)
		return
	}

	sort.Slice(tenders, func(i, j int) bool { return tenders[i].Name < tenders[j].Name })
	fmt.Fprintf(w, "%s%-20s %-18s %-31s %s%s\n", cBold, "Name", "Agent", "Trigger", "Workflow", cReset)
	rule(w, '-')
	for _, t := range tenders {
		fmt.Fprintf(w, "%-20s %-18s %-31s %s\n", t.Name, t.Agent, TriggerSummary(t.Cron, t.Manual), t.WorkflowFile)
	}
	fmt.Fprintln(w)
}

func drawActions(w io.Writer) {
	fmt.Fprintf(w, "%sActions%s\n", colorLabel(cCyan), cReset)
	fmt.Fprintf(w, "  %s1%s Add tender\n", cBold, cReset)
	fmt.Fprintf(w, "  %s2%s Edit tender\n", cBold, cReset)
	fmt.Fprintf(w, "  %s3%s Delete tender\n", cBold, cReset)
	fmt.Fprintf(w, "  %s4%s Quit\n", cBold, cReset)
}

func inputTender(r *bufio.Reader, w io.Writer, root string, base Tender, isNew bool, tty *os.File) (Tender, bool, error) {
	clearScreen(w)
	if isNew {
		fmt.Fprintf(w, "%sCreate Tender%s\n", colorLabel(cCyan), cReset)
	} else {
		fmt.Fprintf(w, "%sEdit Tender%s\n", colorLabel(cCyan), cReset)
	}
	rule(w, '-')

	namePrompt := "Name: "
	if base.Name != "" {
		namePrompt = fmt.Sprintf("Name (default: %s): ", base.Name)
	}
	nameInput, err := prompt(r, w, namePrompt)
	if err != nil {
		return Tender{}, false, err
	}
	nameInput = strings.TrimSpace(nameInput)
	if nameInput == "" {
		if strings.TrimSpace(base.Name) == "" {
			printErr(w, "Name is required.")
			acknowledge(r, w, tty)
			return base, false, nil
		}
		nameInput = base.Name
	}
	name := strings.TrimSpace(nameInput)

	agent, err := chooseAgent(r, w, root, base.Agent, tty)
	if err != nil {
		return Tender{}, false, err
	}

	manual, err := promptBinaryChoice(r, w, tty, "Allow on-demand run from Actions UI?", base.Manual, false)
	if err != nil {
		return Tender{}, false, err
	}

	hasScheduleDefault := strings.TrimSpace(base.Cron) != ""
	hasSchedule, err := promptBinaryChoice(r, w, tty, "Enable recurring schedule?", hasScheduleDefault, isNew)
	if err != nil {
		return Tender{}, false, err
	}

	cron := strings.TrimSpace(base.Cron)
	if hasSchedule {
		defaults, hasDefaults := scheduleDefaultsFromCron(cron)
		if cron != "" && !hasDefaults {
			fmt.Fprintf(w, "%sWarning:%s existing schedule is unsupported in presets; choose a new one.\n", cYellow, cReset)
		}

		defaultMode := 0
		if hasDefaults {
			switch defaults.Mode {
			case "hourly":
				defaultMode = 0
			case "daily":
				defaultMode = 1
			case "weekly":
				defaultMode = 2
			}
		}

		modeIndex, err := selectNumberedOption(r, w, tty, "Schedule", []string{"Hourly", "Daily", "Weekly"}, defaultMode, true)
		if err != nil {
			return Tender{}, false, err
		}

		switch modeIndex {
		case 0:
			minuteDefault := 0
			if hasDefaults {
				minuteDefault = nearestQuarterIndex(defaults.Minute)
			}
			minuteIndex, err := selectNumberedOption(r, w, tty, "Hourly minute", []string{":00", ":15", ":30", ":45"}, minuteDefault, true)
			if err != nil {
				return Tender{}, false, err
			}
			minutes := []int{0, 15, 30, 45}
			built, err := buildHourlyCron(strconv.Itoa(minutes[minuteIndex]))
			if err != nil {
				printErr(w, err.Error())
				acknowledge(r, w, tty)
				return base, false, nil
			}
			cron = built

		case 1:
			timeDefault := 2
			if hasDefaults {
				timeDefault = defaultTimePresetIndex(defaults.Hour, defaults.Minute)
			}
			timeIndex, err := selectNumberedOption(r, w, tty, "Daily time (UTC)", timePresetLabels(), timeDefault, true)
			if err != nil {
				return Tender{}, false, err
			}
			preset := dailyTimePresets[timeIndex]
			built, err := buildDailyCron(formatTime(preset.Hour, preset.Minute))
			if err != nil {
				printErr(w, err.Error())
				acknowledge(r, w, tty)
				return base, false, nil
			}
			cron = built

		case 2:
			dayDefault := 0
			if hasDefaults {
				dayDefault = defaultWeeklyDayPresetIndex(defaults.Days)
			}
			dayIndex, err := selectNumberedOption(r, w, tty, "Weekly days", weeklyDayPresetLabels(), dayDefault, true)
			if err != nil {
				return Tender{}, false, err
			}

			timeDefault := 2
			if hasDefaults {
				timeDefault = defaultTimePresetIndex(defaults.Hour, defaults.Minute)
			}
			timeIndex, err := selectNumberedOption(r, w, tty, "Weekly time (UTC)", timePresetLabels(), timeDefault, true)
			if err != nil {
				return Tender{}, false, err
			}

			days := weeklyDayPresets[dayIndex].Days
			preset := dailyTimePresets[timeIndex]
			built, err := buildWeeklyCron(joinInts(days, ","), formatTime(preset.Hour, preset.Minute))
			if err != nil {
				printErr(w, err.Error())
				acknowledge(r, w, tty)
				return base, false, nil
			}
			cron = built
		}

		fmt.Fprintf(w, "%sSchedule:%s %s\n", cDim, cReset, TriggerSummary(cron, false))
	} else {
		cron = ""
	}

	result := Tender{
		Name:         name,
		Agent:        strings.TrimSpace(agent),
		Prompt:       strings.TrimSpace(base.Prompt),
		Cron:         strings.TrimSpace(cron),
		Manual:       manual,
		WorkflowFile: base.WorkflowFile,
	}

	if err := ValidateTender(result); err != nil {
		printErr(w, err.Error())
		acknowledge(r, w, tty)
		return base, false, nil
	}
	return result, true, nil
}

func buildWeeklyCron(daysInput, timeInput string) (string, error) {
	days, err := parseDays(daysInput)
	if err != nil {
		return "", err
	}

	hour, minute, err := parseTimeHHMM(timeInput)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d %d * * %s", minute, hour, joinInts(days, ",")), nil
}

func buildDailyCron(timeInput string) (string, error) {
	hour, minute, err := parseTimeHHMM(timeInput)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d %d * * *", minute, hour), nil
}

func buildHourlyCron(minuteInput string) (string, error) {
	minute, err := strconv.Atoi(strings.TrimSpace(minuteInput))
	if err != nil || minute < 0 || minute > 59 {
		return "", fmt.Errorf("minute must be 0-59")
	}
	return fmt.Sprintf("%d * * * *", minute), nil
}

func parseTimeHHMM(input string) (hour int, minute int, err error) {
	hm := strings.Split(strings.TrimSpace(input), ":")
	if len(hm) != 2 {
		return 0, 0, fmt.Errorf("time must be HH:MM")
	}
	hour, err = strconv.Atoi(strings.TrimSpace(hm[0]))
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("hour must be 0-23")
	}
	minute, err = strconv.Atoi(strings.TrimSpace(hm[1]))
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("minute must be 0-59")
	}
	return hour, minute, nil
}

func formatTime(hour, minute int) string {
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

type scheduleDefaults struct {
	Mode   string
	Minute int
	Hour   int
	Days   []int
}

func scheduleDefaultsFromCron(cron string) (scheduleDefaults, bool) {
	fields := strings.Fields(strings.TrimSpace(cron))
	if len(fields) != 5 {
		return scheduleDefaults{}, false
	}

	minute, err := strconv.Atoi(fields[0])
	if err != nil || minute < 0 || minute > 59 {
		return scheduleDefaults{}, false
	}
	if fields[2] != "*" || fields[3] != "*" {
		return scheduleDefaults{}, false
	}

	if fields[1] == "*" && fields[4] == "*" {
		return scheduleDefaults{Mode: "hourly", Minute: minute}, true
	}

	hour, err := strconv.Atoi(fields[1])
	if err != nil || hour < 0 || hour > 23 {
		return scheduleDefaults{}, false
	}

	if fields[4] == "*" {
		return scheduleDefaults{Mode: "daily", Hour: hour, Minute: minute}, true
	}

	days, err := parseDays(fields[4])
	if err != nil || len(days) == 0 {
		return scheduleDefaults{}, false
	}
	return scheduleDefaults{Mode: "weekly", Hour: hour, Minute: minute, Days: days}, true
}

func parseDays(raw string) ([]int, error) {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	if len(parts) == 0 {
		return nil, fmt.Errorf("at least one day is required")
	}
	seen := map[int]bool{}
	out := make([]int, 0)
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || v < 0 || v > 6 {
			return nil, fmt.Errorf("days must be 0-6")
		}
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Ints(out)
	return out, nil
}

func joinInts(nums []int, sep string) string {
	parts := make([]string, 0, len(nums))
	for _, n := range nums {
		parts = append(parts, strconv.Itoa(n))
	}
	return strings.Join(parts, sep)
}

func prompt(r *bufio.Reader, w io.Writer, label string) (string, error) {
	fmt.Fprint(w, label)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func clearScreen(w io.Writer) {
	fmt.Fprint(w, "\033[H\033[2J")
}

func acknowledge(r *bufio.Reader, w io.Writer, tty *os.File) {
	_, _ = selectNumberedOption(r, w, tty, "Continue", []string{"Back"}, 0, false)
}

func printErr(w io.Writer, msg string) {
	fmt.Fprintf(w, "%s[ERROR]%s %s\n", cRed, cReset, msg)
}

func printOK(w io.Writer, msg string) {
	fmt.Fprintf(w, "%s[OK]%s %s\n", cGreen, cReset, msg)
}

func chooseAgent(r *bufio.Reader, w io.Writer, root string, current string, tty *os.File) (string, error) {
	agents, err := DiscoverPrimaryAgents(root)
	if err != nil {
		fmt.Fprintf(w, "%sWarning:%s unable to discover OpenCode agents (%v). Falling back to Build.\n", cYellow, cReset, err)
		agents = []string{"Build"}
	}
	if len(agents) == 0 {
		agents = []string{"Build"}
	}

	defaultIndex := 0
	if current != "" {
		if idx := indexOf(agents, current); idx >= 0 {
			defaultIndex = idx
		}
	}
	idx, err := selectNumberedOption(r, w, tty, "Agent", agents, defaultIndex, true)
	if err != nil {
		return "", err
	}
	return agents[idx], nil
}

func selectTender(r *bufio.Reader, w io.Writer, tty *os.File, tenders []Tender, action string) (Tender, bool, error) {
	if len(tenders) == 0 {
		printErr(w, "No tenders available")
		acknowledge(r, w, tty)
		return Tender{}, false, nil
	}

	sort.Slice(tenders, func(i, j int) bool { return tenders[i].Name < tenders[j].Name })
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s%s%s\n", colorLabel(cCyan), action+" Tender", cReset)
	rule(w, '.')
	for i, t := range tenders {
		fmt.Fprintf(w, "  %2d) %-20s %-30s %s\n", i+1, t.Name, TriggerSummary(t.Cron, t.Manual), t.WorkflowFile)
	}
	rule(w, '.')

	for {
		var (
			choice string
			err    error
		)
		if len(tenders) <= 9 {
			choice, err = promptMenuChoice(r, w, tty, fmt.Sprintf("Choose 1-%d (Enter to cancel): ", len(tenders)))
		} else {
			choice, err = prompt(r, w, fmt.Sprintf("Choose number 1-%d (blank to cancel): ", len(tenders)))
		}
		if err != nil {
			return Tender{}, false, err
		}
		choice = strings.TrimSpace(choice)
		if choice == "" {
			printErr(w, "Action cancelled")
			acknowledge(r, w, tty)
			return Tender{}, false, nil
		}

		n, err := strconv.Atoi(choice)
		if err == nil && n >= 1 && n <= len(tenders) {
			return tenders[n-1], true, nil
		}
		printErr(w, "Invalid selection.")
	}
}

func promptBinaryChoice(r *bufio.Reader, w io.Writer, tty *os.File, question string, defaultValue bool, requireExplicit bool) (bool, error) {
	defaultIndex := 1
	if defaultValue {
		defaultIndex = 0
	}
	if requireExplicit {
		defaultIndex = -1
	}
	idx, err := selectNumberedOption(r, w, tty, question, []string{"Yes", "No"}, defaultIndex, !requireExplicit)
	if err != nil {
		return false, err
	}
	return idx == 0, nil
}

func selectNumberedOption(r *bufio.Reader, w io.Writer, tty *os.File, title string, options []string, defaultIndex int, allowDefault bool) (int, error) {
	hasDefault := allowDefault
	if defaultIndex < 0 {
		hasDefault = false
		defaultIndex = 0
	}
	if defaultIndex >= len(options) {
		defaultIndex = 0
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s%s%s\n", colorLabel(cCyan), title, cReset)
	rule(w, '.')
	for i, option := range options {
		marker := " "
		if i == defaultIndex {
			marker = "*"
		}
		fmt.Fprintf(w, "  %d) %s %s\n", i+1, marker, option)
	}
	rule(w, '.')

	for {
		label := fmt.Sprintf("Choose 1-%d: ", len(options))
		if hasDefault {
			label = fmt.Sprintf("Choose 1-%d (default: %d): ", len(options), defaultIndex+1)
		}
		choice, err := promptMenuChoice(r, w, tty, label)
		if err != nil {
			return -1, err
		}
		choice = strings.TrimSpace(choice)
		if choice == "" {
			if hasDefault {
				return defaultIndex, nil
			}
			printErr(w, "Selection required.")
			continue
		}
		n, err := strconv.Atoi(choice)
		if err == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		printErr(w, "Invalid selection.")
	}
}

func promptMenuChoice(r *bufio.Reader, w io.Writer, tty *os.File, label string) (string, error) {
	if tty == nil || !supportsRawTTY(tty) {
		return prompt(r, w, label)
	}

	fmt.Fprint(w, label)
	if err := setTTYRaw(tty); err != nil {
		return prompt(r, w, label)
	}
	defer restoreTTY(tty)

	buf := make([]byte, 1)
	var ch byte
	idle := 0
	for {
		n, err := tty.Read(buf)
		if err != nil {
			if err == io.EOF {
				idle++
				if idle > 600 {
					return "", io.EOF
				}
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return "", err
		}
		if n == 0 {
			idle++
			if idle > 600 {
				return "", io.EOF
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		ch = buf[0]
		break
	}

	for {
		n, err := tty.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		if n == 0 {
			break
		}
	}

	if ch == '\r' || ch == '\n' {
		fmt.Fprintln(w)
		return "", nil
	}
	choice := strings.TrimSpace(string(ch))
	fmt.Fprintln(w, choice)
	return choice, nil
}

func ttyFile(stdin io.Reader) *os.File {
	f, ok := stdin.(*os.File)
	if !ok || !supportsRawTTY(f) {
		return nil
	}
	return f
}

func supportsRawTTY(f *os.File) bool {
	cmd := exec.Command("stty", "-g")
	cmd.Stdin = f
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func setTTYRaw(f *os.File) error {
	cmd := exec.Command("stty", "-icanon", "-echo", "min", "0", "time", "1")
	cmd.Stdin = f
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func restoreTTY(f *os.File) {
	cmd := exec.Command("stty", "sane")
	cmd.Stdin = f
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	_ = cmd.Run()
}

func indexOf(list []string, needle string) int {
	for i, v := range list {
		if strings.EqualFold(v, needle) {
			return i
		}
	}
	return -1
}

type timePreset struct {
	Label  string
	Hour   int
	Minute int
}

var dailyTimePresets = []timePreset{
	{Label: "00:00", Hour: 0, Minute: 0},
	{Label: "06:00", Hour: 6, Minute: 0},
	{Label: "09:00", Hour: 9, Minute: 0},
	{Label: "12:00", Hour: 12, Minute: 0},
	{Label: "18:00", Hour: 18, Minute: 0},
	{Label: "21:00", Hour: 21, Minute: 0},
}

type weeklyDayPreset struct {
	Label string
	Days  []int
}

var weeklyDayPresets = []weeklyDayPreset{
	{Label: "Mon-Fri", Days: []int{1, 2, 3, 4, 5}},
	{Label: "Sat-Sun", Days: []int{0, 6}},
	{Label: "Every day", Days: []int{0, 1, 2, 3, 4, 5, 6}},
	{Label: "Monday", Days: []int{1}},
	{Label: "Tuesday", Days: []int{2}},
	{Label: "Wednesday", Days: []int{3}},
	{Label: "Thursday", Days: []int{4}},
	{Label: "Friday", Days: []int{5}},
	{Label: "Saturday", Days: []int{6}},
	{Label: "Sunday", Days: []int{0}},
}

func timePresetLabels() []string {
	out := make([]string, 0, len(dailyTimePresets))
	for _, preset := range dailyTimePresets {
		out = append(out, preset.Label)
	}
	return out
}

func weeklyDayPresetLabels() []string {
	out := make([]string, 0, len(weeklyDayPresets))
	for _, preset := range weeklyDayPresets {
		out = append(out, preset.Label)
	}
	return out
}

func defaultTimePresetIndex(hour, minute int) int {
	for i, preset := range dailyTimePresets {
		if preset.Hour == hour && preset.Minute == minute {
			return i
		}
	}
	return 2
}

func defaultWeeklyDayPresetIndex(days []int) int {
	for i, preset := range weeklyDayPresets {
		if sameIntSlice(days, preset.Days) {
			return i
		}
	}
	return 0
}

func sameIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func nearestQuarterIndex(minute int) int {
	quarters := []int{0, 15, 30, 45}
	bestIdx := 0
	bestDiff := 60
	for i, q := range quarters {
		d := minute - q
		if d < 0 {
			d = -d
		}
		if d < bestDiff {
			bestDiff = d
			bestIdx = i
		}
	}
	return bestIdx
}

func rule(w io.Writer, ch rune) {
	fmt.Fprintln(w, strings.Repeat(string(ch), 82))
}

func colorLabel(color string) string {
	return color + cBold
}
