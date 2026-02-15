package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UnitConverter interface cho các loại converter khác nhau
type UnitConverter interface {
	GetName() string
	GetDescription() string
	Convert(input string) (string, error)
	GetInputUnit() string
	GetOutputUnit() string
}

// PxToRemConverter converts pixels to rem
type PxToRemConverter struct {
	baseFontSize float64 // default font size (usually 16px)
}

func NewPxToRemConverter() *PxToRemConverter {
	return &PxToRemConverter{
		baseFontSize: 16.0,
	}
}

func (c *PxToRemConverter) GetName() string {
	return "Pixel to REM"
}

func (c *PxToRemConverter) GetDescription() string {
	return fmt.Sprintf("Convert pixels to rem (base: %.0fpx)", c.baseFontSize)
}

func (c *PxToRemConverter) GetInputUnit() string {
	return "px"
}

func (c *PxToRemConverter) GetOutputUnit() string {
	return "rem"
}

func (c *PxToRemConverter) Convert(input string) (string, error) {
	// Remove "px" if present
	input = strings.TrimSpace(input)
	input = strings.TrimSuffix(input, "px")
	input = strings.TrimSuffix(input, "PX")

	px, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return "", fmt.Errorf("invalid number: %v", err)
	}

	rem := px / c.baseFontSize
	// Format without trailing zeros
	result := strconv.FormatFloat(rem, 'f', -1, 64)
	return result, nil
}

// unitConverterModel represents the application state
type unitConverterModel struct {
	converters        []UnitConverter
	selectedIndex     int
	mode              string // "menu" or "convert"
	input             string
	result            string
	err               string
	history           []string
	activeConverter   UnitConverter
	copiedToClipboard bool
}

func unitConverterInitialModel() unitConverterModel {
	converters := []UnitConverter{
		NewPxToRemConverter(),
		// Có thể thêm converters khác ở đây
		// NewRemToPxConverter(),
		// NewPxToEmConverter(),
	}

	return unitConverterModel{
		converters:    converters,
		selectedIndex: 0,
		mode:          "menu",
		history:       []string{},
	}
}

func (m unitConverterModel) Init() tea.Cmd {
	return nil
}

func (m unitConverterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode {
		case "menu":
			return m.updateMenu(msg)
		case "convert":
			return m.updateConvert(msg)
		}
	}
	return m, nil
}

func (m unitConverterModel) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}

	case "down", "j":
		if m.selectedIndex < len(m.converters)-1 {
			m.selectedIndex++
		}

	case "enter":
		m.mode = "convert"
		m.activeConverter = m.converters[m.selectedIndex]
		m.input = ""
		m.result = ""
		m.err = ""
		m.history = []string{}
	}

	return m, nil
}

func (m unitConverterModel) updateConvert(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		// Exit convert mode and return to menu
		m.mode = "menu"
		m.input = ""
		m.result = ""
		m.err = ""
		m.copiedToClipboard = false
		return m, nil

	case "enter":
		if m.input != "" {
			result, err := m.activeConverter.Convert(m.input)
			if err != nil {
				m.err = err.Error()
				m.result = ""
				m.copiedToClipboard = false
			} else {
				m.result = result
				m.err = ""
				// Copy to clipboard
				clipboard.WriteAll(result)
				m.copiedToClipboard = true
				// Add to history
				historyEntry := fmt.Sprintf("%s%s → %s%s",
					m.input,
					m.activeConverter.GetInputUnit(),
					result,
					m.activeConverter.GetOutputUnit())
				m.history = append([]string{historyEntry}, m.history...)
				if len(m.history) > 5 {
					m.history = m.history[:5]
				}
			}
			m.input = ""
		}

	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	default:
		// Add character to input
		if len(msg.String()) == 1 {
			m.input += msg.String()
		}
	}

	return m, nil
}

func (m unitConverterModel) View() string {
	switch m.mode {
	case "menu":
		return m.viewMenu()
	case "convert":
		return m.viewConvert()
	}
	return ""
}

func (m unitConverterModel) viewMenu() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00ff00")).
		MarginBottom(1)

	s.WriteString(titleStyle.Render("🔄 Unit Converter"))
	s.WriteString("\n\n")

	s.WriteString("Select a converter:\n\n")

	for i, converter := range m.converters {
		cursor := " "
		if i == m.selectedIndex {
			cursor = "►"
		}

		style := lipgloss.NewStyle()
		if i == m.selectedIndex {
			style = style.Foreground(lipgloss.Color("#00ff00")).Bold(true)
		}

		s.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(converter.GetName())))
		s.WriteString(fmt.Sprintf("  %s\n\n", converter.GetDescription()))
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		MarginTop(1)

	s.WriteString(helpStyle.Render("\n↑/↓: navigate • enter: select • q: quit"))

	return s.String()
}

func (m unitConverterModel) viewConvert() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00ff00"))

	s.WriteString(titleStyle.Render(fmt.Sprintf("🔄 %s", m.activeConverter.GetName())))
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render(m.activeConverter.GetDescription()))
	s.WriteString("\n\n")

	// Input box
	inputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00ff00")).
		Padding(0, 1).
		Width(40)

	inputLabel := fmt.Sprintf("Input (%s):", m.activeConverter.GetInputUnit())
	s.WriteString(inputLabel + "\n")
	s.WriteString(inputBoxStyle.Render(m.input + "█"))
	s.WriteString("\n\n")

	// Result
	if m.result != "" {
		resultStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ff00"))
		s.WriteString(fmt.Sprintf("Result: %s", resultStyle.Render(m.result+" "+m.activeConverter.GetOutputUnit())))
		if m.copiedToClipboard {
			copiedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))
			s.WriteString(copiedStyle.Render(" ✓ Copied to clipboard"))
		}
		s.WriteString("\n\n")
	}

	// Error
	if m.err != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff0000"))
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s\n\n", m.err)))
	}

	// History
	if len(m.history) > 0 {
		historyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
		s.WriteString(historyStyle.Render("Recent conversions:") + "\n")
		for _, entry := range m.history {
			s.WriteString(historyStyle.Render("  • "+entry) + "\n")
		}
		s.WriteString("\n")
	}

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	s.WriteString(helpStyle.Render("enter: convert • esc: back to menu • ctrl+c: quit"))

	return s.String()
}

func unitConverterMain() {
	p := tea.NewProgram(unitConverterInitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
