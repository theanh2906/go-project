package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type FileType string

const (
	FileTypeDockerCompose FileType = "docker-compose"
	FileTypeDockerfile    FileType = "dockerfile"
	FileTypeJenkinsfile   FileType = "jenkinsfile"
	FileTypeGitignore     FileType = "gitignore"
	FileTypeEnvExample    FileType = "env-example"
)

type FileGeneratorInfo struct {
	ServiceName string
	Port        string
	ModuleName  string
	ImageName   string
	GoVersion   string
}

type FileGenerator struct {
	Path     string
	Content  string
	Info     *FileGeneratorInfo
	FileType FileType
}

func NewFileGenerator(path string, fileType FileType, info *FileGeneratorInfo) *FileGenerator {
	return &FileGenerator{
		Path:     path,
		Info:     info,
		FileType: fileType,
	}
}

// Generate generates the content for the specified file type
func (g *FileGenerator) Generate() string {
	switch g.FileType {
	case FileTypeDockerCompose:
		return g.generateDockerComposeFile()
	case FileTypeDockerfile:
		return g.generateDockerfile()
	case FileTypeJenkinsfile:
		return g.generateJenkinsfile()
	case FileTypeGitignore:
		return g.generateGitignoreFile()
	case FileTypeEnvExample:
		return g.generateEnvExample()
	default:
		return ""
	}
}

// WriteToFile writes the generated content to a file
func (g *FileGenerator) WriteToFile() error {
	content := g.Generate()
	if content == "" {
		return fmt.Errorf("no content generated for file type: %s", g.FileType)
	}

	// Ensure directory exists
	dir := filepath.Dir(g.Path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	return os.WriteFile(g.Path, []byte(content), 0644)
}

// GetContent returns the generated content without writing to file
func (g *FileGenerator) GetContent() string {
	return g.Generate()
}

// ==================== Docker Compose Generator ====================

func (g *FileGenerator) generateDockerComposeFile() string {
	serviceName := g.Info.ServiceName
	if serviceName == "" {
		serviceName = "app"
	}
	port := g.Info.Port
	if port == "" {
		port = "8080"
	}
	imageName := g.Info.ImageName
	if imageName == "" {
		imageName = serviceName
	}

	return fmt.Sprintf(`version: '3.8'

services:
  %s:
    image: %s
    container_name: %s
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "%s:%s"
    environment:
      - PORT=%s
      - ENVIRONMENT=production
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:%s/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    networks:
      - app-network

networks:
  app-network:
    driver: bridge

volumes:
  app-data:
`, serviceName, imageName, serviceName, port, port, port, port)
}

// ==================== Dockerfile Generator ====================

func (g *FileGenerator) generateDockerfile() string {
	goVersion := g.Info.GoVersion
	if goVersion == "" {
		goVersion = "1.21"
	}
	port := g.Info.Port
	if port == "" {
		port = "8080"
	}

	return fmt.Sprintf(`# Build stage
FROM golang:%s-alpine AS builder

# Install git and ca-certificates (needed for fetching dependencies)
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /app/main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE %s

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:%s/health || exit 1

# Run the application
ENTRYPOINT ["./main"]
`, goVersion, port, port)
}

// ==================== Jenkinsfile Generator ====================

func (g *FileGenerator) generateJenkinsfile() string {
	serviceName := g.Info.ServiceName
	if serviceName == "" {
		serviceName = "app"
	}
	imageName := g.Info.ImageName
	if imageName == "" {
		imageName = serviceName
	}

	return fmt.Sprintf(`pipeline {
    agent any
    
    environment {
        APP_NAME = '%s'
        DOCKER_IMAGE = '%s'
        DOCKER_REGISTRY = 'your-registry.com'
        GO_VERSION = '1.21'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Setup Go') {
            steps {
                sh '''
                    go version
                    go mod download
                '''
            }
        }
        
        stage('Lint') {
            steps {
                sh '''
                    go vet ./...
                '''
            }
        }
        
        stage('Test') {
            steps {
                sh '''
                    go test -v -race -coverprofile=coverage.out ./...
                    go tool cover -html=coverage.out -o coverage.html
                '''
            }
            post {
                always {
                    archiveArtifacts artifacts: 'coverage.html', allowEmptyArchive: true
                }
            }
        }
        
        stage('Build') {
            steps {
                sh '''
                    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
                        -ldflags='-w -s' \
                        -o ${APP_NAME} .
                '''
            }
        }
        
        stage('Docker Build') {
            when {
                branch 'main'
            }
            steps {
                script {
                    def imageTag = "${DOCKER_REGISTRY}/${DOCKER_IMAGE}:${BUILD_NUMBER}"
                    sh "docker build -t ${imageTag} ."
                    sh "docker tag ${imageTag} ${DOCKER_REGISTRY}/${DOCKER_IMAGE}:latest"
                }
            }
        }
        
        stage('Docker Push') {
            when {
                branch 'main'
            }
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'docker-registry-credentials',
                    usernameVariable: 'DOCKER_USER',
                    passwordVariable: 'DOCKER_PASS'
                )]) {
                    sh '''
                        echo $DOCKER_PASS | docker login $DOCKER_REGISTRY -u $DOCKER_USER --password-stdin
                        docker push ${DOCKER_REGISTRY}/${DOCKER_IMAGE}:${BUILD_NUMBER}
                        docker push ${DOCKER_REGISTRY}/${DOCKER_IMAGE}:latest
                    '''
                }
            }
        }
        
        stage('Deploy') {
            when {
                branch 'main'
            }
            steps {
                echo 'Deploying to production...'
            }
        }
    }
    
    post {
        always {
            cleanWs()
        }
        success {
            echo 'Pipeline completed successfully!'
        }
        failure {
            echo 'Pipeline failed!'
        }
    }
}
`, serviceName, imageName)
}

// ==================== Gitignore Generator ====================

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

// ==================== Env Example Generator ====================

func (g *FileGenerator) generateEnvExample() string {
	port := g.Info.Port
	if port == "" {
		port = "8080"
	}
	serviceName := g.Info.ServiceName
	if serviceName == "" {
		serviceName = "app"
	}

	return fmt.Sprintf(`# Application
APP_NAME=%s
ENVIRONMENT=development
PORT=%s

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb
DB_USER=postgres
DB_PASSWORD=your_password_here
DB_SSL_MODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=your_jwt_secret_here
JWT_EXPIRY=24h

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json

# External Services
API_KEY=your_api_key_here
API_URL=https://api.example.com
`, serviceName, port)
}

// ==================== Helper Functions ====================

// GetAvailableFileTypes returns all available file types
func GetAvailableFileTypes() []FileType {
	return []FileType{
		FileTypeDockerCompose,
		FileTypeDockerfile,
		FileTypeJenkinsfile,
		FileTypeGitignore,
		FileTypeEnvExample,
	}
}

// GetFileTypeDescription returns a description for each file type
func GetFileTypeDescription(ft FileType) string {
	descriptions := map[FileType]string{
		FileTypeDockerCompose: "Docker Compose configuration file",
		FileTypeDockerfile:    "Dockerfile for containerization",
		FileTypeJenkinsfile:   "Jenkins CI/CD pipeline",
		FileTypeGitignore:     ".gitignore file for Go projects",
		FileTypeEnvExample:    ".env.example template file",
	}
	return descriptions[ft]
}

// GetDefaultFileName returns the default file name for each file type
func GetDefaultFileName(ft FileType) string {
	names := map[FileType]string{
		FileTypeDockerCompose: "docker-compose.yml",
		FileTypeDockerfile:    "Dockerfile",
		FileTypeJenkinsfile:   "Jenkinsfile",
		FileTypeGitignore:     ".gitignore",
		FileTypeEnvExample:    ".env.example",
	}
	return names[ft]
}
