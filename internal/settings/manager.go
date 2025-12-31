package settings

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/egoavara/codex-market/internal/config"
)

// Load loads settings from a file path
func Load(path string) (*ClaudeSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewClaudeSettings(), nil
		}
		return nil, err
	}

	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	if settings.EnabledPlugins == nil {
		settings.EnabledPlugins = make(map[string]bool)
	}
	if settings.ExtraKnownMarketplaces == nil {
		settings.ExtraKnownMarketplaces = make(map[string]ExtraMarketplace)
	}

	return &settings, nil
}

// Save saves settings to a file path
func Save(path string, settings *ClaudeSettings) error {
	if err := config.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// EnablePlugin enables a plugin in the settings
func EnablePlugin(path, pluginID string) error {
	settings, err := Load(path)
	if err != nil {
		return err
	}

	settings.EnabledPlugins[pluginID] = true
	return Save(path, settings)
}

// DisablePlugin disables a plugin in the settings
func DisablePlugin(path, pluginID string) error {
	settings, err := Load(path)
	if err != nil {
		return err
	}

	delete(settings.EnabledPlugins, pluginID)
	return Save(path, settings)
}

// AddMarketplace adds an extra marketplace to the settings
func AddMarketplace(path, name, sourceURL string) error {
	settings, err := Load(path)
	if err != nil {
		return err
	}

	settings.ExtraKnownMarketplaces[name] = ExtraMarketplace{
		Source: MarketplaceSourceRef{
			Source: "url",
			URL:    sourceURL,
		},
	}

	return Save(path, settings)
}

// RemoveMarketplace removes an extra marketplace from the settings
func RemoveMarketplace(path, name string) error {
	settings, err := Load(path)
	if err != nil {
		return err
	}

	delete(settings.ExtraKnownMarketplaces, name)
	return Save(path, settings)
}
