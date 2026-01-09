package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/marketplace"
	"github.com/egoavara/codex-market/internal/plugin"
	"github.com/sahilm/fuzzy"
)

// PluginItem wraps plugin info with marketplace name and install state
type PluginItem struct {
	Plugin      marketplace.PluginEntry
	Marketplace string
	Installed   bool // currently installed (global)
	Selected    bool // user toggled selection
}

// PluginID returns the unique identifier for this plugin
func (p PluginItem) PluginID() string {
	return fmt.Sprintf("%s@%s", p.Plugin.Name, p.Marketplace)
}

// Action returns what action will be performed on this plugin
// Returns: "install", "uninstall", or "" (no action)
func (p PluginItem) Action() string {
	if p.Installed && !p.Selected {
		return "uninstall"
	}
	if !p.Installed && p.Selected {
		return "install"
	}
	return ""
}

// FinderResult holds the result of TUI selection
type FinderResult struct {
	ToInstall   []PluginItem
	ToUninstall []PluginItem
	Cancelled   bool
}

// ViewMode represents the current view mode
type ViewMode int

const (
	ModeList ViewMode = iota
	ModeConfirm
)

// Model is the bubbletea model for plugin finder
type Model struct {
	items         []PluginItem
	filteredItems []PluginItem
	cursor        int
	width         int
	height        int
	searchInput   textinput.Model
	mode          ViewMode
	quitting      bool
	confirmed     bool
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	installedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("34"))

	toInstallStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	toUninstallStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	previewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2).
			Align(lipgloss.Center)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// NewModel creates a new finder model
func NewModel(items []PluginItem) Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 50
	ti.Width = 30

	return Model{
		items:         items,
		filteredItems: items,
		searchInput:   ti,
		mode:          ModeList,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode == ModeConfirm {
		return m.handleConfirmKey(msg)
	}

	return m.handleListKey(msg)
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "esc":
		// If search has text, clear it; otherwise quit
		if m.searchInput.Value() != "" {
			m.searchInput.SetValue("")
			m.applyFilter()
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit

	case "up":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down":
		if m.cursor < len(m.filteredItems)-1 {
			m.cursor++
		}

	case "tab":
		if len(m.filteredItems) > 0 {
			idx := m.findOriginalIndex(m.cursor)
			if idx >= 0 {
				m.items[idx].Selected = !m.items[idx].Selected
				m.applyFilter()
			}
		}

	case "enter":
		if m.hasChanges() {
			m.mode = ModeConfirm
		}

	case "backspace":
		// Handle backspace for search
		val := m.searchInput.Value()
		if len(val) > 0 {
			m.searchInput.SetValue(val[:len(val)-1])
			m.applyFilter()
		}

	default:
		// Any other printable character goes to search
		if len(msg.String()) == 1 && msg.String()[0] >= 32 && msg.String()[0] < 127 {
			m.searchInput.SetValue(m.searchInput.Value() + msg.String())
			m.applyFilter()
		}
	}

	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.confirmed = true
		m.quitting = true
		return m, tea.Quit

	case "n", "N", "esc", "q":
		m.mode = ModeList
		return m, nil
	}
	return m, nil
}

func (m *Model) applyFilter() {
	query := m.searchInput.Value()
	if query == "" {
		m.filteredItems = m.items
		if m.cursor >= len(m.filteredItems) {
			m.cursor = max(0, len(m.filteredItems)-1)
		}
		return
	}

	// Build searchable strings
	searchables := make([]string, len(m.items))
	for i, item := range m.items {
		parts := []string{item.Plugin.Name, item.Marketplace}
		if item.Plugin.Description != "" {
			parts = append(parts, item.Plugin.Description)
		}
		parts = append(parts, item.Plugin.Tags...)
		parts = append(parts, item.Plugin.Keywords...)
		searchables[i] = strings.ToLower(strings.Join(parts, " "))
	}

	matches := fuzzy.Find(strings.ToLower(query), searchables)
	m.filteredItems = make([]PluginItem, len(matches))
	for i, match := range matches {
		m.filteredItems[i] = m.items[match.Index]
	}

	if m.cursor >= len(m.filteredItems) {
		m.cursor = max(0, len(m.filteredItems)-1)
	}
}

func (m Model) findOriginalIndex(filteredIdx int) int {
	if filteredIdx < 0 || filteredIdx >= len(m.filteredItems) {
		return -1
	}
	target := m.filteredItems[filteredIdx]
	for i, item := range m.items {
		if item.PluginID() == target.PluginID() {
			return i
		}
	}
	return -1
}

func (m Model) hasChanges() bool {
	for _, item := range m.items {
		if item.Action() != "" {
			return true
		}
	}
	return false
}

func (m Model) getChanges() (toInstall, toUninstall []PluginItem) {
	for _, item := range m.items {
		switch item.Action() {
		case "install":
			toInstall = append(toInstall, item)
		case "uninstall":
			toUninstall = append(toUninstall, item)
		}
	}
	return
}

func (m Model) View() string {
	if m.quitting && !m.confirmed {
		return ""
	}

	if m.mode == ModeConfirm {
		return m.renderConfirmModal()
	}

	return m.renderListView()
}

func (m Model) renderListView() string {
	var b strings.Builder

	// Header
	header := titleStyle.Render(i18n.T("TUIHeader", map[string]any{"Count": len(m.items)}))
	b.WriteString(header)
	b.WriteString("\n\n")

	// Calculate layout
	listWidth := 40
	previewWidth := max(30, m.width-listWidth-6)
	listHeight := max(5, m.height-8)

	// Build list
	var listLines []string
	for i, item := range m.filteredItems {
		line := m.renderItem(i, item)
		listLines = append(listLines, line)
	}

	// Paginate if needed
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}
	end := min(start+listHeight, len(listLines))

	visibleList := strings.Join(listLines[start:end], "\n")

	// Build preview
	preview := m.renderPreview(previewWidth)

	// Combine list and preview horizontally
	listBox := lipgloss.NewStyle().Width(listWidth).Render(visibleList)
	previewBox := previewStyle.Width(previewWidth).Height(listHeight).Render(preview)

	content := lipgloss.JoinHorizontal(lipgloss.Top, listBox, "  ", previewBox)
	b.WriteString(content)
	b.WriteString("\n\n")

	// Search bar (always visible)
	searchQuery := m.searchInput.Value()
	if searchQuery != "" {
		b.WriteString("> " + searchQuery + "_")
	} else {
		b.WriteString(helpStyle.Render("> type to filter..."))
	}
	b.WriteString("\n")

	// Help
	help := helpStyle.Render("↑/↓: move | Tab: toggle | Enter: confirm | Esc: clear/quit")
	b.WriteString(help)

	return b.String()
}

func (m Model) renderItem(idx int, item PluginItem) string {
	cursor := "  "
	if idx == m.cursor {
		cursor = "> "
	}

	// Checkbox state based on installed status and selection
	var checkbox string
	var style lipgloss.Style

	action := item.Action()
	switch {
	case action == "install":
		checkbox = "[+]"
		style = toInstallStyle
	case action == "uninstall":
		checkbox = "[-]"
		style = toUninstallStyle
	case item.Installed:
		checkbox = "[*]"
		style = installedStyle
	default:
		checkbox = "[ ]"
		style = normalStyle
	}

	version := item.Plugin.Version
	if version == "" {
		version = "latest"
	}

	text := fmt.Sprintf("%s%s %s@%s (v%s)",
		cursor, checkbox, item.Plugin.Name, item.Marketplace, version)

	if idx == m.cursor {
		return selectedStyle.Render(text)
	}
	return style.Render(text)
}

func (m Model) renderPreview(width int) string {
	if len(m.filteredItems) == 0 || m.cursor >= len(m.filteredItems) {
		return i18n.T("TUIPreviewEmpty", nil)
	}

	item := m.filteredItems[m.cursor]
	p := item.Plugin

	var b strings.Builder

	b.WriteString(fmt.Sprintf("Name: %s\n", p.Name))
	b.WriteString(fmt.Sprintf("Marketplace: %s\n", item.Marketplace))

	version := p.Version
	if version == "" {
		version = "latest"
	}
	b.WriteString(fmt.Sprintf("Version: %s\n", version))

	if item.Installed {
		b.WriteString(installedStyle.Render("Status: Installed") + "\n")
	}

	b.WriteString("\n")

	if p.Description != "" {
		b.WriteString(fmt.Sprintf("Description:\n  %s\n\n", p.Description))
	}

	if p.Category != "" {
		b.WriteString(fmt.Sprintf("Category: %s\n", p.Category))
	}

	if len(p.Tags) > 0 {
		b.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(p.Tags, ", ")))
	}

	if p.Author != nil && p.Author.Name != "" {
		b.WriteString(fmt.Sprintf("Author: %s\n", p.Author.Name))
	}

	if p.License != "" {
		b.WriteString(fmt.Sprintf("License: %s\n", p.License))
	}

	return b.String()
}

func (m Model) renderConfirmModal() string {
	toInstall, toUninstall := m.getChanges()

	var b strings.Builder

	b.WriteString(i18n.T("ConfirmTitle", nil))
	b.WriteString("\n\n")

	if len(toInstall) > 0 {
		b.WriteString(toInstallStyle.Render(i18n.T("ToInstall", map[string]any{"Count": len(toInstall)}, len(toInstall))))
		b.WriteString("\n")
		for _, item := range toInstall {
			version := item.Plugin.Version
			if version == "" {
				version = "latest"
			}
			b.WriteString(fmt.Sprintf("  + %s@%s (v%s)\n", item.Plugin.Name, item.Marketplace, version))
		}
		b.WriteString("\n")
	}

	if len(toUninstall) > 0 {
		b.WriteString(toUninstallStyle.Render(i18n.T("ToUninstall", map[string]any{"Count": len(toUninstall)}, len(toUninstall))))
		b.WriteString("\n")
		for _, item := range toUninstall {
			b.WriteString(fmt.Sprintf("  - %s@%s\n", item.Plugin.Name, item.Marketplace))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("[y] " + i18n.T("Confirm", nil) + "  [n] " + i18n.T("Cancel", nil)))

	return modalStyle.Render(b.String())
}

// RunPluginFinder launches the interactive fuzzy finder for plugins
func RunPluginFinder(manifests map[string]*marketplace.MarketplaceManifest) (*FinderResult, error) {
	// Get installed plugins
	installedMgr := plugin.GetInstalled()

	// Collect all plugins into a flat list
	var items []PluginItem
	for mpName, manifest := range manifests {
		if manifest == nil {
			continue
		}
		for _, p := range manifest.Plugins {
			pluginID := fmt.Sprintf("%s@%s", p.Name, mpName)
			isInstalled, _ := installedMgr.Exists(pluginID)

			items = append(items, PluginItem{
				Plugin:      p,
				Marketplace: mpName,
				Installed:   isInstalled,
				Selected:    isInstalled, // Start with installed plugins selected
			})
		}
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("NoPluginsAvailable", nil))
	}

	// Sort by marketplace name, then by plugin name
	sort.Slice(items, func(i, j int) bool {
		if items[i].Marketplace != items[j].Marketplace {
			return items[i].Marketplace < items[j].Marketplace
		}
		return items[i].Plugin.Name < items[j].Plugin.Name
	})

	// Run the TUI
	model := NewModel(items)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	m := finalModel.(Model)

	if !m.confirmed {
		return &FinderResult{Cancelled: true}, nil
	}

	toInstall, toUninstall := m.getChanges()
	return &FinderResult{
		ToInstall:   toInstall,
		ToUninstall: toUninstall,
		Cancelled:   false,
	}, nil
}
