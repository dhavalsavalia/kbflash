package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Duration wraps time.Duration for TOML string parsing.
type Duration time.Duration

func (d *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// Config represents the complete kbflash configuration.
type Config struct {
	Keyboard KeyboardConfig `toml:"keyboard"`
	Build    BuildConfig    `toml:"build"`
	Device   DeviceConfig   `toml:"device"`
}

// KeyboardConfig defines keyboard identification and layout.
type KeyboardConfig struct {
	Name  string   `toml:"name"`
	Type  string   `toml:"type"`
	Sides []string `toml:"sides"`
}

// BuildConfig defines firmware build settings.
type BuildConfig struct {
	Enabled     bool     `toml:"enabled"`
	Mode        string   `toml:"mode"`         // "native" or "docker"
	Command     string   `toml:"command"`      // for native mode
	Args        []string `toml:"args"`         // for native mode
	WorkingDir  string   `toml:"working_dir"`
	FirmwareDir string   `toml:"firmware_dir"`
	FilePattern string   `toml:"file_pattern"`

	// Docker mode settings
	Image  string `toml:"image"`  // Docker image (default: zmkfirmware/zmk-dev-arm:stable)
	Board  string `toml:"board"`  // ZMK board (e.g., nice_nano_v2)
	Shield string `toml:"shield"` // ZMK shield (e.g., corne) - _left/_right added automatically
}

// DeviceConfig defines device detection settings.
type DeviceConfig struct {
	Name         string   `toml:"name"`
	PollInterval Duration `toml:"poll_interval"`
}

// DefaultPath returns the default config file path following XDG conventions.
// On Unix, checks $XDG_CONFIG_HOME first, then falls back to ~/.config.
func DefaultPath() (string, error) {
	// Check XDG_CONFIG_HOME first (Unix standard)
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "kbflash", "config.toml"), nil
	}

	// Fallback to ~/.config on Unix, or os.UserConfigDir on other platforms
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "kbflash", "config.toml"), nil
}

// LocalConfigName is the filename looked for in the current directory.
const LocalConfigName = "config.kbflash.toml"

// Load reads and parses a config file from the given path.
// If path is empty, it looks for config.kbflash.toml in the current directory first,
// then falls back to the default XDG path (~/.config/kbflash/config.toml).
func Load(path string) (*Config, error) {
	if path == "" {
		// Check for local config first
		if _, err := os.Stat(LocalConfigName); err == nil {
			path = LocalConfigName
		} else {
			// Fall back to XDG default
			defaultPath, err := DefaultPath()
			if err != nil {
				return nil, err
			}
			path = defaultPath
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	cfg := &Config{}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config file: %w", err)
	}

	applyDefaults(cfg)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// applyDefaults sets default values for optional fields.
func applyDefaults(cfg *Config) {
	if cfg.Device.PollInterval == 0 {
		cfg.Device.PollInterval = DefaultPollInterval
	}
	if cfg.Build.FilePattern == "" {
		cfg.Build.FilePattern = DefaultFilePattern
	}
	if cfg.Build.Mode == "" {
		cfg.Build.Mode = "native" // default to native for backwards compatibility
	}
	if cfg.Build.Image == "" {
		cfg.Build.Image = DefaultDockerImage
	}
}

// validate checks that required fields are present.
func validate(cfg *Config) error {
	var errs []error

	if cfg.Keyboard.Name == "" {
		errs = append(errs, errors.New("keyboard.name is required"))
	}
	if cfg.Device.Name == "" {
		errs = append(errs, errors.New("device.name is required"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
