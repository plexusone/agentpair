package codex

import (
	"encoding/json"
	"sync/atomic"
)

// JSON-RPC 2.0 types for Codex App Server communication.

var requestID atomic.Int64

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *RPCError) Error() string {
	return e.Message
}

// Notification represents a JSON-RPC 2.0 notification (no id field).
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// NewRequest creates a new JSON-RPC 2.0 request.
func NewRequest(method string, params any) *Request {
	return &Request{
		JSONRPC: "2.0",
		ID:      requestID.Add(1),
		Method:  method,
		Params:  params,
	}
}

// NewNotification creates a new JSON-RPC 2.0 notification.
func NewNotification(method string, params any) *Notification {
	return &Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
}

// Codex-specific methods and params.

// Method names for Codex App Server.
const (
	MethodInitialize       = "initialize"
	MethodShutdown         = "shutdown"
	MethodExecute          = "execute"
	MethodCancel           = "cancel"
	MethodApproveCommand   = "approveCommand"
	MethodApproveFile      = "approveFile"
	MethodGetSessionStatus = "getSessionStatus"
)

// InitializeParams for the initialize method.
type InitializeParams struct {
	WorkDir     string            `json:"workDir"`
	Model       string            `json:"model,omitempty"`
	AutoApprove bool              `json:"autoApprove,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// InitializeResult from the initialize method.
type InitializeResult struct {
	SessionID    string   `json:"sessionId"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// ExecuteParams for the execute method.
type ExecuteParams struct {
	Prompt      string            `json:"prompt"`
	MaxTokens   int               `json:"maxTokens,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
	AutoApprove bool              `json:"autoApprove,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
}

// ExecuteResult from the execute method.
type ExecuteResult struct {
	Output     string          `json:"output"`
	Done       bool            `json:"done"`
	ToolCalls  []ToolCall      `json:"toolCalls,omitempty"`
	Statistics *Statistics     `json:"statistics,omitempty"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Statistics contains usage statistics.
type Statistics struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	DurationMs   int `json:"durationMs"`
}

// ApproveCommandParams for command approval.
type ApproveCommandParams struct {
	RequestID string `json:"requestId"`
	Command   string `json:"command"`
	Approved  bool   `json:"approved"`
}

// ApproveFileParams for file change approval.
type ApproveFileParams struct {
	RequestID string `json:"requestId"`
	FilePath  string `json:"filePath"`
	Approved  bool   `json:"approved"`
}

// SessionStatusResult from getSessionStatus.
type SessionStatusResult struct {
	SessionID string `json:"sessionId"`
	State     string `json:"state"`
	Running   bool   `json:"running"`
}

// Notification methods from Codex.
const (
	NotifyCommandRequest = "commandRequest"
	NotifyFileRequest    = "fileRequest"
	NotifyProgress       = "progress"
	NotifyOutput         = "output"
	NotifyError          = "error"
)

// CommandRequestParams for command approval requests.
type CommandRequestParams struct {
	RequestID   string `json:"requestId"`
	Command     string `json:"command"`
	Description string `json:"description"`
}

// FileRequestParams for file change approval requests.
type FileRequestParams struct {
	RequestID   string `json:"requestId"`
	FilePath    string `json:"filePath"`
	Operation   string `json:"operation"` // "create", "edit", "delete"
	Description string `json:"description"`
}

// ProgressParams for progress updates.
type ProgressParams struct {
	Message    string `json:"message"`
	Percentage int    `json:"percentage,omitempty"`
}

// OutputParams for streaming output.
type OutputParams struct {
	Text   string `json:"text"`
	Stream string `json:"stream"` // "stdout" or "stderr"
}

// ErrorParams for error notifications.
type ErrorParams struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// DecodeResult decodes a response result into the given type.
func (r *Response) DecodeResult(v any) error {
	if r.Result == nil {
		return nil
	}
	return json.Unmarshal(r.Result, v)
}
