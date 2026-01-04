package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Default values for optional config fields.
const (
	DefaultPollInterval = Duration(500 * time.Millisecond)
	DefaultFilePattern  = "*.uf2"
	DefaultDockerImage  = "zmkfirmware/zmk-dev-arm:stable"
)

// ExampleConfig is the template for --init with documentation comments.
const ExampleConfig = `# kbflash configuration
# See: https://github.com/dhavalsavalia/kbflash

[keyboard]
# Required: Name of your keyboard
name = "corne"

# Keyboard type: "split" or "uni"
type = "split"

# For split keyboards, the side names
sides = ["left", "right"]

[build]
# Enable firmware building (set to false for flash-only mode)
enabled = true

# Build mode: "docker" (recommended) or "native"
# Docker mode only requires Docker installed - no ZMK toolchain needed!
mode = "docker"

# --- Docker mode settings ---
# Docker image (default: zmkfirmware/zmk-dev-arm:stable)
image = "zmkfirmware/zmk-dev-arm:stable"

# Your ZMK board (e.g., nice_nano_v2, seeeduino_xiao_ble)
board = "nice_nano_v2"

# Your ZMK shield (without _left/_right suffix)
shield = "corne"

# --- Native mode settings (if mode = "native") ---
# command = "./build.sh"
# args = ["{{side}}"]

# Directory containing your zmk-config (for docker) or to run build in (for native)
working_dir = "."

# Where to output/find firmware files
firmware_dir = "./firmware"

# Glob pattern to match firmware files
file_pattern = "*.uf2"

[device]
# Required: Device name shown when keyboard enters bootloader
# Common values: "NICENANO", "RPI-RP2", "XIAO-SENSE"
name = "NICENANO"

# How often to poll for device
poll_interval = "500ms"
`

// GenerateExampleConfig writes the example config to the given path.
// If path is empty, it uses the default XDG path.
// Returns error if file already exists (won't overwrite).
// Returns the path where the file was written.
func GenerateExampleConfig(path string) (string, error) {
	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return "", err
		}
		path = defaultPath
	}

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("config file already exists: %s (delete it first to regenerate)", path)
	}

	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(ExampleConfig), 0644); err != nil {
		return "", fmt.Errorf("cannot write config file: %w", err)
	}

	return path, nil
}
