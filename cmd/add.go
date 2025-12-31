package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoavara/codex-market/internal/config"
	"github.com/egoavara/codex-market/internal/git"
	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/marketplace"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <git-url>",
	Short: "Add a plugin marketplace repository",
	Long: `Add a plugin marketplace repository from a git URL.
This is similar to 'brew tap' for Homebrew.

Example:
  codex-market add https://github.com/org/my-plugins`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	url := args[0]

	// Extract repository name from URL
	repoName := extractRepoName(url)
	if repoName == "" {
		return fmt.Errorf("failed to extract repository name from URL: %s", url)
	}

	// Check if already exists
	registry := marketplace.GetRegistry()
	exists, err := registry.Exists(repoName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf(i18n.T("AlreadyExists", map[string]any{"Name": repoName}))
	}

	// Ensure marketplaces directory exists
	if err := config.EnsureDir(config.MarketplacesDir()); err != nil {
		return err
	}

	// Clone the repository
	destPath := filepath.Join(config.MarketplacesDir(), repoName)
	gitClient := git.NewClient()

	fmt.Printf("Cloning %s...\n", url)
	if err := gitClient.Clone(url, destPath); err != nil {
		if authErr, ok := err.(*git.AuthError); ok {
			return fmt.Errorf(i18n.T("GitAuthFailed", map[string]any{"URL": authErr.URL}))
		}
		return fmt.Errorf(i18n.T("GitCloneFailed", map[string]any{"Error": err.Error()}))
	}

	// Load and validate marketplace manifest
	manifest, err := marketplace.LoadManifest(destPath)
	if err != nil {
		// Rollback: remove cloned directory
		os.RemoveAll(destPath)
		return fmt.Errorf(i18n.T("InvalidManifest", map[string]any{"Path": destPath}))
	}

	// Use the name from manifest if available
	marketplaceName := manifest.Name
	if marketplaceName == "" {
		marketplaceName = repoName
	}

	// Register the marketplace
	if err := registry.Add(marketplaceName, url, destPath); err != nil {
		os.RemoveAll(destPath)
		return err
	}

	// Success message
	pluginCount := len(manifest.Plugins)
	fmt.Println(i18n.T("AddSuccess", map[string]any{
		"Name":  marketplaceName,
		"Count": pluginCount,
	}, pluginCount))

	return nil
}

// extractRepoName extracts the repository name from a git URL
func extractRepoName(url string) string {
	// Remove trailing .git
	url = strings.TrimSuffix(url, ".git")

	// Handle various URL formats
	// https://github.com/org/repo
	// git@github.com:org/repo
	// github.com/org/repo

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}
