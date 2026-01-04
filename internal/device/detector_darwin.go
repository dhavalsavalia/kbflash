//go:build darwin

package device

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

type darwinDetector struct{}

// New returns a Detector for macOS.
func New() Detector {
	return &darwinDetector{}
}

func (d *darwinDetector) Detect(ctx context.Context, volumeName string, pollInterval time.Duration) <-chan Event {
	events := make(chan Event)

	go func() {
		defer close(events)

		path := filepath.Join("/Volumes", volumeName)
		var lastConnected bool

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		// Check immediately on start
		connected := d.exists(path)
		lastConnected = connected
		select {
		case events <- Event{Connected: connected, Path: path}:
		case <-ctx.Done():
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				connected := d.exists(path)
				if connected != lastConnected {
					lastConnected = connected
					select {
					case events <- Event{Connected: connected, Path: path}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return events
}

func (d *darwinDetector) exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
