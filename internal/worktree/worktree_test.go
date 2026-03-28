package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerateWorktreePath(t *testing.T) {
	tests := []struct {
		repoPath string
		runID    int
		expected string
	}{
		{"/home/user/myrepo", 1, "/home/user/.agentpair-worktree-1"},
		{"/home/user/myrepo", 5, "/home/user/.agentpair-worktree-5"},
		{"/projects/go/src/app", 100, "/projects/go/src/.agentpair-worktree-100"},
	}

	for _, tt := range tests {
		t.Run(tt.repoPath, func(t *testing.T) {
			got := GenerateWorktreePath(tt.repoPath, tt.runID)
			if got != tt.expected {
				t.Errorf("GenerateWorktreePath(%q, %d) = %q, want %q",
					tt.repoPath, tt.runID, got, tt.expected)
			}
		})
	}
}

func TestGenerateBranchName(t *testing.T) {
	tests := []struct {
		runID    int
		expected string
	}{
		{1, "agentpair/run-1"},
		{5, "agentpair/run-5"},
		{100, "agentpair/run-100"},
		{9999, "agentpair/run-9999"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := GenerateBranchName(tt.runID)
			if got != tt.expected {
				t.Errorf("GenerateBranchName(%d) = %q, want %q",
					tt.runID, got, tt.expected)
			}
		})
	}
}

func TestNewWorktree(t *testing.T) {
	wt := New("/repo/path", "/worktree/path", "my-branch")

	if wt.Path() != "/worktree/path" {
		t.Errorf("Path() = %s, want /worktree/path", wt.Path())
	}
	if wt.Branch() != "my-branch" {
		t.Errorf("Branch() = %s, want my-branch", wt.Branch())
	}
	if wt.WasCreated() {
		t.Error("WasCreated() should be false for new worktree")
	}
}

func TestWorktreeExists(t *testing.T) {
	// Test with non-existent path
	wt := New("/repo", "/nonexistent/path", "branch")
	if wt.Exists() {
		t.Error("Exists() should return false for non-existent path")
	}

	// Test with existing path
	dir := t.TempDir()
	wt2 := New("/repo", dir, "branch")
	if !wt2.Exists() {
		t.Error("Exists() should return true for existing path")
	}
}

func TestIsGitWorktree(t *testing.T) {
	dir := t.TempDir()

	// Not a worktree (no .git at all)
	if IsGitWorktree(dir) {
		t.Error("empty dir should not be detected as worktree")
	}

	// Has .git directory (regular repo, not worktree)
	gitDir := filepath.Join(dir, ".git")
	os.Mkdir(gitDir, 0755)
	if IsGitWorktree(dir) {
		t.Error("directory with .git dir should not be detected as worktree")
	}

	// Worktree has .git file (not directory)
	dir2 := t.TempDir()
	gitFile := filepath.Join(dir2, ".git")
	os.WriteFile(gitFile, []byte("gitdir: /path/to/repo/.git/worktrees/xyz"), 0644)
	if !IsGitWorktree(dir2) {
		t.Error("directory with .git file should be detected as worktree")
	}
}

func TestWorktreeInfo(t *testing.T) {
	info := WorktreeInfo{
		Path:   "/path/to/worktree",
		HEAD:   "abc123",
		Branch: "main",
	}

	if info.Path != "/path/to/worktree" {
		t.Error("Path not set correctly")
	}
	if info.HEAD != "abc123" {
		t.Error("HEAD not set correctly")
	}
	if info.Branch != "main" {
		t.Error("Branch not set correctly")
	}
	if info.Bare || info.Detached {
		t.Error("Bare and Detached should be false by default")
	}
}

// Helper to check if git is available
func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func skipIfNoGit(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
}

func TestIsGitRepo(t *testing.T) {
	skipIfNoGit(t)

	// Non-git directory
	dir := t.TempDir()
	if IsGitRepo(dir) {
		t.Error("empty directory should not be a git repo")
	}

	// Actual git repo (initialize one)
	repoDir := t.TempDir()
	cmd := exec.Command("git", "-C", repoDir, "init")
	if err := cmd.Run(); err != nil {
		t.Skipf("failed to init git repo: %v", err)
	}

	if !IsGitRepo(repoDir) {
		t.Error("initialized git repo should be detected")
	}
}

func TestGetRepoRoot(t *testing.T) {
	skipIfNoGit(t)

	// Non-git directory should error
	dir := t.TempDir()
	_, err := GetRepoRoot(dir)
	if err == nil {
		t.Error("GetRepoRoot should error for non-git directory")
	}

	// Git repo should return root
	repoDir := t.TempDir()
	cmd := exec.Command("git", "-C", repoDir, "init")
	if err := cmd.Run(); err != nil {
		t.Skipf("failed to init git repo: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(repoDir, "subdir")
	os.Mkdir(subDir, 0755)

	root, err := GetRepoRoot(subDir)
	if err != nil {
		t.Errorf("GetRepoRoot failed: %v", err)
	}

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	expectedDir, _ := filepath.EvalSymlinks(repoDir)
	actualDir, _ := filepath.EvalSymlinks(root)
	if actualDir != expectedDir {
		t.Errorf("GetRepoRoot = %s, want %s", root, repoDir)
	}
}

func TestListWorktrees(t *testing.T) {
	skipIfNoGit(t)

	// Initialize a git repo
	repoDir := t.TempDir()
	cmd := exec.Command("git", "-C", repoDir, "init")
	if err := cmd.Run(); err != nil {
		t.Skipf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "-C", repoDir, "config", "user.email", "test@example.com")
	cmd.Run()
	cmd = exec.Command("git", "-C", repoDir, "config", "user.name", "Test User")
	cmd.Run()

	// Create initial commit (required for worktrees)
	cmd = exec.Command("git", "-C", repoDir, "commit", "--allow-empty", "-m", "Initial commit")
	if err := cmd.Run(); err != nil {
		t.Skipf("failed to create initial commit: %v", err)
	}

	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Errorf("ListWorktrees failed: %v", err)
	}

	// Should have at least the main worktree
	if len(worktrees) < 1 {
		t.Error("expected at least one worktree (main)")
	}
}
