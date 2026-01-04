package device

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	detector := New()
	if detector == nil {
		t.Fatal("New() returned nil")
	}
}

func TestDetector_InitialDisconnected(t *testing.T) {
	detector := New()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	events := detector.Detect(ctx, "NONEXISTENT_VOLUME_12345", 10*time.Millisecond)

	// First event should indicate disconnected
	event := <-events
	if event.Connected {
		t.Error("expected initial event to be disconnected for nonexistent volume")
	}
}

func TestDetector_InitialConnected(t *testing.T) {
	// Create a temp directory to simulate a mounted volume
	dir := t.TempDir()
	volumeName := filepath.Base(dir)
	volumeParent := filepath.Dir(dir)

	// We need to test with a real path structure
	// Since we can't easily mock /Volumes or /media, we'll test the event emission logic
	detector := &testDetector{basePath: volumeParent}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	events := detector.Detect(ctx, volumeName, 10*time.Millisecond)

	event := <-events
	if !event.Connected {
		t.Error("expected initial event to be connected for existing volume")
	}
	if event.Path != dir {
		t.Errorf("path = %q, want %q", event.Path, dir)
	}
}

func TestDetector_ContextCancellation(t *testing.T) {
	detector := New()
	ctx, cancel := context.WithCancel(context.Background())

	events := detector.Detect(ctx, "NONEXISTENT_VOLUME_12345", 10*time.Millisecond)

	// Read initial event
	<-events

	// Cancel and verify channel closes
	cancel()

	select {
	case _, ok := <-events:
		if ok {
			// Might get one more event, wait for close
			_, ok = <-events
			if ok {
				t.Error("expected channel to close after context cancellation")
			}
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("channel did not close after context cancellation")
	}
}

func TestDetector_ConnectDisconnect(t *testing.T) {
	// Create temp directory structure for testing
	baseDir := t.TempDir()
	volumeName := "TEST_DEVICE"
	volumePath := filepath.Join(baseDir, volumeName)

	detector := &testDetector{basePath: baseDir}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	events := detector.Detect(ctx, volumeName, 20*time.Millisecond)

	// Initial event: disconnected
	event := <-events
	if event.Connected {
		t.Error("expected initial disconnected state")
	}

	// Simulate device connection
	if err := os.Mkdir(volumePath, 0755); err != nil {
		t.Fatalf("failed to create test volume: %v", err)
	}

	// Wait for connect event
	var gotConnect bool
	for i := 0; i < 10; i++ {
		select {
		case event := <-events:
			if event.Connected {
				gotConnect = true
				if event.Path != volumePath {
					t.Errorf("connect path = %q, want %q", event.Path, volumePath)
				}
			}
		case <-time.After(50 * time.Millisecond):
		}
		if gotConnect {
			break
		}
	}
	if !gotConnect {
		t.Error("did not receive connect event")
	}

	// Simulate device disconnection
	if err := os.Remove(volumePath); err != nil {
		t.Fatalf("failed to remove test volume: %v", err)
	}

	// Wait for disconnect event
	var gotDisconnect bool
	for i := 0; i < 10; i++ {
		select {
		case event := <-events:
			if !event.Connected {
				gotDisconnect = true
			}
		case <-time.After(50 * time.Millisecond):
		}
		if gotDisconnect {
			break
		}
	}
	if !gotDisconnect {
		t.Error("did not receive disconnect event")
	}
}

func TestDetector_NoSpuriousEvents(t *testing.T) {
	// Verify detector doesn't emit events when state hasn't changed
	baseDir := t.TempDir()
	volumeName := "STABLE_DEVICE"

	detector := &testDetector{basePath: baseDir}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	events := detector.Detect(ctx, volumeName, 10*time.Millisecond)

	// Get initial event
	<-events

	// Count any additional events (there should be none)
	eventCount := 0
	for {
		select {
		case <-events:
			eventCount++
		case <-ctx.Done():
			if eventCount > 0 {
				t.Errorf("received %d spurious events when state was stable", eventCount)
			}
			return
		}
	}
}

// testDetector is a test implementation that uses a configurable base path
type testDetector struct {
	basePath string
}

func (d *testDetector) Detect(ctx context.Context, volumeName string, pollInterval time.Duration) <-chan Event {
	events := make(chan Event)

	go func() {
		defer close(events)

		path := filepath.Join(d.basePath, volumeName)
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

func (d *testDetector) exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
