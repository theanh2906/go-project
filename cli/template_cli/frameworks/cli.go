package frameworks

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (g *ProjectGenerator) cliDirectories() []string {
	return []string{
		g.RootPath,
		filepath.Join(g.RootPath, "cmd"),
	}
}

// GenerateCLIFiles returns all files for a CLI project
func (g *ProjectGenerator) GenerateCLIFiles() map[string]string {
	return map[string]string{
		filepath.Join(g.RootPath, "main.go"):        g.generateCLIMainFile(),
		filepath.Join(g.RootPath, "go.mod"):         g.generateCLIGoMod(),
		filepath.Join(g.RootPath, ".gitignore"):     g.generateGitignore(),
		filepath.Join(g.RootPath, "README.md"):      g.generateCLIReadme(),
		filepath.Join(g.RootPath, "cmd", "root.go"): g.generateCLIRootCmd(),
	}
}

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
