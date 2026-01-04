package firmware

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanner_Scan_DatedDirectories(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create dated directories with UF2 files
	dates := []string{"20250101", "20250115", "20250102"}
	for _, date := range dates {
		dir := filepath.Join(tmpDir, date)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		// Create a UF2 file
		if err := os.WriteFile(filepath.Join(dir, "firmware.uf2"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	scanner := NewScanner(tmpDir, "*.uf2")
	builds, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(builds) != 3 {
		t.Errorf("expected 3 builds, got %d", len(builds))
	}

	// Should be sorted newest first
	if builds[0].Date != "20250115" {
		t.Errorf("expected newest date 20250115, got %s", builds[0].Date)
	}
	if builds[1].Date != "20250102" {
		t.Errorf("expected second date 20250102, got %s", builds[1].Date)
	}
	if builds[2].Date != "20250101" {
		t.Errorf("expected oldest date 20250101, got %s", builds[2].Date)
	}
}

func TestScanner_Scan_FlatStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create UF2 files directly in the directory
	files := []string{"left.uf2", "right.uf2"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	scanner := NewScanner(tmpDir, "*.uf2")
	builds, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(builds) != 1 {
		t.Errorf("expected 1 build, got %d", len(builds))
	}

	if builds[0].Date != "" {
		t.Errorf("expected empty date for flat structure, got %s", builds[0].Date)
	}

	if len(builds[0].Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(builds[0].Files))
	}
}

func TestScanner_Scan_MixedStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dated directory
	datedDir := filepath.Join(tmpDir, "20250120")
	if err := os.MkdirAll(datedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(datedDir, "dated.uf2"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create flat file
	if err := os.WriteFile(filepath.Join(tmpDir, "flat.uf2"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	scanner := NewScanner(tmpDir, "*.uf2")
	builds, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(builds) != 2 {
		t.Errorf("expected 2 builds, got %d", len(builds))
	}

	// Dated should come first, flat last
	if builds[0].Date != "20250120" {
		t.Errorf("expected dated build first, got date=%s", builds[0].Date)
	}
	if builds[1].Date != "" {
		t.Errorf("expected flat build last, got date=%s", builds[1].Date)
	}
}

func TestScanner_Scan_PatternMatching(t *testing.T) {
	tmpDir := t.TempDir()

	// Create various files
	files := []string{"left.uf2", "right.uf2", "readme.txt", "config.json"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	scanner := NewScanner(tmpDir, "*.uf2")
	builds, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(builds) != 1 {
		t.Fatalf("expected 1 build, got %d", len(builds))
	}

	if len(builds[0].Files) != 2 {
		t.Errorf("expected 2 UF2 files, got %d", len(builds[0].Files))
	}
}

func TestScanner_Scan_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	scanner := NewScanner(tmpDir, "*.uf2")
	builds, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(builds) != 0 {
		t.Errorf("expected 0 builds, got %d", len(builds))
	}
}

func TestScanner_Scan_NonExistentDirectory(t *testing.T) {
	scanner := NewScanner("/nonexistent/path", "*.uf2")
	builds, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("expected no error for nonexistent dir, got: %v", err)
	}

	if len(builds) != 0 {
		t.Errorf("expected 0 builds, got %d", len(builds))
	}
}

func TestScanner_Scan_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	if err := os.WriteFile(filepath.Join(tmpDir, "test.uf2"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	scanner := NewScanner(tmpDir, "*.uf2")
	_, err := scanner.Scan(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestScanner_FindLatest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dated directories
	for _, date := range []string{"20250101", "20250115"} {
		dir := filepath.Join(tmpDir, date)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "firmware.uf2"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	scanner := NewScanner(tmpDir, "*.uf2")
	build, err := scanner.FindLatest(context.Background())
	if err != nil {
		t.Fatalf("FindLatest failed: %v", err)
	}

	if build == nil {
		t.Fatal("expected a build, got nil")
	}

	if build.Date != "20250115" {
		t.Errorf("expected latest date 20250115, got %s", build.Date)
	}
}

func TestScanner_FindLatest_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	scanner := NewScanner(tmpDir, "*.uf2")
	build, err := scanner.FindLatest(context.Background())
	if err != nil {
		t.Fatalf("FindLatest failed: %v", err)
	}

	if build != nil {
		t.Errorf("expected nil build, got %+v", build)
	}
}

func TestIsDateDir(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"20250101", true},
		{"20251231", true},
		{"2025010", false},   // Too short
		{"202501011", false}, // Too long
		{"2025a101", false},  // Contains letter
		{"abcdefgh", false},  // All letters
		{"", false},          // Empty
	}

	for _, tc := range tests {
		got := isDateDir(tc.input)
		if got != tc.expected {
			t.Errorf("isDateDir(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"20250115", "2025-01-15"},
		{"20241231", "2024-12-31"},
		{"invalid", "invalid"},
		{"", ""},
	}

	for _, tc := range tests {
		got := FormatDate(tc.input)
		if got != tc.expected {
			t.Errorf("FormatDate(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tc := range tests {
		got := FormatSize(tc.input)
		if got != tc.expected {
			t.Errorf("FormatSize(%d) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
