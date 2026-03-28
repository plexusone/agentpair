package bridge

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNewMessage(t *testing.T) {
	msg := NewMessage(TypeTask, "claude", "codex", "implement feature X")

	if msg.Type != TypeTask {
		t.Errorf("unexpected type: %s", msg.Type)
	}
	if msg.From != "claude" {
		t.Errorf("unexpected from: %s", msg.From)
	}
	if msg.To != "codex" {
		t.Errorf("unexpected to: %s", msg.To)
	}
	if msg.Content != "implement feature X" {
		t.Errorf("unexpected content: %s", msg.Content)
	}
	if msg.ID == "" {
		t.Error("ID should be generated")
	}
}

func TestMessageIsForAgent(t *testing.T) {
	tests := []struct {
		name     string
		to       string
		agent    string
		expected bool
	}{
		{"targeted message", "claude", "claude", true},
		{"targeted message wrong agent", "claude", "codex", false},
		{"broadcast message", "", "claude", true},
		{"broadcast message codex", "", "codex", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(TypeTask, "system", tt.to, "test")
			if got := msg.IsForAgent(tt.agent); got != tt.expected {
				t.Errorf("IsForAgent(%s) = %v, want %v", tt.agent, got, tt.expected)
			}
		})
	}
}

func TestSignalMessages(t *testing.T) {
	done := NewSignalMessage("claude", SignalDone, "task complete")
	if !done.IsDoneSignal() {
		t.Error("expected IsDoneSignal to be true")
	}

	pass := NewSignalMessage("codex", SignalPass, "looks good")
	if !pass.IsPassSignal() {
		t.Error("expected IsPassSignal to be true")
	}

	fail := NewSignalMessage("claude", SignalFail, "needs changes")
	if !fail.IsFailSignal() {
		t.Error("expected IsFailSignal to be true")
	}
}

func TestBridgeSendAndDrain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bridge.jsonl")

	b := New(path, 1)
	if err := b.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer b.Close()

	ctx := context.Background()

	// Send messages
	msg1 := NewTaskMessage("system", "claude", "task 1")
	sent, err := b.Send(ctx, msg1)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !sent {
		t.Error("first message should be sent")
	}

	msg2 := NewResultMessage("claude", "codex", "result 1")
	sent, err = b.Send(ctx, msg2)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !sent {
		t.Error("second message should be sent")
	}

	// Try duplicate
	sent, err = b.Send(ctx, msg1)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if sent {
		t.Error("duplicate message should not be sent")
	}

	// Drain for claude
	msgs, err := b.Drain(ctx, "claude", make(map[string]bool))
	if err != nil {
		t.Fatalf("Drain failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message for claude, got %d", len(msgs))
	}

	// Drain for codex
	msgs, err = b.Drain(ctx, "codex", make(map[string]bool))
	if err != nil {
		t.Fatalf("Drain failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message for codex, got %d", len(msgs))
	}
}

func TestBridgeStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bridge.jsonl")

	b := New(path, 1)
	if err := b.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer b.Close()

	ctx := context.Background()

	// Send various messages
	b.Send(ctx, NewTaskMessage("system", "claude", "task"))
	b.Send(ctx, NewResultMessage("claude", "codex", "result"))
	b.Send(ctx, NewSignalMessage("codex", SignalPass, "approved"))
	b.Send(ctx, NewSignalMessage("claude", SignalDone, "complete"))

	status := b.Status()

	if status.TotalMessages != 4 {
		t.Errorf("expected 4 messages, got %d", status.TotalMessages)
	}
	if status.PassCount != 1 {
		t.Errorf("expected 1 pass, got %d", status.PassCount)
	}
	if !status.HasDoneSignal {
		t.Error("expected HasDoneSignal to be true")
	}
}
