package frameworks

import (
	"fmt"
	"os"
	"os/exec"
)

// ProjectType defines the type of project to generate
type ProjectType string

const (
	TypeREST      ProjectType = "REST"
	TypeCLI       ProjectType = "CLI"
	TypeTUI       ProjectType = "TUI"
	TypeFullStack ProjectType = "FullStack"
)

// ProjectGenerator handles project scaffolding
type ProjectGenerator struct {
	ProjectName string
	Description string
	Port        string
	RootPath    string
	ModuleName  string
	Type        ProjectType
}

// NewProjectGenerator creates a new ProjectGenerator instance
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

// CreateDirectories creates the project directory structure
func (g *ProjectGenerator) CreateDirectories() error {
	var dirs []string

	switch g.Type {
	case TypeREST:
		dirs = g.restDirectories()
	case TypeCLI:
		dirs = g.cliDirectories()
	case TypeTUI:
		dirs = g.tuiDirectories()
	case TypeFullStack:
		dirs = g.fullStackDirectories()
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// GenerateFiles generates all files for the project
func (g *ProjectGenerator) GenerateFiles() error {
	var files map[string]string

	switch g.Type {
	case TypeREST:
		files = g.GenerateRESTFiles()
	case TypeCLI:
		files = g.GenerateCLIFiles()
	case TypeTUI:
		files = g.GenerateTUIFiles()
	case TypeFullStack:
		files = g.GenerateFullStackFiles()
	}

	for path, content := range files {
		if err := g.writeFile(path, content); err != nil {
			return err
		}
	}

	return nil
}

func (g *ProjectGenerator) writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// RunGoModTidy runs go mod tidy in the given project path
func RunGoModTidy(projectPath string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// IsValidProjectName validates the project name
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

// ==================== Shared Template Generators ====================

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
