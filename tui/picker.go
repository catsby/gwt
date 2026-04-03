package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"

	"github.com/catsby/gwt/gitops"
)

const defaultMaxVisible = 10
const remoteFilterMinChars = 3

type state int

const (
	stateBrowsing state = iota
	stateConfirming
	stateCreating
)

// listItem represents a unified entry in the picker list.
type listItem struct {
	display     string // display text
	path        string // worktree path (empty for remote branches)
	branch      string // branch name
	isWorktree  bool
	isSeparator bool
}

type pickerModel struct {
	items          []listItem
	remoteBranches []listItem
	filtered       []listItem
	cursor         int
	offset         int
	maxVisible     int
	filter         textinput.Model
	spinner        spinner.Model
	state          state
	selected       listItem
	result         string
	err            error
	quitting       bool
	createErr      string
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

	remotes, err := gitops.ListRemoteBranches()
	if err != nil {
		return pickerModel{}, fmt.Errorf("listing remote branches: %w", err)
	}
	var remoteBranches []listItem
	for _, rb := range remotes {
		remoteBranches = append(remoteBranches, listItem{
			display: rb,
			branch:  rb,
		})
	}

	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	m := pickerModel{
		items:          items,
		remoteBranches: remoteBranches,
		filtered:       items,
		filter:         ti,
		spinner:        s,
		state:          stateBrowsing,
		maxVisible:     defaultMaxVisible,
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
	case tea.WindowSizeMsg:
		if avail := msg.Height - 5; avail > 0 && avail < defaultMaxVisible {
			m.maxVisible = avail
		} else {
			m.maxVisible = defaultMaxVisible
		}
		if m.offset+m.maxVisible > len(m.filtered) {
			m.offset = max(0, len(m.filtered)-m.maxVisible)
		}
		if m.cursor >= m.offset+m.maxVisible {
			m.offset = m.cursor - m.maxVisible + 1
		}
		return m, nil
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
			if m.selected.isSeparator {
				return m, nil
			}
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
				if m.cursor >= 0 && m.filtered[m.cursor].isSeparator {
					m.cursor--
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
			return m, nil
		case "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				if m.cursor < len(m.filtered) && m.filtered[m.cursor].isSeparator {
					m.cursor++
				}
				if m.cursor >= len(m.filtered) {
					m.cursor = len(m.filtered) - 1
				}
				if m.cursor >= m.offset+m.maxVisible {
					m.offset = m.cursor - m.maxVisible + 1
				}
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
		m.offset = 0
		return
	}

	// Fuzzy match against local worktrees.
	wtSources := make([]string, len(m.items))
	for i, item := range m.items {
		wtSources[i] = item.display
	}
	wtMatches := fuzzy.Find(query, wtSources)

	var filtered []listItem
	for _, match := range wtMatches {
		filtered = append(filtered, m.items[match.Index])
	}

	// Include remote branches when query is long enough.
	if len(query) >= remoteFilterMinChars && len(m.remoteBranches) > 0 {
		rbSources := make([]string, len(m.remoteBranches))
		for i, item := range m.remoteBranches {
			rbSources[i] = item.display
		}
		rbMatches := fuzzy.Find(query, rbSources)
		if len(rbMatches) > 0 {
			filtered = append(filtered, listItem{isSeparator: true, display: "remote branches"})
			for _, match := range rbMatches {
				filtered = append(filtered, m.remoteBranches[match.Index])
			}
		}
	}

	m.filtered = filtered
	m.cursor = 0
	m.offset = 0
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

		var listContent strings.Builder
		if len(m.filtered) == 0 {
			listContent.WriteString("No matches.\n")
		} else {
			end := m.offset + m.maxVisible
			if end > len(m.filtered) {
				end = len(m.filtered)
			}

			if m.offset > 0 {
				listContent.WriteString(ScrollHintStyle.Render(fmt.Sprintf("↑ %d more", m.offset)) + "\n")
			}

			for i := m.offset; i < end; i++ {
				item := m.filtered[i]

				if item.isSeparator {
					listContent.WriteString(SeparatorStyle.Render("── "+item.display+" ──") + "\n")
					continue
				}

				var cursor string
				if i == m.cursor {
					cursor = CursorStyle.Render(">") + " "
				} else {
					cursor = "  "
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
					line = cursor + SelectedStyle.Render(prefix) + text
				}
				listContent.WriteString(line + "\n")
			}

			if remaining := len(m.filtered) - end; remaining > 0 {
				listContent.WriteString(ScrollHintStyle.Render(fmt.Sprintf("↓ %d more", remaining)) + "\n")
			}
		}
		b.WriteString(ListContainerStyle.Render(listContent.String()))

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

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("TUI error: %w", err)
	}

	result := finalModel.(pickerModel).result
	return result, nil
}
