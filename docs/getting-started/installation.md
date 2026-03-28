# Installation

## Prerequisites

Before installing AgentPair, ensure you have:

- **Go 1.21+** — Required for building from source
- **claude CLI** — [Claude Code](https://claude.ai/code) installed and logged in
- **codex CLI** — [OpenAI Codex](https://github.com/openai/codex) installed and logged in

Optional:

- **tmux** — For side-by-side agent view
- **Git** — For worktree isolation feature

## Install with Go

```bash
go install github.com/plexusone/agentpair@latest
```

Ensure `$GOPATH/bin` is in your PATH:

```bash
export PATH="$GOPATH/bin:$PATH"
```

## Build from Source

```bash
git clone https://github.com/plexusone/agentpair
cd agentpair
go build -o agentpair ./cmd/agentpair
```

Move to a directory in your PATH:

```bash
mv agentpair ~/.local/bin/
```

## Verify Installation

```bash
agentpair version
```

## Agent Setup

### Claude CLI

Install Claude Code following the [official documentation](https://claude.ai/code/docs):

```bash
# Verify claude is working
claude --version
```

### Codex CLI

Install OpenAI Codex following the [official documentation](https://github.com/openai/codex):

```bash
# Verify codex is working
codex --version
```

!!! warning "Safety Note"
    AgentPair runs agents in auto-approve mode by default. Consider running inside a VM or container for safety. The agents can execute commands and modify files.

## tmux (Optional)

For the side-by-side view feature:

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

Verify:

```bash
tmux -V
```

## Next Steps

- [Quick Start](quick-start.md) — Run your first paired session
- [Configuration](configuration.md) — Customize AgentPair settings
