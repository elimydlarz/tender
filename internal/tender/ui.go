package tender

import (
	"bufio"
	"errors"
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

var errQuitRequested = errors.New("quit requested")

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
			if errors.Is(err, errQuitRequested) {
				return nil
			}
			return err
		}

		switch strings.TrimSpace(action) {
		case "1":
			base := Tender{Manual: true, Push: false}
			t, ok, err := inputTender(r, stdout, root, base, true, tty)
			if err != nil {
				if errors.Is(err, errQuitRequested) {
					return nil
				}
				return err
			}
			if !ok {
				continue
			}

			saved, err := SaveNewTender(root, t)
			if err != nil {
				printErr(stdout, err.Error())
				if err := acknowledge(r, stdout, tty); err != nil {
					if errors.Is(err, errQuitRequested) {
						return nil
					}
					return err
				}
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
						if errors.Is(err, errQuitRequested) {
							return nil
						}
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
	// Keep vertical centering aligned with the actual rendered dashboard height.
	w = beginScreen(w, tty, 21+rootTenderSlots())
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
			fmt.Fprintf(w, "  %s  %-20s %-30s\n", numberChip(key), t.Name, paintTrigger(TriggerSummary(t.Cron, t.Manual, t.Push), t.Cron, t.Manual, t.Push))
			continue
		}
		fmt.Fprintln(w)
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
	paintBand(w, cBgMag, cWhite, "                                                                                ")
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

func drawTenderFormScreen(w io.Writer, tty *os.File, root string, isNew bool, draft Tender, notice string, showAgent bool) io.Writer {
	w = beginScreen(w, tty, 21+rootTenderSlots())
	drawHero(w)
	fmt.Fprintln(w)
	drawMeta(w, tenderCount(root))
	fmt.Fprintln(w)

	if isNew {
		fmt.Fprintf(w, "%sCreate Tender%s\n", colorLabel(cCyan), cReset)
	} else {
		fmt.Fprintf(w, "%sEdit Tender%s\n", colorLabel(cCyan), cReset)
	}
	rule(w, '-')

	nameValue := strings.TrimSpace(draft.Name)
	if nameValue == "" {
		nameValue = "(pending)"
	}
	context := fmt.Sprintf("Current: name=%s", nameValue)
	if showAgent {
		agentValue := strings.TrimSpace(draft.Agent)
		if agentValue == "" {
			agentValue = "(pending)"
		}
		context = fmt.Sprintf("%s | agent=%s", context, agentValue)
	}
	fmt.Fprintln(w, context)

	if strings.TrimSpace(notice) != "" {
		fmt.Fprintf(w, "%sNote:%s %s\n", cYellow, cReset, notice)
	}
	fmt.Fprintln(w)
	return w
}

func acknowledgeTenderForm(r *bufio.Reader, w io.Writer, tty *os.File, root string, isNew bool, draft Tender, notice string, msg string, showAgent bool) error {
	screen := drawTenderFormScreen(w, tty, root, isNew, draft, notice, showAgent)
	printErr(screen, msg)
	return acknowledge(r, screen, tty)
}

func inputTender(r *bufio.Reader, w io.Writer, root string, base Tender, isNew bool, tty *os.File) (Tender, bool, error) {
	draft := base
	if isNew {
		// In create flow, agent is not selected yet. Don't preview a default.
		draft.Agent = ""
	}

	namePrompt := "Name: "
	if base.Name != "" {
		namePrompt = fmt.Sprintf("Name (default: %s): ", base.Name)
	}
	nameScreen := drawTenderFormScreen(w, tty, root, isNew, draft, "", false)
	nameInput, err := promptText(r, nameScreen, namePrompt)
	if err != nil {
		return Tender{}, false, err
	}
	nameInput = strings.TrimSpace(nameInput)
	if nameInput == "" {
		if strings.TrimSpace(base.Name) == "" {
			if err := acknowledgeTenderForm(r, w, tty, root, isNew, draft, "", "Name is required.", false); err != nil {
				if errors.Is(err, errQuitRequested) {
					return base, false, nil
				}
				return Tender{}, false, err
			}
			return base, false, nil
		}
		nameInput = base.Name
	}
	name := strings.TrimSpace(nameInput)
	draft.Name = name

	agent, err := chooseAgent(r, w, root, base.Agent, tty, isNew, name)
	if err != nil {
		if errors.Is(err, errQuitRequested) {
			return base, false, nil
		}
		return Tender{}, false, err
	}
	draft.Agent = agent

	pushScreen := drawTenderFormScreen(w, tty, root, isNew, draft, "", true)
	push, err := promptBinaryChoice(r, pushScreen, tty, "Run on every push to main?", base.Push, false)
	if err != nil {
		if errors.Is(err, errQuitRequested) {
			return base, false, nil
		}
		return Tender{}, false, err
	}
	draft.Push = push

	timeoutDefault := normalizeTimeoutMinutes(base.TimeoutMinutes)
	timeoutScreen := drawTenderFormScreen(w, tty, root, isNew, draft, "", true)
	timeoutMinutes, err := promptTimeoutMinutes(r, timeoutScreen, timeoutDefault)
	if err != nil {
		if errors.Is(err, errQuitRequested) {
			return base, false, nil
		}
		return Tender{}, false, err
	}
	draft.TimeoutMinutes = timeoutMinutes

	hasScheduleDefault := isNew || strings.TrimSpace(base.Cron) != ""
	scheduleToggleScreen := drawTenderFormScreen(w, tty, root, isNew, draft, "", true)
	hasSchedule, err := promptBinaryChoice(r, scheduleToggleScreen, tty, "Enable recurring schedule?", hasScheduleDefault, false)
	if err != nil {
		if errors.Is(err, errQuitRequested) {
			return base, false, nil
		}
		return Tender{}, false, err
	}

	cron := strings.TrimSpace(base.Cron)
	if hasSchedule {
		defaults, hasDefaults := scheduleDefaultsFromCron(cron)
		notice := ""
		if cron != "" && !hasDefaults {
			notice = "Existing schedule is unsupported in presets; choose a new one."
		}

		defaultMode := 1
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

		scheduleModeScreen := drawTenderFormScreen(w, tty, root, isNew, draft, notice, true)
		modeIndex, err := selectNumberedOption(r, scheduleModeScreen, tty, "Schedule", []string{"Hourly", "Daily", "Weekly"}, defaultMode, true)
		if err != nil {
			if errors.Is(err, errQuitRequested) {
				return base, false, nil
			}
			return Tender{}, false, err
		}

		switch modeIndex {
		case 0:
			minuteDefault := 0
			if hasDefaults {
				minuteDefault = nearestQuarterIndex(defaults.Minute)
			}
			hourlyMinuteScreen := drawTenderFormScreen(w, tty, root, isNew, draft, "", true)
			minuteIndex, err := selectNumberedOption(r, hourlyMinuteScreen, tty, "Hourly minute", []string{":00", ":15", ":30", ":45"}, minuteDefault, true)
			if err != nil {
				if errors.Is(err, errQuitRequested) {
					return base, false, nil
				}
				return Tender{}, false, err
			}
			minutes := []int{0, 15, 30, 45}
			built, err := buildHourlyCron(strconv.Itoa(minutes[minuteIndex]))
			if err != nil {
				if err := acknowledgeTenderForm(r, w, tty, root, isNew, draft, "", err.Error(), true); err != nil {
					if errors.Is(err, errQuitRequested) {
						return base, false, nil
					}
					return Tender{}, false, err
				}
				return base, false, nil
			}
			cron = built

		case 1:
			timeDefault := 2
			if hasDefaults {
				timeDefault = defaultTimePresetIndex(defaults.Hour, defaults.Minute)
			}
			dailyTimeScreen := drawTenderFormScreen(w, tty, root, isNew, draft, "", true)
			timeIndex, err := selectNumberedOption(r, dailyTimeScreen, tty, "Daily time (UTC)", timePresetLabels(), timeDefault, true)
			if err != nil {
				if errors.Is(err, errQuitRequested) {
					return base, false, nil
				}
				return Tender{}, false, err
			}
			preset := dailyTimePresets[timeIndex]
			built, err := buildDailyCron(formatTime(preset.Hour, preset.Minute))
			if err != nil {
				if err := acknowledgeTenderForm(r, w, tty, root, isNew, draft, "", err.Error(), true); err != nil {
					if errors.Is(err, errQuitRequested) {
						return base, false, nil
					}
					return Tender{}, false, err
				}
				return base, false, nil
			}
			cron = built

		case 2:
			dayDefault := 0
			if hasDefaults {
				dayDefault = defaultWeeklyDayPresetIndex(defaults.Days)
			}
			weeklyDaysScreen := drawTenderFormScreen(w, tty, root, isNew, draft, "", true)
			dayIndex, err := selectNumberedOption(r, weeklyDaysScreen, tty, "Weekly days", weeklyDayPresetLabels(), dayDefault, true)
			if err != nil {
				if errors.Is(err, errQuitRequested) {
					return base, false, nil
				}
				return Tender{}, false, err
			}

			timeDefault := 2
			if hasDefaults {
				timeDefault = defaultTimePresetIndex(defaults.Hour, defaults.Minute)
			}
			weeklyTimeScreen := drawTenderFormScreen(w, tty, root, isNew, draft, "", true)
			timeIndex, err := selectNumberedOption(r, weeklyTimeScreen, tty, "Weekly time (UTC)", timePresetLabels(), timeDefault, true)
			if err != nil {
				if errors.Is(err, errQuitRequested) {
					return base, false, nil
				}
				return Tender{}, false, err
			}

			days := weeklyDayPresets[dayIndex].Days
			preset := dailyTimePresets[timeIndex]
			built, err := buildWeeklyCron(joinInts(days, ","), formatTime(preset.Hour, preset.Minute))
			if err != nil {
				if err := acknowledgeTenderForm(r, w, tty, root, isNew, draft, "", err.Error(), true); err != nil {
					if errors.Is(err, errQuitRequested) {
						return base, false, nil
					}
					return Tender{}, false, err
				}
				return base, false, nil
			}
			cron = built
		}
	} else {
		cron = ""
	}

	result := Tender{
		Name:           name,
		Agent:          strings.TrimSpace(agent),
		Prompt:         strings.TrimSpace(base.Prompt),
		Cron:           strings.TrimSpace(cron),
		Manual:         true,
		Push:           push,
		TimeoutMinutes: timeoutMinutes,
		WorkflowFile:   base.WorkflowFile,
	}

	if err := ValidateTender(result); err != nil {
		if err := acknowledgeTenderForm(r, w, tty, root, isNew, draft, "", err.Error(), true); err != nil {
			if errors.Is(err, errQuitRequested) {
				return base, false, nil
			}
			return Tender{}, false, err
		}
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
	choice, err := promptText(r, w, label)
	if err != nil {
		return "", err
	}
	if isQuitChoice(choice) {
		return "", errQuitRequested
	}
	return choice, nil
}

func promptText(r *bufio.Reader, w io.Writer, label string) (string, error) {
	fmt.Fprint(w, label)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func promptTimeoutMinutes(r *bufio.Reader, w io.Writer, defaultTimeout int) (int, error) {
	defaultTimeout = normalizeTimeoutMinutes(defaultTimeout)
	for {
		label := fmt.Sprintf("Timeout in minutes (default: %d): ", defaultTimeout)
		raw, err := prompt(r, w, label)
		if err != nil {
			return 0, err
		}

		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return defaultTimeout, nil
		}

		minutes, err := strconv.Atoi(trimmed)
		if err != nil || minutes <= 0 {
			printErr(w, "Timeout must be a positive whole number of minutes.")
			continue
		}
		return minutes, nil
	}
}

func clearScreen(w io.Writer) {
	fmt.Fprint(w, "\033[H\033[2J")
}

func acknowledge(r *bufio.Reader, w io.Writer, tty *os.File) error {
	_, err := selectNumberedOption(r, w, tty, "Continue", []string{"Back to dashboard"}, 0, false)
	return err
}

func printErr(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %sERROR:%s %s\n", cRed, cReset, msg)
}

func printInfo(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %sINFO:%s %s\n", cDim, cReset, msg)
}

func printOK(w io.Writer, msg string) {
	fmt.Fprintf(w, "  %sOK:%s %s\n", cGreen, cReset, msg)
}

func chooseAgent(r *bufio.Reader, w io.Writer, root string, current string, tty *os.File, isNew bool, name string) (string, error) {
	agents, err := DiscoverPrimaryAgents(root)
	if err != nil {
		return "", fmt.Errorf("unable to discover OpenCode agents: %w", err)
	}
	if len(agents) == 0 {
		return "", fmt.Errorf("no custom OpenCode agents found")
	}

	defaultIndex := 0
	if current != "" {
		if idx := indexOf(agents, current); idx >= 0 {
			defaultIndex = idx
		}
	}
	// Render a fresh dedicated step screen for agent selection.
	step := Tender{Name: name}
	screen := drawTenderFormScreen(w, tty, root, isNew, step, "", false)
	idx, err := selectNumberedOption(r, screen, tty, "Agent", agents, defaultIndex, true)
	if err != nil {
		return "", err
	}
	return agents[idx], nil
}

func selectTender(r *bufio.Reader, w io.Writer, tty *os.File, tenders []Tender, action string) (Tender, bool, error) {
	if len(tenders) == 0 {
		printErr(w, "No tenders available")
		if err := acknowledge(r, w, tty); err != nil {
			return Tender{}, false, err
		}
		return Tender{}, false, nil
	}

	sort.Slice(tenders, func(i, j int) bool { return tenders[i].Name < tenders[j].Name })
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s%s Tender%s\n", colorLabel(cCyan), action, cReset)
	rule(w, '.')
	for i, t := range tenders {
		fmt.Fprintf(w, "  %s %-20s %-30s %s\n", numberChip(i+1), t.Name, TriggerSummary(t.Cron, t.Manual, t.Push), t.WorkflowFile)
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
			if err := acknowledge(r, w, tty); err != nil {
				return Tender{}, false, err
			}
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
		sw := beginScreen(w, tty, 21+rootTenderSlots())
		drawHero(sw)
		fmt.Fprintln(sw)
		fmt.Fprintf(sw, "%sTender%s %s%s%s\n", colorLabel(cPink), cReset, cBold, selected.Name, cReset)
		fmt.Fprintf(sw, "%s%-9s%s %s\n", cDim, "Agent:", cReset, selected.Agent)
		fmt.Fprintf(sw, "%s%-9s%s %s\n", cDim, "Trigger:", cReset, paintTrigger(TriggerSummary(selected.Cron, selected.Manual, selected.Push), selected.Cron, selected.Manual, selected.Push))
		fmt.Fprintf(sw, "%s%-9s%s %d min\n", cDim, "Timeout:", cReset, normalizeTimeoutMinutes(selected.TimeoutMinutes))
		fmt.Fprintf(sw, "%s%-9s%s %s\n", cDim, "Workflow:", cReset, selected.WorkflowFile)
		fmt.Fprintln(sw)
		rule(sw, '.')
		fmt.Fprintf(sw, "  %s  Back\n", numberChip(1))
		fmt.Fprintf(sw, "  %s  Edit\n", numberChip(2))
		fmt.Fprintf(sw, "  %s  Delete\n", numberChip(3))
		for i := 3; i < rootTenderSlots(); i++ {
			fmt.Fprintln(sw)
		}
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
				if err := acknowledge(r, sw, tty); err != nil {
					return err
				}
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

	if len(options) <= 9 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s%s%s\n", colorLabel(cCyan), title, cReset)
		rule(w, '.')
		for i := 0; i < len(options); i++ {
			line := options[i]
			if hasDefault && i == defaultIndex {
				line = fmt.Sprintf("%s %s(default)%s", line, cDim, cReset)
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

	// Single-key menu mode for long lists: 1-8 select current page, 9/0 scroll.
	const pageFirstKey = 1
	const pageLastKey = 8
	pageSize := pageLastKey - pageFirstKey + 1
	pageRenderLines := pageSize + 8
	offset := (defaultIndex / pageSize) * pageSize
	offset = clampOffset(offset, len(options), pageSize)
	controlWriter := w
	if pw, ok := w.(*prefixedWriter); ok {
		controlWriter = pw.w
	}
	canRepaint := tty != nil && supportsRawTTY(tty)

	renderPage := func(redraw bool, status string) {
		if redraw && canRepaint {
			// Replace previous option block + prior prompt line in place.
			fmt.Fprintf(controlWriter, "\033[%dA\033[J", pageRenderLines+1)
		}
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s%s%s\n", colorLabel(cCyan), title, cReset)
		rule(w, '.')
		for slot := 0; slot < pageSize; slot++ {
			idx := offset + slot
			if idx >= 0 && idx < len(options) {
				line := options[idx]
				if hasDefault && idx == defaultIndex {
					line = fmt.Sprintf("%s %s(default)%s", line, cDim, cReset)
				}
				fmt.Fprintf(w, "  %s  %s\n", numberChip(slot+1), line)
				continue
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "  %s  Scroll up\n", numberChip(rootPageUpKey))
		fmt.Fprintf(w, "  %s  Scroll down\n", numberChip(rootPageDownKey))
		rule(w, '.')
		start := offset + 1
		end := min(offset+pageSize, len(options))
		page := (offset / pageSize) + 1
		pages := (len(options) + pageSize - 1) / pageSize
		fmt.Fprintf(w, "%sShowing %d-%d of %d (page %d/%d)%s\n", cDim, start, end, len(options), page, pages, cReset)
		if strings.TrimSpace(status) != "" {
			printInfo(w, status)
		} else {
			fmt.Fprintln(w)
		}
	}

	status := ""
	redraw := false
	for {
		renderPage(redraw, status)
		redraw = true
		status = ""

		label := "Choose 1-8, 9(up), 0(down): "
		if hasDefault {
			label = "Choose 1-8, 9(up), 0(down) (Enter for default): "
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
			status = "Selection required."
			continue
		}

		switch choice {
		case strconv.Itoa(rootPageUpKey):
			next := offset - pageSize
			if next < 0 {
				next = 0
			}
			if next == offset {
				status = "Already at first page."
				continue
			}
			offset = next
			continue
		case strconv.Itoa(rootPageDownKey):
			next := offset + pageSize
			if next+pageSize > len(options) && next >= len(options) {
				status = "Already at last page."
				continue
			}
			next = clampOffset(next, len(options), pageSize)
			if next == offset {
				status = "Already at last page."
				continue
			}
			offset = next
			continue
		}

		if len(choice) == 1 && choice[0] >= '1' && choice[0] <= '8' {
			slot := int(choice[0] - '1')
			idx := offset + slot
			if idx >= 0 && idx < len(options) {
				return idx, nil
			}
		}
		status = "Invalid selection."
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
	if isQuitChoice(choice) {
		fmt.Fprintln(w, choice)
		return "", errQuitRequested
	}
	fmt.Fprintln(w, choice)
	return choice, nil
}

func isQuitChoice(choice string) bool {
	return strings.EqualFold(strings.TrimSpace(choice), "q")
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

func paintTrigger(summary, cron string, manual bool, push bool) string {
	switch {
	case cron != "" && manual && push:
		return cCyan + summary + cReset
	case cron != "" && manual:
		return cCyan + summary + cReset
	case cron != "" && push:
		return cPink + summary + cReset
	case cron != "":
		return cMagenta + summary + cReset
	case push && manual:
		return cBlue + summary + cReset
	case push:
		return cBlue + summary + cReset
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

func tenderCount(root string) int {
	tenders, err := LoadTenders(root)
	if err != nil {
		return 0
	}
	return len(tenders)
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
