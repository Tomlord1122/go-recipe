package main

import (
	"fmt"
	"os"

	"github.com/Tomlord1122/go-recipe/pkg/config"
	"github.com/Tomlord1122/go-recipe/pkg/model"
	"github.com/Tomlord1122/go-recipe/pkg/update"
	"github.com/Tomlord1122/go-recipe/pkg/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Version information - these will be set during build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Command line flags
var runInBackgroundFlag bool

// Application is the main Bubble Tea application
type Application struct {
	model model.Model
}

func (a Application) Init() tea.Cmd {
	return tea.SetWindowTitle("go-recipe")
}

func (a Application) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.model, cmd = update.Update(msg, a.model)
	return a, cmd
}

func (a Application) View() string {
	return view.Render(a.model)
}

// initializeModel loads configuration and sets up the initial model
func initializeModel() (model.Model, error) {
	// Create basic model
	m := model.NewModel(runInBackgroundFlag)

	// Load commands from config
	commands, err := config.LoadConfig()
	if err != nil {
		return m, fmt.Errorf("Failed to load config: %v", err)
	}

	// Set commands and categories
	m.AllCommands = commands
	m.VisibleCommands = commands
	m.Categories = config.GetCategories(commands)

	return m, nil
}

// CLI command structure
var rootCmd = &cobra.Command{
	Use:   "go-recipe",
	Short: "A TUI application for executing commands",
	Long:  `A Terminal User Interface (TUI) application built with Cobra, Bubble Tea, and Lip Gloss.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize the model
		initialModel, err := initializeModel()
		if err != nil {
			fmt.Printf("Error initializing model: %v\n", err)
			os.Exit(1)
		}

		// Set up the application
		app := Application{
			model: initialModel,
		}

		// Run the program
		p := tea.NewProgram(app, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running program: %v\n", err)
			os.Exit(1)
		}
	},
}

// Version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("go-recipe version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("build date: %s\n", date)
	},
}

func main() {
	// Add the background flag to the root command
	// The message will be displayed in the help command
	rootCmd.PersistentFlags().BoolVarP(&runInBackgroundFlag, "background", "b", false,
		"Run selected commands in the background")

	// Add version command
	rootCmd.AddCommand(versionCmd)

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
