package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type MediaCutterApp struct {
	ffmpegPath     string
	ffprobePath    string
	inputFile      string
	outputFile     string
	duration       float64
	startSeconds   float64
	endSeconds     float64
	isPreviewing   bool
	mu             sync.Mutex
	recentFiles    []string
	configFilePath string
}

// --- Helper Functions ---

// formatTime converts seconds (float) to HH:MM:SS.ms string
func formatTime(seconds float64) string {
	if seconds < 0 {
		return "00:00:00.000"
	}
	millisec := int((seconds - math.Floor(seconds)) * 1000)
	totalSeconds := int(seconds)
	hrs := totalSeconds / 3600
	mins := (totalSeconds % 3600) / 60
	secs := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hrs, mins, secs, millisec)
}

// parseTime converts HH:MM:SS.ms string to seconds (float)
func parseTime(timeStr string) float64 {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return -1
	}
	hrs, err1 := strconv.Atoi(parts[0])
	mins, err2 := strconv.Atoi(parts[1])
	secsParts := strings.Split(parts[2], ".")
	secs, err3 := strconv.Atoi(secsParts[0])
	millisec := 0
	if len(secsParts) > 1 {
		millisec, _ = strconv.Atoi(secsParts[1])
	}
	if err1 != nil || err2 != nil || err3 != nil {
		return -1
	}

	totalSeconds := float64(hrs*3600 + mins*60 + secs)
	totalSeconds += float64(millisec) / 1000.0
	return totalSeconds
}

// getMediaDuration gets media duration in seconds using ffprobe
func getMediaDuration(filepath, ffprobePath string) (float64, error) {
	cmd := exec.Command(ffprobePath, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filepath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("FFprobe error: %s", out.String())
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(out.String()), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format")
	}

	return duration, nil
}

// SaveRecentFiles saves recent files to a config file
func (app *MediaCutterApp) SaveRecentFiles() error {
	data := map[string]interface{}{
		"recent_files": app.recentFiles,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(app.configFilePath, jsonData, 0644)
}

// LoadRecentFiles loads recent files from the config file
func (app *MediaCutterApp) LoadRecentFiles() {
	app.recentFiles = []string{}
	data, err := ioutil.ReadFile(app.configFilePath)
	if err != nil {
		return
	}
	var config map[string]interface{}
	json.Unmarshal(data, &config)
	if recentFiles, ok := config["recent_files"].([]interface{}); ok {
		for _, file := range recentFiles {
			app.recentFiles = append(app.recentFiles, file.(string))
		}
	}
}

// addRecentFile adds a file to the recent files list
func (app *MediaCutterApp) addRecentFile(filepath string) {
	for i, file := range app.recentFiles {
		if file == filepath {
			app.recentFiles = append(app.recentFiles[:i], app.recentFiles[i+1:]...)
			break
		}
	}
	app.recentFiles = append([]string{filepath}, app.recentFiles...)
	if len(app.recentFiles) > 10 {
		app.recentFiles = app.recentFiles[:10]
	}
	app.SaveRecentFiles()
}

// --- Main Application Logic ---

// CutMedia cuts the media file from startSeconds to endSeconds
func (app *MediaCutterApp) CutMedia(format, quality string) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.inputFile == "" {
		return fmt.Errorf("no input file selected")
	}

	if app.outputFile == "" {
		return fmt.Errorf("no output file specified")
	}

	if app.startSeconds < 0 || app.endSeconds <= app.startSeconds || app.endSeconds > app.duration {
		return fmt.Errorf("invalid time range: start=%v, end=%v, duration=%v", app.startSeconds, app.endSeconds, app.duration)
	}

	// Prepare the ffmpeg command
	startTime := formatTime(app.startSeconds)
	duration := app.endSeconds - app.startSeconds

	// Base arguments
	args := []string{
		"-i", app.inputFile,
		"-ss", startTime,
		"-t", fmt.Sprintf("%f", duration),
		"-y", // Overwrite output file if it exists
	}

	// Add format-specific arguments if provided
	if format != "" || quality != "" {
		// Don't use copy codec when format or quality is specified
		switch quality {
		case "high":
			if strings.HasSuffix(app.outputFile, ".mp4") || format == "mp4" {
				args = append(args, "-c:v", "libx264", "-crf", "18", "-preset", "slow", "-c:a", "aac", "-b:a", "192k")
			} else if strings.HasSuffix(app.outputFile, ".mp3") || format == "mp3" {
				args = append(args, "-c:a", "libmp3lame", "-b:a", "320k")
			} else {
				args = append(args, "-q:a", "0")
			}
		case "medium":
			if strings.HasSuffix(app.outputFile, ".mp4") || format == "mp4" {
				args = append(args, "-c:v", "libx264", "-crf", "23", "-preset", "medium", "-c:a", "aac", "-b:a", "128k")
			} else if strings.HasSuffix(app.outputFile, ".mp3") || format == "mp3" {
				args = append(args, "-c:a", "libmp3lame", "-b:a", "192k")
			} else {
				args = append(args, "-q:a", "3")
			}
		case "low":
			if strings.HasSuffix(app.outputFile, ".mp4") || format == "mp4" {
				args = append(args, "-c:v", "libx264", "-crf", "28", "-preset", "fast", "-c:a", "aac", "-b:a", "96k")
			} else if strings.HasSuffix(app.outputFile, ".mp3") || format == "mp3" {
				args = append(args, "-c:a", "libmp3lame", "-b:a", "128k")
			} else {
				args = append(args, "-q:a", "5")
			}
		default:
			// Use copy codec for faster processing if no quality specified
			args = append(args, "-c", "copy")
		}
	} else {
		// Use copy codec for faster processing
		args = append(args, "-c", "copy")
	}

	// Add output file
	args = append(args, app.outputFile)

	// Print command for debugging
	fmt.Println("Executing command:", app.ffmpegPath, strings.Join(args, " "))

	// Create command
	cmd := exec.Command(app.ffmpegPath, args...)

	// Set up pipe for real-time output
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting ffmpeg: %v", err)
	}

	// Create a channel to signal when processing is done
	done := make(chan bool)

	// Process output in a goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "time=") {
				fmt.Print("\rProgress: " + strings.TrimSpace(line))
			}
		}
		done <- true
	}()

	// Wait for the command to finish
	err = cmd.Wait()
	<-done
	fmt.Println() // Print a newline after progress

	if err != nil {
		return fmt.Errorf("ffmpeg error: %v", err)
	}

	// Add to recent files
	app.addRecentFile(app.inputFile)

	return nil
}

// PreviewMedia previews the selected portion of the media
func (app *MediaCutterApp) PreviewMedia() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.inputFile == "" {
		return fmt.Errorf("no input file selected")
	}

	if app.startSeconds < 0 || app.endSeconds <= app.startSeconds || app.endSeconds > app.duration {
		return fmt.Errorf("invalid time range")
	}

	app.isPreviewing = true
	defer func() { app.isPreviewing = false }()

	// Prepare the ffplay command
	startTime := formatTime(app.startSeconds)
	duration := app.endSeconds - app.startSeconds

	args := []string{
		"-i", app.inputFile,
		"-ss", startTime,
		"-t", fmt.Sprintf("%f", duration),
		"-autoexit",
	}

	cmd := exec.Command("ffplay", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ffplay error: %v\n%s", err, stderr.String())
	}

	return nil
}

// LoadMediaFile loads a media file and gets its duration
func (app *MediaCutterApp) LoadMediaFile(filepath string) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Check if file exists
	_, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	// Get media duration
	duration, err := getMediaDuration(filepath, app.ffprobePath)
	if err != nil {
		return fmt.Errorf("error getting media duration: %v", err)
	}

	app.inputFile = filepath
	app.duration = duration
	app.startSeconds = 0
	app.endSeconds = duration

	// Add to recent files
	app.addRecentFile(filepath)

	return nil
}

// SetOutputFile sets the output file path
func (app *MediaCutterApp) SetOutputFile(filepath string) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.outputFile = filepath
}

// SetTimeRange sets the start and end times for cutting
func (app *MediaCutterApp) SetTimeRange(start, end float64) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	if start < 0 || end <= start || (app.duration > 0 && end > app.duration) {
		return fmt.Errorf("invalid time range: start=%v, end=%v, duration=%v", start, end, app.duration)
	}

	app.startSeconds = start
	app.endSeconds = end
	return nil
}

func NewMediaCutterApp() *MediaCutterApp {
	return &MediaCutterApp{
		ffmpegPath:     "ffmpeg",
		ffprobePath:    "ffprobe",
		configFilePath: filepath.Join(os.TempDir(), ".media_cutter_config.json"),
	}
}

// PrintHelp prints the help message
func printHelp() {
	fmt.Println("Media Cutter - A tool to cut video and audio files")
	fmt.Println("\nUsage:")
	fmt.Println("  media_cutter [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  -i, --input <file>       Input media file")
	fmt.Println("  -o, --output <file>      Output media file")
	fmt.Println("  -s, --start <time>       Start time (HH:MM:SS.ms)")
	fmt.Println("  -e, --end <time>         End time (HH:MM:SS.ms)")
	fmt.Println("  -p, --preview            Preview the selected portion")
	fmt.Println("  -f, --format <format>    Output format (mp4, mp3, etc.)")
	fmt.Println("  -q, --quality <quality>  Output quality (high, medium, low)")
	fmt.Println("  -I, --interactive        Run in interactive mode")
	fmt.Println("  -c, --cli                Run in command-line mode (GUI is default)")
	fmt.Println("  -h, --help               Show this help message")
	fmt.Println("\nExamples:")
	fmt.Println("  media_cutter                 (Run with graphical user interface)")
	fmt.Println("  media_cutter -c -i video.mp4 -o output.mp4 -s 00:01:30.000 -e 00:02:45.500")
	fmt.Println("  media_cutter -c -i audio.mp3 -o trimmed.mp3 -s 00:00:30.000 -e 00:01:45.000")
	fmt.Println("  media_cutter -c -i video.mp4 -p -s 00:01:30.000 -e 00:02:45.500")
	fmt.Println("  media_cutter -c -I          (Run in interactive command-line mode)")
}

// This function is now moved to media_cutter_gui.go as originalMain()
// The main function is now in media_cutter_gui.go and launches the GUI by default
func main() {
	// Check if CLI mode is explicitly requested
	useCLI := false
	for _, arg := range os.Args[1:] {
		if arg == "--cli" || arg == "-c" {
			useCLI = true
			break
		}
		if arg == "-h" || arg == "--help" {
			printHelp()
			return
		}
	}

	if useCLI {
		// Run in CLI mode
		app := NewMediaCutterApp()
		app.LoadRecentFiles()

		// Parse command line arguments
		var inputFile, outputFile, startTime, endTime, format, quality string
		var preview, showHelp, interactive bool

		args := os.Args[1:]
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "-i", "--input":
				if i+1 < len(args) {
					inputFile = args[i+1]
					i++
				}
			case "-o", "--output":
				if i+1 < len(args) {
					outputFile = args[i+1]
					i++
				}
			case "-s", "--start":
				if i+1 < len(args) {
					startTime = args[i+1]
					i++
				}
			case "-e", "--end":
				if i+1 < len(args) {
					endTime = args[i+1]
					i++
				}
			case "-f", "--format":
				if i+1 < len(args) {
					format = args[i+1]
					i++
				}
			case "-q", "--quality":
				if i+1 < len(args) {
					quality = args[i+1]
					i++
				}
			case "-p", "--preview":
				preview = true
			case "-I", "--interactive":
				interactive = true
			case "-h", "--help":
				showHelp = true
			case "-c", "--cli":
				// Already handled
			}
		}

		// Check for interactive mode
		if interactive || (len(args) == 0 && !showHelp) {
			runInteractiveMode(app)
			return
		}

		// Show help if requested
		if showHelp {
			printHelp()
			return
		}

		// Load input file
		if inputFile == "" {
			fmt.Println("Error: Input file is required")
			return
		}

		err := app.LoadMediaFile(inputFile)
		if err != nil {
			fmt.Printf("Error loading media file: %v\n", err)
			return
		}

		fmt.Printf("Loaded media file: %s (Duration: %s)\n", 
			inputFile, formatTime(app.duration))

		// Set time range if provided
		if startTime != "" {
			startSeconds := parseTime(startTime)
			if startSeconds < 0 {
				fmt.Println("Error: Invalid start time format")
				return
			}
			app.startSeconds = startSeconds
		}

		if endTime != "" {
			endSeconds := parseTime(endTime)
			if endSeconds < 0 {
				fmt.Println("Error: Invalid end time format")
				return
			}
			app.endSeconds = endSeconds
		}

		fmt.Printf("Time range: %s - %s\n", 
			formatTime(app.startSeconds), formatTime(app.endSeconds))

		// Preview if requested
		if preview {
			fmt.Println("Previewing selected portion...")
			err := app.PreviewMedia()
			if err != nil {
				fmt.Printf("Error previewing media: %v\n", err)
				return
			}
			return
		}

		// Cut media if output file is provided
		if outputFile != "" {
			app.SetOutputFile(outputFile)
			fmt.Printf("Cutting media to: %s\n", outputFile)
			err := app.CutMedia(format, quality)
			if err != nil {
				fmt.Printf("Error cutting media: %v\n", err)
				return
			}
			fmt.Println("Media cutting completed successfully!")
		} else if !preview {
			fmt.Println("Error: Output file is required for cutting")
			return
		}
	} else {
		// Launch GUI by default
		gui := NewMediaCutterGUI()
		gui.Run()
	}
}

// runInteractiveMode runs the application in interactive mode
func runInteractiveMode(app *MediaCutterApp) {
	fmt.Println("Media Cutter - Interactive Mode")
	fmt.Println("===============================")

	// Display recent files if available
	if len(app.recentFiles) > 0 {
		fmt.Println("\nRecent files:")
		for i, file := range app.recentFiles {
			fmt.Printf("%d. %s\n", i+1, file)
		}
	}

	// Get input file
	var inputFile string
	if len(app.recentFiles) > 0 {
		fmt.Print("\nEnter file number or path to input file: ")
		var input string
		fmt.Scanln(&input)

		// Check if input is a number
		if num, err := strconv.Atoi(input); err == nil && num > 0 && num <= len(app.recentFiles) {
			inputFile = app.recentFiles[num-1]
		} else {
			inputFile = input
		}
	} else {
		fmt.Print("\nEnter path to input file: ")
		fmt.Scanln(&inputFile)
	}

	// Load the file
	err := app.LoadMediaFile(inputFile)
	if err != nil {
		fmt.Printf("Error loading media file: %v\n", err)
		return
	}

	fmt.Printf("\nLoaded media file: %s (Duration: %s)\n", 
		inputFile, formatTime(app.duration))

	// Get start time
	fmt.Print("\nEnter start time (HH:MM:SS.ms) [00:00:00.000]: ")
	var startTimeStr string
	fmt.Scanln(&startTimeStr)

	if startTimeStr != "" {
		startSeconds := parseTime(startTimeStr)
		if startSeconds < 0 {
			fmt.Println("Invalid start time format. Using 00:00:00.000")
			app.startSeconds = 0
		} else {
			app.startSeconds = startSeconds
		}
	}

	// Get end time
	fmt.Printf("\nEnter end time (HH:MM:SS.ms) [%s]: ", formatTime(app.duration))
	var endTimeStr string
	fmt.Scanln(&endTimeStr)

	if endTimeStr != "" {
		endSeconds := parseTime(endTimeStr)
		if endSeconds < 0 || endSeconds > app.duration {
			fmt.Printf("Invalid end time. Using %s\n", formatTime(app.duration))
			app.endSeconds = app.duration
		} else {
			app.endSeconds = endSeconds
		}
	}

	fmt.Printf("\nTime range: %s - %s\n", 
		formatTime(app.startSeconds), formatTime(app.endSeconds))

	// Ask for preview
	fmt.Print("\nPreview selection? (y/n): ")
	var previewResponse string
	fmt.Scanln(&previewResponse)

	if strings.ToLower(previewResponse) == "y" || strings.ToLower(previewResponse) == "yes" {
		fmt.Println("Previewing selected portion...")
		err := app.PreviewMedia()
		if err != nil {
			fmt.Printf("Error previewing media: %v\n", err)
		}
	}

	// Get output file
	fmt.Print("\nEnter output file path: ")
	var outputFile string
	fmt.Scanln(&outputFile)

	if outputFile == "" {
		fmt.Println("No output file specified. Exiting.")
		return
	}

	app.SetOutputFile(outputFile)

	// Get quality
	fmt.Print("\nSelect quality (high/medium/low) [medium]: ")
	var quality string
	fmt.Scanln(&quality)

	if quality == "" {
		quality = "medium"
	}

	// Get format (optional)
	fmt.Print("\nSpecify format (leave empty to detect from output file): ")
	var format string
	fmt.Scanln(&format)

	// Cut the media
	fmt.Printf("\nCutting media to: %s\n", outputFile)
	err = app.CutMedia(format, quality)
	if err != nil {
		fmt.Printf("Error cutting media: %v\n", err)
		return
	}

	fmt.Println("Media cutting completed successfully!")
}
