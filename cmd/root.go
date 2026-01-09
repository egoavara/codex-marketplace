package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	verbose bool

	rootCmd = &cobra.Command{
		Use:           "codex-market",
		Short:         "CLI tool for managing Claude Code plugins",
		SilenceErrors: true,
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

// createAliasCommand creates a root-level alias that shares flags with a plugin subcommand
func createAliasCommand(pluginSubCmd *cobra.Command, aliases []string) *cobra.Command {
	aliasCmd := &cobra.Command{
		Use:     pluginSubCmd.Use,
		Short:   pluginSubCmd.Short + " (alias)",
		Long:    pluginSubCmd.Long,
		Args:    pluginSubCmd.Args,
		Aliases: aliases,
		RunE:    pluginSubCmd.RunE,
	}
	// Copy all flags from the original command
	pluginSubCmd.Flags().VisitAll(func(f *pflag.Flag) {
		aliasCmd.Flags().AddFlag(f)
	})
	return aliasCmd
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
}

// RegisterPluginAliases registers root-level aliases for plugin subcommands
// Must be called after plugin subcommands are initialized
func RegisterPluginAliases() {
	rootCmd.AddCommand(createAliasCommand(pluginInstallCmd, nil))
	rootCmd.AddCommand(createAliasCommand(pluginUninstallCmd, []string{"remove", "rm"}))
	rootCmd.AddCommand(createAliasCommand(pluginSearchCmd, nil))
	rootCmd.AddCommand(createAliasCommand(pluginUpdateCmd, nil))
}
