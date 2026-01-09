package autoupdate

// UpdateType represents the type of updatable item
type UpdateType string

const (
	UpdateTypeMarketplace UpdateType = "marketplace"
	UpdateTypePlugin      UpdateType = "plugin"
)

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	Type       UpdateType // "marketplace" or "plugin"
	Name       string     // Name of the item
	CurrentVer string     // Current version/commit
	RemoteVer  string     // Remote version/commit
	HasUpdate  bool       // Whether update is available
	Path       string     // Path to the item (for marketplace) or plugin ID
}

// CheckResult contains the result of update check
type CheckResult struct {
	Marketplaces []UpdateInfo
	Plugins      []UpdateInfo
	HasAnyUpdate bool
	Errors       []error // Non-fatal errors during check
}

// TotalUpdates returns the total number of available updates
func (r *CheckResult) TotalUpdates() int {
	count := 0
	for _, m := range r.Marketplaces {
		if m.HasUpdate {
			count++
		}
	}
	for _, p := range r.Plugins {
		if p.HasUpdate {
			count++
		}
	}
	return count
}
