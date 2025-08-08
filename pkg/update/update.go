package update

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Tomlord1122/tom-recipe/pkg/config"
	"github.com/Tomlord1122/tom-recipe/pkg/model"
	tea "github.com/charmbracelet/bubbletea"
)

// Messages for different events
type (
	ErrorMsg          struct{ Error error }
	ExecuteCommandMsg struct{ Command model.Command }
	CommandResultMsg  struct{ Result Result }
	StreamPollMsg     struct{}
)

// Update handles state transitions based on messages
func Update(msg tea.Msg, m model.Model) (model.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return handleKeyPress(msg, m)
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil
	case ErrorMsg:
		m.Error = msg.Error.Error()
		return m, nil
	case ExecuteCommandMsg:
		return executeCommand(msg.Command, m)
	case CommandResultMsg:
		return handleCommandResult(msg.Result, m)
	case StreamPollMsg:
		return handleStreamPoll(m)
	}

	return m, nil
}

// handleKeyPress processes keyboard input
func handleKeyPress(msg tea.KeyMsg, m model.Model) (model.Model, tea.Cmd) {
	// Clear any previous error messages
	m.Error = ""

	// Check mode-specific handling
	switch m.CurrentMode {
	case model.ModeFilterInput:
		return handleFilterInputMode(msg, m)
	}

	// Form field editing takes priority over all other key handlers
	if m.ShowForm && m.EditingFormField {
		return handleFormFieldEdit(msg, m)
	}

	// Handle global keys
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "h":
		// Only toggle help if not in form mode
		if !m.ShowForm {
			m.ShowHelp = !m.ShowHelp
			return m, nil
		}
	}

	// Handle keys based on current view
	if m.Executing {
		return handleExecutionKeyPress(msg, m)
	}

	if m.ShowHelp {
		return handleHelpKeyPress(msg, m)
	}

	if m.ShowForm {
		return handleFormKeyPress(msg, m)
	}

	return handleMainKeyPress(msg, m)
}

// handleMainKeyPress processes key presses in the main view
func handleMainKeyPress(msg tea.KeyMsg, m model.Model) (model.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedIndex > 0 {
			m.SelectedIndex--
		}
	case "down", "j":
		if m.SelectedIndex < len(m.VisibleCommands)-1 {
			m.SelectedIndex++
		}
	case "enter":
		if len(m.VisibleCommands) > 0 && m.SelectedIndex < len(m.VisibleCommands) {
			return m, func() tea.Msg {
				return ExecuteCommandMsg{Command: m.VisibleCommands[m.SelectedIndex]}
			}
		}
	case "n":
		// Start adding a new command
		m.ShowForm = true
		m.FormCommand = model.Command{
			ID:       "",       // Empty ID indicates new command
			Category: "System", // Default category
			Tags:     []string{},
		}
	case "e":
		// Edit selected command
		if len(m.VisibleCommands) > 0 && m.SelectedIndex < len(m.VisibleCommands) {
			m.ShowForm = true
			m.FormCommand = m.VisibleCommands[m.SelectedIndex]
		}
	case "d":
		// Delete selected command
		if len(m.VisibleCommands) > 0 && m.SelectedIndex < len(m.VisibleCommands) {
			cmdToDelete := m.VisibleCommands[m.SelectedIndex]
			// Remove from all commands
			var newCommands []model.Command
			for _, cmd := range m.AllCommands {
				if cmd.ID != cmdToDelete.ID {
					newCommands = append(newCommands, cmd)
				}
			}
			m.AllCommands = newCommands

			// Apply filter to get updated visible commands
			m.VisibleCommands = filterCommands(m)

			// Adjust selected index if needed
			if m.SelectedIndex >= len(m.VisibleCommands) {
				m.SelectedIndex = len(m.VisibleCommands) - 1
			}
			if m.SelectedIndex < 0 {
				m.SelectedIndex = 0
			}

			// Save updated commands
			if err := config.SaveConfig(m.AllCommands); err != nil {
				m.Error = fmt.Sprintf("Failed to save config: %v", err)
			}
		}
	case "c":
		// Cycle through categories
		found := false
		for i, category := range m.Categories {
			if category == m.ActiveCategory {
				if i < len(m.Categories)-1 {
					m.ActiveCategory = m.Categories[i+1]
				} else {
					m.ActiveCategory = m.Categories[0]
				}
				found = true
				break
			}
		}
		if !found && len(m.Categories) > 0 {
			m.ActiveCategory = m.Categories[0]
		}
		m.VisibleCommands = filterCommands(m)
	case "b":
		// Toggle background mode
		m.RunInBackground = !m.RunInBackground
	case "f":
		// Enter filter mode
		m.CurrentMode = model.ModeFilterInput
		m.InputBuffer = m.FilterText // Start with current filter
		return m, nil
	}

	return m, nil
}

// handleExecutionKeyPress processes key presses in the execution view
func handleExecutionKeyPress(msg tea.KeyMsg, m model.Model) (model.Model, tea.Cmd) {
	// Get the total number of lines in the output
	outputLines := strings.Split(m.ExecutionOutput, "\n")
	totalLines := len(outputLines)

	// Calculate visible lines based on screen height (leave room for headers and footer)
	visibleLines := m.Height - 10
	if visibleLines < 5 {
		visibleLines = 5 // Minimum visible lines
	}

	// Calculate maximum scroll position
	maxScroll := totalLines - visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}

	switch msg.String() {
	case "esc", "q", "enter":
		m.Executing = false
		m.ExecutingCommand = nil
		m.OutputScrollPosition = 0 // Reset scroll position when exiting
	case "up", "k":
		// Scroll up one line
		if m.OutputScrollPosition > 0 {
			m.OutputScrollPosition--
		}
	case "down", "j":
		// Scroll down one line
		if m.OutputScrollPosition < maxScroll {
			m.OutputScrollPosition++
		}
	case "pgup":
		// Scroll up one page (visibleLines - 2 lines to maintain context)
		pageSize := visibleLines - 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.OutputScrollPosition -= pageSize
		if m.OutputScrollPosition < 0 {
			m.OutputScrollPosition = 0
		}
	case "pgdown":
		// Scroll down one page (visibleLines - 2 lines to maintain context)
		pageSize := visibleLines - 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.OutputScrollPosition += pageSize
		if m.OutputScrollPosition > maxScroll {
			m.OutputScrollPosition = maxScroll
		}
	case "home":
		// Scroll to the top
		m.OutputScrollPosition = 0
	case "end":
		// Scroll to the bottom
		m.OutputScrollPosition = maxScroll
	}
	return m, nil
}

// handleHelpKeyPress processes key presses in the help view
func handleHelpKeyPress(msg tea.KeyMsg, m model.Model) (model.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "h":
		m.ShowHelp = false
	}
	return m, nil
}

// handleFormKeyPress processes key presses in the form view
func handleFormKeyPress(msg tea.KeyMsg, m model.Model) (model.Model, tea.Cmd) {
	// If we're editing a field
	if m.EditingFormField {
		return handleFormFieldEdit(msg, m)
	}

	// Otherwise, handle navigation and actions
	switch msg.String() {
	case "esc":
		m.ShowForm = false
		return m, nil
	case "enter":
		// Start editing the current field
		m.EditingFormField = true
		m.FormInputBuffer = m.GetFormFieldValue(m.ActiveFormField)
		return m, nil
	case "y":
		// Save the command
		return saveFormCommand(m)
	case "up", "k":
		// Move to previous field
		if m.ActiveFormField > 0 {
			m.ActiveFormField--
		}
	case "down", "j":
		// Move to next field
		if m.ActiveFormField < model.FieldCount-1 {
			m.ActiveFormField++
		}
	case "tab":
		// Move to next field with wrap-around
		m.ActiveFormField = (m.ActiveFormField + 1) % model.FieldCount
	case "shift+tab":
		// Move to previous field with wrap-around
		if m.ActiveFormField == 0 {
			m.ActiveFormField = model.FieldCount - 1
		} else {
			m.ActiveFormField--
		}
	}

	return m, nil
}

// handleFormFieldEdit handles key presses when editing a form field
func handleFormFieldEdit(msg tea.KeyMsg, m model.Model) (model.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel editing
		m.EditingFormField = false
		m.FormInputBuffer = ""
		return m, nil
	case "enter":
		// Confirm changes
		m.SetFormFieldValue(m.ActiveFormField, m.FormInputBuffer)
		m.EditingFormField = false
		m.FormInputBuffer = ""

		// Move to next field (convenient for quickly filling out the form)
		if m.ActiveFormField < model.FieldCount-1 {
			m.ActiveFormField++
		}
		return m, nil
	case "backspace":
		// Delete last character
		if len(m.FormInputBuffer) > 0 {
			m.FormInputBuffer = m.FormInputBuffer[:len(m.FormInputBuffer)-1]
		}
	case "ctrl+u":
		// Clear entire input
		m.FormInputBuffer = ""
	case "ctrl+a":
		// Select all - not really meaningful in our case, just clear and start fresh
		m.FormInputBuffer = ""
	case "ctrl+e":
		// Not really meaningful in our simple editor, but we're supporting the shortcut for consistency
		// In a real implementation, this would move cursor to end of input
	case "tab":
		// Confirm and move to next field
		m.SetFormFieldValue(m.ActiveFormField, m.FormInputBuffer)
		m.EditingFormField = false
		m.FormInputBuffer = ""
		m.ActiveFormField = (m.ActiveFormField + 1) % model.FieldCount
		return m, nil
	case "shift+tab":
		// Confirm and move to previous field
		m.SetFormFieldValue(m.ActiveFormField, m.FormInputBuffer)
		m.EditingFormField = false
		m.FormInputBuffer = ""
		if m.ActiveFormField == 0 {
			m.ActiveFormField = model.FieldCount - 1
		} else {
			m.ActiveFormField--
		}
		return m, nil
	case "up", "down", "left", "right":
		// Ignore arrow keys in edit mode
		return m, nil
	default:
		// Handle regular key inputs (ignore special keys)
		if len(msg.String()) == 1 || msg.String() == "space" {
			if msg.String() == "space" {
				m.FormInputBuffer += " "
			} else {
				m.FormInputBuffer += msg.String()
			}
		}
	}

	return m, nil
}

// saveFormCommand validates and saves the form command
func saveFormCommand(m model.Model) (model.Model, tea.Cmd) {
	// Validate
	if m.FormCommand.Name == "" || m.FormCommand.Command == "" {
		m.Error = "Name and Command are required fields"
		return m, nil
	}

	// Generate ID if new command
	if m.FormCommand.ID == "" {
		m.FormCommand.ID = fmt.Sprintf("%d", time.Now().Unix())

		// If this is a new command and category is not set, use default
		if m.FormCommand.Category == "" {
			m.FormCommand.Category = "System"
		}

		// If tags are empty, add default tag based on category
		if len(m.FormCommand.Tags) == 0 {
			m.FormCommand.Tags = []string{strings.ToLower(m.FormCommand.Category)}
		}
	}

	// Update or add command
	found := false
	for i, cmd := range m.AllCommands {
		if cmd.ID == m.FormCommand.ID {
			m.AllCommands[i] = m.FormCommand
			found = true
			break
		}
	}

	if !found {
		m.AllCommands = append(m.AllCommands, m.FormCommand)
	}

	// Update categories and visible commands
	m.Categories = config.GetCategories(m.AllCommands)
	m.VisibleCommands = filterCommands(m)

	// Save configuration
	if err := config.SaveConfig(m.AllCommands); err != nil {
		m.Error = fmt.Sprintf("Failed to save config: %v", err)
		return m, nil
	}

	// Exit form mode
	m.ShowForm = false
	return m, nil
}

// executeCommand executes a command and returns the result
func executeCommand(command model.Command, m model.Model) (model.Model, tea.Cmd) {
	// Mark as executing
	m.Executing = true
	m.ExecutingCommand = &command
	m.ExecutionOutput = "Executing command..."
	m.OutputScrollPosition = 0 // Reset scroll position when starting a new command
	m.Error = ""

	// If background mode is enabled, skip the execution
	if m.RunInBackground {
		// create logs dir and file
		logPath, err := createBackgroundLogFile(command)
		if err != nil {
			m.Error = fmt.Sprintf("Failed to create background log: %v", err)
			m.Executing = false
			m.ExecutingCommand = nil
			m.ExecutionOutput = ""
			return m, nil
		}
		go func(p string) {
			f, ferr := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if ferr != nil {
				return
			}
			defer f.Close()
			_ = ExecuteCommandStreaming(command, f)
		}(logPath)
		// show info message
		// Use Error field for now to surface message in UI if Info is not present
		m.Error = fmt.Sprintf("Background task started. Log: %s", logPath)
		m.Executing = false
		m.ExecutingCommand = nil
		m.ExecutionOutput = ""
		return m, nil
	}

	// Interactive command on macOS: open Terminal and return immediately
	if command.Interactive && isDarwin() {
		_ = openInTerminal(command)
		m.Executing = false
		m.ExecutingCommand = nil
		m.ExecutionOutput = ""
		return m, nil
	}

	// Foreground: start streaming to a temp file and poll
	tmpFile, err := os.CreateTemp("", "go-recipe-stream-*.log")
	if err != nil {
		m.Error = fmt.Sprintf("Failed to create temp log: %v", err)
		m.Executing = false
		m.ExecutingCommand = nil
		return m, nil
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	m.ExecutionLogPath = tmpPath
	m.ExecutionLogOffset = 0

	// Command runner returns result when finished
	runCmd := func() tea.Msg {
		f, ferr := os.OpenFile(tmpPath, os.O_WRONLY|os.O_APPEND, 0644)
		if ferr != nil {
			return CommandResultMsg{Result: Result{Command: command, Error: ferr, StartTime: time.Now(), EndTime: time.Now(), ExitCode: -1}}
		}
		defer f.Close()
		res := ExecuteCommandStreaming(command, f)

		// Update command's last run time
		for i, cmd := range m.AllCommands {
			if cmd.ID == command.ID {
				m.AllCommands[i].LastRun = time.Now()
				break
			}
		}
		_ = config.SaveConfig(m.AllCommands)

		return CommandResultMsg{Result: res}
	}

	// Start polling ticks
	poll := tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return StreamPollMsg{} })
	return m, tea.Batch(runCmd, poll)
}

// handleCommandResult processes the result of a command execution
func handleCommandResult(result Result, m model.Model) (model.Model, tea.Cmd) {
	// If we were streaming to a file, read it and compose final output
	if m.ExecutionLogPath != "" {
		content, _ := os.ReadFile(m.ExecutionLogPath)
		result.Output = string(content)
	}
	m.ExecutionOutput = FormatOutput(result)
	m.Executing = false
	m.ExecutingCommand = nil
	m.ExecutionLogPath = ""
	m.ExecutionLogOffset = 0
	return m, nil
}

// filterCommands filters the command list based on category and filter text
func filterCommands(m model.Model) []model.Command {
	var filtered []model.Command

	for _, command := range m.AllCommands {
		// Apply category filter if not "All"
		if m.ActiveCategory != "" && m.ActiveCategory != "All" && command.Category != m.ActiveCategory {
			continue
		}

		// Apply text filter if present
		if m.FilterText != "" {
			lowerFilter := strings.ToLower(m.FilterText)

			// Check name, command, and description
			nameMatch := strings.Contains(strings.ToLower(command.Name), lowerFilter)
			cmdMatch := strings.Contains(strings.ToLower(command.Command), lowerFilter)
			descMatch := strings.Contains(strings.ToLower(command.Description), lowerFilter)

			// Check tags
			tagMatch := false
			for _, tag := range command.Tags {
				if strings.Contains(strings.ToLower(tag), lowerFilter) {
					tagMatch = true
					break
				}
			}

			// Skip if no match found
			if !nameMatch && !cmdMatch && !descMatch && !tagMatch {
				continue
			}
		}

		filtered = append(filtered, command)
	}

	return filtered
}

// handleFilterInputMode handles key presses when in filter input mode
func handleFilterInputMode(msg tea.KeyMsg, m model.Model) (model.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel filtering
		m.CurrentMode = model.ModeNormal
		return m, nil
	case "enter":
		// Apply filter
		m.FilterText = m.InputBuffer
		m.VisibleCommands = filterCommands(m)
		m.CurrentMode = model.ModeNormal
		m.InputBuffer = ""
		return m, nil
	case "backspace":
		// Delete last character
		if len(m.InputBuffer) > 0 {
			m.InputBuffer = m.InputBuffer[:len(m.InputBuffer)-1]
			// Update filter in real time
			m.FilterText = m.InputBuffer
			m.VisibleCommands = filterCommands(m)
		}
	case "ctrl+u":
		// Clear filter
		m.InputBuffer = ""
		m.FilterText = ""
		m.VisibleCommands = filterCommands(m)
	default:
		// Handle regular key inputs
		if len(msg.String()) == 1 || msg.String() == "space" {
			if msg.String() == "space" {
				m.InputBuffer += " "
			} else {
				m.InputBuffer += msg.String()
			}
			// Update filter in real time
			m.FilterText = m.InputBuffer
			m.VisibleCommands = filterCommands(m)
		}
	}

	return m, nil
}

// handleStreamPoll reads new bytes from the temp log and appends to output while executing
func handleStreamPoll(m model.Model) (model.Model, tea.Cmd) {
	if !m.Executing || m.ExecutionLogPath == "" {
		return m, nil
	}
	f, err := os.Open(m.ExecutionLogPath)
	if err != nil {
		return m, tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return StreamPollMsg{} })
	}
	defer f.Close()
	// Seek to last offset
	if m.ExecutionLogOffset > 0 {
		_, _ = f.Seek(m.ExecutionLogOffset, io.SeekStart)
	}
	buf := make([]byte, 64*1024)
	n, _ := f.Read(buf)
	if n > 0 {
		m.ExecutionOutput += string(buf[:n])
		m.ExecutionLogOffset += int64(n)
	}
	// keep polling
	return m, tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return StreamPollMsg{} })
}

// createBackgroundLogFile prepares a log file for background execution output
func createBackgroundLogFile(cmd model.Command) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".go-recipe", "logs")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	ts := time.Now().Format("20060102-150405")
	safeName := strings.ReplaceAll(cmd.Name, " ", "_")
	path := filepath.Join(dir, fmt.Sprintf("%s-%s.log", safeName, ts))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	f.Close()
	return path, nil
}

// isDarwin returns true if running on macOS
func isDarwin() bool {
	goos := runtime.GOOS
	if override := os.Getenv("GOOS_OVERRIDE"); override != "" {
		goos = override
	}
	return strings.ToLower(goos) == "darwin"
}

// openInTerminal opens a new Terminal window on macOS to run the command
func openInTerminal(cmd model.Command) error {
	// Build command string with working dir change if needed
	workDir := ""
	if dir, err := resolveWorkingDir(cmd); err == nil && dir != "" {
		workDir = dir
	}
	body := cmd.Command
	if cmd.UseShell {
		// leave as-is; Terminal will use login shell
	}
	if workDir != "" {
		body = fmt.Sprintf("cd %q; %s", workDir, body)
	}
	// Use AppleScript via osascript
	script := fmt.Sprintf("tell application \"Terminal\" to do script \"%s\"", strings.ReplaceAll(body, "\"", "\\\""))
	execCmd := exec.Command("osascript", "-e", script)
	return execCmd.Run()
}
