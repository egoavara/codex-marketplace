package cmd

import (
	"fmt"
	"os"

	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/plugin"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <plugin>@<marketplace>",
	Aliases: []string{"uninstall", "rm"},
	Short:   "Remove an installed plugin",
	Long: `Remove an installed plugin.

Example:
  codex-market remove my-plugin@my-marketplace`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func runRemove(cmd *cobra.Command, args []string) error {
	pluginID := args[0]

	// Check if installed
	installed := plugin.GetInstalled()
	entries, err := installed.Get(pluginID)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf(i18n.T("NotInstalled", map[string]any{"Plugin": pluginID}))
	}

	// Remove skill directories and cache based on installed.json
	for _, entry := range entries {
		// Remove each skill folder
		for _, skill := range entry.Skills {
			if err := os.RemoveAll(skill.Path); err != nil {
				fmt.Printf("Warning: failed to remove skill %s at %s: %v\n", skill.Name, skill.Path, err)
			} else {
				fmt.Printf("  Removed skill: %s (%s)\n", skill.Name, skill.Path)
			}
		}

		// Remove cache directory
		if entry.Source.CachePath != "" {
			if err := os.RemoveAll(entry.Source.CachePath); err != nil {
				fmt.Printf("Warning: failed to remove cache %s: %v\n", entry.Source.CachePath, err)
			}
		}
	}

	// Remove from installed plugins
	if err := installed.Remove(pluginID); err != nil {
		return err
	}

	// Success message
	fmt.Println(i18n.T("RemoveSuccess", map[string]any{"Plugin": pluginID}))

	return nil
}
