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
	cWhite   = "\033[97m"
	cBlue    = "\033[38;5;45m"
	cCyan    = "\033[38;5;51m"
	cGreen   = "\033[32m"
	cYellow  = "\033[38;5;226m"
	cRed     = "\033[31m"
	cMagenta = "\033[38;5;213m"
	cPink    = "\033[38;5;205m"
	cBgBlue  = "\033[48;5;17m"
	cBgMag   = "\033[48;5;54m"
	cBgBlack = "\033[48;5;16m"
	cBgPink  = "\033[48;5;89m"
)

const (
	rootFirstTenderKey = 2
	rootLastTenderKey  = 7
	rootPageUpKey      = 9
	rootPageDownKey    = 0
)

const (
	panelWidth = 86
)

func RunInteractive(root string, stdin io.Reader, stdout io.Writer) error {
	r := bufio.NewReader(stdin)
	tty := ttyFile(stdin)
	offset := 0

	for {
		tenders, err := LoadTenders(root)
		if err != nil {
			return err
		}
		sort.Slice(tenders, func(i, j int) bool { return tenders[i].Name < tenders[j].Name })
		offset = clampOffset(offset, len(tenders), rootTenderSlots())

		drawHome(stdout, tenders, offset, tty)

		action, err := promptMenuChoice(r, stdout, tty, "")
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

		case "q", "Q":
			return nil
		case strconv.Itoa(rootPageUpKey):
			if offset > 0 {
				offset -= rootTenderSlots()
			}
		case strconv.Itoa(rootPageDownKey):
			if offset+rootTenderSlots() < len(tenders) {
				offset += rootTenderSlots()
			}

		default:
			if len(action) == 1 && action[0] >= '2' && action[0] <= '7' {
				slot := int(action[0]-'0') - rootFirstTenderKey
				selectedIndex := offset + slot
				if selectedIndex >= 0 && selectedIndex < len(tenders) {
					if err := runTenderMenu(r, stdout, root, tty, tenders[selectedIndex].Name); err != nil {
						return err
					}
					continue
				}
			}
			printErr(stdout, "Invalid selection.")
		}
	}
}

func drawHome(w io.Writer, tenders []Tender, offset int, tty *os.File) {
	w = beginScreen(w, tty, 22)
	drawHero(w)
	fmt.Fprintln(w)
	drawMeta(w, len(tenders))
	fmt.Fprintln(w)

	fmt.Fprintf(w, "%sSelect Tender%s\n", colorLabel(cCyan), cReset)
	rule(w, '.')
	fmt.Fprintf(w, "  %s  Create tender\n", numberChip(1))
	for i := 0; i < rootTenderSlots(); i++ {
		key := rootFirstTenderKey + i
		idx := offset + i
		if idx >= 0 && idx < len(tenders) {
			t := tenders[idx]
			fmt.Fprintf(w, "  %s  %-20s %-30s\n", numberChip(key), t.Name, paintTrigger(TriggerSummary(t.Cron, t.Manual), t.Cron, t.Manual))
			continue
		}
		fmt.Fprintf(w, "  %s  %s(empty)%s\n", numberChip(key), cDim, cReset)
	}
	fmt.Fprintf(w, "  %s  Scroll up\n", numberChip(rootPageUpKey))
	fmt.Fprintf(w, "  %s  Scroll down\n", numberChip(rootPageDownKey))
	fmt.Fprintf(w, "  %s  Exit\n", keyChip("q"))
	rule(w, '.')
	if len(tenders) == 0 {
		fmt.Fprintf(w, "%sShowing 0 tenders%s\n", cDim, cReset)
		return
	}
	pageSize := rootTenderSlots()
	start := offset + 1
	end := min(offset+pageSize, len(tenders))
	page := (offset / pageSize) + 1
	pages := (len(tenders) + pageSize - 1) / pageSize
	fmt.Fprintf(w, "%sShowing %d-%d of %d (page %d/%d)%s\n", cDim, start, end, len(tenders), page, pages, cReset)
}

func drawHero(w io.Writer) {
	rule(w, '=')
	paintBand(w, cBgBlack, cWhite, "                                                                                ")
	paintBand(w, cBgBlue, cWhite, "                                    TENDER                                      ")
	paintBand(w, cBgMag, cWhite, "                              VAPORWAVE CONTROL                                 ")
	paintBand(w, cBgPink, cWhite, "                  Autonomous OpenCode runs in GitHub Actions                    ")
	paintBand(w, cBgBlue, cWhite, "                                                                                ")
	paintBand(w, cBgBlack, cWhite, "                                                                                ")
	rule(w, '=')
}

func drawMeta(w io.Writer, count int) {
	fmt.Fprintf(w, "%s%s%s STATE %s GitHub Actions workflows (.github/workflows)\n", cBgMag, cWhite, cBold, cReset)
	fmt.Fprintf(w, "%s%s%s MODE  %s Autonomous commits to main\n", cBgBlue, cWhite, cBold, cReset)
	fmt.Fprintf(w, "%s%s%s COUNT %s %d total tender(s)\n", cBgPink, cWhite, cBold, cReset, count)
}

func inputTender(r *bufio.Reader, w io.Writer, root string, base Tender, isNew bool, tty *os.File) (Tender, bool, error) {
	w = beginScreen(w, tty, 24)
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
	_, _ = selectNumberedOption(r, w, tty, "Continue", []string{"Back to dashboard"}, 0, false)
}

func printErr(w io.Writer, msg string) {
	fmt.Fprintf(w, "%sERROR:%s %s\n", cRed, cReset, msg)
}

func printOK(w io.Writer, msg string) {
	fmt.Fprintf(w, "%sOK:%s %s\n", cGreen, cReset, msg)
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
	fmt.Fprintf(w, "%s%s Tender%s\n", colorLabel(cCyan), action, cReset)
	rule(w, '.')
	for i, t := range tenders {
		fmt.Fprintf(w, "  %s %-20s %-30s %s\n", numberChip(i+1), t.Name, TriggerSummary(t.Cron, t.Manual), t.WorkflowFile)
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

func runTenderMenu(r *bufio.Reader, w io.Writer, root string, tty *os.File, name string) error {
	current := name
	for {
		tenders, err := LoadTenders(root)
		if err != nil {
			return err
		}
		sort.Slice(tenders, func(i, j int) bool { return tenders[i].Name < tenders[j].Name })
		idx := findTenderIndex(tenders, current)
		if idx < 0 {
			printErr(w, "Tender no longer exists.")
			return nil
		}
		selected := tenders[idx]
		sw := beginScreen(w, tty, 18)
		drawHero(sw)
		fmt.Fprintln(sw)
		fmt.Fprintf(sw, "%sTender%s %s%s%s\n", colorLabel(cPink), cReset, cBold, selected.Name, cReset)
		fmt.Fprintf(sw, "%sAgent:%s %s\n", cDim, cReset, selected.Agent)
		fmt.Fprintf(sw, "%sTrigger:%s %s\n", cDim, cReset, paintTrigger(TriggerSummary(selected.Cron, selected.Manual), selected.Cron, selected.Manual))
		fmt.Fprintf(sw, "%sWorkflow:%s %s\n", cDim, cReset, selected.WorkflowFile)
		fmt.Fprintln(sw)
		rule(sw, '.')
		fmt.Fprintf(sw, "  %s  Back\n", numberChip(1))
		fmt.Fprintf(sw, "  %s  Edit\n", numberChip(2))
		fmt.Fprintf(sw, "  %s  Delete\n", numberChip(3))
		rule(sw, '.')

		action, err := promptMenuChoice(r, sw, tty, "")
		if err != nil {
			return err
		}
		switch strings.TrimSpace(action) {
		case "1":
			return nil
		case "2":
			updated, ok, err := inputTender(r, w, root, selected, false, tty)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			if err := UpdateTender(root, selected.Name, updated); err != nil {
				printErr(sw, err.Error())
				acknowledge(r, sw, tty)
				continue
			}
			current = updated.Name
			printOK(sw, "Updated "+selected.WorkflowFile)
		case "3":
			confirm, err := promptBinaryChoice(r, sw, tty, fmt.Sprintf("Delete %q?", selected.Name), false, false)
			if err != nil {
				return err
			}
			if !confirm {
				printErr(sw, "Delete cancelled")
				continue
			}
			if err := RemoveTender(root, selected.Name); err != nil {
				return err
			}
			printOK(sw, "Deleted "+selected.WorkflowFile)
			return nil
		default:
			printErr(sw, "Invalid selection.")
		}
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
		line := option
		if hasDefault && i == defaultIndex {
			line = fmt.Sprintf("%s %s(default)%s", option, cDim, cReset)
		}
		fmt.Fprintf(w, "  %s  %s\n", numberChip(i+1), line)
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
	fmt.Fprintln(w, strings.Repeat(string(ch), 86))
}

func colorLabel(color string) string {
	return color + cBold
}

func paintBand(w io.Writer, bg string, fg string, text string) {
	fmt.Fprintf(w, "%s%s%s%s%s\n", bg, fg, cBold, text, cReset)
}

func paintTrigger(summary, cron string, manual bool) string {
	switch {
	case cron != "" && manual:
		return cCyan + summary + cReset
	case cron != "":
		return cMagenta + summary + cReset
	case manual:
		return cGreen + summary + cReset
	default:
		return summary
	}
}

func numberChip(n int) string {
	bgs := []string{cBgBlue, cBgMag, cBgPink, cBgBlue, cBgMag}
	slot := n
	if slot <= 0 {
		slot = len(bgs)
	}
	bg := bgs[(slot-1)%len(bgs)]
	return fmt.Sprintf("%s%s%s %d %s", bg, cWhite, cBold, n, cReset)
}

func keyChip(key string) string {
	return fmt.Sprintf("%s%s%s %s %s", cBgBlack, cWhite, cBold, key, cReset)
}

type prefixedWriter struct {
	w           io.Writer
	prefix      string
	atLineStart bool
}

func (p *prefixedWriter) Write(b []byte) (int, error) {
	if len(p.prefix) == 0 {
		return p.w.Write(b)
	}
	total := 0
	for len(b) > 0 {
		if p.atLineStart {
			if _, err := io.WriteString(p.w, p.prefix); err != nil {
				return total, err
			}
			p.atLineStart = false
		}
		i := bytesIndexByte(b, '\n')
		if i < 0 {
			n, err := p.w.Write(b)
			total += n
			return total, err
		}
		n, err := p.w.Write(b[:i+1])
		total += n
		if err != nil {
			return total, err
		}
		p.atLineStart = true
		b = b[i+1:]
	}
	return total, nil
}

func beginScreen(w io.Writer, tty *os.File, contentHeight int) io.Writer {
	rows, cols := terminalSize(tty)
	if rows <= 0 {
		rows = 34
	}
	if cols <= 0 {
		cols = 120
	}

	clearScreen(w)
	line := cBgBlack + strings.Repeat(" ", cols) + cReset + "\n"
	for i := 0; i < rows; i++ {
		fmt.Fprint(w, line)
	}
	fmt.Fprint(w, "\033[H")

	topPad := 0
	if rows > contentHeight {
		topPad = (rows - contentHeight) / 2
	}
	for i := 0; i < topPad; i++ {
		fmt.Fprintln(w)
	}

	leftPad := 0
	if cols > panelWidth {
		leftPad = (cols - panelWidth) / 2
	}
	if leftPad <= 0 {
		return w
	}
	return &prefixedWriter{
		w:           w,
		prefix:      strings.Repeat(" ", leftPad),
		atLineStart: true,
	}
}

func terminalSize(tty *os.File) (rows int, cols int) {
	if tty == nil {
		return 0, 0
	}
	cmd := exec.Command("stty", "size")
	cmd.Stdin = tty
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return 0, 0
	}
	parts := strings.Fields(strings.TrimSpace(out.String()))
	if len(parts) != 2 {
		return 0, 0
	}
	r, errR := strconv.Atoi(parts[0])
	c, errC := strconv.Atoi(parts[1])
	if errR != nil || errC != nil {
		return 0, 0
	}
	return r, c
}

func bytesIndexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

func rootTenderSlots() int {
	return rootLastTenderKey - rootFirstTenderKey + 1
}

func clampOffset(offset, total, pageSize int) int {
	if offset < 0 {
		return 0
	}
	if total <= 0 {
		return 0
	}
	max := ((total - 1) / pageSize) * pageSize
	if offset > max {
		return max
	}
	return offset
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
