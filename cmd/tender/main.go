package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tender/internal/tender"
)

const (
	addUsageLine    = "usage: tender add [--name <name>] --agent <agent> [--prompt \"...\"] [--cron \"...\"] [--manual true|false] [--push true|false] [--timeout-minutes <minutes>] [<name>]"
	updateUsageLine = "usage: tender update <name> [--name <new-name>] [--agent <agent>] [--prompt \"...\"] [--cron \"...\"] [--clear-cron] [--manual true|false] [--push true|false] [--timeout-minutes <minutes>]"
	runUsageLine    = "usage: tender run [--prompt \"...\"] <name>"
	rmUsageLine     = "usage: tender rm [--yes] <name>"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}

	if len(os.Args) < 2 {
		if err := tender.EnsureWorkflowDir(root); err != nil {
			fail(err)
		}
		if err := tender.RunInteractive(root, os.Stdin, os.Stdout); err != nil {
			fail(err)
		}
		return
	}

	switch os.Args[1] {
	case "init":
		if err := tender.EnsureWorkflowDir(root); err != nil {
			fail(err)
		}
		fmt.Printf("initialized %s\n", filepath.Join(root, tender.WorkflowDir))

	case "add":
		rawArgs := os.Args[2:]
		if hasHelpFlag(rawArgs, map[string]struct{}{
			"-agent":            {},
			"--agent":           {},
			"-name":             {},
			"--name":            {},
			"-prompt":           {},
			"--prompt":          {},
			"-cron":             {},
			"--cron":            {},
			"-manual":           {},
			"--manual":          {},
			"-push":             {},
			"--push":            {},
			"-timeout-minutes":  {},
			"--timeout-minutes": {},
			"-timeout":          {},
			"--timeout":         {},
		}) {
			usage()
			fmt.Println()
			printAddHelp()
			return
		}
		fs := flag.NewFlagSet("add", flag.ExitOnError)
		name := fs.String("name", "", "tender name")
		agent := fs.String("agent", "", "OpenCode agent name")
		prompt := fs.String("prompt", "", "optional default prompt")
		cron := fs.String("cron", "", "optional cron schedule (5 fields, UTC)")
		manual := fs.String("manual", "", "set workflow_dispatch trigger (true/false)")
		push := fs.String("push", "", "set push-to-main trigger (true/false)")
		timeoutMinutes := tender.DefaultTimeoutMinutes
		fs.IntVar(&timeoutMinutes, "timeout-minutes", tender.DefaultTimeoutMinutes, "job timeout in minutes")
		fs.IntVar(&timeoutMinutes, "timeout", tender.DefaultTimeoutMinutes, "alias for --timeout-minutes")
		positionalName := ""
		if len(rawArgs) > 0 && !strings.HasPrefix(rawArgs[0], "-") {
			positionalName = strings.TrimSpace(rawArgs[0])
			rawArgs = rawArgs[1:]
		}
		_ = fs.Parse(rawArgs)
		args := fs.Args()
		if positionalName != "" && len(args) > 0 {
			fmt.Fprintln(os.Stderr, addUsageLine)
			os.Exit(2)
		}
		if len(args) > 1 {
			fmt.Fprintln(os.Stderr, addUsageLine)
			os.Exit(2)
		}
		finalName := strings.TrimSpace(*name)
		if positionalName != "" {
			if finalName != "" {
				fail(fmt.Errorf("use either positional <name> or --name, not both"))
			}
			finalName = positionalName
		}
		if finalName == "" && len(args) == 1 {
			finalName = strings.TrimSpace(args[0])
		}
		if finalName == "" {
			fmt.Fprintln(os.Stderr, addUsageLine)
			os.Exit(2)
		}

		manualValue := true
		if isFlagSet(fs, "manual") {
			b, err := parseBoolFlag(*manual, "manual")
			if err != nil {
				fail(err)
			}
			manualValue = b
		}
		pushValue := false
		if isFlagSet(fs, "push") {
			b, err := parseBoolFlag(*push, "push")
			if err != nil {
				fail(err)
			}
			pushValue = b
		}
		timeoutValue, err := parseTimeoutMinutesFlag(timeoutMinutes)
		if err != nil {
			fail(err)
		}

		agentName := strings.TrimSpace(*agent)
		if err := requireCustomAgent(root, agentName); err != nil {
			fail(err)
		}
		saved, err := tender.SaveNewTender(root, tender.Tender{
			Name:           finalName,
			Agent:          agentName,
			Prompt:         strings.TrimSpace(*prompt),
			Cron:           strings.TrimSpace(*cron),
			Manual:         manualValue,
			Push:           pushValue,
			TimeoutMinutes: timeoutValue,
		})
		if err != nil {
			fail(err)
		}
		fmt.Printf("saved %s\n", saved.WorkflowFile)

	case "update":
		rawArgs := os.Args[2:]
		if hasHelpFlag(rawArgs, map[string]struct{}{
			"-name":             {},
			"--name":            {},
			"-agent":            {},
			"--agent":           {},
			"-prompt":           {},
			"--prompt":          {},
			"-cron":             {},
			"--cron":            {},
			"-manual":           {},
			"--manual":          {},
			"-push":             {},
			"--push":            {},
			"-clear-cron":       {},
			"--clear-cron":      {},
			"-timeout-minutes":  {},
			"--timeout-minutes": {},
			"-timeout":          {},
			"--timeout":         {},
		}) {
			usage()
			fmt.Println()
			printUpdateHelp()
			return
		}
		fs := flag.NewFlagSet("update", flag.ExitOnError)
		name := fs.String("name", "", "new tender name")
		agent := fs.String("agent", "", "OpenCode agent name")
		prompt := fs.String("prompt", "", "default prompt (set empty string to clear)")
		cron := fs.String("cron", "", "cron schedule (5 fields, UTC)")
		clearCron := fs.Bool("clear-cron", false, "remove schedule")
		manual := fs.String("manual", "", "set workflow_dispatch trigger (true/false)")
		push := fs.String("push", "", "set push-to-main trigger (true/false)")
		timeoutMinutes := 0
		fs.IntVar(&timeoutMinutes, "timeout-minutes", 0, "set job timeout in minutes")
		fs.IntVar(&timeoutMinutes, "timeout", 0, "alias for --timeout-minutes")
		targetName := ""
		if len(rawArgs) > 0 && !strings.HasPrefix(rawArgs[0], "-") {
			targetName = strings.TrimSpace(rawArgs[0])
			rawArgs = rawArgs[1:]
		}
		_ = fs.Parse(rawArgs)
		args := fs.Args()
		if targetName != "" && len(args) > 0 {
			fmt.Fprintln(os.Stderr, updateUsageLine)
			os.Exit(2)
		}
		if targetName == "" {
			if len(args) != 1 {
				fmt.Fprintln(os.Stderr, updateUsageLine)
				os.Exit(2)
			}
			targetName = strings.TrimSpace(args[0])
		}
		if isFlagSet(fs, "cron") && *clearCron {
			fail(fmt.Errorf("use either --cron or --clear-cron, not both"))
		}

		current, err := tender.LoadTenders(root)
		if err != nil {
			fail(err)
		}
		existing, ok := findTenderByName(current, targetName)
		if !ok {
			fail(fmt.Errorf("tender %q not found", targetName))
		}

		updated := existing
		changed := false

		if isFlagSet(fs, "name") {
			updated.Name = strings.TrimSpace(*name)
			changed = true
		}
		if isFlagSet(fs, "agent") {
			updated.Agent = strings.TrimSpace(*agent)
			changed = true
		}
		if isFlagSet(fs, "prompt") {
			updated.Prompt = strings.TrimSpace(*prompt)
			changed = true
		}
		if isFlagSet(fs, "cron") {
			updated.Cron = strings.TrimSpace(*cron)
			changed = true
		}
		if *clearCron {
			updated.Cron = ""
			changed = true
		}
		if isFlagSet(fs, "manual") {
			b, err := parseBoolFlag(*manual, "manual")
			if err != nil {
				fail(err)
			}
			updated.Manual = b
			changed = true
		}
		if isFlagSet(fs, "push") {
			b, err := parseBoolFlag(*push, "push")
			if err != nil {
				fail(err)
			}
			updated.Push = b
			changed = true
		}
		if isFlagSet(fs, "timeout-minutes") || isFlagSet(fs, "timeout") {
			parsedTimeout, err := parseTimeoutMinutesFlag(timeoutMinutes)
			if err != nil {
				fail(err)
			}
			updated.TimeoutMinutes = parsedTimeout
			changed = true
		}

		if !changed {
			fail(fmt.Errorf("no update flags were provided"))
		}
		if err := requireCustomAgent(root, updated.Agent); err != nil {
			fail(err)
		}

		if err := tender.UpdateTender(root, targetName, updated); err != nil {
			fail(err)
		}
		fmt.Printf("updated %s\n", updated.WorkflowFile)

	case "ls":
		if err := tender.PrintList(root, os.Stdout); err != nil {
			fail(err)
		}

	case "rm":
		rawArgs := os.Args[2:]
		if hasHelpFlag(rawArgs, nil) {
			usage()
			fmt.Println()
			printRemoveHelp()
			return
		}
		fs := flag.NewFlagSet("rm", flag.ExitOnError)
		yes := fs.Bool("yes", false, "delete without confirmation")
		_ = fs.Parse(rawArgs)
		args := fs.Args()
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, rmUsageLine)
			os.Exit(2)
		}
		name := args[0]
		if !*yes {
			path, err := tender.ManagedWorkflowPath(root, name)
			if err != nil {
				fail(err)
			}
			fmt.Fprintf(os.Stdout, "Delete tender %q (%s)? (y/N): ", name, path)
			var confirm string
			_, _ = fmt.Fscanln(os.Stdin, &confirm)
			if confirm != "y" && confirm != "Y" && strings.ToLower(confirm) != "yes" {
				fmt.Fprintln(os.Stdout, "cancelled")
				return
			}
		}
		if err := tender.RemoveTender(root, name); err != nil {
			fail(err)
		}
		fmt.Printf("deleted %s\n", name)

	case "run":
		rawArgs := os.Args[2:]
		if hasHelpFlag(rawArgs, map[string]struct{}{
			"-prompt":  {},
			"--prompt": {},
		}) {
			usage()
			fmt.Println()
			printRunHelp()
			return
		}
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		prompt := fs.String("prompt", "", "optional prompt override for this dispatch")
		_ = fs.Parse(rawArgs)
		args := fs.Args()
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, runUsageLine)
			os.Exit(2)
		}
		name := args[0]
		if err := tender.DispatchTenderNow(root, name, *prompt, os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
		fmt.Printf("triggered %s\n", name)

	case "help":
		if len(os.Args) == 2 {
			usage()
			return
		}
		if len(os.Args) > 3 {
			fmt.Fprintln(os.Stderr, "usage: tender help [command]")
			os.Exit(2)
		}
		if err := printCommandHelp(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}

	case "-h", "--help":
		usage()

	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("tender - interactive CLI for autonomous OpenCode schedules")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  tender")
	fmt.Println("  tender <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init            Ensure .github/workflows exists")
	fmt.Println("  add             Add a tender non-interactively (agent-friendly)")
	fmt.Println("  update          Update a tender non-interactively (agent-friendly)")
	fmt.Println("  ls              List managed tender workflows")
	fmt.Println("  run             Trigger an on-demand tender now via GitHub CLI")
	fmt.Println("  rm              Remove a tender workflow")
	fmt.Println("  help [command]  Show command help")
	fmt.Println()
	fmt.Println("Tip:")
	fmt.Println("  Use `tender <command> --help` to show command-specific usage and flags.")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func isFlagSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func parseBoolFlag(raw string, name string) (bool, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return false, fmt.Errorf("--%s requires true or false", name)
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid value for --%s: %q (expected true/false)", name, raw)
	}
	return b, nil
}

func parseTimeoutMinutesFlag(value int) (int, error) {
	if value <= 0 {
		return 0, fmt.Errorf("timeout-minutes must be greater than 0")
	}
	return value, nil
}

func findTenderByName(tenders []tender.Tender, name string) (tender.Tender, bool) {
	needle := strings.TrimSpace(strings.ToLower(name))
	for _, t := range tenders {
		if strings.ToLower(strings.TrimSpace(t.Name)) == needle {
			return t, true
		}
	}
	return tender.Tender{}, false
}

func hasHelpFlag(args []string, valueFlags map[string]struct{}) bool {
	expectValue := false

	for _, arg := range args {
		if expectValue {
			expectValue = false
			continue
		}

		if arg == "--" {
			return false
		}

		if arg == "-h" || arg == "--help" {
			return true
		}

		flagName := arg
		if eq := strings.IndexByte(arg, '='); eq >= 0 {
			flagName = arg[:eq]
		}
		if _, ok := valueFlags[flagName]; ok && !strings.Contains(arg, "=") {
			expectValue = true
		}
	}
	return false
}

func printCommandHelp(command string) error {
	usage()
	fmt.Println()

	switch strings.TrimSpace(command) {
	case "add":
		printAddHelp()
	case "update":
		printUpdateHelp()
	case "run":
		printRunHelp()
	case "rm":
		printRemoveHelp()
	case "ls":
		printListHelp()
	case "init":
		printInitHelp()
	default:
		return fmt.Errorf("unknown command %q", command)
	}
	return nil
}

func printAddHelp() {
	fmt.Println("Command: add")
	fmt.Printf("  %s\n", addUsageLine)
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Provide the tender name either as positional <name> or --name.")
	fmt.Println("  - --manual defaults to true, --push defaults to false.")
	fmt.Println("  - --timeout-minutes defaults to 30.")
}

func printUpdateHelp() {
	fmt.Println("Command: update")
	fmt.Printf("  %s\n", updateUsageLine)
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Target tender name is required as positional <name>.")
	fmt.Println("  - Use --clear-cron to remove schedule.")
	fmt.Println("  - Use --timeout-minutes to override the workflow job timeout.")
}

func printRunHelp() {
	fmt.Println("Command: run")
	fmt.Printf("  %s\n", runUsageLine)
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Requires GitHub CLI auth for the repository.")
}

func printRemoveHelp() {
	fmt.Println("Command: rm")
	fmt.Printf("  %s\n", rmUsageLine)
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Use --yes to skip delete confirmation prompt.")
}

func printListHelp() {
	fmt.Println("Command: ls")
	fmt.Println("  usage: tender ls")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Lists tender workflows currently managed in .github/workflows.")
}

func printInitHelp() {
	fmt.Println("Command: init")
	fmt.Println("  usage: tender init")
	fmt.Println()
	fmt.Println("Notes:")
	fmt.Println("  - Creates .github/workflows if it does not exist.")
}

func requireCustomAgent(root, name string) error {
	agentName := strings.TrimSpace(name)
	if agentName == "" {
		return nil
	}
	if tender.IsSystemAgent(agentName) {
		return fmt.Errorf("agent %q is reserved; choose a custom agent", agentName)
	}

	agents, err := tender.DiscoverPrimaryAgents(root)
	if err != nil {
		return fmt.Errorf("unable to discover custom agents: %w", err)
	}
	for _, agent := range agents {
		if strings.EqualFold(strings.TrimSpace(agent), agentName) {
			return nil
		}
	}
	return fmt.Errorf("agent %q is not a discovered custom primary agent", agentName)
}
