// Package dashboard provides a live terminal UI for monitoring active runs.
package dashboard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/plexusone/agentpair/internal/config"
	"github.com/plexusone/agentpair/internal/run"
)

// Dashboard displays live status of all active runs.
type Dashboard struct {
	paths       *config.Paths
	refreshRate time.Duration
	done        chan struct{}
}

// New creates a new dashboard.
func New(paths *config.Paths) *Dashboard {
	return &Dashboard{
		paths:       paths,
		refreshRate: 2 * time.Second,
		done:        make(chan struct{}),
	}
}

// Run starts the dashboard display loop.
func (d *Dashboard) Run(ctx context.Context) error {
	ticker := time.NewTicker(d.refreshRate)
	defer ticker.Stop()

	// Initial render
	d.render()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.done:
			return nil
		case <-ticker.C:
			d.render()
		}
	}
}

// Stop stops the dashboard.
func (d *Dashboard) Stop() {
	close(d.done)
}

func (d *Dashboard) render() {
	// Clear screen
	fmt.Print("\033[H\033[2J")

	// Header
	fmt.Println("╔══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        AgentPair Dashboard                           ║")
	fmt.Printf("║                     %s                            ║\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("╠══════════════════════════════════════════════════════════════════════╣")

	// Get all runs
	runs, err := d.getAllRuns()
	if err != nil {
		fmt.Printf("║ Error: %v\n", err)
		fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
		return
	}

	if len(runs) == 0 {
		fmt.Println("║ No active runs                                                       ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
		return
	}

	// Separate active and recent runs
	var active, recent []*RunStatus
	for _, r := range runs {
		if r.Manifest.IsComplete() {
			recent = append(recent, r)
		} else {
			active = append(active, r)
		}
	}

	// Show active runs
	if len(active) > 0 {
		fmt.Println("║ ACTIVE RUNS                                                          ║")
		fmt.Println("╟──────────────────────────────────────────────────────────────────────╢")
		for _, r := range active {
			d.renderRun(r)
		}
	}

	// Show recent completed runs (last 5)
	if len(recent) > 0 {
		fmt.Println("╟──────────────────────────────────────────────────────────────────────╢")
		fmt.Println("║ RECENT RUNS                                                          ║")
		fmt.Println("╟──────────────────────────────────────────────────────────────────────╢")

		// Sort by completion time, most recent first
		sort.Slice(recent, func(i, j int) bool {
			if recent[i].Manifest.CompletedAt == nil {
				return false
			}
			if recent[j].Manifest.CompletedAt == nil {
				return true
			}
			return recent[i].Manifest.CompletedAt.After(*recent[j].Manifest.CompletedAt)
		})

		// Show at most 5 recent runs
		limit := 5
		if len(recent) < limit {
			limit = len(recent)
		}
		for _, r := range recent[:limit] {
			d.renderRun(r)
		}
	}

	fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	fmt.Println("\nPress Ctrl+C to exit")
}

func (d *Dashboard) renderRun(r *RunStatus) {
	// Status indicator
	var statusIcon string
	switch r.Manifest.State {
	case run.StateWorking:
		statusIcon = "⚙️ "
	case run.StateReviewing:
		statusIcon = "👀"
	case run.StateCompleted:
		statusIcon = "✅"
	case run.StateFailed:
		statusIcon = "❌"
	default:
		statusIcon = "⏳"
	}

	// Truncate prompt
	prompt := r.Manifest.Prompt
	if len(prompt) > 40 {
		prompt = prompt[:37] + "..."
	}

	// Duration
	var duration string
	if r.Manifest.CompletedAt != nil {
		duration = r.Manifest.CompletedAt.Sub(r.Manifest.CreatedAt).Round(time.Second).String()
	} else {
		duration = time.Since(r.Manifest.CreatedAt).Round(time.Second).String()
	}

	fmt.Printf("║ %s Run #%-3d │ %-10s │ Iter %d/%-3d │ %s\n",
		statusIcon,
		r.Manifest.ID,
		r.Manifest.State,
		r.Manifest.CurrentIteration,
		r.Manifest.MaxIterations,
		duration,
	)
	fmt.Printf("║    └─ %-62s ║\n", prompt)

	// Show bridge status for active runs
	if !r.Manifest.IsComplete() && r.BridgeStatus != nil {
		fmt.Printf("║       Messages: %d │ Pass: %d │ Fail: %d\n",
			r.BridgeStatus.TotalMessages,
			r.BridgeStatus.PassCount,
			r.BridgeStatus.FailCount,
		)
	}
}

func (d *Dashboard) getAllRuns() ([]*RunStatus, error) {
	repoIDs, err := run.ListAllRepos(d.paths)
	if err != nil {
		return nil, err
	}

	var runs []*RunStatus

	for _, repoID := range repoIDs {
		repoDir := filepath.Join(d.paths.RunsDir(), repoID)
		entries, err := os.ReadDir(repoDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			runDir := filepath.Join(repoDir, entry.Name())
			manifest, err := run.LoadManifestByPath(runDir)
			if err != nil {
				continue
			}

			status := &RunStatus{
				Manifest: manifest,
				RepoID:   repoID,
			}

			// Load bridge status for active runs
			if !manifest.IsComplete() {
				bridgePath := filepath.Join(runDir, "bridge.jsonl")
				if _, err := os.Stat(bridgePath); err == nil {
					// Simple message count from file
					status.BridgeStatus = &BridgeStatus{}
					// TODO: Actually parse bridge file for stats
				}
			}

			runs = append(runs, status)
		}
	}

	// Sort by ID descending
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Manifest.ID > runs[j].Manifest.ID
	})

	return runs, nil
}

// RunStatus contains status information for a run.
type RunStatus struct {
	Manifest     *run.Manifest
	RepoID       string
	BridgeStatus *BridgeStatus
}

// BridgeStatus contains simplified bridge statistics.
type BridgeStatus struct {
	TotalMessages int
	PassCount     int
	FailCount     int
}

// PrintSummary prints a one-line summary of all runs.
func (d *Dashboard) PrintSummary() error {
	runs, err := d.getAllRuns()
	if err != nil {
		return err
	}

	var active, completed, failed int
	for _, r := range runs {
		switch r.Manifest.State {
		case run.StateCompleted:
			completed++
		case run.StateFailed:
			failed++
		default:
			active++
		}
	}

	fmt.Printf("AgentPair: %d active, %d completed, %d failed\n", active, completed, failed)
	return nil
}

// FormatDuration formats a duration for display.
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// TruncateString truncates a string to maxLen with ellipsis.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// PadRight pads a string to the right with spaces.
func PadRight(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	return s + strings.Repeat(" ", length-len(s))
}
