package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Installer handles downloading, installing, and removing plugins
// from the local filesystem. Plugins are stored as executables in
// the plugin directory (~/.cloudmock/plugins/ by default).
type Installer struct {
	pluginDir  string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewInstaller creates an Installer that manages plugins in the given directory.
func NewInstaller(pluginDir string, logger *slog.Logger) *Installer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Installer{
		pluginDir:  pluginDir,
		httpClient: &http.Client{Timeout: 2 * time.Minute},
		logger:     logger,
	}
}

// PluginDir returns the directory where plugins are installed.
func (inst *Installer) PluginDir() string {
	return inst.pluginDir
}

// InstalledPlugins returns metadata for all installed plugins by reading
// manifest files in the plugin directory.
func (inst *Installer) InstalledPlugins() ([]InstalledPlugin, error) {
	entries, err := os.ReadDir(inst.pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugin dir: %w", err)
	}

	var plugins []InstalledPlugin
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest := filepath.Join(inst.pluginDir, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifest)
		if err != nil {
			continue // Not a managed plugin
		}
		var p InstalledPlugin
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		plugins = append(plugins, p)
	}
	return plugins, nil
}

// Install downloads and installs a plugin from its registry listing.
// It downloads the binary for the current platform from the GitHub release,
// saves it to the plugin directory, and writes a manifest.
func (inst *Installer) Install(ctx context.Context, listing *PluginListing) error {
	if listing.RepoURL == "" {
		return fmt.Errorf("plugin %s has no repo URL", listing.ID)
	}

	pluginPath := filepath.Join(inst.pluginDir, listing.ID)
	if err := os.MkdirAll(pluginPath, 0o755); err != nil {
		return fmt.Errorf("create plugin dir: %w", err)
	}

	// Determine download URL from repo + version + platform
	downloadURL := inst.resolveDownloadURL(listing)

	inst.logger.Info("downloading plugin",
		"plugin", listing.ID,
		"version", listing.Version,
		"url", downloadURL,
	)

	// Download the binary
	binaryName := listing.ID
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(pluginPath, binaryName)

	if err := inst.downloadFile(ctx, downloadURL, binaryPath); err != nil {
		// Clean up on failure
		os.RemoveAll(pluginPath)
		return fmt.Errorf("download plugin binary: %w", err)
	}

	// Make executable
	if err := os.Chmod(binaryPath, 0o755); err != nil {
		os.RemoveAll(pluginPath)
		return fmt.Errorf("chmod plugin binary: %w", err)
	}

	// Write manifest
	manifest := InstalledPlugin{
		ID:          listing.ID,
		Name:        listing.Name,
		Version:     listing.Version,
		Author:      listing.Author,
		Category:    listing.Category,
		BinaryPath:  binaryPath,
		InstalledAt: time.Now().UTC(),
	}
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	manifestPath := filepath.Join(pluginPath, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		os.RemoveAll(pluginPath)
		return fmt.Errorf("write manifest: %w", err)
	}

	inst.logger.Info("plugin installed",
		"plugin", listing.ID,
		"version", listing.Version,
		"path", binaryPath,
	)
	return nil
}

// Uninstall removes a plugin from the filesystem.
func (inst *Installer) Uninstall(id string) error {
	pluginPath := filepath.Join(inst.pluginDir, id)
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin %s is not installed", id)
	}
	if err := os.RemoveAll(pluginPath); err != nil {
		return fmt.Errorf("remove plugin: %w", err)
	}
	inst.logger.Info("plugin uninstalled", "plugin", id)
	return nil
}

// IsInstalled checks if a plugin is installed locally.
func (inst *Installer) IsInstalled(id string) bool {
	manifest := filepath.Join(inst.pluginDir, id, "manifest.json")
	_, err := os.Stat(manifest)
	return err == nil
}

// InstalledPlugin describes a locally installed plugin.
type InstalledPlugin struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Author      string    `json:"author"`
	Category    string    `json:"category"`
	BinaryPath  string    `json:"binary_path"`
	InstalledAt time.Time `json:"installed_at"`
}

// resolveDownloadURL builds a GitHub release download URL for the current platform.
func (inst *Installer) resolveDownloadURL(listing *PluginListing) string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	ext := ""
	if os == "windows" {
		ext = ".exe"
	}

	// Convention: releases at {repoURL}/releases/download/v{version}/{id}-{os}-{arch}{ext}
	repoURL := listing.RepoURL
	return fmt.Sprintf("%s/releases/download/v%s/%s-%s-%s%s",
		repoURL, listing.Version, listing.ID, os, arch, ext)
}

// downloadFile downloads a URL to a local file path.
func (inst *Installer) downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := inst.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}
