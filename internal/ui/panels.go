package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dhavalsavalia/kbflash/internal/firmware"
)

// Panel identifiers
type Panel int

const (
	PanelFirmware Panel = iota
	PanelStatus
	PanelLog
)

func (p Panel) String() string {
	switch p {
	case PanelFirmware:
		return "Firmware"
	case PanelStatus:
		return "Status"
	case PanelLog:
		return "Log"
	default:
		return "Unknown"
	}
}

// LogEntry represents a log message
type LogEntry struct {
	Time    time.Time
	Message string
	Level   LogLevel
}

// LogLevel for log entries
type LogLevel int

const (
	LogInfo LogLevel = iota
	LogSuccess
	LogWarning
	LogError
)

// FirmwarePanel renders the firmware list
type FirmwarePanel struct {
	builds   []firmware.Build
	selected int
	height   int
	width    int
}

// NewFirmwarePanel creates a new firmware panel
func NewFirmwarePanel() *FirmwarePanel {
	return &FirmwarePanel{
		selected: 0,
	}
}

// SetBuilds updates the firmware builds list
func (p *FirmwarePanel) SetBuilds(builds []firmware.Build) {
	p.builds = builds
	if p.selected >= len(builds) {
		p.selected = len(builds) - 1
	}
	if p.selected < 0 {
		p.selected = 0
	}
}

// Selected returns the selected build
func (p *FirmwarePanel) Selected() *firmware.Build {
	if len(p.builds) == 0 {
		return nil
	}
	return &p.builds[p.selected]
}

// MoveUp moves selection up
func (p *FirmwarePanel) MoveUp() {
	if p.selected > 0 {
		p.selected--
	}
}

// MoveDown moves selection down
func (p *FirmwarePanel) MoveDown() {
	if p.selected < len(p.builds)-1 {
		p.selected++
	}
}

// SetSize sets the panel dimensions
func (p *FirmwarePanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// View renders the firmware panel content
func (p *FirmwarePanel) View() string {
	if len(p.builds) == 0 {
		return DimStyle.Render("  No firmware found")
	}

	var lines []string
	for i, build := range p.builds {
		prefix := "  "
		if i == p.selected {
			prefix = "> "
		}

		// Format date or show "flat" for flat structure
		dateStr := firmware.FormatDate(build.Date)
		if build.Date == "" {
			dateStr = "current"
		}

		// Status indicator - show file count
		status := ""
		if len(build.Files) > 0 {
			status = SuccessStyle.Render(fmt.Sprintf(" (%d)", len(build.Files)))
		}

		line := prefix + dateStr + status
		if i == p.selected {
			line = SelectedStyle.Render(line)
		}
		lines = append(lines, line)

		// Show files for selected build
		if i == p.selected {
			for j, f := range build.Files {
				treeChr := TreeBranch
				if j == len(build.Files)-1 {
					treeChr = TreeLast
				}
				size := firmware.FormatSize(f.Size)
				fileLine := fmt.Sprintf("  %s %s %s", treeChr, f.Name, DimStyle.Render(size))
				lines = append(lines, DimStyle.Render(fileLine))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// StatusPanel renders the status/operation display
type StatusPanel struct {
	width      int
	height     int
	isSplit    bool
	hasBuild   bool
	deviceName string
	sides      []string
}

// NewStatusPanel creates a new status panel
func NewStatusPanel(isSplit bool, hasBuild bool, deviceName string, sides []string) *StatusPanel {
	return &StatusPanel{
		isSplit:    isSplit,
		hasBuild:   hasBuild,
		deviceName: deviceName,
		sides:      sides,
	}
}

// SetSize sets the panel dimensions
func (p *StatusPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// ViewIdle renders idle state
func (p *StatusPanel) ViewIdle(build *firmware.Build) string {
	var lines []string

	boxWidth := p.width - 8
	if boxWidth < 20 {
		boxWidth = 20
	}

	lines = append(lines, "")
	lines = append(lines, centerText("SELECT FIRMWARE", boxWidth))
	lines = append(lines, "")
	lines = append(lines, centerText("Choose a build to flash", boxWidth))
	if p.hasBuild {
		lines = append(lines, centerText("or press B to build new", boxWidth))
	}
	lines = append(lines, "")

	if build != nil {
		lines = append(lines, "")
		dateStr := firmware.FormatDate(build.Date)
		if build.Date == "" {
			dateStr = "current"
		}
		lines = append(lines, DimStyle.Render("Selected: ")+dateStr)
	}

	return strings.Join(lines, "\n")
}

// ViewBuilding renders building state
func (p *StatusPanel) ViewBuilding(percent int, target string) string {
	var lines []string

	spinner := SpinnerFrames[(time.Now().UnixMilli()/100)%int64(len(SpinnerFrames))]

	title := "BUILDING " + strings.ToUpper(target)

	lines = append(lines, "")
	lines = append(lines, AccentStyle.Render(spinner+" "+title))
	lines = append(lines, "")
	lines = append(lines, RenderProgressBar(percent, p.width-10))
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// ViewWaitingDisconnect renders waiting for disconnect state (safety flow)
func (p *StatusPanel) ViewWaitingDisconnect(target string) string {
	var lines []string

	spinner := SpinnerFrames[(time.Now().UnixMilli()/100)%int64(len(SpinnerFrames))]

	lines = append(lines, "")
	lines = append(lines, centerText(WarningStyle.Render(spinner+" UNPLUG DEVICE"), p.width))
	lines = append(lines, "")
	lines = append(lines, centerText("To flash "+strings.ToUpper(target)+":", p.width))
	lines = append(lines, "")
	lines = append(lines, centerText("1. Unplug the device now", p.width))
	lines = append(lines, centerText("2. Connect the "+target+" half", p.width))
	lines = append(lines, centerText("3. Double-tap reset button", p.width))
	lines = append(lines, "")
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render(centerText("Waiting for disconnect...", p.width)))

	return strings.Join(lines, "\n")
}

// ViewWaiting renders waiting for device state
func (p *StatusPanel) ViewWaiting(target string) string {
	var lines []string

	spinner := SpinnerFrames[(time.Now().UnixMilli()/100)%int64(len(SpinnerFrames))]

	lines = append(lines, "")
	lines = append(lines, "")
	lines = append(lines, centerText(SuccessStyle.Render("✓ Disconnected"), p.width))
	lines = append(lines, "")
	lines = append(lines, centerText(WarningStyle.Render(spinner+" WAITING FOR "+strings.ToUpper(target)), p.width))
	lines = append(lines, "")
	lines = append(lines, centerText("Connect "+target+" half", p.width))
	lines = append(lines, centerText("Double-tap reset button", p.width))
	lines = append(lines, "")
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("Looking for "+p.deviceName+"..."))

	return strings.Join(lines, "\n")
}

// ViewFlashing renders flashing in progress
func (p *StatusPanel) ViewFlashing(percent int, filename, target string) string {
	var lines []string

	spinner := SpinnerFrames[(time.Now().UnixMilli()/100)%int64(len(SpinnerFrames))]

	lines = append(lines, "")
	lines = append(lines, AccentStyle.Render(spinner+" FLASHING "+strings.ToUpper(target)))
	lines = append(lines, "")
	lines = append(lines, RenderProgressBar(percent, p.width-10))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Copying: %s", filename))
	lines = append(lines, "")

	// Flash checklist for split keyboards
	if p.isSplit && len(p.sides) >= 2 {
		lines = append(lines, "")
		for i, side := range p.sides {
			icon := "[ ]"
			style := DimStyle
			if strings.EqualFold(target, side) {
				icon = "[>]"
				style = AccentStyle
			} else if i == 0 && !strings.EqualFold(target, side) {
				// First side already done if we're on a later side
				for j, s := range p.sides {
					if j > i && strings.EqualFold(target, s) {
						icon = "[x]"
						style = SuccessStyle
						break
					}
				}
			}
			lines = append(lines, style.Render(icon+" Flash "+side))
		}
	}

	return strings.Join(lines, "\n")
}

// ViewComplete renders completion summary
func (p *StatusPanel) ViewComplete(duration time.Duration, steps []string) string {
	var lines []string

	lines = append(lines, "")
	lines = append(lines, SuccessStyle.Render("FLASH COMPLETE"))
	lines = append(lines, "")

	for _, step := range steps {
		lines = append(lines, SuccessStyle.Render("  [x] "+step))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Duration: %s", duration.Round(time.Second)))
	lines = append(lines, "")
	if p.isSplit {
		lines = append(lines, DimStyle.Render("Test both halves to verify."))
	} else {
		lines = append(lines, DimStyle.Render("Test keyboard to verify."))
	}

	return strings.Join(lines, "\n")
}

// LogPanel renders the log output
type LogPanel struct {
	entries []LogEntry
	width   int
	height  int
}

// NewLogPanel creates a new log panel
func NewLogPanel() *LogPanel {
	return &LogPanel{}
}

// Add adds a log entry
func (p *LogPanel) Add(level LogLevel, msg string) {
	p.entries = append(p.entries, LogEntry{
		Time:    time.Now(),
		Message: msg,
		Level:   level,
	})
	// Keep last N entries
	maxEntries := 50
	if len(p.entries) > maxEntries {
		p.entries = p.entries[len(p.entries)-maxEntries:]
	}
}

// Clear clears all entries
func (p *LogPanel) Clear() {
	p.entries = nil
}

// SetSize sets the panel dimensions
func (p *LogPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// View renders the log panel content
func (p *LogPanel) View() string {
	if len(p.entries) == 0 {
		return DimStyle.Render("  No log entries")
	}

	maxVisible := p.height - 2
	if maxVisible < 1 {
		maxVisible = 10
	}

	start := 0
	if len(p.entries) > maxVisible {
		start = len(p.entries) - maxVisible
	}

	var lines []string
	for _, entry := range p.entries[start:] {
		timestamp := DimStyle.Render(entry.Time.Format("15:04:05"))

		var msgStyle lipgloss.Style
		switch entry.Level {
		case LogSuccess:
			msgStyle = SuccessStyle
		case LogWarning:
			msgStyle = WarningStyle
		case LogError:
			msgStyle = ErrorStyle
		default:
			msgStyle = lipgloss.NewStyle().Foreground(ColorFg)
		}

		// Truncate message if needed
		msg := entry.Message
		maxMsgLen := p.width - 12
		if maxMsgLen > 0 && len(msg) > maxMsgLen {
			msg = msg[:maxMsgLen-3] + "..."
		}

		lines = append(lines, timestamp+"  "+msgStyle.Render(msg))
	}

	return strings.Join(lines, "\n")
}

// Helper functions
func centerText(text string, width int) string {
	textLen := lipgloss.Width(text)
	if textLen >= width {
		return text
	}
	padding := (width - textLen) / 2
	return strings.Repeat(" ", padding) + text
}
