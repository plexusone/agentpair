package tmux

import (
	"testing"
)

func TestGenerateSessionName(t *testing.T) {
	tests := []struct {
		repoName string
		runID    int
		expected string
	}{
		{"myrepo", 1, "agentpair-myrepo-1"},
		{"my-repo", 5, "agentpair-my-repo-5"},
		{"my/repo", 10, "agentpair-my-repo-10"},
		{"my.repo", 20, "agentpair-my-repo-20"},
		{"my/repo.git", 100, "agentpair-my-repo-git-100"},
	}

	for _, tt := range tests {
		t.Run(tt.repoName, func(t *testing.T) {
			got := GenerateSessionName(tt.repoName, tt.runID)
			if got != tt.expected {
				t.Errorf("GenerateSessionName(%q, %d) = %q, want %q",
					tt.repoName, tt.runID, got, tt.expected)
			}
		})
	}
}

func TestNewSession(t *testing.T) {
	session := NewSession("test-session", "/test/dir")

	if session.Name() != "test-session" {
		t.Errorf("Name() = %s, want test-session", session.Name())
	}
	if session.WasCreated() {
		t.Error("WasCreated() should be false for new session")
	}
}

func TestNewLayout(t *testing.T) {
	session := NewSession("test-session", "/test/dir")
	layout := NewLayout(session)

	if layout == nil {
		t.Error("NewLayout should not return nil")
	}
	if layout.session != session {
		t.Error("Layout should reference the session")
	}
}

func TestPaneIndexConstants(t *testing.T) {
	// Verify pane constants are distinct
	if PaneClaude == PaneCodex {
		t.Error("PaneClaude and PaneCodex should be different")
	}
	if PaneClaude == PaneStatus {
		t.Error("PaneClaude and PaneStatus should be different")
	}
	if PaneCodex == PaneStatus {
		t.Error("PaneCodex and PaneStatus should be different")
	}

	// Verify expected values
	if PaneClaude != 0 {
		t.Errorf("PaneClaude = %d, want 0", PaneClaude)
	}
	if PaneCodex != 1 {
		t.Errorf("PaneCodex = %d, want 1", PaneCodex)
	}
	if PaneStatus != 2 {
		t.Errorf("PaneStatus = %d, want 2", PaneStatus)
	}
}

// Integration tests that require tmux to be available
func TestIsTmuxAvailable(t *testing.T) {
	// This just tests that the function doesn't panic
	// The result depends on whether tmux is installed
	available := IsTmuxAvailable()
	t.Logf("tmux available: %v", available)
}

// Skip tests that require tmux if not available
func skipIfNoTmux(t *testing.T) {
	if !IsTmuxAvailable() {
		t.Skip("tmux not available")
	}
}

func TestSessionExistsNonexistent(t *testing.T) {
	skipIfNoTmux(t)

	// Test with a session name that shouldn't exist
	session := NewSession("agentpair-test-nonexistent-12345", "/tmp")
	if session.Exists() {
		t.Error("Exists() should return false for non-existent session")
	}
}

func TestListSessions(t *testing.T) {
	skipIfNoTmux(t)

	// This may return an empty list or existing sessions
	sessions, err := ListSessions()
	if err != nil {
		// "no server running" is acceptable
		t.Logf("ListSessions returned error (may be expected): %v", err)
		return
	}

	t.Logf("Found %d tmux sessions", len(sessions))
	for _, s := range sessions {
		t.Logf("  - %s", s)
	}
}
