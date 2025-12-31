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
	"github.com/spf13/cobra"
)

var (
	installScope string
)

var installCmd = &cobra.Command{
	Use:   "install <plugin>@<marketplace>",
	Short: "Install a plugin from a marketplace",
	Long: `Install a plugin from a registered marketplace.

Example:
  codex-market install my-plugin@my-marketplace
  codex-market install my-plugin@my-marketplace -s project`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringVarP(&installScope, "scope", "s", "global", "install scope (global or project)")
}

func runInstall(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	// Parse plugin identifier
	pluginName, marketplaceName, err := parsePluginIdentifier(identifier)
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
	if installScope == "project" {
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
		Scope:       installScope,
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

	if installScope == "project" {
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

// parsePluginIdentifier parses "plugin@marketplace" format
func parsePluginIdentifier(identifier string) (string, string, error) {
	parts := strings.Split(identifier, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf(i18n.T("InvalidPluginIdentifier", map[string]any{
			"Identifier": identifier,
		}))
	}
	return parts[0], parts[1], nil
}
