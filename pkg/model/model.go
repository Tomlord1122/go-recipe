package model

import (
	"strings"
	"time"
)

// Command represents a shell command with metadata
type Command struct {
	ID          string    // Unique identifier
	Name        string    // Display name
	Command     string    // The actual command to execute
	Category    string    // Category for organization
	Description string    // Description of what the command does
	Tags        []string  // Tags for filtering
	LastRun     time.Time // When the command was last executed
}

// FormField represents a field in the add/edit form
type FormField int

const (
	FieldName FormField = iota
	FieldCommand
	FieldCategory
	FieldDescription
	FieldTags
	FieldCount // Total number of fields
)

// AppMode represents the different text input modes
type AppMode int

const (
	ModeNormal AppMode = iota
	ModeFilterInput
	ModeFormEdit
)

// Model represents the application state
type Model struct {
	AllCommands     []Command // All available commands
	VisibleCommands []Command // Commands after filtering
	Categories      []string  // Available categories
	SelectedIndex   int       // Currently selected command index
	FilterText      string    // Current filter text
	ActiveCategory  string    // Currently selected category

	// UI State
	RunInBackground      bool     // Whether to run commands in background
	ShowHelp             bool     // Whether help is being displayed
	ShowForm             bool     // Whether add/edit form is displayed
	Executing            bool     // Whether a command is currently executing
	ExecutionOutput      string   // Output of the last executed command
	ExecutingCommand     *Command // Currently executing command
	OutputScrollPosition int      // Scroll position for command output

	// Form state for adding/editing commands
	FormCommand      Command   // Command being edited in form
	ActiveFormField  FormField // Currently active form field
	EditingFormField bool      // Whether we're currently editing a form field
	FormInputBuffer  string    // Buffer for text input

	// Mode state for different input modes
	CurrentMode AppMode // Current app mode
	InputBuffer string  // Text input buffer for various modes

	// Error state
	Error string // Current error message, if any

	// Width and height for responsive design
	Width  int
	Height int
}

// NewModel creates a new model with default values
func NewModel(runInBackground bool) Model {
	return Model{
		AllCommands:          []Command{},
		VisibleCommands:      []Command{},
		Categories:           []string{},
		SelectedIndex:        0,
		FilterText:           "",
		ActiveCategory:       "",
		RunInBackground:      runInBackground,
		ShowHelp:             false,
		ShowForm:             false,
		Executing:            false,
		ExecutionOutput:      "",
		ExecutingCommand:     nil,
		OutputScrollPosition: 0,
		ActiveFormField:      FieldName,
		EditingFormField:     false,
		FormInputBuffer:      "",
		CurrentMode:          ModeNormal,
		InputBuffer:          "",
		Error:                "",
		Width:                80,
		Height:               24,
	}
}

// GetFormFieldValue returns the value for the specified form field
func (m *Model) GetFormFieldValue(field FormField) string {
	switch field {
	case FieldName:
		return m.FormCommand.Name
	case FieldCommand:
		return m.FormCommand.Command
	case FieldCategory:
		return m.FormCommand.Category
	case FieldDescription:
		return m.FormCommand.Description
	case FieldTags:
		// Join tags with commas
		result := ""
		for i, tag := range m.FormCommand.Tags {
			if i > 0 {
				result += ", "
			}
			result += tag
		}
		return result
	default:
		return ""
	}
}

// SetFormFieldValue sets the value for the specified form field
func (m *Model) SetFormFieldValue(field FormField, value string) {
	switch field {
	case FieldName:
		m.FormCommand.Name = value
	case FieldCommand:
		m.FormCommand.Command = value
	case FieldCategory:
		m.FormCommand.Category = value
	case FieldDescription:
		m.FormCommand.Description = value
	case FieldTags:
		// Split comma-separated tags
		m.FormCommand.Tags = []string{}
		if value != "" {
			tags := []string{}
			for _, tag := range strings.Split(value, ",") {
				trimmed := strings.TrimSpace(tag)
				if trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
			m.FormCommand.Tags = tags
		}
	}
}
