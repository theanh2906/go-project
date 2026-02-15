package frameworks

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (g *ProjectGenerator) fullStackDirectories() []string {
	return []string{
		g.RootPath,
		filepath.Join(g.RootPath, "internal", "server"),
		filepath.Join(g.RootPath, "internal", "controllers"),
		filepath.Join(g.RootPath, "internal", "services"),
		filepath.Join(g.RootPath, "internal", "models"),
		filepath.Join(g.RootPath, "config"),
		filepath.Join(g.RootPath, "routes"),
		filepath.Join(g.RootPath, "frontend", "src"),
		filepath.Join(g.RootPath, "frontend", "public"),
	}
}

// GenerateFullStackFiles returns all files for a FullStack (React + Go Gin) project
func (g *ProjectGenerator) GenerateFullStackFiles() map[string]string {
	return map[string]string{
		// Go backend files
		filepath.Join(g.RootPath, "main.go"):                                         g.generateFullStackMainFile(),
		filepath.Join(g.RootPath, "go.mod"):                                          g.generateFullStackGoMod(),
		filepath.Join(g.RootPath, ".gitignore"):                                      g.generateFullStackGitignore(),
		filepath.Join(g.RootPath, "README.md"):                                       g.generateFullStackReadme(),
		filepath.Join(g.RootPath, "config", "config.go"):                             g.generateConfigFile(),
		filepath.Join(g.RootPath, "routes", "routes.go"):                             g.generateRoutesFile(),
		filepath.Join(g.RootPath, "internal", "server", "server.go"):                 g.generateFullStackServerFile(),
		filepath.Join(g.RootPath, "internal", "controllers", "health_controller.go"): g.generateHealthController(),
		filepath.Join(g.RootPath, "internal", "controllers", "user_controller.go"):   g.generateUserController(),
		filepath.Join(g.RootPath, "internal", "services", "user_service.go"):         g.generateUserService(),
		filepath.Join(g.RootPath, "internal", "models", "user.go"):                   g.generateUserModel(),
		filepath.Join(g.RootPath, "internal", "models", "response.go"):               g.generateResponseModel(),
		filepath.Join(g.RootPath, "build.bat"):                                       g.generateFullStackBuildBat(),
		filepath.Join(g.RootPath, "Makefile"):                                        g.generateFullStackMakefile(),
		// React frontend files
		filepath.Join(g.RootPath, "frontend", "package.json"):    g.generateReactPackageJSON(),
		filepath.Join(g.RootPath, "frontend", "tsconfig.json"):   g.generateReactTsConfig(),
		filepath.Join(g.RootPath, "frontend", "vite.config.ts"):  g.generateReactViteConfig(),
		filepath.Join(g.RootPath, "frontend", "index.html"):      g.generateReactIndexHTML(),
		filepath.Join(g.RootPath, "frontend", "src", "main.tsx"): g.generateReactMain(),
		filepath.Join(g.RootPath, "frontend", "src", "App.tsx"):  g.generateReactApp(),
		filepath.Join(g.RootPath, "frontend", "src", "App.css"):  g.generateReactAppCSS(),
	}
}

func (g *ProjectGenerator) generateFullStackMainFile() string {
	return fmt.Sprintf(`package main

import (
	"embed"
	"log"

	"%s/config"
	"%s/internal/server"
)

//go:embed frontend/dist/*
var frontendFS embed.FS

func main() {
	cfg := config.LoadConfig()

	srv := server.NewServer(cfg, frontendFS)

	log.Printf("🚀 %s is running on http://localhost:%%s", cfg.Port)
	log.Printf("   Frontend: http://localhost:%%s", cfg.Port)
	log.Printf("   API:      http://localhost:%%s/api", cfg.Port)

	if err := srv.Run(); err != nil {
		log.Fatalf("Failed to start server: %%v", err)
	}
}
`, g.ModuleName, g.ModuleName, g.ProjectName)
}

func (g *ProjectGenerator) generateFullStackGoMod() string {
	return fmt.Sprintf(`module %s

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
)
`, g.ModuleName)
}

func (g *ProjectGenerator) generateFullStackGitignore() string {
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

# Frontend
frontend/node_modules/
frontend/dist/
frontend/.env.local
`
}

func (g *ProjectGenerator) generateFullStackReadme() string {
	return fmt.Sprintf(`# %s

%s

A fullstack application with **React** frontend and **Go (Gin)** backend, bundled into a single executable.

## Architecture

- **Frontend**: React + TypeScript + Vite
- **Backend**: Go + Gin REST API
- **Packaging**: Frontend is embedded into the Go binary via `+"`go:embed`"+`

## Prerequisites

- Go 1.22+
- Node.js 18+
- npm 9+

## Development

### Frontend (dev mode with hot reload)

`+"```bash"+`
cd frontend
npm install
npm run dev
`+"```"+`

Frontend dev server runs on http://localhost:5173 and proxies API calls to Go backend.

### Backend

`+"```bash"+`
go run main.go
`+"```"+`

Backend runs on http://localhost:%s

### Run both in dev mode

1. Start the Go backend: `+"`go run main.go`"+`
2. Start the React dev server: `+"`cd frontend && npm run dev`"+`
3. Open http://localhost:5173

## Build for Production

### Windows

`+"```bat"+`
build.bat
`+"```"+`

### Linux/macOS

`+"```bash"+`
make build
`+"```"+`

This will:
1. Build the React frontend (`+"`npm run build`"+`)
2. Embed the frontend into the Go binary
3. Output a single executable: `+"`%s.exe`"+` (Windows) or `+"`%s`"+` (Linux/macOS)

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /health | Health check |
| GET | /api/users | Get all users |
| GET | /api/users/:id | Get user by ID |
| POST | /api/users | Create user |
| PUT | /api/users/:id | Update user |
| DELETE | /api/users/:id | Delete user |

## License

MIT
`, strings.ToUpper(g.ProjectName), g.Description, g.Port, g.ProjectName, g.ProjectName)
}

func (g *ProjectGenerator) generateFullStackServerFile() string {
	return fmt.Sprintf(`package server

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"%s/config"
	"%s/routes"

	"github.com/gin-gonic/gin"
)

type Server struct {
	config     *config.Config
	router     *gin.Engine
	frontendFS embed.FS
}

func NewServer(cfg *config.Config, frontendFS embed.FS) *Server {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	s := &Server{
		config:     cfg,
		router:     router,
		frontendFS: frontendFS,
	}

	s.setupRoutes()

	return s
}

func (s *Server) setupRoutes() {
	// API routes
	routes.SetupRoutes(s.router)

	// Serve embedded React frontend
	s.serveFrontend()
}

func (s *Server) serveFrontend() {
	// Strip the "frontend/dist" prefix so files are served from root
	distFS, err := fs.Sub(s.frontendFS, "frontend/dist")
	if err != nil {
		panic(fmt.Sprintf("failed to access embedded frontend: %%v", err))
	}

	fileServer := http.FileServer(http.FS(distFS))

	// Serve static files, fallback to index.html for SPA routing
	s.router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Try to serve the file directly
		f, err := distFS.Open(path[1:]) // Remove leading "/"
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Fallback to index.html for SPA client-side routing
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

func (s *Server) Run() error {
	addr := fmt.Sprintf(":%%s", s.config.Port)
	return s.router.Run(addr)
}
`, g.ModuleName, g.ModuleName)
}

func (g *ProjectGenerator) generateFullStackBuildBat() string {
	return fmt.Sprintf(`@echo off
setlocal

echo ========================================
echo  Building %s
echo ========================================
echo.

REM Step 1: Build Frontend
echo [1/3] Installing frontend dependencies...
cd frontend
call npm install
if %%ERRORLEVEL%% neq 0 (
    echo ERROR: Failed to install frontend dependencies
    exit /b 1
)

echo [2/3] Building frontend...
call npm run build
if %%ERRORLEVEL%% neq 0 (
    echo ERROR: Failed to build frontend
    exit /b 1
)
cd ..

REM Step 2: Build Go binary
echo [3/3] Building Go binary...
go build -ldflags="-s -w" -o %s.exe .
if %%ERRORLEVEL%% neq 0 (
    echo ERROR: Failed to build Go binary
    exit /b 1
)

echo.
echo ========================================
echo  Build complete: %s.exe
echo ========================================
echo  Run with: %s.exe
echo ========================================
`, g.ProjectName, g.ProjectName, g.ProjectName, g.ProjectName)
}

func (g *ProjectGenerator) generateFullStackMakefile() string {
	return fmt.Sprintf(`.PHONY: all build clean dev frontend-install frontend-build backend-build

APP_NAME = %s

all: build

# Install frontend dependencies
frontend-install:
	@echo "📦 Installing frontend dependencies..."
	@cd frontend && npm install

# Build frontend
frontend-build: frontend-install
	@echo "⚛️  Building frontend..."
	@cd frontend && npm run build

# Build Go backend (with embedded frontend)
backend-build:
	@echo "🔨 Building Go binary..."
	@go build -ldflags="-s -w" -o $(APP_NAME).exe .

# Full build: frontend + backend
build: frontend-build backend-build
	@echo "✅ Build complete: $(APP_NAME).exe"

# Dev mode - run backend only
dev:
	@go run main.go

# Clean build artifacts
clean:
	@rm -f $(APP_NAME).exe
	@rm -rf frontend/dist
	@echo "🧹 Cleaned build artifacts"
`, g.ProjectName)
}

func (g *ProjectGenerator) generateReactPackageJSON() string {
	return fmt.Sprintf(`{
  "name": "%s-frontend",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.43",
    "@types/react-dom": "^18.2.17",
    "@vitejs/plugin-react": "^4.2.1",
    "typescript": "^5.2.2",
    "vite": "^5.1.0"
  }
}
`, g.ProjectName)
}

func (g *ProjectGenerator) generateReactTsConfig() string {
	return `{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["src"]
}
`
}

func (g *ProjectGenerator) generateReactViteConfig() string {
	return fmt.Sprintf(`import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:%s',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:%s',
        changeOrigin: true,
      },
    },
  },
})
`, g.Port, g.Port)
}

func (g *ProjectGenerator) generateReactIndexHTML() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>%s</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
`, g.ProjectName)
}

func (g *ProjectGenerator) generateReactMain() string {
	return `import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './App.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
`
}

func (g *ProjectGenerator) generateReactApp() string {
	return fmt.Sprintf(`import { useState, useEffect } from 'react'

interface User {
  id: number
  name: string
  email: string
}

interface ApiResponse<T> {
  success: boolean
  data: T
  message?: string
}

function App() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [health, setHealth] = useState('')

  useEffect(() => {
    fetchUsers()
    checkHealth()
  }, [])

  const checkHealth = async () => {
    try {
      const res = await fetch('/health')
      const data = await res.json()
      setHealth(data.status)
    } catch {
      setHealth('disconnected')
    }
  }

  const fetchUsers = async () => {
    try {
      const res = await fetch('/api/users')
      const data: ApiResponse<User[]> = await res.json()
      if (data.success) {
        setUsers(data.data)
      }
    } catch (err) {
      console.error('Failed to fetch users:', err)
    } finally {
      setLoading(false)
    }
  }

  const addUser = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name || !email) return

    try {
      const res = await fetch('/api/users', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, email }),
      })
      const data: ApiResponse<User> = await res.json()
      if (data.success) {
        setUsers([...users, data.data])
        setName('')
        setEmail('')
      }
    } catch (err) {
      console.error('Failed to add user:', err)
    }
  }

  const deleteUser = async (id: number) => {
    try {
      const res = await fetch(`+"`"+`/api/users/${id}`+"`"+`, { method: 'DELETE' })
      const data = await res.json()
      if (data.success) {
        setUsers(users.filter(u => u.id !== id))
      }
    } catch (err) {
      console.error('Failed to delete user:', err)
    }
  }

  return (
    <div className="app">
      <header className="header">
        <h1>⚡ %s</h1>
        <p className="subtitle">React + Go Fullstack Application</p>
        <span className={`+"`"+`health-badge ${health === 'healthy' ? 'healthy' : 'unhealthy'}`+"`"+`}>
          {health === 'healthy' ? '🟢' : '🔴'} API: {health}
        </span>
      </header>

      <main className="main">
        <section className="card">
          <h2>Add User</h2>
          <form onSubmit={addUser} className="form">
            <input
              type="text"
              placeholder="Name"
              value={name}
              onChange={e => setName(e.target.value)}
              className="input"
            />
            <input
              type="email"
              placeholder="Email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              className="input"
            />
            <button type="submit" className="btn">Add User</button>
          </form>
        </section>

        <section className="card">
          <h2>Users {!loading && <span className="count">({users.length})</span>}</h2>
          {loading ? (
            <p className="loading">Loading...</p>
          ) : users.length === 0 ? (
            <p className="empty">No users found</p>
          ) : (
            <ul className="user-list">
              {users.map(user => (
                <li key={user.id} className="user-item">
                  <div>
                    <strong>{user.name}</strong>
                    <span className="user-email">{user.email}</span>
                  </div>
                  <button onClick={() => deleteUser(user.id)} className="btn-delete">
                    Delete
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>
      </main>
    </div>
  )
}

export default App
`, g.ProjectName)
}

func (g *ProjectGenerator) generateReactAppCSS() string {
	return `* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: #0f172a;
  color: #e2e8f0;
  min-height: 100vh;
}

.app {
  max-width: 720px;
  margin: 0 auto;
  padding: 2rem;
}

.header {
  text-align: center;
  margin-bottom: 2rem;
}

.header h1 {
  font-size: 2rem;
  background: linear-gradient(135deg, #818cf8, #c084fc);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin-bottom: 0.5rem;
}

.subtitle {
  color: #94a3b8;
  font-size: 0.9rem;
  margin-bottom: 0.75rem;
}

.health-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 999px;
  font-size: 0.8rem;
  font-weight: 500;
}

.health-badge.healthy {
  background: #064e3b;
  color: #34d399;
}

.health-badge.unhealthy {
  background: #7f1d1d;
  color: #fca5a5;
}

.main {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.card {
  background: #1e293b;
  border-radius: 12px;
  padding: 1.5rem;
  border: 1px solid #334155;
}

.card h2 {
  font-size: 1.1rem;
  margin-bottom: 1rem;
  color: #f1f5f9;
}

.count {
  color: #94a3b8;
  font-weight: normal;
  font-size: 0.9rem;
}

.form {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.input {
  flex: 1;
  min-width: 150px;
  padding: 0.6rem 0.75rem;
  background: #0f172a;
  border: 1px solid #475569;
  border-radius: 8px;
  color: #e2e8f0;
  font-size: 0.9rem;
  outline: none;
  transition: border-color 0.2s;
}

.input:focus {
  border-color: #818cf8;
}

.btn {
  padding: 0.6rem 1.25rem;
  background: #6366f1;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 0.9rem;
  cursor: pointer;
  font-weight: 500;
  transition: background 0.2s;
}

.btn:hover {
  background: #4f46e5;
}

.user-list {
  list-style: none;
}

.user-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  border-bottom: 1px solid #334155;
}

.user-item:last-child {
  border-bottom: none;
}

.user-email {
  display: block;
  color: #94a3b8;
  font-size: 0.85rem;
  margin-top: 0.15rem;
}

.btn-delete {
  padding: 0.35rem 0.75rem;
  background: transparent;
  color: #f87171;
  border: 1px solid #f87171;
  border-radius: 6px;
  font-size: 0.8rem;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-delete:hover {
  background: #f87171;
  color: white;
}

.loading,
.empty {
  text-align: center;
  color: #94a3b8;
  padding: 1rem;
}
`
}
