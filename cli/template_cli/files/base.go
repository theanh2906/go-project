package files

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileType defines the type of config file to generate
type FileType string

const (
	FileTypeDockerCompose FileType = "docker-compose"
	FileTypeDockerfile    FileType = "dockerfile"
	FileTypeJenkinsfile   FileType = "jenkinsfile"
	FileTypeGitignore     FileType = "gitignore"
	FileTypeEnvExample    FileType = "env-example"
)

// FileGeneratorInfo holds metadata used by file generators
type FileGeneratorInfo struct {
	ServiceName string
	Port        string
	ModuleName  string
	ImageName   string
	GoVersion   string
}

// FileGenerator generates config/devops files
type FileGenerator struct {
	Path     string
	Content  string
	Info     *FileGeneratorInfo
	FileType FileType
}

// NewFileGenerator creates a new FileGenerator instance
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
