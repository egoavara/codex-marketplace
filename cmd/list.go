package cmd

import (
	"fmt"
	"strings"

	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/marketplace"
	"github.com/egoavara/codex-market/internal/plugin"
	"github.com/spf13/cobra"
)

var (
	listAll bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered marketplaces and installed plugins",
	Long: `List all registered marketplaces and installed plugins.

Example:
  codex-market list
  codex-market list --all  # Also show available plugins`,
	RunE: runList,
}

func init() {
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "show available plugins from marketplaces")
}

func runList(cmd *cobra.Command, args []string) error {
	// List marketplaces
	registry := marketplace.GetRegistry()
	marketplaces, err := registry.List()
	if err != nil {
		return err
	}

	fmt.Println(i18n.T("ListMarketplacesHeader", nil))
	fmt.Println(strings.Repeat("-", 40))

	if len(marketplaces) == 0 {
		fmt.Println(i18n.T("NoMarketplaces", nil))
	} else {
		for name, mp := range marketplaces {
			fmt.Printf("  %s\n", name)
			fmt.Printf("    URL: %s\n", mp.Source.URL)
			fmt.Printf("    Path: %s\n", mp.InstallLocation)
			fmt.Printf("    Updated: %s\n", mp.LastUpdated)

			// Show available plugins if --all flag
			if listAll {
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
	}

	// List installed plugins
	fmt.Println()
	fmt.Println(i18n.T("ListPluginsHeader", nil))
	fmt.Println(strings.Repeat("-", 40))

	installed, err := plugin.GetInstalled().List()
	if err != nil {
		return err
	}

	if len(installed.Plugins) == 0 {
		fmt.Println(i18n.T("NoPluginsInstalled", nil))
	} else {
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
	}

	return nil
}
