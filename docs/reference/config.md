# Config File Reference

AgentPair configuration file specification.

## Location

Config files are loaded from `~/.agentpair/` in this order:

1. `config.yaml` (preferred)
2. `config.yml`
3. `config.json`

Only the first found file is loaded.

## Format

### YAML (Recommended)

```yaml
# ~/.agentpair/config.yaml

# Primary worker agent: claude or codex
agent: codex

# Maximum loop iterations before stopping
max_iterations: 20

# Proof/verification command
proof: ""

# Review mode: claude, codex, or claudex
review_mode: claudex

# Custom done signal
done_signal: DONE

# Enable tmux side-by-side view
use_tmux: false

# Enable git worktree isolation
use_worktree: false

# Enable verbose logging
verbose: false

# Maximum run duration (Go duration format)
timeout: 2h
```

### JSON

```json
{
  "agent": "codex",
  "max_iterations": 20,
  "proof": "",
  "review_mode": "claudex",
  "done_signal": "DONE",
  "use_tmux": false,
  "use_worktree": false,
  "verbose": false,
  "timeout": "2h"
}
```

## Fields

### agent

**Type:** string
**Default:** `codex`
**Values:** `claude`, `codex`

Primary worker agent. The other agent acts as reviewer.

```yaml
agent: claude
```

### max_iterations

**Type:** integer
**Default:** `20`
**Range:** 1-1000

Maximum loop iterations before stopping.

```yaml
max_iterations: 50
```

### proof

**Type:** string
**Default:** (empty)

Command to verify task completion. Passed to agents.

```yaml
proof: "go test ./... && golangci-lint run"
```

### review_mode

**Type:** string
**Default:** `claudex`
**Values:** `claude`, `codex`, `claudex`

Who reviews the work:

- `claude` — Only Claude reviews
- `codex` — Only Codex reviews
- `claudex` — Both review; consensus required

```yaml
review_mode: claudex
```

### done_signal

**Type:** string
**Default:** `DONE`

Signal agents output to indicate task completion.

```yaml
done_signal: "COMPLETE"
```

### use_tmux

**Type:** boolean
**Default:** `false`

Enable tmux side-by-side panes.

```yaml
use_tmux: true
```

### use_worktree

**Type:** boolean
**Default:** `false`

Create git worktree for run isolation.

```yaml
use_worktree: true
```

### verbose

**Type:** boolean
**Default:** `false`

Enable verbose logging output.

```yaml
verbose: true
```

### timeout

**Type:** string (duration)
**Default:** `2h`

Maximum run duration. Uses Go duration format:

- `30m` — 30 minutes
- `2h` — 2 hours
- `1h30m` — 1 hour 30 minutes

```yaml
timeout: 4h
```

## Precedence

CLI flags override config file values:

```bash
# Config says agent: codex
# CLI overrides to claude
agentpair --agent claude --prompt "Task"
```

## Example Configurations

### Development

```yaml
# Fast feedback, visual monitoring
agent: codex
max_iterations: 10
use_tmux: true
verbose: false
```

### CI/CD

```yaml
# Long-running, thorough
agent: codex
max_iterations: 100
review_mode: claudex
verbose: true
timeout: 8h
```

### Safety-Focused

```yaml
# Isolated, verified
agent: claude
max_iterations: 20
use_worktree: true
proof: "npm test && npm run lint"
review_mode: claudex
```

### Minimal

```yaml
# Accept all defaults except iterations
max_iterations: 30
```

## Validation

AgentPair validates config values:

| Field | Validation |
|-------|------------|
| `agent` | Must be `claude` or `codex` |
| `max_iterations` | Must be positive integer |
| `review_mode` | Must be `claude`, `codex`, or `claudex` |
| `timeout` | Must be valid Go duration |

Invalid values cause startup errors.

## Creating Config

Create with your preferred editor:

```bash
mkdir -p ~/.agentpair
cat > ~/.agentpair/config.yaml << 'EOF'
agent: codex
max_iterations: 20
review_mode: claudex
EOF
```

## See Also

- [CLI Reference](cli.md) — Command-line flags
- [Configuration Guide](../getting-started/configuration.md) — Usage examples
