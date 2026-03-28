// Package logger provides structured logging utilities for agentpair.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// contextKey is the type for context keys.
type contextKey int

const (
	loggerKey contextKey = iota
)

// New creates a new logger with the given options.
func New(opts *Options) *slog.Logger {
	if opts == nil {
		opts = &Options{}
	}

	level := slog.LevelInfo
	if opts.Level != 0 {
		level = opts.Level
	}

	output := opts.Output
	if output == nil {
		output = os.Stderr
	}

	var handler slog.Handler
	if opts.JSON {
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{
			Level:     level,
			AddSource: opts.AddSource,
		})
	} else {
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{
			Level:     level,
			AddSource: opts.AddSource,
		})
	}

	return slog.New(handler)
}

// Options configures the logger.
type Options struct {
	// Level is the minimum log level. Default is Info.
	Level slog.Level
	// Output is the output writer. Default is os.Stderr.
	Output io.Writer
	// JSON enables JSON output format.
	JSON bool
	// AddSource adds source file and line to logs.
	AddSource bool
}

// NewContext returns a context with the logger attached.
func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from the context.
// Returns the default logger if none is set.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// With returns a logger with the given attributes added.
func With(logger *slog.Logger, args ...any) *slog.Logger {
	return logger.With(args...)
}

// WithComponent returns a logger with a component attribute added.
func WithComponent(logger *slog.Logger, component string) *slog.Logger {
	return logger.With("component", component)
}

// WithRunID returns a logger with a run_id attribute added.
func WithRunID(logger *slog.Logger, runID int) *slog.Logger {
	return logger.With("run_id", runID)
}

// WithAgent returns a logger with an agent attribute added.
func WithAgent(logger *slog.Logger, agent string) *slog.Logger {
	return logger.With("agent", agent)
}

// WithIteration returns a logger with an iteration attribute added.
func WithIteration(logger *slog.Logger, iteration int) *slog.Logger {
	return logger.With("iteration", iteration)
}

// Nop returns a logger that discards all output.
func Nop() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
