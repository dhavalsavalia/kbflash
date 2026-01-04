package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DialogOption represents a dialog button
type DialogOption int

const (
	DialogConfirm DialogOption = iota
	DialogCancel
)

// ConfirmDialog renders a confirmation dialog
type ConfirmDialog struct {
	title    string
	message  []string
	selected DialogOption
	width    int
	height   int
}

// NewConfirmDialog creates a new confirmation dialog
func NewConfirmDialog(title string, message []string) *ConfirmDialog {
	return &ConfirmDialog{
		title:    title,
		message:  message,
		selected: DialogCancel, // Default to cancel for safety
	}
}

// SetSize sets dialog dimensions
func (d *ConfirmDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// MoveLeft moves selection left (to confirm)
func (d *ConfirmDialog) MoveLeft() {
	d.selected = DialogConfirm
}

// MoveRight moves selection right (to cancel)
func (d *ConfirmDialog) MoveRight() {
	d.selected = DialogCancel
}

// Selected returns the selected option
func (d *ConfirmDialog) Selected() DialogOption {
	return d.selected
}

// View renders the dialog
func (d *ConfirmDialog) View() string {
	var lines []string

	title := WarningStyle.Render("⚠  " + d.title)
	lines = append(lines, title)
	lines = append(lines, "")

	for _, msg := range d.message {
		lines = append(lines, msg)
	}
	lines = append(lines, "")

	confirmStyle := lipgloss.NewStyle().Padding(0, 2)
	cancelStyle := lipgloss.NewStyle().Padding(0, 2)

	if d.selected == DialogConfirm {
		confirmStyle = confirmStyle.Background(ColorPurple).Foreground(lipgloss.Color("0"))
	}
	if d.selected == DialogCancel {
		cancelStyle = cancelStyle.Background(ColorPurple).Foreground(lipgloss.Color("0"))
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		confirmStyle.Render("Yes, proceed"),
		"  ",
		cancelStyle.Render("Cancel"),
	)
	lines = append(lines, buttons)

	content := strings.Join(lines, "\n")

	boxWidth := 40
	if boxWidth > d.width-10 {
		boxWidth = d.width - 10
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorYellow).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(content)

	boxHeight := lipgloss.Height(box)
	topPadding := (d.height - boxHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	leftPadding := (d.width - boxWidth - 4) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	var result []string
	for i := 0; i < topPadding; i++ {
		result = append(result, "")
	}

	for _, line := range strings.Split(box, "\n") {
		result = append(result, strings.Repeat(" ", leftPadding)+line)
	}

	return strings.Join(result, "\n")
}

// FactoryResetDialog creates the factory reset confirmation dialog
func FactoryResetDialog() *ConfirmDialog {
	return NewConfirmDialog("FACTORY RESET", []string{
		"This will:",
		"  Clear all Bluetooth bonds",
		"  Reset keyboard settings",
		"  Require re-pairing",
		"",
		"Have you unpaired from all",
		"Bluetooth devices?",
	})
}

// BuildMenuDialog renders the build target selection menu
type BuildMenuDialog struct {
	width   int
	height  int
	targets []string // configured build targets (sides)
}

// NewBuildMenuDialog creates a new build menu dialog
func NewBuildMenuDialog(targets []string) *BuildMenuDialog {
	return &BuildMenuDialog{
		targets: targets,
	}
}

// SetSize sets dialog dimensions
func (d *BuildMenuDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View renders the build menu
func (d *BuildMenuDialog) View() string {
	var lines []string

	title := AccentStyle.Render("BUILD FIRMWARE")
	lines = append(lines, title)
	lines = append(lines, "")

	// Build options based on configured targets
	if len(d.targets) > 1 {
		lines = append(lines, "  "+KeyHintStyle.Render("[a]")+" All targets")
	}

	for i, target := range d.targets {
		key := string(rune('1' + i))
		if i < 9 {
			lines = append(lines, "  "+KeyHintStyle.Render("["+key+"]")+" "+target)
		}
	}

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  [esc] Cancel"))

	content := strings.Join(lines, "\n")

	boxWidth := 28
	if boxWidth > d.width-10 {
		boxWidth = d.width - 10
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPurple).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(content)

	boxHeight := lipgloss.Height(box)
	topPadding := (d.height - boxHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	leftPadding := (d.width - boxWidth - 4) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	var result []string
	for i := 0; i < topPadding; i++ {
		result = append(result, "")
	}

	for _, line := range strings.Split(box, "\n") {
		result = append(result, strings.Repeat(" ", leftPadding)+line)
	}

	return strings.Join(result, "\n")
}

// Targets returns the configured build targets
func (d *BuildMenuDialog) Targets() []string {
	return d.targets
}
