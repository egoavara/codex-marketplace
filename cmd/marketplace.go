package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoavara/codex-market/internal/config"
	"github.com/egoavara/codex-market/internal/git"
	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/marketplace"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:     "marketplace",
	Aliases: []string{"mp"},
	Short:   "Manage plugin marketplaces",
	Long: `Manage plugin marketplaces (similar to 'brew tap').

Commands:
  add     Add a new marketplace from git URL
  del     Remove a registered marketplace
  list    List all registered marketplaces
  update  Update marketplace(s)`,
}

var marketplaceAddCmd = &cobra.Command{
	Use:   "add <git-url>",
	Short: "Add a plugin marketplace repository",
	Long: `Add a plugin marketplace repository from a git URL.

Example:
  codex-market marketplace add https://github.com/org/my-plugins
  codex-market mp add git@github.com:org/my-plugins.git`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceAdd,
}

var marketplaceDelCmd = &cobra.Command{
	Use:     "del <name>",
	Aliases: []string{"delete", "remove", "rm"},
	Short:   "Remove a registered marketplace",
	Long: `Remove a registered marketplace.

Example:
  codex-market marketplace del my-marketplace`,
	Args: cobra.ExactArgs(1),
	RunE: runMarketplaceDel,
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered marketplaces",
	Long: `List all registered marketplaces.

Example:
  codex-market marketplace list
  codex-market mp list --all  # Show available plugins`,
	RunE: runMarketplaceList,
}

var marketplaceUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update marketplace(s)",
	Long: `Update all marketplaces or a specific marketplace.

Example:
  codex-market marketplace update              # Update all
  codex-market marketplace update my-marketplace  # Update specific`,
	RunE: runMarketplaceUpdate,
}

var (
	marketplaceListAll bool
)

func init() {
	marketplaceListCmd.Flags().BoolVarP(&marketplaceListAll, "all", "a", false, "show available plugins from marketplaces")

	marketplaceCmd.AddCommand(marketplaceAddCmd)
	marketplaceCmd.AddCommand(marketplaceDelCmd)
	marketplaceCmd.AddCommand(marketplaceListCmd)
	marketplaceCmd.AddCommand(marketplaceUpdateCmd)
}

func runMarketplaceAdd(cmd *cobra.Command, args []string) error {
	url := args[0]

	// Extract repository name from URL
	repoName := extractRepoName(url)
	if repoName == "" {
		return fmt.Errorf("failed to extract repository name from URL: %s", url)
	}

	// Check if already exists
	registry := marketplace.GetRegistry()
	exists, err := registry.Exists(repoName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf(i18n.T("AlreadyExists", map[string]any{"Name": repoName}))
	}

	// Ensure marketplaces directory exists
	if err := config.EnsureDir(config.MarketplacesDir()); err != nil {
		return err
	}

	// Clone the repository
	destPath := filepath.Join(config.MarketplacesDir(), repoName)
	gitClient := git.NewClient()

	fmt.Printf("Cloning %s...\n", url)
	if err := gitClient.Clone(url, destPath); err != nil {
		if authErr, ok := err.(*git.AuthError); ok {
			return fmt.Errorf(i18n.T("GitAuthFailed", map[string]any{"URL": authErr.URL}))
		}
		return fmt.Errorf(i18n.T("GitCloneFailed", map[string]any{"Error": err.Error()}))
	}

	// Load and validate marketplace manifest
	manifest, err := marketplace.LoadManifest(destPath)
	if err != nil {
		// Rollback: remove cloned directory
		os.RemoveAll(destPath)
		return fmt.Errorf(i18n.T("InvalidManifest", map[string]any{"Path": destPath}))
	}

	// Use the name from manifest if available
	marketplaceName := manifest.Name
	if marketplaceName == "" {
		marketplaceName = repoName
	}

	// Register the marketplace
	if err := registry.Add(marketplaceName, url, destPath); err != nil {
		os.RemoveAll(destPath)
		return err
	}

	// Success message
	pluginCount := len(manifest.Plugins)
	fmt.Println(i18n.T("AddSuccess", map[string]any{
		"Name":  marketplaceName,
		"Count": pluginCount,
	}, pluginCount))

	return nil
}

func runMarketplaceDel(cmd *cobra.Command, args []string) error {
	name := args[0]

	registry := marketplace.GetRegistry()
	mp, err := registry.Get(name)
	if err != nil {
		return err
	}
	if mp == nil {
		return fmt.Errorf(i18n.T("MarketplaceNotFound", map[string]any{"Name": name}))
	}

	// Remove the directory
	if mp.InstallLocation != "" {
		if err := os.RemoveAll(mp.InstallLocation); err != nil {
			fmt.Printf("Warning: failed to remove directory %s: %v\n", mp.InstallLocation, err)
		}
	}

	// Remove from registry
	if err := registry.Remove(name); err != nil {
		return err
	}

	fmt.Println(i18n.T("MarketplaceRemoved", map[string]any{"Name": name}))
	return nil
}

func runMarketplaceList(cmd *cobra.Command, args []string) error {
	registry := marketplace.GetRegistry()
	marketplaces, err := registry.List()
	if err != nil {
		return err
	}

	fmt.Println(i18n.T("ListMarketplacesHeader", nil))
	fmt.Println(strings.Repeat("-", 40))

	if len(marketplaces) == 0 {
		fmt.Println(i18n.T("NoMarketplaces", nil))
		return nil
	}

	for name, mp := range marketplaces {
		fmt.Printf("  %s\n", name)
		fmt.Printf("    URL: %s\n", mp.Source.URL)
		fmt.Printf("    Path: %s\n", mp.InstallLocation)
		fmt.Printf("    Updated: %s\n", mp.LastUpdated)

		// Show available plugins if --all flag
		if marketplaceListAll {
			manifest, err := marketplace.LoadManifest(mp.InstallLocation)
			if err == nil && len(manifest.Plugins) > 0 {
				fmt.Println("    Plugins:")
				for _, p := range manifest.Plugins {
					version := p.Version
					if version == "" {
						version = "latest"
					}
					fmt.Printf("      - %s (v%s)\n", p.Name, version)
					if p.Description != "" {
						fmt.Printf("        %s\n", p.Description)
					}
				}
			}
		}
		fmt.Println()
	}

	return nil
}

func runMarketplaceUpdate(cmd *cobra.Command, args []string) error {
	gitClient := git.NewClient()
	registry := marketplace.GetRegistry()

	if len(args) == 0 {
		// Update all marketplaces
		return updateAllMarketplaces(gitClient, registry)
	}

	// Update single marketplace
	return updateMarketplace(gitClient, registry, args[0])
}

// extractRepoName extracts the repository name from a git URL
func extractRepoName(url string) string {
	// Remove trailing .git
	url = strings.TrimSuffix(url, ".git")

	// Handle various URL formats
	// https://github.com/org/repo
	// git@github.com:org/repo
	// github.com/org/repo

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

func updateAllMarketplaces(gitClient *git.DefaultClient, registry *marketplace.Registry) error {
	marketplaces, err := registry.List()
	if err != nil {
		return err
	}

	if len(marketplaces) == 0 {
		fmt.Println(i18n.T("NoMarketplaces", nil))
		return nil
	}

	for name, mp := range marketplaces {
		fmt.Printf("Updating %s...\n", name)
		if err := gitClient.Pull(mp.InstallLocation); err != nil {
			if authErr, ok := err.(*git.AuthError); ok {
				fmt.Printf("  Error: %s\n", i18n.T("GitAuthFailed", map[string]any{"URL": authErr.URL}))
			} else {
				fmt.Printf("  Error: %s\n", i18n.T("GitPullFailed", map[string]any{"Error": err.Error()}))
			}
			continue
		}
		registry.UpdateTimestamp(name)
		fmt.Printf("  Done\n")
	}

	fmt.Println(i18n.T("UpdateAllSuccess", nil))
	return nil
}

func updateMarketplace(gitClient *git.DefaultClient, registry *marketplace.Registry, name string) error {
	mp, err := registry.Get(name)
	if err != nil {
		return err
	}
	if mp == nil {
		return fmt.Errorf(i18n.T("MarketplaceNotFound", map[string]any{"Name": name}))
	}

	fmt.Printf("Updating %s...\n", name)
	if err := gitClient.Pull(mp.InstallLocation); err != nil {
		if authErr, ok := err.(*git.AuthError); ok {
			return fmt.Errorf(i18n.T("GitAuthFailed", map[string]any{"URL": authErr.URL}))
		}
		return fmt.Errorf(i18n.T("GitPullFailed", map[string]any{"Error": err.Error()}))
	}

	registry.UpdateTimestamp(name)
	fmt.Println(i18n.T("UpdateSuccess", map[string]any{"Target": name}))
	return nil
}
