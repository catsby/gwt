package cmd

import (
	"fmt"
	"os"

	"github.com/catsby/gws/tui"
)

func Execute() {
	args := os.Args[1:]

	if len(args) == 0 {
		if os.Getenv("GWS_WRAPPED") == "" {
			fmt.Fprintln(os.Stderr, `Tip: eval "$(gws init zsh)" to enable directory switching`)
		}
		path, err := tui.RunPicker()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if path == "" {
			os.Exit(1)
		}
		fmt.Println(path)
		os.Exit(0)
	}

	switch args[0] {
	case "add":
		runAdd(args[1:])
	case "rm":
		runRm()
	case "list":
		runList()
	case "init":
		runInit(args[1:])
	default:
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: gws [command]

Commands:
  (none)    Launch TUI worktree picker
  add       Create a new worktree
  rm        Remove a worktree (TUI)
  list      List worktrees
  init      Print shell wrapper function`)
}
