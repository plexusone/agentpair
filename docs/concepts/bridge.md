# Bridge

The bridge enables agent-to-agent communication through a JSONL file with SHA256 deduplication.

## Overview

```
┌─────────┐                           ┌─────────┐
│  Claude │                           │  Codex  │
└────┬────┘                           └────┬────┘
     │                                     │
     │  1. send_to_agent("codex", ...)     │
     ├────────────────────────────────────►│
     │                                     │
     │  2. receive_messages("claude")      │
     │◄────────────────────────────────────┤
     │                                     │
     ▼                                     ▼
┌─────────────────────────────────────────────┐
│              bridge.jsonl                    │
└─────────────────────────────────────────────┘
```

## Message Format

Each message is a JSON object:

```json
{
  "id": "sha256-abc123...",
  "run_id": 1,
  "type": "task",
  "from": "claude",
  "to": "codex",
  "content": "Please implement the login form validation",
  "signal": "",
  "timestamp": "2024-03-28T10:00:00Z"
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | SHA256 hash of content for deduplication |
| `run_id` | int | Associated run ID |
| `type` | string | Message type (see below) |
| `from` | string | Sender agent name |
| `to` | string | Target agent name (or empty for broadcast) |
| `content` | string | Message content |
| `signal` | string | Control signal (DONE, PASS, FAIL) |
| `timestamp` | string | ISO 8601 timestamp |

## Message Types

| Type | Description |
|------|-------------|
| `task` | Initial task or follow-up work request |
| `result` | Work output from an agent |
| `review` | Review feedback |
| `signal` | Control signal (DONE, PASS, FAIL) |
| `chat` | Free-form discussion |

## Signals

| Signal | Meaning |
|--------|---------|
| `DONE` | Agent has completed the task |
| `PASS` | Reviewer approves the work |
| `FAIL` | Reviewer rejects the work |

## Deduplication

Messages are deduplicated using SHA256:

```go
func GenerateID(msgType MessageType, from, to, content string) string {
    h := sha256.New()
    h.Write([]byte(string(msgType)))
    h.Write([]byte(from))
    h.Write([]byte(to))
    h.Write([]byte(content))
    return hex.EncodeToString(h.Sum(nil))
}
```

This ensures:

- Same message content → same ID
- Duplicate sends are ignored
- Safe for retries and restarts

## Storage

Messages are stored in `bridge.jsonl`:

```
~/.agentpair/runs/{repo-id}/{run-id}/bridge.jsonl
```

Format is newline-delimited JSON (NDJSON):

```
{"id":"abc...","type":"task","from":"claude",...}
{"id":"def...","type":"result","from":"codex",...}
{"id":"ghi...","type":"review","from":"claude",...}
```

## Bridge Operations

### Send

```go
func (b *Bridge) Send(ctx context.Context, msg *Message) (bool, error) {
    // Check for duplicate
    if b.seen[msg.ID] {
        return false, nil  // Already sent
    }

    // Mark as seen and append to storage
    b.seen[msg.ID] = true
    return true, b.storage.Append(msg)
}
```

### Drain

```go
func (b *Bridge) Drain(ctx context.Context, target string, seenIDs map[string]bool) ([]*Message, error) {
    all, _ := b.storage.ReadAll()

    var messages []*Message
    for _, msg := range all {
        if msg.IsForAgent(target) && !seenIDs[msg.ID] {
            messages = append(messages, msg)
        }
    }
    return messages, nil
}
```

### Status

```go
type Status struct {
    TotalMessages int
    ByAgent       map[string]int
    ByType        map[MessageType]int
    HasDoneSignal bool
    PassCount     int
    FailCount     int
}
```

## Example Flow

1. **Claude sends task to Codex:**
   ```json
   {"type":"task","from":"claude","to":"codex","content":"Implement login form"}
   ```

2. **Codex receives and works:**
   ```json
   {"type":"result","from":"codex","to":"claude","content":"Added LoginForm.tsx..."}
   ```

3. **Claude reviews:**
   ```json
   {"type":"review","from":"claude","to":"codex","content":"Looks good, minor fix needed..."}
   ```

4. **Codex fixes and signals done:**
   ```json
   {"type":"signal","from":"codex","signal":"DONE","content":"Fixed and ready"}
   ```

5. **Claude approves:**
   ```json
   {"type":"signal","from":"claude","signal":"PASS","content":"Approved"}
   ```

## Viewing Bridge Status

```bash
# All runs
agentpair bridge

# Specific run
agentpair bridge --run-id 1
```

Output:

```
Run #1 Bridge Status:
  Total Messages: 12
  Done Signal: true
  Pass Count: 1
  Fail Count: 0
  By Agent:
    claude: 5
    codex: 7
```

## Next Steps

- [MCP Integration](mcp.md) — How agents access bridge tools
- [Bridge Messages Reference](../reference/messages.md) — Full message specification
