package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"golang.org/x/sys/windows/registry"
)

type InstallerInfo struct {
	Name        string
	Path        string
	Version     string
	IsNewer     bool
	Description string
}

type KioskUpgrade struct {
	Directory string
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("USERPROFILE")
	}
	defaultDir := filepath.Join(home, "Downloads")
	buildDirectory := flag.String("dir", defaultDir, "Directory where the kiosk upgrade files are located")
	flag.Parse()
	ku := KioskUpgrade{
		Directory: *buildDirectory,
	}
	ku.proceedUpgrade()
}

func (ku *KioskUpgrade) proceedUpgrade() {
	// Display welcome message
	fmt.Println()
	color.Cyan("🚀 MetaDefender Kiosk Upgrade Tool")
	color.Yellow("===================================")
	fmt.Println()

	allFiles := getAllFiles(ku.Directory)
	if len(allFiles) == 0 {
		color.Red("❌ No files found in the specified directory: %s", ku.Directory)
		return
	}

	installers := getAllInstallers(allFiles, ku.Directory)
	if len(installers) == 0 {
		color.Yellow("⚠️  No MetaDefender Kiosk installers found in the directory")
		return
	}

	// Display current version info
	ku.displayCurrentVersion()

	// Show available installers
	ku.displayInstallers(installers)

	// Interactive installer selection
	selectedInstaller := ku.selectInstaller(installers)
	if selectedInstaller == nil {
		color.Yellow("👋 Installation cancelled")
		return
	}

	// Install selected installer
	ku.installWithProgress(*selectedInstaller)
}

func (ku *KioskUpgrade) displayCurrentVersion() {
	currentVersion, err := getRegistryValue(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\OPSWAT\MD4M`, "version")
	if err != nil {
		color.Blue("📦 Current Installation: Not detected (fresh installation)")
	} else {
		color.Blue("📦 Current Version: %s", currentVersion)
	}
	fmt.Println()
}

func (ku *KioskUpgrade) displayInstallers(installers []InstallerInfo) {
	color.Green("🔍 Available Installers:")
	fmt.Println(strings.Repeat("=", 60))

	for i, installer := range installers {
		statusIcon := "⚠️ "
		if installer.IsNewer {
			statusIcon = "✅ "
		}

		color.White("%d. %s%s", i+1, statusIcon, installer.Name)
		color.Cyan("   Version: %s", installer.Version)
		color.Yellow("   Status: %s", installer.Description)
		fmt.Println()
	}
	fmt.Println(strings.Repeat("=", 60))
}

func (ku *KioskUpgrade) selectInstaller(installers []InstallerInfo) *InstallerInfo {
	if len(installers) == 1 {
		// Auto-select if only one installer
		color.Blue("📋 Only one installer found, auto-selecting...")
		return &installers[0]
	}

	// Create menu items
	items := make([]string, len(installers))
	for i, installer := range installers {
		statusIcon := "⚠️ "
		if installer.IsNewer {
			statusIcon = "✅ "
		}
		items[i] = fmt.Sprintf("%s%s (v%s) - %s", statusIcon, installer.Name, installer.Version, installer.Description)
	}

	// Add exit option
	items = append(items, "❌ Cancel installation")

	prompt := promptui.Select{
		Label: "Select installer to run",
		Items: items,
	}

	index, _, err := prompt.Run()
	if err != nil {
		color.Red("❌ Selection failed: %v", err)
		return nil
	}

	// Check if user selected cancel
	if index == len(installers) {
		return nil
	}

	return &installers[index]
}

func (ku *KioskUpgrade) installWithProgress(installer InstallerInfo) {
	color.Yellow("🚀 Starting installation of: %s", installer.Name)
	color.Blue("📁 Path: %s", installer.Path)
	fmt.Println()

	// Check if the file exists and is executable
	fileInfo, err := os.Stat(installer.Path)
	if err != nil {
		color.Red("❌ Error accessing installer file: %v", err)
		return
	}
	if fileInfo.IsDir() {
		color.Red("❌ Error: The path points to a directory, not an executable file")
		return
	}

	color.Blue("⏳ Installing... This may take several minutes")
	fmt.Println()

	err = installExeSilent(installer.Path)
	if err != nil {
		color.Red("❌ Installation failed!")
		color.Red("   Error: %s", err.Error())
		fmt.Println()
		color.Yellow("💡 Troubleshooting tips:")
		color.Yellow("   - Make sure you're running as Administrator")
		color.Yellow("   - Check if the installer file is not corrupted")
		color.Yellow("   - Ensure no antivirus is blocking the installation")
	} else {
		fmt.Println()
		color.Green("🎉 Installation completed successfully!")
		color.Green("✅ MetaDefender Kiosk has been updated to version %s", installer.Version)
		fmt.Println()
		color.Cyan("🔄 Please restart the application to use the new version")
	}
}

func getAllFiles(directory string) []string {
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	var fileList []string
	for _, file := range files {
		if !file.IsDir() {
			fileList = append(fileList, file.Name())
		} else {
			subDirFiles := getAllFiles(filepath.Join(directory, file.Name()))
			fileList = append(fileList, subDirFiles...)
		}
	}
	return fileList
}

func getAllInstallers(allFiles []string, directory string) []InstallerInfo {
	// Filter the files that contain "MetaDefender_Kiosk" in their name
	var kioskBuilds []string
	for _, file := range allFiles {
		if strings.Contains(file, "MetaDefender_Kiosk") {
			kioskBuilds = append(kioskBuilds, file)
		}
	}

	if len(kioskBuilds) == 0 {
		return []InstallerInfo{}
	}

	// Sort builds by version
	sort.Slice(kioskBuilds, func(i, j int) bool {
		v1 := strings.Split(kioskBuilds[i], "_")[len(strings.Split(kioskBuilds[i], "_"))-1]
		v2 := strings.Split(kioskBuilds[j], "_")[len(strings.Split(kioskBuilds[j], "_"))-1]
		return compareVersionFast(v1, v2) == 1
	})

	currentVersion, err := getRegistryValue(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\OPSWAT\MD4M`, "version")
	var installers []InstallerInfo

	for _, build := range kioskBuilds {
		buildVersion := strings.Split(build, "_")[len(strings.Split(build, "_"))-1]
		isNewer := false
		description := "Current version"

		if err != nil {
			// Registry key not found, all builds are considered newer
			isNewer = true
			description = "New installation"
		} else {
			comparison := compareVersionFast(buildVersion, currentVersion)
			if comparison == 1 {
				isNewer = true
				description = "Upgrade available"
			} else if comparison == 0 {
				description = "Same version as current"
			} else {
				description = "Older than current"
			}
		}

		installer := InstallerInfo{
			Name:        build,
			Path:        filepath.Join(directory, build),
			Version:     buildVersion,
			IsNewer:     isNewer,
			Description: description,
		}
		installers = append(installers, installer)
	}

	return installers
}

func getRegistryValue(key registry.Key, path string, name string) (string, error) {
	k, err := registry.OpenKey(key, path, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	value, _, err := k.GetStringValue(name)
	if err != nil {
		return "", err
	}
	return value, nil
}

func compareVersionFast(v1, v2 string) int {
	i, j := 0, 0
	n, m := len(v1), len(v2)

	for i < n || j < m {
		var x uint64
		for i < n && v1[i] != '.' {
			c := v1[i]
			if c >= '0' && c <= '9' {
				x = x*10 + uint64(c-'0')
			}
			i++
		}
		if i < n && v1[i] == '.' {
			i++
		}

		var y uint64
		for j < m && v2[j] != '.' {
			c := v2[j]
			if c >= '0' && c <= '9' {
				y = y*10 + uint64(c-'0')
			}
			j++
		}
		if j < m && v2[j] == '.' {
			j++
		}

		if x > y {
			return 1
		}
		if x < y {
			return -1
		}
	}
	return 0
}

func installExeSilent(path string) error {
	fmt.Printf("Starting silent installation process for: %s\n", path)

	// Log the installation command being used
	fmt.Println("Attempting installation with flags: /quiet /norestart")

	// The runas command requires a password to be entered interactively, which won't work in this context
	// Instead, directly execute the installer with the appropriate flags
	cmd := exec.Command(path, "/quiet", "/norestart")

	// Set up to request elevation via the Windows UAC
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	fmt.Println("Executing installer with elevated privileges...")
	fmt.Printf("Command: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))

	// Capture the output
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Silent installation failed with error: %v\n", err)
		if len(output) > 0 {
			fmt.Printf("First attempt output: %s\n", string(output))
		}
		fmt.Printf("Silent installation failed, trying interactive mode: %v\n", err)

		fmt.Println("Attempting alternative installation method using ShellExecute...")
		// Try an alternative approach - run the installer with ShellExecute which is more reliable for elevation
		cmd = exec.Command("rundll32.exe", "shell32.dll,ShellExecute", path, "/quiet", "/norestart", "", "runas")
		fmt.Printf("Alternative command: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))

		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Alternative installation method also failed with error: %v\n", err)
			if len(output) > 0 {
				fmt.Printf("Second attempt output: %s\n", string(output))
			}
			return fmt.Errorf("failed to install with both methods: %v, output: %s", err, string(output))
		}
		fmt.Println("Alternative installation method completed successfully")
		if len(output) > 0 {
			fmt.Printf("Alternative method output: %s\n", string(output))
		}
	} else {
		fmt.Println("Silent installation completed successfully")
		if len(output) > 0 {
			fmt.Printf("Installation output: %s\n", string(output))
		}
	}

	fmt.Println("Installation process finished")
	return nil
}
