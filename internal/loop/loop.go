// Package loop provides the main orchestration for agent-to-agent pair programming.
//
// # Error Handling
//
// This package distinguishes between critical and non-critical errors:
//
// Critical errors cause the loop to fail and return immediately:
//   - Agent execution failures (primary or secondary agent cannot execute)
//   - Message draining failures (cannot retrieve messages from bridge)
//   - MCP server startup failures
//   - Agent startup failures
//
// Non-critical errors are logged as warnings but allow the loop to continue:
//   - Transcript logging failures (recording events to transcript.jsonl)
//   - Run save failures (persisting manifest state)
//   - State transition failures (updating internal state machine)
//   - Signal logging failures (recording done/review signals)
//
// This design ensures the core orchestration continues even if auxiliary
// persistence or logging operations fail, while stopping immediately on
// errors that would prevent meaningful progress.
package loop

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/grokify/mogo/log/slogutil"

	"github.com/plexusone/agentpair/internal/agent"
	"github.com/plexusone/agentpair/internal/bridge"
	"github.com/plexusone/agentpair/internal/config"
	"github.com/plexusone/agentpair/internal/logger"
	"github.com/plexusone/agentpair/internal/review"
	"github.com/plexusone/agentpair/internal/run"
)

// Loop orchestrates paired agent execution.
type Loop struct {
	cfg       *config.Config
	run       *run.Run
	primary   agent.Agent
	secondary agent.Agent
	machine   *Machine
	parser    *review.Parser
	consensus *review.Consensus
	mcpServer *bridge.Server
	log       *slog.Logger

	mu        sync.Mutex
	iteration int
	startTime time.Time
}

// New creates a new loop with the given configuration and run.
func New(cfg *config.Config, r *run.Run, primary, secondary agent.Agent) *Loop {
	return &Loop{
		cfg:       cfg,
		run:       r,
		primary:   primary,
		secondary: secondary,
		machine:   NewMachine(),
		parser:    review.NewParser(cfg.DoneSignal),
		consensus: review.NewConsensus(cfg.ReviewMode),
	}
}

// Run executes the main loop until completion or max iterations.
func (l *Loop) Run(ctx context.Context) error {
	l.startTime = time.Now()

	// Initialize logger from context
	l.log = logger.WithComponent(
		logger.WithRunID(logger.FromContext(ctx), l.run.Manifest.ID),
		"loop",
	)
	l.log.Info("starting loop",
		"primary", l.primary.Name(),
		"review_mode", l.cfg.ReviewMode,
		"max_iterations", l.cfg.MaxIterations)

	// Log start
	if err := l.run.Transcript.LogStart(l.cfg.Prompt, map[string]any{
		"primary_agent":  l.primary.Name(),
		"review_mode":    l.cfg.ReviewMode,
		"max_iterations": l.cfg.MaxIterations,
	}); err != nil {
		l.log.Warn("failed to log start to transcript", "error", err)
	}

	// Start MCP server for bridge tools
	if err := l.startMCPServer(ctx); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	defer l.stopMCPServer()
	l.log.Debug("MCP server started", "addr", l.MCPServerAddr())

	// Start agents
	if err := l.startAgents(ctx); err != nil {
		return fmt.Errorf("failed to start agents: %w", err)
	}
	defer l.stopAgents(ctx)
	l.log.Debug("agents started")

	// Transition to working state
	if err := l.machine.Transition(StateWorking); err != nil {
		return err
	}
	l.run.Manifest.SetState(run.StateWorking)
	if err := l.run.Transcript.LogState(run.StateWorking); err != nil {
		l.log.Warn("failed to log state to transcript", "error", err)
	}
	if err := l.run.Save(); err != nil {
		l.log.Warn("failed to save run", "error", err)
	}

	// Send initial task to primary agent
	initialMsg := bridge.NewTaskMessage("system", l.primary.Name(), l.cfg.Prompt)
	initialMsg.WithRunInfo(l.run.Manifest.ID, 0)
	if _, err := l.run.Bridge.Send(ctx, initialMsg); err != nil {
		return fmt.Errorf("failed to send initial task: %w", err)
	}

	// Main loop
	for l.iteration = 1; l.iteration <= l.cfg.MaxIterations; l.iteration++ {
		select {
		case <-ctx.Done():
			l.log.Info("loop cancelled", "reason", ctx.Err())
			return ctx.Err()
		default:
		}

		iterLog := logger.WithIteration(l.log, l.iteration)
		iterLog.Info("starting iteration")

		l.run.Manifest.IncrementIteration()
		if err := l.run.Transcript.LogIteration(l.iteration); err != nil {
			iterLog.Warn("failed to log iteration to transcript", "error", err)
		}

		done, err := l.executeIteration(ctx)
		if err != nil {
			iterLog.Error("iteration failed", "error", err)
			if logErr := l.run.Transcript.LogError("loop", l.iteration, err); logErr != nil {
				iterLog.Warn("failed to log error to transcript", "error", logErr)
			}
			if transErr := l.machine.Transition(StateFailed); transErr != nil {
				iterLog.Warn("failed to transition to failed state", "error", transErr)
			}
			l.run.Manifest.SetState(run.StateFailed)
			l.run.Manifest.Error = err.Error()
			if saveErr := l.run.Save(); saveErr != nil {
				iterLog.Warn("failed to save run", "error", saveErr)
			}
			return err
		}

		if done {
			iterLog.Info("loop completed", "result", "done")
			break
		}
		iterLog.Debug("iteration completed", "continuing", true)
	}

	// Check if we hit max iterations
	if l.iteration > l.cfg.MaxIterations {
		l.log.Warn("max iterations reached", "iterations", l.cfg.MaxIterations)
		if err := l.machine.Transition(StateFailed); err != nil {
			l.log.Warn("failed to transition to failed state", "error", err)
		}
		l.run.Manifest.SetState(run.StateFailed)
		l.run.Manifest.Error = "max iterations reached"
		if err := l.run.Save(); err != nil {
			l.log.Warn("failed to save run", "error", err)
		}
		return errors.New("max iterations reached")
	}

	// Complete
	elapsed := time.Since(l.startTime)
	l.log.Info("loop completed successfully",
		"iterations", l.iteration,
		"elapsed", elapsed)
	if err := l.machine.Transition(StateComplete); err != nil {
		l.log.Warn("failed to transition to complete state", "error", err)
	}
	l.run.Manifest.SetState(run.StateCompleted)
	if err := l.run.Transcript.LogEnd(run.StateCompleted, elapsed); err != nil {
		l.log.Warn("failed to log end to transcript", "error", err)
	}
	if err := l.run.Save(); err != nil {
		l.log.Warn("failed to save run", "error", err)
	}

	return nil
}

func (l *Loop) startAgents(ctx context.Context) error {
	// Set MCP server address on agents
	mcpAddr := l.MCPServerAddr()
	if mcpAddr != "" {
		l.primary.SetMCPServerAddr(mcpAddr)
		if l.secondary != nil {
			l.secondary.SetMCPServerAddr(mcpAddr)
		}
	}

	// Start both agents in parallel
	var wg sync.WaitGroup
	var primaryErr, secondaryErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		primaryErr = l.primary.Start(ctx)
	}()

	if l.secondary != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			secondaryErr = l.secondary.Start(ctx)
		}()
	}

	wg.Wait()

	if primaryErr != nil {
		return fmt.Errorf("primary agent failed to start: %w", primaryErr)
	}
	if secondaryErr != nil {
		return fmt.Errorf("secondary agent failed to start: %w", secondaryErr)
	}

	// Update session IDs in manifest
	l.run.Manifest.ClaudeSessionID = ""
	l.run.Manifest.CodexSessionID = ""
	if l.primary.Name() == "claude" {
		l.run.Manifest.ClaudeSessionID = l.primary.SessionID()
	} else {
		l.run.Manifest.CodexSessionID = l.primary.SessionID()
	}
	if l.secondary != nil {
		if l.secondary.Name() == "claude" {
			l.run.Manifest.ClaudeSessionID = l.secondary.SessionID()
		} else {
			l.run.Manifest.CodexSessionID = l.secondary.SessionID()
		}
	}
	if err := l.run.Save(); err != nil {
		logger := slogutil.LoggerFromContext(ctx, slog.Default())
		logger.Warn("failed to save run after starting agents", "error", err)
	}

	return nil
}

func (l *Loop) stopAgents(ctx context.Context) {
	logger := slogutil.LoggerFromContext(ctx, slog.Default())
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := l.primary.Stop(ctx); err != nil {
			logger.Warn("failed to stop primary agent", "agent", l.primary.Name(), "error", err)
		}
	}()

	if l.secondary != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := l.secondary.Stop(ctx); err != nil {
				logger.Warn("failed to stop secondary agent", "agent", l.secondary.Name(), "error", err)
			}
		}()
	}

	wg.Wait()
}

func (l *Loop) startMCPServer(ctx context.Context) error {
	l.mcpServer = bridge.NewServer(l.run.Bridge)

	// Start server in background on random port
	serverCtx, cancel := context.WithCancel(ctx)
	errCh := make(chan error, 1)

	go func() {
		errCh <- l.mcpServer.ListenAndServe(serverCtx, ":0")
	}()

	// Wait briefly for server to start
	select {
	case err := <-errCh:
		cancel()
		return fmt.Errorf("MCP server failed: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
	}

	// Store cancel func for cleanup (not used directly but context will cancel)
	_ = cancel

	return nil
}

func (l *Loop) stopMCPServer() {
	if l.mcpServer != nil {
		l.mcpServer.Close()
	}
}

// MCPServerAddr returns the MCP server address for agents to connect to.
func (l *Loop) MCPServerAddr() string {
	if l.mcpServer == nil {
		return ""
	}
	return l.mcpServer.Addr()
}

// executeIteration runs one iteration of the agent loop.
// Returns (true, nil) when the task is complete, (false, nil) to continue,
// or (false, error) on critical failures.
//
// Critical errors (returned): agent execution failures, message draining failures.
// Non-critical errors (logged): transcript logging, state transitions, run saves.
func (l *Loop) executeIteration(ctx context.Context) (bool, error) {
	logger := slogutil.LoggerFromContext(ctx, slog.Default())

	// Drain messages for primary agent
	primaryMsgs, err := l.run.Bridge.DrainNew(ctx, l.primary.Name(), "")
	if err != nil {
		return false, fmt.Errorf("failed to drain messages: %w", err)
	}

	// Execute primary agent
	if err := l.run.Transcript.LogExecute(l.primary.Name(), l.iteration, fmt.Sprintf("%d messages", len(primaryMsgs))); err != nil {
		logger.Warn("failed to log execute to transcript", "error", err)
	}
	startTime := time.Now()

	primaryResult, err := l.primary.Execute(ctx, primaryMsgs)
	if err != nil {
		return false, fmt.Errorf("primary agent execution failed: %w", err)
	}

	if err := l.run.Transcript.LogResult(l.primary.Name(), l.iteration, primaryResult.Output, time.Since(startTime)); err != nil {
		logger.Warn("failed to log result to transcript", "error", err)
	}

	// Send result to bridge
	resultMsg := bridge.NewResultMessage(l.primary.Name(), l.secondary.Name(), primaryResult.Output)
	resultMsg.WithRunInfo(l.run.Manifest.ID, l.iteration)
	if _, err := l.run.Bridge.Send(ctx, resultMsg); err != nil {
		logger.Warn("failed to send result to bridge", "error", err)
	}

	// Check for done signal from primary
	if primaryResult.Done {
		if err := l.run.Transcript.LogSignal(l.primary.Name(), l.iteration, "DONE"); err != nil {
			logger.Warn("failed to log signal to transcript", "error", err)
		}

		// If no secondary (single-agent mode), we're done
		if l.secondary == nil {
			return true, nil
		}

		// Transition to reviewing
		if err := l.machine.Transition(StateReviewing); err != nil {
			logger.Warn("failed to transition to reviewing state", "error", err)
		}
		l.run.Manifest.SetState(run.StateReviewing)
		if err := l.run.Transcript.LogState(run.StateReviewing); err != nil {
			logger.Warn("failed to log state to transcript", "error", err)
		}
		if err := l.run.Save(); err != nil {
			logger.Warn("failed to save run", "error", err)
		}
	}

	// If in single-agent mode, check for completion
	if l.secondary == nil {
		return primaryResult.Done, nil
	}

	// Execute secondary agent for review
	secondaryMsgs, err := l.run.Bridge.DrainNew(ctx, l.secondary.Name(), "")
	if err != nil {
		return false, fmt.Errorf("failed to drain secondary messages: %w", err)
	}

	if err := l.run.Transcript.LogExecute(l.secondary.Name(), l.iteration, fmt.Sprintf("%d messages", len(secondaryMsgs))); err != nil {
		logger.Warn("failed to log execute to transcript", "error", err)
	}
	startTime = time.Now()

	secondaryResult, err := l.secondary.Execute(ctx, secondaryMsgs)
	if err != nil {
		return false, fmt.Errorf("secondary agent execution failed: %w", err)
	}

	if err := l.run.Transcript.LogResult(l.secondary.Name(), l.iteration, secondaryResult.Output, time.Since(startTime)); err != nil {
		logger.Warn("failed to log result to transcript", "error", err)
	}

	// Send review result to bridge
	reviewMsg := bridge.NewReviewMessage(l.secondary.Name(), l.primary.Name(), secondaryResult.Output)
	reviewMsg.WithRunInfo(l.run.Manifest.ID, l.iteration)
	if _, err := l.run.Bridge.Send(ctx, reviewMsg); err != nil {
		logger.Warn("failed to send review to bridge", "error", err)
	}

	// Parse review signals
	primaryReview := l.parser.Parse(primaryResult.Output)
	primaryReview.Agent = l.primary.Name()

	secondaryReview := l.parser.Parse(secondaryResult.Output)
	secondaryReview.Agent = l.secondary.Name()

	// Calculate consensus
	var claudeReview, codexReview *review.Result
	if l.primary.Name() == "claude" {
		claudeReview = primaryReview
		codexReview = secondaryReview
	} else {
		claudeReview = secondaryReview
		codexReview = primaryReview
	}

	consensus := l.consensus.Calculate(claudeReview, codexReview)

	// Log signals
	if primaryReview.Signal != review.SignalNone {
		if err := l.run.Transcript.LogSignal(l.primary.Name(), l.iteration, string(primaryReview.Signal)); err != nil {
			logger.Warn("failed to log signal to transcript", "error", err)
		}
	}
	if secondaryReview.Signal != review.SignalNone {
		if err := l.run.Transcript.LogSignal(l.secondary.Name(), l.iteration, string(secondaryReview.Signal)); err != nil {
			logger.Warn("failed to log signal to transcript", "error", err)
		}
	}

	// Check consensus result
	switch consensus.Signal {
	case review.SignalPass:
		// Both approved, check if done signal was given
		if primaryResult.Done || secondaryResult.Done {
			return true, nil
		}
		// Continue working
		if err := l.machine.Transition(StateWorking); err != nil {
			logger.Warn("failed to transition to working state", "error", err)
		}
		l.run.Manifest.SetState(run.StateWorking)
		if err := l.run.Save(); err != nil {
			logger.Warn("failed to save run", "error", err)
		}

	case review.SignalFail:
		// Review failed, continue iterating
		if err := l.machine.Transition(StateWorking); err != nil {
			logger.Warn("failed to transition to working state", "error", err)
		}
		l.run.Manifest.SetState(run.StateWorking)
		if err := l.run.Save(); err != nil {
			logger.Warn("failed to save run", "error", err)
		}

	case review.SignalPending:
		// Waiting for more input, continue
	}

	return false, nil
}

// Resume continues a run from its current state.
func (l *Loop) Resume(ctx context.Context) error {
	// Restore state from manifest
	l.machine = NewMachine()
	l.machine.current = FromRunState(l.run.Manifest.State)
	l.iteration = l.run.Manifest.CurrentIteration

	// Continue the loop
	return l.Run(ctx)
}

// Status returns the current loop status.
func (l *Loop) Status() *Status {
	l.mu.Lock()
	defer l.mu.Unlock()

	return &Status{
		State:         l.machine.Current(),
		Iteration:     l.iteration,
		MaxIterations: l.cfg.MaxIterations,
		RunID:         l.run.Manifest.ID,
		Elapsed:       time.Since(l.startTime),
		BridgeStatus:  l.run.Bridge.Status(),
	}
}

// Status contains loop status information.
type Status struct {
	State         State
	Iteration     int
	MaxIterations int
	RunID         int
	Elapsed       time.Duration
	BridgeStatus  *bridge.Status
}
