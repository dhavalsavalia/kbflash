package firmware

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// File represents a firmware file.
type File struct {
	Name string
	Path string
	Size int64
}

// Build represents a firmware build (dated directory or flat).
type Build struct {
	Date  string // YYYYMMDD format or empty for flat structure
	Path  string
	Files []File
}

// Scanner scans firmware directories for UF2 files.
type Scanner struct {
	firmwareDir string
	filePattern string
}

// NewScanner creates a new firmware scanner.
func NewScanner(firmwareDir, filePattern string) *Scanner {
	return &Scanner{
		firmwareDir: firmwareDir,
		filePattern: filePattern,
	}
}

// Scan scans for firmware builds and returns them sorted by date (newest first).
// Supports both dated subdirectories (YYYYMMDD) and flat structure.
func (s *Scanner) Scan(ctx context.Context) ([]Build, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.firmwareDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Build{}, nil
		}
		return nil, err
	}

	var builds []Build

	// First, check for UF2 files directly in firmware_dir (flat structure)
	flatFiles, err := s.scanDirectory(ctx, s.firmwareDir)
	if err != nil {
		return nil, err
	}
	if len(flatFiles) > 0 {
		builds = append(builds, Build{
			Date:  "",
			Path:  s.firmwareDir,
			Files: flatFiles,
		})
	}

	// Then scan dated subdirectories
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if !entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Check if directory name looks like a date (YYYYMMDD)
		if !isDateDir(name) {
			continue
		}

		buildPath := filepath.Join(s.firmwareDir, name)
		files, err := s.scanDirectory(ctx, buildPath)
		if err != nil {
			continue
		}

		if len(files) > 0 {
			builds = append(builds, Build{
				Date:  name,
				Path:  buildPath,
				Files: files,
			})
		}
	}

	// Sort by date descending (newest first), flat builds at the end
	sort.Slice(builds, func(i, j int) bool {
		// Flat structure (empty date) goes last
		if builds[i].Date == "" {
			return false
		}
		if builds[j].Date == "" {
			return true
		}
		return builds[i].Date > builds[j].Date
	})

	return builds, nil
}

// scanDirectory scans a directory for files matching the pattern.
func (s *Scanner) scanDirectory(ctx context.Context, dir string) ([]File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []File
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matched, err := filepath.Match(s.filePattern, entry.Name())
		if err != nil || !matched {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, File{
			Name: entry.Name(),
			Path: filepath.Join(dir, entry.Name()),
			Size: info.Size(),
		})
	}

	return files, nil
}

// FindLatest returns the most recent build, or nil if none found.
func (s *Scanner) FindLatest(ctx context.Context) (*Build, error) {
	builds, err := s.Scan(ctx)
	if err != nil {
		return nil, err
	}
	if len(builds) == 0 {
		return nil, nil
	}
	return &builds[0], nil
}

// isDateDir checks if a string is in YYYYMMDD format.
func isDateDir(s string) bool {
	if len(s) != 8 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// FormatDate formats YYYYMMDD to human-readable format.
func FormatDate(date string) string {
	if len(date) != 8 {
		return date
	}
	t, err := time.Parse("20060102", date)
	if err != nil {
		return date
	}
	return t.Format("2006-01-02")
}

// FormatSize formats bytes to human-readable size.
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return formatInt(bytes) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return formatFloat(float64(bytes)/float64(div)) + " " + string("KMGTPE"[exp]) + "B"
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	if negative {
		result = append([]byte{'-'}, result...)
	}
	return string(result)
}

func formatFloat(f float64) string {
	intPart := int64(f)
	decPart := int64((f - float64(intPart)) * 10)
	return formatInt(intPart) + "." + formatInt(decPart)
}
