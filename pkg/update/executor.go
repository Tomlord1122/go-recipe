package update

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tomlord1122/tom-recipe/pkg/model"
)

// Result represents the outcome of an executed command
type Result struct {
	Command   model.Command
	Output    string
	Error     error
	StartTime time.Time
	EndTime   time.Time
	ExitCode  int
}

// ExecuteCommand runs a shell command and returns the result
func ExecuteCommand(command model.Command) Result {
	startTime := time.Now()

	// If UseShell, run via shell to preserve pipes/quotes; else split fields
	var cmd *exec.Cmd
	if strings.TrimSpace(command.Command) == "" {
		return Result{
			Command:   command,
			Output:    "",
			Error:     fmt.Errorf("empty command"),
			StartTime: startTime,
			EndTime:   time.Now(),
			ExitCode:  -1,
		}
	}
	if command.UseShell {
		// Unix shells; Windows support can be extended later when needed
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "bash"
		}
		cmd = exec.Command(shell, "-lc", command.Command)
	} else {
		// Split the command string into parts
		cmdParts := strings.Fields(command.Command)
		if len(cmdParts) == 0 {
			return Result{
				Command:   command,
				Output:    "",
				Error:     fmt.Errorf("empty command"),
				StartTime: startTime,
				EndTime:   time.Now(),
				ExitCode:  -1,
			}
		}
		cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
	}

	// Resolve working directory according to command settings
	if dir, derr := resolveWorkingDir(command); derr == nil && dir != "" {
		cmd.Dir = dir
	} else if derr != nil {
		return Result{
			Command:   command,
			Output:    "",
			Error:     derr,
			StartTime: startTime,
			EndTime:   time.Now(),
			ExitCode:  -1,
		}
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	// Calculate exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Create result
	result := Result{
		Command:   command,
		Output:    output,
		Error:     err,
		StartTime: startTime,
		EndTime:   time.Now(),
		ExitCode:  exitCode,
	}

	return result
}

// resolveWorkingDir decides the working directory based on per-command settings.
// Returns empty string to indicate "use current working directory".
func resolveWorkingDir(command model.Command) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(command.WorkingDirMode))
	switch mode {
	case "", "current":
		return "", nil
	case "home":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		return home, nil
	case "absolute":
		if strings.TrimSpace(command.WorkingDirPath) == "" {
			return "", errors.New("working directory path is required when mode is 'absolute'")
		}
		expanded, err := expandDirPlaceholders(command.WorkingDirPath)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(expanded) {
			return "", fmt.Errorf("working directory must be an absolute path: %s", expanded)
		}
		if fi, statErr := os.Stat(expanded); statErr != nil || !fi.IsDir() {
			return "", fmt.Errorf("working directory does not exist or is not a directory: %s", expanded)
		}
		return expanded, nil
	default:
		return "", fmt.Errorf("unknown WorkingDirMode: %s", mode)
	}
}

// expandDirPlaceholders expands ~, $HOME and ${cwd} in the provided path.
func expandDirPlaceholders(p string) (string, error) {
	// Environment variables like $HOME
	p = os.ExpandEnv(p)

	// ~ expansion
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand ~: %w", err)
		}
		if p == "~" {
			p = home
		} else if strings.HasPrefix(p, "~/") {
			p = filepath.Join(home, p[2:])
		}
	}

	// ${cwd} expansion
	if strings.Contains(p, "${cwd}") {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to resolve current working directory: %w", err)
		}
		p = strings.ReplaceAll(p, "${cwd}", cwd)
	}

	return p, nil
}

// FormatOutput formats the execution result for display
func FormatOutput(result Result) string {
	var sb strings.Builder

	// Command info
	sb.WriteString(fmt.Sprintf("Command: %s\n", result.Command.Command))
	sb.WriteString(fmt.Sprintf("Started: %s\n", result.StartTime.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Duration: %s\n", result.EndTime.Sub(result.StartTime)))
	sb.WriteString(fmt.Sprintf("Exit Code: %d\n", result.ExitCode))
	sb.WriteString("\n--- Output ---\n")

	// Command output
	sb.WriteString(result.Output)

	// Add error if present
	if result.Error != nil {
		sb.WriteString("\n--- Error ---\n")
		sb.WriteString(result.Error.Error())
	}

	return sb.String()
}
