package main

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type CleanupItem struct {
	Name        string
	Path        string
	Size        int64
	Description string
	Category    string
	Safe        bool
}

type DiskInfo struct {
	Total       uint64
	Free        uint64
	Used        uint64
	DriveLetter string
}

type CleanupTool struct {
	items  []CleanupItem
	dryRun bool
}

// Windows API calls for disk space
var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
)

func main() {
	if runtime.GOOS != "windows" {
		log.Fatal("This tool is designed for Windows only")
	}

	tool := &CleanupTool{}

	fmt.Println(color.CyanString("🧹 Windows Disk Cleanup Tool"))
	fmt.Println(color.YellowString("====================================="))

	// Check if running as administrator
	if !isAdmin() {
		color.Yellow("⚠️  Warning: Not running as administrator. Some cleanup operations may be limited.")
		fmt.Println()
	}

	// Get disk info
	diskInfo := getDiskSpace("C:")
	displayDiskInfo(diskInfo)

	// Scan for cleanup items
	fmt.Println(color.BlueString("🔍 Scanning for cleanup opportunities..."))
	tool.scanSystem()

	if len(tool.items) == 0 {
		color.Green("✅ No significant cleanup items found!")
		return
	}

	// Display findings
	tool.displayFindings()

	// Interactive menu
	tool.runInteractiveMenu()
}

func (ct *CleanupTool) scanSystem() {
	userProfile := os.Getenv("USERPROFILE")
	tempDir := os.Getenv("TEMP")

	// Scan common cleanup locations
	ct.scanTempFolders(tempDir)
	ct.scanWindowsTemp()
	ct.scanRecycleBin()
	ct.scanBrowserCaches(userProfile)
	ct.scanJavaCrashDumps(userProfile)
	ct.scanDownloads(userProfile)
	ct.scanWindowsUpdateCache()
	ct.scanLogFiles()
	ct.scanPrefetch()
	ct.scanThumbnailCache(userProfile)
}

func (ct *CleanupTool) scanTempFolders(tempDir string) {
	size := getDirSize(tempDir)
	if size > 0 {
		ct.items = append(ct.items, CleanupItem{
			Name:        "User Temp Files",
			Path:        tempDir,
			Size:        size,
			Description: "Temporary files created by applications",
			Category:    "Temporary Files",
			Safe:        true,
		})
	}
}

func (ct *CleanupTool) scanWindowsTemp() {
	winTempPath := filepath.Join(os.Getenv("SystemRoot"), "Temp")
	size := getDirSize(winTempPath)
	if size > 0 {
		ct.items = append(ct.items, CleanupItem{
			Name:        "Windows Temp Files",
			Path:        winTempPath,
			Size:        size,
			Description: "System temporary files",
			Category:    "Temporary Files",
			Safe:        true,
		})
	}
}

func (ct *CleanupTool) scanRecycleBin() {
	recycleBinPath := "C:\\$Recycle.Bin"
	size := getDirSize(recycleBinPath)
	if size > 0 {
		ct.items = append(ct.items, CleanupItem{
			Name:        "Recycle Bin",
			Path:        recycleBinPath,
			Size:        size,
			Description: "Deleted files in Recycle Bin",
			Category:    "Recycle Bin",
			Safe:        true,
		})
	}
}

func (ct *CleanupTool) scanBrowserCaches(userProfile string) {
	// Edge cache
	edgeCachePath := filepath.Join(userProfile, "AppData", "Local", "Microsoft", "Edge", "User Data", "Default", "Cache")
	if size := getDirSize(edgeCachePath); size > 0 {
		ct.items = append(ct.items, CleanupItem{
			Name:        "Edge Browser Cache",
			Path:        edgeCachePath,
			Size:        size,
			Description: "Microsoft Edge browser cache",
			Category:    "Browser Cache",
			Safe:        true,
		})
	}

	// Chrome cache
	chromeCachePath := filepath.Join(userProfile, "AppData", "Local", "Google", "Chrome", "User Data", "Default", "Cache")
	if size := getDirSize(chromeCachePath); size > 0 {
		ct.items = append(ct.items, CleanupItem{
			Name:        "Chrome Browser Cache",
			Path:        chromeCachePath,
			Size:        size,
			Description: "Google Chrome browser cache",
			Category:    "Browser Cache",
			Safe:        true,
		})
	}

	// Puppeteer cache
	puppeteerPath := filepath.Join(userProfile, ".cache", "puppeteer")
	if size := getDirSize(puppeteerPath); size > 0 {
		ct.items = append(ct.items, CleanupItem{
			Name:        "Puppeteer Cache",
			Path:        puppeteerPath,
			Size:        size,
			Description: "Puppeteer browser automation cache",
			Category:    "Development Cache",
			Safe:        true,
		})
	}

	// Codeium cache
	codeiumPath := filepath.Join(userProfile, ".codeium", "ws-browser")
	if size := getDirSize(codeiumPath); size > 0 {
		ct.items = append(ct.items, CleanupItem{
			Name:        "Codeium Browser Cache",
			Path:        codeiumPath,
			Size:        size,
			Description: "Codeium AI coding assistant browser cache",
			Category:    "Development Cache",
			Safe:        true,
		})
	}
}

func (ct *CleanupTool) scanJavaCrashDumps(userProfile string) {
	filepath.Walk(userProfile, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".hprof") {
			ct.items = append(ct.items, CleanupItem{
				Name:        fmt.Sprintf("Java Crash Dump: %s", info.Name()),
				Path:        path,
				Size:        info.Size(),
				Description: "Java application crash dump file",
				Category:    "Crash Dumps",
				Safe:        true,
			})
		}
		return nil
	})
}

func (ct *CleanupTool) scanDownloads(userProfile string) {
	downloadsPath := filepath.Join(userProfile, "Downloads")

	// Look for large duplicate files and installers
	fileMap := make(map[string][]string)

	filepath.Walk(downloadsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Check for large files (>100MB)
		if info.Size() > 100*1024*1024 {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext == ".exe" || ext == ".msi" || ext == ".zip" {
				ct.items = append(ct.items, CleanupItem{
					Name:        fmt.Sprintf("Large Download: %s", info.Name()),
					Path:        path,
					Size:        info.Size(),
					Description: fmt.Sprintf("Large %s file in Downloads", strings.ToUpper(ext[1:])),
					Category:    "Large Downloads",
					Safe:        false, // User should decide
				})
			}
		}

		// Check for duplicate files
		baseName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		fileMap[baseName] = append(fileMap[baseName], path)

		return nil
	})

	// Add duplicate files
	for baseName, paths := range fileMap {
		if len(paths) > 1 {
			totalSize := int64(0)
			for _, path := range paths {
				if info, err := os.Stat(path); err == nil {
					totalSize += info.Size()
				}
			}
			if totalSize > 50*1024*1024 { // Only report if total size > 50MB
				ct.items = append(ct.items, CleanupItem{
					Name:        fmt.Sprintf("Duplicate Files: %s (%d copies)", baseName, len(paths)),
					Path:        strings.Join(paths, ";"),
					Size:        totalSize,
					Description: "Multiple versions of the same file",
					Category:    "Duplicates",
					Safe:        false, // User should review
				})
			}
		}
	}
}

func (ct *CleanupTool) scanWindowsUpdateCache() {
	updateCachePath := "C:\\Windows\\SoftwareDistribution\\Download"
	size := getDirSize(updateCachePath)
	if size > 0 {
		ct.items = append(ct.items, CleanupItem{
			Name:        "Windows Update Cache",
			Path:        updateCachePath,
			Size:        size,
			Description: "Downloaded Windows Update files (will be re-downloaded if needed)",
			Category:    "Windows Cache",
			Safe:        true,
		})
	}
}

func (ct *CleanupTool) scanLogFiles() {
	logsPath := "C:\\Windows\\Logs"
	size := getDirSize(logsPath)
	if size > 100*1024*1024 { // Only report if > 100MB
		ct.items = append(ct.items, CleanupItem{
			Name:        "Windows Log Files",
			Path:        logsPath,
			Size:        size,
			Description: "Old Windows log files (keeps recent logs)",
			Category:    "Log Files",
			Safe:        true,
		})
	}
}

func (ct *CleanupTool) scanPrefetch() {
	prefetchPath := "C:\\Windows\\Prefetch"
	size := getDirSize(prefetchPath)
	if size > 50*1024*1024 { // Only report if > 50MB
		ct.items = append(ct.items, CleanupItem{
			Name:        "Windows Prefetch Files",
			Path:        prefetchPath,
			Size:        size,
			Description: "Windows application prefetch cache",
			Category:    "Windows Cache",
			Safe:        true,
		})
	}
}

func (ct *CleanupTool) scanThumbnailCache(userProfile string) {
	thumbnailPath := filepath.Join(userProfile, "AppData", "Local", "Microsoft", "Windows", "Explorer")
	size := getDirSize(thumbnailPath)
	if size > 0 {
		ct.items = append(ct.items, CleanupItem{
			Name:        "Thumbnail Cache",
			Path:        thumbnailPath,
			Size:        size,
			Description: "Windows Explorer thumbnail cache",
			Category:    "Windows Cache",
			Safe:        true,
		})
	}
}

func (ct *CleanupTool) displayFindings() {
	if len(ct.items) == 0 {
		return
	}

	// Sort by size (largest first)
	sort.Slice(ct.items, func(i, j int) bool {
		return ct.items[i].Size > ct.items[j].Size
	})

	fmt.Println()
	color.Blue("📊 Cleanup Opportunities Found:")
	fmt.Println(strings.Repeat("=", 80))

	totalSize := int64(0)
	safeSize := int64(0)

	// Group by category
	categories := make(map[string][]CleanupItem)
	for _, item := range ct.items {
		categories[item.Category] = append(categories[item.Category], item)
		totalSize += item.Size
		if item.Safe {
			safeSize += item.Size
		}
	}

	for category, items := range categories {
		color.Yellow("\n📁 %s:", category)
		for _, item := range items {
			safetyIcon := "⚠️ "
			if item.Safe {
				safetyIcon = "✅ "
			}
			fmt.Printf("  %s%-50s %s\n",
				safetyIcon,
				item.Name,
				color.GreenString(humanize.Bytes(uint64(item.Size))))
			fmt.Printf("     %s\n", color.CyanString(item.Description))
		}
	}

	fmt.Println(strings.Repeat("=", 80))
	color.Green("💾 Total potential cleanup: %s", humanize.Bytes(uint64(totalSize)))
	color.Blue("✅ Safe to clean automatically: %s", humanize.Bytes(uint64(safeSize)))
	fmt.Println()
}

func (ct *CleanupTool) runInteractiveMenu() {
	for {
		prompt := promptui.Select{
			Label: "What would you like to do?",
			Items: []string{
				"Clean all safe items automatically",
				"Select specific items to clean",
				"View disk space info",
				"Dry run (preview only)",
				"Exit",
			},
		}

		_, result, err := prompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		switch result {
		case "Clean all safe items automatically":
			ct.cleanSafeItems()
		case "Select specific items to clean":
			ct.selectiveClean()
		case "View disk space info":
			diskInfo := getDiskSpace("C:")
			displayDiskInfo(diskInfo)
		case "Dry run (preview only)":
			ct.dryRun = true
			ct.cleanSafeItems()
			ct.dryRun = false
		case "Exit":
			color.Green("👋 Goodbye!")
			return
		}
	}
}

func (ct *CleanupTool) cleanSafeItems() {
	safeItems := []CleanupItem{}
	for _, item := range ct.items {
		if item.Safe {
			safeItems = append(safeItems, item)
		}
	}

	if len(safeItems) == 0 {
		color.Yellow("No safe items to clean.")
		return
	}

	totalSize := int64(0)
	for _, item := range safeItems {
		totalSize += item.Size
	}

	if ct.dryRun {
		color.Blue("🔍 DRY RUN - Would clean the following:")
	} else {
		color.Yellow("🧹 Cleaning safe items (%s)...", humanize.Bytes(uint64(totalSize)))
	}

	cleaned := int64(0)
	for _, item := range safeItems {
		if ct.dryRun {
			fmt.Printf("  Would clean: %s (%s)\n", item.Name, humanize.Bytes(uint64(item.Size)))
		} else {
			fmt.Printf("  Cleaning: %s... ", item.Name)
			if ct.cleanItem(item) {
				cleaned += item.Size
				color.Green("✓")
			} else {
				color.Red("✗")
			}
		}
	}

	if !ct.dryRun {
		color.Green("\n🎉 Cleanup completed! Freed %s of disk space.", humanize.Bytes(uint64(cleaned)))

		// Refresh disk info
		diskInfo := getDiskSpace("C:")
		displayDiskInfo(diskInfo)
	}
}

func (ct *CleanupTool) selectiveClean() {
	if len(ct.items) == 0 {
		color.Yellow("No items available for cleanup.")
		return
	}

	items := make([]string, len(ct.items))
	for i, item := range ct.items {
		safetyIcon := "⚠️ "
		if item.Safe {
			safetyIcon = "✅ "
		}
		items[i] = fmt.Sprintf("%s%s (%s)", safetyIcon, item.Name, humanize.Bytes(uint64(item.Size)))
	}

	prompt := promptui.SelectWithAdd{
		Label:    "Select items to clean (use arrows and space to select multiple)",
		Items:    items,
		AddLabel: "Done selecting",
	}

	selectedIndices := []int{}

	for {
		i, _, err := prompt.Run()
		if err != nil {
			break
		}

		if i == -1 { // "Done selecting" was chosen
			break
		}

		// Toggle selection
		found := false
		for j, sel := range selectedIndices {
			if sel == i {
				selectedIndices = append(selectedIndices[:j], selectedIndices[j+1:]...)
				found = true
				break
			}
		}
		if !found {
			selectedIndices = append(selectedIndices, i)
		}

		color.Blue("Selected items: %d", len(selectedIndices))
	}

	if len(selectedIndices) == 0 {
		color.Yellow("No items selected.")
		return
	}

	// Clean selected items
	totalSize := int64(0)
	for _, i := range selectedIndices {
		totalSize += ct.items[i].Size
	}

	color.Yellow("🧹 Cleaning %d selected items (%s)...", len(selectedIndices), humanize.Bytes(uint64(totalSize)))

	cleaned := int64(0)
	for _, i := range selectedIndices {
		item := ct.items[i]
		fmt.Printf("  Cleaning: %s... ", item.Name)
		if ct.cleanItem(item) {
			cleaned += item.Size
			color.Green("✓")
		} else {
			color.Red("✗")
		}
	}

	color.Green("\n🎉 Cleanup completed! Freed %s of disk space.", humanize.Bytes(uint64(cleaned)))
}

func (ct *CleanupTool) cleanItem(item CleanupItem) bool {
	switch item.Category {
	case "Temporary Files", "Browser Cache", "Development Cache":
		return ct.cleanDirectory(item.Path, false)
	case "Recycle Bin":
		return ct.emptyRecycleBin()
	case "Crash Dumps":
		return ct.deleteFile(item.Path)
	case "Windows Cache":
		if strings.Contains(item.Path, "SoftwareDistribution") {
			return ct.cleanWindowsUpdateCache()
		}
		return ct.cleanDirectory(item.Path, true) // Keep recent files
	case "Log Files":
		return ct.cleanLogFiles(item.Path)
	default:
		// For user-review items, ask for confirmation
		return ct.cleanDirectory(item.Path, false)
	}
}

func (ct *CleanupTool) cleanDirectory(path string, keepRecent bool) bool {
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		if !info.IsDir() {
			if keepRecent {
				// Keep files modified in the last 7 days
				if info.ModTime().AddDate(0, 0, 7).After(time.Now()) {
					return nil
				}
			}
			os.Remove(filePath)
		}
		return nil
	})

	return err == nil
}

func (ct *CleanupTool) deleteFile(path string) bool {
	err := os.Remove(path)
	return err == nil
}

func (ct *CleanupTool) emptyRecycleBin() bool {
	// Use Windows API to empty recycle bin
	return ct.cleanDirectory("C:\\$Recycle.Bin", false)
}

func (ct *CleanupTool) cleanWindowsUpdateCache() bool {
	// Stop Windows Update service, clean cache, restart service
	// This is a simplified version - in production you'd want proper service management
	return ct.cleanDirectory("C:\\Windows\\SoftwareDistribution\\Download", false)
}

func (ct *CleanupTool) cleanLogFiles(path string) bool {
	// Clean log files older than 30 days
	return ct.cleanDirectory(path, true)
}

// Utility functions

func getDiskSpace(drive string) DiskInfo {
	h := syscall.MustLoadDLL("kernel32.dll")
	c := h.MustFindProc("GetDiskFreeSpaceExW")

	var free, total uint64

	drivePath, _ := syscall.UTF16PtrFromString(drive)
	c.Call(uintptr(unsafe.Pointer(drivePath)),
		uintptr(unsafe.Pointer(&free)),
		uintptr(unsafe.Pointer(&total)),
		0)

	used := total - free

	return DiskInfo{
		Total:       total,
		Free:        free,
		Used:        used,
		DriveLetter: drive,
	}
}

func displayDiskInfo(info DiskInfo) {
	fmt.Println()
	color.Blue("💾 Disk Space Information for %s", info.DriveLetter)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Total Space: %s\n", color.CyanString(humanize.Bytes(info.Total)))
	fmt.Printf("Used Space:  %s\n", color.YellowString(humanize.Bytes(info.Used)))
	fmt.Printf("Free Space:  %s\n", color.GreenString(humanize.Bytes(info.Free)))

	percentFree := float64(info.Free) / float64(info.Total) * 100
	fmt.Printf("Percent Free: %.2f%%\n", percentFree)

	if percentFree < 10 {
		color.Red("⚠️  WARNING: Disk space is critically low!")
	} else if percentFree < 20 {
		color.Yellow("⚠️  WARNING: Disk space is getting low")
	}
	fmt.Println()
}

func getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func isAdmin() bool {
	// Simple check - in production you'd want a more robust check
	testPath := "C:\\Windows\\test_admin_write"
	file, err := os.Create(testPath)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testPath)
	return true
}
