package files

func (g *FileGenerator) generateGitignoreFile() string {
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
coverage.html

# Dependency directories
vendor/

# Go workspace file
go.work
go.work.sum

# Build output
bin/
dist/

# Environment variables
.env
.env.local
.env.*.local

# IDE and editors
.idea/
.vscode/
*.swp
*.swo
*~
*.sublime-project
*.sublime-workspace

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Debug files
debug
*.log

# Air (hot reload) temp directory
tmp/

# Docker
.docker/

# Kubernetes
*.kubeconfig
`
}
