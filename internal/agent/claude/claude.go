// Package claude implements the Claude agent via the Claude SDK WebSocket server.
package claude

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/plexusone/agentpair/internal/agent"
	"github.com/plexusone/agentpair/internal/bridge"
)

// Agent implements the Agent interface for Claude.
type Agent struct {
	config    *agent.Config
	sessionID string
	state     string
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	mu        sync.Mutex
	running   bool
	done      chan struct{}
}

// New creates a new Claude agent.
func New(cfg *agent.Config) *Agent {
	return &Agent{
		config: cfg,
		state:  agent.StateIdle,
		done:   make(chan struct{}),
	}
}

// Name returns the agent name.
func (a *Agent) Name() string {
	return "claude"
}

// Start initializes and starts the Claude agent process.
func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return errors.New("agent already running")
	}

	a.state = agent.StateStarting

	// Build command arguments
	args := []string{"--json"}

	if a.config.SessionID != "" {
		args = append(args, "--session", a.config.SessionID)
	}

	if a.config.WorkDir != "" {
		args = append(args, "--cwd", a.config.WorkDir)
	}

	if a.config.AutoApprove {
		args = append(args, "--dangerously-skip-permissions")
	}

	if a.config.Model != "" {
		args = append(args, "--model", a.config.Model)
	}

	// Start the claude command
	a.cmd = exec.CommandContext(ctx, "claude", args...)

	var err error
	a.stdin, err = a.cmd.StdinPipe()
	if err != nil {
		a.state = agent.StateError
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	a.stdout, err = a.cmd.StdoutPipe()
	if err != nil {
		a.state = agent.StateError
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := a.cmd.Start(); err != nil {
		a.state = agent.StateError
		return fmt.Errorf("failed to start claude: %w", err)
	}

	a.running = true
	a.state = agent.StateRunning

	// Start goroutine to wait for process exit
	go func() {
		a.cmd.Wait()
		a.mu.Lock()
		a.running = false
		a.state = agent.StateStopped
		a.mu.Unlock()
		close(a.done)
	}()

	return nil
}

// Execute sends messages to Claude and waits for a result.
func (a *Agent) Execute(ctx context.Context, msgs []*bridge.Message) (*agent.Result, error) {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil, errors.New("agent not running")
	}
	a.state = agent.StateExecuting
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		if a.running {
			a.state = agent.StateRunning
		}
		a.mu.Unlock()
	}()

	// Build prompt from messages
	var prompt strings.Builder
	for _, msg := range msgs {
		prompt.WriteString(fmt.Sprintf("[From %s]: %s\n\n", msg.From, msg.Content))
	}

	// Send user message
	userMsg, err := NewMessage(TypeUserMessage, &UserMessagePayload{
		Content: prompt.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	msgBytes, err := userMsg.Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	if _, err := a.stdin.Write(append(msgBytes, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write to stdin: %w", err)
	}

	// Read response
	result := agent.NewResult()
	scanner := bufio.NewScanner(a.stdout)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-a.done:
			return result, nil
		default:
		}

		if !scanner.Scan() {
			break
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		msg, err := DecodeMessage(line)
		if err != nil {
			continue
		}

		switch msg.Type {
		case TypeResult:
			var payload ResultPayload
			if err := msg.DecodePayload(&payload); err != nil {
				return nil, fmt.Errorf("failed to decode result: %w", err)
			}

			result.Output = payload.Output
			result.SessionID = payload.SessionID
			a.sessionID = payload.SessionID

			if payload.Error != "" {
				result.Error = errors.New(payload.Error)
			}

			// Check for signals in output
			if strings.Contains(strings.ToUpper(payload.Output), "DONE") {
				result.Done = true
			}
			if strings.Contains(strings.ToUpper(payload.Output), "PASS") {
				result.Pass = true
			}
			if strings.Contains(strings.ToUpper(payload.Output), "FAIL") {
				result.Fail = true
			}

			return result, nil

		case TypeControlRequest:
			// Auto-approve if configured
			if a.config.AutoApprove {
				var payload ControlRequestPayload
				if err := msg.DecodePayload(&payload); err != nil {
					continue
				}

				approveMsg, _ := NewMessage(TypeControlApprove, &ControlApprovePayload{
					RequestID: payload.RequestID,
				})
				msgBytes, _ := approveMsg.Encode()
				a.stdin.Write(append(msgBytes, '\n'))
			}

		case TypeError:
			var payload ErrorPayload
			if err := msg.DecodePayload(&payload); err != nil {
				return nil, fmt.Errorf("failed to decode error: %w", err)
			}
			return nil, fmt.Errorf("claude error: %s - %s", payload.Code, payload.Message)

		case TypeStreamEvent:
			// Stream events are informational, continue reading
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return result, nil
}

// Stop gracefully stops the Claude agent.
func (a *Agent) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.state = agent.StateStopping

	// Send stop message
	stopMsg, _ := NewMessage(TypeStop, nil)
	msgBytes, _ := stopMsg.Encode()
	a.stdin.Write(append(msgBytes, '\n'))

	// Close stdin to signal EOF
	a.stdin.Close()

	// Wait for process to exit with timeout
	select {
	case <-a.done:
		return nil
	case <-ctx.Done():
		// Force kill if context cancelled
		if a.cmd.Process != nil {
			a.cmd.Process.Kill()
		}
		return ctx.Err()
	case <-time.After(5 * time.Second):
		// Force kill after timeout
		if a.cmd.Process != nil {
			a.cmd.Process.Kill()
		}
		return nil
	}
}

// SessionID returns the current session ID.
func (a *Agent) SessionID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessionID
}

// IsRunning returns true if the agent is running.
func (a *Agent) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

// Status returns the current agent status.
func (a *Agent) Status() *agent.Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return &agent.Status{
		Name:      "claude",
		Running:   a.running,
		SessionID: a.sessionID,
		State:     a.state,
	}
}

// SetMCPServerAddr sets the MCP server address for bridge tools.
func (a *Agent) SetMCPServerAddr(addr string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.config.MCPServerAddr = addr
}
