package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"

	"github.com/catsby/gws/gitops"
)

type state int

const (
	stateBrowsing state = iota
	stateConfirming
	stateCreating
)

// listItem represents a unified entry in the picker list.
type listItem struct {
	display    string // display text
	path       string // worktree path (empty for remote branches)
	branch     string // branch name
	isWorktree bool
}

type pickerModel struct {
	items     []listItem
	filtered  []listItem
	cursor    int
	filter    textinput.Model
	spinner   spinner.Model
	state     state
	selected  listItem
	result    string
	err       error
	quitting  bool
	createErr string
}

type worktreeCreatedMsg struct {
	path string
	err  error
}

func newPickerModel() (pickerModel, error) {
	worktrees, err := gitops.ListWorktrees()
	if err != nil {
		return pickerModel{}, fmt.Errorf("listing worktrees: %w", err)
	}
	remotes, err := gitops.ListRemoteBranches()
	if err != nil {
		return pickerModel{}, fmt.Errorf("listing remote branches: %w", err)
	}

	var items []listItem
	for _, wt := range worktrees {
		display := wt.Branch
		if display == "" {
			display = wt.Path
		}
		items = append(items, listItem{
			display:    display,
			path:       wt.Path,
			branch:     wt.Branch,
			isWorktree: true,
		})
	}
	for _, rb := range remotes {
		items = append(items, listItem{
			display:    rb,
			branch:     rb,
			isWorktree: false,
		})
	}

	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	m := pickerModel{
		items:    items,
		filtered: items,
		filter:   ti,
		spinner:  s,
		state:    stateBrowsing,
	}
	return m, nil
}

func (m pickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateBrowsing:
		return m.updateBrowsing(msg)
	case stateConfirming:
		return m.updateConfirming(msg)
	case stateCreating:
		return m.updateCreating(msg)
	}
	return m, nil
}

func (m pickerModel) updateBrowsing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) == 0 {
				return m, nil
			}
			m.selected = m.filtered[m.cursor]
			if m.selected.isWorktree {
				m.result = m.selected.path
				m.quitting = true
				return m, tea.Quit
			}
			m.state = stateConfirming
			return m, nil
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	prevValue := m.filter.Value()
	m.filter, cmd = m.filter.Update(msg)
	if m.filter.Value() != prevValue {
		m.applyFilter()
	}
	return m, cmd
}

func (m pickerModel) updateConfirming(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.state = stateCreating
			m.createErr = ""
			return m, tea.Batch(m.spinner.Tick, m.createWorktreeCmd())
		case "n", "N", "esc":
			m.state = stateBrowsing
			return m, nil
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m pickerModel) updateCreating(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case worktreeCreatedMsg:
		if msg.err != nil {
			m.createErr = msg.err.Error()
			m.state = stateBrowsing
			return m, nil
		}
		m.result = msg.path
		m.quitting = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m pickerModel) createWorktreeCmd() tea.Cmd {
	branch := m.selected.branch
	return func() tea.Msg {
		// Derive worktree name from remote branch: "origin/feature-x" -> "feature-x"
		name := branch
		if idx := strings.Index(branch, "/"); idx >= 0 {
			name = branch[idx+1:]
		}
		path, err := gitops.CreateWorktree(name, branch)
		return worktreeCreatedMsg{path: path, err: err}
	}
}

func (m *pickerModel) applyFilter() {
	query := m.filter.Value()
	if query == "" {
		m.filtered = m.items
		m.cursor = 0
		return
	}

	// Build source list for fuzzy matching.
	sources := make([]string, len(m.items))
	for i, item := range m.items {
		sources[i] = item.display
	}
	matches := fuzzy.Find(query, sources)

	m.filtered = make([]listItem, len(matches))
	for i, match := range matches {
		m.filtered[i] = m.items[match.Index]
	}
	m.cursor = 0
}

func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	switch m.state {
	case stateBrowsing:
		b.WriteString(m.filter.View())
		b.WriteString("\n\n")

		if m.createErr != "" {
			b.WriteString(ErrorStyle.Render("Error: "+m.createErr) + "\n\n")
		}

		if len(m.filtered) == 0 {
			b.WriteString("  No matches.\n")
		} else {
			for i, item := range m.filtered {
				cursor := "  "
				if i == m.cursor {
					cursor = "> "
				}

				var prefix, text string
				if item.isWorktree {
					prefix = "● "
					text = WorktreeStyle.Render(item.display)
				} else {
					prefix = "○ "
					text = RemoteBranchStyle.Render(item.display)
				}

				line := cursor + prefix + text
				if i == m.cursor {
					line = SelectedStyle.Render(cursor+prefix) + text
				}
				b.WriteString(line + "\n")
			}
		}

	case stateConfirming:
		b.WriteString(PromptStyle.Render(
			fmt.Sprintf("Create worktree for %s? [y/N] ", m.selected.branch),
		))

	case stateCreating:
		b.WriteString(m.spinner.View() + " Creating worktree...")
	}

	return b.String()
}

// RunPicker launches the main TUI picker. Returns the selected/created worktree path,
// or empty string if cancelled.
func RunPicker() (string, error) {
	m, err := newPickerModel()
	if err != nil {
		return "", err
	}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("TUI error: %w", err)
	}

	result := finalModel.(pickerModel).result
	return result, nil
}
