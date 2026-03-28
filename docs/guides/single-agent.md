# Single-Agent Mode

For simpler tasks, run a single agent without the review loop.

## When to Use

Single-agent mode is ideal for:

- Simple bug fixes
- Documentation updates
- Quick refactoring
- Tasks that don't need review
- Testing agent capabilities

## Usage

### Claude Only

```bash
agentpair --claude-only "Fix the typo in README.md"
```

Or with the prompt flag:

```bash
agentpair --claude-only --prompt "Add JSDoc comments to utils.ts"
```

### Codex Only

```bash
agentpair --codex-only "Update the version number to 2.0.0"
```

## Differences from Paired Mode

| Aspect | Paired Mode | Single-Agent Mode |
|--------|-------------|-------------------|
| Agents | 2 (worker + reviewer) | 1 |
| Review | Yes | No |
| Iterations | Multiple (work→review→work) | Single pass |
| Completion | PASS from reviewer | DONE from agent |
| Complexity | Higher overhead | Lightweight |

## State Machine

Single-agent mode uses a simplified state machine:

```
init → working → complete
          ↓
        failed
```

No reviewing state — the agent works until done or max iterations.

## Examples

### Quick Fixes

```bash
# Fix a specific bug
agentpair --codex-only "Fix the null pointer in handleUser()"

# Update configuration
agentpair --claude-only "Change the default timeout to 30 seconds"
```

### Documentation

```bash
# Generate docs
agentpair --claude-only "Write API documentation for the auth module"

# Update README
agentpair --codex-only "Add installation instructions for Windows"
```

### Code Generation

```bash
# Generate boilerplate
agentpair --codex-only "Create a new React component called UserProfile"

# Add tests
agentpair --claude-only "Write unit tests for the validation functions"
```

### Analysis Tasks

```bash
# Code review
agentpair --claude-only "Review the security of the login handler"

# Performance analysis
agentpair --codex-only "Identify performance bottlenecks in the database queries"
```

## With Proof Requirements

You can still use `--proof` in single-agent mode:

```bash
agentpair --claude-only "Add input validation" \
  --proof "Run 'npm test' and ensure all tests pass"
```

The agent will verify against the proof before signaling done.

## With tmux

Single-agent mode works with tmux:

```bash
agentpair --claude-only --tmux "Implement caching"
```

Layout shows only the active agent pane.

## With Worktree

Isolate changes in a git worktree:

```bash
agentpair --codex-only --worktree "Experimental refactoring"
```

## Choosing an Agent

| Task Type | Recommended Agent |
|-----------|-------------------|
| Code generation | Codex |
| Complex reasoning | Claude |
| Debugging | Either |
| Documentation | Claude |
| Tests | Either |
| Refactoring | Claude |
| Quick fixes | Codex |

## Resume

Single-agent runs can be resumed:

```bash
# Start a run
agentpair --claude-only "Complex task" --max-iterations 5

# Resume if interrupted
agentpair --run-id 1
```

The single-agent mode is preserved on resume.

## Best Practices

1. **Use for simple tasks** — Paired mode is better for complex work
2. **Set appropriate iterations** — Single-agent may need fewer
3. **Include proof when needed** — Helps ensure quality
4. **Watch initial runs** — Verify the agent does what you expect

## Next Steps

- [Paired Sessions](paired-sessions.md) — For complex tasks
- [Resuming Runs](resuming.md) — Continue interrupted sessions
