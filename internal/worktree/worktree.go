// Package worktree provides git worktree automation for isolated run environments.
package worktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree represents a git worktree.
type Worktree struct {
	repoPath string
	path     string
	branch   string
	created  bool
}

// New creates a new worktree manager.
func New(repoPath, worktreePath, branch string) *Worktree {
	return &Worktree{
		repoPath: repoPath,
		path:     worktreePath,
		branch:   branch,
	}
}

// Create creates the git worktree.
func (w *Worktree) Create(ctx context.Context) error {
	// Check if worktree already exists
	if _, err := os.Stat(w.path); err == nil {
		return nil
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Create worktree
	args := []string{"worktree", "add"}

	if w.branch != "" {
		// Create new branch
		args = append(args, "-b", w.branch)
	}

	args = append(args, w.path)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = w.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w\n%s", err, output)
	}

	w.created = true
	return nil
}

// Remove removes the git worktree.
func (w *Worktree) Remove(ctx context.Context, force bool) error {
	if _, err := os.Stat(w.path); os.IsNotExist(err) {
		return nil
	}

	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, w.path)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = w.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w\n%s", err, output)
	}

	// Also delete the branch if we created it
	if w.created && w.branch != "" {
		w.deleteBranch(ctx)
	}

	return nil
}

func (w *Worktree) deleteBranch(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "branch", "-D", w.branch)
	cmd.Dir = w.repoPath
	return cmd.Run()
}

// Path returns the worktree path.
func (w *Worktree) Path() string {
	return w.path
}

// Branch returns the worktree branch name.
func (w *Worktree) Branch() string {
	return w.branch
}

// WasCreated returns true if this instance created the worktree.
func (w *Worktree) WasCreated() bool {
	return w.created
}

// Exists checks if the worktree exists.
func (w *Worktree) Exists() bool {
	_, err := os.Stat(w.path)
	return err == nil
}

// IsGitWorktree checks if a path is a git worktree.
func IsGitWorktree(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	// Worktrees have a .git file (not directory) pointing to the main repo
	return !info.IsDir()
}

// IsGitRepo checks if a path is a git repository.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetRepoRoot returns the root of the git repository.
func GetRepoRoot(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ListWorktrees lists all worktrees for a repository.
func ListWorktrees(repoPath string) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []WorktreeInfo
	var current WorktreeInfo

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if line == "bare" {
			current.Bare = true
		} else if line == "detached" {
			current.Detached = true
		}
	}

	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// WorktreeInfo contains information about a worktree.
type WorktreeInfo struct {
	Path     string
	HEAD     string
	Branch   string
	Bare     bool
	Detached bool
}

// GenerateWorktreePath creates a worktree path for a run.
func GenerateWorktreePath(repoPath string, runID int) string {
	return filepath.Join(filepath.Dir(repoPath), fmt.Sprintf(".agentpair-worktree-%d", runID))
}

// GenerateBranchName creates a branch name for a run.
func GenerateBranchName(runID int) string {
	return fmt.Sprintf("agentpair/run-%d", runID)
}

// Cleanup removes all agentpair worktrees from a repository.
func Cleanup(ctx context.Context, repoPath string) error {
	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		return err
	}

	for _, wt := range worktrees {
		if strings.Contains(wt.Path, ".agentpair-worktree-") {
			w := New(repoPath, wt.Path, wt.Branch)
			if err := w.Remove(ctx, true); err != nil {
				// Log but continue
				fmt.Fprintf(os.Stderr, "warning: failed to remove worktree %s: %v\n", wt.Path, err)
			}
		}
	}

	return nil
}
