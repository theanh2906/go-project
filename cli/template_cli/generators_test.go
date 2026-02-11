package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectGenerator(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gcli-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	tests := []struct {
		name        string
		projectType ProjectType
		projectName string
		description string
		port        string
		checkFiles  []string
	}{
		{
			name:        "REST API Project",
			projectType: TypeREST,
			projectName: "test-api",
			description: "Test REST API",
			port:        "8080",
			checkFiles: []string{
				"main.go",
				"go.mod",
				".gitignore",
				"README.md",
				"config/config.go",
				"routes/routes.go",
			},
		},
		{
			name:        "CLI Project",
			projectType: TypeCLI,
			projectName: "test-cli",
			description: "Test CLI",
			port:        "",
			checkFiles: []string{
				"main.go",
				"go.mod",
				".gitignore",
				"README.md",
				"cmd/root.go",
			},
		},
		{
			name:        "TUI Project",
			projectType: TypeTUI,
			projectName: "test-tui",
			description: "Test TUI",
			port:        "",
			checkFiles: []string{
				"main.go",
				"go.mod",
				".gitignore",
				"README.md",
				"internal/ui/ui.go",
				"internal/ui/styles.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewProjectGenerator(tt.projectName, tt.description, tt.port, tt.projectType)

			if err := generator.CreateDirectories(); err != nil {
				t.Fatalf("Failed to create directories: %v", err)
			}

			if err := generator.GenerateFiles(); err != nil {
				t.Fatalf("Failed to generate files: %v", err)
			}

			for _, file := range tt.checkFiles {
				filePath := filepath.Join(tt.projectName, file)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected file not found: %s", filePath)
				}
			}

			os.RemoveAll(tt.projectName)
		})
	}
}

func TestFileGenerator(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gcli-file-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	info := &FileGeneratorInfo{
		ServiceName: "test-service",
		Port:        "3000",
		ModuleName:  "test-service",
		ImageName:   "test-service",
		GoVersion:   "1.21",
	}

	tests := []struct {
		name     string
		fileType FileType
		expected string
	}{
		{
			name:     "Docker Compose",
			fileType: FileTypeDockerCompose,
			expected: "docker-compose.yml",
		},
		{
			name:     "Dockerfile",
			fileType: FileTypeDockerfile,
			expected: "Dockerfile",
		},
		{
			name:     "Jenkinsfile",
			fileType: FileTypeJenkinsfile,
			expected: "Jenkinsfile",
		},
		{
			name:     "Makefile",
			fileType: FileTypeMakefile,
			expected: "Makefile",
		},
		{
			name:     "Gitignore",
			fileType: FileTypeGitignore,
			expected: ".gitignore",
		},
		{
			name:     "Env Example",
			fileType: FileTypeEnvExample,
			expected: ".env.example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.expected)
			generator := NewFileGenerator(filePath, tt.fileType, info)

			content := generator.GetContent()
			if content == "" {
				t.Errorf("Expected content for %s, got empty string", tt.name)
			}

			if err := generator.WriteToFile(); err != nil {
				t.Fatalf("Failed to write file: %v", err)
			}

			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("Expected file not found: %s", filePath)
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			if len(data) == 0 {
				t.Errorf("File %s is empty", filePath)
			}

			fmt.Printf("Generated %s (%d bytes)\n", tt.name, len(data))
		})
	}
}

func TestIsValidProjectName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid lowercase", "myproject", true},
		{"valid with hyphen", "my-project", true},
		{"valid with underscore", "my_project", true},
		{"valid with numbers", "project123", true},
		{"invalid uppercase", "MyProject", false},
		{"invalid space", "my project", false},
		{"invalid special char", "my@project", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidProjectName(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidProjectName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetDefaultFileName(t *testing.T) {
	tests := []struct {
		fileType FileType
		expected string
	}{
		{FileTypeDockerCompose, "docker-compose.yml"},
		{FileTypeDockerfile, "Dockerfile"},
		{FileTypeJenkinsfile, "Jenkinsfile"},
		{FileTypeMakefile, "Makefile"},
		{FileTypeGitignore, ".gitignore"},
		{FileTypeEnvExample, ".env.example"},
	}

	for _, tt := range tests {
		t.Run(string(tt.fileType), func(t *testing.T) {
			result := GetDefaultFileName(tt.fileType)
			if result != tt.expected {
				t.Errorf("GetDefaultFileName(%s) = %s, want %s", tt.fileType, result, tt.expected)
			}
		})
	}
}
