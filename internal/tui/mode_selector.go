package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/egoavara/codex-market/internal/config"
	"github.com/egoavara/codex-market/internal/i18n"
)

// ModeOption represents an auto-update mode option
type ModeOption struct {
	Mode        config.AutoUpdateMode
	Label       string
	Description string
}

// Animation tick message
type animTickMsg time.Time

// ModeSelectorModel is the bubbletea model for mode selection
type ModeSelectorModel struct {
	options      []ModeOption
	cursor       int
	selected     config.AutoUpdateMode
	width        int
	height       int
	quitting     bool
	confirmed    bool
	animFrame    int       // Current animation frame
	lastMode     int       // Track mode changes to reset animation
}

// Mode selector styles
var (
	modeTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	modeOptionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	modeSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57")).
				Bold(true).
				Padding(0, 1)

	modeDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			MarginLeft(4)

	modeDescSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				MarginLeft(4)

	modeBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	modeHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	// Preview styles
	previewBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Foreground(lipgloss.Color("250"))

	previewTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Bold(true)

	previewPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	previewHighlightStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))
)

// NewModeSelectorModel creates a new mode selector model
func NewModeSelectorModel() ModeSelectorModel {
	options := []ModeOption{
		{
			Mode:        config.AutoUpdateModeNotify,
			Label:       i18n.T("mode.notify.label", nil),
			Description: i18n.T("mode.notify.desc", nil),
		},
		{
			Mode:        config.AutoUpdateModeAuto,
			Label:       i18n.T("mode.auto.label", nil),
			Description: i18n.T("mode.auto.desc", nil),
		},
	}

	return ModeSelectorModel{
		options:  options,
		cursor:   0, // Default to notify
		selected: config.AutoUpdateModeNotify,
	}
}

func (m ModeSelectorModel) Init() tea.Cmd {
	return tickAnimation()
}

func tickAnimation() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

func (m ModeSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case animTickMsg:
		// Increment animation frame
		m.animFrame++
		return m, tickAnimation()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.animFrame = 0 // Reset animation on mode change
			}

		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
				m.animFrame = 0 // Reset animation on mode change
			}

		case "enter", " ":
			m.selected = m.options[m.cursor].Mode
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit

		case "esc":
			// Select notify as default and exit
			m.selected = config.AutoUpdateModeNotify
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m ModeSelectorModel) View() string {
	if m.quitting && !m.confirmed {
		return ""
	}

	// Left side: Options
	var left strings.Builder

	// Title
	title := modeTitleStyle.Render(i18n.T("mode.title", nil))
	left.WriteString(title)
	left.WriteString("\n\n")

	// Options
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

		left.WriteString(labelLine)
		left.WriteString("\n")
		left.WriteString(descLine)
		left.WriteString("\n\n")
	}

	// Help
	help := modeHelpStyle.Render("↑/↓: " + i18n.T("mode.help.move", nil) + " | Enter: " + i18n.T("mode.help.select", nil))
	left.WriteString(help)

	// Right side: Preview with animation
	preview := m.renderAnimatedPreview()

	// Create boxes
	leftBox := modeBoxStyle.Render(left.String())

	// Fixed height preview box (14 lines content)
	rightBoxStyle := previewBoxStyle.Width(48).Height(14)
	rightBox := rightBoxStyle.Render(preview)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "  ", rightBox)
}

// Codex UI that appears at the end
const codexUI = `┌─────────────────────────────────────────┐
│ >_ Codex CLI (v0.1.0)                   │
│                                         │
│ model: claude-sonnet-4-0520             │
│ dir:   ~/my-project                     │
└─────────────────────────────────────────┘
> _`

// Animation frames for notify mode (realistic flow)
var notifyFrames = []string{
	// Typing animation
	"$ c_",
	"$ co_",
	"$ cod_",
	"$ code_",
	"$ codex_",
	"$ codex",
	// Checking
	"$ codex\n\nChecking for updates...",
	"$ codex\n\nChecking for updates...",
	// Update info appears
	"$ codex\n\nChecking for updates...\n\nUpdates available:\n  [Marketplace] my-market abc1234 → def5678\n  [Plugin] my-plugin v1.0 → v1.1\n\nUpdate now? [Y/n] _",
	"$ codex\n\nChecking for updates...\n\nUpdates available:\n  [Marketplace] my-market abc1234 → def5678\n  [Plugin] my-plugin v1.0 → v1.1\n\nUpdate now? [Y/n] _",
	// User types Y
	"$ codex\n\nChecking for updates...\n\nUpdates available:\n  [Marketplace] my-market abc1234 → def5678\n  [Plugin] my-plugin v1.0 → v1.1\n\nUpdate now? [Y/n] Y",
	// Updating (longer)
	"$ codex\n\nChecking for updates...\n\nUpdates available:\n  [Marketplace] my-market abc1234 → def5678\n  [Plugin] my-plugin v1.0 → v1.1\n\nUpdate now? [Y/n] Y\n\nUpdating...",
	"$ codex\n\nChecking for updates...\n\nUpdates available:\n  [Marketplace] my-market abc1234 → def5678\n  [Plugin] my-plugin v1.0 → v1.1\n\nUpdate now? [Y/n] Y\n\nUpdating...\n  ⠋ my-market",
	"$ codex\n\nChecking for updates...\n\nUpdates available:\n  [Marketplace] my-market abc1234 → def5678\n  [Plugin] my-plugin v1.0 → v1.1\n\nUpdate now? [Y/n] Y\n\nUpdating...\n  ⠙ my-market",
	"$ codex\n\nChecking for updates...\n\nUpdates available:\n  [Marketplace] my-market abc1234 → def5678\n  [Plugin] my-plugin v1.0 → v1.1\n\nUpdate now? [Y/n] Y\n\nUpdating...\n  ✓ my-market\n  ⠋ my-plugin",
	"$ codex\n\nChecking for updates...\n\nUpdates available:\n  [Marketplace] my-market abc1234 → def5678\n  [Plugin] my-plugin v1.0 → v1.1\n\nUpdate now? [Y/n] Y\n\nUpdating...\n  ✓ my-market\n  ⠙ my-plugin",
	"$ codex\n\nChecking for updates...\n\nUpdates available:\n  [Marketplace] my-market abc1234 → def5678\n  [Plugin] my-plugin v1.0 → v1.1\n\nUpdate now? [Y/n] Y\n\nUpdating...\n  ✓ my-market\n  ✓ my-plugin",
	// Codex starts (screen cleared)
	codexUI,
	codexUI,
	codexUI,
}

// Animation frames for auto mode (realistic flow)
var autoFrames = []string{
	// Typing animation
	"$ c_",
	"$ co_",
	"$ cod_",
	"$ code_",
	"$ codex_",
	"$ codex",
	// Checking
	"$ codex\n\nChecking for updates...",
	"$ codex\n\nChecking for updates...",
	// Auto updating (longer)
	"$ codex\n\nChecking for updates...\n\nUpdating...",
	"$ codex\n\nChecking for updates...\n\nUpdating...\n  ⠋ my-market",
	"$ codex\n\nChecking for updates...\n\nUpdating...\n  ⠙ my-market",
	"$ codex\n\nChecking for updates...\n\nUpdating...\n  ✓ my-market\n  ⠋ my-plugin",
	"$ codex\n\nChecking for updates...\n\nUpdating...\n  ✓ my-market\n  ⠙ my-plugin",
	"$ codex\n\nChecking for updates...\n\nUpdating...\n  ✓ my-market\n  ✓ my-plugin",
	// Codex starts (screen cleared)
	codexUI,
	codexUI,
	codexUI,
}

// renderAnimatedPreview returns the animated preview for current mode
func (m ModeSelectorModel) renderAnimatedPreview() string {
	var b strings.Builder

	b.WriteString(previewTitleStyle.Render("Preview:"))
	b.WriteString("\n\n")

	currentMode := m.options[m.cursor].Mode
	var frames []string

	switch currentMode {
	case config.AutoUpdateModeNotify:
		frames = notifyFrames
	case config.AutoUpdateModeAuto:
		frames = autoFrames
	default:
		return b.String()
	}

	// Get current frame (loop animation with pause at end)
	totalFrames := len(frames) + 2 // +2 for pause at end before restart
	frameIdx := m.animFrame % totalFrames
	if frameIdx >= len(frames) {
		frameIdx = len(frames) - 1 // Stay on last frame during pause
	}

	// Apply styling to the frame
	frame := frames[frameIdx]
	styledFrame := m.stylePreviewFrame(frame)
	b.WriteString(styledFrame)

	return b.String()
}

// stylePreviewFrame applies colors to preview text
func (m ModeSelectorModel) stylePreviewFrame(frame string) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	cyanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51"))

	result := frame

	// Typing cursor
	result = strings.ReplaceAll(result, "$ c_", dimStyle.Render("$ ")+"c"+greenStyle.Render("_"))
	result = strings.ReplaceAll(result, "$ co_", dimStyle.Render("$ ")+"co"+greenStyle.Render("_"))
	result = strings.ReplaceAll(result, "$ cod_", dimStyle.Render("$ ")+"cod"+greenStyle.Render("_"))
	result = strings.ReplaceAll(result, "$ code_", dimStyle.Render("$ ")+"code"+greenStyle.Render("_"))
	result = strings.ReplaceAll(result, "$ codex_", dimStyle.Render("$ ")+"codex"+greenStyle.Render("_"))
	result = strings.ReplaceAll(result, "$ codex", dimStyle.Render("$ ")+"codex")

	// Status messages
	result = strings.ReplaceAll(result, "Checking for updates...", yellowStyle.Render("Checking for updates..."))
	result = strings.ReplaceAll(result, "Updating...", yellowStyle.Render("Updating..."))

	// Version info
	result = strings.ReplaceAll(result, "abc1234", dimStyle.Render("abc1234"))
	result = strings.ReplaceAll(result, "def5678", greenStyle.Render("def5678"))
	result = strings.ReplaceAll(result, "v1.0", dimStyle.Render("v1.0"))
	result = strings.ReplaceAll(result, "v1.1", greenStyle.Render("v1.1"))

	// Spinners and checkmarks
	result = strings.ReplaceAll(result, "⠋", yellowStyle.Render("⠋"))
	result = strings.ReplaceAll(result, "⠙", yellowStyle.Render("⠙"))
	result = strings.ReplaceAll(result, "✓", greenStyle.Render("✓"))

	// Prompt
	result = strings.ReplaceAll(result, "[Y/n]", greenStyle.Render("[Y/n]"))

	// Codex UI styling
	result = strings.ReplaceAll(result, ">_ Codex CLI", cyanStyle.Render(">_ Codex CLI"))
	result = strings.ReplaceAll(result, "claude-sonnet-4-0520", cyanStyle.Render("claude-sonnet-4-0520"))
	result = strings.ReplaceAll(result, "> _", greenStyle.Render("> _"))

	return result
}

// GetSelected returns the selected mode
func (m ModeSelectorModel) GetSelected() config.AutoUpdateMode {
	return m.selected
}

// IsConfirmed returns whether the user confirmed selection
func (m ModeSelectorModel) IsConfirmed() bool {
	return m.confirmed
}

// RunModeSelector launches the interactive mode selector
func RunModeSelector() (config.AutoUpdateMode, bool, error) {
	model := NewModeSelectorModel()
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return config.AutoUpdateModeNotify, false, err
	}

	m := finalModel.(ModeSelectorModel)
	return m.GetSelected(), m.IsConfirmed(), nil
}
