package firmware

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestBuilder_Build_SideSubstitution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()

	// Create a test script that echoes the argument
	scriptPath := filepath.Join(tmpDir, "build.sh")
	script := `#!/bin/bash
echo "Building side: $1"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	builder := NewBuilder(scriptPath, []string{"{{side}}"}, "")

	var outputs []string
	progressFn := func(p BuildProgress) {
		if p.Output != "" {
			outputs = append(outputs, p.Output)
		}
	}

	result := builder.Build(context.Background(), "left", progressFn)

	if !result.Success {
		t.Fatalf("Build failed: %v", result.Error)
	}

	// Check that {{side}} was substituted
	found := false
	for _, out := range outputs {
		if out == "Building side: left" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Building side: left' in output, got: %v", outputs)
	}
}

func TestBuilder_Build_NinjaProgressParsing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()

	// Create a script that outputs ninja-style progress
	scriptPath := filepath.Join(tmpDir, "build.sh")
	script := `#!/bin/bash
echo "[1/10] Compiling foo.c"
echo "[5/10] Compiling bar.c"
echo "[10/10] Linking"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	builder := NewBuilder(scriptPath, []string{}, "")

	var progressUpdates []BuildProgress
	progressFn := func(p BuildProgress) {
		if p.Percent > 0 || p.Current > 0 {
			progressUpdates = append(progressUpdates, p)
		}
	}

	result := builder.Build(context.Background(), "left", progressFn)

	if !result.Success {
		t.Fatalf("Build failed: %v", result.Error)
	}

	if len(progressUpdates) != 3 {
		t.Fatalf("expected 3 progress updates, got %d", len(progressUpdates))
	}

	// Check first progress
	if progressUpdates[0].Current != 1 || progressUpdates[0].Total != 10 {
		t.Errorf("first progress: got %d/%d, want 1/10", progressUpdates[0].Current, progressUpdates[0].Total)
	}
	if progressUpdates[0].Percent != 10 {
		t.Errorf("first percent: got %d, want 10", progressUpdates[0].Percent)
	}

	// Check final progress
	if progressUpdates[2].Current != 10 || progressUpdates[2].Total != 10 {
		t.Errorf("final progress: got %d/%d, want 10/10", progressUpdates[2].Current, progressUpdates[2].Total)
	}
	if progressUpdates[2].Percent != 100 {
		t.Errorf("final percent: got %d, want 100", progressUpdates[2].Percent)
	}
}

func TestBuilder_Build_WorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()

	// Create working directory
	workDir := filepath.Join(tmpDir, "work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a script that outputs pwd
	scriptPath := filepath.Join(tmpDir, "build.sh")
	script := `#!/bin/bash
pwd
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	builder := NewBuilder(scriptPath, []string{}, workDir)

	var outputs []string
	progressFn := func(p BuildProgress) {
		if p.Output != "" {
			outputs = append(outputs, p.Output)
		}
	}

	result := builder.Build(context.Background(), "left", progressFn)

	if !result.Success {
		t.Fatalf("Build failed: %v", result.Error)
	}

	found := false
	for _, out := range outputs {
		if out == workDir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected working dir %q in output, got: %v", workDir, outputs)
	}
}

func TestBuilder_Build_CommandFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()

	// Create a script that fails
	scriptPath := filepath.Join(tmpDir, "build.sh")
	script := `#!/bin/bash
echo "Starting..."
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	builder := NewBuilder(scriptPath, []string{}, "")
	result := builder.Build(context.Background(), "left", nil)

	if result.Success {
		t.Error("expected Build to fail")
	}

	if result.Error == nil {
		t.Error("expected an error")
	}
}

func TestBuilder_Build_ContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()

	// Create a script that sleeps
	scriptPath := filepath.Join(tmpDir, "build.sh")
	script := `#!/bin/bash
sleep 10
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	builder := NewBuilder(scriptPath, []string{}, "")
	result := builder.Build(ctx, "left", nil)

	if result.Success {
		t.Error("expected Build to fail when context times out")
	}
}

func TestBuilder_Build_CommandNotFound(t *testing.T) {
	builder := NewBuilder("/nonexistent/command", []string{}, "")
	result := builder.Build(context.Background(), "left", nil)

	if result.Success {
		t.Error("expected Build to fail for nonexistent command")
	}

	if result.Error == nil {
		t.Error("expected an error for nonexistent command")
	}
}

func TestBuilder_Build_MultipleArgsSubstitution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tmpDir := t.TempDir()

	// Create a script that echoes all arguments
	scriptPath := filepath.Join(tmpDir, "build.sh")
	script := `#!/bin/bash
echo "args: $@"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	builder := NewBuilder(scriptPath, []string{"--side={{side}}", "--output={{side}}.uf2"}, "")

	var outputs []string
	progressFn := func(p BuildProgress) {
		if p.Output != "" {
			outputs = append(outputs, p.Output)
		}
	}

	result := builder.Build(context.Background(), "right", progressFn)

	if !result.Success {
		t.Fatalf("Build failed: %v", result.Error)
	}

	expected := "args: --side=right --output=right.uf2"
	found := false
	for _, out := range outputs {
		if out == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %q in output, got: %v", expected, outputs)
	}
}

func TestProgressRegex(t *testing.T) {
	tests := []struct {
		input   string
		current int
		total   int
		matches bool
	}{
		{"[1/10] Compiling foo.c", 1, 10, true},
		{"[50/100] Linking bar.o", 50, 100, true},
		{"[999/1000] Final step", 999, 1000, true},
		{"Building left half...", 0, 0, false},
		{"Error: something went wrong", 0, 0, false},
		{"", 0, 0, false},
	}

	for _, tc := range tests {
		matches := progressRegex.FindStringSubmatch(tc.input)
		if tc.matches {
			if len(matches) != 3 {
				t.Errorf("expected match for %q, got none", tc.input)
				continue
			}
		} else {
			if len(matches) == 3 {
				t.Errorf("expected no match for %q, got %v", tc.input, matches)
			}
		}
	}
}
