package cmd

import (
	"fmt"
	"os"

	"github.com/catsby/gwt/gitops"
)

func parseAddArgs(args []string) (name, trackBranch string, err error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("missing worktree name")
	}

	name = args[0]

	for i := 1; i < len(args); i++ {
		if args[i] == "--track" {
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--track requires a branch name")
			}
			trackBranch = args[i+1]
			i++
		} else {
			return "", "", fmt.Errorf("unknown flag %q", args[i])
		}
	}

	return name, trackBranch, nil
}

func runAdd(args []string) {
	name, trackBranch, err := parseAddArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		fmt.Fprintln(os.Stderr, "Usage: gwt add <name> [--track <remote-branch>]")
		os.Exit(2)
	}

	path, err := gitops.CreateWorktree(name, trackBranch)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	fmt.Println(path)
}
