package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Constants & Types ---

type Focus int

const (
	FocusQuery Focus = iota
	FocusRoot
	FocusMode
	FocusResults
)

type SearchMode int

const (
	ModeBoth SearchMode = iota
	ModeFiles
	ModeFolders
)

func (m SearchMode) String() string {
	return [...]string{"Both", "Files", "Folders"}[m]
}

// Directories to skip during search — significantly reduces scan time
var skipDirs = map[string]bool{
	"$recycle.bin":              true,
	"system volume information": true,
	"windows":                   true,
	"program files":             true,
	"program files (x86)":       true,
	"programdata":               true,
	".git":                      true,
	"node_modules":              true,
	".cache":                    true,
	".tmp":                      true,
	"__pycache__":               true,
	".vs":                       true,
	".idea":                     true,
	".gradle":                   true,
	"vendor":                    true,
	"dist":                      true,
	"obj":                       true,
	"bin":                       true,
}

const maxResults = 10000
const numWorkers = 16 // goroutine worker pool size

// --- Messages for Bubble Tea ---

type searchResultMsg struct {
	matched []string
	scanned int
	err     error
	elapsed time.Duration
}

// debounceMsg is sent after the user stops typing for a period
type debounceMsg struct {
	query string
}

// --- Styles ---

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Border(lipgloss.RoundedBorder()).Padding(0, 1)
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Border(lipgloss.RoundedBorder()).Padding(0, 1)
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	fsTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
)

// --- Model ---

type item string

func (i item) FilterValue() string { return string(i) }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}
	str := fmt.Sprintf("%d. %s", index+1, i)
	if index == m.Index() {
		fmt.Fprint(w, lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Render("> "+str))
	} else {
		fmt.Fprint(w, str)
	}
}

type fileSearchModel struct {
	queryInput     textinput.Model
	rootInput      textinput.Model
	mode           SearchMode
	focus          Focus
	results        list.Model
	spinner        spinner.Model
	searching      bool
	status         string
	scanned        int
	lastTime       time.Duration
	skipSystemDirs bool   // toggle to skip system/hidden dirs
	lastQuery      string // for debounce detection
}

func initialModel() fileSearchModel {
	qi := textinput.New()
	qi.Placeholder = "Nhập nội dung cần tìm..."
	qi.Focus()

	ri := textinput.New()
	ri.Placeholder = "Để trống = Toàn bộ máy tính"

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	l := list.New([]list.Item{}, itemDelegate{}, 0, 0)
	l.Title = "Kết quả tìm kiếm"
	l.SetShowHelp(false)

	return fileSearchModel{
		queryInput:     qi,
		rootInput:      ri,
		mode:           ModeBoth,
		focus:          FocusQuery,
		results:        l,
		spinner:        s,
		status:         "Tab để đổi ô, Enter để tìm, Esc để thoát, F2 bật/tắt skip system dirs",
		skipSystemDirs: true, // default: skip system dirs for faster search
	}
}

// --- Search Logic (Parallel Worker Pool) ---

// parallelWalk uses a worker pool of goroutines to walk directories concurrently.
// This is significantly faster than filepath.WalkDir on multi-core machines,
// especially when scanning drives with many directories.
func parallelWalk(root string, query string, mode SearchMode, skipSystemDirs bool) ([]string, int, error) {
	var (
		scanned  int64
		found    int64
		mu       sync.Mutex
		matched  []string
		wg       sync.WaitGroup
		pending  sync.WaitGroup // tracks directories queued but not yet fully processed
		dirQueue = make(chan string, 4096)
	)

	// enqueue safely increments pending and sends to channel
	enqueue := func(dir string) {
		pending.Add(1)
		dirQueue <- dir
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dir := range dirQueue {
				if atomic.LoadInt64(&found) >= maxResults {
					pending.Done()
					continue
				}
				entries, err := os.ReadDir(dir)
				if err != nil {
					pending.Done()
					continue
				}
				for _, entry := range entries {
					if atomic.LoadInt64(&found) >= maxResults {
						break
					}

					atomic.AddInt64(&scanned, 1)
					name := entry.Name()
					nameLower := strings.ToLower(name)

					// Skip system/hidden directories
					if entry.IsDir() && skipSystemDirs {
						if skipDirs[nameLower] || (len(nameLower) > 0 && nameLower[0] == '.') {
							continue
						}
					}

					fullPath := filepath.Join(dir, name)

					// Enqueue subdirectories for other workers to process
					if entry.IsDir() {
						pending.Add(1)
						select {
						case dirQueue <- fullPath:
							// successfully enqueued
						default:
							// Queue full — walk inline instead, then mark done
							walkInline(fullPath, query, mode, skipSystemDirs, &matched, &mu, &scanned, &found)
							pending.Done()
						}
					}

					// Match check — compare against filename (faster) and full path
					if strings.Contains(nameLower, query) || strings.Contains(strings.ToLower(fullPath), query) {
						isMatch := false
						switch mode {
						case ModeBoth:
							isMatch = true
						case ModeFiles:
							isMatch = !entry.IsDir()
						case ModeFolders:
							isMatch = entry.IsDir()
						}
						if isMatch {
							mu.Lock()
							matched = append(matched, fullPath)
							mu.Unlock()
							atomic.AddInt64(&found, 1)
						}
					}
				}
				pending.Done() // this directory is fully processed
			}
		}()
	}

	// Seed the queue with root directory
	enqueue(root)
	// Also seed with first-level dirs to quickly fan out to workers
	topEntries, err := os.ReadDir(root)
	if err == nil {
		for _, e := range topEntries {
			if e.IsDir() {
				nameLower := strings.ToLower(e.Name())
				if skipSystemDirs && (skipDirs[nameLower] || (len(nameLower) > 0 && nameLower[0] == '.')) {
					continue
				}
				enqueue(filepath.Join(root, e.Name()))
			}
		}
	}

	// Close channel only when all pending directories have been fully processed
	go func() {
		pending.Wait()
		close(dirQueue)
	}()

	wg.Wait()

	sort.Strings(matched)
	return matched, int(atomic.LoadInt64(&scanned)), err
}

// walkInline is a fallback when the directory queue is full
func walkInline(dir, query string, mode SearchMode, skipSystemDirs bool, matched *[]string, mu *sync.Mutex, scanned, found *int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if atomic.LoadInt64(found) >= maxResults {
			return
		}
		atomic.AddInt64(scanned, 1)
		name := entry.Name()
		nameLower := strings.ToLower(name)

		if entry.IsDir() && skipSystemDirs {
			if skipDirs[nameLower] || (len(nameLower) > 0 && nameLower[0] == '.') {
				continue
			}
		}

		fullPath := filepath.Join(dir, name)

		if entry.IsDir() {
			walkInline(fullPath, query, mode, skipSystemDirs, matched, mu, scanned, found)
		}

		if strings.Contains(nameLower, query) || strings.Contains(strings.ToLower(fullPath), query) {
			isMatch := false
			switch mode {
			case ModeBoth:
				isMatch = true
			case ModeFiles:
				isMatch = !entry.IsDir()
			case ModeFolders:
				isMatch = entry.IsDir()
			}
			if isMatch {
				mu.Lock()
				*matched = append(*matched, fullPath)
				mu.Unlock()
				atomic.AddInt64(found, 1)
			}
		}
	}
}

func (m fileSearchModel) runSearch() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		query := strings.ToLower(m.queryInput.Value())
		root := m.rootInput.Value()
		if root == "" {
			root = getRootPath()
		}

		matched, scanned, err := parallelWalk(root, query, m.mode, m.skipSystemDirs)

		return searchResultMsg{
			matched: matched,
			scanned: scanned,
			err:     err,
			elapsed: time.Since(start),
		}
	}
}

// --- Helper Functions ---

func getRootPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\" // Đơn giản hóa cho ví dụ, có thể duyệt hết các ổ đĩa như bản Rust
	}
	return "/"
}

func openExplorer(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Nếu là file, dùng /select để highlight file đó trong folder
		fi, _ := os.Stat(path)
		if fi != nil && !fi.IsDir() {
			cmd = exec.Command("explorer", "/select,", filepath.Clean(path))
		} else {
			cmd = exec.Command("explorer", filepath.Clean(path))
		}
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	default: // Linux
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	}
	_ = cmd.Run()
}

// --- Tea Methods ---

func (m fileSearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m fileSearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "f2":
			m.skipSystemDirs = !m.skipSystemDirs
			if m.skipSystemDirs {
				m.status = "✔ Skip system dirs: BẬT (nhanh hơn)"
			} else {
				m.status = "✘ Skip system dirs: TẮT (quét toàn bộ)"
			}

		case "tab":
			m.focus = (m.focus + 1) % 4
			// Cập nhật trạng thái focus cho các input
			m.queryInput.Blur()
			m.rootInput.Blur()
			if m.focus == FocusQuery {
				m.queryInput.Focus()
			} else if m.focus == FocusRoot {
				m.rootInput.Focus()
			}

		case "left", "right":
			if m.focus == FocusMode {
				if msg.String() == "left" {
					m.mode = (m.mode + 2) % 3
				} else {
					m.mode = (m.mode + 1) % 3
				}
			}

		case "enter":
			if m.focus == FocusResults {
				if i, ok := m.results.SelectedItem().(item); ok {
					openExplorer(string(i))
					m.status = "Đã mở: " + string(i)
				}
			} else {
				if m.queryInput.Value() != "" {
					m.searching = true
					m.status = "Đang tìm kiếm..."
					return m, tea.Batch(m.spinner.Tick, m.runSearch())
				}
			}

		}

	// NOTE: debounce live search disabled — memory leak issue
	// case debounceMsg:
	// 	if msg.query == m.queryInput.Value() && !m.searching && len(msg.query) >= 3 {
	// 		m.searching = true
	// 		m.status = "Đang tìm kiếm..."
	// 		return m, tea.Batch(m.spinner.Tick, m.runSearch())
	// 	}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case searchResultMsg:
		m.searching = false
		m.scanned = msg.scanned
		m.lastTime = msg.elapsed
		var items []list.Item
		for _, p := range msg.matched {
			items = append(items, item(p))
		}
		m.results.SetItems(items)
		m.status = fmt.Sprintf("Tìm thấy %d kết quả (Scan %d mục) trong %.2fs", len(msg.matched), msg.scanned, m.lastTime.Seconds())
	}

	// Update components dựa trên focus
	if m.focus == FocusQuery {
		m.queryInput, cmd = m.queryInput.Update(msg)
	} else if m.focus == FocusRoot {
		m.rootInput, cmd = m.rootInput.Update(msg)
	} else if m.focus == FocusResults {
		m.results, cmd = m.results.Update(msg)
	}

	return m, cmd
}

func (m fileSearchModel) View() string {
	var b strings.Builder

	b.WriteString(fsTitleStyle.Render("🔍 FILE SEARCH TUI (GO VERSION — TURBO)") + "\n\n")

	// Render Inputs
	qStyle := blurredStyle
	if m.focus == FocusQuery {
		qStyle = focusedStyle
	}
	b.WriteString(qStyle.Render("Query: "+m.queryInput.View()) + "\n")

	rStyle := blurredStyle
	if m.focus == FocusRoot {
		rStyle = focusedStyle
	}
	b.WriteString(rStyle.Render("Root Path: "+m.rootInput.View()) + "\n")

	// Render Mode
	mStyle := blurredStyle
	if m.focus == FocusMode {
		mStyle = focusedStyle
	}
	b.WriteString(mStyle.Render(fmt.Sprintf("Mode: < %s > (Dùng ←/→)", m.mode.String())) + "\n")

	// Skip system dirs toggle
	skipLabel := "✔ BẬT"
	if !m.skipSystemDirs {
		skipLabel = "✘ TẮT"
	}
	b.WriteString(blurredStyle.Render(fmt.Sprintf("Skip System Dirs: %s (F2 để đổi)", skipLabel)) + "\n")

	// Status Line
	status := m.status
	if m.searching {
		status = m.spinner.View() + " Đang tìm..."
	}
	b.WriteString("\n" + statusStyle.Render(status) + "\n\n")

	// Results
	m.results.SetSize(80, 10)
	b.WriteString(m.results.View())

	return b.String()
}

// debounceCmd creates a tea.Cmd that sends a debounceMsg after a delay
func debounceCmd(query string, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		return debounceMsg{query: query}
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Có lỗi rồi Ben ơi: %v", err)
		os.Exit(1)
	}
}
