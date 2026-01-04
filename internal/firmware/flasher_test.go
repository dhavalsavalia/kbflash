package firmware

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFlasher_Flash_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "firmware.uf2")
	content := []byte("test firmware content")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Create destination directory
	dstDir := filepath.Join(tmpDir, "device")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatal(err)
	}

	flasher := NewFlasher()
	result := flasher.Flash(context.Background(), srcPath, dstDir)

	if !result.Success {
		t.Fatalf("Flash failed: %v", result.Error)
	}

	if result.BytesWritten != int64(len(content)) {
		t.Errorf("expected %d bytes written, got %d", len(content), result.BytesWritten)
	}

	// Verify file was copied
	dstPath := filepath.Join(dstDir, "firmware.uf2")
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}

	if string(dstContent) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", dstContent, content)
	}
}

func TestFlasher_Flash_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	flasher := NewFlasher()
	result := flasher.Flash(context.Background(), "/nonexistent/file.uf2", tmpDir)

	if result.Success {
		t.Error("expected Flash to fail for nonexistent source")
	}

	if result.Error == nil {
		t.Error("expected an error for nonexistent source")
	}
}

func TestFlasher_Flash_DestinationNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "firmware.uf2")
	if err := os.WriteFile(srcPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	flasher := NewFlasher()
	result := flasher.Flash(context.Background(), srcPath, "/nonexistent/path")

	if result.Success {
		t.Error("expected Flash to fail for nonexistent destination")
	}

	if result.Error == nil {
		t.Error("expected an error for nonexistent destination")
	}
}

func TestFlasher_Flash_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "firmware.uf2")
	if err := os.WriteFile(srcPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	dstDir := filepath.Join(tmpDir, "device")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	flasher := NewFlasher()
	result := flasher.Flash(ctx, srcPath, dstDir)

	if result.Success {
		t.Error("expected Flash to fail when context is cancelled")
	}

	if result.Error != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", result.Error)
	}
}

func TestFlasher_Flash_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a larger file (256KB)
	srcPath := filepath.Join(tmpDir, "large.uf2")
	content := make([]byte, 256*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	dstDir := filepath.Join(tmpDir, "device")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatal(err)
	}

	flasher := NewFlasher()
	result := flasher.Flash(context.Background(), srcPath, dstDir)

	if !result.Success {
		t.Fatalf("Flash failed: %v", result.Error)
	}

	if result.BytesWritten != int64(len(content)) {
		t.Errorf("expected %d bytes written, got %d", len(content), result.BytesWritten)
	}

	// Verify content
	dstContent, err := os.ReadFile(filepath.Join(dstDir, "large.uf2"))
	if err != nil {
		t.Fatal(err)
	}

	if len(dstContent) != len(content) {
		t.Errorf("size mismatch: got %d, want %d", len(dstContent), len(content))
	}
}
