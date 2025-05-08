package update

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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

	// Create command
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)

	homeDir, err := os.UserHomeDir()
	if err == nil {
		cmd.Dir = homeDir
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()

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
