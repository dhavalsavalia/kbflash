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
)

// ExampleConfig is the template for --init with documentation comments.
const ExampleConfig = `# kbflash configuration
# See: https://github.com/dhavalsavalia/kbflash

[keyboard]
# Required: Name of your keyboard
name = "corne"

# Optional: Keyboard type (e.g., "split", "unibody")
type = "split"

# Optional: For split keyboards, the side names
sides = ["left", "right"]

[build]
# Enable automatic firmware building before flash
enabled = false

# Build command to run
command = "make"

# Arguments to pass to build command
args = []

# Directory to run build command in
working_dir = ""

# Directory containing built firmware files
firmware_dir = ""

# Glob pattern to match firmware files
file_pattern = "*.uf2"

[device]
# Required: Device name to detect (shown in system when keyboard enters bootloader)
name = "RPI-RP2"

# How often to poll for device (duration string: "500ms", "1s", etc.)
poll_interval = "500ms"
`

// GenerateExampleConfig writes the example config to the given path.
// If path is empty, it uses the default XDG path.
// Returns the path where the file was written.
func GenerateExampleConfig(path string) (string, error) {
	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return "", err
		}
		path = defaultPath
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
