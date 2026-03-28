package loop

import (
	"github.com/plexusone/agentpair/internal/run"
)

// State represents the loop state machine.
type State string

const (
	StateInit      State = "init"
	StateWorking   State = "working"
	StateReviewing State = "reviewing"
	StateComplete  State = "complete"
	StateFailed    State = "failed"
)

// Machine manages state transitions.
type Machine struct {
	current State
	history []State
}

// NewMachine creates a new state machine starting in init state.
func NewMachine() *Machine {
	return &Machine{
		current: StateInit,
		history: []State{StateInit},
	}
}

// Current returns the current state.
func (m *Machine) Current() State {
	return m.current
}

// Transition moves to a new state.
func (m *Machine) Transition(newState State) error {
	// Validate transition
	if !m.canTransition(newState) {
		return &InvalidTransitionError{
			From: m.current,
			To:   newState,
		}
	}

	m.history = append(m.history, newState)
	m.current = newState
	return nil
}

// canTransition checks if a transition is valid.
func (m *Machine) canTransition(newState State) bool {
	validTransitions := map[State][]State{
		StateInit:      {StateWorking, StateFailed},
		StateWorking:   {StateReviewing, StateComplete, StateFailed},
		StateReviewing: {StateWorking, StateComplete, StateFailed},
		StateComplete:  {}, // Terminal state
		StateFailed:    {}, // Terminal state
	}

	allowed, ok := validTransitions[m.current]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == newState {
			return true
		}
	}
	return false
}

// IsTerminal returns true if in a terminal state.
func (m *Machine) IsTerminal() bool {
	return m.current == StateComplete || m.current == StateFailed
}

// History returns the state transition history.
func (m *Machine) History() []State {
	return m.history
}

// InvalidTransitionError is returned for invalid state transitions.
type InvalidTransitionError struct {
	From State
	To   State
}

func (e *InvalidTransitionError) Error() string {
	return "invalid transition from " + string(e.From) + " to " + string(e.To)
}

// ToRunState converts loop state to run state.
func (s State) ToRunState() run.State {
	switch s {
	case StateInit:
		return run.StateSubmitted
	case StateWorking:
		return run.StateWorking
	case StateReviewing:
		return run.StateReviewing
	case StateComplete:
		return run.StateCompleted
	case StateFailed:
		return run.StateFailed
	default:
		return run.StateSubmitted
	}
}

// FromRunState converts run state to loop state.
func FromRunState(rs run.State) State {
	switch rs {
	case run.StateSubmitted:
		return StateInit
	case run.StateWorking:
		return StateWorking
	case run.StateReviewing:
		return StateReviewing
	case run.StateCompleted:
		return StateComplete
	case run.StateFailed, run.StateCancelled:
		return StateFailed
	default:
		return StateInit
	}
}
