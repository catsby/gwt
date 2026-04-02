package cmd

import (
	"strings"
	"testing"
)

func TestShellWrapperZsh(t *testing.T) {
	wrapper := shellWrapperZsh
	if !strings.Contains(wrapper, "GWS_WRAPPED=1") {
		t.Error("zsh wrapper should set GWS_WRAPPED=1")
	}
	if !strings.Contains(wrapper, "command gws") {
		t.Error("zsh wrapper should call 'command gws'")
	}
	if !strings.Contains(wrapper, `cd "$target"`) {
		t.Error("zsh wrapper should cd to target")
	}
	if !strings.Contains(wrapper, "gws()") {
		t.Error("zsh wrapper should define gws function")
	}
}

func TestShellWrapperBash(t *testing.T) {
	if shellWrapperBash != shellWrapperZsh {
		t.Error("bash wrapper should be the same as zsh wrapper")
	}
}

func TestShellWrapperFish(t *testing.T) {
	wrapper := shellWrapperFish
	if !strings.Contains(wrapper, "GWS_WRAPPED=1") {
		t.Error("fish wrapper should set GWS_WRAPPED=1")
	}
	if !strings.Contains(wrapper, "command gws") {
		t.Error("fish wrapper should call 'command gws'")
	}
	if !strings.Contains(wrapper, "cd $target") {
		t.Error("fish wrapper should cd to target")
	}
	if !strings.Contains(wrapper, "function gws") {
		t.Error("fish wrapper should define gws function")
	}
	if !strings.Contains(wrapper, "set -l exit_code $status") {
		t.Error("fish wrapper should capture exit code via $status")
	}
}
