package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose bool

	rootCmd = &cobra.Command{
		Use:   "codex-market",
		Short: "CLI tool for managing Claude Code plugins",
		Long: `codex-market is a third-party CLI tool for managing
Claude Code plugins from various marketplaces.

It works similar to 'brew tap' for Homebrew, allowing you to
add plugin repositories and install plugins from them.

Commands:
  marketplace  Manage plugin marketplaces (add, del, list, update)
  plugin       Manage plugins (install, uninstall, update, list, search)
  list         Show all marketplaces and installed plugins
  config       Manage configuration

Shortcuts (aliases):
  install      = plugin install
  uninstall    = plugin uninstall
  search       = plugin search`,
	}
)

// Alias commands for convenience
var installAliasCmd = &cobra.Command{
	Use:    "install <plugin>@<marketplace>",
	Short:  "Install a plugin (alias for 'plugin install')",
	Args:   cobra.ExactArgs(1),
	Hidden: false,
	RunE:   runPluginInstall,
}

var uninstallAliasCmd = &cobra.Command{
	Use:     "uninstall <plugin>@<marketplace>",
	Aliases: []string{"remove", "rm"},
	Short:   "Uninstall a plugin (alias for 'plugin uninstall')",
	Args:    cobra.ExactArgs(1),
	Hidden:  false,
	RunE:    runPluginUninstall,
}

var searchAliasCmd = &cobra.Command{
	Use:    "search <keyword>",
	Short:  "Search for plugins (alias for 'plugin search')",
	Args:   cobra.ExactArgs(1),
	Hidden: false,
	RunE:   runPluginSearch,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Main commands
	rootCmd.AddCommand(marketplaceCmd)
	rootCmd.AddCommand(pluginCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)

	// Alias commands (shortcuts)
	installAliasCmd.Flags().StringVarP(&pluginInstallScope, "scope", "s", "global", "install scope (global or project)")
	rootCmd.AddCommand(installAliasCmd)
	rootCmd.AddCommand(uninstallAliasCmd)
	rootCmd.AddCommand(searchAliasCmd)
}
