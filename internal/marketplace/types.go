package marketplace

// MarketplaceManifest represents the .claude-plugin/marketplace.json structure
type MarketplaceManifest struct {
	Name     string               `json:"name"`
	Owner    Owner                `json:"owner"`
	Metadata *MarketplaceMetadata `json:"metadata,omitempty"`
	Plugins  []PluginEntry        `json:"plugins"`
}

// Owner represents the marketplace owner information
type Owner struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// MarketplaceMetadata contains optional metadata for the marketplace
type MarketplaceMetadata struct {
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	PluginRoot  string `json:"pluginRoot,omitempty"`
}

// PluginEntry represents a plugin entry in the marketplace
type PluginEntry struct {
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	Author      *Owner   `json:"author,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	License     string   `json:"license,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Strict      bool     `json:"strict,omitempty"`
}

// KnownMarketplace represents an entry in known_marketplaces.json
type KnownMarketplace struct {
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

// KnownMarketplaces is a map of marketplace name to KnownMarketplace
type KnownMarketplaces map[string]KnownMarketplace
