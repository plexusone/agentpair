package run

import (
	"encoding/json"
	"os"
	"time"
)

// State represents the current state of a run.
type State string

const (
	StateSubmitted State = "submitted"
	StateWorking   State = "working"
	StateReviewing State = "reviewing"
	StateCompleted State = "completed"
	StateFailed    State = "failed"
	StateCancelled State = "cancelled"
)

// Manifest contains metadata about a run.
type Manifest struct {
	// ID is the run identifier.
	ID int `json:"id"`

	// RepoPath is the path to the repository.
	RepoPath string `json:"repo_path"`

	// RepoID is the hashed repository identifier.
	RepoID string `json:"repo_id"`

	// Prompt is the initial task prompt.
	Prompt string `json:"prompt"`

	// State is the current run state.
	State State `json:"state"`

	// PrimaryAgent is the main worker agent.
	PrimaryAgent string `json:"primary_agent"`

	// ReviewMode is the review configuration.
	ReviewMode string `json:"review_mode"`

	// MaxIterations is the iteration limit.
	MaxIterations int `json:"max_iterations"`

	// CurrentIteration is the current iteration number.
	CurrentIteration int `json:"current_iteration"`

	// ClaudeSessionID is Claude's session ID for resume.
	ClaudeSessionID string `json:"claude_session_id,omitempty"`

	// CodexSessionID is Codex's session ID for resume.
	CodexSessionID string `json:"codex_session_id,omitempty"`

	// WorktreePath is the git worktree path if enabled.
	WorktreePath string `json:"worktree_path,omitempty"`

	// TmuxSession is the tmux session name if enabled.
	TmuxSession string `json:"tmux_session,omitempty"`

	// ProofCommand is the proof/verification command.
	ProofCommand string `json:"proof_command,omitempty"`

	// DoneSignal is the custom done signal.
	DoneSignal string `json:"done_signal"`

	// CreatedAt is when the run was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the run was last updated.
	UpdatedAt time.Time `json:"updated_at"`

	// CompletedAt is when the run completed.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`

	// Metadata contains additional key-value data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewManifest creates a new manifest with default values.
func NewManifest(id int, repoPath, repoID, prompt string) *Manifest {
	now := time.Now().UTC()
	return &Manifest{
		ID:            id,
		RepoPath:      repoPath,
		RepoID:        repoID,
		Prompt:        prompt,
		State:         StateSubmitted,
		PrimaryAgent:  "codex",
		ReviewMode:    "claudex",
		MaxIterations: 20,
		DoneSignal:    "DONE",
		CreatedAt:     now,
		UpdatedAt:     now,
		Metadata:      make(map[string]any),
	}
}

// Save writes the manifest to a file.
func (m *Manifest) Save(path string) error {
	m.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// LoadManifest loads a manifest from a file.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SetState updates the state and timestamp.
func (m *Manifest) SetState(state State) {
	m.State = state
	m.UpdatedAt = time.Now().UTC()
	if state == StateCompleted || state == StateFailed || state == StateCancelled {
		now := time.Now().UTC()
		m.CompletedAt = &now
	}
}

// IncrementIteration increments the iteration counter.
func (m *Manifest) IncrementIteration() {
	m.CurrentIteration++
	m.UpdatedAt = time.Now().UTC()
}

// IsComplete returns true if the run has finished.
func (m *Manifest) IsComplete() bool {
	return m.State == StateCompleted || m.State == StateFailed || m.State == StateCancelled
}

// IsSingleAgent returns true if running in single-agent mode.
func (m *Manifest) IsSingleAgent() bool {
	return m.ReviewMode == "claude" || m.ReviewMode == "codex"
}
