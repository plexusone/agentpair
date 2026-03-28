# AgentPair Tasks

## Open Items

### High Priority

- [ ] **Verify Claude CLI protocol** - Test against actual `claude --json` output to ensure NDJSON parsing is correct
- [ ] **Verify Codex App Server protocol** - Test against actual Codex server to ensure JSON-RPC 2.0 implementation is correct

### Medium Priority

- [ ] **Add integration tests** - End-to-end tests with mock agents
- [ ] **Improve dashboard** - Add real-time bridge message counts, better formatting

### Low Priority

- [ ] **Add shell completion** - Custom completions for --agent, --review flags
- [ ] **Add metrics** - Track iteration times, token usage, success rates
- [ ] **Add retry logic** - Retry failed agent executions with backoff
- [ ] **Documentation** - README with usage examples, architecture docs

## Completed

### Core Implementation

- [x] Create directory structure and go.mod
- [x] Implement pkg/jsonl
- [x] Implement internal/config
- [x] Implement internal/bridge
- [x] Implement internal/run
- [x] Implement internal/review
- [x] Implement internal/agent (interface)
- [x] Implement internal/agent/claude
- [x] Implement internal/agent/codex
- [x] Implement internal/loop
- [x] Implement internal/tmux
- [x] Implement internal/worktree
- [x] Implement internal/dashboard
- [x] Implement internal/update
- [x] Implement cmd/agentpair CLI

### Infrastructure

- [x] Integrate MCP server with loop (using official go-sdk v1.4.1)
- [x] Add config file support (YAML/JSON from ~/.agentpair/)
- [x] Add structured logging via slog (internal/logger package)

### Tests

- [x] Add unit tests for pkg/jsonl
- [x] Add unit tests for internal/bridge
- [x] Add unit tests for internal/review
- [x] Add unit tests for internal/loop/state
- [x] Add unit tests for internal/config
- [x] Add unit tests for internal/run (manifest, transcript, run manager)
- [x] Add unit tests for internal/logger
- [x] Add unit tests for internal/tmux
- [x] Add unit tests for internal/worktree

## Test Coverage

| Package | Tests | Status |
|---------|-------|--------|
| pkg/jsonl | 5 | ✓ |
| internal/bridge | 8 | ✓ |
| internal/config | 12 | ✓ |
| internal/logger | 13 | ✓ |
| internal/loop | 6 | ✓ |
| internal/review | 10 | ✓ |
| internal/run | 20 | ✓ |
| internal/tmux | 7 | ✓ |
| internal/worktree | 10 | ✓ |
