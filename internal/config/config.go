package config

import (
	"encoding/json"
	"os"
	"sync"
)

// ShareMode defines how to share registry with Claude
type ShareMode string

const (
	// ShareSync directly modifies Claude's settings
	ShareSync ShareMode = "sync"
	// ShareMerge merges codex-market + claude marketplace when listing
	ShareMerge ShareMode = "merge"
	// ShareIgnore uses only codex-market's own settings
	ShareIgnore ShareMode = "ignore"
)

// AutoUpdateMode defines the auto-update behavior
type AutoUpdateMode string

const (
	// AutoUpdateModeNotify shows update notification and asks user
	AutoUpdateModeNotify AutoUpdateMode = "notify"
	// AutoUpdateModeAuto automatically applies updates without asking
	AutoUpdateModeAuto AutoUpdateMode = "auto"
	// AutoUpdateModeDisabled disables auto-update check
	AutoUpdateModeDisabled AutoUpdateMode = "disabled"
)

// AutoUpdateConfig contains auto-update settings
type AutoUpdateConfig struct {
	Enabled              bool           `json:"enabled"`              // Enable auto-update feature (default: true)
	Mode                 AutoUpdateMode `json:"mode"`                 // "notify", "auto", "disabled" (default: notify)
	RequestOverrideCodex bool           `json:"requestOverrideCodex"` // Whether alias setup was already offered
}

// Config represents the main configuration file structure
type Config struct {
	Locale       string                 `json:"locale"`     // "auto" or ISO format (e.g., "ko-KR", "en-US")
	AutoUpdate   AutoUpdateConfig       `json:"autoUpdate"` // Auto-update settings
	Claude       ClaudeConfig           `json:"claude"`
	Marketplaces map[string]Marketplace `json:"marketplaces"`
}

// ClaudeConfig contains Claude-related settings
type ClaudeConfig struct {
	Registry RegistryConfig `json:"registry"`
}

// RegistryConfig contains registry sharing settings
type RegistryConfig struct {
	Share ShareMode `json:"share"`
}

// Marketplace represents a registered marketplace
type Marketplace struct {
	Source          MarketplaceSource `json:"source"`
	InstallLocation string            `json:"installLocation"`
	LastUpdated     string            `json:"lastUpdated"`
}

// MarketplaceSource describes the source of a marketplace
type MarketplaceSource struct {
	Source string `json:"source"` // "git", "directory"
	URL    string `json:"url,omitempty"`
	Path   string `json:"path,omitempty"`
}

var (
	cfg     *Config
	cfgOnce sync.Once
	cfgMu   sync.RWMutex
)

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		Locale: "auto", // default: auto-detect system locale
		AutoUpdate: AutoUpdateConfig{
			Enabled:              true,               // default: enabled
			Mode:                 AutoUpdateModeNotify, // default: notify user
			RequestOverrideCodex: false,              // default: not yet offered
		},
		Claude: ClaudeConfig{
			Registry: RegistryConfig{
				Share: ShareIgnore, // default: ignore Claude's registry
			},
		},
		Marketplaces: make(map[string]Marketplace),
	}
}

// Load loads the configuration from file
func Load() (*Config, error) {
	cfgMu.RLock()
	defer cfgMu.RUnlock()

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return NewConfig(), nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Ensure maps are initialized
	if config.Marketplaces == nil {
		config.Marketplaces = make(map[string]Marketplace)
	}

	// Set default share mode if empty
	if config.Claude.Registry.Share == "" {
		config.Claude.Registry.Share = ShareIgnore
	}

	// Set default locale if empty
	if config.Locale == "" {
		config.Locale = "auto"
	}

	// Set default auto-update mode if empty
	if config.AutoUpdate.Mode == "" {
		config.AutoUpdate.Mode = AutoUpdateModeNotify
	}

	return &config, nil
}

// Save saves the configuration to file
func Save(config *Config) error {
	cfgMu.Lock()
	defer cfgMu.Unlock()

	if err := EnsureDir(CodexMarketDir()); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0644)
}

// Get returns the current configuration (singleton)
func Get() *Config {
	cfgOnce.Do(func() {
		var err error
		cfg, err = Load()
		if err != nil {
			cfg = NewConfig()
		}
	})
	return cfg
}

// Reload reloads the configuration from file
func Reload() error {
	cfgMu.Lock()
	defer cfgMu.Unlock()

	newCfg, err := Load()
	if err != nil {
		return err
	}
	cfg = newCfg
	return nil
}

// GetShareMode returns the current share mode
func GetShareMode() ShareMode {
	return Get().Claude.Registry.Share
}

// SetShareMode sets the share mode and saves
func SetShareMode(mode ShareMode) error {
	config := Get()
	config.Claude.Registry.Share = mode
	return Save(config)
}

// GetLocale returns the configured locale
func GetLocale() string {
	return Get().Locale
}

// SetLocale sets the locale and saves
func SetLocale(locale string) error {
	config := Get()
	config.Locale = locale
	return Save(config)
}
