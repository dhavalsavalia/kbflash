package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dhavalsavalia/kbflash/internal/config"
	"github.com/dhavalsavalia/kbflash/internal/device"
	"github.com/dhavalsavalia/kbflash/internal/firmware"
)

// AppState represents the application state
type AppState int

const (
	StateIdle AppState = iota
	StateBuilding
	StateWaitingDisconnect // Safety: wait for user to unplug device
	StateWaitingDevice
	StateFlashing
	StateComplete
)

// DeviceStatus represents the device connection state
type DeviceStatus int

const (
	DeviceDisconnected DeviceStatus = iota
	DeviceConnected
	DeviceWaiting
)

// Model is the main bubbletea model
type Model struct {
	// Dimensions
	width  int
	height int

	// State
	state        AppState
	activePanel  Panel
	showHelp     bool
	showDialog   bool
	deviceStatus DeviceStatus
	devicePath   string

	// Panels
	firmwarePanel *FirmwarePanel
	statusPanel   *StatusPanel
	logPanel      *LogPanel

	// Overlays
	helpOverlay     *HelpOverlay
	confirmDialog   *ConfirmDialog
	buildMenuDialog *BuildMenuDialog
	showBuildMenu   bool

	// Config-driven components
	cfg      *config.Config
	scanner  *firmware.Scanner
	detector device.Detector
	builder  firmware.FirmwareBuilder
	flasher  *firmware.Flasher

	// Detection context and channel
	detectCtx    context.Context
	detectCancel context.CancelFunc
	detectEvents <-chan device.Event

	// Build progress channel
	buildProgress chan firmware.BuildProgress

	// Operation state
	buildPercent   int
	buildTarget    string
	flashPercent   int
	flashTarget    string // current side being flashed
	flashIndex     int    // index in sides array
	startTime      time.Time
	completedSteps []string
}

// NewModel creates a new model from config
func NewModel(cfg *config.Config) *Model {
	isSplit := cfg.Keyboard.Type == "split"
	sides := cfg.Keyboard.Sides
	if len(sides) == 0 {
		if isSplit {
			sides = []string{"left", "right"}
		} else {
			sides = []string{"main"}
		}
	}

	m := &Model{
		cfg:             cfg,
		state:           StateIdle,
		activePanel:     PanelFirmware,
		deviceStatus:    DeviceDisconnected,
		firmwarePanel:   NewFirmwarePanel(),
		statusPanel:     NewStatusPanel(isSplit, cfg.Build.Enabled, cfg.Device.Name, sides),
		logPanel:        NewLogPanel(),
		helpOverlay:     NewHelpOverlay(isSplit, cfg.Build.Enabled),
		buildMenuDialog: NewBuildMenuDialog(sides),
		scanner:         firmware.NewScanner(cfg.Build.FirmwareDir, cfg.Build.FilePattern),
		detector:        device.New(),
		flasher:         firmware.NewFlasher(),
	}

	if cfg.Build.Enabled {
		if cfg.Build.Mode == "docker" {
			m.builder = firmware.NewDockerBuilder(
				cfg.Build.Image,
				cfg.Build.Board,
				cfg.Build.Shield,
				cfg.Build.WorkingDir,
				cfg.Build.FirmwareDir,
			)
		} else {
			m.builder = firmware.NewBuilder(cfg.Build.Command, cfg.Build.Args, cfg.Build.WorkingDir)
		}
	}

	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	m.logPanel.Add(LogInfo, "Started - "+m.cfg.Keyboard.Name)

	// Scan for firmware
	ctx := context.Background()
	builds, err := m.scanner.Scan(ctx)
	if err != nil {
		m.logPanel.Add(LogError, "Scan failed: "+err.Error())
	} else {
		m.firmwarePanel.SetBuilds(builds)
		m.logPanel.Add(LogInfo, "Found "+formatInt(len(builds))+" build(s)")
	}

	// Start device detection
	return m.startDetection()
}

// startDetection starts the device detection loop
func (m *Model) startDetection() tea.Cmd {
	// Cancel any existing detection
	if m.detectCancel != nil {
		m.detectCancel()
	}

	m.detectCtx, m.detectCancel = context.WithCancel(context.Background())
	pollInterval := time.Duration(m.cfg.Device.PollInterval)
	m.detectEvents = m.detector.Detect(m.detectCtx, m.cfg.Device.Name, pollInterval)

	return m.listenForNextEvent()
}

// deviceEventMsg wraps device events
type deviceEventMsg struct {
	event device.Event
}

// tickMsg for spinner animation
type tickMsg struct{}

// buildProgressMsg for build progress updates
type buildProgressMsg struct {
	progress firmware.BuildProgress
}

// buildCompleteMsg for build completion
type buildCompleteMsg struct {
	result firmware.BuildResult
}

// flashCompleteMsg for flash completion
type flashCompleteMsg struct {
	result firmware.FlashResult
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updatePanelSizes()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case deviceEventMsg:
		if msg.event.Connected {
			m.deviceStatus = DeviceConnected
			m.devicePath = msg.event.Path
			m.logPanel.Add(LogSuccess, "Device connected")
			if m.state == StateWaitingDevice {
				return m.startFlash()
			}
		} else {
			m.deviceStatus = DeviceDisconnected
			m.devicePath = ""
			m.logPanel.Add(LogInfo, "Device disconnected")
			// Safety: if waiting for disconnect, transition to waiting for connect
			if m.state == StateWaitingDisconnect {
				m.state = StateWaitingDevice
				m.logPanel.Add(LogInfo, "Now connect "+m.flashTarget+" half...")
			}
		}
		// Continue listening for events
		return m, m.listenForNextEvent()

	case buildProgressMsg:
		m.buildPercent = msg.progress.Percent
		// Continue listening for more progress
		return m, m.listenForBuildProgress()

	case buildCompleteMsg:
		if msg.result.Success {
			m.logPanel.Add(LogSuccess, "Build complete")
			m.buildPercent = 100
			// Refresh firmware list
			ctx := context.Background()
			builds, _ := m.scanner.Scan(ctx)
			m.firmwarePanel.SetBuilds(builds)
			m.state = StateIdle
		} else {
			m.logPanel.Add(LogError, "Build failed: "+msg.result.Error.Error())
			m.state = StateIdle
		}
		return m, nil

	case flashCompleteMsg:
		if msg.result.Success {
			m.logPanel.Add(LogSuccess, m.flashTarget+" flashed")
			m.completedSteps = append(m.completedSteps, m.flashTarget+" flashed")

			// Check if we need to flash more sides
			sides := m.cfg.Keyboard.Sides
			if len(sides) == 0 {
				sides = []string{"main"}
			}

			m.flashIndex++
			if m.flashIndex < len(sides) {
				// Safety: require disconnect before flashing next side
				m.flashTarget = sides[m.flashIndex]
				m.state = StateWaitingDisconnect
				m.logPanel.Add(LogWarning, "Unplug device, then connect "+m.flashTarget)
				return m, nil
			}

			// All done
			m.state = StateComplete
			m.logPanel.Add(LogSuccess, "Flash complete")
		} else {
			m.logPanel.Add(LogError, "Flash failed: "+msg.result.Error.Error())
			m.state = StateIdle
		}
		return m, nil

	case tickMsg:
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg{}
		})
	}

	return m, nil
}

// listenForNextEvent continues listening on the existing device channel
func (m *Model) listenForNextEvent() tea.Cmd {
	events := m.detectEvents
	return func() tea.Msg {
		if events == nil {
			return nil
		}
		for event := range events {
			return deviceEventMsg{event: event}
		}
		return nil
	}
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c":
		if m.detectCancel != nil {
			m.detectCancel()
		}
		return m, tea.Quit
	case "q":
		if !m.showDialog && !m.showBuildMenu && (m.state == StateIdle || m.state == StateComplete) {
			if m.detectCancel != nil {
				m.detectCancel()
			}
			return m, tea.Quit
		}
	case "?":
		if m.state == StateIdle {
			m.showHelp = !m.showHelp
		}
		return m, nil
	case "esc":
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		if m.showBuildMenu {
			m.showBuildMenu = false
			return m, nil
		}
		if m.showDialog {
			m.showDialog = false
			m.confirmDialog = nil
			return m, nil
		}
		if m.state == StateWaitingDisconnect || m.state == StateWaitingDevice {
			m.state = StateIdle
			m.logPanel.Add(LogInfo, "Cancelled")
			return m, nil
		}
		if m.state == StateComplete {
			m.state = StateIdle
			m.completedSteps = nil
			return m, nil
		}
	}

	// Build menu keys
	if m.showBuildMenu {
		return m.handleBuildMenuKey(msg)
	}

	// Dialog keys
	if m.showDialog && m.confirmDialog != nil {
		switch msg.String() {
		case "left", "h":
			m.confirmDialog.MoveLeft()
		case "right", "l":
			m.confirmDialog.MoveRight()
		case "enter":
			if m.confirmDialog.Selected() == DialogConfirm {
				m.showDialog = false
				return m.startFactoryReset()
			}
			m.showDialog = false
			m.confirmDialog = nil
		}
		return m, nil
	}

	// Help overlay blocks other keys
	if m.showHelp {
		return m, nil
	}

	// State-specific keys
	switch m.state {
	case StateIdle:
		return m.handleIdleKey(msg)
	case StateComplete:
		if msg.String() == "enter" {
			m.state = StateIdle
			m.completedSteps = nil
		}
	}

	return m, nil
}

func (m *Model) handleBuildMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	targets := m.buildMenuDialog.Targets()

	switch msg.String() {
	case "a":
		if len(targets) > 1 {
			m.showBuildMenu = false
			return m.startBuild("all")
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '1')
		if idx < len(targets) {
			m.showBuildMenu = false
			return m.startBuild(targets[idx])
		}
	case "esc":
		m.showBuildMenu = false
	}
	return m, nil
}

func (m *Model) handleIdleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Navigation
	case "up", "k":
		if m.activePanel == PanelFirmware {
			m.firmwarePanel.MoveUp()
		}
	case "down", "j":
		if m.activePanel == PanelFirmware {
			m.firmwarePanel.MoveDown()
		}
	case "tab":
		m.activePanel = (m.activePanel + 1) % 3
	case "1":
		m.activePanel = PanelFirmware
	case "2":
		m.activePanel = PanelStatus
	case "3":
		m.activePanel = PanelLog

	// Actions
	case "b":
		if m.cfg.Build.Enabled {
			m.showBuildMenu = true
			m.buildMenuDialog.SetSize(m.width, m.height)
		}
		return m, nil
	case "f", "enter":
		if m.firmwarePanel.Selected() != nil {
			return m.prepareFlash()
		}
	case "r":
		// Factory reset only for split keyboards
		if m.cfg.Keyboard.Type == "split" {
			m.confirmDialog = FactoryResetDialog()
			m.confirmDialog.SetSize(m.width, m.height)
			m.showDialog = true
		}
	}

	return m, nil
}

func (m *Model) startBuild(target string) (tea.Model, tea.Cmd) {
	if m.builder == nil {
		m.logPanel.Add(LogError, "Build not enabled in config")
		return m, nil
	}

	// For Docker mode, check Docker is available first
	if m.cfg.Build.Mode == "docker" {
		ctx := context.Background()
		if err := firmware.CheckDocker(ctx); err != nil {
			m.logPanel.Add(LogError, err.Error())
			return m, nil
		}
	}

	m.state = StateBuilding
	m.buildPercent = 0
	m.buildTarget = target
	m.startTime = time.Now()
	m.logPanel.Add(LogInfo, "Building: "+target)

	// Create progress channel
	m.buildProgress = make(chan firmware.BuildProgress, 10)

	ctx := context.Background()
	return m, tea.Batch(
		func() tea.Msg {
			// For Docker mode, ensure image is pulled first
			if dockerBuilder, ok := m.builder.(*firmware.DockerBuilder); ok {
				if err := dockerBuilder.EnsureImage(ctx, func(msg string) {
					select {
					case m.buildProgress <- firmware.BuildProgress{Percent: 0, Message: msg}:
					default:
					}
				}); err != nil {
					return buildCompleteMsg{result: firmware.BuildResult{Success: false, Error: err}}
				}
			}

			result := m.builder.Build(ctx, target, func(p firmware.BuildProgress) {
				// Send progress to channel (non-blocking)
				select {
				case m.buildProgress <- p:
				default:
				}
			})
			close(m.buildProgress)
			return buildCompleteMsg{result: result}
		},
		m.listenForBuildProgress(),
		tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg{}
		}),
	)
}

// listenForBuildProgress listens for build progress updates
func (m *Model) listenForBuildProgress() tea.Cmd {
	return func() tea.Msg {
		if m.buildProgress == nil {
			return nil
		}
		progress, ok := <-m.buildProgress
		if !ok {
			return nil
		}
		return buildProgressMsg{progress: progress}
	}
}

func (m *Model) prepareFlash() (tea.Model, tea.Cmd) {
	build := m.firmwarePanel.Selected()
	if build == nil || len(build.Files) == 0 {
		m.logPanel.Add(LogError, "No firmware files found")
		return m, nil
	}

	m.completedSteps = nil
	m.flashIndex = 0

	sides := m.cfg.Keyboard.Sides
	if len(sides) == 0 {
		sides = []string{"main"}
	}

	m.flashTarget = sides[0]
	m.startTime = time.Now()

	// Safety: always require disconnect-reconnect cycle to prevent flashing wrong side
	targetName := m.flashTarget
	if m.cfg.Keyboard.Type != "split" {
		targetName = "keyboard"
	}

	if m.deviceStatus == DeviceConnected {
		// Device is connected - require disconnect first
		m.state = StateWaitingDisconnect
		m.logPanel.Add(LogWarning, "Unplug device, then connect "+targetName)
	} else {
		// Device already disconnected - wait for correct side to connect
		m.state = StateWaitingDevice
		m.logPanel.Add(LogInfo, "Connect "+targetName+" and double-tap reset...")
	}

	return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m *Model) startFlash() (tea.Model, tea.Cmd) {
	build := m.firmwarePanel.Selected()
	if build == nil {
		return m, nil
	}

	m.state = StateFlashing
	m.flashPercent = 0
	m.logPanel.Add(LogInfo, "Flashing "+m.flashTarget)

	// Find the firmware file for this target
	var filePath string
	target := strings.ToLower(m.flashTarget)
	for _, f := range build.Files {
		fname := strings.ToLower(f.Name)
		if strings.Contains(fname, target) {
			filePath = f.Path
			break
		}
	}

	// If no target-specific file found and only one file, use it
	if filePath == "" && len(build.Files) == 1 {
		filePath = build.Files[0].Path
	}

	if filePath == "" {
		m.logPanel.Add(LogError, "No firmware file for "+m.flashTarget)
		m.state = StateIdle
		return m, nil
	}

	ctx := context.Background()
	return m, tea.Batch(
		func() tea.Msg {
			result := m.flasher.Flash(ctx, filePath, m.devicePath)
			return flashCompleteMsg{result: result}
		},
		tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tickMsg{}
		}),
	)
}

func (m *Model) startFactoryReset() (tea.Model, tea.Cmd) {
	build := m.firmwarePanel.Selected()
	if build == nil {
		return m, nil
	}

	// Find reset firmware
	var resetPath string
	for _, f := range build.Files {
		fname := strings.ToLower(f.Name)
		if strings.Contains(fname, "reset") || strings.Contains(fname, "settings") {
			resetPath = f.Path
			break
		}
	}

	if resetPath == "" {
		m.logPanel.Add(LogError, "No reset firmware found")
		return m, nil
	}

	m.completedSteps = nil
	m.flashIndex = 0
	sides := m.cfg.Keyboard.Sides
	if len(sides) == 0 {
		sides = []string{"left", "right"}
	}
	m.flashTarget = sides[0] + " (reset)"
	m.startTime = time.Now()
	m.logPanel.Add(LogWarning, "Factory reset started")

	if m.deviceStatus == DeviceConnected {
		m.state = StateFlashing
		ctx := context.Background()
		return m, m.flashReset(ctx, resetPath)
	}

	m.state = StateWaitingDevice
	return m, nil
}

func (m *Model) flashReset(ctx context.Context, resetPath string) tea.Cmd {
	return func() tea.Msg {
		result := m.flasher.Flash(ctx, resetPath, m.devicePath)
		return flashCompleteMsg{result: result}
	}
}

func (m *Model) updatePanelSizes() {
	contentHeight := m.height - 4

	leftWidth := m.width * 30 / 100
	centerWidth := m.width * 40 / 100
	rightWidth := m.width - leftWidth - centerWidth - 6

	m.firmwarePanel.SetSize(leftWidth, contentHeight)
	m.statusPanel.SetSize(centerWidth, contentHeight)
	m.logPanel.SetSize(rightWidth, contentHeight)
	m.helpOverlay.SetSize(m.width, m.height)
	if m.confirmDialog != nil {
		m.confirmDialog.SetSize(m.width, m.height)
	}
}

// View renders the UI
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var s strings.Builder

	s.WriteString(m.renderHeader())
	s.WriteString("\n")
	s.WriteString(m.renderPanels())
	s.WriteString("\n")
	s.WriteString(m.renderFooter())

	// Overlays
	if m.showHelp {
		return m.helpOverlay.View()
	}
	if m.showBuildMenu {
		return m.buildMenuDialog.View()
	}
	if m.showDialog && m.confirmDialog != nil {
		return m.confirmDialog.View()
	}

	return s.String()
}

func (m *Model) renderHeader() string {
	title := TitleStyle.Render("KB " + strings.ToUpper(m.cfg.Keyboard.Name))

	// Device status
	var statusIcon, statusText string
	switch m.deviceStatus {
	case DeviceConnected:
		statusIcon = SuccessStyle.Render(StatusConnected)
		statusText = m.cfg.Device.Name + " Connected"
	case DeviceWaiting:
		statusIcon = WarningStyle.Render(StatusWaiting)
		statusText = m.cfg.Device.Name + " Waiting..."
	default:
		statusIcon = DimStyle.Render(StatusDisconnected)
		statusText = m.cfg.Device.Name + " Disconnected"
	}
	status := statusIcon + " " + statusText

	version := DimStyle.Render("kbflash")

	leftPart := title
	rightPart := status + "   " + version
	spacing := m.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart) - 2
	if spacing < 1 {
		spacing = 1
	}

	headerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Width(m.width - 2)

	content := leftPart + strings.Repeat(" ", spacing) + rightPart
	return headerStyle.Render(content)
}

func (m *Model) renderPanels() string {
	leftWidth := m.width * 30 / 100
	centerWidth := m.width * 40 / 100
	rightWidth := m.width - leftWidth - centerWidth - 6

	contentHeight := m.height - 6

	// Firmware panel
	firmwareStyle := PanelStyle.Width(leftWidth).Height(contentHeight)
	if m.activePanel == PanelFirmware {
		firmwareStyle = ActivePanelStyle.Width(leftWidth).Height(contentHeight)
	}
	firmwareTitle := " Firmware "
	firmwareContent := m.firmwarePanel.View()
	firmwarePanel := firmwareStyle.Render(AccentStyle.Render(firmwareTitle) + "\n\n" + firmwareContent)

	// Status panel
	statusStyle := PanelStyle.Width(centerWidth).Height(contentHeight)
	if m.activePanel == PanelStatus {
		statusStyle = ActivePanelStyle.Width(centerWidth).Height(contentHeight)
	}
	statusTitle := " Status "
	var statusContent string
	switch m.state {
	case StateIdle:
		statusContent = m.statusPanel.ViewIdle(m.firmwarePanel.Selected())
	case StateBuilding:
		statusContent = m.statusPanel.ViewBuilding(m.buildPercent, m.buildTarget)
	case StateWaitingDisconnect:
		statusContent = m.statusPanel.ViewWaitingDisconnect(m.flashTarget)
	case StateWaitingDevice:
		statusContent = m.statusPanel.ViewWaiting(m.flashTarget)
	case StateFlashing:
		build := m.firmwarePanel.Selected()
		filename := ""
		if build != nil && len(build.Files) > 0 {
			// Find matching file
			target := strings.ToLower(m.flashTarget)
			for _, f := range build.Files {
				if strings.Contains(strings.ToLower(f.Name), target) {
					filename = f.Name
					break
				}
			}
			if filename == "" && len(build.Files) == 1 {
				filename = build.Files[0].Name
			}
		}
		statusContent = m.statusPanel.ViewFlashing(m.flashPercent, filename, m.flashTarget)
	case StateComplete:
		duration := time.Since(m.startTime)
		statusContent = m.statusPanel.ViewComplete(duration, m.completedSteps)
	}
	statusPanel := statusStyle.Render(AccentStyle.Render(statusTitle) + "\n\n" + statusContent)

	// Log panel
	logStyle := PanelStyle.Width(rightWidth).Height(contentHeight)
	if m.activePanel == PanelLog {
		logStyle = ActivePanelStyle.Width(rightWidth).Height(contentHeight)
	}
	logTitle := " Log "
	logContent := m.logPanel.View()
	logPanel := logStyle.Render(AccentStyle.Render(logTitle) + "\n\n" + logContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, firmwarePanel, statusPanel, logPanel)
}

func (m *Model) renderFooter() string {
	var hints []string

	switch m.state {
	case StateIdle:
		hints = []string{
			"j/k Navigate",
			"Enter Select",
		}
		if m.cfg.Build.Enabled {
			hints = append(hints, "b Build")
		}
		hints = append(hints, "f Flash")
		if m.cfg.Keyboard.Type == "split" {
			hints = append(hints, "r Reset")
		}
		hints = append(hints, "q Quit")
	case StateBuilding:
		hints = []string{"Building..."}
	case StateWaitingDisconnect:
		hints = []string{"Unplug device to continue", "Esc Cancel"}
	case StateWaitingDevice:
		hints = []string{"Connect device, double-tap reset", "Esc Cancel"}
	case StateFlashing:
		hints = []string{"Flashing... Do not disconnect device"}
	case StateComplete:
		hints = []string{"Enter Continue", "q Quit"}
	}

	left := DimStyle.Render(strings.Join(hints, "   "))
	right := DimStyle.Render("? Help")

	spacing := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if spacing < 1 {
		spacing = 1
	}

	return " " + left + strings.Repeat(" ", spacing) + right
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}
