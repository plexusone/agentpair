# Quick Start

This guide walks you through your first AgentPair session.

## Your First Paired Session

Navigate to a project directory and run:

```bash
cd ~/my-project
agentpair --prompt "Add input validation to the user registration form"
```

AgentPair will:

1. Create a new run with a unique ID
2. Start Codex as the primary worker
3. Start Claude as the reviewer
4. Execute the task in a loop until completion

## Understanding the Output

```
2024/03/28 10:00:00 INFO starting agentpair version=dev
2024/03/28 10:00:00 INFO created run run_id=1 prompt="Add input validation..." agent=codex review_mode=claudex
2024/03/28 10:00:00 INFO starting paired-agent mode primary=codex secondary=claude
2024/03/28 10:00:01 INFO iteration started iteration=1 state=working
...
2024/03/28 10:05:00 INFO run complete iterations=3 state=completed
```

## Using tmux Mode

For a visual side-by-side view of both agents working:

```bash
agentpair --tmux --prompt "Implement user authentication"
```

This creates a tmux session with:

- Left pane: Claude
- Right pane: Codex
- Bottom pane: Status dashboard

Attach to watch:

```bash
tmux attach -t agentpair-myproject-1
```

## Single-Agent Mode

For simpler tasks, use one agent:

```bash
# Claude only
agentpair --claude-only "Fix the typo in README.md"

# Codex only
agentpair --codex-only "Add unit tests for utils.go"
```

## Specifying Proof Requirements

Use `--proof` to tell agents how to verify their work:

```bash
agentpair --prompt "Add caching to the API" --proof "Run 'go test ./...' and ensure all tests pass"
```

## Viewing the Dashboard

Monitor all active runs:

```bash
agentpair dashboard
```

## Checking Bridge Status

See message flow between agents:

```bash
# All runs
agentpair bridge

# Specific run
agentpair bridge --run-id 1
```

## Example Workflows

### Feature Implementation

```bash
agentpair --tmux --prompt "Implement OAuth2 login with Google" \
  --proof "Run the app and verify login works"
```

### Bug Fix

```bash
agentpair --prompt "Fix the race condition in cache.go" \
  --proof "Run 'go test -race ./...' with no races detected"
```

### Code Review

```bash
agentpair --claude-only "Review the changes in the last 3 commits for security issues"
```

### Refactoring

```bash
agentpair --worktree --prompt "Refactor the database layer to use repository pattern" \
  --proof "All existing tests must pass"
```

## Next Steps

- [Configuration](configuration.md) — Customize default settings
- [Paired Sessions](../guides/paired-sessions.md) — Deep dive into paired mode
- [Bridge](../concepts/bridge.md) — Understand agent communication
