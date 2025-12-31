package marketplace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoavara/codex-market/internal/config"
)

var (
	registry     *Registry
	registryOnce sync.Once
)

// Registry manages known marketplaces
type Registry struct {
	mu sync.RWMutex
}

// GetRegistry returns the singleton registry instance
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = &Registry{}
	})
	return registry
}

// List returns all known marketplaces based on share mode
func (r *Registry) List() (KnownMarketplaces, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg := config.Get()
	result := make(KnownMarketplaces)

	// Add codex-market's own marketplaces
	for name, mp := range cfg.Marketplaces {
		result[name] = KnownMarketplace{
			Source:          MarketplaceSource(mp.Source),
			InstallLocation: mp.InstallLocation,
			LastUpdated:     mp.LastUpdated,
		}
	}

	// Handle share mode
	switch cfg.Claude.Registry.Share {
	case config.ShareMerge:
		// Merge with Claude's marketplaces
		claudeMarketplaces, err := loadClaudeMarketplaces()
		if err == nil {
			for name, mp := range claudeMarketplaces {
				if _, exists := result[name]; !exists {
					result[name] = mp
				}
			}
		}
	case config.ShareSync, config.ShareIgnore:
		// sync: we manage Claude's settings directly when adding/removing
		// ignore: only use our own marketplaces
	}

	return result, nil
}

// Get returns a single marketplace by name
func (r *Registry) Get(name string) (*KnownMarketplace, error) {
	marketplaces, err := r.List()
	if err != nil {
		return nil, err
	}

	mp, ok := marketplaces[name]
	if !ok {
		return nil, nil
	}

	return &mp, nil
}

// Add adds a new marketplace to the registry
func (r *Registry) Add(name string, url string, installLocation string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg := config.Get()

	mp := config.Marketplace{
		Source: config.MarketplaceSource{
			Source: "git",
			URL:    url,
		},
		InstallLocation: installLocation,
		LastUpdated:     time.Now().Format(time.RFC3339),
	}

	cfg.Marketplaces[name] = mp

	if err := config.Save(cfg); err != nil {
		return err
	}

	// If sync mode, also update Claude's settings
	if cfg.Claude.Registry.Share == config.ShareSync {
		return syncToClaudeSettings(name, url)
	}

	return nil
}

// Remove removes a marketplace from the registry
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg := config.Get()

	delete(cfg.Marketplaces, name)

	if err := config.Save(cfg); err != nil {
		return err
	}

	// If sync mode, also remove from Claude's settings
	if cfg.Claude.Registry.Share == config.ShareSync {
		return removeFromClaudeSettings(name)
	}

	return nil
}

// UpdateTimestamp updates the last updated timestamp for a marketplace
func (r *Registry) UpdateTimestamp(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfg := config.Get()

	if mp, ok := cfg.Marketplaces[name]; ok {
		mp.LastUpdated = time.Now().Format(time.RFC3339)
		cfg.Marketplaces[name] = mp
		return config.Save(cfg)
	}

	return nil
}

// Exists checks if a marketplace exists
func (r *Registry) Exists(name string) (bool, error) {
	mp, err := r.Get(name)
	if err != nil {
		return false, err
	}
	return mp != nil, nil
}

// loadClaudeMarketplaces loads marketplaces from Claude's known_marketplaces.json
func loadClaudeMarketplaces() (KnownMarketplaces, error) {
	claudePath := filepath.Join(config.ClaudeDir(), "plugins", "known_marketplaces.json")

	data, err := os.ReadFile(claudePath)
	if err != nil {
		return nil, err
	}

	var marketplaces KnownMarketplaces
	if err := json.Unmarshal(data, &marketplaces); err != nil {
		return nil, err
	}

	return marketplaces, nil
}

// syncToClaudeSettings adds a marketplace to Claude's settings.json
func syncToClaudeSettings(name, url string) error {
	settingsPath := config.GlobalSettingsPath()

	// Convert git SSH URL to HTTPS URL for Claude compatibility
	httpsURL := convertToHTTPS(url)

	// Load existing settings
	data, err := os.ReadFile(settingsPath)
	var settings map[string]interface{}

	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		settings = make(map[string]interface{})
	} else {
		if err := json.Unmarshal(data, &settings); err != nil {
			return err
		}
	}

	// Ensure extraKnownMarketplaces exists
	extra, ok := settings["extraKnownMarketplaces"].(map[string]interface{})
	if !ok {
		extra = make(map[string]interface{})
	}

	// Add marketplace
	extra[name] = map[string]interface{}{
		"source": map[string]interface{}{
			"source": "url",
			"url":    httpsURL,
		},
	}

	settings["extraKnownMarketplaces"] = extra

	// Save settings
	if err := config.EnsureDir(filepath.Dir(settingsPath)); err != nil {
		return err
	}

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, newData, 0644)
}

// convertToHTTPS converts git SSH URL to HTTPS URL
// e.g., "git@github.daumkakao.com:Infra-sysdev/skills.git" -> "https://github.daumkakao.com/Infra-sysdev/skills.git"
func convertToHTTPS(url string) string {
	// Already HTTPS
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		return url
	}

	// Convert git@host:path format to https://host/path
	if strings.HasPrefix(url, "git@") {
		// Remove "git@" prefix
		url = strings.TrimPrefix(url, "git@")
		// Replace first ":" with "/"
		url = strings.Replace(url, ":", "/", 1)
		return "https://" + url
	}

	// If no protocol, assume https
	return "https://" + url
}

// removeFromClaudeSettings removes a marketplace from Claude's settings.json
func removeFromClaudeSettings(name string) error {
	settingsPath := config.GlobalSettingsPath()

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	// Remove from extraKnownMarketplaces
	if extra, ok := settings["extraKnownMarketplaces"].(map[string]interface{}); ok {
		delete(extra, name)
		settings["extraKnownMarketplaces"] = extra
	}

	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, newData, 0644)
}
