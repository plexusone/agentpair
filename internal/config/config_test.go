package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agent != "codex" {
		t.Errorf("expected default agent 'codex', got %s", cfg.Agent)
	}
	if cfg.MaxIterations != 20 {
		t.Errorf("expected default max iterations 20, got %d", cfg.MaxIterations)
	}
	if cfg.ReviewMode != "claudex" {
		t.Errorf("expected default review mode 'claudex', got %s", cfg.ReviewMode)
	}
	if cfg.DoneSignal != "DONE" {
		t.Errorf("expected default done signal 'DONE', got %s", cfg.DoneSignal)
	}
}

func TestConfigIsSingleAgentMode(t *testing.T) {
	tests := []struct {
		name       string
		claudeOnly bool
		codexOnly  bool
		expected   bool
	}{
		{"default", false, false, false},
		{"claude only", true, false, true},
		{"codex only", false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ClaudeOnly: tt.claudeOnly,
				CodexOnly:  tt.codexOnly,
			}
			if got := cfg.IsSingleAgentMode(); got != tt.expected {
				t.Errorf("IsSingleAgentMode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfigPrimaryAgent(t *testing.T) {
	tests := []struct {
		name       string
		agent      string
		claudeOnly bool
		codexOnly  bool
		expected   string
	}{
		{"default codex", "codex", false, false, "codex"},
		{"set to claude", "claude", false, false, "claude"},
		{"claude only override", "codex", true, false, "claude"},
		{"codex only override", "claude", false, true, "codex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Agent:      tt.agent,
				ClaudeOnly: tt.claudeOnly,
				CodexOnly:  tt.codexOnly,
			}
			if got := cfg.PrimaryAgent(); got != tt.expected {
				t.Errorf("PrimaryAgent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadConfigFromPath_YAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `agent: claude
max_iterations: 50
review_mode: codex
done_signal: FINISHED
use_tmux: true
verbose: true
timeout: 30m
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath failed: %v", err)
	}

	if cfg.Agent != "claude" {
		t.Errorf("Agent = %s, want claude", cfg.Agent)
	}
	if cfg.MaxIterations != 50 {
		t.Errorf("MaxIterations = %d, want 50", cfg.MaxIterations)
	}
	if cfg.ReviewMode != "codex" {
		t.Errorf("ReviewMode = %s, want codex", cfg.ReviewMode)
	}
	if cfg.DoneSignal != "FINISHED" {
		t.Errorf("DoneSignal = %s, want FINISHED", cfg.DoneSignal)
	}
	if !cfg.UseTmux {
		t.Error("UseTmux should be true")
	}
	if !cfg.Verbose {
		t.Error("Verbose should be true")
	}
	if cfg.Timeout != "30m" {
		t.Errorf("Timeout = %s, want 30m", cfg.Timeout)
	}
}

func TestLoadConfigFromPath_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	content := `{
  "agent": "codex",
  "max_iterations": 100,
  "review_mode": "claudex"
}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath failed: %v", err)
	}

	if cfg.Agent != "codex" {
		t.Errorf("Agent = %s, want codex", cfg.Agent)
	}
	if cfg.MaxIterations != 100 {
		t.Errorf("MaxIterations = %d, want 100", cfg.MaxIterations)
	}
}

func TestLoadConfigFromPath_NotExist(t *testing.T) {
	_, err := LoadConfigFromPath("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestFileConfigApply(t *testing.T) {
	cfg := DefaultConfig()

	fileCfg := &FileConfig{
		Agent:         "claude",
		MaxIterations: 30,
		Timeout:       "1h",
	}

	fileCfg.Apply(cfg)

	if cfg.Agent != "claude" {
		t.Errorf("Agent should be claude, got %s", cfg.Agent)
	}
	if cfg.MaxIterations != 30 {
		t.Errorf("MaxIterations should be 30, got %d", cfg.MaxIterations)
	}
	if cfg.Timeout != time.Hour {
		t.Errorf("Timeout should be 1h, got %v", cfg.Timeout)
	}
	// Default values should be preserved
	if cfg.ReviewMode != "claudex" {
		t.Errorf("ReviewMode should be preserved as claudex, got %s", cfg.ReviewMode)
	}
}

func TestFileConfigApply_Nil(t *testing.T) {
	cfg := DefaultConfig()
	var fileCfg *FileConfig = nil

	// Should not panic
	fileCfg.Apply(cfg)

	// Config should be unchanged
	if cfg.Agent != "codex" {
		t.Errorf("Agent should be unchanged")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := &FileConfig{
		Agent:         "claude",
		MaxIterations: 25,
		ReviewMode:    "codex",
		UseTmux:       true,
	}

	if err := SaveConfigToPath(path, original); err != nil {
		t.Fatalf("SaveConfigToPath failed: %v", err)
	}

	loaded, err := LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath failed: %v", err)
	}

	if loaded.Agent != original.Agent {
		t.Errorf("Agent mismatch: got %s, want %s", loaded.Agent, original.Agent)
	}
	if loaded.MaxIterations != original.MaxIterations {
		t.Errorf("MaxIterations mismatch: got %d, want %d", loaded.MaxIterations, original.MaxIterations)
	}
	if loaded.ReviewMode != original.ReviewMode {
		t.Errorf("ReviewMode mismatch: got %s, want %s", loaded.ReviewMode, original.ReviewMode)
	}
	if loaded.UseTmux != original.UseTmux {
		t.Errorf("UseTmux mismatch: got %v, want %v", loaded.UseTmux, original.UseTmux)
	}
}

func TestSaveConfigToPath_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := &FileConfig{
		Agent: "codex",
	}

	if err := SaveConfigToPath(path, cfg); err != nil {
		t.Fatalf("SaveConfigToPath failed: %v", err)
	}

	// Verify it's valid JSON
	loaded, err := LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath failed: %v", err)
	}

	if loaded.Agent != "codex" {
		t.Errorf("Agent = %s, want codex", loaded.Agent)
	}
}
