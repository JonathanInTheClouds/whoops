package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/JonathanInTheClouds/whoops/internal/git"
	"github.com/JonathanInTheClouds/whoops/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var Version = "dev"

func main() {
	history := flag.Bool("history", false, "open interactive TUI to pick an action to undo")
	dryRun  := flag.Bool("dry-run", false, "show what would be undone without doing it")
	version := flag.Bool("version", false, "print version and exit")
	debug   := flag.Bool("debug", false, "print raw reflog entries and exit")
	flag.Parse()

	if *version {
		fmt.Println("whoops version", Version)
		os.Exit(0)
	}

	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "✗ Not inside a git repository.")
		os.Exit(1)
	}

	if *debug {
		actions, err := git.ReadReflog(20)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%-14s  %-40s  %s\n", "TYPE", "DESCRIPTION", "DATE")
		fmt.Println(strings.Repeat("-", 72))
		for _, a := range actions {
			fmt.Printf("%-14s  %-40s  %s\n",
				a.Type,
				truncate(a.Description, 40),
				git.RelativeTime(a.Date),
			)
		}
		os.Exit(0)
	}

	if *history {
		// Interactive TUI mode
		m := ui.NewModel(*dryRun)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Simple mode — undo the last action immediately
	actions, err := git.ReadReflog(1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ %v\n", err)
		os.Exit(1)
	}
	if len(actions) == 0 {
		fmt.Fprintln(os.Stderr, "✗ No git actions found to undo.")
		os.Exit(1)
	}

	last := actions[0]

	if *dryRun {
		result, err := git.Undo(last, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Cannot undo: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Dry run — would undo: %s %q\n", last.Type, last.Description)
		fmt.Printf("Command: %s\n", result.Command)
		return
	}

	// Confirm before undoing
	fmt.Printf("Undo %s %q? [y/N] ", last.Type, truncate(last.Description, 50))
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Aborted.")
		os.Exit(0)
	}

	result, err := git.Undo(last, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Undid: %s %q\n", result.Action.Type, result.Action.Description)
	fmt.Printf("  Ran: %s\n", result.Command)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}