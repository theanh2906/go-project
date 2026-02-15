package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-cli/files"
	"go-cli/frameworks"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const VERSION = "2.0.0"

var (
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	accentColor    = lipgloss.Color("#F59E0B")
	errorColor     = lipgloss.Color("#EF4444")
	mutedColor     = lipgloss.Color("#6B7280")

	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(1, 0).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true).
				PaddingLeft(2)

	cursorStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(2)

	successStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	errorMsgStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			MarginTop(1)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginBottom(1)

	dimStyle = lipgloss.NewStyle().
			Foreground(mutedColor)
)

type appState int

const (
	stateMainMenu appState = iota
	stateSelectProjectType
	stateProjectNameInput
	stateProjectDescInput
	stateProjectPortInput
	stateSelectFileType
	stateFileServiceNameInput
	stateFilePortInput
	stateCopyOnlyConfirm
	stateFilePathInput
	stateGenerating
	stateSuccess
	stateError
)

type model struct {
	state            appState
	cursor           int
	textInput        textinput.Model
	spinner          spinner.Model
	err              error
	message          string
	width            int
	height           int
	projectType      frameworks.ProjectType
	projectName      string
	projectDesc      string
	projectPort      string
	fileType         files.FileType
	fileServiceName  string
	filePort         string
	copyOnly         bool
	filePath         string
	generatedContent string
}

var mainMenuItems = []string{
	"🚀 Generate Go Project",
	"📄 Generate Config File",
	"ℹ️  About",
	"🚪 Exit",
}

var projectTypeItems = []struct {
	Type frameworks.ProjectType
	Name string
	Desc string
	Icon string
}{
	{frameworks.TypeREST, "REST API", "REST API server with Gin framework", "🌐"},
	{frameworks.TypeCLI, "CLI Tool", "Command-line tool with Cobra", "⌨️"},
	{frameworks.TypeTUI, "TUI App", "Terminal UI app with Bubble Tea", "🎨"},
	{frameworks.TypeFullStack, "FullStack App", "React + Go Gin, bundled into single exe", "⚡"},
}

var fileTypeItems = []struct {
	Type files.FileType
	Name string
	Desc string
	Icon string
}{
	{files.FileTypeDockerCompose, "Docker Compose", "docker-compose.yml configuration", "🐳"},
	{files.FileTypeDockerfile, "Dockerfile", "Multi-stage Dockerfile for Go", "📦"},
	{files.FileTypeJenkinsfile, "Jenkinsfile", "Jenkins CI/CD pipeline", "🔧"},
	{files.FileTypeGitignore, ".gitignore", "Git ignore file for Go projects", "🙈"},
	{files.FileTypeEnvExample, ".env.example", "Environment variables template", "🔐"},
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	return model{
		state:     stateMainMenu,
		textInput: ti,
		spinner:   s,
		copyOnly:  false,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.state != stateMainMenu {
				m.state = stateMainMenu
				m.cursor = 0
				m.err = nil
				m.message = ""
				return m, nil
			}
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case generationDoneMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
		} else {
			m.state = stateSuccess
			m.message = msg.message
		}
		return m, nil
	}

	switch m.state {
	case stateMainMenu:
		return m.updateMainMenu(msg)
	case stateSelectProjectType:
		return m.updateProjectTypeSelect(msg)
	case stateProjectNameInput:
		return m.updateProjectNameInput(msg)
	case stateProjectDescInput:
		return m.updateProjectDescInput(msg)
	case stateProjectPortInput:
		return m.updateProjectPortInput(msg)
	case stateSelectFileType:
		return m.updateFileTypeSelect(msg)
	case stateFileServiceNameInput:
		return m.updateFileServiceNameInput(msg)
	case stateFilePortInput:
		return m.updateFilePortInput(msg)
	case stateCopyOnlyConfirm:
		return m.updateCopyOnlyConfirm(msg)
	case stateFilePathInput:
		return m.updateFilePathInput(msg)
	case stateSuccess, stateError:
		return m.updateFinalState(msg)
	}

	return m, nil
}

func (m model) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(mainMenuItems)-1 {
				m.cursor++
			}
		case "enter":
			switch m.cursor {
			case 0:
				m.state = stateSelectProjectType
				m.cursor = 0
			case 1:
				m.state = stateSelectFileType
				m.cursor = 0
			case 2:
				m.message = fmt.Sprintf("Go CLI v%s\nA scaffolding tool for Go projects\nBuilt with ❤️ using Bubble Tea", VERSION)
				m.state = stateSuccess
			case 3:
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m model) updateProjectTypeSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(projectTypeItems)-1 {
				m.cursor++
			}
		case "enter":
			m.projectType = projectTypeItems[m.cursor].Type
			m.state = stateProjectNameInput
			m.textInput.SetValue("")
			m.textInput.Placeholder = "my-awesome-app"
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateProjectNameInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "enter" {
			name := strings.TrimSpace(m.textInput.Value())
			if name == "" {
				m.err = fmt.Errorf("project name cannot be empty")
				return m, nil
			}
			if !frameworks.IsValidProjectName(name) {
				m.err = fmt.Errorf("invalid name: use only lowercase letters, numbers, and hyphens")
				return m, nil
			}
			m.err = nil
			m.projectName = name
			m.state = stateProjectDescInput
			m.textInput.SetValue("")
			m.textInput.Placeholder = fmt.Sprintf("A Go %s application", m.projectType)
			return m, nil
		}
	}
	return m, cmd
}

func (m model) updateProjectDescInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "enter" {
			desc := strings.TrimSpace(m.textInput.Value())
			if desc == "" {
				desc = fmt.Sprintf("A Go %s application", m.projectType)
			}
			m.projectDesc = desc

			if m.projectType == frameworks.TypeREST || m.projectType == frameworks.TypeFullStack {
				m.state = stateProjectPortInput
				m.textInput.SetValue("")
				m.textInput.Placeholder = "8080"
			} else {
				m.projectPort = ""
				return m, m.generateProject()
			}
			return m, nil
		}
	}
	return m, cmd
}

func (m model) updateProjectPortInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "enter" {
			port := strings.TrimSpace(m.textInput.Value())
			if port == "" {
				port = "8080"
			}
			m.projectPort = port
			return m, m.generateProject()
		}
	}
	return m, cmd
}

func (m model) updateFileTypeSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(fileTypeItems)-1 {
				m.cursor++
			}
		case "enter":
			m.fileType = fileTypeItems[m.cursor].Type
			m.state = stateFileServiceNameInput
			m.textInput.SetValue("")
			m.textInput.Placeholder = "my-service"
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateFileServiceNameInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "enter" {
			name := strings.TrimSpace(m.textInput.Value())
			if name == "" {
				name = "app"
			}
			m.fileServiceName = name
			m.state = stateFilePortInput
			m.textInput.SetValue("")
			m.textInput.Placeholder = "8080"
			return m, nil
		}
	}
	return m, cmd
}

func (m model) updateFilePortInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "enter" {
			port := strings.TrimSpace(m.textInput.Value())
			if port == "" {
				port = "8080"
			}
			m.filePort = port
			m.state = stateCopyOnlyConfirm
			m.cursor = 1
			return m, nil
		}
	}
	return m, cmd
}

func (m model) updateCopyOnlyConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k", "left", "h":
			m.cursor = 0
		case "down", "j", "right", "l":
			m.cursor = 1
		case "y":
			m.cursor = 0
		case "n":
			m.cursor = 1
		case "enter":
			m.copyOnly = m.cursor == 0
			if m.copyOnly {
				return m, m.generateFile()
			} else {
				m.state = stateFilePathInput
				m.textInput.SetValue("")
				cwd, _ := os.Getwd()
				m.textInput.Placeholder = filepath.Join(cwd, files.GetDefaultFileName(m.fileType))
				return m, nil
			}
		}
	}
	return m, nil
}

func (m model) updateFilePathInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "enter" {
			path := strings.TrimSpace(m.textInput.Value())
			if path == "" {
				cwd, _ := os.Getwd()
				path = filepath.Join(cwd, files.GetDefaultFileName(m.fileType))
			}
			m.filePath = path
			return m, m.generateFile()
		}
	}
	return m, cmd
}

func (m model) updateFinalState(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter", "esc", "q":
			m.state = stateMainMenu
			m.cursor = 0
			m.err = nil
			m.message = ""
			return m, nil
		}
	}
	return m, nil
}

type generationDoneMsg struct {
	message string
	err     error
}

func (m *model) generateProject() tea.Cmd {
	m.state = stateGenerating
	return func() tea.Msg {
		generator := frameworks.NewProjectGenerator(m.projectName, m.projectDesc, m.projectPort, m.projectType)

		if err := generator.CreateDirectories(); err != nil {
			return generationDoneMsg{err: fmt.Errorf("failed to create directories: %w", err)}
		}

		if err := generator.GenerateFiles(); err != nil {
			return generationDoneMsg{err: fmt.Errorf("failed to generate files: %w", err)}
		}

		_ = frameworks.RunGoModTidy(m.projectName)

		var msg string
		if m.projectType == frameworks.TypeFullStack {
			msg = fmt.Sprintf("✨ Project '%s' created successfully!\n\nNext steps:\n  cd %s\n  cd frontend && npm install && cd ..\n  go mod tidy\n\nDev mode:\n  go run main.go          (backend)\n  cd frontend && npm run dev (frontend)\n\nBuild exe:\n  build.bat               (Windows)\n  make build              (Linux/macOS)\n\nHappy coding! 🎉", m.projectName, m.projectName)
		} else {
			msg = fmt.Sprintf("✨ Project '%s' created successfully!\n\nNext steps:\n  cd %s\n  go run .\n\nHappy coding! 🎉", m.projectName, m.projectName)
		}

		return generationDoneMsg{message: msg}
	}
}

func (m *model) generateFile() tea.Cmd {
	m.state = stateGenerating
	return func() tea.Msg {
		info := &files.FileGeneratorInfo{
			ServiceName: m.fileServiceName,
			Port:        m.filePort,
			ModuleName:  m.fileServiceName,
			ImageName:   m.fileServiceName,
			GoVersion:   "1.21",
		}

		generator := files.NewFileGenerator(m.filePath, m.fileType, info)
		content := generator.GetContent()

		if m.copyOnly {
			err := clipboard.WriteAll(content)
			if err != nil {
				return generationDoneMsg{err: fmt.Errorf("failed to copy to clipboard: %w", err)}
			}
			return generationDoneMsg{message: fmt.Sprintf("📋 Content copied to clipboard!\n\nFile type: %s", files.GetDefaultFileName(m.fileType))}
		}

		generator.Path = m.filePath
		if err := generator.WriteToFile(); err != nil {
			return generationDoneMsg{err: fmt.Errorf("failed to write file: %w", err)}
		}

		return generationDoneMsg{message: fmt.Sprintf("✅ File created successfully!\n\nPath: %s", m.filePath)}
	}
}

func (m model) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("🛠️  Go CLI Generator"))
	s.WriteString("\n")
	s.WriteString(subtitleStyle.Render(fmt.Sprintf("v%s - Generate Go projects and config files", VERSION)))
	s.WriteString("\n\n")

	switch m.state {
	case stateMainMenu:
		s.WriteString(m.viewMainMenu())
	case stateSelectProjectType:
		s.WriteString(m.viewProjectTypeSelect())
	case stateProjectNameInput:
		s.WriteString(m.viewTextInput("Enter project name:", "Project name (lowercase, hyphens, underscores)"))
	case stateProjectDescInput:
		s.WriteString(m.viewTextInput("Enter project description:", "Press Enter for default"))
	case stateProjectPortInput:
		s.WriteString(m.viewTextInput("Enter server port:", "Press Enter for 8080"))
	case stateSelectFileType:
		s.WriteString(m.viewFileTypeSelect())
	case stateFileServiceNameInput:
		s.WriteString(m.viewTextInput("Enter service name:", "Used in Docker/Jenkins configs"))
	case stateFilePortInput:
		s.WriteString(m.viewTextInput("Enter port:", "Press Enter for 8080"))
	case stateCopyOnlyConfirm:
		s.WriteString(m.viewCopyOnlyConfirm())
	case stateFilePathInput:
		s.WriteString(m.viewTextInput("Enter file path:", "Press Enter for current directory"))
	case stateGenerating:
		s.WriteString(m.viewGenerating())
	case stateSuccess:
		s.WriteString(m.viewSuccess())
	case stateError:
		s.WriteString(m.viewError())
	}

	s.WriteString(m.viewHelp())

	return s.String()
}

func (m model) viewMainMenu() string {
	var s strings.Builder

	s.WriteString(inputLabelStyle.Render("What would you like to do?"))
	s.WriteString("\n\n")

	for i, item := range mainMenuItems {
		cursor := "  "
		style := menuItemStyle
		if m.cursor == i {
			cursor = cursorStyle.Render("▸ ")
			style = selectedItemStyle
		}
		s.WriteString(cursor + style.Render(item) + "\n")
	}

	return s.String()
}

func (m model) viewProjectTypeSelect() string {
	var s strings.Builder

	s.WriteString(inputLabelStyle.Render("Select project type:"))
	s.WriteString("\n\n")

	for i, item := range projectTypeItems {
		cursor := "  "
		style := menuItemStyle
		if m.cursor == i {
			cursor = cursorStyle.Render("▸ ")
			style = selectedItemStyle
		}
		line := fmt.Sprintf("%s %s", item.Icon, item.Name)
		s.WriteString(cursor + style.Render(line) + "\n")
		if m.cursor == i {
			s.WriteString(dimStyle.Render(fmt.Sprintf("     %s", item.Desc)) + "\n")
		}
	}

	return s.String()
}

func (m model) viewFileTypeSelect() string {
	var s strings.Builder

	s.WriteString(inputLabelStyle.Render("Select file type to generate:"))
	s.WriteString("\n\n")

	for i, item := range fileTypeItems {
		cursor := "  "
		style := menuItemStyle
		if m.cursor == i {
			cursor = cursorStyle.Render("▸ ")
			style = selectedItemStyle
		}
		line := fmt.Sprintf("%s %s", item.Icon, item.Name)
		s.WriteString(cursor + style.Render(line) + "\n")
		if m.cursor == i {
			s.WriteString(dimStyle.Render(fmt.Sprintf("     %s", item.Desc)) + "\n")
		}
	}

	return s.String()
}

func (m model) viewTextInput(label, hint string) string {
	var s strings.Builder

	s.WriteString(inputLabelStyle.Render(label))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render(hint))
	s.WriteString("\n\n")
	s.WriteString(boxStyle.Render(m.textInput.View()))
	s.WriteString("\n")

	if m.err != nil {
		s.WriteString("\n")
		s.WriteString(errorMsgStyle.Render("⚠️  " + m.err.Error()))
	}

	return s.String()
}

func (m model) viewCopyOnlyConfirm() string {
	var s strings.Builder

	s.WriteString(inputLabelStyle.Render("Copy to clipboard only?"))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render("If No, the file will be saved to disk"))
	s.WriteString("\n\n")

	options := []string{"Yes - Copy to clipboard", "No - Save to file"}
	for i, opt := range options {
		cursor := "  "
		style := menuItemStyle
		if m.cursor == i {
			cursor = cursorStyle.Render("▸ ")
			style = selectedItemStyle
		}
		s.WriteString(cursor + style.Render(opt) + "\n")
	}

	return s.String()
}

func (m model) viewGenerating() string {
	return fmt.Sprintf("%s Generating...\n", m.spinner.View())
}

func (m model) viewSuccess() string {
	var s strings.Builder
	s.WriteString(boxStyle.Copy().BorderForeground(secondaryColor).Render(
		successStyle.Render("✓ Success!\n\n") + m.message,
	))
	s.WriteString("\n\n")
	s.WriteString(dimStyle.Render("Press Enter or Esc to continue..."))
	return s.String()
}

func (m model) viewError() string {
	var s strings.Builder
	s.WriteString(boxStyle.Copy().BorderForeground(errorColor).Render(
		errorMsgStyle.Render("✗ Error\n\n") + m.err.Error(),
	))
	s.WriteString("\n\n")
	s.WriteString(dimStyle.Render("Press Enter or Esc to continue..."))
	return s.String()
}

func (m model) viewHelp() string {
	var help string
	switch m.state {
	case stateMainMenu, stateSelectProjectType, stateSelectFileType:
		help = "↑/↓: Navigate • Enter: Select • Esc: Back • Ctrl+C: Quit"
	case stateCopyOnlyConfirm:
		help = "↑/↓ or y/n: Select • Enter: Confirm • Esc: Back"
	default:
		help = "Enter: Confirm • Esc: Back • Ctrl+C: Quit"
	}
	return helpStyle.Render(help)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
