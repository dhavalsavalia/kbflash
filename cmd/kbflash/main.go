package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dhavalsavalia/kbflash/internal/config"
	"github.com/dhavalsavalia/kbflash/internal/device"
	"github.com/dhavalsavalia/kbflash/internal/firmware"
	"github.com/dhavalsavalia/kbflash/internal/ui"
)

var version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.BoolVar(versionFlag, "v", false, "Print version and exit (shorthand)")

	configPath := flag.String("config", "", "Path to config file")
	initConfig := flag.Bool("init", false, "Generate example config file")
	noTUI := flag.Bool("no-tui", false, "Headless mode for CI/scripting")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("kbflash %s\n", version)
		os.Exit(0)
	}

	if *initConfig {
		path, err := config.GenerateExampleConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created config at %s\n", path)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *noTUI {
		if err := runHeadless(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Launch TUI
	model := ui.NewModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runHeadless runs the flash operation without TUI
func runHeadless(cfg *config.Config) error {
	fmt.Printf("kbflash %s - Headless mode\n", version)
	fmt.Printf("Keyboard: %s (%s)\n", cfg.Keyboard.Name, cfg.Keyboard.Type)

	// Scan for firmware
	scanner := firmware.NewScanner(cfg.Build.FirmwareDir, cfg.Build.FilePattern)
	ctx := context.Background()

	builds, err := scanner.Scan(ctx)
	if err != nil {
		return fmt.Errorf("scan firmware: %w", err)
	}
	if len(builds) == 0 {
		return fmt.Errorf("no firmware found in %s", cfg.Build.FirmwareDir)
	}

	build := builds[0] // Use latest
	fmt.Printf("Using firmware: %s (%d files)\n", formatBuildDate(build.Date), len(build.Files))

	// Get sides to flash
	sides := cfg.Keyboard.Sides
	if len(sides) == 0 {
		sides = []string{"main"}
	}

	detector := device.New()
	flasher := firmware.NewFlasher()
	pollInterval := time.Duration(cfg.Device.PollInterval)

	for _, side := range sides {
		fmt.Printf("\nFlashing %s...\n", side)

		// Find firmware file for this side
		var filePath string
		target := strings.ToLower(side)
		for _, f := range build.Files {
			fname := strings.ToLower(f.Name)
			if strings.Contains(fname, target) {
				filePath = f.Path
				break
			}
		}
		if filePath == "" && len(build.Files) == 1 {
			filePath = build.Files[0].Path
		}
		if filePath == "" {
			return fmt.Errorf("no firmware file for %s", side)
		}

		fmt.Printf("File: %s\n", filePath)

		// Wait for device
		fmt.Printf("Waiting for %s...\n", cfg.Device.Name)

		detectCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		events := detector.Detect(detectCtx, cfg.Device.Name, pollInterval)

		var devicePath string
		for event := range events {
			if event.Connected {
				devicePath = event.Path
				break
			}
		}
		cancel()

		if devicePath == "" {
			return fmt.Errorf("timeout waiting for device")
		}

		fmt.Printf("Device found at %s\n", devicePath)

		// Flash
		result := flasher.Flash(ctx, filePath, devicePath)
		if !result.Success {
			return fmt.Errorf("flash failed: %w", result.Error)
		}

		fmt.Printf("Flashed %s (%d bytes)\n", side, result.BytesWritten)
	}

	fmt.Println("\nFlash complete!")
	return nil
}

func formatBuildDate(date string) string {
	if date == "" {
		return "current"
	}
	return firmware.FormatDate(date)
}
