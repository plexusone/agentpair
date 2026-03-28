# Paired Sessions

Paired mode is the default in AgentPair. One agent works while the other reviews.

## How It Works

```
┌─────────────────────────────────────────────────────────┐
│                    Paired Session                        │
│                                                          │
│   Primary (Codex)              Secondary (Claude)        │
│   ┌─────────────┐              ┌─────────────┐          │
│   │   Working   │──────────────│  Reviewing  │          │
│   └──────┬──────┘              └──────┬──────┘          │
│          │                            │                  │
│          │     Bridge Messages        │                  │
│          └────────────────────────────┘                  │
│                                                          │
│   Iteration 1: Codex implements, Claude reviews          │
│   Iteration 2: Codex fixes issues, Claude re-reviews     │
│   Iteration 3: Both signal DONE/PASS → Complete          │
└─────────────────────────────────────────────────────────┘
```

## Starting a Paired Session

```bash
# Default: Codex works, Claude reviews
agentpair --prompt "Implement user authentication"

# Claude works, Codex reviews
agentpair --agent claude --prompt "Implement user authentication"
```

## Review Modes

Control who reviews with `--review`:

| Mode | Behavior |
|------|----------|
| `claude` | Only Claude reviews |
| `codex` | Only Codex reviews |
| `claudex` | Both review; consensus required |

```bash
# Only Claude reviews Codex's work
agentpair --review claude --prompt "Add API endpoints"

# Both review (default)
agentpair --review claudex --prompt "Implement OAuth"
```

### Consensus Mode (claudex)

In `claudex` mode:

1. Primary agent completes work
2. Both agents review in parallel
3. Both must signal PASS for completion
4. If either signals FAIL, work continues

## Proof Requirements

Use `--proof` to specify verification criteria:

```bash
agentpair --prompt "Add input validation" \
  --proof "Run 'npm test' and ensure all tests pass"
```

The proof requirement is passed to agents so they know how to verify their work.

## Iteration Limits

Control maximum iterations:

```bash
# Stop after 10 iterations
agentpair --max-iterations 10 --prompt "Complex refactoring task"
```

Default is 20 iterations.

## Custom Done Signal

Change the completion signal:

```bash
agentpair --done "COMPLETE" --prompt "Task"
```

Agents should output this signal when they're done.

## Example Workflow

### Feature Development

```bash
agentpair --tmux --prompt "Implement a REST API for blog posts" \
  --proof "Run 'go test ./... && curl localhost:8080/posts'" \
  --max-iterations 15
```

Flow:

1. **Iteration 1**: Codex creates basic API structure
2. **Iteration 2**: Claude reviews, requests error handling
3. **Iteration 3**: Codex adds error handling
4. **Iteration 4**: Claude reviews, requests tests
5. **Iteration 5**: Codex adds tests
6. **Iteration 6**: Claude reviews, signals PASS
7. **Complete**: Both agents satisfied

### Bug Fix

```bash
agentpair --prompt "Fix the race condition in cache.go" \
  --proof "Run 'go test -race ./...' with no races" \
  --agent claude  # Claude is better at concurrency
```

### Code Review Focus

```bash
agentpair --prompt "Review security of auth module" \
  --review claudex \
  --max-iterations 5
```

## Monitoring

### Watch in Real-Time

```bash
# With tmux
agentpair --tmux --prompt "Task"

# Attach to session
tmux attach -t agentpair-myrepo-1
```

### Check Status

```bash
# Dashboard
agentpair dashboard

# Bridge status
agentpair bridge --run-id 1
```

## Best Practices

1. **Be specific with prompts** — Clear prompts lead to faster convergence
2. **Include proof requirements** — Helps agents know when they're done
3. **Use appropriate iterations** — Complex tasks need more iterations
4. **Monitor initial runs** — Watch with `--tmux` until you trust the flow
5. **Start small** — Test with simple tasks before complex ones

## Troubleshooting

### Agents Not Converging

- Increase `--max-iterations`
- Add clearer `--proof` requirements
- Simplify the task prompt

### Review Loops

If agents keep requesting changes:

- Check `--review` mode (try single reviewer)
- Add explicit acceptance criteria
- Review bridge messages for patterns

### Communication Issues

Check bridge status:

```bash
agentpair bridge --run-id 1
```

Look for:

- Message counts (should increase over time)
- Signal patterns (DONE/PASS/FAIL)
- Agent distribution (both should be sending)

## Next Steps

- [Single-Agent Mode](single-agent.md) — Simpler tasks
- [tmux Layout](tmux.md) — Visual monitoring
- [Resuming Runs](resuming.md) — Continue interrupted sessions
