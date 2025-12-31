package plugin

// PluginManifest represents the .claude-plugin/plugin.json structure
type PluginManifest struct {
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	Author      *Author  `json:"author,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	License     string   `json:"license,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Commands    any      `json:"commands,omitempty"`   // string or []string
	Agents      string   `json:"agents,omitempty"`
	Skills      string   `json:"skills,omitempty"`
	Hooks       string   `json:"hooks,omitempty"`
	MCPServers  string   `json:"mcpServers,omitempty"`
	LSPServers  string   `json:"lspServers,omitempty"`
}

// Author represents the plugin author information
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// InstalledPlugins represents the installed_plugins.json structure
type InstalledPlugins struct {
	Version int                               `json:"version"`
	Plugins map[string][]InstalledPluginEntry `json:"plugins"`
}

// InstalledPluginEntry represents a single installed plugin entry
type InstalledPluginEntry struct {
	Scope       string       `json:"scope"`                 // "global" or "project"
	ProjectPath string       `json:"projectPath,omitempty"` // only for project scope
	Version     string       `json:"version"`
	InstalledAt string       `json:"installedAt"`
	LastUpdated string       `json:"lastUpdated"`
	Source      PluginSource `json:"source"`                // where it was installed from
	Skills      []SkillEntry `json:"skills"`                // installed skills with paths
}

// PluginSource represents the source of an installed plugin
type PluginSource struct {
	Marketplace string `json:"marketplace"`        // marketplace name
	URL         string `json:"url"`                // git URL
	CachePath   string `json:"cachePath"`          // local cache path for tracking
}

// SkillEntry represents an installed skill with its path
type SkillEntry struct {
	Name string `json:"name"` // skill name
	Path string `json:"path"` // full path to skill folder (for deletion)
}

// NewInstalledPlugins creates a new InstalledPlugins instance
func NewInstalledPlugins() *InstalledPlugins {
	return &InstalledPlugins{
		Version: 1,
		Plugins: make(map[string][]InstalledPluginEntry),
	}
}
