package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// GitRepoAnalyzerGUI represents the GUI for the git repository analyzer
type GitRepoAnalyzerGUI struct {
	app    fyne.App
	window fyne.Window

	selectedPathLabel *widget.Label
	analyzeButton     *widget.Button
}

// isGitRepo checks if the given path is a valid git repository
func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// analyzeContributions runs git shortlog to get commit counts per author
func analyzeContributions(repoPath string) (map[string]int, error) {
	cmd := exec.Command("git", "shortlog", "-s", "-n", "--all", "--no-merges")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	contribs := make(map[string]int)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		count := parts[0]
		name := strings.Join(parts[1:], " ")
		var n int
		fmt.Sscanf(count, "%d", &n)
		contribs[name] = n
	}
	return contribs, nil
}

// showContributions displays the contribution percentages in a dialog
func showContributions(win fyne.Window, contribs map[string]int) {
	total := 0
	for _, v := range contribs {
		total += v
	}
	if total == 0 {
		dialog.ShowInformation("No Data", "No contributions found.", win)
		return
	}
	var result strings.Builder
	for name, count := range contribs {
		percent := float64(count) / float64(total) * 100
		result.WriteString(fmt.Sprintf("%s: %d commits (%.2f%%)\n", name, count, percent))
	}
	dialog.ShowInformation("Contributions", result.String(), win)
}

// NewGitRepoAnalyzerGUI creates and shows the GUI
func NewGitRepoAnalyzerGUI() {
	a := app.New()
	w := a.NewWindow("Git Repo Analyzer")

	selectedPathLabel := widget.NewLabel("No folder selected.")
	analyzeButton := widget.NewButton("Analyze Contributions", nil)
	analyzeButton.Disable()

	selectButton := widget.NewButton("Select Git Repo Folder", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil {
				return
			}
			path := uri.Path()
			if !isGitRepo(path) {
				dialog.ShowError(fmt.Errorf("Selected folder is not a git repository."), w)
				selectedPathLabel.SetText("No folder selected.")
				analyzeButton.Disable()
				return
			}
			selectedPathLabel.SetText("Selected: " + path)
			analyzeButton.Enable()
			analyzeButton.OnTapped = func() {
				contribs, err := analyzeContributions(path)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				showContributions(w, contribs)
			}
		}, w)
	})

	w.SetContent(container.NewVBox(
		widget.NewLabel("Git Repository Analyzer"),
		selectButton,
		selectedPathLabel,
		analyzeButton,
	))
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}

func main() {
	NewGitRepoAnalyzerGUI()
}
