package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HelpOverlay renders the help screen
type HelpOverlay struct {
	width    int
	height   int
	isSplit  bool
	hasBuild bool
}

// NewHelpOverlay creates a new help overlay
func NewHelpOverlay(isSplit, hasBuild bool) *HelpOverlay {
	return &HelpOverlay{
		isSplit:  isSplit,
		hasBuild: hasBuild,
	}
}

// SetSize sets overlay dimensions
func (h *HelpOverlay) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// View renders the help overlay
func (h *HelpOverlay) View() string {
	content := h.buildContent()

	boxWidth := 50
	if boxWidth > h.width-10 {
		boxWidth = h.width - 10
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPurple).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(content)

	boxHeight := lipgloss.Height(box)
	topPadding := (h.height - boxHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	leftPadding := (h.width - boxWidth - 4) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	var lines []string
	for i := 0; i < topPadding; i++ {
		lines = append(lines, "")
	}

	for _, line := range strings.Split(box, "\n") {
		lines = append(lines, strings.Repeat(" ", leftPadding)+line)
	}

	return strings.Join(lines, "\n")
}

func (h *HelpOverlay) buildContent() string {
	var lines []string

	title := TitleStyle.Render("KEYBINDINGS")
	lines = append(lines, title)
	lines = append(lines, "")

	// Navigation section
	lines = append(lines, AccentStyle.Render("Navigation"))
	lines = append(lines, DimStyle.Render(strings.Repeat("─", 40)))
	lines = append(lines, h.keyLine("↑ / k", "Move up"))
	lines = append(lines, h.keyLine("↓ / j", "Move down"))
	lines = append(lines, h.keyLine("Tab", "Switch panel"))
	lines = append(lines, h.keyLine("1 / 2 / 3", "Jump to panel"))
	lines = append(lines, "")

	// Actions section
	lines = append(lines, AccentStyle.Render("Actions"))
	lines = append(lines, DimStyle.Render(strings.Repeat("─", 40)))
	lines = append(lines, h.keyLine("Enter", "Select / Confirm"))
	if h.hasBuild {
		lines = append(lines, h.keyLine("b", "Build menu"))
	}
	lines = append(lines, h.keyLine("f", "Flash selected firmware"))
	if h.isSplit {
		lines = append(lines, h.keyLine("r", "Factory reset"))
	}
	lines = append(lines, "")

	// General section
	lines = append(lines, AccentStyle.Render("General"))
	lines = append(lines, DimStyle.Render(strings.Repeat("─", 40)))
	lines = append(lines, h.keyLine("?", "Toggle this help"))
	lines = append(lines, h.keyLine("Esc", "Cancel / Back"))
	lines = append(lines, h.keyLine("q", "Quit"))

	return strings.Join(lines, "\n")
}

func (h *HelpOverlay) keyLine(key, desc string) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(ColorCyan).
		Width(14)
	return keyStyle.Render(key) + desc
}
