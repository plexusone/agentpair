package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Options{
		Output: &buf,
		Level:  slog.LevelInfo,
	})

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected 'test message' in output, got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("expected 'key=value' in output, got: %s", output)
	}
}

func TestNewJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Options{
		Output: &buf,
		JSON:   true,
	})

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("expected JSON msg field, got: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("expected JSON key field, got: %s", output)
	}
}

func TestNewNilOptions(t *testing.T) {
	logger := New(nil)
	if logger == nil {
		t.Error("expected non-nil logger")
	}
}

func TestContext(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Options{Output: &buf})

	ctx := context.Background()
	ctx = NewContext(ctx, logger)

	retrieved := FromContext(ctx)
	retrieved.Info("context test")

	if !strings.Contains(buf.String(), "context test") {
		t.Error("expected logged message from context logger")
	}
}

func TestFromContextDefault(t *testing.T) {
	ctx := context.Background()
	logger := FromContext(ctx)

	// Should return default logger, not nil
	if logger == nil {
		t.Error("expected non-nil default logger")
	}
}

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Options{Output: &buf})

	componentLogger := WithComponent(logger, "bridge")
	componentLogger.Info("test")

	if !strings.Contains(buf.String(), "component=bridge") {
		t.Errorf("expected component field, got: %s", buf.String())
	}
}

func TestWithRunID(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Options{Output: &buf})

	runLogger := WithRunID(logger, 42)
	runLogger.Info("test")

	if !strings.Contains(buf.String(), "run_id=42") {
		t.Errorf("expected run_id field, got: %s", buf.String())
	}
}

func TestWithAgent(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Options{Output: &buf})

	agentLogger := WithAgent(logger, "claude")
	agentLogger.Info("test")

	if !strings.Contains(buf.String(), "agent=claude") {
		t.Errorf("expected agent field, got: %s", buf.String())
	}
}

func TestWithIteration(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Options{Output: &buf})

	iterLogger := WithIteration(logger, 5)
	iterLogger.Info("test")

	if !strings.Contains(buf.String(), "iteration=5") {
		t.Errorf("expected iteration field, got: %s", buf.String())
	}
}

func TestNop(t *testing.T) {
	logger := Nop()
	// Should not panic
	logger.Info("this should be discarded")
}

func TestLogLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Options{
		Output: &buf,
		Level:  slog.LevelWarn,
	})

	logger.Info("info message")   // Should be filtered
	logger.Warn("warn message")   // Should appear
	logger.Error("error message") // Should appear

	output := buf.String()
	if strings.Contains(output, "info message") {
		t.Error("info message should be filtered at Warn level")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("warn message should appear")
	}
	if !strings.Contains(output, "error message") {
		t.Error("error message should appear")
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Options{Output: &buf})

	withLogger := With(logger, "custom", "attr", "number", 123)
	withLogger.Info("test")

	output := buf.String()
	if !strings.Contains(output, "custom=attr") {
		t.Errorf("expected custom field, got: %s", output)
	}
	if !strings.Contains(output, "number=123") {
		t.Errorf("expected number field, got: %s", output)
	}
}

func TestChainedWith(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Options{Output: &buf})

	chainedLogger := WithComponent(WithRunID(WithAgent(logger, "claude"), 1), "loop")
	chainedLogger.Info("test")

	output := buf.String()
	if !strings.Contains(output, "agent=claude") {
		t.Errorf("expected agent field, got: %s", output)
	}
	if !strings.Contains(output, "run_id=1") {
		t.Errorf("expected run_id field, got: %s", output)
	}
	if !strings.Contains(output, "component=loop") {
		t.Errorf("expected component field, got: %s", output)
	}
}
