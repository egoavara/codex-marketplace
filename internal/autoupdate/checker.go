package autoupdate

import (
	"github.com/egoavara/codex-market/internal/git"
	"github.com/egoavara/codex-market/internal/marketplace"
	"github.com/egoavara/codex-market/internal/plugin"
)

// Note: marketplace.LoadManifest was removed as we now rely on
// marketplace update status to determine plugin updates

// Checker handles update checking logic
type Checker struct {
	gitClient git.Client
}

// NewChecker creates a new update checker
func NewChecker() *Checker {
	return &Checker{
		gitClient: git.NewClient(),
	}
}

// CheckAll checks for updates in all marketplaces and plugins
func CheckAll() (*CheckResult, error) {
	checker := NewChecker()
	return checker.CheckAll()
}

// CheckAll checks for updates in all marketplaces and plugins
func (c *Checker) CheckAll() (*CheckResult, error) {
	result := &CheckResult{
		Marketplaces: []UpdateInfo{},
		Plugins:      []UpdateInfo{},
		Errors:       []error{},
	}

	// Check marketplaces first
	mpUpdates, mpErrors := c.CheckMarketplaces()
	result.Marketplaces = mpUpdates
	result.Errors = append(result.Errors, mpErrors...)

	// Build a set of marketplaces that have updates
	updatedMarketplaces := make(map[string]bool)
	for _, mp := range mpUpdates {
		if mp.HasUpdate {
			updatedMarketplaces[mp.Name] = true
		}
	}

	// Check plugins (pass updated marketplaces info)
	pluginUpdates, pluginErrors := c.CheckPlugins(updatedMarketplaces)
	result.Plugins = pluginUpdates
	result.Errors = append(result.Errors, pluginErrors...)

	// Determine if any updates are available
	result.HasAnyUpdate = result.TotalUpdates() > 0

	return result, nil
}

// CheckMarketplaces checks for updates in all registered marketplaces
func (c *Checker) CheckMarketplaces() ([]UpdateInfo, []error) {
	var updates []UpdateInfo
	var errors []error

	registry := marketplace.GetRegistry()
	marketplaces, err := registry.List()
	if err != nil {
		errors = append(errors, err)
		return updates, errors
	}

	for name, mp := range marketplaces {
		// Only check git-based marketplaces
		if mp.Source.Source != "git" {
			continue
		}

		info := UpdateInfo{
			Type: UpdateTypeMarketplace,
			Name: name,
			Path: mp.InstallLocation,
		}

		// Get current commit
		currentCommit, err := c.gitClient.GetCurrentCommit(mp.InstallLocation)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		info.CurrentVer = shortCommit(currentCommit)

		// Check for updates (this also fetches)
		hasUpdate, err := c.gitClient.HasUpdates(mp.InstallLocation)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if hasUpdate {
			// Get remote commit for display
			remoteCommit, err := c.gitClient.GetRemoteCommit(mp.InstallLocation, "")
			if err == nil {
				info.RemoteVer = shortCommit(remoteCommit)
			}
			info.HasUpdate = true
		}

		updates = append(updates, info)
	}

	return updates, errors
}

// CheckPlugins checks for updates in all installed plugins
// updatedMarketplaces contains marketplaces that have pending updates
func (c *Checker) CheckPlugins(updatedMarketplaces map[string]bool) ([]UpdateInfo, []error) {
	var updates []UpdateInfo
	var errors []error

	installed := plugin.GetInstalled()
	installedPlugins, err := installed.List()
	if err != nil {
		errors = append(errors, err)
		return updates, errors
	}
	plugins := installedPlugins.Plugins

	for pluginID, entries := range plugins {
		for _, entry := range entries {
			info := UpdateInfo{
				Type:       UpdateTypePlugin,
				Name:       pluginID,
				CurrentVer: entry.Version,
				Path:       entry.Source.CachePath,
			}

			// If the marketplace has updates, the plugin also needs update
			if updatedMarketplaces[entry.Source.Marketplace] {
				info.HasUpdate = true
				info.RemoteVer = "(marketplace updated)"
			}

			// Only add to list if there's an update
			if info.HasUpdate {
				updates = append(updates, info)
			}
		}
	}

	return updates, errors
}

// shortCommit returns first 7 characters of a commit hash
func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

// extractPluginName extracts the plugin name from pluginID (plugin@marketplace format)
func extractPluginName(pluginID string) string {
	for i := len(pluginID) - 1; i >= 0; i-- {
		if pluginID[i] == '@' {
			return pluginID[:i]
		}
	}
	return pluginID
}
