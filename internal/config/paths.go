package config

import (
	"os"
	"path/filepath"
)

var (
	homeDir string
)

func init() {
	var err error
	homeDir, err = os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}
}

// CodexMarketDir returns the codex-market config directory path
// ~/.config/codex-market/
func CodexMarketDir() string {
	return filepath.Join(homeDir, ".config", "codex-market")
}

// ConfigPath returns the config.json file path
// ~/.config/codex-market/config.json
func ConfigPath() string {
	return filepath.Join(CodexMarketDir(), "config.json")
}

// InstalledPath returns the installed.json file path
// ~/.config/codex-market/installed.json
func InstalledPath() string {
	return filepath.Join(CodexMarketDir(), "installed.json")
}

// MarketplacesDir returns the marketplaces directory path
// ~/.config/codex-market/marketplaces/
func MarketplacesDir() string {
	return filepath.Join(CodexMarketDir(), "marketplaces")
}

// PluginCacheDir returns the plugin cache directory path
// ~/.config/codex-market/cache/
func PluginCacheDir() string {
	return filepath.Join(CodexMarketDir(), "cache")
}

// ClaudeDir returns the .claude directory path (for Claude settings)
func ClaudeDir() string {
	return filepath.Join(homeDir, ".claude")
}

// CodexDir returns the .codex directory path
func CodexDir() string {
	return filepath.Join(homeDir, ".codex")
}

// CodexConfigPath returns the Codex config.toml file path
// ~/.codex/config.toml
func CodexConfigPath() string {
	return filepath.Join(CodexDir(), "config.toml")
}

// CodexSkillsDir returns the Codex global skills directory path
// ~/.codex/skills/
func CodexSkillsDir() string {
	return filepath.Join(CodexDir(), "skills")
}

// CodexPromptsDir returns the Codex global prompts directory path
// ~/.codex/prompts/ (Codex's custom commands location)
func CodexPromptsDir() string {
	return filepath.Join(CodexDir(), "prompts")
}

// ProjectCodexSkillsDir returns the project-level Codex skills directory path
// .codex/skills/
func ProjectCodexSkillsDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(cwd, ".codex", "skills")
}

// ProjectCodexPromptsDir returns the project-level Codex prompts directory path
// .codex/prompts/
func ProjectCodexPromptsDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(cwd, ".codex", "prompts")
}

// GlobalSettingsPath returns the global Claude settings.json file path
func GlobalSettingsPath() string {
	return filepath.Join(ClaudeDir(), "settings.json")
}

// ProjectSettingsPath returns the project settings.json file path
// Returns empty string if not in a project with .claude directory
func ProjectSettingsPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	projectSettings := filepath.Join(cwd, ".claude", "settings.json")
	if _, err := os.Stat(filepath.Dir(projectSettings)); os.IsNotExist(err) {
		return ""
	}
	return projectSettings
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
