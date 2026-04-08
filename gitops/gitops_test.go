package gitops

import (
	"os"
	"path/filepath"
	"testing"
)

// initTestRepo creates a temporary git repo with an initial commit and returns its path.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		runCmd(t, c...)
	}

	// Create a file and commit so HEAD exists.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCmd(t, "git", "-C", dir, "add", ".")
	runCmd(t, "git", "-C", dir, "commit", "-m", "initial commit")

	return dir
}

func runCmd(t *testing.T, args ...string) {
	t.Helper()
	cmd := args[0]
	out, err := runGit(args[1:]...)
	if cmd != "git" {
		t.Fatalf("runCmd only supports git, got %s", cmd)
	}
	_ = out
	if err != nil {
		t.Fatalf("command failed: %s %v: %v", cmd, args[1:], err)
	}
}

func TestGitRoot(t *testing.T) {
	dir := initTestRepo(t)

	// Run GitRoot from inside the repo.
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	root, err := GitRoot()
	if err != nil {
		t.Fatalf("GitRoot() error: %v", err)
	}

	// Resolve symlinks for comparison (macOS /tmp is a symlink).
	expected, _ := filepath.EvalSymlinks(dir)
	got, _ := filepath.EvalSymlinks(root)

	if got != expected {
		t.Errorf("GitRoot() = %q, want %q", got, expected)
	}
}

func TestListWorktrees(t *testing.T) {
	dir := initTestRepo(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	wts, err := ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}

	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(wts))
	}

	if !wts[0].IsRoot {
		t.Error("first worktree should be root")
	}

	resolvedDir, _ := filepath.EvalSymlinks(dir)
	resolvedPath, _ := filepath.EvalSymlinks(wts[0].Path)
	if resolvedPath != resolvedDir {
		t.Errorf("worktree path = %q, want %q", resolvedPath, resolvedDir)
	}
}

func TestListWorktrees_Multiple(t *testing.T) {
	dir := initTestRepo(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Create a second worktree.
	wtPath := filepath.Join(dir, "wt-feature")
	runCmd(t, "git", "-C", dir, "worktree", "add", wtPath, "-b", "feature-branch")

	wts, err := ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}

	if len(wts) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(wts))
	}

	if !wts[0].IsRoot {
		t.Error("first worktree should be root")
	}
	if wts[1].Branch != "feature-branch" {
		t.Errorf("second worktree branch = %q, want %q", wts[1].Branch, "feature-branch")
	}
}

func TestParseWorktreeOutput(t *testing.T) {
	input := `worktree /home/user/project
HEAD abc123
branch refs/heads/main
bare

worktree /home/user/project-feature
HEAD def456
branch refs/heads/feature-x

`
	wts := parseWorktreeOutput(input)
	if len(wts) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(wts))
	}

	if wts[0].Path != "/home/user/project" {
		t.Errorf("wt[0].Path = %q", wts[0].Path)
	}
	if wts[0].Branch != "main" {
		t.Errorf("wt[0].Branch = %q", wts[0].Branch)
	}
	if wts[1].Path != "/home/user/project-feature" {
		t.Errorf("wt[1].Path = %q", wts[1].Path)
	}
	if wts[1].Branch != "feature-x" {
		t.Errorf("wt[1].Branch = %q", wts[1].Branch)
	}
}

func TestWorktreeDir_Default(t *testing.T) {
	dir := initTestRepo(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Ensure env var is not set.
	os.Unsetenv("gwt_WORKTREE_DIR")

	wtDir, err := WorktreeDir()
	if err != nil {
		t.Fatalf("WorktreeDir() error: %v", err)
	}

	resolvedDir, _ := filepath.EvalSymlinks(dir)
	expected := filepath.Join(resolvedDir, ".claude", "worktrees")
	resolvedWtDir, _ := filepath.EvalSymlinks(wtDir)

	if resolvedWtDir != expected {
		t.Errorf("WorktreeDir() = %q, want %q", resolvedWtDir, expected)
	}

	// Verify directory was created.
	info, err := os.Stat(wtDir)
	if err != nil {
		t.Fatalf("WorktreeDir directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("WorktreeDir path is not a directory")
	}
}

func TestWorktreeDir_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	customDir := filepath.Join(dir, "custom-worktrees")

	t.Setenv("gwt_WORKTREE_DIR", customDir)

	wtDir, err := WorktreeDir()
	if err != nil {
		t.Fatalf("WorktreeDir() error: %v", err)
	}

	if wtDir != customDir {
		t.Errorf("WorktreeDir() = %q, want %q", wtDir, customDir)
	}

	info, err := os.Stat(customDir)
	if err != nil {
		t.Fatalf("custom worktree dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("custom worktree dir is not a directory")
	}
}

func TestCreateAndRemoveWorktree(t *testing.T) {
	dir := initTestRepo(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	t.Setenv("gwt_WORKTREE_DIR", filepath.Join(dir, ".claude", "worktrees"))

	wtPath, err := CreateWorktree("test-feature", "")
	if err != nil {
		t.Fatalf("CreateWorktree() error: %v", err)
	}

	// Verify worktree exists.
	info, err := os.Stat(wtPath)
	if err != nil {
		t.Fatalf("worktree path does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("worktree path is not a directory")
	}

	// Verify it shows up in list.
	wts, err := ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error: %v", err)
	}
	if len(wts) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(wts))
	}

	// Remove the worktree.
	err = RemoveWorktree(wtPath)
	if err != nil {
		t.Fatalf("RemoveWorktree() error: %v", err)
	}

	wts, err = ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() after remove error: %v", err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree after removal, got %d", len(wts))
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-feature", "my-feature"},
		{"feat/http2", "feat-http2"},
		{"org/team/feature", "org-team-feature"},
		{"no-slashes-here", "no-slashes-here"},
		{"a/b/c/d", "a-b-c-d"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreateWorktree_SlashedName(t *testing.T) {
	dir := initTestRepo(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	t.Setenv("gwt_WORKTREE_DIR", filepath.Join(dir, ".claude", "worktrees"))

	wtPath, err := CreateWorktree("feat/thing", "")
	if err != nil {
		t.Fatalf("CreateWorktree() error: %v", err)
	}

	// Path should use dashes, not nested directories.
	expectedPath := filepath.Join(dir, ".claude", "worktrees", "feat-thing")
	if wtPath != expectedPath {
		t.Errorf("worktree path = %q, want %q", wtPath, expectedPath)
	}

	info, err := os.Stat(wtPath)
	if err != nil {
		t.Fatalf("worktree path does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("worktree path is not a directory")
	}
}

func TestCheckGitVersion(t *testing.T) {
	err := checkGitVersion()
	if err != nil {
		t.Fatalf("checkGitVersion() error: %v (is git 2.31+ installed?)", err)
	}
}
