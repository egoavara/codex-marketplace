package marketplace

import (
	"encoding/json"
	"fmt"
)

// MarketplaceManifest represents the .claude-plugin/marketplace.json structure
type MarketplaceManifest struct {
	Name     string               `json:"name"`
	Owner    Owner                `json:"owner"`
	Metadata *MarketplaceMetadata `json:"metadata,omitempty"`
	Plugins  []PluginEntry        `json:"plugins"`
}

// PluginSource can be either a string path or an object with source details
// String format: "./plugins/xxx"
// Object format (url): {"source": "url", "url": "https://..."}
// Object format (github): {"source": "github", "repo": "owner/repo"}
type PluginSource struct {
	Path string // local path (when source is a string)
	Type string // "path", "url", or "github"
	URL  string // git URL (when source is an object with type "url")
	Repo string // GitHub repo in "owner/repo" format (when source is "github")
}

// UnmarshalJSON implements custom JSON unmarshaling for PluginSource
func (p *PluginSource) UnmarshalJSON(data []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		p.Path = str
		p.Type = "path"
		return nil
	}

	// Try object format
	var obj struct {
		Source string `json:"source"`
		URL    string `json:"url"`
		Repo   string `json:"repo"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		p.Type = obj.Source
		p.URL = obj.URL
		p.Repo = obj.Repo
		return nil
	}

	return fmt.Errorf("invalid source format: expected string or object")
}

// GetSourceURL returns the effective URL for the plugin source
func (p *PluginSource) GetSourceURL() string {
	switch p.Type {
	case "url":
		return p.URL
	case "github":
		return "https://github.com/" + p.Repo + ".git"
	default:
		return ""
	}
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
	Name        string       `json:"name"`
	Source      PluginSource `json:"source"`
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
