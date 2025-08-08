# Go Recipe - Command Manager

Go Recipe is a modern Terminal User Interface (TUI) application for managing and executing shell commands. It allows you to organize commands into categories, add new commands, and execute them either in foreground or background mode.

## Features

- Organize commands with categories and tags
- Filter commands by category or text search
- Execute commands and view output
- Background execution mode
- Add, edit, and delete commands
- Persistent configuration

## Installation

### Prerequisites

- Go 1.18 or higher

### Install via go install

```bash
go install github.com/Tomlord1122/go-recipe/cmd/go-recipe@latest
```

### Building from source

```bash
# Clone the repository
git clone https://github.com/Tomlord1122/go-recipe.git
cd go-recipe

# Build the application
make build
```


## Usage

Run the application:

```bash
cd build
./go-recipe
```

### Keyboard Shortcuts

- `↑/↓` or `k/j`: Navigate up and down the command list
- `Enter`: Execute the selected command
- `n`: Add a new command
- `e`: Edit the selected command
- `d`: Delete the selected command
- `f`: Filter commands by name
- `c`: Cycle through categories
- `h`: Show/hide help screen
- `b`: Toggle background execution mode
- `q/Esc`: Quit the application

## Architecture

The application follows the Model-View-Update (MVU) architecture pattern:

1. **Model**: Core data structures and state management
2. **View**: UI rendering based on current state
3. **Update**: State transitions and event handling

## Libraries Used

- [Bubble Tea](https://github.com/charmbracelet/bubbletea): TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss): Terminal styling
- [Cobra](https://github.com/spf13/cobra): CLI framework

## Configuration

The application stores your commands in JSON format at:

```
~/.go-recipe/commands.json
```

## License

MIT License

