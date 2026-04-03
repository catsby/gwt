package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

func init() {
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stderr))
}

// Adaptive colors that work on both light and dark terminal backgrounds.
// Format: AdaptiveColor{Light: "<dark-bg-friendly>", Dark: "<light-bg-friendly>"}
var (
	colorGreen  = lipgloss.AdaptiveColor{Light: "#0a7e07", Dark: "#5af78e"}
	colorGray   = lipgloss.AdaptiveColor{Light: "#585858", Dark: "#8a8a8a"}
	colorCyan   = lipgloss.AdaptiveColor{Light: "#0b6e6e", Dark: "#9aedfe"}
	colorRed    = lipgloss.AdaptiveColor{Light: "#c41a16", Dark: "#ff5c57"}
	colorPurple = lipgloss.AdaptiveColor{Light: "#6c3ec2", Dark: "#b48ead"}
)

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple).
			PaddingBottom(1)

	ListContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorGray).
				PaddingLeft(1).
				PaddingRight(1)

	CursorStyle = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	WorktreeStyle     = lipgloss.NewStyle().Foreground(colorGreen)
	RemoteBranchStyle = lipgloss.NewStyle().Foreground(colorGray)
	SelectedStyle     = lipgloss.NewStyle().Bold(true)
	PromptStyle       = lipgloss.NewStyle().Foreground(colorCyan)
	ErrorStyle        = lipgloss.NewStyle().Foreground(colorRed)
	SpinnerStyle      = lipgloss.NewStyle().Foreground(colorCyan)
	ScrollHintStyle   = lipgloss.NewStyle().Foreground(colorGray).Faint(true)
	SeparatorStyle    = lipgloss.NewStyle().Foreground(colorGray).Faint(true)
)
