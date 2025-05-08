package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Tomlord1122/tom-recipe/pkg/model"
)

const (
	configDir  = ".go-recipe"
	configFile = "commands.json"
)

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDirPath := filepath.Join(homeDir, configDir)
	if err := os.MkdirAll(configDirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDirPath, configFile), nil
}

// LoadConfig loads commands from the config file
func LoadConfig() ([]model.Command, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If the file doesn't exist yet, return an empty array
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default commands for first-time users
		defaultCommands := getDefaultCommands()
		if err := SaveConfig(defaultCommands); err != nil {
			return nil, err
		}
		return defaultCommands, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var commands []model.Command
	if err := json.Unmarshal(data, &commands); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return commands, nil
}

// SaveConfig saves commands to the config file
func SaveConfig(commands []model.Command) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(commands, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal commands: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCategories extracts unique categories from commands
func GetCategories(commands []model.Command) []string {
	// Use a map to track unique categories
	categoryMap := map[string]bool{}
	for _, cmd := range commands {
		if cmd.Category != "" {
			categoryMap[cmd.Category] = true
		}
	}

	// Convert map keys to slice
	categories := []string{"All"} // Always include "All" category
	for category := range categoryMap {
		categories = append(categories, category)
	}

	return categories
}

// getDefaultCommands returns a set of default commands for first-time users
func getDefaultCommands() []model.Command {
	if runtime.GOOS == "darwin" {
		return getDarwinCommands()
	}

	// Default to Linux commands
	return getLinuxCommands()
}

// getDarwinCommands returns macOS-specific commands
func getDarwinCommands() []model.Command {
	return []model.Command{
		{
			ID:          "1",
			Name:        "Disk Space",
			Command:     "df -h",
			Category:    "System",
			Description: "Shows disk space usage",
			Tags:        []string{"system", "disk"},
			LastRun:     time.Time{},
		},
		{
			ID:          "2",
			Name:        "Memory Usage",
			Command:     "vm_stat",
			Category:    "System",
			Description: "Shows virtual memory statistics",
			Tags:        []string{"system", "memory"},
			LastRun:     time.Time{},
		},
		{
			ID:          "3",
			Name:        "Network Interfaces",
			Command:     "ifconfig",
			Category:    "Network",
			Description: "Lists network interfaces",
			Tags:        []string{"network"},
			LastRun:     time.Time{},
		},
		{
			ID:          "4",
			Name:        "System Info",
			Command:     "system_profiler SPSoftwareDataType SPHardwareDataType",
			Category:    "System",
			Description: "Shows system hardware and software information",
			Tags:        []string{"system", "hardware"},
			LastRun:     time.Time{},
		},
		{
			ID:          "5",
			Name:        "Running Processes",
			Command:     "ps aux",
			Category:    "System",
			Description: "Shows all running processes",
			Tags:        []string{"system", "process"},
			LastRun:     time.Time{},
		},
		{
			ID:          "6",
			Name:        "Network Stats",
			Command:     "netstat -an",
			Category:    "Network",
			Description: "Shows network statistics",
			Tags:        []string{"network"},
			LastRun:     time.Time{},
		},
		{
			ID:          "7",
			Name:        "CPU Info",
			Command:     "sysctl -n machdep.cpu.brand_string",
			Category:    "System",
			Description: "Shows CPU information",
			Tags:        []string{"system", "cpu"},
			LastRun:     time.Time{},
		},
	}
}

// getLinuxCommands returns Linux-specific commands
func getLinuxCommands() []model.Command {
	return []model.Command{
		{
			ID:          "1",
			Name:        "Disk Space",
			Command:     "df -h",
			Category:    "System",
			Description: "Shows disk space usage",
			Tags:        []string{"system", "disk"},
			LastRun:     time.Time{},
		},
		{
			ID:          "2",
			Name:        "Memory Usage",
			Command:     "free -h",
			Category:    "System",
			Description: "Shows memory usage",
			Tags:        []string{"system", "memory"},
			LastRun:     time.Time{},
		},
		{
			ID:          "3",
			Name:        "Network Interfaces",
			Command:     "ifconfig",
			Category:    "Network",
			Description: "Lists network interfaces",
			Tags:        []string{"network"},
			LastRun:     time.Time{},
		},
		{
			ID:          "4",
			Name:        "System Info",
			Command:     "uname -a",
			Category:    "System",
			Description: "Shows system information",
			Tags:        []string{"system"},
			LastRun:     time.Time{},
		},
		{
			ID:          "5",
			Name:        "Running Processes",
			Command:     "ps aux",
			Category:    "System",
			Description: "Shows all running processes",
			Tags:        []string{"system", "process"},
			LastRun:     time.Time{},
		},
		{
			ID:          "6",
			Name:        "Network Stats",
			Command:     "netstat -tuln",
			Category:    "Network",
			Description: "Shows network statistics",
			Tags:        []string{"network"},
			LastRun:     time.Time{},
		},
		{
			ID:          "7",
			Name:        "CPU Info",
			Command:     "lscpu",
			Category:    "System",
			Description: "Shows CPU information",
			Tags:        []string{"system", "cpu"},
			LastRun:     time.Time{},
		},
	}
}
