// Package update provides auto-update mechanism for the agentpair binary.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// GitHubRepo is the repository for releases.
	GitHubRepo = "plexusone/agentpair"

	// ReleasesURL is the GitHub releases API endpoint.
	ReleasesURL = "https://api.github.com/repos/" + GitHubRepo + "/releases/latest"

	// CheckInterval is how often to check for updates.
	CheckInterval = 24 * time.Hour
)

// Version is the current version (set at build time).
var Version = "dev"

// Updater handles checking and applying updates.
type Updater struct {
	currentVersion string
	httpClient     *http.Client
	lastCheckFile  string
}

// New creates a new updater.
func New(currentVersion string) *Updater {
	homeDir, _ := os.UserHomeDir()
	return &Updater{
		currentVersion: currentVersion,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		lastCheckFile: filepath.Join(homeDir, ".agentpair", ".last_update_check"),
	}
}

// Release represents a GitHub release.
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckResult contains the result of an update check.
type CheckResult struct {
	UpdateAvailable bool
	CurrentVersion  string
	LatestVersion   string
	DownloadURL     string
	ReleaseNotes    string
}

// ShouldCheck returns true if enough time has passed since last check.
func (u *Updater) ShouldCheck() bool {
	data, err := os.ReadFile(u.lastCheckFile)
	if err != nil {
		return true
	}

	var lastCheck time.Time
	if err := json.Unmarshal(data, &lastCheck); err != nil {
		return true
	}

	return time.Since(lastCheck) > CheckInterval
}

// Check checks for available updates.
func (u *Updater) Check(ctx context.Context) (*CheckResult, error) {
	result := &CheckResult{
		CurrentVersion: u.currentVersion,
	}

	// Record check time
	u.recordCheck()

	// Fetch latest release
	req, err := http.NewRequestWithContext(ctx, "GET", ReleasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "agentpair/"+u.currentVersion)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	result.LatestVersion = strings.TrimPrefix(release.TagName, "v")
	result.ReleaseNotes = release.Body

	// Compare versions
	if u.isNewer(result.LatestVersion, u.currentVersion) {
		result.UpdateAvailable = true
		result.DownloadURL = u.findAssetURL(release.Assets)
	}

	return result, nil
}

func (u *Updater) recordCheck() {
	dir := filepath.Dir(u.lastCheckFile)
	os.MkdirAll(dir, 0755)

	data, _ := json.Marshal(time.Now())
	os.WriteFile(u.lastCheckFile, data, 0644)
}

func (u *Updater) isNewer(latest, current string) bool {
	// Simple version comparison (assumes semver-like versions)
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	if current == "dev" || current == "" {
		return false // Don't update dev versions
	}

	return latest != current && latest > current
}

func (u *Updater) findAssetURL(assets []Asset) string {
	// Build expected asset name
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// Map arch names
	if archName == "amd64" {
		archName = "x86_64"
	} else if archName == "arm64" {
		archName = "aarch64"
	}

	expected := fmt.Sprintf("agentpair_%s_%s", osName, archName)

	for _, asset := range assets {
		if strings.Contains(strings.ToLower(asset.Name), strings.ToLower(expected)) {
			return asset.BrowserDownloadURL
		}
	}

	return ""
}

// Update downloads and installs the update.
func (u *Updater) Update(ctx context.Context, downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("no download URL available")
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Download new binary
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp(filepath.Dir(execPath), "agentpair-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Download to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write update: %w", err)
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Backup current binary
	backupPath := execPath + ".bak"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary into place
	if err := os.Rename(tmpPath, execPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)

	return nil
}

// PrintUpdateNotice prints a notice if an update is available.
func (u *Updater) PrintUpdateNotice(ctx context.Context) {
	if !u.ShouldCheck() {
		return
	}

	result, err := u.Check(ctx)
	if err != nil {
		return // Silently ignore errors
	}

	if result.UpdateAvailable {
		fmt.Fprintf(os.Stderr, "\n📦 Update available: %s → %s\n", result.CurrentVersion, result.LatestVersion)
		fmt.Fprintf(os.Stderr, "   Run 'agentpair update' to install\n\n")
	}
}

// GetVersion returns the current version.
func GetVersion() string {
	return Version
}
