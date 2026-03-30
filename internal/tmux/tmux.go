// Package tmux provides tmux session management for side-by-side agent panes.
package tmux

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/grokify/mogo/log/slogutil"
)

// Session represents a tmux session.
type Session struct {
	name    string
	workDir string
	created bool
}

// NewSession creates a new tmux session manager.
func NewSession(name, workDir string) *Session {
	return &Session{
		name:    name,
		workDir: workDir,
	}
}

// Exists checks if the tmux session already exists.
func (s *Session) Exists() bool {
	// G204: s.name is intentionally user-provided for tmux session management
	cmd := exec.Command("tmux", "has-session", "-t", s.name) //nolint:gosec
	return cmd.Run() == nil
}

// Create creates the tmux session with side-by-side panes.
func (s *Session) Create(ctx context.Context) error {
	if s.Exists() {
		return nil
	}

	// Create new session (detached)
	// G204: s.name and s.workDir are intentionally user-provided for tmux session management
	cmd := exec.CommandContext(ctx, "tmux", "new-session", //nolint:gosec
		"-d",         // detached
		"-s", s.name, // session name
		"-c", s.workDir, // working directory
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	s.created = true
	return nil
}

// SetupLayout creates the side-by-side layout for Claude and Codex.
func (s *Session) SetupLayout(ctx context.Context) error {
	logger := slogutil.LoggerFromContext(ctx, slog.Default())

	// Split window horizontally (side by side)
	// G204: s.name and s.workDir are intentionally user-provided for tmux session management
	cmd := exec.CommandContext(ctx, "tmux", "split-window", //nolint:gosec
		"-h",         // horizontal split
		"-t", s.name, // target session
		"-c", s.workDir, // working directory
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to split window: %w", err)
	}

	// Select even-horizontal layout
	// G204: s.name is intentionally user-provided for tmux session management
	cmd = exec.CommandContext(ctx, "tmux", "select-layout", //nolint:gosec
		"-t", s.name,
		"even-horizontal",
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set layout: %w", err)
	}

	// Rename panes for clarity
	if err := s.RenamePane(ctx, 0, "claude"); err != nil {
		logger.Warn("failed to rename claude pane", "error", err)
	}
	if err := s.RenamePane(ctx, 1, "codex"); err != nil {
		logger.Warn("failed to rename codex pane", "error", err)
	}

	return nil
}

// RenamePane renames a pane (sets pane title).
func (s *Session) RenamePane(ctx context.Context, paneIndex int, name string) error {
	target := fmt.Sprintf("%s:%d.%d", s.name, 0, paneIndex)
	cmd := exec.CommandContext(ctx, "tmux", "select-pane",
		"-t", target,
		"-T", name, // title
	)
	return cmd.Run()
}

// SendKeys sends keys to a specific pane.
func (s *Session) SendKeys(ctx context.Context, paneIndex int, keys string) error {
	target := fmt.Sprintf("%s:%d.%d", s.name, 0, paneIndex)
	cmd := exec.CommandContext(ctx, "tmux", "send-keys",
		"-t", target,
		keys,
		"Enter",
	)
	return cmd.Run()
}

// RunInPane runs a command in a specific pane.
func (s *Session) RunInPane(ctx context.Context, paneIndex int, command string) error {
	return s.SendKeys(ctx, paneIndex, command)
}

// Attach attaches to the tmux session.
func (s *Session) Attach(ctx context.Context) error {
	// Check if we're already in tmux
	if os.Getenv("TMUX") != "" {
		// Switch to session
		// G204: s.name is intentionally user-provided for tmux session management
		cmd := exec.CommandContext(ctx, "tmux", "switch-client", //nolint:gosec
			"-t", s.name,
		)
		return cmd.Run()
	}

	// Attach to session
	// G204: s.name is intentionally user-provided for tmux session management
	cmd := exec.CommandContext(ctx, "tmux", "attach-session", //nolint:gosec
		"-t", s.name,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Kill terminates the tmux session.
func (s *Session) Kill(ctx context.Context) error {
	if !s.Exists() {
		return nil
	}

	// G204: s.name is intentionally user-provided for tmux session management
	cmd := exec.CommandContext(ctx, "tmux", "kill-session", //nolint:gosec
		"-t", s.name,
	)
	return cmd.Run()
}

// Name returns the session name.
func (s *Session) Name() string {
	return s.name
}

// WasCreated returns true if this instance created the session.
func (s *Session) WasCreated() bool {
	return s.created
}

// IsTmuxAvailable checks if tmux is installed and available.
func IsTmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// ListSessions returns a list of all tmux sessions.
func ListSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// No sessions is not an error
		if strings.Contains(err.Error(), "no server running") {
			return nil, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var sessions []string
	for _, line := range lines {
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

// GenerateSessionName creates a session name for a run.
func GenerateSessionName(repoName string, runID int) string {
	// Clean repo name for tmux
	clean := strings.ReplaceAll(repoName, "/", "-")
	clean = strings.ReplaceAll(clean, ".", "-")
	return fmt.Sprintf("agentpair-%s-%d", clean, runID)
}
