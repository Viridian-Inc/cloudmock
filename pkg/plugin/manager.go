package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Manager discovers, loads, and manages plugin lifecycle.
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]*managedPlugin // keyed by plugin name
	logger  *slog.Logger
}

type managedPlugin struct {
	plugin     Plugin
	descriptor *Descriptor
	mode       PluginMode
	healthy    bool
	lastCheck  time.Time
}

// PluginMode indicates how a plugin is loaded.
type PluginMode int

const (
	ModeInProcess    PluginMode = iota // Go plugin loaded in-process
	ModeExternalGRPC                   // External binary via gRPC over stdio
)

// String returns a human-readable name for the mode.
func (m PluginMode) String() string {
	switch m {
	case ModeInProcess:
		return "in-process"
	case ModeExternalGRPC:
		return "external-grpc"
	default:
		return "unknown"
	}
}

// PluginInfo holds metadata about a loaded plugin for the admin API.
type PluginInfo struct {
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Protocol  string            `json:"protocol"`
	Mode      string            `json:"mode"`
	Healthy   bool              `json:"healthy"`
	Actions   []string          `json:"actions"`
	APIPaths  []string          `json:"api_paths,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	LastCheck time.Time         `json:"last_check"`
}

// NewManager creates a plugin manager.
func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		plugins: make(map[string]*managedPlugin),
		logger:  logger,
	}
}

// RegisterInProcess registers a Plugin that runs in-process (Go).
func (m *Manager) RegisterInProcess(ctx context.Context, p Plugin) error {
	desc, err := p.Describe(ctx)
	if err != nil {
		return fmt.Errorf("plugin describe: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[desc.Name]; exists {
		return fmt.Errorf("plugin %q already registered", desc.Name)
	}

	m.plugins[desc.Name] = &managedPlugin{
		plugin:     p,
		descriptor: desc,
		mode:       ModeInProcess,
		healthy:    true,
		lastCheck:  time.Now(),
	}

	m.logger.Info("registered plugin", "name", desc.Name, "version", desc.Version, "mode", "in-process", "actions", len(desc.Actions), "paths", desc.APIPaths)
	return nil
}

// Lookup returns the plugin registered under the given name.
func (m *Manager) Lookup(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mp, ok := m.plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin: no plugin registered for %q", name)
	}
	return mp.plugin, nil
}

// LookupByPath finds the plugin whose api_paths best match the given URL path.
// Returns nil if no path-based plugin matches.
func (m *Manager) LookupByPath(path string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var bestPlugin *managedPlugin
	bestLen := 0

	for _, mp := range m.plugins {
		for _, pattern := range mp.descriptor.APIPaths {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(path, prefix) && len(prefix) > bestLen {
				bestPlugin = mp
				bestLen = len(prefix)
			}
		}
	}

	if bestPlugin == nil {
		return nil, nil
	}
	return bestPlugin.plugin, nil
}

// List returns info about all registered plugins.
func (m *Manager) List() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(m.plugins))
	for _, mp := range m.plugins {
		infos = append(infos, PluginInfo{
			Name:      mp.descriptor.Name,
			Version:   mp.descriptor.Version,
			Protocol:  mp.descriptor.Protocol,
			Mode:      mp.mode.String(),
			Healthy:   mp.healthy,
			Actions:   mp.descriptor.Actions,
			APIPaths:  mp.descriptor.APIPaths,
			Metadata:  mp.descriptor.Metadata,
			LastCheck: mp.lastCheck,
		})
	}
	return infos
}

// HealthCheckAll runs health checks on all registered plugins and updates their status.
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]HealthCheckResult {
	m.mu.RLock()
	plugins := make(map[string]*managedPlugin, len(m.plugins))
	for k, v := range m.plugins {
		plugins[k] = v
	}
	m.mu.RUnlock()

	results := make(map[string]HealthCheckResult, len(plugins))
	for name, mp := range plugins {
		status, msg, err := mp.plugin.HealthCheck(ctx)
		if err != nil {
			status = HealthUnhealthy
			msg = err.Error()
		}

		results[name] = HealthCheckResult{
			Name:    name,
			Status:  status,
			Message: msg,
		}

		m.mu.Lock()
		if mp2, ok := m.plugins[name]; ok {
			mp2.healthy = status == HealthHealthy || status == HealthDegraded
			mp2.lastCheck = time.Now()
		}
		m.mu.Unlock()
	}
	return results
}

// HealthCheckResult holds the result of a single plugin health check.
type HealthCheckResult struct {
	Name    string       `json:"name"`
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
}

// ShutdownAll shuts down all registered plugins.
func (m *Manager) ShutdownAll(ctx context.Context) {
	m.mu.RLock()
	plugins := make([]Plugin, 0, len(m.plugins))
	for _, mp := range m.plugins {
		plugins = append(plugins, mp.plugin)
	}
	m.mu.RUnlock()

	for _, p := range plugins {
		if err := p.Shutdown(ctx); err != nil {
			m.logger.Warn("plugin shutdown error", "error", err)
		}
	}
}

// RegisterServiceAdapter wraps a service.Service and registers it as an in-process plugin.
// This is the primary migration path for existing AWS services.
func (m *Manager) RegisterServiceAdapter(ctx context.Context, adapter *ServiceAdapter) error {
	return m.RegisterInProcess(ctx, adapter)
}

// Names returns the names of all registered plugins.
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}
