package run

import (
	"time"

	"github.com/plexusone/agentpair/pkg/jsonl"
)

// EntryType represents the type of transcript entry.
type EntryType string

const (
	EntryTypeStart     EntryType = "start"
	EntryTypeIteration EntryType = "iteration"
	EntryTypeMessage   EntryType = "message"
	EntryTypeExecute   EntryType = "execute"
	EntryTypeResult    EntryType = "result"
	EntryTypeSignal    EntryType = "signal"
	EntryTypeState     EntryType = "state"
	EntryTypeError     EntryType = "error"
	EntryTypeEnd       EntryType = "end"
)

// TranscriptEntry represents a single entry in the audit log.
type TranscriptEntry struct {
	// Type categorizes the entry.
	Type EntryType `json:"type"`

	// Timestamp is when the entry was created.
	Timestamp time.Time `json:"timestamp"`

	// RunID is the run identifier.
	RunID int `json:"run_id"`

	// Iteration is the loop iteration number.
	Iteration int `json:"iteration,omitempty"`

	// Agent is the agent name (if applicable).
	Agent string `json:"agent,omitempty"`

	// State is the run state (for state entries).
	State State `json:"state,omitempty"`

	// Content is the main content (message, prompt, result, etc.).
	Content string `json:"content,omitempty"`

	// Signal is the signal value (DONE, PASS, FAIL).
	Signal string `json:"signal,omitempty"`

	// Error is the error message (for error entries).
	Error string `json:"error,omitempty"`

	// Duration is the elapsed time (for result entries).
	Duration time.Duration `json:"duration,omitempty"`

	// Metadata contains additional data.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Transcript manages the audit log for a run.
type Transcript struct {
	path  string
	runID int
}

// NewTranscript creates a new transcript for the given path.
func NewTranscript(path string, runID int) *Transcript {
	return &Transcript{
		path:  path,
		runID: runID,
	}
}

// Append adds an entry to the transcript.
func (t *Transcript) Append(entry *TranscriptEntry) error {
	entry.Timestamp = time.Now().UTC()
	entry.RunID = t.runID
	return jsonl.AppendOne(t.path, entry)
}

// LogStart logs the start of a run.
func (t *Transcript) LogStart(prompt string, metadata map[string]any) error {
	return t.Append(&TranscriptEntry{
		Type:     EntryTypeStart,
		Content:  prompt,
		Metadata: metadata,
	})
}

// LogIteration logs the start of an iteration.
func (t *Transcript) LogIteration(iteration int) error {
	return t.Append(&TranscriptEntry{
		Type:      EntryTypeIteration,
		Iteration: iteration,
	})
}

// LogExecute logs an execution request to an agent.
func (t *Transcript) LogExecute(agent string, iteration int, prompt string) error {
	return t.Append(&TranscriptEntry{
		Type:      EntryTypeExecute,
		Agent:     agent,
		Iteration: iteration,
		Content:   prompt,
	})
}

// LogResult logs a result from an agent.
func (t *Transcript) LogResult(agent string, iteration int, result string, duration time.Duration) error {
	return t.Append(&TranscriptEntry{
		Type:      EntryTypeResult,
		Agent:     agent,
		Iteration: iteration,
		Content:   result,
		Duration:  duration,
	})
}

// LogSignal logs a signal (DONE, PASS, FAIL).
func (t *Transcript) LogSignal(agent string, iteration int, signal string) error {
	return t.Append(&TranscriptEntry{
		Type:      EntryTypeSignal,
		Agent:     agent,
		Iteration: iteration,
		Signal:    signal,
	})
}

// LogState logs a state transition.
func (t *Transcript) LogState(state State) error {
	return t.Append(&TranscriptEntry{
		Type:  EntryTypeState,
		State: state,
	})
}

// LogError logs an error.
func (t *Transcript) LogError(agent string, iteration int, err error) error {
	return t.Append(&TranscriptEntry{
		Type:      EntryTypeError,
		Agent:     agent,
		Iteration: iteration,
		Error:     err.Error(),
	})
}

// LogEnd logs the end of a run.
func (t *Transcript) LogEnd(state State, totalDuration time.Duration) error {
	return t.Append(&TranscriptEntry{
		Type:     EntryTypeEnd,
		State:    state,
		Duration: totalDuration,
	})
}

// ReadAll reads all entries from the transcript.
func (t *Transcript) ReadAll() ([]*TranscriptEntry, error) {
	return jsonl.ReadAll[*TranscriptEntry](t.path)
}

// Path returns the transcript file path.
func (t *Transcript) Path() string {
	return t.path
}
