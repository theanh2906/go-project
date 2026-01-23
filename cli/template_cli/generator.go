package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ProjectGenerator struct {
	ProjectName string
	Description string
	Port        string
	RootPath    string
	ModuleName  string
	Type        ProjectType
}

func NewProjectGenerator(name, description, port string, projectType ProjectType) *ProjectGenerator {
	return &ProjectGenerator{
		ProjectName: name,
		Description: description,
		Port:        port,
		RootPath:    name,
		ModuleName:  name,
		Type:        projectType,
	}
}

func (g *ProjectGenerator) CreateDirectories() error {
	var dirs []string

	switch g.Type {
	case TypeREST:
		dirs = []string{
			g.RootPath,
			filepath.Join(g.RootPath, "cmd"),
			filepath.Join(g.RootPath, "internal", "controllers"),
			filepath.Join(g.RootPath, "internal", "services"),
			filepath.Join(g.RootPath, "internal", "models"),
			filepath.Join(g.RootPath, "config"),
			filepath.Join(g.RootPath, "routes"),
		}
	case TypeCLI:
		dirs = []string{
			g.RootPath,
			filepath.Join(g.RootPath, "cmd"),
		}
	case TypeTUI:
		dirs = []string{
			g.RootPath,
			filepath.Join(g.RootPath, "internal", "ui"),
		}
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (g *ProjectGenerator) GenerateFiles() error {
	var files map[string]string

	switch g.Type {
	case TypeREST:
		files = g.generateRESTFiles()
	case TypeCLI:
		files = g.generateCLIFiles()
	case TypeTUI:
		files = g.generateTUIFiles()
	}

	for path, content := range files {
		if err := g.writeFile(path, content); err != nil {
			return err
		}
	}

	return nil
}

func (g *ProjectGenerator) generateRESTFiles() map[string]string {
	return map[string]string{
		filepath.Join(g.RootPath, "main.go"):                                         g.generateRESTMainFile(),
		filepath.Join(g.RootPath, "go.mod"):                                          g.generateRESTGoMod(),
		filepath.Join(g.RootPath, ".gitignore"):                                      g.generateGitignore(),
		filepath.Join(g.RootPath, "README.md"):                                       g.generateRESTReadme(),
		filepath.Join(g.RootPath, "config", "config.go"):                             g.generateConfigFile(),
		filepath.Join(g.RootPath, "routes", "routes.go"):                             g.generateRoutesFile(),
		filepath.Join(g.RootPath, "internal", "controllers", "health_controller.go"): g.generateHealthController(),
		filepath.Join(g.RootPath, "internal", "controllers", "user_controller.go"):   g.generateUserController(),
		filepath.Join(g.RootPath, "internal", "services", "user_service.go"):         g.generateUserService(),
		filepath.Join(g.RootPath, "internal", "models", "user.go"):                   g.generateUserModel(),
		filepath.Join(g.RootPath, "internal", "models", "response.go"):               g.generateResponseModel(),
	}
}

func (g *ProjectGenerator) generateCLIFiles() map[string]string {
	return map[string]string{
		filepath.Join(g.RootPath, "main.go"):        g.generateCLIMainFile(),
		filepath.Join(g.RootPath, "go.mod"):         g.generateCLIGoMod(),
		filepath.Join(g.RootPath, ".gitignore"):     g.generateGitignore(),
		filepath.Join(g.RootPath, "README.md"):      g.generateCLIReadme(),
		filepath.Join(g.RootPath, "cmd", "root.go"): g.generateCLIRootCmd(),
	}
}

func (g *ProjectGenerator) generateTUIFiles() map[string]string {
	return map[string]string{
		filepath.Join(g.RootPath, "main.go"):                     g.generateTUIMainFile(),
		filepath.Join(g.RootPath, "go.mod"):                      g.generateTUIGoMod(),
		filepath.Join(g.RootPath, ".gitignore"):                  g.generateGitignore(),
		filepath.Join(g.RootPath, "README.md"):                   g.generateTUIReadme(),
		filepath.Join(g.RootPath, "internal", "ui", "ui.go"):     g.generateTUIUIFile(),
		filepath.Join(g.RootPath, "internal", "ui", "styles.go"): g.generateTUIStylesFile(),
	}
}

func (g *ProjectGenerator) writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// ==================== REST API Generators ====================

func (g *ProjectGenerator) generateRESTMainFile() string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"log"

	"%s/config"
	"%s/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := gin.Default()

	// Setup routes
	routes.SetupRoutes(router)

	// Start server
	addr := fmt.Sprintf(":%%s", cfg.Port)
	log.Printf("🚀 Server starting on http://localhost:%%s", cfg.Port)
	
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %%v", err)
	}
}
`, g.ModuleName, g.ModuleName)
}

func (g *ProjectGenerator) generateRESTGoMod() string {
	return fmt.Sprintf(`module %s

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
)
`, g.ModuleName)
}

func (g *ProjectGenerator) generateGitignore() string {
	return `# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with go test -c
*.test

# Output of the go coverage tool
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# Environment variables
.env
.env.local

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db
`
}

func (g *ProjectGenerator) generateReadme() string {
	return fmt.Sprintf(`# %s

%s

## Project Structure

%s/
├── cmd/                    # Command-line applications
├── internal/
│   ├── controllers/        # HTTP handlers (like Spring Boot @RestController)
│   ├── services/          # Business logic layer
│   └── models/            # Data models and DTOs
├── config/                # Configuration files
├── routes/                # Route definitions
├── main.go                # Application entry point
└── README.md

## Getting Started

### Prerequisites

- Go 1.21 or higher

### Installation

1. Install dependencies:
%s%s%s
go mod tidy
%s%s%s

2. Run the application:
%s%s%s
go run main.go
%s%s%s

The server will start on http://localhost:%s

## API Endpoints

- GET /health - Health check endpoint
- GET /api/users - Get all users
- GET /api/users/:id - Get user by ID
- POST /api/users - Create a new user
- PUT /api/users/:id - Update user
- DELETE /api/users/:id - Delete user

## Development

### Running in development mode

%s%s%s
go run main.go
%s%s%s

### Building for production

%s%s%s
go build -o app
./app
%s%s%s

## License

MIT
`, strings.ToUpper(g.ProjectName), g.Description, g.ProjectName,
		"```", "bash", "", "", "", "```",
		"```", "bash", "", "", "", "```",
		g.Port,
		"```", "bash", "", "", "", "```",
		"```", "bash", "", "", "", "```")
}

func (g *ProjectGenerator) generateConfigFile() string {
	return fmt.Sprintf(`package config

import "os"

type Config struct {
	Port        string
	Environment string
	AppName     string
}

func LoadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", "%s"),
		Environment: getEnv("ENVIRONMENT", "development"),
		AppName:     "%s",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
`, g.Port, g.ProjectName)
}

func (g *ProjectGenerator) generateRoutesFile() string {
	return fmt.Sprintf(`package routes

import (
	"%s/internal/controllers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine) {
	// Health check endpoint
	router.GET("/health", controllers.HealthCheck)

	// API v1 group
	v1 := router.Group("/api")
	{
		// User routes
		users := v1.Group("/users")
		{
			users.GET("", controllers.GetAllUsers)
			users.GET("/:id", controllers.GetUserByID)
			users.POST("", controllers.CreateUser)
			users.PUT("/:id", controllers.UpdateUser)
			users.DELETE("/:id", controllers.DeleteUser)
		}
	}
}
`, g.ModuleName)
}

func (g *ProjectGenerator) generateHealthController() string {
	return `package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck handles health check requests
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Service is running",
	})
}
`
}

func (g *ProjectGenerator) generateUserController() string {
	return fmt.Sprintf(`package controllers

import (
	"net/http"
	"strconv"

	"%s/internal/models"
	"%s/internal/services"

	"github.com/gin-gonic/gin"
)

var userService = services.NewUserService()

// GetAllUsers retrieves all users
func GetAllUsers(c *gin.Context) {
	users := userService.GetAll()
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    users,
		Message: "Users retrieved successfully",
	})
}

// GetUserByID retrieves a user by ID
func GetUserByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
		return
	}

	user, err := userService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    user,
		Message: "User retrieved successfully",
	})
}

// CreateUser creates a new user
func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	createdUser := userService.Create(&user)
	c.JSON(http.StatusCreated, models.SuccessResponse{
		Success: true,
		Data:    createdUser,
		Message: "User created successfully",
	})
}

// UpdateUser updates an existing user
func UpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
		return
	}

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	user.ID = id
	updatedUser, err := userService.Update(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    updatedUser,
		Message: "User updated successfully",
	})
}

// DeleteUser deletes a user
func DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "Invalid user ID",
		})
		return
	}

	if err := userService.Delete(id); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    nil,
		Message: "User deleted successfully",
	})
}
`, g.ModuleName, g.ModuleName)
}

func (g *ProjectGenerator) generateUserService() string {
	return fmt.Sprintf(`package services

import (
	"errors"
	"sync"

	"%s/internal/models"
)

// UserService handles business logic for users
type UserService struct {
	users  []*models.User
	nextID int64
	mu     sync.RWMutex
}

// NewUserService creates a new UserService instance
func NewUserService() *UserService {
	service := &UserService{
		users:  make([]*models.User, 0),
		nextID: 1,
	}
	
	// Add some sample data
	service.users = append(service.users, &models.User{
		ID:    service.nextID,
		Name:  "John Doe",
		Email: "john@example.com",
	})
	service.nextID++
	
	service.users = append(service.users, &models.User{
		ID:    service.nextID,
		Name:  "Jane Smith",
		Email: "jane@example.com",
	})
	service.nextID++
	
	return service
}

// GetAll returns all users
func (s *UserService) GetAll() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users
}

// GetByID finds a user by ID
func (s *UserService) GetByID(id int64) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, user := range s.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

// Create adds a new user
func (s *UserService) Create(user *models.User) *models.User {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user.ID = s.nextID
	s.nextID++
	s.users = append(s.users, user)
	return user
}

// Update modifies an existing user
func (s *UserService) Update(user *models.User) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for i, u := range s.users {
		if u.ID == user.ID {
			s.users[i] = user
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

// Delete removes a user by ID
func (s *UserService) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for i, user := range s.users {
		if user.ID == id {
			s.users = append(s.users[:i], s.users[i+1:]...)
			return nil
		}
	}
	return errors.New("user not found")
}
`, g.ModuleName)
}

func (g *ProjectGenerator) generateUserModel() string {
	return `package models

// User represents a user in the system
type User struct {
	ID    int64  ` + "`json:\"id\"`" + `
	Name  string ` + "`json:\"name\" binding:\"required\"`" + `
	Email string ` + "`json:\"email\" binding:\"required,email\"`" + `
}
`
}

func (g *ProjectGenerator) generateRESTReadme() string {
	return g.generateReadme()
}

func (g *ProjectGenerator) generateResponseModel() string {
	return `package models

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success bool        ` + "`json:\"success\"`" + `
	Data    interface{} ` + "`json:\"data,omitempty\"`" + `
	Message string      ` + "`json:\"message,omitempty\"`" + `
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Success bool   ` + "`json:\"success\"`" + `
	Error   string ` + "`json:\"error\"`" + `
}
`
}

// ==================== CLI Generators ====================

func (g *ProjectGenerator) generateCLIMainFile() string {
	return fmt.Sprintf(`package main

import (
	"fmt"

	"%s/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
`, g.ModuleName)
}

func (g *ProjectGenerator) generateCLIGoMod() string {
	return fmt.Sprintf(`module %s

go 1.21

require (
	github.com/spf13/cobra v1.8.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)
`, g.ModuleName)
}

func (g *ProjectGenerator) generateCLIReadme() string {
	return fmt.Sprintf(`# %s

%s

## Installation

Build the CLI:

%s%s%s
go build -o %s
%s%s%s

## Usage

%s%s%s
./%s --help
./%s greet "World"
./%s version
%s%s%s

## Project Structure

%s%s%s
%s/
├── cmd/
│   └── root.go         # Root command and subcommands
├── main.go             # Entry point
├── go.mod
└── README.md
%s%s%s

## Adding Commands

Edit [cmd/root.go](cmd/root.go) to add new commands using Cobra.

Example:
%s%s%sgo
var newCmd = &cobra.Command{
    Use:   "new",
    Short: "Create something new",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("Creating new...")
    },
}

func init() {
    rootCmd.AddCommand(newCmd)
}
%s%s%s

## License

MIT
`, strings.ToUpper(g.ProjectName), g.Description,
		"```", "bash", "", g.ProjectName, "", "", "```",
		"```", "bash", "", g.ProjectName, g.ProjectName, g.ProjectName, "", "", "```",
		"```", "", "", g.ProjectName, "", "", "```",
		"```", "", "", "", "", "```")
}

func (g *ProjectGenerator) generateCLIRootCmd() string {
	return fmt.Sprintf(`package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const Version = "1.0.0"

var rootCmd = &cobra.Command{
	Use:     "%s",
	Short:   "%s",
	Version: Version,
	Long:    "%s\n\nA command-line tool built with Cobra.",
}

// Subcommands
var greetCmd = &cobra.Command{
	Use:   "greet [name]",
	Short: "Greet someone",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		fmt.Printf("Hello, %%s! 👋\n", name)
	},
}

func init() {
	rootCmd.AddCommand(greetCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
`, g.ProjectName, g.Description, g.Description)
}

// ==================== TUI Generators ====================

func (g *ProjectGenerator) generateTUIMainFile() string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	"%s/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m := ui.NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %%v\n", err)
		os.Exit(1)
	}
}
`, g.ModuleName)
}

func (g *ProjectGenerator) generateTUIGoMod() string {
	return fmt.Sprintf(`module %s

go 1.21

require (
	github.com/charmbracelet/bubbletea v0.25.0
	github.com/charmbracelet/lipgloss v0.9.1
)

require (
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/containerd/console v1.0.4-0.20230313162750-1ae8d489ac81 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/muesli/ansi v0.0.0-20211018074035-2e021307bc4b // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/term v0.6.0 // indirect
	golang.org/x/text v0.3.8 // indirect
)
`, g.ModuleName)
}

func (g *ProjectGenerator) generateTUIReadme() string {
	return fmt.Sprintf(`# %s

%s

## Installation

Build and run:

%s%s%s
go build
./%s
%s%s%s

Or run directly:

%s%s%s
go run .
%s%s%s

## Project Structure

%s%s%s
%s/
├── internal/
│   ├── ui/
│   │   ├── ui.go           # Main UI logic and update loop
│   │   └── styles.go       # Lipgloss styles
│   └── models/
│       └── model.go        # Data models
├── main.go                  # Entry point
├── go.mod
└── README.md
%s%s%s

## Features

- 🎨 Beautiful terminal UI with Lipgloss
- ⌨️  Interactive navigation with Bubble Tea
- 🚀 Simple and extensible architecture

## Controls

- **↑/↓** or **k/j** - Navigate
- **Enter** - Select
- **q** or **Ctrl+C** - Quit

## Customization

Edit the files in [internal/ui/](internal/ui/) to customize the appearance and behavior.

## License

MIT
`, strings.ToUpper(g.ProjectName), g.Description,
		"```", "bash", "", g.ProjectName, "", "", "```",
		"```", "bash", "", "", "", "```",
		"```", "", "", g.ProjectName, "", "", "```")
}

func (g *ProjectGenerator) generateTUIUIFile() string {
	return `package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Choices  []string
	Cursor   int
	Selected map[int]struct{}
}

func NewModel() Model {
	return Model{
		Choices:  []string{"Option 1", "Option 2", "Option 3", "Quit"},
		Selected: make(map[int]struct{}),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}

		case "down", "j":
			if m.Cursor < len(m.Choices)-1 {
				m.Cursor++
			}

		case "enter", " ":
			// Handle selection
			if m.Cursor == len(m.Choices)-1 {
				return m, tea.Quit
			}
			if _, ok := m.Selected[m.Cursor]; ok {
				delete(m.Selected, m.Cursor)
			} else {
				m.Selected[m.Cursor] = struct{}{}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	return RenderView(m)
}
`
}

func (g *ProjectGenerator) generateTUIStylesFile() string {
	return `package ui

import (
	"fmt"
	"strings"
	
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF00FF")).
		Bold(true).
		Padding(1, 0)

	selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true)

	cursorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FFFF"))

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true)
)

func RenderView(m Model) string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("🎨 Interactive TUI App"))
	s.WriteString("\n\n")

	for i, choice := range m.Choices {
		cursor := " "
		if m.Cursor == i {
			cursor = cursorStyle.Render("▸")
		}

		checked := " "
		if _, ok := m.Selected[i]; ok {
			checked = "✓"
		}

		if m.Cursor == i {
			s.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, selectedStyle.Render(choice)))
		} else {
			s.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice))
		}
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("↑/↓: Navigate • Space: Select • q: Quit"))
	s.WriteString("\n")

	return s.String()
}
`
}
