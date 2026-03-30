# AgentPair v0.1.0 Release Notes

**Release Date:** 2024-XX-XX

This is the initial release of AgentPair, a Go implementation of agent-to-agent pair programming orchestration between Claude and Codex.

## Overview

AgentPair enables two AI coding agents to collaborate on development tasks in a structured driver-navigator pattern. One agent implements while the other reviews, iterating until the task is complete or max iterations are reached. This collaborative approach produces higher-quality code through built-in review cycles and feedback loops.

Inspired by [loop](https://github.com/axeldelafosse/loop) (TypeScript).

## Features

### Core Execution Modes

- **Paired Mode (Default)**: Two agents work together—Codex implements, Claude reviews (or vice versa)
- **Single-Agent Mode**: Run with `--claude-only` or `--codex-only` for simpler tasks
- **Configurable Primary Agent**: Use `-a claude` or `-a codex` to choose the implementer

### Communication & Integration

- **JSONL Bridge**: File-based message passing with SHA256 content-addressed deduplication
- **MCP Integration**: Agents communicate via Model Context Protocol tools:
  - `send_to_agent` - Send messages between agents
  - `receive_messages` - Get pending messages
  - `bridge_status` - Check bridge state
- **Message Types**: task, result, review, signal (DONE/PASS/FAIL), chat

### Review Modes

| Mode | Behavior |
|------|----------|
| `claude` | Only Claude reviews; Codex's DONE signal completes the run |
| `codex` | Only Codex reviews; Claude's DONE signal completes the run |
| `claudex` | Both agents review; consensus (both PASS) required for completion |

### Workspace Features

- **tmux Support**: Side-by-side terminal panes for real-time agent monitoring (`--tmux`)
- **Git Worktree Isolation**: Each run can use an isolated worktree for safe experimentation (`--worktree`)
- **Proof Commands**: Specify verification commands (e.g., `--proof "go test ./..."`)

### Session Management

- **Run Persistence**: All runs stored in `~/.agentpair/runs/{repo-id}/{run-id}/`
- **Session Resumption**: Resume interrupted runs with `--run-id` or `--session`
- **Live Dashboard**: Monitor active runs with `agentpair dashboard`
- **Auto-Update**: Self-updating binary via `agentpair update`

## Installation

### From Source

```bash
go install github.com/plexusone/agentpair/cmd/agentpair@v0.1.0
```

### Build Locally

```bash
git clone https://github.com/plexusone/agentpair
cd agentpair
git checkout v0.1.0
go build -o agentpair ./cmd/agentpair
```

## Prerequisites

- Go 1.21+
- `claude` CLI (Claude Code) - installed and authenticated
- `codex` CLI (OpenAI Codex) - installed and authenticated
- Optional: `tmux` for side-by-side terminal view
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

# With verification command
agentpair --prompt "Fix failing tests" --proof "go test ./..."

# With tmux side-by-side view
agentpair --tmux --prompt "Implement OAuth2 authentication"

# With git worktree isolation
agentpair --worktree --prompt "Experimental feature X"

# Resume a previous run
agentpair --run-id 5
```

## CLI Reference

```
agentpair [flags] [prompt]

Core Flags:
  -p, --prompt string       Task prompt (or provide as positional argument)
  -a, --agent string        Primary worker: claude or codex (default "codex")
  -m, --max-iterations int  Maximum loop iterations (default 20)
      --proof string        Proof/verification command
      --review string       Review mode: claude, codex, claudex (default "claudex")
      --done string         Custom done signal (default "DONE")

Mode Flags:
      --claude-only         Run Claude in single-agent mode
      --codex-only          Run Codex in single-agent mode

Workspace Flags:
      --tmux                Use tmux for side-by-side panes
      --worktree            Create git worktree for isolation

Resume Flags:
      --run-id int          Resume by run ID
      --session string      Resume by session ID

Output Flags:
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
│   ├── agent/              # Agent interface & implementations
│   │   ├── claude/         # Claude CLI wrapper (NDJSON protocol)
│   │   └── codex/          # Codex App Server client (JSON-RPC 2.0)
│   ├── bridge/             # JSONL messaging with MCP server
│   ├── loop/               # Orchestration state machine
│   ├── run/                # Run persistence
│   ├── review/             # PASS/FAIL signal parsing
│   ├── tmux/               # tmux session management
│   ├── worktree/           # Git worktree automation
│   ├── dashboard/          # Live dashboard UI
│   ├── config/             # Configuration management
│   ├── logger/             # Structured logging (slog)
│   └── update/             # Auto-update mechanism
└── pkg/jsonl/              # JSONL utilities
```

### State Machine

```
init → working → reviewing → working → ... → complete
                    ↓                           ↑
                  fail ←────────────────────────┘
```

1. **init**: Run created, agents starting
2. **working**: Primary agent executes the task
3. **reviewing**: Secondary agent reviews the work
4. **complete**: Reviewer signals PASS (or both in claudex mode)
5. **failed**: Max iterations reached or agent error

## Configuration

Create `~/.agentpair/config.yaml` for persistent settings:

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

CLI flags take precedence over config file values.

## Run Storage

Runs are persisted to `~/.agentpair/runs/`:

```
~/.agentpair/runs/
└── github-plexusone-myrepo/    # repo-id (normalized from path)
    └── 1/                       # run-id
        ├── manifest.json        # Run metadata and state
        ├── bridge.jsonl         # Agent messages
        └── transcript.jsonl     # Audit log
```

## Known Limitations

- Auto-approve mode is enabled by default for both agents (recommended for VM/container environments)
- Both `claude` and `codex` CLI tools must be installed and authenticated separately
- Integration tests with live agents are not included in this release
- Codex agent requires a running Codex App Server (default: `ws://localhost:3000/ws`)

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| github.com/spf13/cobra | v1.10.2 | CLI framework |
| github.com/coder/websocket | v1.8.14 | WebSocket for Codex |
| github.com/modelcontextprotocol/go-sdk | v1.4.1 | MCP integration |
| go.yaml.in/yaml/v3 | v3.0.4 | YAML configuration |

## Contributors

- Initial implementation

## License

MIT License - see [LICENSE](LICENSE) for details.

## Links

- Repository: https://github.com/plexusone/agentpair
- Issues: https://github.com/plexusone/agentpair/issues
- Documentation: https://pkg.go.dev/github.com/plexusone/agentpair
