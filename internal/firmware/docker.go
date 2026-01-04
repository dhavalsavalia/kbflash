package firmware

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DockerBuilder builds ZMK firmware using Docker.
type DockerBuilder struct {
	image      string
	board      string
	shield     string
	workingDir string
	outputDir  string
}

// NewDockerBuilder creates a new Docker-based builder.
func NewDockerBuilder(image, board, shield, workingDir, outputDir string) *DockerBuilder {
	return &DockerBuilder{
		image:      image,
		board:      board,
		shield:     shield,
		workingDir: workingDir,
		outputDir:  outputDir,
	}
}

// CheckDocker verifies Docker is installed and running.
func CheckDocker(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker is not running. Please start Docker Desktop and try again")
	}
	return nil
}

// EnsureImage pulls the Docker image if not present.
func (b *DockerBuilder) EnsureImage(ctx context.Context, progress func(string)) error {
	// Check if image exists locally
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", b.image)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if cmd.Run() == nil {
		progress("Image ready: " + b.image)
		return nil
	}

	// Pull the image
	progress("Pulling " + b.image + " (this may take a few minutes)...")

	cmd = exec.CommandContext(ctx, "docker", "pull", b.image)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// Parse Docker pull progress
		if strings.Contains(line, "Pulling") || strings.Contains(line, "Download") ||
			strings.Contains(line, "Pull complete") || strings.Contains(line, "Already exists") {
			progress(line)
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	progress("Image ready: " + b.image)
	return nil
}

// Build builds firmware for the given side using Docker.
func (b *DockerBuilder) Build(ctx context.Context, side string, progress func(BuildProgress)) BuildResult {
	startTime := time.Now()

	// Resolve working directory to absolute path
	workDir, err := filepath.Abs(b.workingDir)
	if err != nil {
		return BuildResult{Success: false, Error: fmt.Errorf("invalid working directory: %w", err)}
	}

	// Resolve output directory
	outputDir, err := filepath.Abs(b.outputDir)
	if err != nil {
		return BuildResult{Success: false, Error: fmt.Errorf("invalid output directory: %w", err)}
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return BuildResult{Success: false, Error: fmt.Errorf("cannot create output directory: %w", err)}
	}

	// Determine shield name with side suffix
	shieldName := b.shield
	if side != "" && side != "all" && side != "main" {
		shieldName = b.shield + "_" + side
	}

	// Build directory inside container
	buildDir := fmt.Sprintf("/workdir/build/%s", side)
	if side == "" || side == "all" || side == "main" {
		buildDir = "/workdir/build/main"
	}

	// Construct west build command
	// west build -s zmk/app -p -b <board> -d <build_dir> -- -DSHIELD=<shield> -DZMK_CONFIG=/workdir/config
	westCmd := []string{
		"west", "build",
		"-s", "zmk/app",
		"-p", // pristine build
		"-b", b.board,
		"-d", buildDir,
		"--",
		"-DSHIELD=" + shieldName,
		"-DZMK_CONFIG=/workdir/config",
	}

	// Docker run command
	args := []string{
		"run", "--rm",
		"-v", workDir + ":/workdir",
		"-w", "/workdir",
		b.image,
	}
	args = append(args, westCmd...)

	progress(BuildProgress{Percent: 5, Message: "Starting Docker build for " + side})

	cmd := exec.CommandContext(ctx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return BuildResult{Success: false, Error: err}
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return BuildResult{Success: false, Error: fmt.Errorf("failed to start Docker: %w", err)}
	}

	// Parse ninja progress
	ninjaRe := regexp.MustCompile(`\[(\d+)/(\d+)\]`)
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse ninja progress: [current/total]
		if matches := ninjaRe.FindStringSubmatch(line); len(matches) == 3 {
			var current, total int
			fmt.Sscanf(matches[1], "%d", &current)
			fmt.Sscanf(matches[2], "%d", &total)
			if total > 0 {
				pct := 10 + (current * 85 / total) // 10-95%
				progress(BuildProgress{Percent: pct, Message: line})
			}
		} else if strings.Contains(line, "error:") || strings.Contains(line, "Error:") {
			progress(BuildProgress{Percent: -1, Message: line})
		}
	}

	if err := cmd.Wait(); err != nil {
		return BuildResult{Success: false, Error: fmt.Errorf("build failed: %w", err), Duration: time.Since(startTime)}
	}

	progress(BuildProgress{Percent: 95, Message: "Copying firmware..."})

	// Copy UF2 from build directory to output
	uf2Path := filepath.Join(workDir, "build", side, "zephyr", "zmk.uf2")
	if side == "" || side == "all" || side == "main" {
		uf2Path = filepath.Join(workDir, "build", "main", "zephyr", "zmk.uf2")
	}

	// Create dated output directory
	dateStr := time.Now().Format("20060102")
	datedOutputDir := filepath.Join(outputDir, dateStr)
	if err := os.MkdirAll(datedOutputDir, 0755); err != nil {
		return BuildResult{Success: false, Error: fmt.Errorf("cannot create dated output directory: %w", err)}
	}

	// Determine output filename
	outputName := fmt.Sprintf("%s_%s.uf2", b.shield, side)
	if side == "" || side == "all" || side == "main" {
		outputName = b.shield + ".uf2"
	}
	outputPath := filepath.Join(datedOutputDir, outputName)

	// Copy the file
	data, err := os.ReadFile(uf2Path)
	if err != nil {
		return BuildResult{Success: false, Error: fmt.Errorf("cannot read built firmware: %w", err)}
	}
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return BuildResult{Success: false, Error: fmt.Errorf("cannot write firmware to output: %w", err)}
	}

	progress(BuildProgress{Percent: 100, Message: "Build complete: " + outputName})

	return BuildResult{
		Success:    true,
		Duration:   time.Since(startTime),
		OutputPath: outputPath,
	}
}

// BuildAll builds firmware for all sides (for split keyboards).
func (b *DockerBuilder) BuildAll(ctx context.Context, sides []string, progress func(BuildProgress)) []BuildResult {
	results := make([]BuildResult, len(sides))
	for i, side := range sides {
		basePercent := i * 100 / len(sides)
		sideProgress := func(p BuildProgress) {
			// Scale progress for this side
			scaledPercent := basePercent + (p.Percent * 100 / len(sides) / 100)
			progress(BuildProgress{Percent: scaledPercent, Message: fmt.Sprintf("[%s] %s", side, p.Message)})
		}
		results[i] = b.Build(ctx, side, sideProgress)
		if !results[i].Success {
			return results[:i+1]
		}
	}
	return results
}
