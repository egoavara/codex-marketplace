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
add plugin repositories and install plugins from them.`,
	}
)

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(configCmd)
}
