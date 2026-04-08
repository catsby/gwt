package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Worktree represents a git worktree.
type Worktree struct {
	Path   string // absolute path
	Branch string // branch name (e.g. "main", "feature-x")
	IsRoot bool   // true if this is the main working tree
}

// GitRoot returns the root of the main working tree by running
// git rev-parse --path-format=absolute --git-common-dir and stripping /.git.
// Requires Git 2.31+.
func GitRoot() (string, error) {
	if err := checkGitVersion(); err != nil {
		return "", err
	}
	out, err := runGit("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("failed to determine git root: %w", err)
	}
	root := strings.TrimSpace(out)
	root = strings.TrimSuffix(root, "/.git")
	return root, nil
}

// ListWorktrees returns all worktrees sorted with root first, then alphabetically by branch.
func ListWorktrees() ([]Worktree, error) {
	out, err := runGit("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}
	worktrees := parseWorktreeOutput(out)
	if len(worktrees) > 0 {
		worktrees[0].IsRoot = true
	}

	// Sort: root first, then alphabetically by branch name.
	sort.SliceStable(worktrees, func(i, j int) bool {
		if worktrees[i].IsRoot {
			return true
		}
		if worktrees[j].IsRoot {
			return false
		}
		return worktrees[i].Branch < worktrees[j].Branch
	})

	return worktrees, nil
}

// ListRemoteBranches returns remote branches that do not already have a local worktree.
// Filters out origin/HEAD and stale refs. Returns names like "origin/feature-x".
func ListRemoteBranches() ([]string, error) {
	out, err := runGit("branch", "-r")
	if err != nil {
		return nil, fmt.Errorf("failed to list remote branches: %w", err)
	}

	worktrees, err := ListWorktrees()
	if err != nil {
		return nil, err
	}

	// Build set of branches that already have worktrees.
	wtBranches := make(map[string]bool)
	for _, wt := range worktrees {
		wtBranches[wt.Branch] = true
	}

	var branches []string
	for _, line := range strings.Split(out, "\n") {
		branch := strings.TrimSpace(line)
		if branch == "" {
			continue
		}
		// Filter out HEAD pointer and stale refs.
		if strings.Contains(branch, "->") {
			continue
		}
		if branch == "origin/HEAD" {
			continue
		}
		// Check if a worktree already tracks this branch.
		// Remote branch "origin/foo" corresponds to local branch "foo".
		localName := branch
		if idx := strings.Index(branch, "/"); idx >= 0 {
			localName = branch[idx+1:]
		}
		if wtBranches[localName] {
			continue
		}
		branches = append(branches, branch)
	}
	sort.Strings(branches)
	return branches, nil
}

// sanitizeName replaces slashes with dashes so branch names like "feat/http2"
// become "feat-http2" on disk instead of creating nested directories.
func sanitizeName(name string) string {
	return strings.ReplaceAll(name, "/", "-")
}

// CreateWorktree creates a new worktree under the worktree directory.
// If trackBranch is non-empty, the worktree tracks that remote branch.
// If trackBranch is empty and origin/<name> exists, it auto-tracks that remote branch.
// Returns the absolute path of the new worktree.
func CreateWorktree(name string, trackBranch string) (string, error) {
	dir, err := WorktreeDir()
	if err != nil {
		return "", err
	}
	safeName := sanitizeName(name)
	wtPath := filepath.Join(dir, safeName)

	var args []string
	if trackBranch != "" {
		args = []string{"worktree", "add", wtPath, "-b", safeName, "--track", trackBranch}
	} else {
		// Auto-detect: if origin/<name> exists, track it.
		_, detectErr := runGit("rev-parse", "--verify", "origin/"+name)
		if detectErr == nil {
			args = []string{"worktree", "add", wtPath, "-b", safeName, "--track", "origin/" + name}
		} else {
			args = []string{"worktree", "add", wtPath}
		}
	}

	_, err = runGit(args...)
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %w", err)
	}
	return wtPath, nil
}

// RemoveWorktree removes the worktree at the given path.
func RemoveWorktree(path string) error {
	_, err := runGit("worktree", "remove", path)
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}
	return nil
}

// WorktreeDir returns the directory where worktrees are created.
// Uses gwt_WORKTREE_DIR env var if set, otherwise defaults to <git-root>/.claude/worktrees/.
// Creates the directory if it does not exist.
func WorktreeDir() (string, error) {
	dir := os.Getenv("gwt_WORKTREE_DIR")
	if dir == "" {
		root, err := GitRoot()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(root, ".claude", "worktrees")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create worktree directory %s: %w", dir, err)
	}
	return dir, nil
}

// parseWorktreeOutput parses the porcelain output of git worktree list.
func parseWorktreeOutput(output string) []Worktree {
	var worktrees []Worktree
	var current Worktree
	inEntry := false

	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			if inEntry {
				worktrees = append(worktrees, current)
			}
			current = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
			inEntry = true
		} else if strings.HasPrefix(line, "branch refs/heads/") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if line == "" && inEntry {
			worktrees = append(worktrees, current)
			current = Worktree{}
			inEntry = false
		}
	}
	if inEntry {
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// checkGitVersion ensures git is at least version 2.31.
func checkGitVersion() error {
	out, err := runGit("version")
	if err != nil {
		return fmt.Errorf("git not found: %w", err)
	}
	// Output: "git version 2.39.0" or similar.
	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) < 3 {
		return fmt.Errorf("unexpected git version output: %s", out)
	}
	version := parts[2]
	// Parse major.minor.
	vParts := strings.SplitN(version, ".", 3)
	if len(vParts) < 2 {
		return fmt.Errorf("cannot parse git version: %s", version)
	}
	major, err := strconv.Atoi(vParts[0])
	if err != nil {
		return fmt.Errorf("cannot parse git major version: %s", version)
	}
	minor, err := strconv.Atoi(vParts[1])
	if err != nil {
		return fmt.Errorf("cannot parse git minor version: %s", version)
	}
	if major < 2 || (major == 2 && minor < 31) {
		return fmt.Errorf("git version 2.31+ required, found %s", version)
	}
	return nil
}

// runGit executes a git command and returns its combined stdout. On error,
// it includes stderr in the error message.
func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}
