// Package config provides configuration types and path management.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
)

const (
	// DefaultBaseDir is the default base directory for agentpair data.
	DefaultBaseDir = ".agentpair"
	// RunsDir is the subdirectory for run data.
	RunsDir = "runs"
)

// Paths manages file paths for agentpair data storage.
type Paths struct {
	baseDir string
}

// NewPaths creates a new Paths instance with the default base directory (~/.agentpair).
func NewPaths() (*Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &Paths{
		baseDir: filepath.Join(home, DefaultBaseDir),
	}, nil
}

// NewPathsWithBase creates a new Paths instance with a custom base directory.
func NewPathsWithBase(baseDir string) *Paths {
	return &Paths{baseDir: baseDir}
}

// BaseDir returns the base directory path.
func (p *Paths) BaseDir() string {
	return p.baseDir
}

// RunsDir returns the path to the runs directory.
func (p *Paths) RunsDir() string {
	return filepath.Join(p.baseDir, RunsDir)
}

// RepoDir returns the path to a repository's run directory.
// The repoPath is hashed to create a safe directory name.
func (p *Paths) RepoDir(repoPath string) string {
	repoID := hashPath(repoPath)
	return filepath.Join(p.RunsDir(), repoID)
}

// RunDir returns the path to a specific run's directory.
func (p *Paths) RunDir(repoPath string, runID int) string {
	return filepath.Join(p.RepoDir(repoPath), strconv.Itoa(runID))
}

// ManifestPath returns the path to the manifest.json file for a run.
func (p *Paths) ManifestPath(repoPath string, runID int) string {
	return filepath.Join(p.RunDir(repoPath, runID), "manifest.json")
}

// TranscriptPath returns the path to the transcript.jsonl file for a run.
func (p *Paths) TranscriptPath(repoPath string, runID int) string {
	return filepath.Join(p.RunDir(repoPath, runID), "transcript.jsonl")
}

// BridgePath returns the path to the bridge.jsonl file for a run.
func (p *Paths) BridgePath(repoPath string, runID int) string {
	return filepath.Join(p.RunDir(repoPath, runID), "bridge.jsonl")
}

// EnsureRunDir creates the run directory if it doesn't exist.
func (p *Paths) EnsureRunDir(repoPath string, runID int) error {
	dir := p.RunDir(repoPath, runID)
	return os.MkdirAll(dir, 0755)
}

// NextRunID returns the next available run ID for a repository.
func (p *Paths) NextRunID(repoPath string) (int, error) {
	repoDir := p.RepoDir(repoPath)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return 0, err
	}

	maxID := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1, nil
}

// hashPath creates a short hash of a path for use as a directory name.
func hashPath(path string) string {
	h := sha256.Sum256([]byte(path))
	return hex.EncodeToString(h[:8]) // First 8 bytes = 16 hex chars
}

// RepoIDFromPath returns the hashed repo ID for a path.
func RepoIDFromPath(path string) string {
	return hashPath(path)
}
