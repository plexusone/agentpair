package run

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/plexusone/agentpair/internal/config"
)

func TestNewManifest(t *testing.T) {
	m := NewManifest(1, "/repo/path", "repo-id", "test prompt")

	if m.ID != 1 {
		t.Errorf("ID = %d, want 1", m.ID)
	}
	if m.RepoPath != "/repo/path" {
		t.Errorf("RepoPath = %s, want /repo/path", m.RepoPath)
	}
	if m.RepoID != "repo-id" {
		t.Errorf("RepoID = %s, want repo-id", m.RepoID)
	}
	if m.Prompt != "test prompt" {
		t.Errorf("Prompt = %s, want test prompt", m.Prompt)
	}
	if m.State != StateSubmitted {
		t.Errorf("State = %s, want %s", m.State, StateSubmitted)
	}
	if m.PrimaryAgent != "codex" {
		t.Errorf("PrimaryAgent = %s, want codex", m.PrimaryAgent)
	}
	if m.ReviewMode != "claudex" {
		t.Errorf("ReviewMode = %s, want claudex", m.ReviewMode)
	}
	if m.MaxIterations != 20 {
		t.Errorf("MaxIterations = %d, want 20", m.MaxIterations)
	}
	if m.DoneSignal != "DONE" {
		t.Errorf("DoneSignal = %s, want DONE", m.DoneSignal)
	}
	if m.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if m.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
	if m.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}

func TestManifestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	original := NewManifest(5, "/test/repo", "test-repo-id", "implement feature X")
	original.PrimaryAgent = "claude"
	original.ReviewMode = "codex"
	original.MaxIterations = 30
	original.ClaudeSessionID = "claude-session-123"
	original.CodexSessionID = "codex-session-456"
	original.WorktreePath = "/test/worktree"
	original.TmuxSession = "agentpair-test"
	original.ProofCommand = "go test ./..."
	original.Metadata["custom"] = "value"

	if err := original.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if loaded.ID != original.ID {
		t.Errorf("ID = %d, want %d", loaded.ID, original.ID)
	}
	if loaded.RepoPath != original.RepoPath {
		t.Errorf("RepoPath = %s, want %s", loaded.RepoPath, original.RepoPath)
	}
	if loaded.RepoID != original.RepoID {
		t.Errorf("RepoID = %s, want %s", loaded.RepoID, original.RepoID)
	}
	if loaded.Prompt != original.Prompt {
		t.Errorf("Prompt = %s, want %s", loaded.Prompt, original.Prompt)
	}
	if loaded.PrimaryAgent != original.PrimaryAgent {
		t.Errorf("PrimaryAgent = %s, want %s", loaded.PrimaryAgent, original.PrimaryAgent)
	}
	if loaded.ReviewMode != original.ReviewMode {
		t.Errorf("ReviewMode = %s, want %s", loaded.ReviewMode, original.ReviewMode)
	}
	if loaded.MaxIterations != original.MaxIterations {
		t.Errorf("MaxIterations = %d, want %d", loaded.MaxIterations, original.MaxIterations)
	}
	if loaded.ClaudeSessionID != original.ClaudeSessionID {
		t.Errorf("ClaudeSessionID = %s, want %s", loaded.ClaudeSessionID, original.ClaudeSessionID)
	}
	if loaded.CodexSessionID != original.CodexSessionID {
		t.Errorf("CodexSessionID = %s, want %s", loaded.CodexSessionID, original.CodexSessionID)
	}
	if loaded.WorktreePath != original.WorktreePath {
		t.Errorf("WorktreePath = %s, want %s", loaded.WorktreePath, original.WorktreePath)
	}
	if loaded.TmuxSession != original.TmuxSession {
		t.Errorf("TmuxSession = %s, want %s", loaded.TmuxSession, original.TmuxSession)
	}
	if loaded.ProofCommand != original.ProofCommand {
		t.Errorf("ProofCommand = %s, want %s", loaded.ProofCommand, original.ProofCommand)
	}
}

func TestManifestLoadNotExist(t *testing.T) {
	_, err := LoadManifest("/nonexistent/manifest.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestManifestSetState(t *testing.T) {
	tests := []struct {
		name          string
		state         State
		expectComplete bool
	}{
		{"submitted", StateSubmitted, false},
		{"working", StateWorking, false},
		{"reviewing", StateReviewing, false},
		{"completed", StateCompleted, true},
		{"failed", StateFailed, true},
		{"cancelled", StateCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManifest(1, "/repo", "id", "prompt")
			originalUpdatedAt := m.UpdatedAt

			// Small delay to ensure UpdatedAt changes
			time.Sleep(time.Millisecond)

			m.SetState(tt.state)

			if m.State != tt.state {
				t.Errorf("State = %s, want %s", m.State, tt.state)
			}
			if m.UpdatedAt == originalUpdatedAt {
				t.Error("UpdatedAt should have changed")
			}
			if tt.expectComplete && m.CompletedAt == nil {
				t.Error("CompletedAt should be set for terminal states")
			}
			if !tt.expectComplete && m.CompletedAt != nil {
				t.Error("CompletedAt should not be set for non-terminal states")
			}
		})
	}
}

func TestManifestIncrementIteration(t *testing.T) {
	m := NewManifest(1, "/repo", "id", "prompt")
	originalUpdatedAt := m.UpdatedAt

	time.Sleep(time.Millisecond)

	if m.CurrentIteration != 0 {
		t.Errorf("CurrentIteration = %d, want 0", m.CurrentIteration)
	}

	m.IncrementIteration()

	if m.CurrentIteration != 1 {
		t.Errorf("CurrentIteration = %d, want 1", m.CurrentIteration)
	}
	if m.UpdatedAt == originalUpdatedAt {
		t.Error("UpdatedAt should have changed")
	}

	m.IncrementIteration()
	m.IncrementIteration()

	if m.CurrentIteration != 3 {
		t.Errorf("CurrentIteration = %d, want 3", m.CurrentIteration)
	}
}

func TestManifestIsComplete(t *testing.T) {
	tests := []struct {
		state    State
		expected bool
	}{
		{StateSubmitted, false},
		{StateWorking, false},
		{StateReviewing, false},
		{StateCompleted, true},
		{StateFailed, true},
		{StateCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			m := &Manifest{State: tt.state}
			if got := m.IsComplete(); got != tt.expected {
				t.Errorf("IsComplete() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestManifestIsSingleAgent(t *testing.T) {
	tests := []struct {
		reviewMode string
		expected   bool
	}{
		{"claude", true},
		{"codex", true},
		{"claudex", false},
		{"other", false},
	}

	for _, tt := range tests {
		t.Run(tt.reviewMode, func(t *testing.T) {
			m := &Manifest{ReviewMode: tt.reviewMode}
			if got := m.IsSingleAgent(); got != tt.expected {
				t.Errorf("IsSingleAgent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTranscript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	transcript := NewTranscript(path, 1)

	if transcript.Path() != path {
		t.Errorf("Path() = %s, want %s", transcript.Path(), path)
	}

	// Test LogStart
	if err := transcript.LogStart("test prompt", map[string]any{"key": "value"}); err != nil {
		t.Fatalf("LogStart failed: %v", err)
	}

	// Test LogIteration
	if err := transcript.LogIteration(1); err != nil {
		t.Fatalf("LogIteration failed: %v", err)
	}

	// Test LogExecute
	if err := transcript.LogExecute("claude", 1, "execute this"); err != nil {
		t.Fatalf("LogExecute failed: %v", err)
	}

	// Test LogResult
	if err := transcript.LogResult("claude", 1, "result content", 5*time.Second); err != nil {
		t.Fatalf("LogResult failed: %v", err)
	}

	// Test LogSignal
	if err := transcript.LogSignal("codex", 1, "PASS"); err != nil {
		t.Fatalf("LogSignal failed: %v", err)
	}

	// Test LogState
	if err := transcript.LogState(StateWorking); err != nil {
		t.Fatalf("LogState failed: %v", err)
	}

	// Test LogError
	if err := transcript.LogError("claude", 2, errors.New("test error")); err != nil {
		t.Fatalf("LogError failed: %v", err)
	}

	// Test LogEnd
	if err := transcript.LogEnd(StateCompleted, 10*time.Minute); err != nil {
		t.Fatalf("LogEnd failed: %v", err)
	}

	// Test ReadAll
	entries, err := transcript.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if len(entries) != 8 {
		t.Errorf("len(entries) = %d, want 8", len(entries))
	}

	// Verify entry types in order
	expectedTypes := []EntryType{
		EntryTypeStart,
		EntryTypeIteration,
		EntryTypeExecute,
		EntryTypeResult,
		EntryTypeSignal,
		EntryTypeState,
		EntryTypeError,
		EntryTypeEnd,
	}

	for i, et := range expectedTypes {
		if entries[i].Type != et {
			t.Errorf("entries[%d].Type = %s, want %s", i, entries[i].Type, et)
		}
		if entries[i].RunID != 1 {
			t.Errorf("entries[%d].RunID = %d, want 1", i, entries[i].RunID)
		}
		if entries[i].Timestamp.IsZero() {
			t.Errorf("entries[%d].Timestamp should not be zero", i)
		}
	}

	// Verify specific entry content
	if entries[0].Content != "test prompt" {
		t.Errorf("start entry Content = %s, want test prompt", entries[0].Content)
	}
	if entries[1].Iteration != 1 {
		t.Errorf("iteration entry Iteration = %d, want 1", entries[1].Iteration)
	}
	if entries[2].Agent != "claude" {
		t.Errorf("execute entry Agent = %s, want claude", entries[2].Agent)
	}
	if entries[3].Duration != 5*time.Second {
		t.Errorf("result entry Duration = %v, want 5s", entries[3].Duration)
	}
	if entries[4].Signal != "PASS" {
		t.Errorf("signal entry Signal = %s, want PASS", entries[4].Signal)
	}
	if entries[5].State != StateWorking {
		t.Errorf("state entry State = %s, want %s", entries[5].State, StateWorking)
	}
	if entries[6].Error != "test error" {
		t.Errorf("error entry Error = %s, want test error", entries[6].Error)
	}
	if entries[7].Duration != 10*time.Minute {
		t.Errorf("end entry Duration = %v, want 10m", entries[7].Duration)
	}
}

func TestTranscriptReadAllEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")

	transcript := NewTranscript(path, 1)
	entries, err := transcript.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

// setupTestPaths creates a temporary Paths instance for testing
func setupTestPaths(t *testing.T) (*config.Paths, string) {
	baseDir := t.TempDir()
	paths := config.NewPathsWithBase(baseDir)
	return paths, baseDir
}

func TestManagerCreateAndLoad(t *testing.T) {
	paths, _ := setupTestPaths(t)
	repoPath := "/test/repo"
	manager := NewManager(paths, repoPath)

	cfg := config.DefaultConfig()
	cfg.Prompt = "test task"
	cfg.Agent = "claude"
	cfg.ReviewMode = "codex"
	cfg.MaxIterations = 15
	cfg.Proof = "make test"

	// Create a run
	run, err := manager.Create("test task", cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer run.Close()

	if run.Manifest.ID != 1 {
		t.Errorf("Manifest.ID = %d, want 1", run.Manifest.ID)
	}
	if run.Manifest.Prompt != "test task" {
		t.Errorf("Manifest.Prompt = %s, want test task", run.Manifest.Prompt)
	}
	if run.Manifest.PrimaryAgent != "claude" {
		t.Errorf("Manifest.PrimaryAgent = %s, want claude", run.Manifest.PrimaryAgent)
	}
	if run.Manifest.ReviewMode != "codex" {
		t.Errorf("Manifest.ReviewMode = %s, want codex", run.Manifest.ReviewMode)
	}
	if run.Manifest.MaxIterations != 15 {
		t.Errorf("Manifest.MaxIterations = %d, want 15", run.Manifest.MaxIterations)
	}
	if run.Manifest.ProofCommand != "make test" {
		t.Errorf("Manifest.ProofCommand = %s, want make test", run.Manifest.ProofCommand)
	}

	// Save the run
	if err := run.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load the run
	loaded, err := manager.Load(1)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer loaded.Close()

	if loaded.Manifest.ID != run.Manifest.ID {
		t.Errorf("loaded ID = %d, want %d", loaded.Manifest.ID, run.Manifest.ID)
	}
	if loaded.Manifest.Prompt != run.Manifest.Prompt {
		t.Errorf("loaded Prompt = %s, want %s", loaded.Manifest.Prompt, run.Manifest.Prompt)
	}
}

func TestManagerLoadNotExist(t *testing.T) {
	paths, _ := setupTestPaths(t)
	manager := NewManager(paths, "/test/repo")

	_, err := manager.Load(999)
	if err == nil {
		t.Error("expected error for non-existent run")
	}
}

func TestManagerList(t *testing.T) {
	paths, _ := setupTestPaths(t)
	repoPath := "/test/repo"
	manager := NewManager(paths, repoPath)
	cfg := config.DefaultConfig()

	// Initially empty
	ids, err := manager.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("len(ids) = %d, want 0", len(ids))
	}

	// Create runs
	run1, err := manager.Create("task 1", cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	run1.Close()

	run2, err := manager.Create("task 2", cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	run2.Close()

	// List should now have 2
	ids, err = manager.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("len(ids) = %d, want 2", len(ids))
	}
}

func TestManagerListActive(t *testing.T) {
	paths, _ := setupTestPaths(t)
	repoPath := "/test/repo"
	manager := NewManager(paths, repoPath)
	cfg := config.DefaultConfig()

	// Create runs
	run1, err := manager.Create("active task", cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	run1.Save()
	run1.Close()

	run2, err := manager.Create("completed task", cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	run2.Manifest.SetState(StateCompleted)
	run2.Save()
	run2.Close()

	// ListActive should only return 1
	active, err := manager.ListActive()
	if err != nil {
		t.Fatalf("ListActive failed: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("len(active) = %d, want 1", len(active))
	}
	if active[0].Prompt != "active task" {
		t.Errorf("active[0].Prompt = %s, want active task", active[0].Prompt)
	}
}

func TestManagerFindBySessionID(t *testing.T) {
	paths, _ := setupTestPaths(t)
	repoPath := "/test/repo"
	manager := NewManager(paths, repoPath)
	cfg := config.DefaultConfig()

	// Create run with session IDs
	run, err := manager.Create("test task", cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	run.Manifest.ClaudeSessionID = "claude-abc123"
	run.Manifest.CodexSessionID = "codex-xyz789"
	run.Save()
	run.Close()

	// Find by Claude session ID
	found, err := manager.FindBySessionID("claude-abc123")
	if err != nil {
		t.Fatalf("FindBySessionID failed: %v", err)
	}
	defer found.Close()
	if found.Manifest.ID != run.Manifest.ID {
		t.Errorf("found ID = %d, want %d", found.Manifest.ID, run.Manifest.ID)
	}

	// Find by Codex session ID
	found2, err := manager.FindBySessionID("codex-xyz789")
	if err != nil {
		t.Fatalf("FindBySessionID failed: %v", err)
	}
	defer found2.Close()
	if found2.Manifest.ID != run.Manifest.ID {
		t.Errorf("found ID = %d, want %d", found2.Manifest.ID, run.Manifest.ID)
	}

	// Not found
	_, err = manager.FindBySessionID("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent session ID")
	}
}

func TestRunDir(t *testing.T) {
	paths, _ := setupTestPaths(t)
	repoPath := "/test/repo"
	manager := NewManager(paths, repoPath)
	cfg := config.DefaultConfig()

	run, err := manager.Create("test task", cfg)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	defer run.Close()

	dir := run.Dir()
	if dir == "" {
		t.Error("Dir() should not be empty")
	}

	// Verify directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("run directory does not exist: %s", dir)
	}
}

func TestListAllRepos(t *testing.T) {
	paths, baseDir := setupTestPaths(t)

	// Create some repo directories
	runsDir := filepath.Join(baseDir, "runs")
	os.MkdirAll(filepath.Join(runsDir, "repo1"), 0755)
	os.MkdirAll(filepath.Join(runsDir, "repo2"), 0755)
	os.MkdirAll(filepath.Join(runsDir, "repo3"), 0755)

	repos, err := ListAllRepos(paths)
	if err != nil {
		t.Fatalf("ListAllRepos failed: %v", err)
	}
	if len(repos) != 3 {
		t.Errorf("len(repos) = %d, want 3", len(repos))
	}
}

func TestListAllReposEmpty(t *testing.T) {
	paths, _ := setupTestPaths(t)

	repos, err := ListAllRepos(paths)
	if err != nil {
		t.Fatalf("ListAllRepos failed: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("len(repos) = %d, want 0", len(repos))
	}
}

func TestLoadManifestByPath(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "1")
	os.MkdirAll(runDir, 0755)

	manifest := NewManifest(1, "/test/repo", "test-id", "test prompt")
	manifestPath := filepath.Join(runDir, "manifest.json")
	if err := manifest.Save(manifestPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadManifestByPath(runDir)
	if err != nil {
		t.Fatalf("LoadManifestByPath failed: %v", err)
	}

	if loaded.ID != manifest.ID {
		t.Errorf("ID = %d, want %d", loaded.ID, manifest.ID)
	}
	if loaded.Prompt != manifest.Prompt {
		t.Errorf("Prompt = %s, want %s", loaded.Prompt, manifest.Prompt)
	}
}

func TestMultipleRunsIncrementID(t *testing.T) {
	paths, _ := setupTestPaths(t)
	repoPath := "/test/repo"
	manager := NewManager(paths, repoPath)
	cfg := config.DefaultConfig()

	// Create multiple runs and verify IDs increment
	for i := 1; i <= 5; i++ {
		run, err := manager.Create("task", cfg)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if run.Manifest.ID != i {
			t.Errorf("run %d ID = %d, want %d", i, run.Manifest.ID, i)
		}
		run.Close()
	}
}
