package firmware

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FlashResult represents the outcome of a flash operation.
type FlashResult struct {
	Success      bool
	Error        error
	BytesWritten int64
}

// Flasher handles copying firmware files to devices.
type Flasher struct{}

// NewFlasher creates a new flasher.
func NewFlasher() *Flasher {
	return &Flasher{}
}

// Flash copies a firmware file to the device path with size validation.
func (f *Flasher) Flash(ctx context.Context, srcPath, devicePath string) FlashResult {
	if err := ctx.Err(); err != nil {
		return FlashResult{Success: false, Error: err}
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return FlashResult{Success: false, Error: fmt.Errorf("open source: %w", err)}
	}
	defer src.Close()

	srcInfo, err := src.Stat()
	if err != nil {
		return FlashResult{Success: false, Error: fmt.Errorf("stat source: %w", err)}
	}

	dstPath := filepath.Join(devicePath, filepath.Base(srcPath))
	dst, err := os.Create(dstPath)
	if err != nil {
		return FlashResult{Success: false, Error: fmt.Errorf("create destination: %w", err)}
	}
	defer dst.Close()

	// Use a cancellable copy
	written, err := copyWithContext(ctx, dst, src)
	if err != nil {
		return FlashResult{Success: false, Error: fmt.Errorf("copy: %w", err), BytesWritten: written}
	}

	// Validate size
	if written != srcInfo.Size() {
		return FlashResult{
			Success:      false,
			Error:        fmt.Errorf("size mismatch: wrote %d, expected %d", written, srcInfo.Size()),
			BytesWritten: written,
		}
	}

	// Sync to ensure data is written
	if err := dst.Sync(); err != nil {
		return FlashResult{
			Success:      false,
			Error:        fmt.Errorf("sync: %w", err),
			BytesWritten: written,
		}
	}

	return FlashResult{Success: true, BytesWritten: written}
}

// copyWithContext copies from src to dst, respecting context cancellation.
func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64

	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		nr, err := src.Read(buf)
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if err != nil {
			if err == io.EOF {
				return written, nil
			}
			return written, err
		}
	}
}
