package loop

import (
	"context"
	"fmt"
	"time"

	"github.com/plexusone/agentpair/internal/agent"
	"github.com/plexusone/agentpair/internal/bridge"
	"github.com/plexusone/agentpair/internal/config"
	"github.com/plexusone/agentpair/internal/review"
	"github.com/plexusone/agentpair/internal/run"
)

// SingleLoop orchestrates a single agent execution.
type SingleLoop struct {
	cfg       *config.Config
	run       *run.Run
	agent     agent.Agent
	machine   *Machine
	parser    *review.Parser
	mcpServer *bridge.Server

	iteration int
	startTime time.Time
}

// NewSingle creates a new single-agent loop.
func NewSingle(cfg *config.Config, r *run.Run, a agent.Agent) *SingleLoop {
	return &SingleLoop{
		cfg:     cfg,
		run:     r,
		agent:   a,
		machine: NewMachine(),
		parser:  review.NewParser(cfg.DoneSignal),
	}
}

// Run executes the single-agent loop until completion.
func (l *SingleLoop) Run(ctx context.Context) error {
	l.startTime = time.Now()

	// Log start
	l.run.Transcript.LogStart(l.cfg.Prompt, map[string]any{
		"agent":          l.agent.Name(),
		"mode":           "single",
		"max_iterations": l.cfg.MaxIterations,
	})

	// Start MCP server for bridge tools
	if err := l.startMCPServer(ctx); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	defer l.stopMCPServer()

	// Set MCP server address on agent
	if mcpAddr := l.MCPServerAddr(); mcpAddr != "" {
		l.agent.SetMCPServerAddr(mcpAddr)
	}

	// Start agent
	if err := l.agent.Start(ctx); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}
	defer l.agent.Stop(ctx)

	// Update session ID in manifest
	if l.agent.Name() == "claude" {
		l.run.Manifest.ClaudeSessionID = l.agent.SessionID()
	} else {
		l.run.Manifest.CodexSessionID = l.agent.SessionID()
	}

	// Transition to working
	l.machine.Transition(StateWorking)
	l.run.Manifest.SetState(run.StateWorking)
	l.run.Transcript.LogState(run.StateWorking)
	l.run.Save()

	// Send initial task
	initialMsg := bridge.NewTaskMessage("system", l.agent.Name(), l.cfg.Prompt)
	initialMsg.WithRunInfo(l.run.Manifest.ID, 0)
	l.run.Bridge.Send(ctx, initialMsg)

	// Main loop
	for l.iteration = 1; l.iteration <= l.cfg.MaxIterations; l.iteration++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		l.run.Manifest.IncrementIteration()
		l.run.Transcript.LogIteration(l.iteration)

		done, err := l.executeIteration(ctx)
		if err != nil {
			l.run.Transcript.LogError(l.agent.Name(), l.iteration, err)
			l.machine.Transition(StateFailed)
			l.run.Manifest.SetState(run.StateFailed)
			l.run.Manifest.Error = err.Error()
			l.run.Save()
			return err
		}

		if done {
			break
		}
	}

	// Check max iterations
	if l.iteration > l.cfg.MaxIterations {
		l.machine.Transition(StateFailed)
		l.run.Manifest.SetState(run.StateFailed)
		l.run.Manifest.Error = "max iterations reached"
		l.run.Save()
		return fmt.Errorf("max iterations reached")
	}

	// Complete
	l.machine.Transition(StateComplete)
	l.run.Manifest.SetState(run.StateCompleted)
	l.run.Transcript.LogEnd(run.StateCompleted, time.Since(l.startTime))
	l.run.Save()

	return nil
}

func (l *SingleLoop) executeIteration(ctx context.Context) (bool, error) {
	// Drain messages for agent
	msgs, err := l.run.Bridge.DrainNew(ctx, l.agent.Name(), "")
	if err != nil {
		return false, fmt.Errorf("failed to drain messages: %w", err)
	}

	// Execute agent
	l.run.Transcript.LogExecute(l.agent.Name(), l.iteration, fmt.Sprintf("%d messages", len(msgs)))
	startTime := time.Now()

	result, err := l.agent.Execute(ctx, msgs)
	if err != nil {
		return false, fmt.Errorf("agent execution failed: %w", err)
	}

	l.run.Transcript.LogResult(l.agent.Name(), l.iteration, result.Output, time.Since(startTime))

	// Send result to bridge for logging
	resultMsg := bridge.NewResultMessage(l.agent.Name(), "system", result.Output)
	resultMsg.WithRunInfo(l.run.Manifest.ID, l.iteration)
	l.run.Bridge.Send(ctx, resultMsg)

	// Check for done signal
	if result.Done || l.parser.IsDone(result.Output) {
		l.run.Transcript.LogSignal(l.agent.Name(), l.iteration, "DONE")
		return true, nil
	}

	// Check for explicit signals
	parsed := l.parser.Parse(result.Output)
	if parsed.Signal == review.SignalPass {
		l.run.Transcript.LogSignal(l.agent.Name(), l.iteration, "PASS")
		return true, nil
	}

	return false, nil
}

// Resume continues a single-agent run from its current state.
func (l *SingleLoop) Resume(ctx context.Context) error {
	l.machine = NewMachine()
	l.machine.current = FromRunState(l.run.Manifest.State)
	l.iteration = l.run.Manifest.CurrentIteration

	return l.Run(ctx)
}

// Status returns the current loop status.
func (l *SingleLoop) Status() *Status {
	return &Status{
		State:         l.machine.Current(),
		Iteration:     l.iteration,
		MaxIterations: l.cfg.MaxIterations,
		RunID:         l.run.Manifest.ID,
		Elapsed:       time.Since(l.startTime),
		BridgeStatus:  l.run.Bridge.Status(),
	}
}

func (l *SingleLoop) startMCPServer(ctx context.Context) error {
	l.mcpServer = bridge.NewServer(l.run.Bridge)

	// Start server in background on random port
	errCh := make(chan error, 1)

	go func() {
		errCh <- l.mcpServer.ListenAndServe(ctx, ":0")
	}()

	// Wait briefly for server to start
	select {
	case err := <-errCh:
		return fmt.Errorf("MCP server failed: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
	}

	return nil
}

func (l *SingleLoop) stopMCPServer() {
	if l.mcpServer != nil {
		l.mcpServer.Close()
	}
}

// MCPServerAddr returns the MCP server address for the agent to connect to.
func (l *SingleLoop) MCPServerAddr() string {
	if l.mcpServer == nil {
		return ""
	}
	return l.mcpServer.Addr()
}
