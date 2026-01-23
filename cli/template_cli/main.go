package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

const VERSION = "1.0.0"

type ProjectType string

const (
	TypeREST ProjectType = "REST"
	TypeCLI  ProjectType = "CLI"
	TypeTUI  ProjectType = "TUI"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "new", "n":
		if len(os.Args) < 3 {
			color.Red("✗ Project name is required")
			fmt.Println("\nUsage: go-cli new <project-name> [--type TYPE]")
			os.Exit(1)
		}

		// Parse flags
		newCmd := flag.NewFlagSet("new", flag.ExitOnError)
		projectType := newCmd.String("type", "", "Project type: REST, CLI, or TUI")
		newCmd.Parse(os.Args[3:])

		projectName := os.Args[2]
		createNewProject(projectName, *projectType)
	case "version", "-v", "--version":
		fmt.Printf("Go CLI version %s\n", VERSION)
	case "help", "-h", "--help":
		printHelp()
	default:
		color.Red("✗ Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println()
	color.Cyan("Go CLI - A scaffolding tool for Go projects")
	fmt.Println()
	color.Yellow("Usage:")
	fmt.Println("  go-cli <command> [options]")
	fmt.Println()
	color.Yellow("Commands:")
	fmt.Println("  new|n <name> [--type TYPE]   Generate a new Go project")
	fmt.Println("  version|-v                   Show CLI version")
	fmt.Println("  help|-h                      Show this help message")
	fmt.Println()
	color.Yellow("Flags:")
	fmt.Println("  --type TYPE      Project type: REST, CLI, or TUI (optional)")
	fmt.Println()
	color.Yellow("Examples:")
	fmt.Println("  go-cli new my-api                    # Interactive menu to choose type")
	fmt.Println("  go-cli new my-api --type REST        # Generate REST API project")
	fmt.Println("  go-cli new my-tool --type CLI        # Generate CLI tool project")
	fmt.Println("  go-cli new my-tui --type TUI         # Generate TUI application")
	fmt.Println()
}

func createNewProject(projectName string, projectTypeFlag string) {
	// Validate project name
	if !isValidProjectName(projectName) {
		color.Red("✗ Invalid project name. Use only lowercase letters, numbers, and hyphens")
		os.Exit(1)
	}

	// Determine project type
	var projectType ProjectType
	if projectTypeFlag != "" {
		// Validate and use provided type
		typeUpper := strings.ToUpper(projectTypeFlag)
		switch typeUpper {
		case "REST", "CLI", "TUI":
			projectType = ProjectType(typeUpper)
		default:
			color.Red("✗ Invalid project type: %s", projectTypeFlag)
			fmt.Println("  Valid types: REST, CLI, TUI")
			os.Exit(1)
		}
	} else {
		// Interactive menu
		projectType = selectProjectType()
	}

	color.Cyan("\n🚀 Creating a new Go %s project...\n", projectType)

	// Type-specific prompts
	var description, port string

	if projectType == TypeREST {
		prompt := promptui.Prompt{
			Label:   "Project description",
			Default: "A Go REST API using Gin framework",
		}
		description, _ = prompt.Run()

		portPrompt := promptui.Prompt{
			Label:   "Server port",
			Default: "8080",
		}
		port, _ = portPrompt.Run()
	} else {
		prompt := promptui.Prompt{
			Label:   "Project description",
			Default: fmt.Sprintf("A Go %s application", projectType),
		}
		description, _ = prompt.Run()
	}

	// Generate project structure
	generator := NewProjectGenerator(projectName, description, port, projectType)

	color.Green("\n✓ Creating project structure...")
	if err := generator.CreateDirectories(); err != nil {
		color.Red("✗ Failed to create directories: %v", err)
		os.Exit(1)
	}

	color.Green("✓ Generating files...")
	if err := generator.GenerateFiles(); err != nil {
		color.Red("✗ Failed to generate files: %v", err)
		os.Exit(1)
	}

	color.Green("✓ Installing dependencies...")
	if err := runGoModTidy(projectName); err != nil {
		color.Yellow("⚠ Warning: Failed to run 'go mod tidy': %v", err)
		color.Yellow("  You may need to run 'go mod tidy' manually")
	} else {
		color.Green("✓ Dependencies installed successfully!")
	}

	// Success message
	fmt.Println()
	color.Green("✓ Project created successfully!")
	fmt.Println()
	color.Cyan("Next steps:")
	fmt.Printf("  cd %s\n", projectName)
	if projectType == TypeREST {
		fmt.Println("  go run main.go")
	} else {
		fmt.Println("  go run .")
	}
	fmt.Println()
	color.Yellow("Happy coding! 🎉")
	fmt.Println()
}

func selectProjectType() ProjectType {
	prompt := promptui.Select{
		Label: "Select project type",
		Items: []string{
			"REST - REST API server with Gin framework",
			"CLI  - Command-line tool with Cobra",
			"TUI  - Terminal UI app with Bubble Tea",
		},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | cyan | bold }}",
			Active:   "▸ {{ . | cyan | bold }}",
			Inactive: "  {{ . }}",
			Selected: "✓ {{ . | green | bold }}",
		},
	}

	idx, _, err := prompt.Run()
	if err != nil {
		color.Red("✗ Selection cancelled")
		os.Exit(1)
	}

	types := []ProjectType{TypeREST, TypeCLI, TypeTUI}
	return types[idx]
}

func isValidProjectName(name string) bool {
	if len(name) == 0 {
		return false
	}
	// Check if name contains only valid characters
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_') {
			return false
		}
	}
	return true
}

func runGoModTidy(projectPath string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
