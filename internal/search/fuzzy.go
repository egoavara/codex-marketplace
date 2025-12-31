package search

import (
	"sort"
	"strings"

	"github.com/egoavara/codex-market/internal/marketplace"
	"github.com/sahilm/fuzzy"
)

// SearchResult represents a search result
type SearchResult struct {
	Plugin      marketplace.PluginEntry
	Marketplace string
	Score       int // Higher is better
}

// PluginSearchable wraps plugins for fuzzy searching
type PluginSearchable struct {
	Plugins     []marketplace.PluginEntry
	Marketplace string
}

// String returns the searchable string for a plugin
func (p PluginSearchable) String(i int) string {
	plugin := p.Plugins[i]
	parts := []string{plugin.Name}

	if plugin.Description != "" {
		parts = append(parts, plugin.Description)
	}

	parts = append(parts, plugin.Tags...)
	parts = append(parts, plugin.Keywords...)

	if plugin.Category != "" {
		parts = append(parts, plugin.Category)
	}

	return strings.ToLower(strings.Join(parts, " "))
}

// Len returns the number of plugins
func (p PluginSearchable) Len() int {
	return len(p.Plugins)
}

// FuzzySearch performs a fuzzy search across all plugins
func FuzzySearch(marketplaces map[string]*marketplace.MarketplaceManifest, query string) []SearchResult {
	var results []SearchResult
	query = strings.ToLower(query)

	for mpName, manifest := range marketplaces {
		if manifest == nil || len(manifest.Plugins) == 0 {
			continue
		}

		searchable := PluginSearchable{
			Plugins:     manifest.Plugins,
			Marketplace: mpName,
		}

		matches := fuzzy.FindFrom(query, searchable)

		for _, match := range matches {
			results = append(results, SearchResult{
				Plugin:      manifest.Plugins[match.Index],
				Marketplace: mpName,
				Score:       match.Score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// SimpleSearch performs a simple substring search
func SimpleSearch(marketplaces map[string]*marketplace.MarketplaceManifest, query string) []SearchResult {
	var results []SearchResult
	query = strings.ToLower(query)

	for mpName, manifest := range marketplaces {
		if manifest == nil {
			continue
		}

		for _, plugin := range manifest.Plugins {
			if matchesQuery(plugin, query) {
				results = append(results, SearchResult{
					Plugin:      plugin,
					Marketplace: mpName,
					Score:       100, // Default score for simple matches
				})
			}
		}
	}

	return results
}

// matchesQuery checks if a plugin matches the search query
func matchesQuery(plugin marketplace.PluginEntry, query string) bool {
	// Check name
	if strings.Contains(strings.ToLower(plugin.Name), query) {
		return true
	}

	// Check description
	if strings.Contains(strings.ToLower(plugin.Description), query) {
		return true
	}

	// Check tags
	for _, tag := range plugin.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}

	// Check keywords
	for _, keyword := range plugin.Keywords {
		if strings.Contains(strings.ToLower(keyword), query) {
			return true
		}
	}

	// Check category
	if strings.Contains(strings.ToLower(plugin.Category), query) {
		return true
	}

	return false
}
