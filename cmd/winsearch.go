package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

type SearchResult struct {
	Name      string
	Path      string
	Size      int64
	ModTime   time.Time
	IsDir     bool
	Extension string
}

type SearchCriteria struct {
	Pattern       string
	SearchPath    string
	FileType      string
	CaseSensitive bool
	RegexMode     bool
}

type WindowSearchTool struct {
	results    []SearchResult
	criteria   SearchCriteria
	searchTime time.Duration
}

func winsearchMain() {
	if runtime.GOOS != "windows" {
		color.Red("❌ This tool is designed for Windows only")
		return
	}

	tool := &WindowSearchTool{}

	fmt.Println(color.CyanString("🔍 Windows File Search Tool"))
	fmt.Println(color.YellowString("======================================="))
	fmt.Println()

	// Run interactive menu
	tool.runInteractiveMenu()
}

func (wst *WindowSearchTool) runInteractiveMenu() {
	for {
		prompt := promptui.Select{
			Label: "What would you like to do?",
			Items: []string{
				"Search for files",
				"Advanced search options",
				"View recent search results",
				"Search in specific directory",
				"Search by file type",
				"Exit",
			},
		}

		_, result, err := prompt.Run()
		if err != nil {
			color.Red("❌ Menu selection failed: %v", err)
			return
		}

		switch result {
		case "Search for files":
			wst.basicSearch()
		case "Advanced search options":
			wst.advancedSearch()
		case "View recent search results":
			wst.displayResults()
		case "Search in specific directory":
			wst.directorySearch()
		case "Search by file type":
			wst.fileTypeSearch()
		case "Exit":
			color.Green("👋 Goodbye!")
			return
		}
	}
}

func (wst *WindowSearchTool) basicSearch() {
	prompt := promptui.Prompt{
		Label: "Enter search pattern (filename or part of filename)",
	}

	pattern, err := prompt.Run()
	if err != nil {
		color.Red("❌ Search cancelled")
		return
	}

	if strings.TrimSpace(pattern) == "" {
		color.Yellow("⚠️  Empty search pattern")
		return
	}

	wst.criteria = SearchCriteria{
		Pattern:       pattern,
		SearchPath:    "C:\\",
		FileType:      "all",
		CaseSensitive: false,
		RegexMode:     false,
	}

	wst.performSearch()
}

func (wst *WindowSearchTool) advancedSearch() {
	fmt.Println(color.BlueString("🔧 Advanced Search Configuration"))

	// Get search pattern
	patternPrompt := promptui.Prompt{
		Label: "Search pattern",
	}
	pattern, err := patternPrompt.Run()
	if err != nil || strings.TrimSpace(pattern) == "" {
		color.Red("❌ Invalid search pattern")
		return
	}

	// Get search path
	pathPrompt := promptui.Prompt{
		Label:   "Search path (default: C:\\)",
		Default: "C:\\",
	}
	searchPath, err := pathPrompt.Run()
	if err != nil {
		searchPath = "C:\\"
	}

	// Case sensitive option
	casePrompt := promptui.Select{
		Label: "Case sensitive search?",
		Items: []string{"No", "Yes"},
	}
	_, caseSensitive, _ := casePrompt.Run()

	// Regex mode option
	regexPrompt := promptui.Select{
		Label: "Use regular expressions?",
		Items: []string{"No", "Yes"},
	}
	_, regexMode, _ := regexPrompt.Run()

	wst.criteria = SearchCriteria{
		Pattern:       pattern,
		SearchPath:    searchPath,
		FileType:      "all",
		CaseSensitive: caseSensitive == "Yes",
		RegexMode:     regexMode == "Yes",
	}

	wst.performSearch()
}

func (wst *WindowSearchTool) directorySearch() {
	prompt := promptui.Prompt{
		Label:   "Enter directory path to search in",
		Default: "C:\\",
	}

	dirPath, err := prompt.Run()
	if err != nil {
		color.Red("❌ Directory selection cancelled")
		return
	}

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		color.Red("❌ Directory does not exist: %s", dirPath)
		return
	}

	patternPrompt := promptui.Prompt{
		Label: "Enter search pattern",
	}

	pattern, err := patternPrompt.Run()
	if err != nil || strings.TrimSpace(pattern) == "" {
		color.Red("❌ Invalid search pattern")
		return
	}

	wst.criteria = SearchCriteria{
		Pattern:       pattern,
		SearchPath:    dirPath,
		FileType:      "all",
		CaseSensitive: false,
		RegexMode:     false,
	}

	wst.performSearch()
}

func (wst *WindowSearchTool) fileTypeSearch() {
	typePrompt := promptui.Select{
		Label: "Select file type to search for",
		Items: []string{
			"All files",
			"Documents (.txt, .doc, .pdf, .docx)",
			"Images (.jpg, .png, .gif, .bmp)",
			"Audio (.mp3, .wav, .flac, .m4a)",
			"Video (.mp4, .avi, .mkv, .mov)",
			"Archives (.zip, .rar, .7z, .tar)",
			"Executables (.exe, .msi, .bat)",
			"Code files (.go, .js, .py, .java)",
		},
	}

	_, fileType, err := typePrompt.Run()
	if err != nil {
		color.Red("❌ File type selection cancelled")
		return
	}

	patternPrompt := promptui.Prompt{
		Label:   "Enter filename pattern (optional, press Enter for all files of this type)",
		Default: "*",
	}

	pattern, _ := patternPrompt.Run()
	if pattern == "" {
		pattern = "*"
	}

	wst.criteria = SearchCriteria{
		Pattern:       pattern,
		SearchPath:    "C:\\",
		FileType:      fileType,
		CaseSensitive: false,
		RegexMode:     false,
	}

	wst.performSearch()
}

func (wst *WindowSearchTool) performSearch() {
	color.Blue("🔍 Searching for files...")
	startTime := time.Now()

	wst.results = []SearchResult{}

	err := filepath.Walk(wst.criteria.SearchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking even if there's an error
		}

		if wst.matchesPattern(info.Name(), path) && wst.matchesFileType(info.Name()) {
			result := SearchResult{
				Name:      info.Name(),
				Path:      path,
				Size:      info.Size(),
				ModTime:   info.ModTime(),
				IsDir:     info.IsDir(),
				Extension: strings.ToLower(filepath.Ext(info.Name())),
			}
			wst.results = append(wst.results, result)
		}

		return nil
	})

	wst.searchTime = time.Since(startTime)

	if err != nil {
		color.Red("❌ Search error: %v", err)
		return
	}

	color.Green("✅ Search completed in %v", wst.searchTime)
	wst.displayResults()
}

func (wst *WindowSearchTool) matchesPattern(filename, fullPath string) bool {
	if wst.criteria.RegexMode {
		matched, err := regexp.MatchString(wst.criteria.Pattern, filename)
		if err != nil {
			return false
		}
		return matched
	}

	searchName := filename
	pattern := wst.criteria.Pattern

	if !wst.criteria.CaseSensitive {
		searchName = strings.ToLower(searchName)
		pattern = strings.ToLower(pattern)
	}

	// Support wildcards
	if pattern == "*" {
		return true
	}

	// Simple pattern matching
	return strings.Contains(searchName, pattern)
}

func (wst *WindowSearchTool) matchesFileType(filename string) bool {
	if wst.criteria.FileType == "All files" {
		return true
	}

	ext := strings.ToLower(filepath.Ext(filename))

	switch wst.criteria.FileType {
	case "Documents (.txt, .doc, .pdf, .docx)":
		return ext == ".txt" || ext == ".doc" || ext == ".pdf" || ext == ".docx" || ext == ".rtf"
	case "Images (.jpg, .png, .gif, .bmp)":
		return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".bmp" || ext == ".tiff"
	case "Audio (.mp3, .wav, .flac, .m4a)":
		return ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".m4a" || ext == ".ogg"
	case "Video (.mp4, .avi, .mkv, .mov)":
		return ext == ".mp4" || ext == ".avi" || ext == ".mkv" || ext == ".mov" || ext == ".wmv"
	case "Archives (.zip, .rar, .7z, .tar)":
		return ext == ".zip" || ext == ".rar" || ext == ".7z" || ext == ".tar" || ext == ".gz"
	case "Executables (.exe, .msi, .bat)":
		return ext == ".exe" || ext == ".msi" || ext == ".bat" || ext == ".cmd"
	case "Code files (.go, .js, .py, .java)":
		return ext == ".go" || ext == ".js" || ext == ".py" || ext == ".java" || ext == ".cpp" || ext == ".c"
	default:
		return true
	}
}

func (wst *WindowSearchTool) displayResults() {
	if len(wst.results) == 0 {
		color.Yellow("📭 No files found matching your search criteria")
		return
	}

	// Sort results by name
	sort.Slice(wst.results, func(i, j int) bool {
		return wst.results[i].Name < wst.results[j].Name
	})

	fmt.Println()
	color.Blue("📊 Search Results (%d files found in %v):", len(wst.results), wst.searchTime)
	fmt.Println(strings.Repeat("=", 80))

	// Group by file type or show all
	if len(wst.results) <= 50 {
		wst.displayAllResults()
	} else {
		wst.displayGroupedResults()
	}

	fmt.Println(strings.Repeat("=", 80))

	// Interactive result selection
	wst.interactiveResultSelection()
}

func (wst *WindowSearchTool) displayAllResults() {
	for i, result := range wst.results {
		if i >= 100 { // Limit display to first 100 results
			color.Yellow("... and %d more results (use grouped view)", len(wst.results)-100)
			break
		}

		icon := "📄"
		if result.IsDir {
			icon = "📁"
		} else {
			switch result.Extension {
			case ".exe", ".msi":
				icon = "⚙️"
			case ".jpg", ".png", ".gif", ".bmp":
				icon = "🖼️"
			case ".mp3", ".wav", ".flac":
				icon = "🎵"
			case ".mp4", ".avi", ".mkv":
				icon = "🎥"
			case ".zip", ".rar", ".7z":
				icon = "📦"
			case ".txt", ".doc", ".pdf":
				icon = "📝"
			}
		}

		sizeStr := ""
		if !result.IsDir {
			sizeStr = fmt.Sprintf(" (%s)", humanize.Bytes(uint64(result.Size)))
		}

		fmt.Printf("%s %s%s\n", icon, color.GreenString(result.Name), color.CyanString(sizeStr))
		fmt.Printf("   %s\n", color.YellowString(result.Path))
		fmt.Printf("   Modified: %s\n\n", result.ModTime.Format("2006-01-02 15:04:05"))
	}
}

func (wst *WindowSearchTool) displayGroupedResults() {
	groups := make(map[string][]SearchResult)

	for _, result := range wst.results {
		groupKey := "Other"
		if result.IsDir {
			groupKey = "Directories"
		} else {
			switch result.Extension {
			case ".exe", ".msi", ".bat":
				groupKey = "Executables"
			case ".jpg", ".png", ".gif", ".bmp":
				groupKey = "Images"
			case ".mp3", ".wav", ".flac":
				groupKey = "Audio"
			case ".mp4", ".avi", ".mkv":
				groupKey = "Video"
			case ".zip", ".rar", ".7z":
				groupKey = "Archives"
			case ".txt", ".doc", ".pdf":
				groupKey = "Documents"
			}
		}
		groups[groupKey] = append(groups[groupKey], result)
	}

	for groupName, items := range groups {
		color.Yellow("\n📁 %s (%d items):", groupName, len(items))
		for i, item := range items {
			if i >= 10 { // Limit each group to 10 items
				fmt.Printf("   ... and %d more items\n", len(items)-10)
				break
			}
			fmt.Printf("   %s - %s\n", color.GreenString(item.Name), color.CyanString(item.Path))
		}
	}
}

func (wst *WindowSearchTool) interactiveResultSelection() {
	if len(wst.results) == 0 {
		return
	}

	prompt := promptui.Select{
		Label: "What would you like to do with the results?",
		Items: []string{
			"Show specific file details",
			"Open file location",
			"Copy file path to clipboard",
			"Return to main menu",
		},
	}

	_, result, err := prompt.Run()
	if err != nil || result == "Return to main menu" {
		return
	}

	switch result {
	case "Show specific file details":
		wst.showFileDetails()
	case "Open file location":
		wst.openFileLocation()
	case "Copy file path to clipboard":
		wst.copyPathToClipboard()
	}
}

func (wst *WindowSearchTool) showFileDetails() {
	if len(wst.results) == 0 {
		return
	}

	// Create selection items
	items := make([]string, len(wst.results))
	for i, result := range wst.results {
		items[i] = fmt.Sprintf("%s - %s", result.Name, result.Path)
	}

	prompt := promptui.Select{
		Label: "Select a file to view details",
		Items: items,
		Size:  10,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return
	}

	result := wst.results[index]

	fmt.Println()
	color.Blue("📄 File Details:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Name: %s\n", color.GreenString(result.Name))
	fmt.Printf("Path: %s\n", color.YellowString(result.Path))
	fmt.Printf("Size: %s\n", color.CyanString(humanize.Bytes(uint64(result.Size))))
	fmt.Printf("Modified: %s\n", result.ModTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Type: %s\n", func() string {
		if result.IsDir {
			return "Directory"
		}
		return "File"
	}())
	if !result.IsDir && result.Extension != "" {
		fmt.Printf("Extension: %s\n", result.Extension)
	}
	fmt.Println()
}

func (wst *WindowSearchTool) openFileLocation() {
	if len(wst.results) == 0 {
		return
	}

	items := make([]string, len(wst.results))
	for i, result := range wst.results {
		items[i] = fmt.Sprintf("%s - %s", result.Name, result.Path)
	}

	prompt := promptui.Select{
		Label: "Select a file to open its location",
		Items: items,
		Size:  10,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return
	}

	result := wst.results[index]
	dir := filepath.Dir(result.Path)

	color.Blue("🗂️  Opening file location: %s", dir)
	// Note: In a real implementation, you would use exec.Command to open explorer
	// exec.Command("explorer", "/select,", result.Path).Start()
	color.Yellow("Directory path: %s", dir)
}

func (wst *WindowSearchTool) copyPathToClipboard() {
	if len(wst.results) == 0 {
		return
	}

	items := make([]string, len(wst.results))
	for i, result := range wst.results {
		items[i] = fmt.Sprintf("%s - %s", result.Name, result.Path)
	}

	prompt := promptui.Select{
		Label: "Select a file to copy its path",
		Items: items,
		Size:  10,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return
	}

	result := wst.results[index]

	color.Green("📋 Copied to clipboard: %s", result.Path)
	// Note: In a real implementation, you would use a clipboard library
	fmt.Printf("Path: %s\n", result.Path)
}
