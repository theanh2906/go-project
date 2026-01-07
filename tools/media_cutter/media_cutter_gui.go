package main

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MediaCutterGUI represents the GUI for the media cutter application
type MediaCutterGUI struct {
	app          fyne.App
	window       fyne.Window
	mediaCutter  *MediaCutterApp
	isProcessing bool

	// Input fields
	inputFileLabel  *widget.Label
	outputFileLabel *widget.Label
	startTimeEntry  *widget.Entry
	endTimeEntry    *widget.Entry
	qualitySelect   *widget.Select
	formatEntry     *widget.Entry

	// Media info
	durationLabel  *widget.Label
	timeRangeLabel *widget.Label

	// Control buttons
	selectInputButton  *widget.Button
	selectOutputButton *widget.Button
	previewButton      *widget.Button
	cutButton          *widget.Button

	// Recent files
	recentFilesSelect *widget.Select

	// Output and progress
	outputTextArea *widget.TextGrid
	progressBar    *widget.ProgressBar
	statusLabel    *widget.Label
}

// NewMediaCutterGUI creates a new GUI for the media cutter
func NewMediaCutterGUI() *MediaCutterGUI {
	gui := &MediaCutterGUI{
		app:          app.New(),
		mediaCutter:  NewMediaCutterApp(),
		isProcessing: false,
	}

	// Set custom theme with vibrant colors
	gui.app.Settings().SetTheme(NewMediaCutterTheme())

	gui.window = gui.app.NewWindow("Media Cutter")
	gui.window.Resize(fyne.NewSize(900, 700))

	// Load recent files
	gui.mediaCutter.LoadRecentFiles()

	gui.createUI()

	return gui
}

// Run starts the GUI
func (g *MediaCutterGUI) Run() {
	g.window.ShowAndRun()
}

// createUI creates the user interface
func (g *MediaCutterGUI) createUI() {
	// Create input fields
	g.createInputFields()

	// Create output area
	g.createOutputArea()

	// Create control buttons
	g.createControlButtons()

	// Create status bar
	g.createStatusBar()

	// Layout everything
	g.layoutUI()
}

// createInputFields creates the input fields for the media cutter configuration
func (g *MediaCutterGUI) createInputFields() {
	// Input file
	g.inputFileLabel = widget.NewLabel("No file selected")
	g.inputFileLabel.Wrapping = fyne.TextWrapBreak

	// Output file
	g.outputFileLabel = widget.NewLabel("No output file selected")
	g.outputFileLabel.Wrapping = fyne.TextWrapBreak

	// Start time
	g.startTimeEntry = widget.NewEntry()
	g.startTimeEntry.SetPlaceHolder("00:00:00.000")

	// End time
	g.endTimeEntry = widget.NewEntry()
	g.endTimeEntry.SetPlaceHolder("00:00:00.000")

	// Quality
	g.qualitySelect = widget.NewSelect([]string{"high", "medium", "low"}, func(value string) {})
	g.qualitySelect.SetSelected("medium")

	// Format
	g.formatEntry = widget.NewEntry()
	g.formatEntry.SetPlaceHolder("Auto-detect from output file")

	// Duration
	g.durationLabel = widget.NewLabel("Duration: 00:00:00.000")

	// Time range
	g.timeRangeLabel = widget.NewLabel("Time range: 00:00:00.000 - 00:00:00.000")

	// Recent files
	recentFiles := []string{"Select a recent file..."}
	if len(g.mediaCutter.recentFiles) > 0 {
		recentFiles = append(recentFiles, g.mediaCutter.recentFiles...)
	}
	g.recentFilesSelect = widget.NewSelect(recentFiles, func(value string) {
		if value != "Select a recent file..." {
			g.loadMediaFile(value)
		}
	})
	g.recentFilesSelect.SetSelected("Select a recent file...")
}

// createOutputArea creates the output area for the media cutter results
func (g *MediaCutterGUI) createOutputArea() {
	g.outputTextArea = widget.NewTextGrid()
	g.outputTextArea.SetText("Media Cutter\n===========\n\nSelect a media file to begin.")
}

// createControlButtons creates the control buttons
func (g *MediaCutterGUI) createControlButtons() {
	g.selectInputButton = widget.NewButtonWithIcon("Select Input", theme.FolderOpenIcon(), func() {
		g.selectInputFile()
	})

	g.selectOutputButton = widget.NewButtonWithIcon("Select Output", theme.DocumentSaveIcon(), func() {
		g.selectOutputFile()
	})

	g.previewButton = widget.NewButtonWithIcon("Preview", theme.MediaPlayIcon(), func() {
		g.previewMedia()
	})
	g.previewButton.Disable()

	g.cutButton = widget.NewButtonWithIcon("Cut Media", theme.ContentCutIcon(), func() {
		g.cutMedia()
	})
	g.cutButton.Disable()
}

// createStatusBar creates the status bar
func (g *MediaCutterGUI) createStatusBar() {
	g.statusLabel = widget.NewLabel("Ready")
	g.progressBar = widget.NewProgressBar()
	g.progressBar.SetValue(0)
}

// layoutUI lays out the user interface
func (g *MediaCutterGUI) layoutUI() {
	// Create header with logo
	logo := canvas.NewText("Media Cutter", theme.PrimaryColor())
	logo.TextSize = 24
	logo.TextStyle = fyne.TextStyle{Bold: true}

	header := container.NewVBox(
		container.NewCenter(logo),
		widget.NewSeparator(),
	)

	// Create form for input fields
	inputFileBox := container.NewBorder(nil, nil, widget.NewLabel("Input File:"), g.selectInputButton, g.inputFileLabel)
	outputFileBox := container.NewBorder(nil, nil, widget.NewLabel("Output File:"), g.selectOutputButton, g.outputFileLabel)

	timeRangeBox := container.NewHBox(
		widget.NewLabel("Start:"),
		g.startTimeEntry,
		widget.NewLabel("End:"),
		g.endTimeEntry,
	)

	optionsBox := container.NewHBox(
		widget.NewLabel("Quality:"),
		g.qualitySelect,
		widget.NewLabel("Format:"),
		g.formatEntry,
	)

	recentFilesBox := container.NewBorder(nil, nil, widget.NewLabel("Recent Files:"), nil, g.recentFilesSelect)

	infoBox := container.NewVBox(
		g.durationLabel,
		g.timeRangeLabel,
	)

	form := container.NewVBox(
		recentFilesBox,
		inputFileBox,
		outputFileBox,
		widget.NewSeparator(),
		timeRangeBox,
		optionsBox,
		widget.NewSeparator(),
		infoBox,
	)

	// Create button container
	buttonContainer := container.NewHBox(
		layout.NewSpacer(),
		g.previewButton,
		g.cutButton,
		layout.NewSpacer(),
	)

	// Create status bar container
	statusContainer := container.NewBorder(
		nil, nil, g.statusLabel, nil,
		g.progressBar,
	)

	// Create scrollable output area
	outputScroll := container.NewScroll(g.outputTextArea)
	outputScroll.SetMinSize(fyne.NewSize(600, 200))

	// Create main container
	mainContainer := container.NewBorder(
		header,
		container.NewVBox(
			buttonContainer,
			statusContainer,
		),
		nil, nil,
		container.NewVSplit(
			form,
			outputScroll,
		),
	)

	g.window.SetContent(mainContainer)
}

// selectInputFile opens a file dialog to select an input media file
func (g *MediaCutterGUI) selectInputFile() {
	dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, g.window)
			return
		}
		if uri == nil {
			return
		}

		filePath := uri.URI().Path()
		g.loadMediaFile(filePath)
	}, g.window)
}

// loadMediaFile loads a media file and updates the UI
func (g *MediaCutterGUI) loadMediaFile(filePath string) {
	g.appendOutput(fmt.Sprintf("Loading media file: %s\n", filePath))
	g.statusLabel.SetText("Loading media file...")

	err := g.mediaCutter.LoadMediaFile(filePath)
	if err != nil {
		g.appendOutput(fmt.Sprintf("Error loading media file: %v\n", err))
		g.statusLabel.SetText("Error loading media file")
		dialog.ShowError(err, g.window)
		return
	}

	g.inputFileLabel.SetText(filePath)
	g.durationLabel.SetText(fmt.Sprintf("Duration: %s", formatTime(g.mediaCutter.duration)))
	g.timeRangeLabel.SetText(fmt.Sprintf("Time range: %s - %s",
		formatTime(g.mediaCutter.startSeconds), formatTime(g.mediaCutter.endSeconds)))

	g.startTimeEntry.SetText(formatTime(g.mediaCutter.startSeconds))
	g.endTimeEntry.SetText(formatTime(g.mediaCutter.endSeconds))

	// Suggest output filename based on input
	dir, file := filepath.Split(filePath)
	ext := filepath.Ext(file)
	baseName := strings.TrimSuffix(file, ext)
	suggestedOutput := filepath.Join(dir, baseName+"_cut"+ext)
	g.outputFileLabel.SetText(suggestedOutput)
	g.mediaCutter.SetOutputFile(suggestedOutput)

	g.appendOutput(fmt.Sprintf("Media loaded successfully. Duration: %s\n",
		formatTime(g.mediaCutter.duration)))
	g.statusLabel.SetText("Ready")

	// Enable buttons
	g.previewButton.Enable()
	g.cutButton.Enable()

	// Update recent files dropdown
	g.updateRecentFilesDropdown()
}

// updateRecentFilesDropdown updates the recent files dropdown
func (g *MediaCutterGUI) updateRecentFilesDropdown() {
	recentFiles := []string{"Select a recent file..."}
	if len(g.mediaCutter.recentFiles) > 0 {
		recentFiles = append(recentFiles, g.mediaCutter.recentFiles...)
	}
	g.recentFilesSelect.Options = recentFiles
	g.recentFilesSelect.SetSelected("Select a recent file...")
}

// selectOutputFile opens a file dialog to select an output file
func (g *MediaCutterGUI) selectOutputFile() {
	dialog.ShowFileSave(func(uri fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, g.window)
			return
		}
		if uri == nil {
			return
		}

		filePath := uri.URI().Path()
		g.outputFileLabel.SetText(filePath)
		g.mediaCutter.SetOutputFile(filePath)
		g.appendOutput(fmt.Sprintf("Output file set to: %s\n", filePath))
	}, g.window)
}

// previewMedia previews the selected portion of the media
func (g *MediaCutterGUI) previewMedia() {
	if g.isProcessing {
		return
	}

	// Parse time range
	startTime := g.startTimeEntry.Text
	endTime := g.endTimeEntry.Text

	startSeconds := parseTime(startTime)
	endSeconds := parseTime(endTime)

	if startSeconds < 0 || endSeconds < 0 {
		dialog.ShowError(fmt.Errorf("invalid time format"), g.window)
		return
	}

	err := g.mediaCutter.SetTimeRange(startSeconds, endSeconds)
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	g.timeRangeLabel.SetText(fmt.Sprintf("Time range: %s - %s",
		formatTime(g.mediaCutter.startSeconds), formatTime(g.mediaCutter.endSeconds)))

	g.appendOutput(fmt.Sprintf("Previewing time range: %s - %s\n",
		formatTime(g.mediaCutter.startSeconds), formatTime(g.mediaCutter.endSeconds)))
	g.statusLabel.SetText("Previewing...")

	// Preview in a goroutine to not block the UI
	go func() {
		g.isProcessing = true
		g.previewButton.Disable()
		g.cutButton.Disable()

		err := g.mediaCutter.PreviewMedia()

		g.isProcessing = false
		g.previewButton.Enable()
		g.cutButton.Enable()

		if err != nil {
			g.appendOutput(fmt.Sprintf("Error previewing media: %v\n", err))
			g.statusLabel.SetText("Error previewing media")
			dialog.ShowError(err, g.window)
			return
		}

		g.statusLabel.SetText("Ready")
	}()
}

// cutMedia cuts the media file
func (g *MediaCutterGUI) cutMedia() {
	if g.isProcessing {
		return
	}

	// Parse time range
	startTime := g.startTimeEntry.Text
	endTime := g.endTimeEntry.Text

	startSeconds := parseTime(startTime)
	endSeconds := parseTime(endTime)

	if startSeconds < 0 || endSeconds < 0 {
		dialog.ShowError(fmt.Errorf("invalid time format"), g.window)
		return
	}

	err := g.mediaCutter.SetTimeRange(startSeconds, endSeconds)
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	// Get format and quality
	format := g.formatEntry.Text
	quality := g.qualitySelect.Selected

	g.appendOutput(fmt.Sprintf("Cutting media from %s to %s with quality: %s\n",
		formatTime(g.mediaCutter.startSeconds), formatTime(g.mediaCutter.endSeconds), quality))
	g.statusLabel.SetText("Cutting media...")
	g.progressBar.SetValue(0.1) // Show some progress

	// Cut in a goroutine to not block the UI
	go func() {
		g.isProcessing = true
		g.previewButton.Disable()
		g.cutButton.Disable()

		// Create a progress updater
		progressChan := make(chan float64)
		go func() {
			for progress := range progressChan {
				g.progressBar.SetValue(progress)
			}
		}()

		err := g.mediaCutter.CutMedia(format, quality)
		close(progressChan)

		g.isProcessing = false
		g.previewButton.Enable()
		g.cutButton.Enable()

		if err != nil {
			g.appendOutput(fmt.Sprintf("Error cutting media: %v\n", err))
			g.statusLabel.SetText("Error cutting media")
			g.progressBar.SetValue(0)
			dialog.ShowError(err, g.window)
			return
		}

		g.progressBar.SetValue(1.0)
		g.appendOutput("Media cutting completed successfully!\n")
		g.statusLabel.SetText("Cutting completed")

		// Show success dialog
		dialog.ShowInformation("Success", "Media cutting completed successfully!", g.window)

		// Update recent files dropdown
		g.updateRecentFilesDropdown()
	}()
}

// appendOutput appends text to the output area
func (g *MediaCutterGUI) appendOutput(text string) {
	current := g.outputTextArea.Text()
	g.outputTextArea.SetText(current + text)
}

// NewMediaCutterTheme creates a custom theme for the media cutter
type MediaCutterTheme struct {
	fyne.Theme
}

func NewMediaCutterTheme() fyne.Theme {
	return &MediaCutterTheme{Theme: theme.DefaultTheme()}
}

func (m *MediaCutterTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0, G: 120, B: 215, A: 255} // Bright blue
	case theme.ColorNameBackground:
		return color.NRGBA{R: 240, G: 240, B: 240, A: 255} // Light gray
	case theme.ColorNameButton:
		return color.NRGBA{R: 220, G: 220, B: 220, A: 255} // Slightly darker gray
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 180, G: 180, B: 180, A: 255} // Medium gray
	case theme.ColorNameForeground:
		return color.NRGBA{R: 40, G: 40, B: 40, A: 255} // Dark gray (almost black)
	case theme.ColorNameHover:
		return color.NRGBA{R: 0, G: 100, B: 195, A: 255} // Slightly darker blue
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 140, G: 140, B: 140, A: 255} // Medium gray
	case theme.ColorNamePressed:
		return color.NRGBA{R: 0, G: 80, B: 175, A: 255} // Even darker blue
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 200, G: 200, B: 200, A: 255} // Light gray
	default:
		return m.Theme.Color(name, variant)
	}
}

// Note: The main function is in media_cutter.go
// This file only contains the GUI implementation
