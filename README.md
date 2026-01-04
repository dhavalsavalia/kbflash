# kbflash

Hackable keyboard firmware flasher TUI. ZMK-focused, cross-platform.

## Install

```bash
# Homebrew (macOS)
brew install dhavalsavalia/kbflash/kbflash

# Go
go install github.com/dhavalsavalia/kbflash@latest
```

## Usage

```bash
# Launch TUI
kbflash

# Use custom config
kbflash --config ./my-keyboard.toml

# Generate example config
kbflash --init
```

## Configuration

Create `~/.config/kbflash/config.toml`:

```toml
[keyboard]
name = "corne"
type = "split"  # "split" or "uni"
sides = ["left", "right"]

[build]
enabled = true
command = "./build.sh"
args = ["{{side}}"]
firmware_dir = "./firmware"
file_pattern = "*.uf2"

[device]
name = "NICENANO"
poll_interval = 500
```

## License

MIT
