// Package run provides run management and state persistence.
package run

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/plexusone/agentpair/internal/bridge"
	"github.com/plexusone/agentpair/internal/config"
)

// Manager handles run lifecycle and persistence.
type Manager struct {
	paths    *config.Paths
	repoPath string
}

// NewManager creates a new run manager.
func NewManager(paths *config.Paths, repoPath string) *Manager {
	return &Manager{
		paths:    paths,
		repoPath: repoPath,
	}
}

// Run represents an active run with all its resources.
type Run struct {
	Manifest   *Manifest
	Transcript *Transcript
	Bridge     *bridge.Bridge

	paths    *config.Paths
	repoPath string
}

// Create creates a new run.
func (m *Manager) Create(prompt string, cfg *config.Config) (*Run, error) {
	runID, err := m.paths.NextRunID(m.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get next run ID: %w", err)
	}

	if err := m.paths.EnsureRunDir(m.repoPath, runID); err != nil {
		return nil, fmt.Errorf("failed to create run directory: %w", err)
	}

	repoID := config.RepoIDFromPath(m.repoPath)
	manifest := NewManifest(runID, m.repoPath, repoID, prompt)

	// Apply config
	manifest.PrimaryAgent = cfg.PrimaryAgent()
	manifest.ReviewMode = cfg.ReviewMode
	manifest.MaxIterations = cfg.MaxIterations
	manifest.DoneSignal = cfg.DoneSignal
	if cfg.Proof != "" {
		manifest.ProofCommand = cfg.Proof
	}

	manifestPath := m.paths.ManifestPath(m.repoPath, runID)
	if err := manifest.Save(manifestPath); err != nil {
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	bridgePath := m.paths.BridgePath(m.repoPath, runID)
	b := bridge.New(bridgePath, runID)
	if err := b.Open(); err != nil {
		return nil, fmt.Errorf("failed to open bridge: %w", err)
	}

	transcriptPath := m.paths.TranscriptPath(m.repoPath, runID)
	transcript := NewTranscript(transcriptPath, runID)

	return &Run{
		Manifest:   manifest,
		Transcript: transcript,
		Bridge:     b,
		paths:      m.paths,
		repoPath:   m.repoPath,
	}, nil
}

// Load loads an existing run by ID.
func (m *Manager) Load(runID int) (*Run, error) {
	manifestPath := m.paths.ManifestPath(m.repoPath, runID)
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	bridgePath := m.paths.BridgePath(m.repoPath, runID)
	b := bridge.New(bridgePath, runID)
	if err := b.Open(); err != nil {
		return nil, fmt.Errorf("failed to open bridge: %w", err)
	}

	transcriptPath := m.paths.TranscriptPath(m.repoPath, runID)
	transcript := NewTranscript(transcriptPath, runID)

	return &Run{
		Manifest:   manifest,
		Transcript: transcript,
		Bridge:     b,
		paths:      m.paths,
		repoPath:   m.repoPath,
	}, nil
}

// List returns all run IDs for the repository.
func (m *Manager) List() ([]int, error) {
	repoDir := m.paths.RepoDir(m.repoPath)
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var ids []int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ListActive returns all active (non-complete) runs.
func (m *Manager) ListActive() ([]*Manifest, error) {
	ids, err := m.List()
	if err != nil {
		return nil, err
	}

	var active []*Manifest
	for _, id := range ids {
		manifestPath := m.paths.ManifestPath(m.repoPath, id)
		manifest, err := LoadManifest(manifestPath)
		if err != nil {
			continue
		}
		if !manifest.IsComplete() {
			active = append(active, manifest)
		}
	}
	return active, nil
}

// Save saves the run's manifest.
func (r *Run) Save() error {
	manifestPath := r.paths.ManifestPath(r.repoPath, r.Manifest.ID)
	return r.Manifest.Save(manifestPath)
}

// Close closes the run's resources.
func (r *Run) Close() error {
	return r.Bridge.Close()
}

// Dir returns the run's directory path.
func (r *Run) Dir() string {
	return r.paths.RunDir(r.repoPath, r.Manifest.ID)
}

// FindBySessionID finds a run by agent session ID.
func (m *Manager) FindBySessionID(sessionID string) (*Run, error) {
	ids, err := m.List()
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		manifestPath := m.paths.ManifestPath(m.repoPath, id)
		manifest, err := LoadManifest(manifestPath)
		if err != nil {
			continue
		}
		if manifest.ClaudeSessionID == sessionID || manifest.CodexSessionID == sessionID {
			return m.Load(id)
		}
	}
	return nil, fmt.Errorf("no run found with session ID: %s", sessionID)
}

// ListAllRepos returns all repository IDs with runs.
func ListAllRepos(paths *config.Paths) ([]string, error) {
	runsDir := paths.RunsDir()
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var repoIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			repoIDs = append(repoIDs, entry.Name())
		}
	}
	return repoIDs, nil
}

// LoadManifestByPath loads a manifest directly from a directory path.
func LoadManifestByPath(runDir string) (*Manifest, error) {
	manifestPath := filepath.Join(runDir, "manifest.json")
	return LoadManifest(manifestPath)
}
