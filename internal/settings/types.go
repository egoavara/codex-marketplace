package settings

// ClaudeSettings represents the settings.json structure
type ClaudeSettings struct {
	Schema                 string                       `json:"$schema,omitempty"`
	Env                    map[string]string            `json:"env,omitempty"`
	Permissions            *Permissions                 `json:"permissions,omitempty"`
	EnabledPlugins         map[string]bool              `json:"enabledPlugins,omitempty"`
	ExtraKnownMarketplaces map[string]ExtraMarketplace  `json:"extraKnownMarketplaces,omitempty"`
	AlwaysThinkingEnabled  bool                         `json:"alwaysThinkingEnabled,omitempty"`
}

// Permissions represents the permissions section in settings
type Permissions struct {
	Allow                 []string `json:"allow,omitempty"`
	Deny                  []string `json:"deny,omitempty"`
	AdditionalDirectories []string `json:"additionalDirectories,omitempty"`
}

// ExtraMarketplace represents an extra marketplace entry
type ExtraMarketplace struct {
	Source MarketplaceSourceRef `json:"source"`
}

// MarketplaceSourceRef describes the source reference for a marketplace
type MarketplaceSourceRef struct {
	Source string `json:"source"` // "url", "git", "directory"
	URL    string `json:"url,omitempty"`
	Path   string `json:"path,omitempty"`
}

// NewClaudeSettings creates a new ClaudeSettings instance
func NewClaudeSettings() *ClaudeSettings {
	return &ClaudeSettings{
		EnabledPlugins:         make(map[string]bool),
		ExtraKnownMarketplaces: make(map[string]ExtraMarketplace),
	}
}
