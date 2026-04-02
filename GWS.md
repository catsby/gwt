# gws — Git Worktree Switcher

CLI tool for navigating and managing git worktrees. Built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Core Concept

`gws` launches a fuzzy-filterable TUI listing existing worktrees and remote branches. Selecting a worktree outputs its path to stdout. Selecting a remote branch confirms, creates a worktree tracking it, then outputs the new path. A shell wrapper function handles the actual `cd`.

Worktree location defaults to `.claude/worktrees/` relative to the git root. Configurable via `GWS_WORKTREE_DIR` env var (will move to `gws.toml` in a future version).

## Commands

### `gws` (default, no subcommand)

Launch TUI with a unified list:

- **Worktrees** — existing worktrees including the root. Visually distinct (e.g. green, `●` prefix). Selecting one prints its absolute path to stdout and exits 0.
- **Remote branches** (no local worktree) — shown below worktrees with different styling (e.g. dim/grey, `○` prefix). Selecting one shows a confirmation prompt before creating the worktree. On confirm, show a spinner during creation, then print the new path.
- Fuzzy text filter narrows both sections.
- Filter out `origin/HEAD` and stale remote refs.
- `Enter` to select, `Esc`/`Ctrl+C` to cancel (exit 1, no output).

TUI renders to stderr (Bubbletea alternate screen) so stdout stays clean for the path output.

### `gws add <name> [--track <remote-branch>]`

Non-interactive worktree creation.

- `gws add my-feature` → `git worktree add .claude/worktrees/my-feature`
- `gws add review/fix-bug --track origin/fix-bug` → `git worktree add .claude/worktrees/review/fix-bug --track origin/fix-bug`
- Prints the new worktree path to stdout on success.

### `gws rm`

TUI picker listing existing worktrees (excludes root and the current worktree). On selection, show full path and branch name in the confirmation prompt. Runs `git worktree remove <path>` on confirm. If git fails, show its stderr output and exit 2.

### `gws init <shell>`

Prints the shell wrapper function to stdout. Supported: `zsh`, `bash`, `fish`.

The shell wrapper sets `GWS_WRAPPED=1` so the binary can detect it. If the binary runs without `GWS_WRAPPED`, print a hint to stderr on first run suggesting `gws init`.

Example output for zsh:

```zsh
gws() {
  local target
  GWS_WRAPPED=1 target=$(command gws "$@")
  if [ $? -eq 0 ] && [ -n "$target" ] && [ -d "$target" ]; then
    cd "$target"
  fi
}
```

Usage: `gws init zsh >> ~/.zshrc`

### `gws list`

Non-interactive. Tab-delimited, two columns: `<absolute-path>\t<branch-name>`. One line per worktree. Useful for scripting with `cut`/`awk`.

## Implementation Notes

- Use `git worktree list --porcelain` for parsing worktrees.
- Use `git branch -r` for remote branches. Filter out branches that already have a worktree, `origin/HEAD`, and stale refs.
- Git root: `git rev-parse --path-format=absolute --git-common-dir` → strip trailing `/.git` to get the main working tree root. This works correctly from inside any worktree, unlike `--show-toplevel` which returns the current worktree's root. Requires Git 2.31+ — check and fail with a clear message if older.
- Shell out to git for all git operations — don't use a git library.
- Use [Bubbles](https://github.com/charmbracelet/bubbles) components: `textinput` for filter, `list` or `viewport` for the worktree/branch display, `spinner` for worktree creation.
- Use [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling.

## Exit Codes

- `0` — success, path printed to stdout
- `1` — user cancelled (no stdout output)
- `2` — error

## Out of Scope (v0.2+)

- Post-create symlink/copy setup (`.worktreeinclude`, Claude Code `worktree.symlinkDirectories`)
- `gws.toml` config file (worktree path, copy/link rules)
- Worktree status indicators (dirty, ahead/behind)