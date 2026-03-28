// Package bridge provides agent-to-agent communication via JSONL files.
package bridge

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/plexusone/agentpair/internal/logger"
)

// Bridge handles message passing between agents with SHA256 deduplication.
type Bridge struct {
	storage *Storage
	seen    map[string]bool
	mu      sync.RWMutex
	runID   int
	log     *slog.Logger
}

// New creates a new bridge with the given storage path.
func New(storagePath string, runID int) *Bridge {
	return &Bridge{
		storage: NewStorage(storagePath),
		seen:    make(map[string]bool),
		runID:   runID,
		log:     logger.WithComponent(logger.WithRunID(slog.Default(), runID), "bridge"),
	}
}

// SetLogger sets the logger for the bridge.
func (b *Bridge) SetLogger(log *slog.Logger) {
	b.log = logger.WithComponent(log, "bridge")
}

// Open initializes the bridge, loading existing messages for deduplication.
func (b *Bridge) Open() error {
	if err := b.storage.Open(); err != nil {
		return err
	}

	// Load existing message IDs for deduplication
	msgs, err := b.storage.ReadAll()
	if err != nil {
		return err
	}

	b.mu.Lock()
	for _, msg := range msgs {
		b.seen[msg.ID] = true
	}
	b.mu.Unlock()

	b.log.Debug("bridge opened", "existing_messages", len(msgs))
	return nil
}

// Close closes the bridge storage.
func (b *Bridge) Close() error {
	return b.storage.Close()
}

// Send sends a message through the bridge. Returns false if the message is a duplicate.
func (b *Bridge) Send(ctx context.Context, msg *Message) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	msg.RunID = b.runID

	b.mu.Lock()
	if b.seen[msg.ID] {
		b.mu.Unlock()
		b.log.Debug("duplicate message ignored", "id", msg.ID)
		return false, nil // Duplicate
	}
	b.seen[msg.ID] = true
	b.mu.Unlock()

	if err := b.storage.Append(msg); err != nil {
		// Rollback seen on error
		b.mu.Lock()
		delete(b.seen, msg.ID)
		b.mu.Unlock()
		b.log.Error("failed to append message", "error", err, "id", msg.ID)
		return false, err
	}

	b.log.Debug("message sent",
		"id", msg.ID,
		"type", msg.Type,
		"from", msg.From,
		"to", msg.To)
	return true, nil
}

// Drain retrieves all unprocessed messages for a target agent.
// Messages are marked as seen to prevent re-delivery.
func (b *Bridge) Drain(ctx context.Context, target string, seenIDs map[string]bool) ([]*Message, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	all, err := b.storage.ReadAll()
	if err != nil {
		return nil, err
	}

	var messages []*Message
	for _, msg := range all {
		if msg.IsForAgent(target) && !seenIDs[msg.ID] {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// DrainNew retrieves messages for a target agent since a given message ID.
func (b *Bridge) DrainNew(ctx context.Context, target string, sinceID string) ([]*Message, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	all, err := b.storage.ReadAll()
	if err != nil {
		return nil, err
	}

	// Find the index of sinceID
	startIdx := 0
	if sinceID != "" {
		for i, msg := range all {
			if msg.ID == sinceID {
				startIdx = i + 1
				break
			}
		}
	}

	var messages []*Message
	for i := startIdx; i < len(all); i++ {
		msg := all[i]
		if msg.IsForAgent(target) {
			messages = append(messages, msg)
		}
	}

	b.log.Debug("drained messages",
		"target", target,
		"count", len(messages),
		"since_id", sinceID)
	return messages, nil
}

// GetByID retrieves a message by its ID.
func (b *Bridge) GetByID(id string) (*Message, error) {
	all, err := b.storage.ReadAll()
	if err != nil {
		return nil, err
	}

	for _, msg := range all {
		if msg.ID == id {
			return msg, nil
		}
	}
	return nil, fmt.Errorf("message not found: %s", id)
}

// Status returns the current bridge status.
func (b *Bridge) Status() *Status {
	all, _ := b.storage.ReadAll()

	status := &Status{
		TotalMessages: len(all),
		ByAgent:       make(map[string]int),
		ByType:        make(map[MessageType]int),
	}

	for _, msg := range all {
		status.ByAgent[msg.From]++
		status.ByType[msg.Type]++

		if msg.IsDoneSignal() {
			status.HasDoneSignal = true
		}
		if msg.IsPassSignal() {
			status.PassCount++
		}
		if msg.IsFailSignal() {
			status.FailCount++
		}
	}

	return status
}

// Status contains bridge status information.
type Status struct {
	TotalMessages int
	ByAgent       map[string]int
	ByType        map[MessageType]int
	HasDoneSignal bool
	PassCount     int
	FailCount     int
}

// String returns a human-readable status string.
func (s *Status) String() string {
	return fmt.Sprintf("messages=%d done=%v pass=%d fail=%d",
		s.TotalMessages, s.HasDoneSignal, s.PassCount, s.FailCount)
}
