# Configuration

AgentPair can be configured via command-line flags or a config file. CLI flags always take precedence.

## Config File

Create `~/.agentpair/config.yaml`:

```yaml
# Primary worker agent: claude or codex
agent: codex

# Maximum loop iterations before stopping
max_iterations: 20

# Review mode: claude, codex, or claudex
review_mode: claudex

# Custom done signal to look for
done_signal: DONE

# Proof requirements for task verification
proof: ""

# Enable tmux side-by-side view
use_tmux: false

# Enable git worktree isolation
use_worktree: false

# Enable verbose logging
verbose: false

# Maximum run duration
timeout: 2h
```

JSON format is also supported (`~/.agentpair/config.json`).

## CLI Flags

Flags override config file values:

```bash
agentpair --agent claude --max-iterations 10 --prompt "Task"
```

### All Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--prompt` | `-p` | | Task prompt |
| `--agent` | `-a` | `codex` | Primary worker (claude/codex) |
| `--max-iterations` | `-m` | `20` | Maximum loop iterations |
| `--proof` | | | Proof/verification command |
| `--review` | | `claudex` | Review mode |
| `--done` | | `DONE` | Custom done signal |
| `--claude-only` | | `false` | Single-agent Claude mode |
| `--codex-only` | | `false` | Single-agent Codex mode |
| `--tmux` | | `false` | Use tmux panes |
| `--worktree` | | `false` | Use git worktree |
| `--run-id` | | `0` | Resume by run ID |
| `--session` | | | Resume by session ID |
| `--verbose` | `-v` | `false` | Verbose output |

## Environment Variables

Environment variables are not currently supported. Use the config file or CLI flags.

## Directory Structure

AgentPair stores data in `~/.agentpair/`:

```
~/.agentpair/
├── config.yaml           # Global configuration
└── runs/
    └── {repo-id}/
        └── {run-id}/
            ├── manifest.json    # Run metadata
            ├── bridge.jsonl     # Agent messages
            └── transcript.jsonl # Audit log
```

## Repo ID

The repo ID is derived from the Git remote URL or directory path:

- `github.com/user/repo` → `github-user-repo`
- `/home/user/myproject` → `home-user-myproject`

## Example Configurations

### CI/CD Pipeline

```yaml
agent: codex
max_iterations: 50
review_mode: claudex
verbose: true
timeout: 4h
```

### Local Development

```yaml
agent: claude
max_iterations: 10
use_tmux: true
verbose: false
```

### Safety-Focused

```yaml
agent: codex
max_iterations: 5
review_mode: claudex
use_worktree: true
proof: "go test ./... && golangci-lint run"
```

## Next Steps

- [Architecture](../concepts/architecture.md) — Understand the system design
- [Review Modes](../guides/paired-sessions.md#review-modes) — Choose the right review strategy
