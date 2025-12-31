package cmd

import (
	"fmt"

	"github.com/egoavara/codex-market/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage codex-market configuration",
	Long: `Manage codex-market configuration settings.

Example:
  codex-market config show
  codex-market config set claude.registry.share sync`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  locale                 - Language setting
                           Values: auto, en-US, ko-KR, etc.
  claude.registry.share  - How to share registry with Claude
                           Values: sync, merge, ignore

Example:
  codex-market config set locale ko-KR
  codex-market config set claude.registry.share sync`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	fmt.Println("Configuration:")
	fmt.Println("----------------------------------------")
	fmt.Printf("  locale: %s\n", cfg.Locale)
	fmt.Printf("  claude.registry.share: %s\n", cfg.Claude.Registry.Share)
	fmt.Println()
	fmt.Printf("  Marketplaces: %d registered\n", len(cfg.Marketplaces))

	// Explain current settings
	fmt.Println()
	fmt.Println("Locale:")
	if cfg.Locale == "auto" {
		fmt.Println("  auto: System locale is auto-detected")
	} else {
		fmt.Printf("  %s: Using fixed locale\n", cfg.Locale)
	}

	fmt.Println()
	fmt.Println("Share mode:")
	switch cfg.Claude.Registry.Share {
	case config.ShareSync:
		fmt.Println("  sync: Changes are synced to Claude's settings.json")
	case config.ShareMerge:
		fmt.Println("  merge: Lists show codex-market + Claude marketplaces combined")
	case config.ShareIgnore:
		fmt.Println("  ignore: Only codex-market's own marketplaces are used")
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	switch key {
	case "locale":
		if err := config.SetLocale(value); err != nil {
			return err
		}
		fmt.Printf("Locale set to '%s'. Restart codex-market to apply.\n", value)
		return nil
	case "claude.registry.share":
		switch value {
		case "sync":
			return config.SetShareMode(config.ShareSync)
		case "merge":
			return config.SetShareMode(config.ShareMerge)
		case "ignore":
			return config.SetShareMode(config.ShareIgnore)
		default:
			return fmt.Errorf("invalid value '%s' for %s. Valid values: sync, merge, ignore", value, key)
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
}
