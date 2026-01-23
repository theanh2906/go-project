package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Language mapping from B files to A files
var languageMap = map[string]string{
	"arabic.json":     "ab.json",
	"vietnamese.json": "vi.json",
	"french.json":     "fr.json",
	"german.json":     "ge.json",
	"hebrew.json":     "he.json",
	"japanese.json":   "ja.json",
	"korean.json":     "ko.json",
	"spanish.json":    "sp.json",
}

// Paths - relative to project root
const (
	sourceLangsPath = "C:\\Program Files (x86)\\OPSWAT\\Metadefender Kiosk\\Client\\en\\resources\\languages"
	destAssetsPath  = "dist\\assets"
	translateLangsPath = "kiosk-translate-generator\\languages"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D9FF")).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			MarginBottom(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00D9FF")).
			Padding(1, 2)
)

// Menu item
type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Application states
type state int

const (
	stateMenu state = iota
	stateRunning
	stateResult
)

// Model
type model struct {
	list     list.Model
	state    state
	logs     []string
	err      error
	projectRoot string
}

// Messages
type runCompleteMsg struct {
	logs []string
	err  error
}

func initialModel(projectRoot string) model {
	items := []list.Item{
		item{title: "🔄 Test Translate", desc: "Copy & force replace translation files for testing"},
		item{title: "❌ Exit", desc: "Exit the application"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#00D9FF")).
		BorderForeground(lipgloss.Color("#00D9FF"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#888888"))

	l := list.New(items, delegate, 60, 10)
	l.Title = "🌐 Translate Support Tool"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle

	return model{
		list:        l,
		state:       stateMenu,
		projectRoot: projectRoot,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == stateMenu {
				return m, tea.Quit
			}
			m.state = stateMenu
			m.logs = nil
			m.err = nil
			return m, nil
		case "enter":
			if m.state == stateMenu {
				i, ok := m.list.SelectedItem().(item)
				if ok {
					switch i.title {
					case "🔄 Test Translate":
						m.state = stateRunning
						m.logs = []string{}
						return m, m.runTestTranslate()
					case "❌ Exit":
						return m, tea.Quit
					}
				}
			} else if m.state == stateResult {
				m.state = stateMenu
				m.logs = nil
				m.err = nil
				return m, nil
			}
		case "esc":
			if m.state != stateMenu {
				m.state = stateMenu
				m.logs = nil
				m.err = nil
				return m, nil
			}
		}
	case runCompleteMsg:
		m.state = stateResult
		m.logs = msg.logs
		m.err = msg.err
		return m, nil
	}

	if m.state == stateMenu {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	switch m.state {
	case stateRunning:
		return m.viewRunning()
	case stateResult:
		return m.viewResult()
	default:
		return m.viewMenu()
	}
}

func (m model) viewMenu() string {
	return "\n" + m.list.View() + "\n\n" + infoStyle.Render("Press Enter to select, q to quit")
}

func (m model) viewRunning() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(headerStyle.Render("🔄 Running Test Translate..."))
	sb.WriteString("\n\n")
	sb.WriteString(infoStyle.Render("Please wait while processing translations..."))
	sb.WriteString("\n")
	return sb.String()
}

func (m model) viewResult() string {
	var sb strings.Builder
	sb.WriteString("\n")
	
	if m.err != nil {
		sb.WriteString(errorStyle.Render("❌ Error: " + m.err.Error()))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(successStyle.Render("✅ Test Translate Completed!"))
		sb.WriteString("\n\n")
	}

	// Show logs in a box
	logsContent := strings.Join(m.logs, "\n")
	sb.WriteString(boxStyle.Render(logsContent))
	sb.WriteString("\n\n")
	sb.WriteString(infoStyle.Render("Press Enter or Esc to go back to menu"))
	
	return sb.String()
}

func (m model) runTestTranslate() tea.Cmd {
	return func() tea.Msg {
		logs := []string{}
		var err error

		// Step 1: Copy JSON files from source to dist/assets
		logs = append(logs, headerStyle.Render("Step 1: Copying language files..."))
		
		destPath := filepath.Join(m.projectRoot, destAssetsPath)
		
		// Check if source exists
		if _, err := os.Stat(sourceLangsPath); os.IsNotExist(err) {
			return runCompleteMsg{
				logs: logs,
				err:  fmt.Errorf("source path not found: %s", sourceLangsPath),
			}
		}

		// Copy all JSON files
		files, err := os.ReadDir(sourceLangsPath)
		if err != nil {
			return runCompleteMsg{logs: logs, err: err}
		}

		copiedCount := 0
		for _, f := range files {
			if !f.IsDir() {
				src := filepath.Join(sourceLangsPath, f.Name())
				dst := filepath.Join(destPath, f.Name())
				
				if err := copyFile(src, dst); err != nil {
					logs = append(logs, errorStyle.Render(fmt.Sprintf("  ❌ Failed to copy %s: %v", f.Name(), err)))
				} else {
					copiedCount++
				}
			}
		}
		logs = append(logs, successStyle.Render(fmt.Sprintf("  ✅ Copied %d files to dist/assets", copiedCount)))

		// Step 2: Replace translations
		logs = append(logs, "")
		logs = append(logs, headerStyle.Render("Step 2: Replacing translations (force)..."))

		translatePath := filepath.Join(m.projectRoot, translateLangsPath)
		
		for srcFile, dstFile := range languageMap {
			srcPath := filepath.Join(translatePath, srcFile)
			dstPath := filepath.Join(destPath, dstFile)

			result := replaceTranslationFile(srcPath, dstPath)
			if result.err != nil {
				logs = append(logs, errorStyle.Render(fmt.Sprintf("  ❌ %s: %v", srcFile, result.err)))
			} else {
				msg := fmt.Sprintf("  ✅ %s → %s: %d replaced", srcFile, dstFile, result.replaced)
				if result.added > 0 {
					msg += infoStyle.Render(fmt.Sprintf(" (+%d new)", result.added))
				}
				logs = append(logs, successStyle.Render(msg))
			}
		}

		return runCompleteMsg{logs: logs, err: nil}
	}
}

// ReplaceResult holds the result of replacing translations in a file
type ReplaceResult struct {
	replaced int
	added    int
	err      error
}

// Source JSON structure (B files)
type SourceJSON struct {
	Languages []struct {
		LangID string                       `json:"lang_id"`
		Data   map[string]map[string]string `json:"data"`
	} `json:"languages"`
}

func replaceTranslationFile(srcPath, dstPath string) ReplaceResult {
	// Read source file
	srcData, err := os.ReadFile(srcPath)
	if err != nil {
		return ReplaceResult{err: fmt.Errorf("failed to read source: %w", err)}
	}

	var srcJSON SourceJSON
	if err := json.Unmarshal(srcData, &srcJSON); err != nil {
		return ReplaceResult{err: fmt.Errorf("failed to parse source: %w", err)}
	}

	if len(srcJSON.Languages) == 0 || srcJSON.Languages[0].Data == nil {
		return ReplaceResult{err: fmt.Errorf("no languages data found in source")}
	}

	sourceData := srcJSON.Languages[0].Data

	// Read destination file
	dstData, err := os.ReadFile(dstPath)
	if err != nil {
		return ReplaceResult{err: fmt.Errorf("failed to read destination: %w", err)}
	}

	// Parse as generic map to preserve all fields
	var dstJSON map[string]interface{}
	if err := json.Unmarshal(dstData, &dstJSON); err != nil {
		return ReplaceResult{err: fmt.Errorf("failed to parse destination: %w", err)}
	}

	dataField, ok := dstJSON["data"].(map[string]interface{})
	if !ok {
		return ReplaceResult{err: fmt.Errorf("no data field found in destination")}
	}

	// Force replace translations from source B to destination A
	replaced := 0
	added := 0

	for key, srcValue := range sourceData {
		if text, hasText := srcValue["text"]; hasText {
			if dstEntry, exists := dataField[key]; exists {
				// Key exists in A - force replace the text
				if dstMap, ok := dstEntry.(map[string]interface{}); ok {
					dstMap["text"] = text
					replaced++
				}
			} else {
				// Key doesn't exist in A - add it from B
				dataField[key] = map[string]interface{}{
					"text": text,
				}
				added++
			}
		}
	}

	// Write back with formatting
	output, err := json.MarshalIndent(dstJSON, "", "  ")
	if err != nil {
		return ReplaceResult{err: fmt.Errorf("failed to marshal: %w", err)}
	}

	if err := os.WriteFile(dstPath, output, 0644); err != nil {
		return ReplaceResult{err: fmt.Errorf("failed to write: %w", err)}
	}

	return ReplaceResult{replaced: replaced, added: added}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func findProjectRoot() (string, error) {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Look for markers that indicate project root
	markers := []string{"package.json", "dist", "kiosk-translate-generator"}
	
	for {
		allFound := true
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(dir, marker)); os.IsNotExist(err) {
				allFound = false
				break
			}
		}
		
		if allFound {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// If not found, return current directory
	return os.Getwd()
}

func runCLI(projectRoot string) {
	fmt.Println("🌐 Translate Support Tool - CLI Mode")
	fmt.Println("=====================================")
	fmt.Println()
	
	// Step 1: Copy JSON files
	fmt.Println("📁 Step 1: Copying language files from Kiosk installation...")
	destPath := filepath.Join(projectRoot, destAssetsPath)
	
	if _, err := os.Stat(sourceLangsPath); os.IsNotExist(err) {
		fmt.Printf("❌ Source path not found: %s\n", sourceLangsPath)
		os.Exit(1)
	}

	files, err := os.ReadDir(sourceLangsPath)
	if err != nil {
		fmt.Printf("❌ Error reading source: %v\n", err)
		os.Exit(1)
	}

	copiedCount := 0
	for _, f := range files {
		if !f.IsDir() {
			src := filepath.Join(sourceLangsPath, f.Name())
			dst := filepath.Join(destPath, f.Name())
			if err := copyFile(src, dst); err != nil {
				fmt.Printf("  ❌ Failed to copy %s: %v\n", f.Name(), err)
			} else {
				copiedCount++
			}
		}
	}
	fmt.Printf("  ✅ Copied %d files to dist/assets\n\n", copiedCount)

	// Step 2: Replace translations (force)
	fmt.Println("🔄 Step 2: Replacing translations (force)...")
	translatePath := filepath.Join(projectRoot, translateLangsPath)

	for srcFile, dstFile := range languageMap {
		srcPath := filepath.Join(translatePath, srcFile)
		dstPath := filepath.Join(destPath, dstFile)

		result := replaceTranslationFile(srcPath, dstPath)
		if result.err != nil {
			fmt.Printf("  ❌ %s: %v\n", srcFile, result.err)
		} else {
			msg := fmt.Sprintf("  ✅ %s → %s: %d replaced", srcFile, dstFile, result.replaced)
			if result.added > 0 {
				msg += fmt.Sprintf(" (+%d new)", result.added)
			}
			fmt.Println(msg)
		}
	}

	fmt.Println()
	fmt.Println("✅ Test Translate completed!")
}

func main() {
	cliMode := flag.Bool("cli", false, "Run in CLI mode (non-interactive)")
	testTranslate := flag.Bool("test-translate", false, "Run test translate directly (CLI mode)")
	flag.Parse()

	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Printf("Error finding project root: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Project root: %s\n\n", projectRoot)

	// CLI mode or test-translate flag
	if *cliMode || *testTranslate {
		runCLI(projectRoot)
		return
	}

	// TUI mode
	p := tea.NewProgram(initialModel(projectRoot), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
