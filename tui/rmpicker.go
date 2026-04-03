package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"

	"github.com/catsby/gwt/gitops"
)

type rmState int

const (
	rmBrowsing rmState = iota
	rmConfirming
	rmRemoving
)

type rmPickerModel struct {
	items     []listItem
	filtered  []listItem
	cursor    int
	filter    textinput.Model
	state     rmState
	selected  listItem
	err       error
	quitting  bool
	removed   bool
	removeErr string
}

type worktreeRemovedMsg struct {
	err error
}

func newRmPickerModel() (rmPickerModel, error) {
	worktrees, err := gitops.ListWorktrees()
	if err != nil {
		return rmPickerModel{}, fmt.Errorf("listing worktrees: %w", err)
	}

	// Determine current working directory to exclude current worktree.
	cwd, err := os.Getwd()
	if err != nil {
		return rmPickerModel{}, fmt.Errorf("getting working directory: %w", err)
	}

	var items []listItem
	for _, wt := range worktrees {
		// Exclude root and current worktree.
		if wt.IsRoot {
			continue
		}
		if wt.Path == cwd {
			continue
		}
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

	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.Focus()

	return rmPickerModel{
		items:    items,
		filtered: items,
		filter:   ti,
		state:    rmBrowsing,
	}, nil
}

func (m rmPickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m rmPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case rmBrowsing:
		return m.updateBrowsing(msg)
	case rmConfirming:
		return m.updateConfirming(msg)
	case rmRemoving:
		return m.updateRemoving(msg)
	}
	return m, nil
}

func (m rmPickerModel) updateBrowsing(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.state = rmConfirming
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

func (m rmPickerModel) updateConfirming(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.state = rmRemoving
			return m, m.removeWorktreeCmd()
		case "n", "N", "esc":
			m.state = rmBrowsing
			return m, nil
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m rmPickerModel) updateRemoving(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case worktreeRemovedMsg:
		if msg.err != nil {
			m.removeErr = msg.err.Error()
			m.err = msg.err
			m.quitting = true
			return m, tea.Quit
		}
		m.removed = true
		m.quitting = true
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m rmPickerModel) removeWorktreeCmd() tea.Cmd {
	path := m.selected.path
	return func() tea.Msg {
		err := gitops.RemoveWorktree(path)
		return worktreeRemovedMsg{err: err}
	}
}

func (m *rmPickerModel) applyFilter() {
	query := m.filter.Value()
	if query == "" {
		m.filtered = m.items
		m.cursor = 0
		return
	}

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

func (m rmPickerModel) View() string {
	if m.quitting {
		if m.removeErr != "" {
			return ErrorStyle.Render("Error: "+m.removeErr) + "\n"
		}
		return ""
	}

	var b strings.Builder

	switch m.state {
	case rmBrowsing:
		b.WriteString(m.filter.View())
		b.WriteString("\n\n")

		if len(m.filtered) == 0 {
			b.WriteString("  No worktrees to remove.\n")
		} else {
			for i, item := range m.filtered {
				cursor := "  "
				if i == m.cursor {
					cursor = "> "
				}
				text := WorktreeStyle.Render("● " + item.display)
				line := cursor + text
				if i == m.cursor {
					line = SelectedStyle.Render(cursor+"● ") + WorktreeStyle.Render(item.display)
				}
				b.WriteString(line + "\n")
			}
		}

	case rmConfirming:
		b.WriteString(PromptStyle.Render(
			fmt.Sprintf("Remove worktree %s (%s)? [y/N] ", m.selected.branch, m.selected.path),
		))

	case rmRemoving:
		b.WriteString("Removing worktree...")
	}

	return b.String()
}

// RunRemovePicker launches the rm TUI picker. Returns nil on success, error on failure.
func RunRemovePicker() error {
	m, err := newRmPickerModel()
	if err != nil {
		return err
	}

	if len(m.items) == 0 {
		fmt.Fprintln(os.Stderr, "No worktrees to remove.")
		return nil
	}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	fm := finalModel.(rmPickerModel)
	if fm.err != nil {
		return fm.err
	}
	return nil
}
