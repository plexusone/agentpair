# Bridge Messages Reference

Complete specification for bridge message format.

## Message Structure

Each message is a JSON object in the bridge JSONL file:

```json
{
  "id": "sha256-hash-of-content",
  "run_id": 1,
  "type": "task",
  "from": "claude",
  "to": "codex",
  "content": "Message content here",
  "signal": "",
  "timestamp": "2024-03-28T10:00:00Z"
}
```

## Fields

### id

**Type:** string
**Format:** SHA256 hex string

Unique message identifier generated from content hash. Used for deduplication.

```
"id": "a1b2c3d4e5f6..."
```

### run_id

**Type:** integer

Associated run ID.

```
"run_id": 7
```

### type

**Type:** string
**Values:** `task`, `result`, `review`, `signal`, `chat`

Message type (see [Message Types](#message-types) below).

```
"type": "task"
```

### from

**Type:** string

Sender agent name.

```
"from": "claude"
```

### to

**Type:** string

Target agent name. Empty for broadcast messages.

```
"to": "codex"
```

### content

**Type:** string

Message content. Can be any text including markdown.

```
"content": "Please implement the login validation..."
```

### signal

**Type:** string
**Values:** `DONE`, `PASS`, `FAIL`, or empty

Control signal for `signal` type messages.

```
"signal": "PASS"
```

### timestamp

**Type:** string
**Format:** ISO 8601

Message creation time.

```
"timestamp": "2024-03-28T10:00:00Z"
```

## Message Types

### task

Work request or follow-up task.

```json
{
  "type": "task",
  "from": "claude",
  "to": "codex",
  "content": "Please implement input validation for the email field. Requirements:\n1. Check for valid email format\n2. Show error message for invalid input\n3. Disable submit button until valid"
}
```

### result

Work output from an agent.

```json
{
  "type": "result",
  "from": "codex",
  "to": "claude",
  "content": "Implemented email validation:\n\n```typescript\nfunction validateEmail(email: string): boolean {\n  return /^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(email);\n}\n```\n\nAdded to LoginForm component with error state management."
}
```

### review

Review feedback.

```json
{
  "type": "review",
  "from": "claude",
  "to": "codex",
  "content": "The implementation looks good. A few suggestions:\n\n1. Consider using a more comprehensive email regex\n2. Add aria-label for accessibility\n3. Consider debouncing the validation\n\nPlease address these points."
}
```

### signal

Control signal for loop state.

```json
{
  "type": "signal",
  "from": "claude",
  "to": "",
  "content": "All requirements met, validation working correctly",
  "signal": "PASS"
}
```

### chat

Free-form discussion or clarification.

```json
{
  "type": "chat",
  "from": "codex",
  "to": "claude",
  "content": "Should we also validate the password field in this PR, or keep it separate?"
}
```

## Signals

### DONE

Agent has completed its work.

```json
{
  "type": "signal",
  "signal": "DONE",
  "content": "Task completed successfully"
}
```

### PASS

Reviewer approves the work.

```json
{
  "type": "signal",
  "signal": "PASS",
  "content": "All requirements met, code looks good"
}
```

### FAIL

Reviewer rejects the work.

```json
{
  "type": "signal",
  "signal": "FAIL",
  "content": "Tests are failing, please fix before proceeding"
}
```

## ID Generation

Message IDs are SHA256 hashes:

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

Same content always generates the same ID, enabling deduplication.

## Storage Format

Messages are stored in JSONL (newline-delimited JSON):

```
{"id":"abc...","type":"task","from":"claude","to":"codex","content":"...","timestamp":"..."}
{"id":"def...","type":"result","from":"codex","to":"claude","content":"...","timestamp":"..."}
{"id":"ghi...","type":"review","from":"claude","to":"codex","content":"...","timestamp":"..."}
```

File location: `~/.agentpair/runs/{repo-id}/{run-id}/bridge.jsonl`

## Example Conversation

Complete message flow for a simple task:

```json
// 1. Claude assigns task
{"type":"task","from":"claude","to":"codex","content":"Add email validation to LoginForm","timestamp":"2024-03-28T10:00:00Z"}

// 2. Codex implements
{"type":"result","from":"codex","to":"claude","content":"Added validateEmail function...","timestamp":"2024-03-28T10:01:00Z"}

// 3. Claude reviews with feedback
{"type":"review","from":"claude","to":"codex","content":"Please add error message display","timestamp":"2024-03-28T10:02:00Z"}

// 4. Codex fixes
{"type":"result","from":"codex","to":"claude","content":"Added error message UI...","timestamp":"2024-03-28T10:03:00Z"}

// 5. Codex signals done
{"type":"signal","from":"codex","signal":"DONE","content":"Validation complete","timestamp":"2024-03-28T10:03:30Z"}

// 6. Claude approves
{"type":"signal","from":"claude","signal":"PASS","content":"Looks good","timestamp":"2024-03-28T10:04:00Z"}
```

## MCP Tool Mapping

Bridge messages are accessed via MCP tools:

| MCP Tool | Bridge Operation |
|----------|------------------|
| `send_to_agent` | Append message to JSONL |
| `receive_messages` | Read messages for agent |
| `bridge_status` | Count messages and signals |

## See Also

- [Bridge Concept](../concepts/bridge.md) — How the bridge works
- [MCP Integration](../concepts/mcp.md) — Tool access
