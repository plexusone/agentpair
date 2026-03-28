# AgentPair

Agent-to-agent pair programming between Claude and Codex.

AgentPair orchestrates pair programming sessions between AI agents. One agent works on the task while the other reviews, iterating until completion or max iterations reached.

## Features

- 🤝 **Paired mode**: Claude and Codex work together, one implements while the other reviews
- 👤 **Single-agent mode**: Run `--claude-only` or `--codex-only` for simpler tasks
- 🌉 **Bridge communication**: JSONL-based messaging with SHA256 deduplication
- 🔌 **MCP integration**: Agents communicate via Model Context Protocol tools
- 🔍 **Review modes**: `claude`, `codex`, or `claudex` (both review)
- 🖥️ **tmux support**: Side-by-side terminal panes for watching agents work
- 🌲 **Git worktree isolation**: Each run can use an isolated worktree
- 💾 **Session persistence**: Resume runs from any state
- 📊 **Live dashboard**: Monitor active runs in real-time
- 🔄 **Auto-update**: Self-updating binary

## Installation

```bash
go install github.com/plexusone/agentpair@latest
```

Or build from source:

```bash
git clone https://github.com/plexusone/agentpair
cd agentpair
go build -o agentpair ./cmd/agentpair
```

## Prerequisites

- Go 1.21+
- `claude` CLI (Claude Code)
- `codex` CLI (OpenAI Codex)
- Optional: `tmux` for side-by-side view
- Optional: Git for worktree isolation

## Quick Start

```bash
# Paired session (Codex works, Claude reviews)
agentpair --prompt "Implement a REST API for user management"

# Claude as primary worker
agentpair --agent claude --prompt "Add unit tests for the auth module"

# Single-agent mode
agentpair --claude-only "Refactor the logging system"
agentpair --codex-only "Fix the memory leak in cache.go"

# With tmux side-by-side view
agentpair --tmux --prompt "Implement OAuth2 authentication"

# With git worktree isolation
agentpair --worktree --prompt "Experimental feature X"
```

## Usage

```
agentpair [flags] [prompt]

Flags:
  -p, --prompt string       Task prompt (or provide as argument)
  -a, --agent string        Primary worker: claude or codex (default "codex")
  -m, --max-iterations int  Maximum loop iterations (default 20)
      --proof string        Proof/verification command (e.g., "go test ./...")
      --review string       Review mode: claude, codex, claudex (default "claudex")
      --done string         Custom done signal (default "DONE")
      --claude-only         Run Claude in single-agent mode
      --codex-only          Run Codex in single-agent mode
      --tmux                Use tmux for side-by-side panes
      --worktree            Create git worktree for isolation
      --run-id int          Resume by run ID
      --session string      Resume by session ID
  -v, --verbose             Verbose output

Commands:
  dashboard    Show live dashboard of active runs
  bridge       Show bridge status for a run
  update       Check for and install updates
  version      Print version information
```

## Architecture

```
github.com/plexusone/agentpair/
├── cmd/agentpair/          # CLI entry point
├── internal/
│   ├── agent/              # Agent interface
│   │   ├── claude/         # Claude CLI wrapper (NDJSON protocol)
│   │   └── codex/          # Codex App Server client (JSON-RPC 2.0)
│   ├── bridge/             # Agent-to-agent messaging
│   │   ├── bridge.go       # JSONL storage with SHA256 dedup
│   │   └── server.go       # MCP server for bridge tools
│   ├── loop/               # Orchestration state machine
│   ├── run/                # Run persistence (~/.agentpair/runs/)
│   ├── review/             # PASS/FAIL signal parsing
│   ├── tmux/               # tmux session management
│   ├── worktree/           # Git worktree automation
│   ├── dashboard/          # Live dashboard UI
│   ├── config/             # Configuration and paths
│   ├── logger/             # Structured logging (slog)
│   └── update/             # Auto-update mechanism
└── pkg/jsonl/              # JSONL utilities
```

## How It Works

### State Machine

```
init → working → reviewing → working → ... → complete
                    ↓                           ↑
                  fail ←────────────────────────┘
```

1. **Init**: Run created, agents starting
2. **Working**: Primary agent executes the task
3. **Reviewing**: Secondary agent reviews the work
4. **Complete**: Both agents signal DONE/PASS
5. **Failed**: Max iterations reached or agent error

### Bridge Communication

Agents communicate through a JSONL bridge file with MCP tools:

- `send_to_agent`: Send a message to the other agent
- `receive_messages`: Get pending messages
- `bridge_status`: Check bridge state

Message types:

- `task`: Initial task or follow-up work
- `result`: Work output
- `review`: Review feedback
- `signal`: Control signals (DONE, PASS, FAIL)
- `chat`: Free-form discussion

### Review Modes

| Mode | Behavior |
|------|----------|
| `claude` | Only Claude reviews |
| `codex` | Only Codex reviews |
| `claudex` | Both review; consensus required |

## Configuration

Create `~/.agentpair/config.yaml`:

```yaml
agent: codex
max_iterations: 20
review_mode: claudex
done_signal: DONE
use_tmux: false
use_worktree: false
verbose: false
timeout: 2h
```

CLI flags override config file values.

## Run Persistence

Runs are stored in `~/.agentpair/runs/{repo-id}/{run-id}/`:

```
~/.agentpair/runs/
└── github-plexusone-myrepo/
    └── 1/
        ├── manifest.json      # Run metadata
        ├── bridge.jsonl       # Agent messages
        └── transcript.jsonl   # Audit log
```

Resume a run:

```bash
agentpair --run-id 5
agentpair --session abc123
```

## Development

```bash
# Run tests
go test -v ./...

# Run tests with race detection
go test -race ./...

# Lint
golangci-lint run

# Build
go build -o agentpair ./cmd/agentpair
```

### Test Coverage

| Package | Status |
|---------|--------|
| pkg/jsonl | ✓ |
| internal/bridge | ✓ |
| internal/config | ✓ |
| internal/logger | ✓ |
| internal/loop | ✓ |
| internal/review | ✓ |
| internal/run | ✓ |
| internal/tmux | ✓ |
| internal/worktree | ✓ |

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [websocket](https://github.com/coder/websocket) - WebSocket client/server
- [go-sdk](https://github.com/modelcontextprotocol/go-sdk) - MCP SDK
- [yaml](https://go.yaml.in/yaml/v3) - YAML parsing

## License

MIT
