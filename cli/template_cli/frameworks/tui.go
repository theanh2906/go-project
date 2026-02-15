package frameworks

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (g *ProjectGenerator) tuiDirectories() []string {
	return []string{
		g.RootPath,
		filepath.Join(g.RootPath, "internal", "ui"),
	}
}

// GenerateTUIFiles returns all files for a TUI project
func (g *ProjectGenerator) GenerateTUIFiles() map[string]string {
	return map[string]string{
		filepath.Join(g.RootPath, "main.go"):                     g.generateTUIMainFile(),
		filepath.Join(g.RootPath, "go.mod"):                      g.generateTUIGoMod(),
		filepath.Join(g.RootPath, ".gitignore"):                  g.generateGitignore(),
		filepath.Join(g.RootPath, "README.md"):                   g.generateTUIReadme(),
		filepath.Join(g.RootPath, "internal", "ui", "ui.go"):     g.generateTUIUIFile(),
		filepath.Join(g.RootPath, "internal", "ui", "styles.go"): g.generateTUIStylesFile(),
	}
}

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
