package cmd

import (
	"fmt"

	"github.com/egoavara/codex-market/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("codex-market %s\n", version.Version)
		if version.GitCommit != "" {
			fmt.Printf("  commit: %s\n", version.GitCommit)
		}
		if version.BuildDate != "" {
			fmt.Printf("  built:  %s\n", version.BuildDate)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
