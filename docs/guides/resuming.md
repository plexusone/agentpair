# Resuming Runs

AgentPair persists run state to disk, allowing you to resume interrupted sessions.

## Run Persistence

Every run creates a directory:

```
~/.agentpair/runs/{repo-id}/{run-id}/
├── manifest.json      # Run metadata and state
├── bridge.jsonl       # Agent messages
└── transcript.jsonl   # Audit log
```

## Resume by Run ID

The simplest way to resume:

```bash
agentpair --run-id 5
```

This loads the run from disk and continues from the last state.

## Resume by Session ID

Resume using a Claude or Codex session ID:

```bash
agentpair --session abc123-def456
```

AgentPair searches for a run with that session ID.

## Finding Your Run ID

### Dashboard

```bash
agentpair dashboard
```

Shows active and recent runs with their IDs.

### Bridge Command

```bash
agentpair bridge
```

Lists all runs with their IDs and status.

### Manual Lookup

```bash
ls ~/.agentpair/runs/
```

## What Gets Restored

When resuming, AgentPair restores:

| State | Restored |
|-------|----------|
| Prompt | Yes |
| Configuration | Yes |
| Iteration count | Yes |
| Loop state | Yes |
| Bridge messages | Yes |
| Agent sessions | Yes (if agents support) |

## Example Workflow

### 1. Start a Long Task

```bash
agentpair --prompt "Major refactoring" --max-iterations 50
```

Run ID is displayed:

```
INFO created run run_id=7 prompt="Major refactoring..."
```

### 2. Interrupt (Ctrl+C or Disconnect)

The run state is saved automatically.

### 3. Check Status

```bash
agentpair bridge --run-id 7
```

Output:

```
Run #7 Bridge Status:
  Total Messages: 23
  Done Signal: false
  Pass Count: 0
  Fail Count: 0
```

### 4. Resume

```bash
agentpair --run-id 7
```

Continues from iteration 8 (or wherever it stopped).

## tmux and Worktree Alignment

When resuming runs that used `--tmux` or `--worktree`:

- **tmux**: Session name stays aligned (`agentpair-repo-7`)
- **worktree**: Same worktree is re-entered or recreated

```bash
# Original
agentpair --tmux --worktree --prompt "Task"

# Resume maintains both
agentpair --run-id 7
```

## Manifest Structure

The `manifest.json` contains:

```json
{
  "id": 7,
  "repo_id": "github-user-myrepo",
  "prompt": "Major refactoring task",
  "state": "working",
  "current_iteration": 8,
  "max_iterations": 50,
  "primary_agent": "codex",
  "review_mode": "claudex",
  "done_signal": "DONE",
  "claude_session_id": "abc123",
  "codex_session_id": "def456",
  "tmux_session": "agentpair-myrepo-7",
  "worktree_path": "/home/user/.agentpair-worktree-7",
  "repo_path": "/home/user/myrepo",
  "created_at": "2024-03-28T10:00:00Z",
  "updated_at": "2024-03-28T11:30:00Z"
}
```

## Resuming Different States

| State | On Resume |
|-------|-----------|
| `submitted` | Start fresh |
| `working` | Continue primary agent |
| `reviewing` | Continue review |
| `completed` | Nothing to do |
| `failed` | Start fresh or fix issue |

## Clearing State

To start fresh instead of resuming:

```bash
# Delete the run
rm -rf ~/.agentpair/runs/github-user-myrepo/7/

# Or start with new run
agentpair --prompt "Same task but fresh"
```

## Agent Session Resume

Some agents support session persistence:

| Agent | Session Resume |
|-------|----------------|
| Claude | Via session ID |
| Codex | Via thread ID |

The manifest stores these IDs for seamless resume.

## Troubleshooting

### Run Not Found

```
Error: failed to load run: run not found
```

Check the run exists:

```bash
ls ~/.agentpair/runs/
cat ~/.agentpair/runs/*/7/manifest.json
```

### Corrupted State

If manifest is corrupted:

```bash
# View the manifest
cat ~/.agentpair/runs/github-user-myrepo/7/manifest.json | jq .

# Delete and start fresh
rm -rf ~/.agentpair/runs/github-user-myrepo/7/
```

### Agent Session Expired

If agent sessions have expired, AgentPair starts new sessions but continues from the saved iteration.

### Different Directory

Resume works from any directory:

```bash
# Started in /home/user/project-a
agentpair --prompt "Task"

# Can resume from anywhere
cd /tmp
agentpair --run-id 7  # Uses stored repo_path
```

## Best Practices

1. **Note your run ID** — Displayed when starting
2. **Use dashboard** — Easy way to find runs
3. **Check bridge status** — Verify state before resuming
4. **Clean up old runs** — Delete finished runs periodically

## Next Steps

- [Configuration](../getting-started/configuration.md) — Default settings
- [CLI Reference](../reference/cli.md) — All commands and flags
