package bridge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// MessageType represents the type of bridge message.
type MessageType string

const (
	// TypeTask indicates a task assignment message.
	TypeTask MessageType = "task"
	// TypeResult indicates a task result message.
	TypeResult MessageType = "result"
	// TypeReview indicates a review request/response.
	TypeReview MessageType = "review"
	// TypeSignal indicates a control signal (e.g., DONE, PASS, FAIL).
	TypeSignal MessageType = "signal"
	// TypeChat indicates a general chat/context message.
	TypeChat MessageType = "chat"
)

// Signal represents control signals.
type Signal string

const (
	SignalDone Signal = "DONE"
	SignalPass Signal = "PASS"
	SignalFail Signal = "FAIL"
)

// Message represents a message passed between agents via the bridge.
type Message struct {
	// ID is the unique SHA256 hash of the message content.
	ID string `json:"id"`

	// Type categorizes the message.
	Type MessageType `json:"type"`

	// From identifies the sender agent.
	From string `json:"from"`

	// To identifies the target agent (empty for broadcast).
	To string `json:"to,omitempty"`

	// Content is the message payload.
	Content string `json:"content"`

	// Signal is set for signal-type messages.
	Signal Signal `json:"signal,omitempty"`

	// Metadata contains optional key-value data.
	Metadata map[string]any `json:"metadata,omitempty"`

	// Timestamp is when the message was created.
	Timestamp time.Time `json:"timestamp"`

	// RunID identifies the run this message belongs to.
	RunID int `json:"run_id"`

	// Iteration is the loop iteration number.
	Iteration int `json:"iteration"`
}

// NewMessage creates a new message with auto-generated ID.
func NewMessage(msgType MessageType, from, to, content string) *Message {
	m := &Message{
		Type:      msgType,
		From:      from,
		To:        to,
		Content:   content,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]any),
	}
	m.ID = m.computeID()
	return m
}

// NewTaskMessage creates a task assignment message.
func NewTaskMessage(from, to, task string) *Message {
	return NewMessage(TypeTask, from, to, task)
}

// NewResultMessage creates a result message.
func NewResultMessage(from, to, result string) *Message {
	return NewMessage(TypeResult, from, to, result)
}

// NewReviewMessage creates a review request/response message.
func NewReviewMessage(from, to, review string) *Message {
	return NewMessage(TypeReview, from, to, review)
}

// NewSignalMessage creates a signal message.
func NewSignalMessage(from string, signal Signal, content string) *Message {
	m := NewMessage(TypeSignal, from, "", content)
	m.Signal = signal
	m.ID = m.computeID() // Recompute with signal
	return m
}

// WithMetadata adds metadata to the message and returns it for chaining.
func (m *Message) WithMetadata(key string, value any) *Message {
	if m.Metadata == nil {
		m.Metadata = make(map[string]any)
	}
	m.Metadata[key] = value
	return m
}

// WithRunInfo sets run ID and iteration and returns it for chaining.
func (m *Message) WithRunInfo(runID, iteration int) *Message {
	m.RunID = runID
	m.Iteration = iteration
	return m
}

// computeID generates a SHA256 hash of the message content for deduplication.
func (m *Message) computeID() string {
	// Hash key fields to create a unique identifier
	data := struct {
		Type      MessageType `json:"type"`
		From      string      `json:"from"`
		To        string      `json:"to"`
		Content   string      `json:"content"`
		Signal    Signal      `json:"signal"`
		Timestamp int64       `json:"ts"`
	}{
		Type:      m.Type,
		From:      m.From,
		To:        m.To,
		Content:   m.Content,
		Signal:    m.Signal,
		Timestamp: m.Timestamp.UnixNano(),
	}

	b, _ := json.Marshal(data)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// IsForAgent returns true if this message is targeted at the specified agent.
func (m *Message) IsForAgent(agent string) bool {
	return m.To == "" || m.To == agent
}

// IsDoneSignal returns true if this is a DONE signal.
func (m *Message) IsDoneSignal() bool {
	return m.Type == TypeSignal && m.Signal == SignalDone
}

// IsPassSignal returns true if this is a PASS signal.
func (m *Message) IsPassSignal() bool {
	return m.Type == TypeSignal && m.Signal == SignalPass
}

// IsFailSignal returns true if this is a FAIL signal.
func (m *Message) IsFailSignal() bool {
	return m.Type == TypeSignal && m.Signal == SignalFail
}
