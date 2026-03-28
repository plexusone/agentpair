# CLI Reference

Complete command-line reference for AgentPair.

## Synopsis

```bash
agentpair [command] [flags] [prompt]
```

## Commands

### Main Command

```bash
agentpair [flags] [prompt]
```

Run a paired or single-agent session.

### dashboard

```bash
agentpair dashboard
```

Show live dashboard of active runs.

### bridge

```bash
agentpair bridge [--run-id ID]
```

Show bridge status for runs.

### update

```bash
agentpair update
```

Check for and install updates.

### version

```bash
agentpair version
```

Print version information.

## Flags

### Task Configuration

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--prompt` | `-p` | | Task prompt (or provide as positional argument) |
| `--proof` | | | Proof/verification command for task completion |
| `--done` | | `DONE` | Custom done signal to look for |

### Agent Configuration

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--agent` | `-a` | `codex` | Primary worker agent (`claude` or `codex`) |
| `--review` | | `claudex` | Review mode (`claude`, `codex`, or `claudex`) |
| `--claude-only` | | `false` | Run Claude in single-agent mode |
| `--codex-only` | | `false` | Run Codex in single-agent mode |

### Loop Configuration

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--max-iterations` | `-m` | `20` | Maximum loop iterations before stopping |

### Workspace Configuration

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--tmux` | | `false` | Use tmux for side-by-side panes |
| `--worktree` | | `false` | Create git worktree for isolation |

### Resume Configuration

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--run-id` | | `0` | Resume by run ID |
| `--session` | | | Resume by session ID (Claude or Codex) |

### Output Configuration

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--verbose` | `-v` | `false` | Enable verbose output |

## Examples

### Basic Usage

```bash
# Paired session with prompt as argument
agentpair "Implement user authentication"

# Paired session with --prompt flag
agentpair --prompt "Implement user authentication"

# Single-agent mode
agentpair --claude-only "Fix the bug in login.go"
agentpair --codex-only "Add unit tests"
```

### Agent Selection

```bash
# Claude as primary worker (Codex reviews)
agentpair --agent claude --prompt "Complex reasoning task"

# Codex as primary worker (Claude reviews) - default
agentpair --agent codex --prompt "Code generation task"
```

### Review Modes

```bash
# Only Claude reviews
agentpair --review claude --prompt "Task"

# Only Codex reviews
agentpair --review codex --prompt "Task"

# Both review (default)
agentpair --review claudex --prompt "Task"
```

### Proof Requirements

```bash
# With test verification
agentpair --prompt "Add validation" --proof "go test ./..."

# With multiple checks
agentpair --prompt "Security fix" --proof "go test ./... && golangci-lint run"
```

### Iteration Control

```bash
# Increase iterations for complex tasks
agentpair --max-iterations 50 --prompt "Major refactoring"

# Limit iterations for quick tasks
agentpair --max-iterations 5 --prompt "Simple fix"
```

### Workspace Options

```bash
# With tmux side-by-side view
agentpair --tmux --prompt "Task"

# With git worktree isolation
agentpair --worktree --prompt "Experimental feature"

# Both
agentpair --tmux --worktree --prompt "Risky changes"
```

### Resuming

```bash
# Resume by run ID
agentpair --run-id 7

# Resume by session ID
agentpair --session abc123-def456
```

### Combined Options

```bash
# Full-featured run
agentpair --tmux --worktree \
  --agent claude \
  --review claudex \
  --max-iterations 30 \
  --proof "npm test && npm run lint" \
  --verbose \
  --prompt "Implement OAuth2 with Google"
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (see stderr for details) |

## Environment

AgentPair does not currently read environment variables. Use the config file or CLI flags.

## Files

| Path | Purpose |
|------|---------|
| `~/.agentpair/config.yaml` | Global configuration |
| `~/.agentpair/runs/` | Run persistence |

## See Also

- [Configuration](config.md) — Config file reference
- [Bridge Messages](messages.md) — Message format reference
