# AgentPair

Agent-to-agent pair programming between Claude and Codex.

AgentPair orchestrates pair programming sessions between AI agents. One agent works on the task while the other reviews, iterating until completion or max iterations reached.

Inspired by [loop](https://github.com/axeldelafosse/loop), AgentPair is a Go implementation with the same core concepts: paired agent execution, bridge-based communication, and persistent run state.

## Key Features

- **Paired mode** — Claude and Codex work together; one implements, the other reviews
- **Single-agent mode** — Run `--claude-only` or `--codex-only` for simpler tasks
- **Bridge communication** — JSONL-based messaging with SHA256 deduplication
- **MCP integration** — Agents communicate via Model Context Protocol tools
- **Review modes** — `claude`, `codex`, or `claudex` (both review)
- **tmux support** — Side-by-side terminal panes for watching agents work
- **Git worktree isolation** — Each run can use an isolated worktree
- **Session persistence** — Resume runs from any state
- **Live dashboard** — Monitor active runs in real-time
- **Auto-update** — Self-updating binary

## Quick Example

```bash
# Paired session (Codex works, Claude reviews)
agentpair --prompt "Implement a REST API for user management"

# With side-by-side tmux view
agentpair --tmux --prompt "Add OAuth2 authentication"

# Single-agent mode
agentpair --claude-only "Refactor the logging system"
```

## How It Works

```
┌────────────────────────────────────────────┐
│                AgentPair Loop              │
│                                            │
│   ┌─────────┐  bridge.jsonl  ┌─────────┐   │
│   │  Claude │<──────────────>│  Codex  │   │
│   │  (work) │                │(review) │   │
│   └────┬────┘                └────┬────┘   │
│        │                          │        │
│        └──────────┬───────────────┘        │
│                   │                        │
│              State Machine                 │
│   init → working → reviewing → complete    │
└────────────────────────────────────────────┘
```

1. Primary agent (Codex by default) works on the task
2. Secondary agent (Claude by default) reviews the work
3. Agents communicate through bridge messages
4. Loop continues until DONE/PASS signals or max iterations

## Next Steps

- [Installation](getting-started/installation.md) — Get agentpair running
- [Quick Start](getting-started/quick-start.md) — Your first paired session
- [Architecture](concepts/architecture.md) — How it all fits together
