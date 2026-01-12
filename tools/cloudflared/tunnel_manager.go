package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TunnelSpec struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	LocalURL string `json:"localUrl"` // example: 127.0.0.1:27017 or localhost:29017
	Enabled  bool   `json:"enabled"`
}

type TunnelRuntime struct {
	spec TunnelSpec

	mu       sync.Mutex
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
	cmd      *exec.Cmd
	logBuf   bytes.Buffer
	logLines []string
}

func (tr *TunnelRuntime) IsRunning() bool {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.running
}

func (tr *TunnelRuntime) AppendLog(line string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	// Keep last ~800 lines
	tr.logLines = append(tr.logLines, line)
	if len(tr.logLines) > 800 {
		tr.logLines = tr.logLines[len(tr.logLines)-800:]
	}
	tr.logBuf.WriteString(line)
	if !strings.HasSuffix(line, "\n") {
		tr.logBuf.WriteString("\n")
	}
}

func (tr *TunnelRuntime) LogsText() string {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return strings.Join(tr.logLines, "\n")
}

func cloudflaredPath() string {
	exeName := "cloudflared"
	if runtime.GOOS == "windows" {
		exeName = "cloudflared.exe"
	}

	// 1. Check same folder as executable (for deployed binary)
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		sameFolderPath := filepath.Join(exeDir, exeName)
		if _, err := os.Stat(sameFolderPath); err == nil {
			return sameFolderPath
		}
	}

	// 2. Check project externals folder (for development)
	cwd, err := os.Getwd()
	if err == nil {
		cwdPath := filepath.Join(cwd, "..", "..", "externals", exeName)
		if _, err := os.Stat(cwdPath); err == nil {
			absPath, _ := filepath.Abs(cwdPath)
			return absPath
		}
	}

	// 3. Fall back to PATH
	return exeName
}

func (tr *TunnelRuntime) Start() error {
	tr.mu.Lock()
	if tr.running {
		tr.mu.Unlock()
		return nil
	}
	tr.ctx, tr.cancel = context.WithCancel(context.Background())

	args := []string{
		"access", "tcp",
		"--hostname", tr.spec.Hostname,
		"--url", tr.spec.LocalURL,
	}

	tr.cmd = exec.CommandContext(tr.ctx, cloudflaredPath(), args...)
	stdout, err := tr.cmd.StdoutPipe()
	if err != nil {
		tr.mu.Unlock()
		return err
	}
	stderr, err := tr.cmd.StderrPipe()
	if err != nil {
		tr.mu.Unlock()
		return err
	}

	tr.running = true
	tr.mu.Unlock()

	tr.AppendLog(fmt.Sprintf("[%s] START: cloudflared %s", tr.spec.Name, strings.Join(args, " ")))

	if err := tr.cmd.Start(); err != nil {
		tr.mu.Lock()
		tr.running = false
		tr.mu.Unlock()
		return err
	}

	// Stream logs
	readPipe := func(prefix string, r *bufio.Scanner) {
		for r.Scan() {
			tr.AppendLog(fmt.Sprintf("%s %s", prefix, r.Text()))
		}
	}

	go func() {
		sc := bufio.NewScanner(stdout)
		readPipe("OUT:", sc)
	}()
	go func() {
		sc := bufio.NewScanner(stderr)
		readPipe("ERR:", sc)
	}()

	// Wait in background
	go func() {
		err := tr.cmd.Wait()

		tr.mu.Lock()
		tr.running = false
		tr.mu.Unlock()

		if err != nil && tr.ctx.Err() != context.Canceled {
			tr.AppendLog(fmt.Sprintf("[%s] EXIT (error): %v", tr.spec.Name, err))
		} else {
			tr.AppendLog(fmt.Sprintf("[%s] EXIT", tr.spec.Name))
		}
	}()

	return nil
}

func (tr *TunnelRuntime) Stop() {
	tr.mu.Lock()
	if !tr.running {
		tr.mu.Unlock()
		return
	}
	cancel := tr.cancel
	tr.mu.Unlock()

	tr.AppendLog(fmt.Sprintf("[%s] STOP requested", tr.spec.Name))

	if cancel != nil {
		cancel()
	}
}

type AppConfig struct {
	Tunnels []TunnelSpec `json:"tunnels"`
}

func defaultConfig() AppConfig {
	return AppConfig{
		Tunnels: []TunnelSpec{
			{Name: "MongoDB", Hostname: "mongodb.benna.life", LocalURL: "127.0.0.1:27017", Enabled: true},
			{Name: "RabbitMQ", Hostname: "rabbitmq.benna.life", LocalURL: "127.0.0.1:5672", Enabled: true},
			{Name: "Kafka", Hostname: "kafka.benna.life", LocalURL: "127.0.0.1:9092", Enabled: false},
		},
	}
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tunnel-gui.json"), nil
}

func loadConfig() (AppConfig, string) {
	p, err := configPath()
	if err != nil {
		return defaultConfig(), ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return defaultConfig(), p
	}
	var cfg AppConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return defaultConfig(), p
	}
	if len(cfg.Tunnels) == 0 {
		cfg = defaultConfig()
	}
	return cfg, p
}

func saveConfig(cfg AppConfig) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0644)
}

// Bubbletea Model
type model struct {
	runtimes  []*TunnelRuntime
	cfg       AppConfig
	cursor    int
	viewMode  string // "list" or "logs"
	statusMsg string
	quitting  bool
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.runtimes)-1 {
				m.cursor++
			}

		case "enter", " ":
			if m.cursor >= 0 && m.cursor < len(m.runtimes) {
				rt := m.runtimes[m.cursor]
				if rt.IsRunning() {
					rt.Stop()
					m.statusMsg = fmt.Sprintf("Stopped: %s", rt.spec.Name)
				} else {
					if err := rt.Start(); err != nil {
						m.statusMsg = fmt.Sprintf("Error: %v", err)
					} else {
						m.statusMsg = fmt.Sprintf("Started: %s", rt.spec.Name)
					}
				}
			}

		case "s":
			count := 0
			for _, rt := range m.runtimes {
				if rt.spec.Enabled && !rt.IsRunning() {
					if err := rt.Start(); err == nil {
						count++
					}
				}
			}
			m.statusMsg = fmt.Sprintf("Started %d enabled tunnels", count)

		case "x":
			count := 0
			for _, rt := range m.runtimes {
				if rt.IsRunning() {
					rt.Stop()
					count++
				}
			}
			m.statusMsg = fmt.Sprintf("Stopped %d tunnels", count)

		case "l":
			if m.viewMode == "list" {
				m.viewMode = "logs"
			} else {
				m.viewMode = "list"
			}

		case "e":
			if m.cursor >= 0 && m.cursor < len(m.runtimes) {
				rt := m.runtimes[m.cursor]
				if rt.IsRunning() {
					m.statusMsg = "Stop tunnel first before toggling enabled"
				} else {
					rt.spec.Enabled = !rt.spec.Enabled
					m.cfg.Tunnels[m.cursor] = rt.spec
					saveConfig(m.cfg)
					status := "disabled"
					if rt.spec.Enabled {
						status = "enabled"
					}
					m.statusMsg = fmt.Sprintf("Toggled %s to %s", rt.spec.Name, status)
				}
			}

		case "?":
			if m.statusMsg == "" || !strings.HasPrefix(m.statusMsg, "Keys:") {
				m.statusMsg = "Keys: ↑↓=move enter=start/stop s=start all x=stop all l=logs e=toggle enabled q=quit"
			} else {
				m.statusMsg = ""
			}
		}

	case tickMsg:
		return m, tickCmd()
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Render("Stopping all tunnels... Goodbye! 👋\n")
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00D7FF")).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#00D7FF")).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	runningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00"))

	stoppedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000"))

	enabledStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00"))

	disabledStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFF00"))

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00D7FF")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFA500")).
		Bold(true)

	var s strings.Builder

	// Title
	s.WriteString(titleStyle.Render("🌐 Cloudflared Tunnel Manager"))
	s.WriteString("\n\n")

	if m.viewMode == "logs" {
		// Logs view
		s.WriteString(headerStyle.Render("━━━ Logs ━━━"))
		s.WriteString("\n\n")
		if m.cursor >= 0 && m.cursor < len(m.runtimes) {
			rt := m.runtimes[m.cursor]
			s.WriteString(fmt.Sprintf("Tunnel: %s\n\n", rt.spec.Name))
			logs := rt.LogsText()
			if logs == "" {
				s.WriteString(helpStyle.Render("(No logs yet)"))
			} else {
				// Show last 15 lines
				lines := strings.Split(logs, "\n")
				start := len(lines) - 15
				if start < 0 {
					start = 0
				}
				s.WriteString(strings.Join(lines[start:], "\n"))
			}
		}
	} else {
		// List view
		s.WriteString(headerStyle.Render("━━━ Tunnels ━━━"))
		s.WriteString("\n\n")

		for i, rt := range m.runtimes {
			var line string

			status := stoppedStyle.Render("● STOPPED")
			if rt.IsRunning() {
				status = runningStyle.Render("● RUNNING")
			}

			enabled := disabledStyle.Render("[DISABLED]")
			if rt.spec.Enabled {
				enabled = enabledStyle.Render("[ENABLED]")
			}

			tunnelInfo := fmt.Sprintf("%-20s %s %s", rt.spec.Name, status, enabled)
			details := fmt.Sprintf("  %s → %s", rt.spec.Hostname, rt.spec.LocalURL)

			if i == m.cursor {
				line = selectedStyle.Render(fmt.Sprintf("▶ %s", tunnelInfo))
				line += "\n" + selectedStyle.Render(details)
			} else {
				line = normalStyle.Render(fmt.Sprintf("  %s", tunnelInfo))
				line += "\n" + helpStyle.Render(details)
			}

			s.WriteString(line)
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(headerStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	s.WriteString("\n")

	// Help bar
	help := "↑↓:move  enter:start/stop  s:start-all  x:stop-all  l:logs  e:toggle-enabled  ?:help  q:quit"
	s.WriteString(helpStyle.Render(help))
	s.WriteString("\n")

	// Status message
	if m.statusMsg != "" {
		s.WriteString(statusStyle.Render(fmt.Sprintf("» %s", m.statusMsg)))
		s.WriteString("\n")
	}

	return s.String()
}

func main() {
	cfg, _ := loadConfig()

	runtimes := make([]*TunnelRuntime, 0, len(cfg.Tunnels))
	for _, t := range cfg.Tunnels {
		rt := &TunnelRuntime{spec: t}
		runtimes = append(runtimes, rt)
	}

	m := model{
		runtimes: runtimes,
		cfg:      cfg,
		cursor:   0,
		viewMode: "list",
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Stop all tunnels on exit
	for _, rt := range runtimes {
		rt.Stop()
	}
	time.Sleep(500 * time.Millisecond)
}
