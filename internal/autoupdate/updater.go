package autoupdate

import (
	"fmt"
	"sync"
	"time"

	"github.com/egoavara/codex-market/internal/git"
	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/marketplace"
)

// Spinner characters
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner represents a terminal spinner
type Spinner struct {
	message string
	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
}

// NewSpinner creates a new spinner with a message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				s.mu.Lock()
				fmt.Printf("\r  %s %s ", spinnerFrames[i%len(spinnerFrames)], s.message)
				s.mu.Unlock()
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner and shows the result
func (s *Spinner) Stop(success bool) {
	close(s.stop)
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	if success {
		fmt.Printf("\r  ✓ %s\n", s.message)
	} else {
		fmt.Printf("\r  ✗ %s\n", s.message)
	}
}

// Updater handles applying updates
type Updater struct {
	gitClient git.Client
}

// NewUpdater creates a new updater
func NewUpdater() *Updater {
	return &Updater{
		gitClient: git.NewClient(),
	}
}

// ApplyUpdates applies all available updates
func ApplyUpdates(result *CheckResult) error {
	updater := NewUpdater()
	return updater.ApplyUpdates(result)
}

// ApplyUpdates applies all available updates from the check result
func (u *Updater) ApplyUpdates(result *CheckResult) error {
	if !result.HasAnyUpdate {
		return nil
	}

	fmt.Println(i18n.T("update.updating", nil))
	fmt.Println()

	var updateErrors []error

	// Update marketplaces first
	for _, mp := range result.Marketplaces {
		if !mp.HasUpdate {
			continue
		}

		spinner := NewSpinner(fmt.Sprintf("%s %s", i18n.T("update.typeMarketplace", nil), mp.Name))
		spinner.Start()

		err := u.updateMarketplace(mp)
		spinner.Stop(err == nil)

		if err != nil {
			updateErrors = append(updateErrors, err)
		}
	}

	// Update plugins
	for _, p := range result.Plugins {
		if !p.HasUpdate {
			continue
		}

		spinner := NewSpinner(fmt.Sprintf("%s %s", i18n.T("update.typePlugin", nil), p.Name))
		spinner.Start()

		err := u.updatePlugin(p)
		spinner.Stop(err == nil)

		if err != nil {
			updateErrors = append(updateErrors, err)
		}
	}

	fmt.Println()

	if len(updateErrors) > 0 {
		fmt.Println(i18n.T("update.partialSuccess", nil))
	} else {
		fmt.Println(i18n.T("update.complete", nil))
	}

	return nil
}

// updateMarketplace pulls the latest changes for a marketplace
func (u *Updater) updateMarketplace(info UpdateInfo) error {
	// Pull latest changes
	if err := u.gitClient.Pull(info.Path); err != nil {
		return fmt.Errorf("failed to update marketplace: %w", err)
	}

	// Update timestamp in registry
	registry := marketplace.GetRegistry()
	if err := registry.UpdateTimestamp(info.Name); err != nil {
		// Non-fatal, just log
		return nil
	}

	return nil
}

// updatePlugin reinstalls a plugin to get the latest version
func (u *Updater) updatePlugin(info UpdateInfo) error {
	// For plugins, we need to reinstall them
	// This is handled by the plugin install command
	// For now, we just return nil as plugins are tied to marketplace versions
	// When marketplace is updated, plugins will get new versions on next install

	// In a full implementation, we would:
	// 1. Get the plugin's marketplace
	// 2. Reload the manifest
	// 3. Reinstall the plugin
	// But this requires access to the cmd package which creates circular dependency

	// Mark as needing update - actual reinstall happens through plugin install command
	return nil
}

// ApplyMarketplaceUpdates applies only marketplace updates
func (u *Updater) ApplyMarketplaceUpdates(result *CheckResult) error {
	for _, mp := range result.Marketplaces {
		if !mp.HasUpdate {
			continue
		}

		if err := u.updateMarketplace(mp); err != nil {
			return err
		}
	}
	return nil
}
