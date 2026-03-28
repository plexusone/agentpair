package loop

import (
	"testing"

	"github.com/plexusone/agentpair/internal/run"
)

func TestMachineTransitions(t *testing.T) {
	tests := []struct {
		name        string
		from        State
		to          State
		shouldError bool
	}{
		{"init to working", StateInit, StateWorking, false},
		{"init to failed", StateInit, StateFailed, false},
		{"working to reviewing", StateWorking, StateReviewing, false},
		{"working to complete", StateWorking, StateComplete, false},
		{"working to failed", StateWorking, StateFailed, false},
		{"reviewing to working", StateReviewing, StateWorking, false},
		{"reviewing to complete", StateReviewing, StateComplete, false},
		{"reviewing to failed", StateReviewing, StateFailed, false},
		{"complete to working", StateComplete, StateWorking, true}, // Invalid
		{"failed to working", StateFailed, StateWorking, true},     // Invalid
		{"init to complete", StateInit, StateComplete, true},       // Invalid
		{"init to reviewing", StateInit, StateReviewing, true},     // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMachine()
			m.current = tt.from

			err := m.Transition(tt.to)
			hasError := err != nil

			if hasError != tt.shouldError {
				t.Errorf("Transition(%s -> %s) error = %v, wantError = %v",
					tt.from, tt.to, err, tt.shouldError)
			}

			if !hasError && m.Current() != tt.to {
				t.Errorf("after transition, current = %s, want %s", m.Current(), tt.to)
			}
		})
	}
}

func TestMachineIsTerminal(t *testing.T) {
	tests := []struct {
		state    State
		terminal bool
	}{
		{StateInit, false},
		{StateWorking, false},
		{StateReviewing, false},
		{StateComplete, true},
		{StateFailed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			m := NewMachine()
			m.current = tt.state

			if got := m.IsTerminal(); got != tt.terminal {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.terminal)
			}
		})
	}
}

func TestMachineHistory(t *testing.T) {
	m := NewMachine()

	// Initial history
	if len(m.History()) != 1 || m.History()[0] != StateInit {
		t.Error("initial history should contain only StateInit")
	}

	// Make transitions
	m.Transition(StateWorking)
	m.Transition(StateReviewing)
	m.Transition(StateComplete)

	history := m.History()
	expected := []State{StateInit, StateWorking, StateReviewing, StateComplete}

	if len(history) != len(expected) {
		t.Fatalf("history length = %d, want %d", len(history), len(expected))
	}

	for i, s := range expected {
		if history[i] != s {
			t.Errorf("history[%d] = %s, want %s", i, history[i], s)
		}
	}
}

func TestStateConversions(t *testing.T) {
	tests := []struct {
		loopState State
		runState  run.State
	}{
		{StateInit, run.StateSubmitted},
		{StateWorking, run.StateWorking},
		{StateReviewing, run.StateReviewing},
		{StateComplete, run.StateCompleted},
		{StateFailed, run.StateFailed},
	}

	for _, tt := range tests {
		t.Run(string(tt.loopState), func(t *testing.T) {
			// Test ToRunState
			if got := tt.loopState.ToRunState(); got != tt.runState {
				t.Errorf("ToRunState() = %s, want %s", got, tt.runState)
			}

			// Test FromRunState
			if got := FromRunState(tt.runState); got != tt.loopState {
				t.Errorf("FromRunState(%s) = %s, want %s", tt.runState, got, tt.loopState)
			}
		})
	}
}

func TestInvalidTransitionError(t *testing.T) {
	err := &InvalidTransitionError{From: StateComplete, To: StateWorking}
	expected := "invalid transition from complete to working"

	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}
