package cmd

import (
	"fmt"
	"strings"

	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/marketplace"
	"github.com/egoavara/codex-market/internal/search"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search for plugins across all marketplaces",
	Long: `Search for plugins using fuzzy matching across all registered marketplaces.

The search looks through plugin names, descriptions, tags, and keywords.

Example:
  codex-market search formatter
  codex-market search code-review`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
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
