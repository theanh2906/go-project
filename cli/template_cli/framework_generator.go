package main

import (
	"fmt"
	"os"
	"os/exec"
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

func RunGoModTidy(projectPath string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func IsValidProjectName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_') {
			return false
		}
	}
	return true
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
	cfg := config.LoadConfig()

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	routes.SetupRoutes(router)

	addr := fmt.Sprintf(":%%s", cfg.Port)
	log.Printf("Server starting on http://localhost:%%s", cfg.Port)
	
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
	return `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out

# Directories
vendor/
bin/
dist/

# Environment
.env
.env.local

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
`
}

func (g *ProjectGenerator) generateRESTReadme() string {
	return fmt.Sprintf(`# %s

%s

## Getting Started

1. Install dependencies:
`+"```bash"+`
go mod tidy
`+"```"+`

2. Run the application:
`+"```bash"+`
go run main.go
`+"```"+`

The server will start on http://localhost:%s

## API Endpoints

- GET /health - Health check
- GET /api/users - Get all users
- GET /api/users/:id - Get user by ID
- POST /api/users - Create user
- PUT /api/users/:id - Update user
- DELETE /api/users/:id - Delete user

## License

MIT
`, strings.ToUpper(g.ProjectName), g.Description, g.Port)
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
	router.GET("/health", controllers.HealthCheck)

	v1 := router.Group("/api")
	{
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

func GetAllUsers(c *gin.Context) {
	users := userService.GetAll()
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    users,
		Message: "Users retrieved successfully",
	})
}

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

type UserService struct {
	users  []*models.User
	nextID int64
	mu     sync.RWMutex
}

func NewUserService() *UserService {
	service := &UserService{
		users:  make([]*models.User, 0),
		nextID: 1,
	}
	
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

func (s *UserService) GetAll() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users
}

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

func (s *UserService) Create(user *models.User) *models.User {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user.ID = s.nextID
	s.nextID++
	s.users = append(s.users, user)
	return user
}

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

type User struct {
	ID    int64  ` + "`json:\"id\"`" + `
	Name  string ` + "`json:\"name\" binding:\"required\"`" + `
	Email string ` + "`json:\"email\" binding:\"required,email\"`" + `
}
`
}

func (g *ProjectGenerator) generateResponseModel() string {
	return `package models

type SuccessResponse struct {
	Success bool        ` + "`json:\"success\"`" + `
	Data    interface{} ` + "`json:\"data,omitempty\"`" + `
	Message string      ` + "`json:\"message,omitempty\"`" + `
}

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
	"os"

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
`, g.ModuleName)
}

func (g *ProjectGenerator) generateCLIReadme() string {
	return fmt.Sprintf(`# %s

%s

## Installation

`+"```bash"+`
go build -o %s
`+"```"+`

## Usage

`+"```bash"+`
./%s --help
./%s greet "World"
`+"```"+`

## License

MIT
`, strings.ToUpper(g.ProjectName), g.Description, g.ProjectName, g.ProjectName, g.ProjectName)
}

func (g *ProjectGenerator) generateCLIRootCmd() string {
	return fmt.Sprintf(`package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "1.0.0"

var rootCmd = &cobra.Command{
	Use:     "%s",
	Short:   "%s",
	Version: Version,
}

var greetCmd = &cobra.Command{
	Use:   "greet [name]",
	Short: "Greet someone",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		fmt.Printf("Hello, %%s!\n", name)
	},
}

func init() {
	rootCmd.AddCommand(greetCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
`, g.ProjectName, g.Description)
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
`, g.ModuleName)
}

func (g *ProjectGenerator) generateTUIReadme() string {
	return fmt.Sprintf(`# %s

%s

## Installation

`+"```bash"+`
go build
./%s
`+"```"+`

## Controls

- **↑/↓** or **k/j** - Navigate
- **Enter** - Select
- **q** or **Ctrl+C** - Quit

## License

MIT
`, strings.ToUpper(g.ProjectName), g.Description, g.ProjectName)
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

	s.WriteString(titleStyle.Render("Interactive TUI App"))
	s.WriteString("\n\n")

	for i, choice := range m.Choices {
		cursor := " "
		if m.Cursor == i {
			cursor = cursorStyle.Render(">")
		}

		checked := " "
		if _, ok := m.Selected[i]; ok {
			checked = "x"
		}

		if m.Cursor == i {
			s.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, selectedStyle.Render(choice)))
		} else {
			s.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice))
		}
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("up/down: Navigate | space: Select | q: Quit"))
	s.WriteString("\n")

	return s.String()
}
`
}
