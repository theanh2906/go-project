package frameworks

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (g *ProjectGenerator) restDirectories() []string {
	return []string{
		g.RootPath,
		filepath.Join(g.RootPath, "cmd"),
		filepath.Join(g.RootPath, "internal", "controllers"),
		filepath.Join(g.RootPath, "internal", "services"),
		filepath.Join(g.RootPath, "internal", "models"),
		filepath.Join(g.RootPath, "config"),
		filepath.Join(g.RootPath, "routes"),
	}
}

// GenerateRESTFiles returns all files for a REST API project
func (g *ProjectGenerator) GenerateRESTFiles() map[string]string {
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
