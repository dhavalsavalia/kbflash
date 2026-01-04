package firmware

import (
	"bufio"
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// BuildProgress represents the current build state.
type BuildProgress struct {
	Current int
	Total   int
	Percent int
	Output  string
}

// BuildResult represents the outcome of a build operation.
type BuildResult struct {
	Success bool
	Error   error
}

// progressRegex matches ninja's [current/total] output.
var progressRegex = regexp.MustCompile(`^\[(\d+)/(\d+)\]`)

// Builder executes firmware build commands.
type Builder struct {
	command    string
	args       []string
	workingDir string
}

// NewBuilder creates a new builder with the specified configuration.
func NewBuilder(command string, args []string, workingDir string) *Builder {
	return &Builder{
		command:    command,
		args:       args,
		workingDir: workingDir,
	}
}

// Build executes the build command for the specified side.
// The progressFn callback is called for each progress update.
// Returns when the build completes or context is cancelled.
func (b *Builder) Build(ctx context.Context, side string, progressFn func(BuildProgress)) BuildResult {
	if progressFn == nil {
		progressFn = func(BuildProgress) {}
	}

	// Substitute {{side}} in args
	args := make([]string, len(b.args))
	for i, arg := range b.args {
		args[i] = strings.ReplaceAll(arg, "{{side}}", side)
	}

	cmd := exec.CommandContext(ctx, b.command, args...)
	if b.workingDir != "" {
		cmd.Dir = b.workingDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return BuildResult{Success: false, Error: err}
	}

	// Also capture stderr to stdout for ninja progress
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return BuildResult{Success: false, Error: err}
	}

	var maxTotal int
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if ctx.Err() != nil {
			_ = cmd.Process.Kill()
			return BuildResult{Success: false, Error: ctx.Err()}
		}

		line := scanner.Text()

		// Parse ninja progress [current/total]
		if matches := progressRegex.FindStringSubmatch(line); len(matches) == 3 {
			current, _ := strconv.Atoi(matches[1])
			total, _ := strconv.Atoi(matches[2])

			// Track the maximum total seen (ninja increments total as it discovers deps)
			if total > maxTotal {
				maxTotal = total
			}

			percent := 0
			if maxTotal > 0 {
				percent = (current * 100) / maxTotal
			}

			progressFn(BuildProgress{
				Current: current,
				Total:   maxTotal,
				Percent: percent,
				Output:  line,
			})
		} else {
			// Non-progress output
			progressFn(BuildProgress{
				Output: line,
			})
		}
	}

	if err := cmd.Wait(); err != nil {
		return BuildResult{Success: false, Error: err}
	}

	return BuildResult{Success: true}
}
