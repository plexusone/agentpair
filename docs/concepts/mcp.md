# MCP Integration

AgentPair exposes bridge functionality to agents via the Model Context Protocol (MCP).

## Overview

MCP provides a standard way for AI models to interact with external tools. AgentPair runs an MCP server that gives agents access to bridge communication tools.

```
┌─────────────────────────────────────────────────┐
│                  AgentPair                       │
│                                                  │
│  ┌─────────┐         ┌─────────────┐            │
│  │  Claude │◄───────►│  MCP Server │            │
│  └─────────┘   MCP   │             │            │
│                      │  - send_to_agent         │
│  ┌─────────┐         │  - receive_messages      │
│  │  Codex  │◄───────►│  - bridge_status         │
│  └─────────┘   MCP   └──────┬──────┘            │
│                             │                    │
│                             ▼                    │
│                      ┌─────────────┐            │
│                      │   Bridge    │            │
│                      └─────────────┘            │
└─────────────────────────────────────────────────┘
```

## Available Tools

### send_to_agent

Send a message to another agent through the bridge.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `to` | string | Yes | Target agent name (claude or codex) |
| `message_type` | string | Yes | Type: task, result, review, signal, chat |
| `content` | string | Yes | Message content |
| `signal` | string | No | Signal value: DONE, PASS, or FAIL |

**Example:**

```json
{
  "to": "codex",
  "message_type": "task",
  "content": "Please implement input validation for the email field"
}
```

**Response:**

```
Message sent: id=abc123...
```

Or if duplicate:

```
Message duplicate (already sent): id=abc123...
```

### receive_messages

Receive pending messages from the bridge.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `agent` | string | Yes | Agent name to receive messages for |
| `since_id` | string | No | Only return messages after this ID |

**Example:**

```json
{
  "agent": "claude",
  "since_id": "abc123..."
}
```

**Response:**

```
[1] from=codex type=result
Here's my implementation of the email validation...

[2] from=codex type=signal signal=DONE
Task completed successfully.
```

### bridge_status

Get the current status of the bridge.

**Parameters:** None

**Response:**

```
Bridge Status:
  Total Messages: 15
  Done Signal: true
  Pass Count: 1
  Fail Count: 0
  Messages by Agent:
    claude: 7
    codex: 8
  Messages by Type:
    task: 3
    result: 5
    review: 4
    signal: 3
```

## Server Implementation

The MCP server is implemented using the official `modelcontextprotocol/go-sdk`:

```go
import "github.com/modelcontextprotocol/go-sdk/mcp"

func NewServer(bridge *Bridge) *Server {
    s := &Server{bridge: bridge}

    s.server = mcp.NewServer(&mcp.Implementation{
        Name:    "agentpair-bridge",
        Version: "1.0.0",
    }, nil)

    // Register tools
    mcp.AddTool(s.server, &mcp.Tool{
        Name:        "send_to_agent",
        Description: "Send a message to another agent",
    }, s.handleSendToAgent)

    return s
}
```

## Transport

The MCP server supports multiple transports:

### stdio

Standard input/output for subprocess communication:

```go
server.ListenAndServe(ctx, "stdio")
```

### TCP

Socket-based for network access:

```go
server.ListenAndServe(ctx, ":9100")
```

## Agent Configuration

Agents receive the MCP server address via configuration:

```go
agent.SetMCPServerAddr("localhost:9100")
```

The agent then connects to this server to access bridge tools.

## Tool Flow

1. **Agent starts** — Connects to MCP server
2. **Agent lists tools** — Gets send_to_agent, receive_messages, bridge_status
3. **Agent sends message** — Calls send_to_agent tool
4. **AgentPair routes** — Message stored in bridge.jsonl
5. **Other agent receives** — Calls receive_messages tool
6. **Message delivered** — Agent processes and responds

## Error Handling

MCP tools return errors in the result:

```go
func errorResult(text string) *mcp.CallToolResult {
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            &mcp.TextContent{Text: text},
        },
        IsError: true,
    }
}
```

Agents see these as tool execution failures and can retry or handle appropriately.

## Next Steps

- [Bridge](bridge.md) — Message format and deduplication
- [Architecture](architecture.md) — System overview
