# gws — Git Worktree Switcher

CLI tool for navigating and managing git worktrees. Built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Core Concept

`gws` launches a fuzzy-filterable TUI listing existing worktrees and remote branches. Selecting a worktree outputs its path to stdout. Selecting a remote branch creates a worktree tracking it, then outputs the new path. A shell wrapper function handles the actual `cd`.

Worktrees are created under `.claude/worktrees/` relative to the git root.

## Commands

### `gws` (default, no subcommand)

Launch TUI with a unified list:

- **Worktrees** — existing worktrees including the root. Selecting one prints its absolute path to stdout and exits 0.
- **Remote branches** (no local worktree) — shown below worktrees with a visual distinction. Selecting one creates a worktree under `.claude/worktrees/<branch-name>`, then prints the new path.
- Fuzzy text filter narrows both sections.
- `Enter` to select, `Esc`/`Ctrl+C` to cancel (exit 1, no output).

TUI renders to stderr (Bubbletea alternate screen) so stdout stays clean for the path output.

### `gws add <name> [--track <remote-branch>]`

Non-interactive worktree creation.

- `gws add my-feature` → `git worktree add .claude/worktrees/my-feature`
- `gws add review/fix-bug --track origin/fix-bug` → `git worktree add .claude/worktrees/review/fix-bug --track origin/fix-bug`
- Prints the new worktree path to stdout on success.

### `gws rm`

TUI picker listing existing worktrees (not root). Select one, confirm, then runs `git worktree remove <path>`.

### `gws init <shell>`

Prints the shell wrapper function to stdout. Supported: `zsh`, `bash`, `fish`.

Example output for zsh:

```zsh
gws() {
  local target
  target=$(command gws "$@")
  if [ $? -eq 0 ] && [ -n "$target" ] && [ -d "$target" ]; then
    cd "$target"
  fi
}
```

Usage: `gws init zsh >> ~/.zshrc`

### `gws list`

Non-interactive. Prints worktree paths and branch names to stdout. Useful for scripting.

## Implementation Notes

- Use `git worktree list --porcelain` for parsing worktrees.
- Use `git branch -r` for remote branches. Filter out branches that already have a worktree.
- Git root: `git rev-parse --path-format=absolute --git-common-dir` → strip trailing `/.git` to get the main working tree root. This works correctly from inside any worktree, unlike `--show-toplevel` which returns the current worktree's root.
- Shell out to git for all git operations — don't use a git library.
- Use [Bubbles](https://github.com/charmbracelet/bubbles) components: `textinput` for filter, `list` or `viewport` for the worktree/branch display.
- Use [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling.

## Exit Codes

- `0` — success, path printed to stdout
- `1` — user cancelled or error (no stdout output)

## Out of Scope (v0.2+)

- Post-create symlink setup (`.worktreeinclude`, Claude Code `worktree.symlinkDirectories`)
- Config file
- Worktree status indicators (dirty, ahead/behind)