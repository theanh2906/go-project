# Go CLI Generator 🛠️

A beautiful TUI (Terminal User Interface) tool for generating Go projects and configuration files.

## Features

✨ **Interactive TUI** - Beautiful terminal interface with Bubble Tea  
🚀 **Project Generation** - Generate Go projects (REST API, CLI, TUI)  
📄 **File Generation** - Generate config files (Dockerfile, docker-compose, Jenkinsfile, Makefile, .gitignore, .env.example)  
📋 **Clipboard Support** - Copy generated content directly to clipboard  
🎨 **Beautiful Design** - Colorful and modern terminal UI with lipgloss

## Installation

Build the CLI tool:

```bash
cd cli/template_cli
go build -o gcli.exe .
```

Or install it globally:

```bash
go install
```

## Usage

Simply run the executable to start the interactive TUI:

```bash
./gcli.exe
```

### Navigation

- **↑/↓** or **k/j**: Navigate through menu items
- **Enter**: Select option / Confirm input
- **Esc**: Go back to previous screen
- **Ctrl+C**: Quit application
- **y/n**: Quick select Yes/No in confirmation dialogs

## Main Menu Options

### 1. 🚀 Generate Go Project

Create a new Go project with one of the following types:

| Type | Description | Framework |
|------|-------------|-----------|
| **REST API** | REST API server | Gin |
| **CLI Tool** | Command-line tool | Cobra |
| **TUI App** | Terminal UI application | Bubble Tea |

#### Project Generation Flow:
1. Select project type
2. Enter project name
3. Enter description (optional)
4. Enter port (REST API only)
5. Project is generated!

### 2. 📄 Generate Config File

Generate configuration files for your project:

| File Type | Output File | Description |
|-----------|-------------|-------------|
| Docker Compose | `docker-compose.yml` | Docker Compose configuration |
| Dockerfile | `Dockerfile` | Multi-stage Dockerfile for Go |
| Jenkinsfile | `Jenkinsfile` | Jenkins CI/CD pipeline |
| Makefile | `Makefile` | Build automation |
| .gitignore | `.gitignore` | Git ignore for Go projects |
| .env.example | `.env.example` | Environment variables template |

#### File Generation Flow:
1. Select file type
2. Enter service name
3. Enter port
4. Choose: Copy to clipboard only? (Yes/No)
   - **Yes**: Content copied to clipboard
   - **No**: Enter file path (default: current directory)
5. File is generated!

## Project Structure

```
template_cli/
├── main.go                 # TUI application entry point
├── types.go                # Shared types (ProjectType)
├── framework_generator.go  # Go project generation logic
├── file_generator.go       # Config file generation logic
├── go.mod
├── go.sum
└── README.md
```

## Generated Project Structures

### REST API Project
```
my-api/
├── main.go
├── go.mod
├── .gitignore
├── README.md
├── config/
│   └── config.go
├── routes/
│   └── routes.go
└── internal/
    ├── controllers/
    │   ├── health_controller.go
    │   └── user_controller.go
    ├── services/
    │   └── user_service.go
    └── models/
        ├── user.go
        └── response.go
```

### CLI Project
```
my-cli/
├── main.go
├── go.mod
├── .gitignore
├── README.md
└── cmd/
    └── root.go
```

### TUI Project
```
my-tui/
├── main.go
├── go.mod
├── .gitignore
├── README.md
└── internal/
    └── ui/
        ├── ui.go
        └── styles.go
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [clipboard](https://github.com/atotto/clipboard) - Clipboard support

## Screenshots

```
🛠️  Go CLI Generator

v2.0.0 - Generate Go projects and config files

What would you like to do?

▸ 🚀 Generate Go Project
  📄 Generate Config File
  ℹ️  About
  🚪 Exit

↑/↓: Navigate • Enter: Select • Esc: Back • Ctrl+C: Quit
```

## Version

Current version: **2.0.0**

## License

MIT
