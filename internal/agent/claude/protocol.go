package claude

import (
	"encoding/json"
	"time"
)

// MessageType represents NDJSON message types from Claude SDK.
type MessageType string

const (
	// Incoming from Claude
	TypeStreamEvent    MessageType = "stream_event"
	TypeResult         MessageType = "result"
	TypeControlRequest MessageType = "control_request"
	TypeError          MessageType = "error"

	// Outgoing to Claude
	TypeInit           MessageType = "init"
	TypeUserMessage    MessageType = "user_message"
	TypeControlApprove MessageType = "control_approve"
	TypeControlDeny    MessageType = "control_deny"
	TypeStop           MessageType = "stop"
)

// Message is the base NDJSON message structure.
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// InitPayload is sent to initialize a Claude session.
type InitPayload struct {
	SessionID    string         `json:"session_id,omitempty"`
	WorkDir      string         `json:"work_dir"`
	Prompt       string         `json:"prompt,omitempty"`
	Model        string         `json:"model,omitempty"`
	MaxTokens    int            `json:"max_tokens,omitempty"`
	Tools        []string       `json:"tools,omitempty"`
	MCPServers   []MCPServer    `json:"mcp_servers,omitempty"`
	SystemPrompt string         `json:"system_prompt,omitempty"`
	Permissions  map[string]any `json:"permissions,omitempty"`
}

// MCPServer represents an MCP server configuration.
type MCPServer struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// UserMessagePayload is sent to provide user input.
type UserMessagePayload struct {
	Content string `json:"content"`
}

// StreamEventPayload represents streaming events from Claude.
type StreamEventPayload struct {
	Event     string          `json:"event"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp time.Time       `json:"timestamp,omitempty"`
}

// ResultPayload is the final result from Claude.
type ResultPayload struct {
	SessionID  string         `json:"session_id"`
	Output     string         `json:"output"`
	ToolCalls  []ToolCall     `json:"tool_calls,omitempty"`
	Done       bool           `json:"done"`
	Error      string         `json:"error,omitempty"`
	Statistics *Statistics    `json:"statistics,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ToolCall represents a tool invocation by Claude.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Result    string          `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// Statistics contains usage statistics.
type Statistics struct {
	InputTokens  int           `json:"input_tokens"`
	OutputTokens int           `json:"output_tokens"`
	Duration     time.Duration `json:"duration"`
}

// ControlRequestPayload is sent when Claude needs permission.
type ControlRequestPayload struct {
	RequestID   string `json:"request_id"`
	Type        string `json:"type"` // "command", "file_write", "file_edit", etc.
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	Content     string `json:"content,omitempty"`
}

// ControlApprovePayload approves a control request.
type ControlApprovePayload struct {
	RequestID string `json:"request_id"`
}

// ControlDenyPayload denies a control request.
type ControlDenyPayload struct {
	RequestID string `json:"request_id"`
	Reason    string `json:"reason,omitempty"`
}

// ErrorPayload represents an error from Claude.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewMessage creates a new message with the given type and payload.
func NewMessage(msgType MessageType, payload any) (*Message, error) {
	var rawPayload json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		rawPayload = data
	}
	return &Message{
		Type:    msgType,
		Payload: rawPayload,
	}, nil
}

// DecodePayload decodes the message payload into the given type.
func (m *Message) DecodePayload(v any) error {
	if m.Payload == nil {
		return nil
	}
	return json.Unmarshal(m.Payload, v)
}

// Encode encodes the message to JSON bytes.
func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// DecodeMessage decodes a JSON message.
func DecodeMessage(data []byte) (*Message, error) {
	var m Message
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
