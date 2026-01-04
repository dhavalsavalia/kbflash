//go:build linux

package device

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

type linuxDetector struct{}

// New returns a Detector for Linux.
func New() Detector {
	return &linuxDetector{}
}

func (d *linuxDetector) Detect(ctx context.Context, volumeName string, pollInterval time.Duration) <-chan Event {
	events := make(chan Event)

	go func() {
		defer close(events)

		user := os.Getenv("USER")
		paths := []string{
			filepath.Join("/run/media", user, volumeName),
			filepath.Join("/media", user, volumeName),
		}

		var lastConnected bool
		var lastPath string

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		// Check immediately on start
		connected, path := d.findDevice(paths)
		lastConnected = connected
		lastPath = path
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
				connected, path := d.findDevice(paths)
				if connected != lastConnected || path != lastPath {
					lastConnected = connected
					lastPath = path
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

func (d *linuxDetector) findDevice(paths []string) (bool, string) {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true, p
		}
	}
	// Return first path as the expected location when not connected
	if len(paths) > 0 {
		return false, paths[0]
	}
	return false, ""
}
