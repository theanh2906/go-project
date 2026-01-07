package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Live Reload Tool - Automatically runs your code when file changes are detected")
		fmt.Println("Usage: go run live_reload.go <filename>")
		fmt.Println("\nSupported file extensions:")
		fmt.Println("  Compiled: .go, .java, .cpp, .c, .cs")
		fmt.Println("  Interpreted: .py, .js, .ts, .rb, .php, .pl, .sh, .bat, .ps1")
		fmt.Println("  Web: .html, .css")
		fmt.Println("  Other: .r, .swift, .kt, .rs")
		return
	}

	filename := os.Args[1]

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("Error: File '%s' does not exist\n", filename)
		return
	}

	var commandProcess *exec.Cmd
	lastModifiedTime := getModifiedTime(filename)

	// Initial run of the file
	extension := strings.Split(filename, ".")
	if len(extension) < 2 {
		fmt.Println("Invalid filename format. Expected format: filename.extension")
		return
	}

	buildCommand := LanguageBuildCommandMap(filename)
	if buildCommand == "" {
		fmt.Printf("Unsupported file extension: %s\n", extension[len(extension)-1])
		return
	}

	fmt.Println("Starting live reload...")
	process, err := runCommand(buildCommand)
	if err != nil {
		fmt.Printf("Initial run failed: %v\n", err)
	} else {
		commandProcess = process
	}

	liveReload(filename, lastModifiedTime, commandProcess)
}

func liveReload(filename string, lastModifiedTime time.Time, commandProcess *exec.Cmd) {
	// Get file extension
	parts := strings.Split(filename, ".")
	extension := parts[len(parts)-1]

	// Get build command (already validated in main)
	buildCommand := LanguageBuildCommandMap(extension)

	fmt.Printf("Watching %s for changes. Press Ctrl+C to stop.\n", filename)

	for {
		// Check if file still exists
		fileInfo, err := os.Stat(filename)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("File %s no longer exists. Exiting...\n", filename)
				return
			}
			fmt.Printf("Error getting file info: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Check if file was modified
		modifiedTime := fileInfo.ModTime()
		if modifiedTime.After(lastModifiedTime) {
			fmt.Printf("File changed at %s\n", modifiedTime.Format("15:04:05"))

			// Kill previous process if it exists
			if commandProcess != nil && commandProcess.Process != nil {
				err := commandProcess.Process.Kill()
				if err != nil {
					fmt.Printf("Error killing previous process: %v\n", err)
				}
			}

			// Run the build command
			process, err := runCommand(buildCommand)
			if err != nil {
				fmt.Printf("Error running command: %v\n", err)
			} else {
				commandProcess = process
				fmt.Println("Build completed. Watching for changes...")
			}

			// Update the last modified time
			lastModifiedTime = modifiedTime
		}

		// Sleep to reduce CPU usage
		time.Sleep(500 * time.Millisecond)
	}
}

func LanguageBuildCommandMap(filename string) string {
	strPart := strings.Split(filename, ".")
	extension := strPart[len(strPart)-1]
	commandMap := map[string]string{
		// Compiled languages
		"go":   fmt.Sprintf("go run %s", filename),
		"java": fmt.Sprintf("javac %s && java %s", filename, strings.TrimSuffix(filename, ".java")),
		"cpp":  fmt.Sprintf("g++ -o %s.exe %s && %s.exe", strings.TrimSuffix(filename, ".cpp"), filename, strings.TrimSuffix(filename, ".cpp")),
		"c":    fmt.Sprintf("gcc -o %s.exe %s && %s.exe", strings.TrimSuffix(filename, ".c"), filename, strings.TrimSuffix(filename, ".c")),
		"cs":   fmt.Sprintf("csc %s && %s.exe", filename, strings.TrimSuffix(filename, ".cs")),

		// Interpreted languages
		"py":  fmt.Sprintf("python %s", filename),
		"js":  fmt.Sprintf("node %s", filename),
		"ts":  fmt.Sprintf("tsc %s && node %s.js", filename, strings.TrimSuffix(filename, ".ts")),
		"rb":  fmt.Sprintf("ruby %s", filename),
		"php": fmt.Sprintf("php %s", filename),
		"pl":  fmt.Sprintf("perl %s", filename),
		"sh":  fmt.Sprintf("bash %s", filename),
		"bat": fmt.Sprintf("%s", filename),
		"ps1": fmt.Sprintf("powershell -File %s", filename),

		// Web languages
		"html": fmt.Sprintf("start %s", filename),
		"css":  fmt.Sprintf("start %s", filename),

		// Other languages
		"r":     fmt.Sprintf("Rscript %s", filename),
		"swift": fmt.Sprintf("swift %s", filename),
		"kt":    fmt.Sprintf("kotlinc %s -include-runtime -d %s.jar && java -jar %s.jar", filename, strings.TrimSuffix(filename, ".kt"), strings.TrimSuffix(filename, ".kt")),
		"rs":    fmt.Sprintf("rustc %s && %s.exe", filename, strings.TrimSuffix(filename, ".rs")),
	}

	command, exists := commandMap[extension]
	if !exists {
		return ""
	}
	return command
}

func runCommand(cmd string) (*exec.Cmd, error) {
	// Execute the command
	fmt.Printf("Running command: %s\n", cmd)

	// Split the command into program and arguments
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	program := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	// Create and configure the command
	process := exec.Command(program, args...)
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr

	// Start the command (non-blocking)
	err := process.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}

	return process, nil
}

func getModifiedTime(filename string) time.Time {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		fmt.Printf("Warning: Could not get modification time for %s: %v\n", filename, err)
		return time.Time{}
	}
	return fileInfo.ModTime()
}
