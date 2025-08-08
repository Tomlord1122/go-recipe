package update

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tomlord1122/go-recipe/pkg/model"
	"github.com/creack/pty"
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

// ExecuteCommandStreaming runs a command and streams output to the provided writer.
func ExecuteCommandStreaming(command model.Command, stream io.Writer) Result {
	startTime := time.Now()

	if strings.TrimSpace(command.Command) == "" {
		return Result{Command: command, Error: fmt.Errorf("empty command"), StartTime: startTime, EndTime: time.Now(), ExitCode: -1}
	}

	var cmd *exec.Cmd
	if command.UseShell || command.Interactive {
		// For interactive commands, prefer running via shell to preserve environment
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "bash"
		}
		cmd = exec.Command(shell, "-lc", command.Command)
	} else {
		parts := strings.Fields(command.Command)
		if len(parts) == 0 {
			return Result{Command: command, Error: fmt.Errorf("empty command"), StartTime: startTime, EndTime: time.Now(), ExitCode: -1}
		}
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	if dir, derr := resolveWorkingDir(command); derr == nil && dir != "" {
		cmd.Dir = dir
	} else if derr != nil {
		return Result{Command: command, Error: derr, StartTime: startTime, EndTime: time.Now(), ExitCode: -1}
	}

	// Attach streaming writer
	cmd.Stdout = stream
	cmd.Stderr = stream

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return Result{Command: command, Output: "", Error: err, StartTime: startTime, EndTime: time.Now(), ExitCode: exitCode}
}

// ExecuteCommandInteractiveAttached runs an interactive command attached to the current TTY.
// Stdout/Stderr/Stdin are bound to the parent process so full-screen TUIs (e.g., htop) work
// in the same terminal session.
func ExecuteCommandInteractiveAttached(command model.Command) Result {
	startTime := time.Now()
	if strings.TrimSpace(command.Command) == "" {
		return Result{Command: command, Error: fmt.Errorf("empty command"), StartTime: startTime, EndTime: time.Now(), ExitCode: -1}
	}

	var cmd *exec.Cmd
	if command.UseShell || command.Interactive {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "bash"
		}
		cmd = exec.Command(shell, "-lc", command.Command)
	} else {
		parts := strings.Fields(command.Command)
		if len(parts) == 0 {
			return Result{Command: command, Error: fmt.Errorf("empty command"), StartTime: startTime, EndTime: time.Now(), ExitCode: -1}
		}
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	if dir, derr := resolveWorkingDir(command); derr == nil && dir != "" {
		cmd.Dir = dir
	} else if derr != nil {
		return Result{Command: command, Error: derr, StartTime: startTime, EndTime: time.Now(), ExitCode: -1}
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return Result{Command: command, Output: "", Error: err, StartTime: startTime, EndTime: time.Now(), ExitCode: exitCode}
}

// StartInteractiveProcess starts a long-running process and returns the *exec.Cmd so caller can manage lifecycle.
// Stdout/Stderr are streamed to the provided writer.
func StartInteractiveProcess(command model.Command, stream io.Writer) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	if command.UseShell || command.Interactive {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "bash"
		}
		cmd = exec.Command(shell, "-lc", command.Command)
	} else {
		parts := strings.Fields(command.Command)
		if len(parts) == 0 {
			return nil, fmt.Errorf("empty command")
		}
		cmd = exec.Command(parts[0], parts[1:]...)
	}
	if dir, derr := resolveWorkingDir(command); derr == nil && dir != "" {
		cmd.Dir = dir
	} else if derr != nil {
		return nil, derr
	}
	cmd.Stdout = stream
	cmd.Stderr = stream
	// do not bind Stdin so we keep control; many tools exit on 'q' printed to stdout is not correct
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// StartInteractivePTY starts the command attached to a PTY so full-screen TUIs can render.
// The PTY output is continuously copied to the provided stream until the process exits or the PTY is closed.
func StartInteractivePTY(command model.Command, stream io.Writer) (*exec.Cmd, *os.File, error) {
	var cmd *exec.Cmd
	if command.UseShell || command.Interactive {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "bash"
		}
		cmd = exec.Command(shell, "-lc", command.Command)
	} else {
		parts := strings.Fields(command.Command)
		if len(parts) == 0 {
			return nil, nil, fmt.Errorf("empty command")
		}
		cmd = exec.Command(parts[0], parts[1:]...)
	}
	if dir, derr := resolveWorkingDir(command); derr == nil && dir != "" {
		cmd.Dir = dir
	} else if derr != nil {
		return nil, nil, derr
	}
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, err
	}
	go func() {
		_, _ = io.Copy(stream, ptmx)
	}()
	return cmd, ptmx, nil
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
