package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PortInfo represents information about a port and its associated process
type PortInfo struct {
	Port        int
	Protocol    string
	State       string
	ProcessName string
	PID         int
	LocalAddr   string
	RemoteAddr  string
}

// Model represents the application state for the TUI
type Model struct {
	ports         []PortInfo
	selectedIndex int
	loading       bool
	error         string
	lastUpdate    time.Time
}

// Messages for the TUI
type PortsLoadedMsg []PortInfo
type ErrorMsg string
type TickMsg time.Time

// runPowershellCommand executes a PowerShell command and returns the output
func runPowershellCommand(command string) (string, error) {
	cmd := exec.Command("powershell", "-Command", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run PowerShell command %s: %w", command, err)
	}
	return string(output), nil
}

// getPortInformation retrieves port information using netstat
func getPortInformation() ([]PortInfo, error) {
	// Use netstat to get network connections with process information
	cmd := "netstat -ano | Select-String -Pattern 'TCP|UDP'"
	output, err := runPowershellCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get port information: %w", err)
	}

	return parseNetstatOutput(output)
}

// parseNetstatOutput parses the netstat command output
func parseNetstatOutput(output string) ([]PortInfo, error) {
	lines := strings.Split(output, "\n")
	var ports []PortInfo
	
	// Regex to parse netstat output
	re := regexp.MustCompile(`^\s*(TCP|UDP)\s+([^\s]+):(\d+)\s+([^\s]*)\s+([^\s]*)\s+(\d+)`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 7 {
			port, err := strconv.Atoi(matches[3])
			if err != nil {
				continue
			}
			
			pid, err := strconv.Atoi(matches[6])
			if err != nil {
				continue
			}
			
			// Get process name from PID
			processName := getProcessName(pid)
			
			portInfo := PortInfo{
				Protocol:    matches[1],
				LocalAddr:   matches[2],
				Port:        port,
				RemoteAddr:  matches[4],
				State:       matches[5],
				PID:         pid,
				ProcessName: processName,
			}
			
			ports = append(ports, portInfo)
		}
	}
	
	// Sort by port number
	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Port < ports[j].Port
	})
	
	return ports, nil
}

// getProcessName gets the process name for a given PID
func getProcessName(pid int) string {
	cmd := fmt.Sprintf("(Get-Process -Id %d -ErrorAction SilentlyContinue).ProcessName", pid)
	output, err := runPowershellCommand(cmd)
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(output)
}

// killProcess terminates a process by PID
func killProcess(pid int) error {
	cmd := fmt.Sprintf("Stop-Process -Id %d -Force", pid)
	_, err := runPowershellCommand(cmd)
	return err
}

// Styles for the TUI
var (
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Padding(0, 1)

	selectedRowStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("15"))

	normalRowStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
)

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadPorts(),
		tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
			return TickMsg(t)
		}),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
		case "down", "j":
			if m.selectedIndex < len(m.ports)-1 {
				m.selectedIndex++
			}
		case "r":
			m.loading = true
			m.error = ""
			return m, loadPorts()
		case "x", "delete":
			if len(m.ports) > 0 && m.selectedIndex < len(m.ports) {
				pid := m.ports[m.selectedIndex].PID
				err := killProcess(pid)
				if err != nil {
					m.error = fmt.Sprintf("Failed to kill process %d: %v", pid, err)
				} else {
					// Refresh the list after killing process
					m.loading = true
					m.error = ""
					return m, loadPorts()
				}
			}
		case "enter":
			if len(m.ports) > 0 && m.selectedIndex < len(m.ports) {
				pid := m.ports[m.selectedIndex].PID
				err := killProcess(pid)
				if err != nil {
					m.error = fmt.Sprintf("Failed to kill process %d: %v", pid, err)
				} else {
					// Refresh the list after killing process
					m.loading = true
					m.error = ""
					return m, loadPorts()
				}
			}
		}

	case PortsLoadedMsg:
		m.ports = []PortInfo(msg)
		m.loading = false
		m.error = ""
		m.lastUpdate = time.Now()
		if m.selectedIndex >= len(m.ports) {
			m.selectedIndex = len(m.ports) - 1
		}
		if m.selectedIndex < 0 {
			m.selectedIndex = 0
		}

	case ErrorMsg:
		m.error = string(msg)
		m.loading = false

	case TickMsg:
		// Auto-refresh every 5 seconds
		if !m.loading {
			m.loading = true
			return m, loadPorts()
		}
		return m, tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})
	}

	return m, nil
}

// View renders the TUI
func (m Model) View() string {
	var s strings.Builder

	// Title
	s.WriteString(headerStyle.Render("Port Manager - Network Connections"))
	s.WriteString("\n\n")

	if m.loading {
		s.WriteString("Loading port information...\n")
		return s.String()
	}

	if m.error != "" {
		s.WriteString(errorStyle.Render("Error: " + m.error))
		s.WriteString("\n\n")
	}

	// Table header
	header := fmt.Sprintf("%-8s %-8s %-20s %-15s %-8s %-15s", 
		"Protocol", "Port", "Local Address", "Remote Address", "PID", "Process")
	s.WriteString(headerStyle.Render(header))
	s.WriteString("\n")

	// Table rows
	for i, port := range m.ports {
		row := fmt.Sprintf("%-8s %-8d %-20s %-15s %-8d %-15s",
			port.Protocol,
			port.Port,
			truncateString(port.LocalAddr, 20),
			truncateString(port.RemoteAddr, 15),
			port.PID,
			truncateString(port.ProcessName, 15))

		if i == m.selectedIndex {
			s.WriteString(selectedRowStyle.Render(row))
		} else {
			s.WriteString(normalRowStyle.Render(row))
		}
		s.WriteString("\n")
	}

	// Status and help
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("Total connections: %d | Last update: %s\n",
		len(m.ports), m.lastUpdate.Format("15:04:05")))
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Controls: ↑/↓ or j/k to navigate | Enter/x to kill process | r to refresh | q to quit"))

	return s.String()
}

// loadPorts loads port information asynchronously
func loadPorts() tea.Cmd {
	return func() tea.Msg {
		ports, err := getPortInformation()
		if err != nil {
			return ErrorMsg(err.Error())
		}
		return PortsLoadedMsg(ports)
	}
}

// truncateString truncates a string to the specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	if length <= 3 {
		return s[:length]
	}
	return s[:length-3] + "..."
}

func main() {
	// Initialize the model
	m := Model{
		loading:    true,
		lastUpdate: time.Now(),
	}

	// Create the Bubbletea program
	p := tea.NewProgram(&m, tea.WithAltScreen())
	
	// Start the program
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
