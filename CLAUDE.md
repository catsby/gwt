# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`gws` (Git Worktree Switcher) is a Go CLI tool for navigating and managing git worktrees via a fuzzy-filterable TUI. It outputs worktree paths to stdout so a shell wrapper can `cd` into them. See `GWS.md` for the full design spec.

## Build & Run

```bash
go build -o gws .
go run .
```

## Test

```bash
go test ./...              # all tests
go test ./... -run TestFoo # single test
go test ./... -v           # verbose
```

## Lint & Format

```bash
gofmt -w .
go vet ./...
```

## Key Design Decisions

- **TUI renders to stderr** (Bubbletea alternate screen) so stdout stays clean for path output consumed by the shell wrapper.
- **Shell out to git** for all git operations — no git library.
- **Worktrees created under `.claude/worktrees/`** relative to the git root.
- **Git root detection**: use `git rev-parse --path-format=absolute --git-common-dir` and strip `/.git`, not `--show-toplevel` (which returns the current worktree root, not the main root).
- **TUI framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea) with [Bubbles](https://github.com/charmbracelet/bubbles) components and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling.

## Commands

- `gws` — TUI picker showing worktrees and remote branches
- `gws add <name> [--track <remote-branch>]` — non-interactive worktree creation
- `gws rm` — TUI picker to remove a worktree
- `gws list` — non-interactive list of worktrees
- `gws init <shell>` — print shell wrapper function (zsh/bash/fish)
