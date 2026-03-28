# tmux Layout

AgentPair can run agents in a tmux session with side-by-side panes for visual monitoring.

## Prerequisites

Install tmux:

=== "macOS"
    ```bash
    brew install tmux
    ```

=== "Ubuntu/Debian"
    ```bash
    sudo apt install tmux
    ```

=== "Fedora"
    ```bash
    sudo dnf install tmux
    ```

## Basic Usage

```bash
agentpair --tmux --prompt "Implement feature X"
```

This creates a detached tmux session with the layout:

```
┌─────────────────────────────────────────────────────────┐
│                                                          │
│   Claude Pane (0)          │    Codex Pane (1)          │
│                            │                            │
│   [Claude output here]     │    [Codex output here]     │
│                            │                            │
│                            │                            │
│                            │                            │
├────────────────────────────┴────────────────────────────┤
│   Status Pane (2)                                        │
│   [Bridge status, iteration info]                        │
└─────────────────────────────────────────────────────────┘
```

## Attaching to Sessions

After starting:

```bash
# Attach to the session
tmux attach -t agentpair-myrepo-1
```

Session names follow the pattern: `agentpair-{repo}-{run-id}`

## Session Management

### List Sessions

```bash
tmux list-sessions
```

Output:

```
agentpair-myrepo-1: 3 windows (created Sat Mar 28 10:00:00 2024)
agentpair-myrepo-2: 3 windows (created Sat Mar 28 11:30:00 2024)
```

### Detach

While attached, press `Ctrl+B` then `D` to detach.

### Kill Session

```bash
tmux kill-session -t agentpair-myrepo-1
```

## Pane Navigation

Standard tmux navigation:

| Keys | Action |
|------|--------|
| `Ctrl+B` `←` | Move to left pane |
| `Ctrl+B` `→` | Move to right pane |
| `Ctrl+B` `↑` | Move to upper pane |
| `Ctrl+B` `↓` | Move to lower pane |
| `Ctrl+B` `z` | Zoom current pane |
| `Ctrl+B` `[` | Enter scroll mode |
| `q` | Exit scroll mode |

## Scrolling

To scroll through agent output:

1. Press `Ctrl+B` then `[` to enter scroll mode
2. Use arrow keys or Page Up/Down to scroll
3. Press `q` to exit scroll mode

## Layout Customization

The default layout uses equal horizontal splits for agents and a smaller bottom pane for status.

tmux configuration can be customized in `~/.tmux.conf`:

```bash
# Example: increase scrollback
set-option -g history-limit 50000

# Mouse support for scrolling
set -g mouse on
```

## Persistent Sessions

tmux sessions persist even if:

- You close your terminal
- You disconnect from SSH
- Your shell exits

Reattach anytime with:

```bash
tmux attach -t agentpair-myrepo-1
```

## With SSH

tmux is particularly useful over SSH:

```bash
# SSH into remote machine
ssh user@remote

# Start AgentPair with tmux
agentpair --tmux --prompt "Task"

# Detach (Ctrl+B, D)

# Disconnect SSH safely

# Later, reconnect
ssh user@remote
tmux attach -t agentpair-myrepo-1
```

## With Worktree

Combine tmux with worktree isolation:

```bash
agentpair --tmux --worktree --prompt "Experimental feature"
```

## Example Workflow

1. **Start session:**
   ```bash
   agentpair --tmux --prompt "Implement OAuth login"
   ```

2. **Attach to watch:**
   ```bash
   tmux attach -t agentpair-myproject-1
   ```

3. **Navigate panes:**
   - `Ctrl+B` `→` to see Codex
   - `Ctrl+B` `←` to see Claude
   - `Ctrl+B` `↓` to see status

4. **Scroll history:**
   - `Ctrl+B` `[` to enter scroll mode
   - Page Up/Down to navigate
   - `q` to exit

5. **Detach when satisfied:**
   - `Ctrl+B` `D` to detach

6. **Check status later:**
   ```bash
   agentpair dashboard
   agentpair bridge --run-id 1
   ```

## Troubleshooting

### "tmux not available"

Install tmux (see Prerequisites above).

### Session Not Found

```bash
# List all sessions
tmux list-sessions

# Session might have completed
agentpair dashboard
```

### Panes Not Updating

- Check if agents are still running
- View bridge status: `agentpair bridge --run-id 1`
- Check for errors in agent panes

### Terminal Issues

If the terminal looks corrupted:

```bash
# Reset terminal
reset

# Or restart tmux server
tmux kill-server
```

## Next Steps

- [Git Worktrees](worktrees.md) — Isolate changes
- [Resuming Runs](resuming.md) — Continue sessions
