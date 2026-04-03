package cmd

import (
	"fmt"
	"os"
)

const shellWrapperZsh = `gwt() {
  local target
  gwt_WRAPPED=1 target=$(command gwt "$@")
  local exit_code=$?
  if [ $exit_code -eq 0 ] && [ -n "$target" ] && [ -d "$target" ]; then
    cd "$target"
  else
    return $exit_code
  fi
}
`

const shellWrapperBash = shellWrapperZsh

const shellWrapperFish = `function gwt
  set -l target (gwt_WRAPPED=1 command gwt $argv)
  set -l exit_code $status
  if test $exit_code -eq 0; and test -n "$target"; and test -d "$target"
    cd $target
  else
    return $exit_code
  end
end
`

func runInit(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: gwt init <shell>  (zsh, bash, fish)")
		os.Exit(2)
	}

	switch args[0] {
	case "zsh":
		fmt.Print(shellWrapperZsh)
	case "bash":
		fmt.Print(shellWrapperBash)
	case "fish":
		fmt.Print(shellWrapperFish)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported shell %q (supported: zsh, bash, fish)\n", args[0])
		os.Exit(2)
	}
}
