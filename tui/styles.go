package tui

import "github.com/charmbracelet/lipgloss"

var (
	WorktreeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	RemoteBranchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	SelectedStyle     = lipgloss.NewStyle().Bold(true)
	PromptStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	ErrorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	SpinnerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)
