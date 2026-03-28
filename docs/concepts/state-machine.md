# State Machine

The loop state machine controls the flow of agent execution.

## States

| State | Description |
|-------|-------------|
| `init` | Run created, agents starting |
| `working` | Primary agent executing task |
| `reviewing` | Secondary agent reviewing work |
| `complete` | Task finished successfully |
| `failed` | Error or max iterations reached |

## Transitions

```
                    ┌──────────────────────────────────┐
                    │                                  │
                    ▼                                  │
┌──────┐      ┌─────────┐      ┌───────────┐      ┌───┴─────┐
│ init │─────►│ working │─────►│ reviewing │─────►│complete │
└──┬───┘      └────┬────┘      └─────┬─────┘      └─────────┘
   │               │                 │
   │               │                 │
   │               ▼                 │
   │          ┌────────┐◄────────────┘
   └─────────►│ failed │
              └────────┘
```

### Valid Transitions

| From | To | Trigger |
|------|-----|---------|
| `init` | `working` | Agents started successfully |
| `init` | `failed` | Agent failed to start |
| `working` | `reviewing` | Primary agent finished iteration |
| `working` | `complete` | Primary agent signaled DONE |
| `working` | `failed` | Agent error or max iterations |
| `reviewing` | `working` | Review requires more work |
| `reviewing` | `complete` | Review passed (PASS signal) |
| `reviewing` | `failed` | Unrecoverable review failure |

## Completion Conditions

The loop completes when:

1. **DONE signal** — Agent outputs the done signal (default: `DONE`)
2. **PASS signal** — Reviewing agent approves with `PASS`
3. **Consensus** — In `claudex` mode, both agents signal completion

## Failure Conditions

The loop fails when:

1. **Max iterations** — Reached `--max-iterations` limit
2. **Agent error** — Agent process crashed or returned error
3. **FAIL signal** — Agent explicitly fails with `FAIL`
4. **Context cancelled** — User interrupted (Ctrl+C)

## Implementation

```go
type State string

const (
    StateInit      State = "init"
    StateWorking   State = "working"
    StateReviewing State = "reviewing"
    StateComplete  State = "complete"
    StateFailed    State = "failed"
)

type Machine struct {
    current State
    history []State
}

func (m *Machine) Transition(newState State) error {
    if !m.canTransition(newState) {
        return &InvalidTransitionError{From: m.current, To: newState}
    }
    m.history = append(m.history, newState)
    m.current = newState
    return nil
}
```

## Run State Mapping

Loop states map to run states for persistence:

| Loop State | Run State |
|------------|-----------|
| `init` | `submitted` |
| `working` | `working` |
| `reviewing` | `reviewing` |
| `complete` | `completed` |
| `failed` | `failed` |

## Iteration Flow

Each iteration follows this pattern:

```
1. Check state is not terminal
2. Increment iteration counter
3. Drain messages for primary agent
4. Execute primary agent
5. Process result (check signals)
6. If not done, transition to reviewing
7. Drain messages for secondary agent
8. Execute secondary agent
9. Process review result
10. Transition back to working or complete
```

## Signal Detection

Signals are detected in agent output:

```go
type Result struct {
    Output string
    Done   bool  // DONE signal found
    Pass   bool  // PASS signal found
    Fail   bool  // FAIL signal found
}
```

The `review` package parses agent output for these signals.

## Resuming

When resuming a run, the state machine:

1. Loads the last known state from manifest
2. Converts run state to loop state
3. Continues from that point

```go
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
    default:
        return StateFailed
    }
}
```

## Next Steps

- [Bridge](bridge.md) — How agents communicate
- [Paired Sessions](../guides/paired-sessions.md) — Running the full loop
