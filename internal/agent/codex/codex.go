// Package codex implements the Codex agent via the Codex App Server.
package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/plexusone/agentpair/internal/agent"
	"github.com/plexusone/agentpair/internal/bridge"
)

// Agent implements the Agent interface for Codex.
type Agent struct {
	config    *agent.Config
	sessionID string
	state     string
	conn      *websocket.Conn
	mu        sync.Mutex
	running   bool
	done      chan struct{}
	responses map[int64]chan *Response
	respMu    sync.Mutex
}

// New creates a new Codex agent.
func New(cfg *agent.Config) *Agent {
	return &Agent{
		config:    cfg,
		state:     agent.StateIdle,
		done:      make(chan struct{}),
		responses: make(map[int64]chan *Response),
	}
}

// Name returns the agent name.
func (a *Agent) Name() string {
	return "codex"
}

// Start initializes and starts the Codex agent.
func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return errors.New("agent already running")
	}

	a.state = agent.StateStarting

	// Connect to Codex App Server
	// Default to localhost:3000 if not specified
	serverURL := "ws://localhost:3000/ws"

	var err error
	a.conn, _, err = websocket.Dial(ctx, serverURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"User-Agent": []string{"agentpair/1.0"},
		},
	})
	if err != nil {
		a.state = agent.StateError
		return fmt.Errorf("failed to connect to codex server: %w", err)
	}

	// Start message reader.
	// Note: readMessages uses context.Background() internally for long-lived reads
	// because the reader goroutine outlives individual request contexts.
	go a.readMessages() //nolint:gosec

	// Initialize session
	initReq := NewRequest(MethodInitialize, &InitializeParams{
		WorkDir:     a.config.WorkDir,
		Model:       a.config.Model,
		AutoApprove: a.config.AutoApprove,
	})

	resp, err := a.sendRequest(ctx, initReq)
	if err != nil {
		a.conn.Close(websocket.StatusNormalClosure, "init failed")
		a.state = agent.StateError
		return fmt.Errorf("failed to initialize: %w", err)
	}

	var initResult InitializeResult
	if err := resp.DecodeResult(&initResult); err != nil {
		a.conn.Close(websocket.StatusNormalClosure, "init decode failed")
		a.state = agent.StateError
		return fmt.Errorf("failed to decode init result: %w", err)
	}

	a.sessionID = initResult.SessionID
	a.running = true
	a.state = agent.StateRunning

	return nil
}

func (a *Agent) readMessages() {
	defer func() {
		a.mu.Lock()
		a.running = false
		a.state = agent.StateStopped
		a.mu.Unlock()
		close(a.done)
	}()

	for {
		_, data, err := a.conn.Read(context.Background())
		if err != nil {
			return
		}

		// Try to decode as response
		var resp Response
		if err := json.Unmarshal(data, &resp); err == nil && resp.ID != 0 {
			a.respMu.Lock()
			ch, ok := a.responses[resp.ID]
			if ok {
				ch <- &resp
				delete(a.responses, resp.ID)
			}
			a.respMu.Unlock()
			continue
		}

		// Try to decode as notification
		var notif Notification
		if err := json.Unmarshal(data, &notif); err == nil {
			a.handleNotification(&notif)
		}
	}
}

func (a *Agent) handleNotification(notif *Notification) {
	switch notif.Method {
	case NotifyCommandRequest:
		if a.config.AutoApprove {
			var params CommandRequestParams
			data, err := json.Marshal(notif.Params)
			if err != nil {
				return // Can't marshal params, skip auto-approve
			}
			if err := json.Unmarshal(data, &params); err != nil {
				return // Can't unmarshal params, skip auto-approve
			}

			approveReq := NewRequest(MethodApproveCommand, &ApproveCommandParams{
				RequestID: params.RequestID,
				Command:   params.Command,
				Approved:  true,
			})
			a.sendRequestAsync(approveReq)
		}

	case NotifyFileRequest:
		if a.config.AutoApprove {
			var params FileRequestParams
			data, err := json.Marshal(notif.Params)
			if err != nil {
				return // Can't marshal params, skip auto-approve
			}
			if err := json.Unmarshal(data, &params); err != nil {
				return // Can't unmarshal params, skip auto-approve
			}

			approveReq := NewRequest(MethodApproveFile, &ApproveFileParams{
				RequestID: params.RequestID,
				FilePath:  params.FilePath,
				Approved:  true,
			})
			a.sendRequestAsync(approveReq)
		}

	case NotifyProgress, NotifyOutput:
		// Informational notifications, logged but not acted upon
	}
}

func (a *Agent) sendRequest(ctx context.Context, req *Request) (*Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Create response channel
	respCh := make(chan *Response, 1)
	a.respMu.Lock()
	a.responses[req.ID] = respCh
	a.respMu.Unlock()

	// Send request
	if err := a.conn.Write(ctx, websocket.MessageText, data); err != nil {
		a.respMu.Lock()
		delete(a.responses, req.ID)
		a.respMu.Unlock()
		return nil, err
	}

	// Wait for response with timeout
	timeout := time.Duration(a.config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	select {
	case resp := <-respCh:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, errors.New("request timeout")
	case <-a.done:
		return nil, errors.New("agent stopped")
	}
}

// sendRequestAsync sends a request without waiting for response.
// Errors are intentionally not returned as this is fire-and-forget for auto-approve.
func (a *Agent) sendRequestAsync(req *Request) {
	data, err := json.Marshal(req)
	if err != nil {
		return // Can't marshal, skip sending
	}
	// Best-effort send; error ignored as this is fire-and-forget
	_ = a.conn.Write(context.Background(), websocket.MessageText, data)
}

// Execute sends messages to Codex and waits for a result.
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

	// Send execute request
	execReq := NewRequest(MethodExecute, &ExecuteParams{
		Prompt:      prompt.String(),
		MaxTokens:   a.config.MaxTokens,
		Timeout:     a.config.Timeout,
		AutoApprove: a.config.AutoApprove,
	})

	resp, err := a.sendRequest(ctx, execReq)
	if err != nil {
		return nil, fmt.Errorf("execute failed: %w", err)
	}

	var execResult ExecuteResult
	if err := resp.DecodeResult(&execResult); err != nil {
		return nil, fmt.Errorf("failed to decode result: %w", err)
	}

	result := agent.NewResult()
	result.Output = execResult.Output
	result.SessionID = a.sessionID

	// Check for signals in output
	upper := strings.ToUpper(execResult.Output)
	if strings.Contains(upper, "DONE") || execResult.Done {
		result.Done = true
	}
	if strings.Contains(upper, "PASS") {
		result.Pass = true
	}
	if strings.Contains(upper, "FAIL") {
		result.Fail = true
	}

	return result, nil
}

// Stop gracefully stops the Codex agent.
func (a *Agent) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.state = agent.StateStopping

	// Send shutdown request
	shutdownReq := NewRequest(MethodShutdown, nil)
	a.sendRequestAsync(shutdownReq)

	// Close connection
	a.conn.Close(websocket.StatusNormalClosure, "shutdown")

	// Wait for reader to exit
	select {
	case <-a.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
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
		Name:      "codex",
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
