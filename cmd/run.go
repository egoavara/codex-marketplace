package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/egoavara/codex-market/internal/autoupdate"
	"github.com/egoavara/codex-market/internal/config"
	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/shell"
	"github.com/egoavara/codex-market/internal/tui"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:                "run [codex args...]",
	Short:              "Run codex with auto-update check",
	Long:               `Wrapper for codex that checks for updates before execution.`,
	DisableFlagParsing: true,
	RunE:               runCodexWrapper,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runCodexWrapper(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// 1. First-time alias setup prompt (TUI)
	if !cfg.AutoUpdate.RequestOverrideCodex {
		accepted, confirmed, err := tui.RunAliasConfirm()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: alias prompt failed: %v\n", err)
		}

		if confirmed && accepted {
			// User agreed to alias setup
			if err := setupAlias(); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("alias.error", nil), err)
			}

			// Enable auto-update
			cfg.AutoUpdate.Enabled = true

			// Show mode selector TUI
			fmt.Println()
			selectedMode, modeConfirmed, err := tui.RunModeSelector()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: mode selector failed: %v\n", err)
				selectedMode = config.AutoUpdateModeNotify
			}
			if modeConfirmed {
				cfg.AutoUpdate.Mode = selectedMode
			} else {
				cfg.AutoUpdate.Mode = config.AutoUpdateModeNotify
			}
			fmt.Println()
		}
		// Mark as prompted regardless of result
		cfg.AutoUpdate.RequestOverrideCodex = true
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
		}
	}

	// 2. Check for updates (if enabled and not disabled mode)
	if cfg.AutoUpdate.Enabled && cfg.AutoUpdate.Mode != config.AutoUpdateModeDisabled {
		fmt.Println(i18n.T("update.checking", nil))

		result, err := autoupdate.CheckAll()
		if err != nil {
			// Non-fatal: just continue to codex
			fmt.Fprintf(os.Stderr, "Warning: update check failed: %v\n", err)
		} else if result.HasAnyUpdate {
			if cfg.AutoUpdate.Mode == config.AutoUpdateModeAuto {
				// Auto mode: apply updates without asking
				if err := autoupdate.ApplyUpdates(result); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: update failed: %v\n", err)
				}
			} else {
				// Notify mode: show summary and ask
				autoupdate.ShowUpdateSummary(result)
				if autoupdate.PromptUpdate(result) {
					if err := autoupdate.ApplyUpdates(result); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: update failed: %v\n", err)
					}
				} else {
					fmt.Println(i18n.T("update.skipped", nil))
				}
			}
		} else {
			fmt.Println(i18n.T("update.noUpdates", nil))
		}
		fmt.Println()
	}

	// 3. Execute codex with all arguments
	return execCodex(args)
}

func setupAlias() error {
	shellType, err := shell.DetectShell()
	if err != nil {
		if errors.Is(err, shell.ErrUnsupportedShell) {
			// Unsupported shell - show manual setup instructions
			fmt.Println(i18n.T("alias.unsupportedShell", nil))
			fmt.Println()
			fmt.Println("  " + shell.AliasLine)
			fmt.Println()
			return nil
		}
		return fmt.Errorf("%s: %w", i18n.T("alias.shellDetectFailed", nil), err)
	}

	configPath, err := shell.GetShellConfigPath(shellType)
	if err != nil {
		if errors.Is(err, shell.ErrUnsupportedShell) {
			// Unsupported shell - show manual setup instructions
			fmt.Println(i18n.T("alias.unsupportedShell", nil))
			fmt.Println()
			fmt.Println("  " + shell.AliasLine)
			fmt.Println()
			return nil
		}
		return fmt.Errorf("%s: %w", i18n.T("alias.configPathFailed", nil), err)
	}

	// Check if alias already exists
	hasAlias, err := shell.HasCodexAlias(configPath)
	if err != nil {
		return err
	}
	if hasAlias {
		fmt.Println(i18n.T("alias.alreadyExists", nil))
		return nil
	}

	// Add alias
	if err := shell.AddCodexAlias(configPath); err != nil {
		return err
	}

	fmt.Println(i18n.T("alias.added", nil))
	fmt.Printf("%s: source %s\n", i18n.T("alias.reload", nil), configPath)

	return nil
}

func execCodex(args []string) error {
	codexPath, err := exec.LookPath("codex")
	if err != nil {
		return fmt.Errorf("codex not found in PATH: %w", err)
	}

	// Use syscall.Exec to replace the current process with codex
	// This ensures codex runs in the same terminal context
	argv := append([]string{"codex"}, args...)
	envv := os.Environ()

	return syscall.Exec(codexPath, argv, envv)
}
