package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/egoavara/codex-market/internal/i18n"
	"github.com/egoavara/codex-market/internal/shell"
)

// AliasOption represents an alias setup option
type AliasOption struct {
	Value       bool
	Label       string
	Description string
}

// AliasConfirmModel is the bubbletea model for alias confirmation
type AliasConfirmModel struct {
	options   []AliasOption
	cursor    int
	selected  bool
	quitting  bool
	confirmed bool
}

// Alias confirm styles - reuse mode selector styles for consistency
var (
	aliasCodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
)

// NewAliasConfirmModel creates a new alias confirmation model
func NewAliasConfirmModel() AliasConfirmModel {
	options := []AliasOption{
		{
			Value:       true,
			Label:       i18n.T("alias.option.yes", nil),
			Description: i18n.T("alias.option.yes.desc", nil),
		},
		{
			Value:       false,
			Label:       i18n.T("alias.option.no", nil),
			Description: i18n.T("alias.option.no.desc", nil),
		},
	}

	return AliasConfirmModel{
		options:  options,
		cursor:   0, // Default to yes
		selected: true,
	}
}

func (m AliasConfirmModel) Init() tea.Cmd {
	return nil
}

func (m AliasConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			m.selected = false
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}

		case "enter", " ":
			m.selected = m.options[m.cursor].Value
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit

		case "esc":
			// Select no as default and exit
			m.selected = false
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m AliasConfirmModel) View() string {
	if m.quitting && !m.confirmed {
		return ""
	}

	var b strings.Builder

	// Title - reuse mode selector style
	title := modeTitleStyle.Render(i18n.T("alias.prompt", nil))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Show the alias command
	aliasCmd := aliasCodeStyle.Render(shell.AliasLine)
	b.WriteString("  " + aliasCmd)
	b.WriteString("\n\n")

	// Options - reuse mode selector styles
	for i, opt := range m.options {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		var labelLine string
		var descLine string

		if i == m.cursor {
			labelLine = modeSelectedStyle.Render(fmt.Sprintf("%s%s", cursor, opt.Label))
			descLine = modeDescSelectedStyle.Render(opt.Description)
		} else {
			labelLine = modeOptionStyle.Render(fmt.Sprintf("%s%s", cursor, opt.Label))
			descLine = modeDescStyle.Render(opt.Description)
		}

		b.WriteString(labelLine)
		b.WriteString("\n")
		b.WriteString(descLine)
		b.WriteString("\n\n")
	}

	// Help - reuse mode selector style
	help := modeHelpStyle.Render("↑/↓: " + i18n.T("mode.help.move", nil) + " | Enter: " + i18n.T("mode.help.select", nil))
	b.WriteString(help)

	return modeBoxStyle.Render(b.String())
}

// GetSelected returns whether user selected yes
func (m AliasConfirmModel) GetSelected() bool {
	return m.selected
}

// IsConfirmed returns whether the user confirmed selection
func (m AliasConfirmModel) IsConfirmed() bool {
	return m.confirmed
}

// RunAliasConfirm launches the interactive alias confirmation
func RunAliasConfirm() (bool, bool, error) {
	model := NewAliasConfirmModel()
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return false, false, err
	}

	m := finalModel.(AliasConfirmModel)
	return m.GetSelected(), m.IsConfirmed(), nil
}
