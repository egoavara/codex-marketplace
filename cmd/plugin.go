package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoavara/codex-market/internal/config"
	"github.com/egoavara/codex-market/internal/git"
	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/marketplace"
	"github.com/egoavara/codex-market/internal/plugin"
	"github.com/egoavara/codex-market/internal/search"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long: `Manage plugins from registered marketplaces.

Commands:
  install    Install a plugin
  uninstall  Uninstall an installed plugin
  update     Update installed plugin(s)
  list       List installed plugins
  search     Search for plugins`,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install <plugin>@<marketplace>",
	Short: "Install a plugin from a marketplace",
	Long: `Install a plugin from a registered marketplace.

Example:
  codex-market plugin install my-plugin@my-marketplace
  codex-market plugin install my-plugin@my-marketplace -s project`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInstall,
}

var pluginUninstallCmd = &cobra.Command{
	Use:     "uninstall <plugin>@<marketplace>",
	Aliases: []string{"remove", "rm"},
	Short:   "Uninstall an installed plugin",
	Long: `Uninstall an installed plugin.

Scope options:
  -s global   Remove from global installation only (default)
  -s project  Remove from current project only
  -s all      Remove from all installations

Example:
  codex-market plugin uninstall my-plugin@my-marketplace
  codex-market plugin uninstall my-plugin@my-marketplace -s all`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginUninstall,
}

var pluginUsageCmd = &cobra.Command{
	Use:   "usage <plugin>@<marketplace>",
	Short: "Show where a plugin is installed",
	Long: `Show all installation locations for a plugin.

Example:
  codex-market plugin usage my-plugin@my-marketplace`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginUsage,
}

var pluginUpdateCmd = &cobra.Command{
	Use:   "update [plugin@marketplace]",
	Short: "Update installed plugin(s)",
	Long: `Update all installed plugins or a specific plugin.

Example:
  codex-market plugin update                     # Update all plugins
  codex-market plugin update my-plugin@my-marketplace  # Update specific`,
	RunE: runPluginUpdate,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long: `List all installed plugins.

Example:
  codex-market plugin list`,
	RunE: runPluginList,
}

var pluginSearchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search for plugins across all marketplaces",
	Long: `Search for plugins using fuzzy matching across all registered marketplaces.

The search looks through plugin names, descriptions, tags, and keywords.

Example:
  codex-market plugin search formatter
  codex-market plugin search code-review`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginSearch,
}

var (
	pluginInstallScope   string
	pluginUninstallScope string
)

func init() {
	pluginInstallCmd.Flags().StringVarP(&pluginInstallScope, "scope", "s", "global", "install scope (global or project)")
	pluginUninstallCmd.Flags().StringVarP(&pluginUninstallScope, "scope", "s", "global", "uninstall scope (global, project, or all)")

	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginUninstallCmd)
	pluginCmd.AddCommand(pluginUpdateCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginSearchCmd)
	pluginCmd.AddCommand(pluginUsageCmd)
}

func runPluginInstall(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	// Parse plugin identifier
	pluginName, marketplaceName, err := parsePluginID(identifier)
	if err != nil {
		return err
	}

	// Get marketplace
	registry := marketplace.GetRegistry()
	mp, err := registry.Get(marketplaceName)
	if err != nil {
		return err
	}
	if mp == nil {
		return fmt.Errorf(i18n.T("MarketplaceNotFound", map[string]any{"Name": marketplaceName}))
	}

	// Load marketplace manifest
	manifest, err := marketplace.LoadManifest(mp.InstallLocation)
	if err != nil {
		return err
	}

	// Find plugin
	pluginEntry := manifest.FindPlugin(pluginName)
	if pluginEntry == nil {
		return fmt.Errorf(i18n.T("PluginNotFound", map[string]any{
			"Plugin":      pluginName,
			"Marketplace": marketplaceName,
		}))
	}

	// Get source path
	sourcePath := manifest.GetPluginSourcePath(mp.InstallLocation, pluginEntry)

	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("plugin source not found: %s", sourcePath)
	}

	// Determine version
	version := pluginEntry.Version
	if version == "" {
		gitClient := git.NewClient()
		commit, err := gitClient.GetCurrentCommit(mp.InstallLocation)
		if err == nil && len(commit) > 12 {
			version = commit[:12]
		} else {
			version = "latest"
		}
	}

	pluginID := fmt.Sprintf("%s@%s", pluginName, marketplaceName)
	fmt.Printf("Installing %s...\n", pluginID)

	// Determine Codex skills directory based on scope
	var codexSkillsDir string
	if pluginInstallScope == "project" {
		codexSkillsDir = config.ProjectCodexSkillsDir()
		if codexSkillsDir == "" {
			cwd, _ := os.Getwd()
			codexSkillsDir = filepath.Join(cwd, ".codex", "skills")
		}
	} else {
		codexSkillsDir = config.CodexSkillsDir()
	}

	// Find and copy skills from the plugin's skills folder
	skillsSourceDir := filepath.Join(sourcePath, "skills")
	if _, err := os.Stat(skillsSourceDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin has no skills folder: %s", skillsSourceDir)
	}

	// Read skills directories
	skillEntries, err := os.ReadDir(skillsSourceDir)
	if err != nil {
		return fmt.Errorf("failed to read skills folder: %w", err)
	}

	var installedSkills []plugin.SkillEntry
	for _, entry := range skillEntries {
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()
		skillSourcePath := filepath.Join(skillsSourceDir, skillName)

		// Check if SKILL.md exists
		skillMdPath := filepath.Join(skillSourcePath, "SKILL.md")
		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			continue // Skip directories without SKILL.md
		}

		// Copy skill to Codex skills directory
		skillDestPath := filepath.Join(codexSkillsDir, skillName)
		if err := config.EnsureDir(skillDestPath); err != nil {
			return fmt.Errorf("failed to create skill directory: %w", err)
		}

		if err := plugin.CopyDir(skillSourcePath, skillDestPath); err != nil {
			os.RemoveAll(skillDestPath)
			return fmt.Errorf("failed to copy skill files: %w", err)
		}

		installedSkills = append(installedSkills, plugin.SkillEntry{
			Name: skillName,
			Path: skillDestPath,
		})
	}

	if len(installedSkills) == 0 {
		return fmt.Errorf("no valid skills found in plugin (SKILL.md required)")
	}

	// Also keep a cache copy for tracking
	cachePath := filepath.Join(config.PluginCacheDir(), marketplaceName, pluginName, version)
	if err := config.EnsureDir(cachePath); err != nil {
		return err
	}
	if err := plugin.CopyDir(sourcePath, cachePath); err != nil {
		os.RemoveAll(cachePath)
		return fmt.Errorf("failed to cache plugin files: %w", err)
	}

	// Add to installed plugins
	now := time.Now().Format(time.RFC3339)
	entry := plugin.InstalledPluginEntry{
		Scope:       pluginInstallScope,
		Version:     version,
		InstalledAt: now,
		LastUpdated: now,
		Source: plugin.PluginSource{
			Marketplace: marketplaceName,
			URL:         mp.Source.URL,
			CachePath:   cachePath,
		},
		Skills: installedSkills,
	}

	if pluginInstallScope == "project" {
		cwd, _ := os.Getwd()
		entry.ProjectPath = cwd
	}

	if err := plugin.GetInstalled().Add(pluginID, entry); err != nil {
		return err
	}

	// Success message
	fmt.Println(i18n.T("InstallSuccess", map[string]any{
		"Plugin":      pluginName,
		"Marketplace": marketplaceName,
		"Version":     version,
	}))
	skillNames := make([]string, len(installedSkills))
	for i, s := range installedSkills {
		skillNames[i] = s.Name
	}
	fmt.Printf("  Skills: %s\n", strings.Join(skillNames, ", "))
	fmt.Printf("  Location: %s\n", codexSkillsDir)

	return nil
}

func runPluginUninstall(cmd *cobra.Command, args []string) error {
	pluginID := args[0]

	// Validate scope
	scope := pluginUninstallScope
	if scope != "global" && scope != "project" && scope != "all" {
		return fmt.Errorf("invalid scope: %s (must be global, project, or all)", scope)
	}

	// Get current project path for project scope
	cwd, _ := os.Getwd()

	// Check if installed with the given scope
	installed := plugin.GetInstalled()
	entries, err := installed.GetByScope(pluginID, scope, cwd)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		if scope == "all" {
			return fmt.Errorf(i18n.T("NotInstalled", map[string]any{"Plugin": pluginID}))
		}
		return fmt.Errorf("plugin %s is not installed with scope '%s'", pluginID, scope)
	}

	// Remove by scope
	removed, err := installed.RemoveByScope(pluginID, scope, cwd)
	if err != nil {
		return err
	}

	// Remove skill directories and cache for removed entries
	for _, entry := range removed {
		scopeInfo := entry.Scope
		if entry.Scope == "project" {
			scopeInfo = fmt.Sprintf("project:%s", entry.ProjectPath)
		}
		fmt.Printf("Removing from %s...\n", scopeInfo)

		// Remove each skill folder
		for _, skill := range entry.Skills {
			if err := os.RemoveAll(skill.Path); err != nil {
				fmt.Printf("  Warning: failed to remove skill %s at %s: %v\n", skill.Name, skill.Path, err)
			} else {
				fmt.Printf("  Removed skill: %s (%s)\n", skill.Name, skill.Path)
			}
		}

		// Remove cache directory
		if entry.Source.CachePath != "" {
			if err := os.RemoveAll(entry.Source.CachePath); err != nil {
				fmt.Printf("  Warning: failed to remove cache %s: %v\n", entry.Source.CachePath, err)
			}
		}
	}

	// Success message
	fmt.Printf("\n%s\n", i18n.T("RemoveSuccess", map[string]any{"Plugin": pluginID}))
	fmt.Printf("Removed %d installation(s)\n", len(removed))

	return nil
}

func runPluginUsage(cmd *cobra.Command, args []string) error {
	pluginID := args[0]

	installed := plugin.GetInstalled()
	entries, err := installed.Get(pluginID)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return fmt.Errorf(i18n.T("NotInstalled", map[string]any{"Plugin": pluginID}))
	}

	fmt.Printf("Plugin: %s\n", pluginID)
	fmt.Println(strings.Repeat("-", 40))

	for i, entry := range entries {
		fmt.Printf("\n[%d] Scope: %s\n", i+1, entry.Scope)
		if entry.Scope == "project" {
			fmt.Printf("    Project: %s\n", entry.ProjectPath)
		}
		fmt.Printf("    Version: %s\n", entry.Version)
		fmt.Printf("    Source: %s\n", entry.Source.URL)
		fmt.Printf("    Installed: %s\n", entry.InstalledAt)
		fmt.Printf("    Skills:\n")
		for _, skill := range entry.Skills {
			fmt.Printf("      - %s: %s\n", skill.Name, skill.Path)
		}
	}

	fmt.Printf("\nTotal: %d installation(s)\n", len(entries))
	return nil
}

func runPluginUpdate(cmd *cobra.Command, args []string) error {
	installed := plugin.GetInstalled()
	installedPlugins, err := installed.List()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		// Update all installed plugins
		if len(installedPlugins.Plugins) == 0 {
			fmt.Println(i18n.T("NoPluginsInstalled", nil))
			return nil
		}

		gitClient := git.NewClient()
		registry := marketplace.GetRegistry()

		// First, update all marketplaces
		fmt.Println("Updating marketplaces...")
		if err := updateAllMarketplaces(gitClient, registry); err != nil {
			return err
		}

		// Then reinstall all plugins
		fmt.Println("\nReinstalling plugins...")
		for pluginID := range installedPlugins.Plugins {
			fmt.Printf("Reinstalling %s...\n", pluginID)
			if err := runPluginInstall(nil, []string{pluginID}); err != nil {
				fmt.Printf("  Error: %v\n", err)
				continue
			}
		}

		fmt.Println(i18n.T("UpdateAllSuccess", nil))
		return nil
	}

	// Update specific plugin
	pluginID := args[0]
	pluginName, marketplaceName, err := parsePluginID(pluginID)
	if err != nil {
		return err
	}

	gitClient := git.NewClient()
	registry := marketplace.GetRegistry()

	// First update the marketplace
	if err := updateMarketplace(gitClient, registry, marketplaceName); err != nil {
		return err
	}

	// Then reinstall the plugin
	fmt.Printf("Reinstalling %s@%s...\n", pluginName, marketplaceName)
	return runPluginInstall(nil, []string{pluginID})
}

func runPluginList(cmd *cobra.Command, args []string) error {
	installed, err := plugin.GetInstalled().List()
	if err != nil {
		return err
	}

	fmt.Println(i18n.T("ListPluginsHeader", nil))
	fmt.Println(strings.Repeat("-", 40))

	if len(installed.Plugins) == 0 {
		fmt.Println(i18n.T("NoPluginsInstalled", nil))
		return nil
	}

	for id, entries := range installed.Plugins {
		for _, entry := range entries {
			fmt.Printf("  %s (v%s)\n", id, entry.Version)
			fmt.Printf("    Scope: %s\n", entry.Scope)
			fmt.Printf("    Source: %s\n", entry.Source.URL)
			fmt.Printf("    Skills:\n")
			for _, skill := range entry.Skills {
				fmt.Printf("      - %s: %s\n", skill.Name, skill.Path)
			}
			fmt.Printf("    Installed: %s\n", entry.InstalledAt)
			fmt.Println()
		}
	}

	return nil
}

func runPluginSearch(cmd *cobra.Command, args []string) error {
	keyword := args[0]

	registry := marketplace.GetRegistry()
	knownMarketplaces, err := registry.List()
	if err != nil {
		return err
	}

	if len(knownMarketplaces) == 0 {
		fmt.Println(i18n.T("NoMarketplaces", nil))
		return nil
	}

	// Load all marketplace manifests
	manifests := make(map[string]*marketplace.MarketplaceManifest)
	for name, mp := range knownMarketplaces {
		manifest, err := marketplace.LoadManifest(mp.InstallLocation)
		if err != nil {
			continue
		}
		manifests[name] = manifest
	}

	// Perform fuzzy search
	results := search.FuzzySearch(manifests, keyword)

	if len(results) == 0 {
		fmt.Println(i18n.T("NoResults", map[string]any{"Keyword": keyword}))
		return nil
	}

	// Print results
	fmt.Println(i18n.T("SearchResults", map[string]any{"Count": len(results)}, len(results)))
	fmt.Println()

	for _, r := range results {
		version := r.Plugin.Version
		if version == "" {
			version = "latest"
		}

		fmt.Printf("  %s@%s (v%s)\n", r.Plugin.Name, r.Marketplace, version)

		if r.Plugin.Description != "" {
			fmt.Printf("    %s\n", r.Plugin.Description)
		}

		if len(r.Plugin.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(r.Plugin.Tags, ", "))
		}

		if r.Plugin.Category != "" {
			fmt.Printf("    Category: %s\n", r.Plugin.Category)
		}

		fmt.Println()
	}

	return nil
}

// parsePluginID parses "plugin@marketplace" format
func parsePluginID(identifier string) (string, string, error) {
	parts := strings.Split(identifier, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf(i18n.T("InvalidPluginIdentifier", map[string]any{
			"Identifier": identifier,
		}))
	}
	return parts[0], parts[1], nil
}
