package autoupdate

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/egoavara/codex-market/internal/i18n"
)

// ShowUpdateSummary displays a summary of available updates
func ShowUpdateSummary(result *CheckResult) {
	if !result.HasAnyUpdate {
		fmt.Println(i18n.T("update.noUpdates", nil))
		return
	}

	fmt.Println()
	fmt.Println(i18n.T("update.available", nil))
	fmt.Println()

	// Show marketplace updates
	for _, mp := range result.Marketplaces {
		if mp.HasUpdate {
			fmt.Printf("  [%s] %s (%s → %s)\n",
				i18n.T("update.typeMarketplace", nil),
				mp.Name,
				mp.CurrentVer,
				mp.RemoteVer,
			)
		}
	}

	// Show plugin updates
	for _, p := range result.Plugins {
		if p.HasUpdate {
			fmt.Printf("  [%s] %s (%s → %s)\n",
				i18n.T("update.typePlugin", nil),
				p.Name,
				p.CurrentVer,
				p.RemoteVer,
			)
		}
	}

	fmt.Println()
}

// PromptUpdate asks the user if they want to apply updates
func PromptUpdate(result *CheckResult) bool {
	if !result.HasAnyUpdate {
		return false
	}

	fmt.Print(i18n.T("update.prompt", nil) + " [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// Default to yes if empty or explicit yes
	return input == "" || input == "y" || input == "yes"
}

// PromptAliasSetup asks the user if they want to set up the codex alias
func PromptAliasSetup() bool {
	fmt.Println()
	fmt.Println(i18n.T("alias.prompt", nil))
	fmt.Println()
	fmt.Println("  alias codex=\"codex-market run\"")
	fmt.Println()
	fmt.Print(i18n.T("alias.confirm", nil) + " [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))

	return input == "" || input == "y" || input == "yes"
}
