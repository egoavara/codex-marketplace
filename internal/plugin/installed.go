package plugin

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/egoavara/codex-market/internal/config"
)

var (
	installed     *InstalledManager
	installedOnce sync.Once
)

// InstalledManager manages installed plugins
type InstalledManager struct {
	mu   sync.RWMutex
	path string
}

// GetInstalled returns the singleton installed manager instance
func GetInstalled() *InstalledManager {
	installedOnce.Do(func() {
		installed = &InstalledManager{
			path: config.InstalledPath(),
		}
	})
	return installed
}

// Load loads installed plugins from the JSON file
func (m *InstalledManager) Load() (*InstalledPlugins, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewInstalledPlugins(), nil
		}
		return nil, err
	}

	var plugins InstalledPlugins
	if err := json.Unmarshal(data, &plugins); err != nil {
		return nil, err
	}

	if plugins.Plugins == nil {
		plugins.Plugins = make(map[string][]InstalledPluginEntry)
	}

	return &plugins, nil
}

// Save saves installed plugins to the JSON file
func (m *InstalledManager) Save(plugins *InstalledPlugins) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := config.EnsureDir(config.CodexMarketDir()); err != nil {
		return err
	}

	data, err := json.MarshalIndent(plugins, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.path, data, 0644)
}

// Add adds a new installed plugin entry
func (m *InstalledManager) Add(pluginID string, entry InstalledPluginEntry) error {
	plugins, err := m.Load()
	if err != nil {
		return err
	}

	// Check if already exists with same scope
	entries := plugins.Plugins[pluginID]
	for i, e := range entries {
		if e.Scope == entry.Scope && e.ProjectPath == entry.ProjectPath {
			// Update existing entry
			entries[i] = entry
			plugins.Plugins[pluginID] = entries
			return m.Save(plugins)
		}
	}

	// Add new entry
	plugins.Plugins[pluginID] = append(plugins.Plugins[pluginID], entry)
	return m.Save(plugins)
}

// Remove removes an installed plugin
func (m *InstalledManager) Remove(pluginID string) error {
	plugins, err := m.Load()
	if err != nil {
		return err
	}

	delete(plugins.Plugins, pluginID)
	return m.Save(plugins)
}

// Get returns entries for a specific plugin
func (m *InstalledManager) Get(pluginID string) ([]InstalledPluginEntry, error) {
	plugins, err := m.Load()
	if err != nil {
		return nil, err
	}

	return plugins.Plugins[pluginID], nil
}

// List returns all installed plugins
func (m *InstalledManager) List() (*InstalledPlugins, error) {
	return m.Load()
}

// Exists checks if a plugin is installed
func (m *InstalledManager) Exists(pluginID string) (bool, error) {
	entries, err := m.Get(pluginID)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}
