package marketplace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// ManifestDir is the directory containing marketplace.json
	ManifestDir = ".claude-plugin"
	// ManifestFile is the marketplace manifest filename
	ManifestFile = "marketplace.json"
)

// LoadManifest loads a marketplace manifest from the given directory
func LoadManifest(marketplacePath string) (*MarketplaceManifest, error) {
	manifestPath := filepath.Join(marketplacePath, ManifestDir, ManifestFile)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("manifest not found: %s", manifestPath)
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest MarketplaceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// FindPlugin finds a plugin by name in the manifest
func (m *MarketplaceManifest) FindPlugin(name string) *PluginEntry {
	for i := range m.Plugins {
		if m.Plugins[i].Name == name {
			return &m.Plugins[i]
		}
	}
	return nil
}

// GetPluginSourcePath returns the full path to the plugin source
func (m *MarketplaceManifest) GetPluginSourcePath(marketplacePath string, plugin *PluginEntry) string {
	basePath := marketplacePath

	// Apply pluginRoot if specified
	if m.Metadata != nil && m.Metadata.PluginRoot != "" {
		basePath = filepath.Join(marketplacePath, m.Metadata.PluginRoot)
	}

	return filepath.Join(basePath, plugin.Source)
}
