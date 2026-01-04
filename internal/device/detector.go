package device

import (
	"context"
	"time"
)

// Event represents a device connection state change.
type Event struct {
	Connected bool
	Path      string
}

// Detector watches for device connection/disconnection.
type Detector interface {
	// Detect starts watching for the named volume and sends events on state changes.
	// Returns a channel that emits events when device connects or disconnects.
	// The channel is closed when the context is cancelled.
	Detect(ctx context.Context, volumeName string, pollInterval time.Duration) <-chan Event
}
