# Architecture

AgentPair is organized into focused packages with clear responsibilities.

## Package Structure

```
github.com/plexusone/agentpair/
├── cmd/agentpair/          # CLI entry point
├── internal/
│   ├── agent/              # Agent interface and implementations
│   │   ├── claude/         # Claude CLI wrapper
│   │   └── codex/          # Codex App Server client
│   ├── bridge/             # Agent-to-agent messaging
│   ├── loop/               # Orchestration state machine
│   ├── run/                # Run persistence
│   ├── review/             # Signal parsing
│   ├── tmux/               # Terminal multiplexer
│   ├── worktree/           # Git worktree management
│   ├── dashboard/          # Live monitoring UI
│   ├── config/             # Configuration handling
│   ├── logger/             # Structured logging
│   └── update/             # Auto-update mechanism
└── pkg/jsonl/              # JSONL utilities
```

## Core Components

### Agent Interface

The `Agent` interface defines how AgentPair interacts with AI agents:

```go
type Agent interface {
    Name() string
    Start(ctx context.Context) error
    Execute(ctx context.Context, msgs []*bridge.Message) (*Result, error)
    Stop(ctx context.Context) error
    SessionID() string
    IsRunning() bool
    SetMCPServerAddr(addr string)
}
```

Two implementations exist:

- **Claude** — Wraps the `claude` CLI using NDJSON protocol
- **Codex** — Connects to Codex App Server via JSON-RPC 2.0

### Bridge

The bridge handles message passing between agents:

```
┌─────────┐                      ┌─────────┐
│  Claude │                      │  Codex  │
└────┬────┘                      └────┬────┘
     │                                │
     │  send_to_agent("codex", ...)   │
     ├───────────────────────────────►│
     │                                │
     │◄───────────────────────────────┤
     │  send_to_agent("claude", ...)  │
     │                                │
     ▼                                ▼
┌─────────────────────────────────────────┐
│            bridge.jsonl                  │
│  {"id":"abc","from":"claude","to":"codex"...}
│  {"id":"def","from":"codex","to":"claude"...}
└─────────────────────────────────────────┘
```

Key features:

- **JSONL storage** — Messages persisted to disk
- **SHA256 deduplication** — Prevents duplicate processing
- **MCP server** — Exposes bridge tools to agents

### Loop

The orchestration loop manages the agent lifecycle:

```go
type Loop struct {
    config    *config.Config
    run       *run.Run
    primary   agent.Agent
    secondary agent.Agent
    machine   *Machine
}
```

Main flow:

1. Start both agents
2. Execute primary agent with pending messages
3. Drain bridge, execute secondary agent
4. Check for completion signals
5. Repeat until done or max iterations

### Run Manager

Persists run state to disk:

```
~/.agentpair/runs/{repo-id}/{run-id}/
├── manifest.json      # Run metadata
├── bridge.jsonl       # Agent messages
└── transcript.jsonl   # Audit log
```

The manifest tracks:

- Run ID and prompt
- Agent configuration
- Current state and iteration
- Session IDs for resume
- Timestamps

## Data Flow

```
User
  │
  ▼
┌─────────────────┐
│  cmd/agentpair  │  Parse flags, load config
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   run.Manager   │  Create/load run
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     Loop        │  Orchestrate agents
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌───────┐ ┌───────┐
│Claude │ │ Codex │  Execute tasks
└───┬───┘ └───┬───┘
    │         │
    └────┬────┘
         │
         ▼
┌─────────────────┐
│     Bridge      │  Message passing
└─────────────────┘
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/coder/websocket` | WebSocket client/server |
| `github.com/modelcontextprotocol/go-sdk` | MCP SDK |
| `go.yaml.in/yaml/v3` | YAML configuration |

## Design Decisions

### Why JSONL for Bridge?

- Simple, append-only format
- Human-readable for debugging
- Easy to parse and stream
- Survives crashes (no corruption)

### Why SHA256 for Deduplication?

- Content-addressable messages
- Deterministic IDs across restarts
- No coordination needed between agents

### Why Separate Agent Implementations?

- Different protocols (NDJSON vs JSON-RPC)
- Different session management
- Different tool capabilities
- Easier to add new agents

## Next Steps

- [State Machine](state-machine.md) — Loop state transitions
- [Bridge](bridge.md) — Message format and tools
- [MCP Integration](mcp.md) — Model Context Protocol
