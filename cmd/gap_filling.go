package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const Version = "1.0.0"

type state int

const (
	stateLoading state = iota
	stateCollectingInput
	stateProcessing
	statePreview
	stateAskContinue
	stateDone
)

type model struct {
	templatePath  string
	template      string
	placeholders  []string
	currentIndex  int
	values        map[string]string
	currentInput  string
	result        string
	state         state
	err           error
	continueInput string
	width         int
	height        int
}

var (
	// Color styles using lipgloss
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF00FF")).
			Bold(true)

	bannerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFFF")).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FFFF")).
			Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFF00")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	placeholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FFFF"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF00FF")).
			Bold(true)

	resultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFFF"))
)

func (m model) Init() tea.Cmd {
	return loadTemplate(m.templatePath)
}

type templateLoaded struct {
	content string
	err     error
}

type processingDone struct{}

func loadTemplate(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := os.ReadFile(path)
		if err != nil {
			return templateLoaded{err: err}
		}
		return templateLoaded{content: string(content)}
	}
}

func processTemplate() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(300 * time.Millisecond)
		return processingDone{}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case stateCollectingInput:
			switch msg.Type {
			case tea.KeyCtrlC:
				m.state = stateDone
				return m, tea.Quit
			case tea.KeyEnter:
				if m.currentInput != "" {
					m.values[m.placeholders[m.currentIndex]] = m.currentInput
					m.currentIndex++
					m.currentInput = ""
					if m.currentIndex >= len(m.placeholders) {
						m.state = stateProcessing
						return m, processTemplate()
					}
				}
				return m, nil
			case tea.KeyBackspace:
				if len(m.currentInput) > 0 {
					m.currentInput = m.currentInput[:len(m.currentInput)-1]
				}
				return m, nil
			default:
				if msg.Type == tea.KeyRunes {
					m.currentInput += string(msg.Runes)
				}
				return m, nil
			}

		case stateAskContinue:
			switch msg.String() {
			case "ctrl+c":
				m.state = stateDone
				return m, tea.Quit
			case "y", "Y":
				m.currentIndex = 0
				m.values = make(map[string]string)
				m.currentInput = ""
				m.result = ""
				m.continueInput = ""
				m.state = stateCollectingInput
				return m, nil
			case "n", "N":
				m.state = stateDone
				return m, tea.Quit
			}
		}

	case templateLoaded:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateDone
			return m, tea.Quit
		}
		m.template = msg.content
		m.placeholders = extractPlaceholders(msg.content)

		if len(m.placeholders) == 0 {
			m.result = strings.ReplaceAll(m.template, "\x00", "")
			clipboard.WriteAll(m.result)
			m.state = stateAskContinue
		} else {
			m.state = stateCollectingInput
		}
		return m, nil

	case processingDone:
		m.result = fillTemplate(m.template, m.values)
		clipboard.WriteAll(m.result)
		m.state = statePreview
		time.Sleep(200 * time.Millisecond)
		m.state = stateAskContinue
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	switch m.state {
	case stateLoading:
		return renderBanner() + "\n\n" + headerStyle.Render("📋 Loading Template...") + "\n"

	case stateCollectingInput:
		var s strings.Builder
		s.WriteString(renderBanner())
		s.WriteString("\n\n")
		s.WriteString(headerStyle.Render("🔍 Detected Placeholders") + "\n")
		s.WriteString(separatorStyle.Render(strings.Repeat("─", 66)) + "\n")
		s.WriteString(fmt.Sprintf("Found %s placeholder(s):\n\n", placeholderStyle.Render(fmt.Sprintf("%d", len(m.placeholders)))))

		for i, ph := range m.placeholders {
			if i == m.currentIndex {
				s.WriteString(promptStyle.Render(fmt.Sprintf("  %d. %s", i+1, ph)) + "\n")
			} else if i < m.currentIndex {
				s.WriteString(successStyle.Render(fmt.Sprintf("  %d. %s ✓", i+1, ph)) + "\n")
			} else {
				s.WriteString(dimStyle.Render(fmt.Sprintf("  %d. %s", i+1, ph)) + "\n")
			}
		}

		s.WriteString("\n" + headerStyle.Render("✏️  Fill Values") + "\n")
		s.WriteString(separatorStyle.Render(strings.Repeat("─", 66)) + "\n")
		s.WriteString(dimStyle.Render("Please provide a value for each placeholder") + "\n\n")

		progress := dimStyle.Render(fmt.Sprintf("[%d/%d]", m.currentIndex+1, len(m.placeholders)))
		prompt := fmt.Sprintf("%s %s %s ", progress, placeholderStyle.Render(m.placeholders[m.currentIndex]), promptStyle.Render("❯"))
		s.WriteString(prompt + m.currentInput + "█\n")

		return s.String()

	case stateProcessing:
		return renderBanner() + "\n\n" + headerStyle.Render("⚙️  Processing...") + "\n"

	case statePreview:
		var s strings.Builder
		s.WriteString(separatorStyle.Render("╔"+strings.Repeat("═", 64)+"╗") + "\n")
		s.WriteString(separatorStyle.Render("║") + "  " + headerStyle.Render("📄 RESULT PREVIEW") + strings.Repeat(" ", 46) + separatorStyle.Render("║") + "\n")
		s.WriteString(separatorStyle.Render("╚"+strings.Repeat("═", 64)+"╝") + "\n\n")

		for _, line := range strings.Split(m.result, "\n") {
			s.WriteString(resultStyle.Render(line) + "\n")
		}

		s.WriteString("\n" + separatorStyle.Render(strings.Repeat("─", 66)) + "\n")
		s.WriteString(successStyle.Render("✓ Result copied to clipboard!") + "\n")
		return s.String()

	case stateAskContinue:
		var s strings.Builder
		if m.result != "" {
			s.WriteString(separatorStyle.Render("╔"+strings.Repeat("═", 64)+"╗") + "\n")
			s.WriteString(separatorStyle.Render("║") + "  " + headerStyle.Render("📄 RESULT PREVIEW") + strings.Repeat(" ", 46) + separatorStyle.Render("║") + "\n")
			s.WriteString(separatorStyle.Render("╚"+strings.Repeat("═", 64)+"╝") + "\n\n")

			lines := strings.Split(m.result, "\n")
			maxLines := 10
			if len(lines) > maxLines {
				for _, line := range lines[:maxLines] {
					s.WriteString(resultStyle.Render(line) + "\n")
				}
				s.WriteString(dimStyle.Render(fmt.Sprintf("... (%d more lines)", len(lines)-maxLines)) + "\n")
			} else {
				for _, line := range lines {
					s.WriteString(resultStyle.Render(line) + "\n")
				}
			}

			s.WriteString("\n" + separatorStyle.Render(strings.Repeat("─", 66)) + "\n")
			s.WriteString(successStyle.Render("✓ Result copied to clipboard!") + "\n\n")
		}
		s.WriteString(separatorStyle.Render(strings.Repeat("═", 66)) + "\n\n")
		s.WriteString(promptStyle.Render("❯ ") + "Generate another? " + dimStyle.Render("(y/n)") + ": ")
		return s.String()

	case stateDone:
		if m.err != nil {
			return errorStyle.Render(fmt.Sprintf("✗ Error: %v\n", m.err))
		}
		var s strings.Builder
		s.WriteString("\n" + separatorStyle.Render("╔"+strings.Repeat("═", 64)+"╗") + "\n")
		s.WriteString(separatorStyle.Render("║") + strings.Repeat(" ", 66) + separatorStyle.Render("║") + "\n")
		s.WriteString(separatorStyle.Render("║") + "     " + titleStyle.Render("✨ Thanks for using Template Generator! ✨") + "      " + separatorStyle.Render("║") + "\n")
		s.WriteString(separatorStyle.Render("║") + strings.Repeat(" ", 66) + separatorStyle.Render("║") + "\n")
		s.WriteString(separatorStyle.Render("║") + "     " + resultStyle.Render("Have a great day! 👋") + strings.Repeat(" ", 32) + separatorStyle.Render("║") + "\n")
		s.WriteString(separatorStyle.Render("║") + strings.Repeat(" ", 66) + separatorStyle.Render("║") + "\n")
		s.WriteString(separatorStyle.Render("╚"+strings.Repeat("═", 64)+"╝") + "\n\n")
		return s.String()
	}

	return ""
}

func renderBanner() string {
	banner := `
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║     ✨ Interactive Template Generator ✨                    ║
║                                                              ║
║     Transform templates into reality                         ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝`
	return bannerStyle.Render(banner)
}

func extractPlaceholders(template string) []string {
	re := regexp.MustCompile(`:(\w+)`)
	matches := re.FindAllStringSubmatch(template, -1)

	var placeholders []string
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			placeholder := match[1]
			if !seen[placeholder] {
				placeholders = append(placeholders, placeholder)
				seen[placeholder] = true
			}
		}
	}

	return placeholders
}

func fillTemplate(template string, values map[string]string) string {
	result := template
	for placeholder, value := range values {
		result = strings.ReplaceAll(result, ":"+placeholder, value)
	}
	return result
}

func main() {
	var templatePath string

	rootCmd := &cobra.Command{
		Use:     "tg",
		Short:   "🚀 Interactive Template Generator - Fill templates with placeholders 🚀",
		Version: Version,
		Long: `Interactive Template Generator - Transform templates into reality

This tool helps you fill templates with placeholders in the format :placeholder_name.
It provides a beautiful interactive interface to collect values and copies the result to your clipboard.`,
		Example: `  tg -t template.txt
  tg --template my_template.txt
  tg -t emails/welcome.txt`,
		Run: func(cmd *cobra.Command, args []string) {
			if templatePath == "" {
				fmt.Println(errorStyle.Render("✗ Template path is required. Use -t or --template flag"))
				os.Exit(1)
			}

			m := model{
				templatePath: templatePath,
				values:       make(map[string]string),
				state:        stateLoading,
			}

			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVarP(&templatePath, "template", "t", "", "Path to the template file (required)")
	rootCmd.MarkFlagRequired("template")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
