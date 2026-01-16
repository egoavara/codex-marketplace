package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoavara/codex-market/internal/autoupdate"
	"github.com/egoavara/codex-market/internal/config"
	"github.com/egoavara/codex-market/internal/git"
	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/marketplace"
	"github.com/egoavara/codex-market/internal/mcp"
	"github.com/egoavara/codex-market/internal/plugin"
	"github.com/egoavara/codex-market/internal/search"
	"github.com/egoavara/codex-market/internal/tui"
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

By default, only updates plugins with version changes.
Use --force to reinstall all plugins regardless of version.

Example:
  codex-market plugin update                     # Update plugins with changes
  codex-market plugin update --force             # Force reinstall all plugins
  codex-market plugin update my-plugin@my-marketplace  # Update specific`,
	RunE: runPluginUpdate,
}

var pluginUpdateForce bool

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Long: `List all installed plugins.

Example:
  codex-market plugin list`,
	RunE: runPluginList,
}

var pluginSearchCmd = &cobra.Command{
	Use:   "search [keyword]",
	Short: "Search for plugins across all marketplaces",
	Long: `Search for plugins using fuzzy matching across all registered marketplaces.

Without arguments, opens an interactive fuzzy finder (TUI mode).
With a keyword, performs a text-based search.

The search looks through plugin names, descriptions, tags, and keywords.

Example:
  codex-market plugin search              # Interactive TUI mode
  codex-market plugin search formatter    # Text search mode`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPluginSearch,
}

var (
	pluginInstallScope   string
	pluginUninstallScope string
	pluginQuietMode      bool // Suppress output during batch operations
)

func init() {
	pluginInstallCmd.Flags().StringVarP(&pluginInstallScope, "scope", "s", "global", "install scope (global or project)")
	pluginUninstallCmd.Flags().StringVarP(&pluginUninstallScope, "scope", "s", "global", "uninstall scope (global, project, or all)")
	pluginUpdateCmd.Flags().BoolVarP(&pluginUpdateForce, "force", "f", false, "force reinstall regardless of version")

	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginUninstallCmd)
	pluginCmd.AddCommand(pluginUpdateCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginSearchCmd)
	pluginCmd.AddCommand(pluginUsageCmd)
}

func runPluginInstall(cmd *cobra.Command, args []string) error {
	if cmd != nil {
		cmd.SilenceUsage = true
	}
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

	// For remote sources (url, github), clone to temp directory
	var tempCloneDir string
	if pluginEntry.IsRemoteSource() {
		gitClient := git.NewClient()
		remoteURL := pluginEntry.Source.GetSourceURL()

		// Create temp directory for cloning
		tempCloneDir, err = os.MkdirTemp("", "codex-plugin-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tempCloneDir) // Clean up temp directory when done

		if !pluginQuietMode {
			fmt.Printf("Cloning %s...\n", remoteURL)
		}

		if err := gitClient.Clone(remoteURL, tempCloneDir); err != nil {
			return fmt.Errorf("failed to clone plugin repository: %w", err)
		}

		// Use cloned directory as source path
		sourcePath = tempCloneDir
	} else {
		// Check if source exists (only for local path sources)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			return fmt.Errorf("plugin source not found: %s", sourcePath)
		}
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

	// Check if already installed in the same scope
	var projectPath string
	if pluginInstallScope == "project" {
		projectPath, _ = os.Getwd()
	}
	existingEntries, err := plugin.GetInstalled().GetByScope(pluginID, pluginInstallScope, projectPath)
	if err != nil {
		return fmt.Errorf("failed to check installed plugins: %w", err)
	}
	if len(existingEntries) > 0 {
		return fmt.Errorf(i18n.T("AlreadyInstalled", map[string]any{
			"Plugin": pluginID,
			"Scope":  pluginInstallScope,
		}))
	}

	if !pluginQuietMode {
		fmt.Printf("Installing %s...\n", pluginID)
	}

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
	var installedSkills []plugin.SkillEntry

	if _, err := os.Stat(skillsSourceDir); err == nil {
		// Read skills directories
		skillEntries, err := os.ReadDir(skillsSourceDir)
		if err != nil {
			return fmt.Errorf("failed to read skills folder: %w", err)
		}

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
			skillDestPath, actualSkillName, err := plugin.ResolveUniqueSkillPath(codexSkillsDir, skillName)
			if err != nil {
				return fmt.Errorf("failed to resolve skill path: %w", err)
			}

			if actualSkillName != skillName && !pluginQuietMode {
				fmt.Println(i18n.T("SkillNameConflict", map[string]any{
					"Original": skillName,
					"Resolved": actualSkillName,
				}))
			}

			if err := config.EnsureDir(skillDestPath); err != nil {
				return fmt.Errorf("failed to create skill directory: %w", err)
			}

			if err := plugin.CopyDir(skillSourcePath, skillDestPath); err != nil {
				os.RemoveAll(skillDestPath)
				return fmt.Errorf("failed to copy skill files: %w", err)
			}

			installedSkills = append(installedSkills, plugin.SkillEntry{
				Name: actualSkillName,
				Path: skillDestPath,
			})
		}
	}

	// Find and copy commands from the plugin's commands folder
	commandsSourceDir := filepath.Join(sourcePath, "commands")
	var installedCommands []plugin.CommandEntry
	var codexPromptsDir string

	if _, err := os.Stat(commandsSourceDir); err == nil {
		// Determine Codex prompts directory based on scope
		if pluginInstallScope == "project" {
			codexPromptsDir = config.ProjectCodexPromptsDir()
			if codexPromptsDir == "" {
				cwd, _ := os.Getwd()
				codexPromptsDir = filepath.Join(cwd, ".codex", "prompts")
			}
		} else {
			codexPromptsDir = config.CodexPromptsDir()
		}

		// Ensure prompts directory exists
		if err := config.EnsureDir(codexPromptsDir); err != nil {
			return fmt.Errorf("failed to create prompts directory: %w", err)
		}

		// Read command files
		commandEntries, err := os.ReadDir(commandsSourceDir)
		if err != nil {
			return fmt.Errorf("failed to read commands folder: %w", err)
		}

		for _, entry := range commandEntries {
			if entry.IsDir() {
				continue // Skip directories
			}

			fileName := entry.Name()
			if !strings.HasSuffix(fileName, ".md") {
				continue // Skip non-markdown files
			}

			commandSourcePath := filepath.Join(commandsSourceDir, fileName)

			// Resolve unique path (handle conflicts)
			commandDestPath, actualFileName, err := plugin.ResolveUniquePromptPath(codexPromptsDir, fileName)
			if err != nil {
				return fmt.Errorf("failed to resolve prompt path: %w", err)
			}

			if actualFileName != fileName && !pluginQuietMode {
				fmt.Println(i18n.T("PromptNameConflict", map[string]any{
					"Original": fileName,
					"Resolved": actualFileName,
				}))
			}

			// Copy command file
			if err := plugin.CopyFile(commandSourcePath, commandDestPath); err != nil {
				return fmt.Errorf("failed to copy command file %s: %w", fileName, err)
			}

			// Command name without .md extension
			commandName := strings.TrimSuffix(actualFileName, ".md")
			installedCommands = append(installedCommands, plugin.CommandEntry{
				Name: commandName,
				Path: commandDestPath,
			})
		}
	}

	// Find and install MCP servers from .mcp.json
	mcpJsonPath := filepath.Join(sourcePath, ".mcp.json")
	var installedMCPServers []plugin.MCPServerEntry

	if _, err := os.Stat(mcpJsonPath); err == nil {
		mcpData, err := os.ReadFile(mcpJsonPath)
		if err != nil {
			if !pluginQuietMode {
				fmt.Printf("Warning: failed to read .mcp.json: %v\n", err)
			}
		} else {
			servers, err := mcp.ParseMCPJSON(mcpData)
			if err != nil {
				if !pluginQuietMode {
					fmt.Printf("Warning: failed to parse .mcp.json: %v\n", err)
				}
			} else if len(servers) > 0 {
				// Check for conflicts with user-managed servers
				conflicts, err := mcp.CheckServerNameConflicts(config.CodexConfigPath(), servers)
				if err != nil && !pluginQuietMode {
					fmt.Printf("Warning: failed to check MCP server conflicts: %v\n", err)
				}

				for _, conflict := range conflicts {
					if !pluginQuietMode {
						fmt.Println(i18n.T("MCPServerExists", map[string]any{
							"Name":    conflict,
							"Manager": "user",
						}))
					}
					// Remove conflicting server from installation
					delete(servers, conflict)
				}

				if len(servers) > 0 {
					// Add MCP servers to config.toml with markers
					mismatches, err := mcp.AddMCPServers(config.CodexConfigPath(), pluginName, marketplaceName, servers)
					if err != nil {
						if !pluginQuietMode {
							fmt.Printf("Warning: %s: %v\n", i18n.T("MCPConfigError", nil), err)
						}
					} else {
						for name := range servers {
							installedMCPServers = append(installedMCPServers, plugin.MCPServerEntry{
								Name:   name,
								Plugin: fmt.Sprintf("%s@%s", pluginName, marketplaceName),
							})
						}
						// Warn about env var mismatches
						if !pluginQuietMode {
							for _, m := range mismatches {
								fmt.Println(i18n.T("MCPEnvVarMismatch", map[string]any{
									"Key":     m.Key,
									"VarName": m.VarName,
								}))
							}
						}
					}
				}
			}
		}
	}

	// Warn if no skills, commands, or MCP servers found (but continue installation)
	if len(installedSkills) == 0 && len(installedCommands) == 0 && len(installedMCPServers) == 0 && !pluginQuietMode {
		fmt.Println("Warning: no skills, commands, or MCP servers found in plugin")
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
		Skills:     installedSkills,
		Commands:   installedCommands,
		MCPServers: installedMCPServers,
	}

	if pluginInstallScope == "project" {
		cwd, _ := os.Getwd()
		entry.ProjectPath = cwd
	}

	if err := plugin.GetInstalled().Add(pluginID, entry); err != nil {
		return err
	}

	// Success message
	if !pluginQuietMode {
		fmt.Println(i18n.T("InstallSuccess", map[string]any{
			"Plugin":      pluginName,
			"Marketplace": marketplaceName,
			"Version":     version,
		}))

		if len(installedSkills) > 0 {
			skillNames := make([]string, len(installedSkills))
			for i, s := range installedSkills {
				skillNames[i] = s.Name
			}
			fmt.Printf("  Skills: %s\n", strings.Join(skillNames, ", "))
			fmt.Printf("  Skills Location: %s\n", codexSkillsDir)
		}

		if len(installedCommands) > 0 {
			commandNames := make([]string, len(installedCommands))
			for i, c := range installedCommands {
				commandNames[i] = "/" + c.Name
			}
			fmt.Printf("  Commands: %s\n", strings.Join(commandNames, ", "))
			fmt.Printf("  Commands Location: %s\n", codexPromptsDir)
		}

		if len(installedMCPServers) > 0 {
			mcpNames := make([]string, len(installedMCPServers))
			for i, m := range installedMCPServers {
				mcpNames[i] = m.Name
			}
			fmt.Println(i18n.T("MCPServersInstalled", map[string]any{
				"Servers": strings.Join(mcpNames, ", "),
			}))
			fmt.Printf("  MCP Config: %s\n", config.CodexConfigPath())
		}
	}

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
		if !pluginQuietMode {
			scopeInfo := entry.Scope
			if entry.Scope == "project" {
				scopeInfo = fmt.Sprintf("project:%s", entry.ProjectPath)
			}
			fmt.Printf("Removing from %s...\n", scopeInfo)
		}

		// Remove each skill folder
		for _, skill := range entry.Skills {
			if err := os.RemoveAll(skill.Path); err != nil {
				if !pluginQuietMode {
					fmt.Printf("  Warning: failed to remove skill %s at %s: %v\n", skill.Name, skill.Path, err)
				}
			} else if !pluginQuietMode {
				fmt.Printf("  Removed skill: %s (%s)\n", skill.Name, skill.Path)
			}
		}

		// Remove each command file
		for _, command := range entry.Commands {
			if err := os.Remove(command.Path); err != nil {
				if !os.IsNotExist(err) && !pluginQuietMode {
					fmt.Printf("  Warning: failed to remove command %s at %s: %v\n", command.Name, command.Path, err)
				}
			} else if !pluginQuietMode {
				fmt.Printf("  Removed command: /%s (%s)\n", command.Name, command.Path)
			}
		}

		// Remove MCP servers from config.toml (by marker)
		if len(entry.MCPServers) > 0 {
			// Extract plugin name from pluginID (format: pluginName@marketplace)
			pluginName := pluginID
			if idx := strings.Index(pluginID, "@"); idx > 0 {
				pluginName = pluginID[:idx]
			}

			err := mcp.RemoveMCPServers(config.CodexConfigPath(), pluginName)
			if err != nil {
				if !pluginQuietMode {
					fmt.Printf("  Warning: %s: %v\n", i18n.T("MCPConfigError", nil), err)
				}
			} else if !pluginQuietMode {
				mcpNames := make([]string, len(entry.MCPServers))
				for i, m := range entry.MCPServers {
					mcpNames[i] = m.Name
				}
				fmt.Println(i18n.T("MCPServersRemoved", map[string]any{
					"Servers": strings.Join(mcpNames, ", "),
				}))
			}
		}

		// Remove cache directory
		if entry.Source.CachePath != "" {
			if err := os.RemoveAll(entry.Source.CachePath); err != nil {
				if !pluginQuietMode {
					fmt.Printf("  Warning: failed to remove cache %s: %v\n", entry.Source.CachePath, err)
				}
			}
		}
	}

	// Success message
	if !pluginQuietMode {
		fmt.Printf("\n%s\n", i18n.T("RemoveSuccess", map[string]any{"Plugin": pluginID}))
		fmt.Printf("Removed %d installation(s)\n", len(removed))
	}

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
		if len(entry.Skills) > 0 {
			fmt.Printf("    Skills:\n")
			for _, skill := range entry.Skills {
				fmt.Printf("      - %s: %s\n", skill.Name, skill.Path)
			}
		}
		if len(entry.Commands) > 0 {
			fmt.Printf("    Commands:\n")
			for _, command := range entry.Commands {
				fmt.Printf("      - /%s: %s\n", command.Name, command.Path)
			}
		}
		if len(entry.MCPServers) > 0 {
			fmt.Printf("    MCP Servers:\n")
			for _, mcpServer := range entry.MCPServers {
				fmt.Printf("      - %s\n", mcpServer.Name)
			}
			fmt.Printf("    MCP Config: %s\n", config.CodexConfigPath())
		}
	}

	fmt.Printf("\nTotal: %d installation(s)\n", len(entries))
	return nil
}

// pluginUpdateItem holds info about a plugin to update
type pluginUpdateItem struct {
	pluginID   string
	entry      plugin.InstalledPluginEntry
	newVersion string
	isForce    bool
}

func runPluginUpdate(cmd *cobra.Command, args []string) error {
	installed := plugin.GetInstalled()
	installedPlugins, err := installed.List()
	if err != nil {
		return err
	}

	gitClient := git.NewClient()
	registry := marketplace.GetRegistry()

	if len(args) == 0 {
		// Update all installed plugins
		if len(installedPlugins.Plugins) == 0 {
			fmt.Println(i18n.T("NoPluginsInstalled", nil))
			return nil
		}

		// First, update all marketplaces
		fmt.Println("Updating marketplaces...")
		if err := updateAllMarketplaces(gitClient, registry); err != nil {
			return err
		}

		// Phase 1: Collect plugins that need updates
		fmt.Println("\nChecking for plugin updates...")
		var toUpdate []pluginUpdateItem
		var warnings []string

		for pluginID, entries := range installedPlugins.Plugins {
			for _, entry := range entries {
				needsUpdate, newVersion, err := checkPluginNeedsUpdate(pluginID, entry, registry, gitClient)
				if err != nil {
					warnings = append(warnings, fmt.Sprintf("  ⚠ %s: %v", pluginID, err))
					continue
				}

				if !needsUpdate && !pluginUpdateForce {
					continue
				}

				toUpdate = append(toUpdate, pluginUpdateItem{
					pluginID:   pluginID,
					entry:      entry,
					newVersion: newVersion,
					isForce:    pluginUpdateForce,
				})
			}
		}

		// Show warnings
		for _, w := range warnings {
			fmt.Println(w)
		}

		if len(toUpdate) == 0 {
			fmt.Println("\n" + i18n.T("update.noUpdates", nil))
			return nil
		}

		// Phase 2: Show what will be updated
		fmt.Println()
		for _, item := range toUpdate {
			if item.isForce {
				fmt.Printf("  • %s (force reinstall)\n", item.pluginID)
			} else {
				fmt.Printf("  • %s: %s → %s\n", item.pluginID, item.entry.Version, item.newVersion)
			}
		}
		fmt.Println()

		// Phase 3: Apply updates with spinner
		updatedCount := 0
		for _, item := range toUpdate {
			spinner := autoupdate.NewSpinner(item.pluginID)
			spinner.Start()
			err := reinstallPlugin(item.pluginID, item.entry)
			spinner.Stop(err == nil)
			if err == nil {
				updatedCount++
			}
		}

		fmt.Printf("\n%d plugin(s) updated\n", updatedCount)
		return nil
	}

	// Update specific plugin
	pluginID := args[0]
	pluginName, marketplaceName, err := parsePluginID(pluginID)
	if err != nil {
		return err
	}

	// Get installed entry
	entries, err := installed.Get(pluginID)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf(i18n.T("NotInstalled", map[string]any{"Plugin": pluginID}))
	}

	// First update the marketplace
	fmt.Printf("Updating marketplace %s...\n", marketplaceName)
	if err := updateMarketplace(gitClient, registry, marketplaceName); err != nil {
		return err
	}

	// Check if update needed
	for _, entry := range entries {
		needsUpdate, newVersion, err := checkPluginNeedsUpdate(pluginID, entry, registry, gitClient)
		if err != nil {
			return err
		}

		if !needsUpdate && !pluginUpdateForce {
			fmt.Printf("%s is already up to date (v%s)\n", pluginID, entry.Version)
			continue
		}

		fmt.Println()
		if pluginUpdateForce {
			fmt.Printf("  • %s (force reinstall)\n", pluginID)
		} else {
			fmt.Printf("  • %s@%s: %s → %s\n", pluginName, marketplaceName, entry.Version, newVersion)
		}
		fmt.Println()

		spinner := autoupdate.NewSpinner(pluginID)
		spinner.Start()
		err = reinstallPlugin(pluginID, entry)
		spinner.Stop(err == nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// checkPluginNeedsUpdate checks if a plugin has a newer version available
func checkPluginNeedsUpdate(pluginID string, entry plugin.InstalledPluginEntry, registry *marketplace.Registry, gitClient git.Client) (bool, string, error) {
	pluginName, marketplaceName, err := parsePluginID(pluginID)
	if err != nil {
		return false, "", err
	}

	// Get marketplace
	mp, err := registry.Get(marketplaceName)
	if err != nil || mp == nil {
		return false, "", fmt.Errorf("marketplace not found: %s", marketplaceName)
	}

	// Load manifest
	manifest, err := marketplace.LoadManifest(mp.InstallLocation)
	if err != nil {
		return false, "", err
	}

	// Find plugin in manifest
	pluginEntry := manifest.FindPlugin(pluginName)
	if pluginEntry == nil {
		return false, "", fmt.Errorf(i18n.T("PluginNotFound", map[string]any{
			"Plugin":      pluginName,
			"Marketplace": marketplaceName,
		}))
	}

	// Get new version
	newVersion := pluginEntry.Version
	if newVersion == "" {
		commit, err := gitClient.GetCurrentCommit(mp.InstallLocation)
		if err == nil && len(commit) > 12 {
			newVersion = commit[:12]
		} else {
			newVersion = "latest"
		}
	}

	// Compare versions
	return entry.Version != newVersion, newVersion, nil
}

// reinstallPlugin uninstalls and reinstalls a plugin (quiet mode)
func reinstallPlugin(pluginID string, entry plugin.InstalledPluginEntry) error {
	// Save scope info for reinstall
	originalScope := entry.Scope
	originalProjectPath := entry.ProjectPath

	// Enable quiet mode for batch operation
	pluginQuietMode = true
	defer func() { pluginQuietMode = false }()

	// Uninstall
	pluginUninstallScope = entry.Scope
	if err := runPluginUninstall(nil, []string{pluginID}); err != nil {
		return fmt.Errorf("uninstall failed: %w", err)
	}

	// Reinstall with same scope
	pluginInstallScope = originalScope
	if originalScope == "project" {
		// Change to project directory for project scope
		if originalProjectPath != "" {
			oldDir, _ := os.Getwd()
			os.Chdir(originalProjectPath)
			defer os.Chdir(oldDir)
		}
	}

	if err := runPluginInstall(nil, []string{pluginID}); err != nil {
		return fmt.Errorf("reinstall failed: %w", err)
	}

	return nil
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
			if len(entry.Skills) > 0 {
				fmt.Printf("    Skills:\n")
				for _, skill := range entry.Skills {
					fmt.Printf("      - %s: %s\n", skill.Name, skill.Path)
				}
			}
			if len(entry.Commands) > 0 {
				fmt.Printf("    Commands:\n")
				for _, command := range entry.Commands {
					fmt.Printf("      - /%s: %s\n", command.Name, command.Path)
				}
			}
			fmt.Printf("    Installed: %s\n", entry.InstalledAt)
			fmt.Println()
		}
	}

	return nil
}

func runPluginSearch(cmd *cobra.Command, args []string) error {
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

	// Branch: TUI mode (no args) or text mode (with keyword)
	if len(args) == 0 {
		return runInteractiveSearch(manifests)
	}

	return runTextSearch(manifests, args[0])
}

// runInteractiveSearch runs the TUI fuzzy finder with install/uninstall support
func runInteractiveSearch(manifests map[string]*marketplace.MarketplaceManifest) error {
	result, err := tui.RunPluginFinder(manifests)
	if err != nil {
		return err
	}

	if result.Cancelled {
		fmt.Println(i18n.T("SearchCancelled", nil))
		return nil
	}

	// Check if there are any changes
	if len(result.ToInstall) == 0 && len(result.ToUninstall) == 0 {
		fmt.Println(i18n.T("NoChanges", nil))
		return nil
	}

	// Process installs
	if len(result.ToInstall) > 0 {
		fmt.Println()
		fmt.Println(i18n.T("InstallingPlugins", map[string]any{"Count": len(result.ToInstall)}, len(result.ToInstall)))
		for _, item := range result.ToInstall {
			pluginID := fmt.Sprintf("%s@%s", item.Plugin.Name, item.Marketplace)
			if err := runPluginInstall(nil, []string{pluginID}); err != nil {
				fmt.Printf("  %s: %v\n", i18n.T("InstallFailed", map[string]any{"Plugin": pluginID}), err)
			}
		}
	}

	// Process uninstalls
	if len(result.ToUninstall) > 0 {
		fmt.Println()
		fmt.Println(i18n.T("UninstallingPlugins", map[string]any{"Count": len(result.ToUninstall)}, len(result.ToUninstall)))
		for _, item := range result.ToUninstall {
			pluginID := fmt.Sprintf("%s@%s", item.Plugin.Name, item.Marketplace)
			// Use global scope for uninstall
			pluginUninstallScope = "global"
			if err := runPluginUninstall(nil, []string{pluginID}); err != nil {
				fmt.Printf("  %s: %v\n", i18n.T("UninstallFailed", map[string]any{"Plugin": pluginID}), err)
			}
		}
	}

	fmt.Println()
	return nil
}

// runTextSearch performs the existing text-based search
func runTextSearch(manifests map[string]*marketplace.MarketplaceManifest, keyword string) error {
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
