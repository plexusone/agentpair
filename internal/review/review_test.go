package review

import (
	"testing"

	"github.com/plexusone/agentpair/internal/bridge"
)

func TestParserParse(t *testing.T) {
	parser := NewParser("DONE")

	tests := []struct {
		name     string
		input    string
		expected Signal
		explicit bool
	}{
		{"explicit PASS", "PASS - looks good", SignalPass, true},
		{"explicit FAIL", "FAIL - needs changes", SignalFail, true},
		{"lowercase pass", "pass: all tests succeed", SignalPass, true},
		{"lowercase fail", "fail: tests broken", SignalFail, true},
		{"approval phrase", "looks good to me", SignalPass, false},
		{"rejection phrase", "needs work on error handling", SignalFail, false},
		{"no signal", "here is some output", SignalNone, false},
		{"LGTM", "LGTM ship it", SignalPass, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.input)
			if result.Signal != tt.expected {
				t.Errorf("Parse(%q) signal = %v, want %v", tt.input, result.Signal, tt.expected)
			}
			if result.Explicit != tt.explicit {
				t.Errorf("Parse(%q) explicit = %v, want %v", tt.input, result.Explicit, tt.explicit)
			}
		})
	}
}

func TestParserIsDone(t *testing.T) {
	tests := []struct {
		name       string
		doneSignal string
		text       string
		expected   bool
	}{
		{"standard DONE", "DONE", "Task is DONE", true},
		{"lowercase done", "DONE", "task is done", true},
		{"custom signal", "COMPLETE", "Task COMPLETE", true},
		{"no signal", "DONE", "Task in progress", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.doneSignal)
			if got := parser.IsDone(tt.text); got != tt.expected {
				t.Errorf("IsDone(%q) = %v, want %v", tt.text, got, tt.expected)
			}
		})
	}
}

func TestParserParseMessage(t *testing.T) {
	parser := NewParser("DONE")

	// Test signal message
	signalMsg := bridge.NewSignalMessage("claude", bridge.SignalPass, "approved")
	result := parser.ParseMessage(signalMsg)

	if result.Signal != SignalPass {
		t.Errorf("expected SignalPass, got %v", result.Signal)
	}
	if result.Agent != "claude" {
		t.Errorf("expected agent claude, got %s", result.Agent)
	}
	if !result.Explicit {
		t.Error("signal message should be explicit")
	}
}

func TestConsensusCalculate(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		claudeSignal Signal
		codexSignal  Signal
		expected     Signal
		unanimous    bool
	}{
		// Claude mode - Claude's decision is final
		{"claude mode pass", "claude", SignalPass, SignalNone, SignalPass, true},
		{"claude mode fail", "claude", SignalFail, SignalPass, SignalFail, true},

		// Codex mode - Codex's decision is final
		{"codex mode pass", "codex", SignalNone, SignalPass, SignalPass, true},
		{"codex mode fail", "codex", SignalPass, SignalFail, SignalFail, true},

		// Claudex mode - Both must agree
		{"claudex both pass", "claudex", SignalPass, SignalPass, SignalPass, true},
		{"claudex claude fail", "claudex", SignalFail, SignalPass, SignalFail, false},
		{"claudex codex fail", "claudex", SignalPass, SignalFail, SignalFail, false},
		{"claudex both fail", "claudex", SignalFail, SignalFail, SignalFail, true},
		{"claudex pending", "claudex", SignalPass, SignalNone, SignalPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consensus := NewConsensus(tt.mode)

			claudeResult := &Result{Signal: tt.claudeSignal, Agent: "claude"}
			codexResult := &Result{Signal: tt.codexSignal, Agent: "codex"}

			result := consensus.Calculate(claudeResult, codexResult)

			if result.Signal != tt.expected {
				t.Errorf("Calculate() signal = %v, want %v", result.Signal, tt.expected)
			}
			if result.Unanimous != tt.unanimous {
				t.Errorf("Calculate() unanimous = %v, want %v", result.Unanimous, tt.unanimous)
			}
		})
	}
}

func TestContainsDoneSignal(t *testing.T) {
	msgs := []*bridge.Message{
		bridge.NewTaskMessage("system", "claude", "implement feature"),
		bridge.NewResultMessage("claude", "codex", "here is the result"),
	}

	if ContainsDoneSignal(msgs, "DONE") {
		t.Error("should not contain done signal")
	}

	// Add done signal
	msgs = append(msgs, bridge.NewSignalMessage("claude", bridge.SignalDone, "complete"))

	if !ContainsDoneSignal(msgs, "DONE") {
		t.Error("should contain done signal")
	}
}
