package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

// Config holds the runtime configuration for agentpair.
type Config struct {
	// Prompt is the initial task prompt.
	Prompt string

	// Agent specifies the primary worker agent ("claude" or "codex").
	Agent string

	// MaxIterations is the maximum number of loop iterations.
	MaxIterations int

	// Proof specifies the proof requirements (e.g., "run tests").
	Proof string

	// ReviewMode specifies who reviews: "claude", "codex", or "claudex".
	ReviewMode string

	// DoneSignal is the custom done signal to look for.
	DoneSignal string

	// UseTmux enables tmux side-by-side panes.
	UseTmux bool

	// UseWorktree enables git worktree isolation.
	UseWorktree bool

	// ClaudeOnly runs Claude in single-agent mode.
	ClaudeOnly bool

	// CodexOnly runs Codex in single-agent mode.
	CodexOnly bool

	// RunID is the run ID to resume.
	RunID int

	// SessionID is the session ID to resume.
	SessionID string

	// RepoPath is the path to the repository.
	RepoPath string

	// Verbose enables verbose output.
	Verbose bool

	// Timeout is the maximum time for the entire run.
	Timeout time.Duration
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Agent:         "codex",
		MaxIterations: 20,
		ReviewMode:    "claudex",
		DoneSignal:    "DONE",
		Timeout:       2 * time.Hour,
	}
}

// ReviewMode constants.
const (
	ReviewModeClaude  = "claude"
	ReviewModeCodex   = "codex"
	ReviewModeClaudex = "claudex"
)

// Agent constants.
const (
	AgentClaude = "claude"
	AgentCodex  = "codex"
)

// IsSingleAgentMode returns true if running in single-agent mode.
func (c *Config) IsSingleAgentMode() bool {
	return c.ClaudeOnly || c.CodexOnly
}

// PrimaryAgent returns the name of the primary worker agent.
func (c *Config) PrimaryAgent() string {
	if c.ClaudeOnly {
		return AgentClaude
	}
	if c.CodexOnly {
		return AgentCodex
	}
	return c.Agent
}

// SecondaryAgent returns the name of the secondary/reviewer agent.
func (c *Config) SecondaryAgent() string {
	if c.Agent == AgentClaude {
		return AgentCodex
	}
	return AgentClaude
}

// FileConfig represents configuration that can be loaded from a file.
// This is a subset of Config that makes sense to persist.
type FileConfig struct {
	// Agent specifies the primary worker agent ("claude" or "codex").
	Agent string `json:"agent,omitempty" yaml:"agent,omitempty"`

	// MaxIterations is the maximum number of loop iterations.
	MaxIterations int `json:"max_iterations,omitempty" yaml:"max_iterations,omitempty"`

	// Proof specifies the proof requirements (e.g., "run tests").
	Proof string `json:"proof,omitempty" yaml:"proof,omitempty"`

	// ReviewMode specifies who reviews: "claude", "codex", or "claudex".
	ReviewMode string `json:"review_mode,omitempty" yaml:"review_mode,omitempty"`

	// DoneSignal is the custom done signal to look for.
	DoneSignal string `json:"done_signal,omitempty" yaml:"done_signal,omitempty"`

	// UseTmux enables tmux side-by-side panes.
	UseTmux bool `json:"use_tmux,omitempty" yaml:"use_tmux,omitempty"`

	// UseWorktree enables git worktree isolation.
	UseWorktree bool `json:"use_worktree,omitempty" yaml:"use_worktree,omitempty"`

	// Verbose enables verbose output.
	Verbose bool `json:"verbose,omitempty" yaml:"verbose,omitempty"`

	// Timeout is the maximum time for the entire run (as duration string).
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// ConfigFilenames are the supported config file names in order of preference.
var ConfigFilenames = []string{"config.yaml", "config.yml", "config.json"}

// LoadConfigFile loads configuration from the default config file location.
// Returns nil if no config file exists.
func LoadConfigFile() (*FileConfig, error) {
	paths, err := NewPaths()
	if err != nil {
		return nil, err
	}

	for _, filename := range ConfigFilenames {
		configPath := filepath.Join(paths.BaseDir(), filename)
		cfg, err := LoadConfigFromPath(configPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		return cfg, nil
	}

	return nil, nil // No config file found
}

// LoadConfigFromPath loads configuration from a specific file path.
func LoadConfigFromPath(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg FileConfig
	ext := filepath.Ext(path)

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			if err := json.Unmarshal(data, &cfg); err != nil {
				return nil, errors.New("config file must be YAML or JSON")
			}
		}
	}

	return &cfg, nil
}

// Apply applies file config values to a Config, only overriding non-zero values.
func (fc *FileConfig) Apply(c *Config) {
	if fc == nil {
		return
	}

	if fc.Agent != "" {
		c.Agent = fc.Agent
	}
	if fc.MaxIterations > 0 {
		c.MaxIterations = fc.MaxIterations
	}
	if fc.Proof != "" {
		c.Proof = fc.Proof
	}
	if fc.ReviewMode != "" {
		c.ReviewMode = fc.ReviewMode
	}
	if fc.DoneSignal != "" {
		c.DoneSignal = fc.DoneSignal
	}
	if fc.UseTmux {
		c.UseTmux = fc.UseTmux
	}
	if fc.UseWorktree {
		c.UseWorktree = fc.UseWorktree
	}
	if fc.Verbose {
		c.Verbose = fc.Verbose
	}
	if fc.Timeout != "" {
		if d, err := time.ParseDuration(fc.Timeout); err == nil {
			c.Timeout = d
		}
	}
}

// SaveConfigFile saves configuration to the default config file location.
func SaveConfigFile(cfg *FileConfig) error {
	paths, err := NewPaths()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(paths.BaseDir(), 0755); err != nil {
		return err
	}

	configPath := filepath.Join(paths.BaseDir(), "config.yaml")
	return SaveConfigToPath(configPath, cfg)
}

// SaveConfigToPath saves configuration to a specific file path.
func SaveConfigToPath(path string, cfg *FileConfig) error {
	var data []byte
	var err error

	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		data, err = json.MarshalIndent(cfg, "", "  ")
	default:
		data, err = yaml.Marshal(cfg)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
