package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tender/internal/tender"
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

	case "ls":
		if err := tender.PrintList(root, os.Stdout); err != nil {
			fail(err)
		}

	case "rm":
		fs := flag.NewFlagSet("rm", flag.ExitOnError)
		yes := fs.Bool("yes", false, "delete without confirmation")
		_ = fs.Parse(os.Args[2:])
		args := fs.Args()
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "usage: tender rm [--yes] <name>")
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
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		prompt := fs.String("prompt", "", "optional prompt override for this dispatch")
		_ = fs.Parse(os.Args[2:])
		args := fs.Args()
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "usage: tender run [--prompt \"...\"] <name>")
			os.Exit(2)
		}
		name := args[0]
		if err := tender.DispatchTenderNow(root, name, *prompt, os.Stdout, os.Stderr); err != nil {
			fail(err)
		}
		fmt.Printf("triggered %s\n", name)

	case "help", "-h", "--help":
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
	fmt.Println("  tender             Launch interactive TUI")
	fmt.Println("  tender init        Ensure .github/workflows exists")
	fmt.Println("  tender ls          List managed tender workflows")
	fmt.Println("  tender run <name>  Trigger an on-demand tender now via GitHub CLI")
	fmt.Println("  tender rm <name>   Remove a tender workflow")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
