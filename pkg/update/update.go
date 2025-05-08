package update

import (
	"fmt"
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

	// If background mode is enabled, skip the execution
	if m.RunInBackground {
		go func() {
			_ = ExecuteCommand(command)
		}()
		m.Executing = false
		m.ExecutingCommand = nil
		m.ExecutionOutput = ""
		return m, nil
	}

	// Execute command in foreground
	return m, func() tea.Msg {
		result := ExecuteCommand(command)

		// Update command's last run time
		for i, cmd := range m.AllCommands {
			if cmd.ID == command.ID {
				m.AllCommands[i].LastRun = time.Now()
				break
			}
		}

		// Save configuration
		_ = config.SaveConfig(m.AllCommands)

		return CommandResultMsg{Result: result}
	}
}

// handleCommandResult processes the result of a command execution
func handleCommandResult(result Result, m model.Model) (model.Model, tea.Cmd) {
	m.ExecutionOutput = FormatOutput(result)
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
