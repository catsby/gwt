package cmd

import (
	"fmt"
	"os"

	"github.com/catsby/gwt/gitops"
)

func runList() {
	worktrees, err := gitops.ListWorktrees()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	for _, wt := range worktrees {
		fmt.Printf("%s\t%s\n", wt.Path, wt.Branch)
	}
}
