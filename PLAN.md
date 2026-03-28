# AgentPair: Full Go Port of Loop

Port the TypeScript "loop" CLI (~11,000 lines) to Go at `github.com/plexusone/agentpair`.

**Scope**: Full port including Claude + Codex agents, tmux, worktrees, dashboard, and auto-update.

## Overview

AgentPair orchestrates agent-to-agent pair programming between Claude and Codex. One agent works, the other reviews, iterating until task completion.

## Package Structure

```
github.com/plexusone/agentpair/
├── cmd/agentpair/main.go           # CLI entry point
├── internal/
│   ├── agent/
│   │   ├── agent.go                # Agent interface
│   │   ├── claude/
│   │   │   ├── claude.go           # Claude SDK WebSocket server
│   │   │   └── protocol.go         # NDJSON message types
│   │   └── codex/
│   │       ├── codex.go            # Codex App Server client
│   │       └── jsonrpc.go          # JSON-RPC 2.0 types
│   ├── bridge/
│   │   ├── bridge.go               # Agent-to-agent messaging
│   │   ├── message.go              # Message types + SHA256 signatures
│   │   ├── mcp.go                  # MCP server for bridge tools
│   │   └── storage.go              # JSONL file operations
│   ├── loop/
│   │   ├── loop.go                 # Paired loop orchestration
│   │   ├── single.go               # Single-agent mode
│   │   └── state.go                # State machine
│   ├── run/
│   │   ├── run.go                  # Run manager
│   │   ├── manifest.go             # manifest.json handling
│   │   └── transcript.go           # transcript.jsonl audit log
│   ├── review/review.go            # PASS/FAIL parsing, consensus
│   ├── tmux/
│   │   ├── tmux.go                 # tmux session management
│   │   └── layout.go               # Side-by-side panes
│   ├── worktree/worktree.go        # Git worktree automation
│   ├── dashboard/dashboard.go      # Live dashboard UI
│   ├── update/update.go            # Auto-update mechanism
│   └── config/
│       ├── config.go               # Configuration types
│       └── paths.go                # ~/.agentpair/runs paths
└── pkg/jsonl/                      # JSONL utilities
```

## Core Interfaces

```go
// Agent interface - implemented by Claude and Codex
type Agent interface {
    Name() string
    Start(ctx context.Context) error
    Execute(ctx context.Context, msgs []bridge.Message) (*Result, error)
    Stop(ctx context.Context) error
    SessionID() string
}

// Bridge - agent-to-agent communication via JSONL
type Bridge interface {
    Send(ctx context.Context, msg Message) error
    Drain(ctx context.Context, target string) ([]Message, error)
}

// Loop - main orchestration
type Loop interface {
    Run(ctx context.Context) error
    Resume(ctx context.Context, runID string) error
}
```

## Implementation Phases

### Phase 1: Foundation
1. `pkg/jsonl` - JSONL reader/writer utilities
2. `internal/config` - Path management (~/.agentpair/runs/{repoId}/{runId}/)
3. `internal/bridge` - Message passing with SHA256 deduplication
4. `internal/bridge/mcp` - MCP server exposing bridge tools to agents

### Phase 2: State Management
5. `internal/run` - Run manager, manifest.json, transcript.jsonl
6. `internal/review` - PASS/FAIL signal parsing, consensus logic

### Phase 3: Agents
7. `internal/agent/claude` - WebSocket server for Claude SDK
   - NDJSON protocol (stream_event, result, control_request)
   - Background task detection
   - Session persistence and resume
8. `internal/agent/codex` - Codex App Server client
   - JSON-RPC 2.0 over WebSocket
   - Auto-approve commands/file changes
   - Fallback to exec mode

### Phase 4: Orchestration
9. `internal/loop` - Paired loop orchestration
   - Start agents in parallel goroutines
   - Main loop: drain → execute → drain → check done
   - Max iterations, state transitions
10. `internal/loop/single` - Single-agent mode (claude-only, codex-only)

### Phase 5: Workspace
11. `internal/tmux` - tmux session management
    - Side-by-side Claude/Codex panes
    - Session naming, attach/detach
    - Permission prompt handling
12. `internal/worktree` - Git worktree automation
    - Create isolated worktrees per run
    - Cleanup on completion

### Phase 6: CLI & UI
13. `cmd/agentpair` - Cobra CLI with all flags
14. `internal/dashboard` - Live dashboard showing active runs
15. `internal/update` - Auto-update mechanism

## Dependencies

```go
require (
    github.com/spf13/cobra v1.10.2
    github.com/coder/websocket v1.8.14
    github.com/google/uuid v1.6.0
)
```

## Key Features

| Feature | Description |
|---------|-------------|
| Bridge | JSONL file-based messaging with SHA256 dedup |
| MCP Tools | send_to_agent, receive_messages, bridge_status |
| Paired Loop | Claude + Codex work in parallel goroutines |
| Single Mode | --claude-only or --codex-only |
| Review | Both agents review; claudex consensus mode |
| Resume | Continue from --run-id or --session |
| State | submitted → working → reviewing → completed |
| tmux | Side-by-side Claude/Codex terminal panes |
| Worktree | Git worktree isolation per run |
| Dashboard | Live UI showing active runs |
| Auto-update | Binary self-update mechanism |

## CLI Commands

```bash
# Main commands
agentpair                           # Interactive mode
agentpair --prompt "Implement X"    # Start paired session
agentpair --claude-only "Task"      # Claude only
agentpair --codex-only "Task"       # Codex only

# Options
agentpair --agent codex             # Primary worker (default: codex)
agentpair --max-iterations 20       # Max loops (default: 20)
agentpair --proof "run tests"       # Proof requirements
agentpair --review claudex          # Review mode: claude|codex|claudex
agentpair --done "<custom>DONE"     # Custom done signal

# Workspace
agentpair --tmux                    # Side-by-side tmux panes
agentpair --worktree                # Create git worktree

# Resume
agentpair --run-id 5                # Resume by run ID
agentpair --session <id>            # Resume by session ID

# Subcommands
agentpair dashboard                 # Live dashboard
agentpair bridge                    # Bridge status/debug
```

## Verification

1. **Unit tests**: `go test -v -race ./...`
2. **Lint**: `golangci-lint run`
3. **Integration**:
   - Start paired session with mock prompts
   - Verify bridge messages flow between agents
   - Test resume from various states
4. **Manual testing**:
   - Run with real `claude` and `codex` CLIs
   - Test tmux layout renders correctly
   - Verify worktree creation/cleanup
   - Check dashboard displays active runs
