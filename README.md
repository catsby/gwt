# gwt — Git Worktree Switcher

A CLI tool for navigating and managing git worktrees via a fuzzy-filterable TUI. Built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Install

```bash
go install github.com/catsby/gwt@latest
```

## Setup

Add the shell wrapper to your shell config so `gwt` can `cd` into worktrees:

```bash
# zsh
gwt init zsh >> ~/.zshrc

# bash
gwt init bash >> ~/.bashrc

# fish
gwt init fish >> ~/.config/fish/config.fish
```

## Usage

```
gwt              # TUI picker for worktrees and remote branches
gwt add <name>   # Create a worktree
gwt rm           # TUI picker to remove a worktree
gwt list         # List worktrees (tab-delimited, scriptable)
```

## License

MIT
