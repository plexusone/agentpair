package tmux

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/grokify/mogo/log/slogutil"
)

// Layout represents a tmux pane layout configuration.
type Layout struct {
	session *Session
}

// NewLayout creates a new layout manager for a session.
func NewLayout(session *Session) *Layout {
	return &Layout{session: session}
}

// PairedLayout creates the standard paired layout:
// +-------------+-------------+
// |   Claude    |   Codex     |
// +-------------+-------------+
func (l *Layout) PairedLayout(ctx context.Context) error {
	// Ensure session exists
	if err := l.session.Create(ctx); err != nil {
		return err
	}

	// Setup side-by-side layout
	return l.session.SetupLayout(ctx)
}

// TripleLayout creates a layout with a status pane:
// +-------------+-------------+
// |   Claude    |   Codex     |
// +-------------+-------------+
// |         Status            |
// +---------------------------+
func (l *Layout) TripleLayout(ctx context.Context) error {
	// First create paired layout
	if err := l.PairedLayout(ctx); err != nil {
		return err
	}

	// Split bottom for status pane
	target := fmt.Sprintf("%s:0.0", l.session.Name())
	cmd := exec.CommandContext(ctx, "tmux", "split-window",
		"-v", // vertical split
		"-t", target,
		"-p", "20", // 20% height
		"-c", l.session.workDir,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create status pane: %w", err)
	}

	// Rename the status pane
	if err := l.session.RenamePane(ctx, 2, "status"); err != nil {
		logger := slogutil.LoggerFromContext(ctx, slog.Default())
		logger.Warn("failed to rename status pane", "error", err)
	}

	return nil
}

// SetPaneTitle sets the title of a pane.
func (l *Layout) SetPaneTitle(ctx context.Context, paneIndex int, title string) error {
	return l.session.RenamePane(ctx, paneIndex, title)
}

// FocusPane focuses on a specific pane.
func (l *Layout) FocusPane(ctx context.Context, paneIndex int) error {
	target := fmt.Sprintf("%s:0.%d", l.session.Name(), paneIndex)
	cmd := exec.CommandContext(ctx, "tmux", "select-pane", "-t", target)
	return cmd.Run()
}

// SyncPanes enables synchronized input to all panes.
func (l *Layout) SyncPanes(ctx context.Context, enable bool) error {
	value := "off"
	if enable {
		value = "on"
	}

	target := fmt.Sprintf("%s:0", l.session.Name())
	cmd := exec.CommandContext(ctx, "tmux", "setw",
		"-t", target,
		"synchronize-panes", value,
	)
	return cmd.Run()
}

// ClearPane clears a pane's content.
func (l *Layout) ClearPane(ctx context.Context, paneIndex int) error {
	return l.session.SendKeys(ctx, paneIndex, "clear")
}

// GetPaneContent captures the content of a pane.
func (l *Layout) GetPaneContent(ctx context.Context, paneIndex int) (string, error) {
	target := fmt.Sprintf("%s:0.%d", l.session.Name(), paneIndex)
	cmd := exec.CommandContext(ctx, "tmux", "capture-pane",
		"-t", target,
		"-p", // print to stdout
	)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// ResizePanes resizes panes to equal size.
func (l *Layout) ResizePanes(ctx context.Context) error {
	target := fmt.Sprintf("%s:0", l.session.Name())
	cmd := exec.CommandContext(ctx, "tmux", "select-layout",
		"-t", target,
		"even-horizontal",
	)
	return cmd.Run()
}

// PaneIndex constants for the paired layout.
const (
	PaneClaude = 0
	PaneCodex  = 1
	PaneStatus = 2
)
