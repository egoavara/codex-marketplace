package cmd

import (
	"fmt"
	"strings"

	"github.com/egoavara/codex-market/internal/git"
	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/marketplace"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [marketplace or plugin@marketplace]",
	Short: "Update marketplaces or plugins",
	Long: `Update all marketplaces, a specific marketplace, or a specific plugin.

Example:
  codex-market update                    # Update all marketplaces
  codex-market update my-marketplace     # Update specific marketplace
  codex-market update plugin@marketplace # Update specific plugin`,
	RunE: runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	gitClient := git.NewClient()
	registry := marketplace.GetRegistry()

	if len(args) == 0 {
		// Update all marketplaces
		return updateAllMarketplaces(gitClient, registry)
	}

	target := args[0]

	// Check if it's a plugin identifier (contains @)
	if strings.Contains(target, "@") {
		pluginName, marketplaceName, err := parsePluginIdentifier(target)
		if err != nil {
			return err
		}
		return updatePlugin(gitClient, registry, pluginName, marketplaceName)
	}

	// Update single marketplace
	return updateMarketplace(gitClient, registry, target)
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

func updatePlugin(gitClient *git.DefaultClient, registry *marketplace.Registry, pluginName, marketplaceName string) error {
	// First update the marketplace
	if err := updateMarketplace(gitClient, registry, marketplaceName); err != nil {
		return err
	}

	// Then reinstall the plugin
	pluginID := fmt.Sprintf("%s@%s", pluginName, marketplaceName)
	fmt.Printf("Reinstalling %s...\n", pluginID)

	// Use the install command logic
	installArgs := []string{pluginID}
	return runInstall(nil, installArgs)
}
