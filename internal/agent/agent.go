// Package agent defines the Agent interface and common types.
package agent

import (
	"context"

	"github.com/plexusone/agentpair/internal/bridge"
)

// Agent represents an AI agent that can execute tasks.
type Agent interface {
	// Name returns the agent's identifier (e.g., "claude" or "codex").
	Name() string

	// Start initializes and starts the agent.
	Start(ctx context.Context) error

	// Execute sends messages to the agent and waits for a result.
	Execute(ctx context.Context, msgs []*bridge.Message) (*Result, error)

	// Stop gracefully stops the agent.
	Stop(ctx context.Context) error

	// SessionID returns the current session identifier.
	SessionID() string

	// IsRunning returns true if the agent is currently running.
	IsRunning() bool

	// SetMCPServerAddr sets the MCP server address for bridge tools.
	SetMCPServerAddr(addr string)
}

// Result represents the result of an agent execution.
type Result struct {
	// Output is the agent's response content.
	Output string

	// Messages are any bridge messages the agent wants to send.
	Messages []*bridge.Message

	// Done indicates the agent has signaled completion.
	Done bool

	// Pass indicates the agent has approved (review passed).
	Pass bool

	// Fail indicates the agent has rejected (review failed).
	Fail bool

	// Error is set if the agent encountered an error.
	Error error

	// SessionID is the session ID for resume purposes.
	SessionID string

	// Metadata contains additional result data.
	Metadata map[string]any
}

// NewResult creates a new empty result.
func NewResult() *Result {
	return &Result{
		Metadata: make(map[string]any),
	}
}

// HasSignal returns true if any control signal is set.
func (r *Result) HasSignal() bool {
	return r.Done || r.Pass || r.Fail
}

// Config holds agent configuration.
type Config struct {
	// WorkDir is the working directory for the agent.
	WorkDir string

	// Prompt is the initial task prompt.
	Prompt string

	// SessionID is an existing session to resume.
	SessionID string

	// AutoApprove enables automatic approval of commands.
	AutoApprove bool

	// Verbose enables verbose output.
	Verbose bool

	// Timeout is the per-execution timeout.
	Timeout int // seconds

	// Model specifies the model to use (if applicable).
	Model string

	// MaxTokens limits the response length.
	MaxTokens int

	// Tools specifies which tools the agent can use.
	Tools []string

	// BridgePath is the path to the bridge JSONL file.
	BridgePath string

	// MCPServers lists MCP servers to connect to.
	MCPServers []string

	// MCPServerAddr is the address of the bridge MCP server.
	MCPServerAddr string

	// MCPConfigPath is the path to the MCP config file.
	MCPConfigPath string
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		AutoApprove: true,
		Timeout:     300,
		MaxTokens:   16384,
	}
}

// Status represents the current status of an agent.
type Status struct {
	Name      string `json:"name"`
	Running   bool   `json:"running"`
	SessionID string `json:"session_id,omitempty"`
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
}

// State constants for agent status.
const (
	StateIdle      = "idle"
	StateStarting  = "starting"
	StateRunning   = "running"
	StateExecuting = "executing"
	StateStopping  = "stopping"
	StateStopped   = "stopped"
	StateError     = "error"
)
