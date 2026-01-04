package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_ValidConfig(t *testing.T) {
	content := `
[keyboard]
name = "corne"
type = "split"
sides = ["left", "right"]

[build]
enabled = true
command = "make"
args = ["-j4"]
working_dir = "/tmp/firmware"
firmware_dir = "/tmp/firmware/build"
file_pattern = "*.uf2"

[device]
name = "RPI-RP2"
poll_interval = "1s"
`
	path := writeTempConfig(t, content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Keyboard
	if cfg.Keyboard.Name != "corne" {
		t.Errorf("keyboard.name = %q, want %q", cfg.Keyboard.Name, "corne")
	}
	if cfg.Keyboard.Type != "split" {
		t.Errorf("keyboard.type = %q, want %q", cfg.Keyboard.Type, "split")
	}
	if len(cfg.Keyboard.Sides) != 2 {
		t.Errorf("keyboard.sides len = %d, want 2", len(cfg.Keyboard.Sides))
	}

	// Build
	if !cfg.Build.Enabled {
		t.Error("build.enabled = false, want true")
	}
	if cfg.Build.Command != "make" {
		t.Errorf("build.command = %q, want %q", cfg.Build.Command, "make")
	}

	// Device
	if cfg.Device.Name != "RPI-RP2" {
		t.Errorf("device.name = %q, want %q", cfg.Device.Name, "RPI-RP2")
	}
	if cfg.Device.PollInterval != Duration(time.Second) {
		t.Errorf("device.poll_interval = %v, want %v", cfg.Device.PollInterval, Duration(time.Second))
	}
}

func TestLoad_Defaults(t *testing.T) {
	content := `
[keyboard]
name = "test"

[device]
name = "TEST-DEVICE"
`
	path := writeTempConfig(t, content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Device.PollInterval != DefaultPollInterval {
		t.Errorf("poll_interval = %v, want default %v", cfg.Device.PollInterval, DefaultPollInterval)
	}
	if cfg.Build.FilePattern != DefaultFilePattern {
		t.Errorf("file_pattern = %q, want default %q", cfg.Build.FilePattern, DefaultFilePattern)
	}
}

func TestLoad_MissingKeyboardName(t *testing.T) {
	content := `
[keyboard]
type = "split"

[device]
name = "RPI-RP2"
`
	path := writeTempConfig(t, content)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing keyboard.name")
	}
}

func TestLoad_MissingDeviceName(t *testing.T) {
	content := `
[keyboard]
name = "corne"

[device]
poll_interval = "500ms"
`
	path := writeTempConfig(t, content)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing device.name")
	}
}

func TestLoad_MissingBothRequired(t *testing.T) {
	content := `
[keyboard]
type = "split"

[device]
poll_interval = "500ms"
`
	path := writeTempConfig(t, content)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	content := `this is not valid toml {{{`
	path := writeTempConfig(t, content)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestGenerateExampleConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kbflash", "config.toml")

	result, err := GenerateExampleConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != path {
		t.Errorf("returned path = %q, want %q", result, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read generated file: %v", err)
	}

	if string(data) != ExampleConfig {
		t.Error("generated config does not match ExampleConfig")
	}

	// Verify the generated config is valid and loadable
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("generated config is not loadable: %v", err)
	}
	if cfg.Keyboard.Name != "corne" {
		t.Errorf("example keyboard.name = %q, want %q", cfg.Keyboard.Name, "corne")
	}
}

func TestDefaultPath(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !filepath.IsAbs(path) {
		t.Errorf("path %q is not absolute", path)
	}

	if filepath.Base(path) != "config.toml" {
		t.Errorf("path base = %q, want config.toml", filepath.Base(path))
	}
}

func TestDefaultPath_XDGConfigHome(t *testing.T) {
	// Save and restore original value
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	os.Setenv("XDG_CONFIG_HOME", "/custom/xdg/config")

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "/custom/xdg/config/kbflash/config.toml"
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
}

func TestGenerateExampleConfig_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Create existing file
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatalf("cannot create existing file: %v", err)
	}

	_, err := GenerateExampleConfig(path)
	if err == nil {
		t.Fatal("expected error for existing file")
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("cannot write temp config: %v", err)
	}
	return path
}
