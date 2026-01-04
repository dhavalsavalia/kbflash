package ui

import "github.com/charmbracelet/lipgloss"

// Standard ANSI colors - works with any terminal colorscheme
var (
	ColorFg        = lipgloss.AdaptiveColor{Light: "0", Dark: "15"}
	ColorGreen     = lipgloss.Color("2")
	ColorRed       = lipgloss.Color("1")
	ColorYellow    = lipgloss.Color("3")
	ColorCyan      = lipgloss.Color("6")
	ColorPurple    = lipgloss.Color("5")
	ColorDim       = lipgloss.Color("8")
	ColorBorder    = lipgloss.Color("8")
	ColorBorderAct = lipgloss.Color("5")
)

// Status indicators
const (
	StatusConnected    = "●"
	StatusDisconnected = "○"
	StatusWaiting      = "◐"
)

// Spinner frames (braille pattern)
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Base styles
var (
	BaseStyle = lipgloss.NewStyle()

	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorFg).
			Bold(true).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorDim).
			Padding(0, 1)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorderAct).
				Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPurple).
			Bold(true)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(ColorFg).
			Background(ColorPurple).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorCyan)

	AccentStyle = lipgloss.NewStyle().
			Foreground(ColorPurple)

	KeyHintStyle = lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true)
)

// Progress bar characters
const (
	ProgressFull  = "█"
	ProgressEmpty = "░"
)

// Tree characters
const (
	TreeBranch = "├"
	TreeLast   = "└"
	TreeLine   = "─"
)

// Box drawing
const (
	BoxTopLeft     = "╭"
	BoxTopRight    = "╮"
	BoxBottomLeft  = "╰"
	BoxBottomRight = "╯"
	BoxHorizontal  = "─"
	BoxVertical    = "│"
)

// RenderProgressBar renders a progress bar with the given percentage
func RenderProgressBar(percent int, width int) string {
	if width < 10 {
		width = 10
	}
	barWidth := width - 7 // Account for percentage text and padding
	filled := (percent * barWidth) / 100
	empty := barWidth - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += ProgressFull
	}
	for i := 0; i < empty; i++ {
		bar += ProgressEmpty
	}

	return lipgloss.JoinHorizontal(lipgloss.Center,
		bar,
		DimStyle.Render(" "+padLeft(percent, 3)+"%"),
	)
}

func padLeft(n, width int) string {
	s := ""
	num := n
	if num == 0 {
		s = "0"
	} else {
		for num > 0 {
			s = string(rune('0'+num%10)) + s
			num /= 10
		}
	}
	for len(s) < width {
		s = " " + s
	}
	return s
}
