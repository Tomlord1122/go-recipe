package view

import (
	"fmt"
	"strings"

	"github.com/Tomlord1122/tom-recipe/pkg/model"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Define styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Width(80)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#383838")).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7D56F4")).
				Bold(true).
				Padding(0, 1)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD")).
			Padding(0, 1)

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#36A9E0")).
			Padding(0, 1)

	descriptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4CAF50")).
				Padding(0, 1)

	categoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			Padding(0, 1)

	selectedCategoryStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7D56F4")).
				Bold(true).
				Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true).
			Padding(0, 1)

	outputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Background(lipgloss.Color("#222222")).
			Padding(1, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BBBBBB")).
			Padding(1, 2)
)

// Render renders the UI based on the current model state
func Render(m model.Model) string {
	if m.Executing {
		return renderExecution(m)
	}

	if m.ShowHelp {
		return renderHelp()
	}

	if m.ShowForm {
		return renderForm(m)
	}

	return renderMain(m)
}

// renderMain renders the main command list view
func renderMain(m model.Model) string {
	var sb strings.Builder

	// Render title
	sb.WriteString(titleStyle.Render("go-recipe - command manager"))
	sb.WriteString("\n\n")

	// Render categories
	sb.WriteString("Categories: ")
	for i, category := range m.Categories {
		if category == m.ActiveCategory {
			sb.WriteString(selectedCategoryStyle.Render(category))
		} else {
			sb.WriteString(categoryStyle.Render(category))
		}
		if i < len(m.Categories)-1 {
			sb.WriteString(" | ")
		}
	}
	sb.WriteString("\n\n")

	// Render filter information
	filterTextStyle := commandStyle
	if m.CurrentMode == model.ModeFilterInput {
		filterTextStyle = selectedItemStyle
		sb.WriteString("Filter: ")
		sb.WriteString(filterTextStyle.Render(m.InputBuffer))
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#FF00FF")).
			Render("_"))
		sb.WriteString("\n\n")
	} else if m.FilterText != "" {
		sb.WriteString(fmt.Sprintf("Filter: %s", filterTextStyle.Render(m.FilterText)))
		sb.WriteString("\n\n")
	}

	// Render commands
	if len(m.VisibleCommands) == 0 {
		sb.WriteString(itemStyle.Render("No commands found."))
	} else {
		for i, cmd := range m.VisibleCommands {
			if i == m.SelectedIndex {
				sb.WriteString(selectedItemStyle.Render(fmt.Sprintf("%s (%s)", cmd.Name, cmd.Category)))
				sb.WriteString("\n")
				sb.WriteString(commandStyle.Render(fmt.Sprintf("  Command: %s", cmd.Command)))
				sb.WriteString("\n")
				sb.WriteString(descriptionStyle.Render(fmt.Sprintf("  Description: %s", cmd.Description)))
			} else {
				sb.WriteString(itemStyle.Render(fmt.Sprintf("%s (%s)", cmd.Name, cmd.Category)))
			}
			sb.WriteString("\n")
		}
	}

	// Render error
	if m.Error != "" {
		sb.WriteString("\n")
		sb.WriteString(errorStyle.Render(m.Error))
	}

	// Render help shortcuts
	sb.WriteString("\n\n")

	// Show different help text based on current mode
	if m.CurrentMode == model.ModeFilterInput {
		sb.WriteString(helpStyle.Render("Enter: Apply Filter  |  Esc: Cancel  |  Ctrl+u: Clear Filter"))
	} else {
		sb.WriteString(helpStyle.Render("↑/↓: Navigate  |  Enter: Execute  |  n: New  |  f: Filter  |  c: Category  |  d: Delete  |  h: Help  |  q: Quit"))
	}

	return sb.String()
}

// renderExecution renders the command execution view
func renderExecution(m model.Model) string {
	var sb strings.Builder

	if m.ExecutingCommand == nil {
		sb.WriteString(errorStyle.Render("No command is being executed"))
		return sb.String()
	}

	// Render title
	sb.WriteString(titleStyle.Render(fmt.Sprintf("Executing: %s", m.ExecutingCommand.Name)))
	sb.WriteString("\n\n")

	// Render command info
	sb.WriteString(subtitleStyle.Render(fmt.Sprintf("Command: %s", m.ExecutingCommand.Command)))
	sb.WriteString("\n\n")

	// Handle scrollable output
	outputLines := strings.Split(m.ExecutionOutput, "\n")
	totalLines := len(outputLines)

	// Calculate visible lines based on screen height
	// Leave room for headers and footer (about 10 lines)
	visibleLines := m.Height - 10
	if visibleLines < 5 {
		visibleLines = 5 // Minimum visible lines
	}

	// Calculate max scroll position
	maxScroll := totalLines - visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Adjust scroll position if it's out of bounds
	if m.OutputScrollPosition > maxScroll {
		m.OutputScrollPosition = maxScroll
	}

	// Determine the range of lines to display
	startLine := m.OutputScrollPosition
	endLine := startLine + visibleLines
	if endLine > totalLines {
		endLine = totalLines
	}

	// For very large outputs, show a warning and trimmed content
	const maxProcessableLines = 5000
	showingSummary := false

	if totalLines > maxProcessableLines {
		// For extremely large outputs, we'll show a warning and a subset of lines
		if startLine < 100 {
			// Near the top: show first 100 lines and 100 lines after scroll position
			if endLine > startLine+100 {
				endLine = startLine + 100
			}
		} else if startLine > totalLines-200 {
			// Near the bottom: show last 200 lines
			if startLine < totalLines-200 {
				startLine = totalLines - 200
			}
		} else {
			// In the middle: show 100 lines before and after scroll position
			midPoint := startLine + (endLine-startLine)/2
			startLine = midPoint - 50
			if startLine < 0 {
				startLine = 0
			}
			endLine = midPoint + 50
			if endLine > totalLines {
				endLine = totalLines
			}
		}

		showingSummary = true
	}

	// Show scroll position indicator
	if totalLines > visibleLines {
		scrollPercent := 0.0
		if maxScroll > 0 {
			scrollPercent = float64(startLine) / float64(maxScroll) * 100
		}

		// Create a visual scroll bar
		const scrollBarWidth = 30
		progressChars := 0
		if maxScroll > 0 {
			progressChars = int(float64(scrollBarWidth) * float64(startLine) / float64(maxScroll))
		}
		if progressChars > scrollBarWidth {
			progressChars = scrollBarWidth
		}

		scrollBar := strings.Repeat("█", progressChars) + strings.Repeat("░", scrollBarWidth-progressChars)
		scrollInfo := fmt.Sprintf(" %d/%d lines (%.0f%%)", startLine+1, totalLines, scrollPercent)

		scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)

		sb.WriteString(scrollStyle.Render(scrollBar + scrollInfo))

		if showingSummary {
			sb.WriteString("\n")
			sb.WriteString(errorStyle.Render(fmt.Sprintf("⚠️ Output is very large (%d lines). Showing partial content.", totalLines)))
		}

		sb.WriteString("\n\n")
	}

	// Render visible output lines
	visibleOutput := strings.Join(outputLines[startLine:endLine], "\n")
	sb.WriteString(outputStyle.Render(visibleOutput))

	// Render help shortcuts
	sb.WriteString("\n\n")

	// Add scroll instructions if content is scrollable
	if totalLines > visibleLines {
		sb.WriteString(helpStyle.Render("↑/↓: Scroll  |  PgUp/PgDn: Page Scroll  |  Home/End: Top/Bottom  |  Enter/Esc: Back"))
	} else {
		sb.WriteString(helpStyle.Render("Enter/Esc: Back to list"))
	}

	return sb.String()
}

// renderHelp renders the help view
func renderHelp() string {
	var sb strings.Builder

	// Render title
	sb.WriteString(titleStyle.Render("Help - Keyboard Shortcuts"))
	sb.WriteString("\n\n")

	// Render shortcuts
	shortcuts := []struct {
		key         string
		description string
	}{
		{"↑/↓", "Navigate up and down the command list"},
		{"Enter", "Execute the selected command"},
		{"n", "Add a new command"},
		{"e", "Edit the selected command"},
		{"d", "Delete the selected command"},
		{"f", "Filter commands by name or tags"},
		{"c", "Filter by category"},
		{"h", "Show/hide this help screen"},
		{"b", "Toggle background execution mode"},
		{"q/Esc", "Quit the application"},
	}

	for _, s := range shortcuts {
		sb.WriteString(fmt.Sprintf("%s: %s\n", categoryStyle.Render(s.key), s.description))
	}

	// Render back instruction
	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("Press Esc or h to return to the command list"))

	return sb.String()
}

// renderForm renders the add/edit command form
func renderForm(m model.Model) string {
	var sb strings.Builder

	// Render title
	if m.FormCommand.ID == "" {
		sb.WriteString(titleStyle.Render("Add New Command"))
	} else {
		sb.WriteString(titleStyle.Render("Edit Command"))
	}
	sb.WriteString("\n\n")

	// Create form field styles
	formLabelStyle := categoryStyle
	formValueStyle := commandStyle
	activeFormValueStyle := selectedItemStyle
	editingFormStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#008800")).
		Bold(true)
	formCursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#FF00FF"))

	// Define form fields with their labels and help text
	type formFieldInfo struct {
		label string
		field model.FormField
		help  string
	}

	formFields := []formFieldInfo{
		{"Name", model.FieldName, "Display name for the command"},
		{"Command", model.FieldCommand, "The actual shell command to execute"},
		{"Category", model.FieldCategory, "Category for organization (e.g., System, Network)"},
		{"Description", model.FieldDescription, "Brief description of what the command does"},
		{"Tags", model.FieldTags, "Comma-separated tags for filtering"},
		{"WorkingDirMode", model.FieldWorkingDirMode, "current|home|absolute"},
		{"WorkingDirPath", model.FieldWorkingDirPath, "Used when mode is absolute; supports ~, $HOME, ${cwd}"},
		{"UseShell", model.FieldUseShell, "true/false – run via shell to support pipes and quotes"},
		{"Interactive", model.FieldInteractive, "true/false – run attached (e.g., htop, ssh)"},
	}

	for _, fieldInfo := range formFields {
		isActive := m.ActiveFormField == fieldInfo.field

		// Get the actual value
		value := m.GetFormFieldValue(fieldInfo.field)

		// Render field label (highlight if active)
		if isActive {
			if m.EditingFormField {
				sb.WriteString(editingFormStyle.Render(fieldInfo.label + ": "))
			} else {
				sb.WriteString(selectedCategoryStyle.Render(fieldInfo.label + ": "))
			}
		} else {
			sb.WriteString(formLabelStyle.Render(fieldInfo.label + ": "))
		}

		// Render field value with appropriate styling
		if m.EditingFormField && isActive {
			// When editing, show the input buffer with cursor
			sb.WriteString(editingFormStyle.Render(m.FormInputBuffer))
			sb.WriteString(formCursorStyle.Render("_"))
		} else if value == "" {
			// Show placeholder text for empty fields
			if isActive {
				sb.WriteString(activeFormValueStyle.Render("<" + fieldInfo.help + ">"))
			} else {
				sb.WriteString(lipgloss.NewStyle().
					Foreground(lipgloss.Color("#666666")).
					Render("<" + fieldInfo.help + ">"))
			}
		} else {
			// Show the value with appropriate styling
			if isActive {
				sb.WriteString(activeFormValueStyle.Render(value))
			} else {
				sb.WriteString(formValueStyle.Render(value))
			}
		}

		sb.WriteString("\n")
	}

	// Render error
	if m.Error != "" {
		sb.WriteString("\n")
		sb.WriteString(errorStyle.Render(m.Error))
	}

	// Render help shortcuts and field hints
	sb.WriteString("\n\n")

	if m.EditingFormField {
		sb.WriteString(helpStyle.Render("Enter: Confirm  |  Tab: Next Field  |  Esc: Cancel Edit  |  Ctrl+u: Clear Input"))
	} else {
		sb.WriteString(helpStyle.Render("↑/↓: Navigate Fields  |  Enter: Edit Field  |  Tab: Next Field  |  y: Save  |  Esc: Cancel"))
		sb.WriteString("\n")
		sb.WriteString(descriptionStyle.Render("Fill in the fields above to add your new command."))
	}

	return sb.String()
}
