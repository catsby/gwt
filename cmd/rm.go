package cmd

import (
	"fmt"
	"os"

	"github.com/catsby/gwt/tui"
)

func runRm() {
	err := tui.RunRemovePicker()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
