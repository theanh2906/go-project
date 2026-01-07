package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// CrawlItem represents a generic item crawled from a webpage
type CrawlItem struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Attributes  map[string]string `json:"attributes"`
	Content     []string          `json:"content"`
	Links       []string          `json:"links"`
	Timestamp   time.Time         `json:"timestamp"`
}

// FileConfig holds the configuration loaded from a JSON file
type FileConfig struct {
	BaseURL             string              `json:"BaseURL"`
	StartPage           int                 `json:"StartPage"`
	EndPage             int                 `json:"EndPage"`
	PagePattern         string              `json:"PagePattern"`
	Selector            string              `json:"Selector"`
	AttributeSelector   string              `json:"AttributeSelector"`
	AdvancedConfig      AdvancedConfig      `json:"AdvancedConfig"`
	TwoPhaseCrawlConfig TwoPhaseCrawlConfig `json:"TwoPhaseCrawlConfig"`
	OutputFile          string              `json:"OutputFile"`
	TwoPhaseCrawl       bool                `json:"TwoPhaseCrawl"`
	FollowLinks         bool                `json:"FollowLinks"`
}

// AdvancedConfig holds advanced configuration options for the crawler
type AdvancedConfig struct {
	MaxConcurrent int      `json:"MaxConcurrent"`
	MaxRetries    int      `json:"MaxRetries"`
	RetryDelay    int      `json:"RetryDelay"`
	RateLimit     int      `json:"RateLimit"`
	MaxDepth      int      `json:"MaxDepth"`
	CustomFilters []string `json:"CustomFilters"`
}

// AttributeConfig holds configuration for attributes to extract in two-phase crawl
type AttributeConfig struct {
	Selector          string `json:"Selector"`
	ElementAttribute  string `json:"ElementAttribute"`
	JsonAttribute     string `json:"JsonAttribute"`
	GetElementContent bool   `json:"GetElementContent"`
}

// TwoPhaseCrawlConfig holds configuration for two-phase crawling
type TwoPhaseCrawlConfig struct {
	Attributes []AttributeConfig `json:"Attributes"`
}

// CrawlConfig holds the configuration for the crawler
type CrawlConfig struct {
	BaseURL           string
	StartPage         int
	EndPage           int
	PagePattern       string
	Selector          string
	AttributeSelector string
	ContentSelector   string
	MaxConcurrent     int
	MaxRetries        int
	RetryDelay        time.Duration
	RateLimit         time.Duration
	OutputFile        string
	TwoPhaseCrawl     bool
	FollowLinks       bool
	MaxDepth          int
	CustomFilters     []string
}

// WebCrawlerGUI represents the GUI for the web crawler
type WebCrawlerGUI struct {
	app    fyne.App
	window fyne.Window

	// Config file
	configFile       string
	fileConfig       FileConfig
	configLoaded     bool
	configUploadBtn  *widget.Button
	configStatusLabel *widget.Label
	configDisplayArea *widget.Entry

	// Input fields (for backward compatibility and display only)
	urlEntry             *widget.Entry
	pagePatternEntry     *widget.Entry
	startPageEntry       *widget.Entry
	endPageEntry         *widget.Entry
	selectorEntry        *widget.Entry
	attrSelectorEntry    *widget.Entry
	contentSelectorEntry *widget.Entry
	maxConcurrentEntry   *widget.Entry
	maxRetriesEntry      *widget.Entry
	retryDelayEntry      *widget.Entry
	rateLimitEntry       *widget.Entry
	outputFileEntry      *widget.Entry
	maxDepthEntry        *widget.Entry
	customFiltersEntry   *widget.Entry

	// Checkboxes
	twoPhaseCrawlCheck *widget.Check
	followLinksCheck   *widget.Check

	// Output and status
	outputTextArea *widget.Entry
	progressBar    *widget.ProgressBar
	statusLabel    *widget.Label

	// Control buttons
	startButton *widget.Button
	stopButton  *widget.Button
	clearButton *widget.Button
	saveButton  *widget.Button

	// Advanced features
	previewButton  *widget.Button
	templateSelect *widget.Select

	// State
	crawlInProgress bool
	stopRequested   bool
	crawlResults    []CrawlItem
	crawlConfig     CrawlConfig

	// Templates
	templates map[string]CrawlConfig
}

// NewWebCrawlerGUI creates a new instance of the web crawler GUI
func NewWebCrawlerGUI() *WebCrawlerGUI {
	gui := &WebCrawlerGUI{
		app:          app.New(),
		crawlResults: make([]CrawlItem, 0),
		templates:    make(map[string]CrawlConfig),
		configLoaded: false,
	}

	gui.window = gui.app.NewWindow("Generic Web Crawler")

	// Try to load default config.json
	defaultConfigPath := "config.json"
	if _, err := os.Stat(defaultConfigPath); err == nil {
		if err := gui.loadConfigFromFile(defaultConfigPath); err == nil {
			gui.configFile = defaultConfigPath
			gui.configLoaded = true
		}
	}

	gui.createUI()
	gui.loadTemplates()

	return gui
}

// loadConfigFromFile loads configuration from a JSON file
func (g *WebCrawlerGUI) loadConfigFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	// Remove comments from JSON (comments start with // and continue to end of line)
	re := regexp.MustCompile(`//.*`)
	cleanJSON := re.ReplaceAllString(string(data), "")

	var config FileConfig
	if err := json.Unmarshal([]byte(cleanJSON), &config); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	// Validate the loaded configuration
	if err := g.validateFileConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	g.fileConfig = config
	g.updateUIFromConfig()
	return nil
}

// validateFileConfig validates the FileConfig structure
func (g *WebCrawlerGUI) validateFileConfig(config FileConfig) error {
	// Check required fields
	if config.BaseURL == "" {
		return fmt.Errorf("BaseURL is required")
	}

	if config.PagePattern == "" {
		return fmt.Errorf("PagePattern is required")
	}

	if !strings.Contains(config.BaseURL, config.PagePattern) {
		return fmt.Errorf("BaseURL must contain the PagePattern: %s", config.PagePattern)
	}

	if config.Selector == "" {
		return fmt.Errorf("Selector is required")
	}

	if config.AttributeSelector == "" {
		return fmt.Errorf("AttributeSelector is required")
	}

	if config.StartPage < 1 {
		return fmt.Errorf("StartPage must be at least 1")
	}

	if config.EndPage < config.StartPage {
		return fmt.Errorf("EndPage must be greater than or equal to StartPage")
	}

	if config.OutputFile == "" {
		return fmt.Errorf("OutputFile is required")
	}

	// Validate AdvancedConfig
	if config.AdvancedConfig.MaxConcurrent < 1 {
		return fmt.Errorf("AdvancedConfig.MaxConcurrent must be at least 1")
	}

	if config.AdvancedConfig.MaxRetries < 0 {
		return fmt.Errorf("AdvancedConfig.MaxRetries must be at least 0")
	}

	if config.AdvancedConfig.RetryDelay < 0 {
		return fmt.Errorf("AdvancedConfig.RetryDelay must be at least 0")
	}

	if config.AdvancedConfig.RateLimit < 0 {
		return fmt.Errorf("AdvancedConfig.RateLimit must be at least 0")
	}

	// Validate TwoPhaseCrawlConfig if TwoPhaseCrawl is enabled
	if config.TwoPhaseCrawl && len(config.TwoPhaseCrawlConfig.Attributes) == 0 {
		return fmt.Errorf("TwoPhaseCrawlConfig.Attributes is required when TwoPhaseCrawl is enabled")
	}

	// Validate each attribute in TwoPhaseCrawlConfig
	for i, attr := range config.TwoPhaseCrawlConfig.Attributes {
		if attr.Selector == "" {
			return fmt.Errorf("TwoPhaseCrawlConfig.Attributes[%d].Selector is required", i)
		}

		if attr.JsonAttribute == "" {
			return fmt.Errorf("TwoPhaseCrawlConfig.Attributes[%d].JsonAttribute is required", i)
		}

		// If GetElementContent is false, ElementAttribute is required
		if !attr.GetElementContent && attr.ElementAttribute == "" {
			return fmt.Errorf("TwoPhaseCrawlConfig.Attributes[%d].ElementAttribute is required when GetElementContent is false", i)
		}
	}

	return nil
}

// updateUIFromConfig updates the UI fields from the loaded configuration
func (g *WebCrawlerGUI) updateUIFromConfig() {
	if !g.configLoaded {
		return
	}

	// Update input fields
	g.urlEntry.SetText(g.fileConfig.BaseURL)
	g.pagePatternEntry.SetText(g.fileConfig.PagePattern)
	g.startPageEntry.SetText(fmt.Sprintf("%d", g.fileConfig.StartPage))
	g.endPageEntry.SetText(fmt.Sprintf("%d", g.fileConfig.EndPage))
	g.selectorEntry.SetText(g.fileConfig.Selector)
	g.attrSelectorEntry.SetText(g.fileConfig.AttributeSelector)

	// Update advanced settings
	g.maxConcurrentEntry.SetText(fmt.Sprintf("%d", g.fileConfig.AdvancedConfig.MaxConcurrent))
	g.maxRetriesEntry.SetText(fmt.Sprintf("%d", g.fileConfig.AdvancedConfig.MaxRetries))
	g.retryDelayEntry.SetText(fmt.Sprintf("%d", g.fileConfig.AdvancedConfig.RetryDelay))
	g.rateLimitEntry.SetText(fmt.Sprintf("%d", g.fileConfig.AdvancedConfig.RateLimit))
	g.maxDepthEntry.SetText(fmt.Sprintf("%d", g.fileConfig.AdvancedConfig.MaxDepth))

	// Update custom filters
	if len(g.fileConfig.AdvancedConfig.CustomFilters) > 0 {
		g.customFiltersEntry.SetText(strings.Join(g.fileConfig.AdvancedConfig.CustomFilters, "\n"))
	}

	// Update checkboxes
	g.twoPhaseCrawlCheck.SetChecked(g.fileConfig.TwoPhaseCrawl)
	g.followLinksCheck.SetChecked(g.fileConfig.FollowLinks)

	// Update output file
	g.outputFileEntry.SetText(g.fileConfig.OutputFile)

	// Update config display area if it exists
	if g.configDisplayArea != nil {
		configJSON, _ := json.MarshalIndent(g.fileConfig, "", "  ")
		g.configDisplayArea.SetText(string(configJSON))
	}

	// Update status label if it exists
	if g.configStatusLabel != nil {
		g.configStatusLabel.SetText("Configuration loaded from: " + g.configFile)
	}
}

// fileConfigToCrawlConfig converts a FileConfig to a CrawlConfig
func (g *WebCrawlerGUI) fileConfigToCrawlConfig() CrawlConfig {
	return CrawlConfig{
		BaseURL:           g.fileConfig.BaseURL,
		StartPage:         g.fileConfig.StartPage,
		EndPage:           g.fileConfig.EndPage,
		PagePattern:       g.fileConfig.PagePattern,
		Selector:          g.fileConfig.Selector,
		AttributeSelector: g.fileConfig.AttributeSelector,
		ContentSelector:   "", // Not in FileConfig
		MaxConcurrent:     g.fileConfig.AdvancedConfig.MaxConcurrent,
		MaxRetries:        g.fileConfig.AdvancedConfig.MaxRetries,
		RetryDelay:        time.Duration(g.fileConfig.AdvancedConfig.RetryDelay) * time.Second,
		RateLimit:         time.Duration(g.fileConfig.AdvancedConfig.RateLimit) * time.Millisecond,
		OutputFile:        g.fileConfig.OutputFile,
		TwoPhaseCrawl:     g.fileConfig.TwoPhaseCrawl,
		FollowLinks:       g.fileConfig.FollowLinks,
		MaxDepth:          g.fileConfig.AdvancedConfig.MaxDepth,
		CustomFilters:     g.fileConfig.AdvancedConfig.CustomFilters,
	}
}

// validateConfig validates the configuration and returns an error if invalid
func (g *WebCrawlerGUI) validateConfig(config CrawlConfig) error {
	// Check required fields
	if config.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	if config.PagePattern == "" {
		return fmt.Errorf("page pattern is required")
	}

	if !strings.Contains(config.BaseURL, config.PagePattern) {
		return fmt.Errorf("base URL must contain the page pattern: %s", config.PagePattern)
	}

	if config.Selector == "" {
		return fmt.Errorf("selector is required")
	}

	if config.StartPage < 1 {
		return fmt.Errorf("start page must be at least 1")
	}

	if config.EndPage < config.StartPage {
		return fmt.Errorf("end page must be greater than or equal to start page")
	}

	if config.MaxConcurrent < 1 {
		return fmt.Errorf("max concurrent requests must be at least 1")
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("max retries must be at least 0")
	}

	if config.RetryDelay < 0 {
		return fmt.Errorf("retry delay must be at least 0")
	}

	if config.RateLimit < 0 {
		return fmt.Errorf("rate limit must be at least 0")
	}

	if config.OutputFile == "" {
		return fmt.Errorf("output file is required")
	}

	return nil
}

// Run starts the GUI application
func (g *WebCrawlerGUI) Run() {
	g.window.ShowAndRun()
}

// createUI initializes all UI components
func (g *WebCrawlerGUI) createUI() {
	// Create config display area
	g.configDisplayArea = widget.NewMultiLineEntry()
	g.configDisplayArea.SetPlaceHolder("No configuration loaded. Please upload a config file.")
	g.configDisplayArea.Disable()

	// Create config status label
	g.configStatusLabel = widget.NewLabel("No configuration loaded")
	if g.configLoaded {
		g.configStatusLabel.SetText("Configuration loaded from: " + g.configFile)

		// Display the loaded config
		configJSON, _ := json.MarshalIndent(g.fileConfig, "", "  ")
		g.configDisplayArea.SetText(string(configJSON))
	} else {
		// Check if default config exists
		defaultConfigPath := "config.json"
		if _, err := os.Stat(defaultConfigPath); err != nil {
			g.configStatusLabel.SetText("Warning: Default config.json not found. Please upload a config file.")
		}
	}

	g.createInputFields()
	g.createOutputArea()
	g.createControlButtons()
	g.createStatusBar()
	g.layoutUI()
}

// createInputFields initializes all input fields
func (g *WebCrawlerGUI) createInputFields() {
	g.urlEntry = widget.NewEntry()
	g.urlEntry.SetPlaceHolder("https://example.com/page/{page}")

	g.pagePatternEntry = widget.NewEntry()
	g.pagePatternEntry.SetPlaceHolder("{page}")
	g.pagePatternEntry.SetText("{page}")

	g.startPageEntry = widget.NewEntry()
	g.startPageEntry.SetText("1")

	g.endPageEntry = widget.NewEntry()
	g.endPageEntry.SetText("10")

	g.selectorEntry = widget.NewEntry()
	g.selectorEntry.SetPlaceHolder("div.item, h2, a.product, etc.")

	g.attrSelectorEntry = widget.NewEntry()
	g.attrSelectorEntry.SetPlaceHolder("href, src, data-url, etc.")

	g.contentSelectorEntry = widget.NewEntry()
	g.contentSelectorEntry.SetPlaceHolder("div.content, p.description, etc.")

	g.maxConcurrentEntry = widget.NewEntry()
	g.maxConcurrentEntry.SetText("10")

	g.maxRetriesEntry = widget.NewEntry()
	g.maxRetriesEntry.SetText("3")

	g.retryDelayEntry = widget.NewEntry()
	g.retryDelayEntry.SetText("2")

	g.rateLimitEntry = widget.NewEntry()
	g.rateLimitEntry.SetText("200")

	g.outputFileEntry = widget.NewEntry()
	g.outputFileEntry.SetText("crawl_results.json")

	g.maxDepthEntry = widget.NewEntry()
	g.maxDepthEntry.SetText("1")

	g.customFiltersEntry = widget.NewMultiLineEntry()
	g.customFiltersEntry.SetPlaceHolder("Enter regex patterns to filter content, one per line")

	g.twoPhaseCrawlCheck = widget.NewCheck("Two-phase crawl (first get links, then crawl each link)", nil)
	g.twoPhaseCrawlCheck.SetChecked(true)

	g.followLinksCheck = widget.NewCheck("Follow links found in pages", nil)
	g.followLinksCheck.SetChecked(false)

	// Template selector
	g.templateSelect = widget.NewSelect([]string{"Custom", "News Site", "E-commerce", "Blog", "Forum"}, func(selected string) {
		if selected != "Custom" {
			g.loadTemplate(selected)
		}
	})
	g.templateSelect.SetSelected("Custom")
}

// createOutputArea initializes the output text area
func (g *WebCrawlerGUI) createOutputArea() {
	g.outputTextArea = widget.NewMultiLineEntry()
	g.outputTextArea.SetPlaceHolder("Crawl results will appear here...")
	//g.outputTextArea.Disable()
}

// createControlButtons initializes all control buttons
func (g *WebCrawlerGUI) createControlButtons() {
	g.startButton = widget.NewButton("Start Crawling", func() {
		g.startCrawling()
	})

	g.stopButton = widget.NewButton("Stop", func() {
		g.stopCrawling()
	})
	g.stopButton.Disable()

	g.clearButton = widget.NewButton("Clear", func() {
		g.clearOutput()
	})

	g.saveButton = widget.NewButton("Save Config", func() {
		g.saveCurrentConfig()
	})

	g.previewButton = widget.NewButton("Preview URL", func() {
		g.previewURL()
	})

	// Config file upload button
	g.configUploadBtn = widget.NewButton("Upload Config File", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, g.window)
				return
			}
			if reader == nil {
				return
			}

			defer reader.Close()

			// Create a temporary file to save the uploaded config
			tempFile, err := os.CreateTemp("", "config-*.json")
			if err != nil {
				dialog.ShowError(fmt.Errorf("error creating temp file: %v", err), g.window)
				return
			}
			defer tempFile.Close()

			// Copy the uploaded file to the temp file
			data, err := io.ReadAll(reader)
			if err != nil {
				dialog.ShowError(fmt.Errorf("error reading uploaded file: %v", err), g.window)
				return
			}

			if _, err := tempFile.Write(data); err != nil {
				dialog.ShowError(fmt.Errorf("error writing to temp file: %v", err), g.window)
				return
			}

			// Load the config from the temp file
			if err := g.loadConfigFromFile(tempFile.Name()); err != nil {
				dialog.ShowError(fmt.Errorf("error loading config: %v", err), g.window)
				return
			}

			g.configFile = reader.URI().Name()
			g.configLoaded = true

			dialog.ShowInformation("Config Loaded", "Configuration loaded successfully from "+g.configFile, g.window)
		}, g.window)

		fd.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
		fd.Show()
	})
}

// createStatusBar initializes the status bar
func (g *WebCrawlerGUI) createStatusBar() {
	g.progressBar = widget.NewProgressBar()
	g.statusLabel = widget.NewLabel("Ready")
}

// layoutUI arranges all UI components
func (g *WebCrawlerGUI) layoutUI() {
	// Create config section
	configSection := container.NewVBox(
		widget.NewLabel("Configuration"),
		g.configStatusLabel,
		g.configUploadBtn,
		container.NewScroll(g.configDisplayArea),
	)

	// Create tabs for basic and advanced settings
	basicSettings := container.NewVBox(
		widget.NewLabel("URL Pattern (use {page} for pagination):"),
		g.urlEntry,

		container.NewGridWithColumns(2,
			container.NewVBox(
				widget.NewLabel("Start Page:"),
				g.startPageEntry,
			),
			container.NewVBox(
				widget.NewLabel("End Page:"),
				g.endPageEntry,
			),
		),

		widget.NewLabel("Element Selector (CSS-like):"),
		g.selectorEntry,

		container.NewGridWithColumns(2,
			container.NewVBox(
				widget.NewLabel("Attribute to Extract:"),
				g.attrSelectorEntry,
			),
			container.NewVBox(
				widget.NewLabel("Content Selector:"),
				g.contentSelectorEntry,
			),
		),

		widget.NewLabel("Output File:"),
		g.outputFileEntry,

		container.NewHBox(
			g.twoPhaseCrawlCheck,
			g.followLinksCheck,
		),
	)

	advancedSettings := container.NewVBox(
		container.NewGridWithColumns(2,
			container.NewVBox(
				widget.NewLabel("Max Concurrent Requests:"),
				g.maxConcurrentEntry,
			),
			container.NewVBox(
				widget.NewLabel("Max Retries:"),
				g.maxRetriesEntry,
			),
		),

		container.NewGridWithColumns(2,
			container.NewVBox(
				widget.NewLabel("Retry Delay (seconds):"),
				g.retryDelayEntry,
			),
			container.NewVBox(
				widget.NewLabel("Rate Limit (ms):"),
				g.rateLimitEntry,
			),
		),

		container.NewGridWithColumns(2,
			container.NewVBox(
				widget.NewLabel("Max Depth (for following links):"),
				g.maxDepthEntry,
			),
			container.NewVBox(
				widget.NewLabel("Template:"),
				g.templateSelect,
			),
		),

		widget.NewLabel("Custom Filters (regex patterns):"),
		g.customFiltersEntry,
	)

	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Configuration", configSection),
		container.NewTabItem("Basic Settings", basicSettings),
		container.NewTabItem("Advanced Settings", advancedSettings),
	)

	// Control buttons
	controls := container.NewHBox(
		g.startButton,
		g.stopButton,
		g.clearButton,
		g.saveButton,
		g.previewButton,
	)

	// Status bar
	statusBar := container.NewBorder(nil, nil, nil, nil,
		container.NewVBox(
			g.progressBar,
			g.statusLabel,
		),
	)

	// Main layout
	content := container.NewBorder(
		tabs,
		container.NewVBox(
			controls,
			statusBar,
		),
		nil, nil,
		container.NewScroll(g.outputTextArea),
	)

	g.window.SetContent(content)
	g.window.Resize(fyne.NewSize(900, 700))
}

// startCrawling begins the crawling process
func (g *WebCrawlerGUI) startCrawling() {
	if g.crawlInProgress {
		return
	}

	g.crawlInProgress = true
	g.stopRequested = false
	g.crawlResults = make([]CrawlItem, 0)

	g.startButton.Disable()
	g.stopButton.Enable()
	g.clearOutput()

	var config CrawlConfig

	// Use loaded configuration if available, otherwise get from UI
	if g.configLoaded {
		g.appendOutput("Using configuration from file: " + g.configFile)
		config = g.fileConfigToCrawlConfig()
	} else {
		// Check if default config exists
		defaultConfigPath := "config.json"
		if _, err := os.Stat(defaultConfigPath); err == nil {
			// Try to load default config
			if err := g.loadConfigFromFile(defaultConfigPath); err == nil {
				g.configFile = defaultConfigPath
				g.configLoaded = true
				g.appendOutput("Using default configuration from: " + defaultConfigPath)
				config = g.fileConfigToCrawlConfig()

				// Update UI to show loaded config
				g.updateUIFromConfig()
			} else {
				g.appendOutput("Warning: Failed to load default config.json: " + err.Error())
				g.appendOutput("Using configuration from UI inputs")
				config = g.getConfigFromUI()
			}
		} else {
			g.appendOutput("Warning: No configuration file found. Using configuration from UI inputs")
			config = g.getConfigFromUI()
		}
	}

	// Validate the configuration
	if err := g.validateConfig(config); err != nil {
		g.appendOutput("Error: " + err.Error())
		g.updateStatus("Configuration error")
		g.crawlInProgress = false
		g.stopButton.Disable()
		g.startButton.Enable()
		dialog.ShowError(fmt.Errorf("Configuration error: %v", err), g.window)
		return
	}

	g.crawlConfig = config

	// Start crawling in a goroutine
	go func() {
		defer func() {
			g.crawlInProgress = false
			g.stopButton.Disable()
			g.startButton.Enable()
			g.updateStatus("Crawling completed")
		}()

		g.updateStatus("Crawling started...")

		if config.TwoPhaseCrawl {
			g.twoPhaseWebCrawl(config)
		} else {
			g.singlePhaseWebCrawl(config)
		}

		// Save results
		g.saveResults(config.OutputFile)
	}()
}

// stopCrawling stops the ongoing crawling process
func (g *WebCrawlerGUI) stopCrawling() {
	if g.crawlInProgress {
		g.stopRequested = true
		g.updateStatus("Stopping crawl...")
	}
}

// clearOutput clears the output text area
func (g *WebCrawlerGUI) clearOutput() {
	g.outputTextArea.SetText("")
	g.progressBar.SetValue(0)
	g.statusLabel.SetText("Ready")
}

// appendOutput adds text to the output area
func (g *WebCrawlerGUI) appendOutput(text string) {
	current := g.outputTextArea.Text
	g.outputTextArea.SetText(current + text + "\n")
}

// updateStatus updates the status label
func (g *WebCrawlerGUI) updateStatus(status string) {
	g.statusLabel.SetText(status)
}

// updateProgress updates the progress bar
func (g *WebCrawlerGUI) updateProgress(value float64) {
	g.progressBar.SetValue(value)
}

// getConfigFromUI extracts the crawler configuration from UI inputs
func (g *WebCrawlerGUI) getConfigFromUI() CrawlConfig {
	startPage := 1
	fmt.Sscanf(g.startPageEntry.Text, "%d", &startPage)

	endPage := 10
	fmt.Sscanf(g.endPageEntry.Text, "%d", &endPage)

	maxConcurrent := 10
	fmt.Sscanf(g.maxConcurrentEntry.Text, "%d", &maxConcurrent)

	maxRetries := 3
	fmt.Sscanf(g.maxRetriesEntry.Text, "%d", &maxRetries)

	retryDelay := 2
	fmt.Sscanf(g.retryDelayEntry.Text, "%d", &retryDelay)

	rateLimit := 200
	fmt.Sscanf(g.rateLimitEntry.Text, "%d", &rateLimit)

	maxDepth := 1
	fmt.Sscanf(g.maxDepthEntry.Text, "%d", &maxDepth)

	// Parse custom filters
	customFilters := []string{}
	if g.customFiltersEntry.Text != "" {
		customFilters = strings.Split(g.customFiltersEntry.Text, "\n")
	}

	return CrawlConfig{
		BaseURL:           g.urlEntry.Text,
		StartPage:         startPage,
		EndPage:           endPage,
		PagePattern:       g.pagePatternEntry.Text,
		Selector:          g.selectorEntry.Text,
		AttributeSelector: g.attrSelectorEntry.Text,
		ContentSelector:   g.contentSelectorEntry.Text,
		MaxConcurrent:     maxConcurrent,
		MaxRetries:        maxRetries,
		RetryDelay:        time.Duration(retryDelay) * time.Second,
		RateLimit:         time.Duration(rateLimit) * time.Millisecond,
		OutputFile:        g.outputFileEntry.Text,
		TwoPhaseCrawl:     g.twoPhaseCrawlCheck.Checked,
		FollowLinks:       g.followLinksCheck.Checked,
		MaxDepth:          maxDepth,
		CustomFilters:     customFilters,
	}
}

// singlePhaseWebCrawl performs a one-time crawl without following links
func (g *WebCrawlerGUI) singlePhaseWebCrawl(config CrawlConfig) {
	// Control concurrency with a semaphore
	semaphore := make(chan struct{}, config.MaxConcurrent)

	// Create a custom HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Rate limiter to avoid overloading the server
	rl := time.Tick(config.RateLimit)

	// Track results for reporting
	var (
		mutex      sync.Mutex
		successful int
		failed     int
	)

	// Wait group to track completion
	var wg sync.WaitGroup

	// Function to fetch a single page with retries
	fetchPage := func(pageNum int) {
		defer func() {
			wg.Done()
			<-semaphore // Release the semaphore slot
		}()

		if g.stopRequested {
			return
		}

		pageUrl := strings.Replace(config.BaseURL, config.PagePattern, fmt.Sprintf("%d", pageNum), 1)

		// Implement retry logic
		var doc *html.Node
		success := false

		for attempt := 0; attempt < config.MaxRetries; attempt++ {
			if g.stopRequested {
				return
			}

			if attempt > 0 {
				g.appendOutput(fmt.Sprintf("Retrying page %d (attempt %d/%d)...", pageNum, attempt+1, config.MaxRetries))
				time.Sleep(config.RetryDelay * time.Duration(attempt)) // Exponential backoff
			}

			<-rl // Rate limiting
			g.appendOutput(fmt.Sprintf("Fetching page %d: %s", pageNum, pageUrl))

			resp, err := client.Get(pageUrl)
			if err != nil {
				g.appendOutput(fmt.Sprintf("Error fetching page %d (attempt %d/%d): %v",
					pageNum, attempt+1, config.MaxRetries, err))
				continue // Try again
			}

			// Use a safe way to close the body
			func() {
				defer resp.Body.Close()

				// Check for non-successful status code
				if resp.StatusCode != http.StatusOK {
					g.appendOutput(fmt.Sprintf("Error on page %d (attempt %d/%d): status code %d",
						pageNum, attempt+1, config.MaxRetries, resp.StatusCode))
					return
				}

				// Try to parse the HTML
				var parseErr error
				doc, parseErr = html.Parse(resp.Body)
				if parseErr != nil {
					g.appendOutput(fmt.Sprintf("Error parsing page %d (attempt %d/%d): %v",
						pageNum, attempt+1, config.MaxRetries, parseErr))
					return
				}

				// If we reach here, we succeeded
				success = true
			}()

			if success {
				break // Exit retry loop on success
			}
		}

		// If we still couldn't fetch the page after all retries
		if !success {
			g.appendOutput(fmt.Sprintf("Failed to fetch page %d after %d attempts", pageNum, config.MaxRetries))
			mutex.Lock()
			failed++
			mutex.Unlock()
			return
		}

		// Process the HTML document
		var f func(*html.Node)
		elementCount := 0
		f = func(n *html.Node) {
			if g.stopRequested {
				return
			}

			if n.Type == html.ElementNode && n.Data == config.Selector {
				elementCount++

				item := CrawlItem{
					URL:        pageUrl,
					Timestamp:  time.Now(),
					Attributes: make(map[string]string),
					Content:    make([]string, 0),
					Links:      make([]string, 0),
				}

				// Extract title if available
				if n.FirstChild != nil {
					item.Title = strings.TrimSpace(extractTextContent(n))
				}

				// Extract attributes
				for _, attr := range n.Attr {
					item.Attributes[attr.Key] = attr.Val

					// If this is the attribute we're looking for
					if attr.Key == config.AttributeSelector {
						// If it's a link, add it to links
						if attr.Key == "href" || attr.Key == "src" {
							item.Links = append(item.Links, attr.Val)
						}
					}
				}

				// Extract content from specified selector
				if config.ContentSelector != "" {
					extractContent(n, config.ContentSelector, &item)
				}

				// Apply custom filters if any
				if len(config.CustomFilters) > 0 {
					for _, filter := range config.CustomFilters {
						re, err := regexp.Compile(filter)
						if err == nil {
							for i, content := range item.Content {
								if re.MatchString(content) {
									item.Content[i] = re.ReplaceAllString(content, "")
								}
							}
						}
					}
				}

				mutex.Lock()
				g.crawlResults = append(g.crawlResults, item)
				mutex.Unlock()

				g.appendOutput(fmt.Sprintf("Found item: %s", item.Title))
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}

		f(doc)

		g.appendOutput(fmt.Sprintf("Successfully processed page %d, found %d %s elements",
			pageNum, elementCount, config.Selector))

		mutex.Lock()
		successful++
		mutex.Unlock()

		// Update progress
		progress := float64(pageNum-config.StartPage+1) / float64(config.EndPage-config.StartPage+1)
		g.updateProgress(progress)
	}

	// Start crawling pages
	startTime := time.Now()

	for page := config.StartPage; page <= config.EndPage; page++ {
		if g.stopRequested {
			break
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire a semaphore slot
		go fetchPage(page)
	}

	// Wait for all pages to be processed
	wg.Wait()

	// Report results
	duration := time.Since(startTime)
	totalPages := config.EndPage - config.StartPage + 1
	g.appendOutput("\nCrawling summary:")
	g.appendOutput(fmt.Sprintf("Total pages attempted: %d", totalPages))
	g.appendOutput(fmt.Sprintf("Successfully crawled: %d (%.1f%%)", successful, float64(successful)/float64(totalPages)*100))
	g.appendOutput(fmt.Sprintf("Failed pages: %d (%.1f%%)", failed, float64(failed)/float64(totalPages)*100))
	g.appendOutput(fmt.Sprintf("Total time: %s", duration))
	g.appendOutput(fmt.Sprintf("Total items found: %d", len(g.crawlResults)))
}

// twoPhaseWebCrawl performs a two-phase crawl: first get all links, then crawl each link
func (g *WebCrawlerGUI) twoPhaseWebCrawl(config CrawlConfig) {
	// Phase 1: Get all links
	g.appendOutput("Phase 1: Collecting links...")

	links := make([]string, 0)
	var linksMutex sync.Mutex

	// Control concurrency with a semaphore
	semaphore := make(chan struct{}, config.MaxConcurrent)

	// Create a custom HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Rate limiter to avoid overloading the server
	rl := time.Tick(config.RateLimit)

	// Wait group to track completion
	var wg sync.WaitGroup

	// Function to fetch links from a single page
	fetchLinks := func(pageNum int) {
		defer func() {
			wg.Done()
			<-semaphore // Release the semaphore slot
		}()

		if g.stopRequested {
			return
		}

		pageUrl := strings.Replace(config.BaseURL, config.PagePattern, fmt.Sprintf("%d", pageNum), 1)

		// Implement retry logic
		var doc *html.Node
		success := false

		for attempt := 0; attempt < config.MaxRetries; attempt++ {
			if g.stopRequested {
				return
			}

			if attempt > 0 {
				time.Sleep(config.RetryDelay * time.Duration(attempt)) // Exponential backoff
			}

			<-rl // Rate limiting

			resp, err := client.Get(pageUrl)
			if err != nil {
				continue // Try again
			}

			// Use a safe way to close the body
			func() {
				defer resp.Body.Close()

				// Check for non-successful status code
				if resp.StatusCode != http.StatusOK {
					return
				}

				// Try to parse the HTML
				var parseErr error
				doc, parseErr = html.Parse(resp.Body)
				if parseErr != nil {
					return
				}

				// If we reach here, we succeeded
				success = true
			}()

			if success {
				break // Exit retry loop on success
			}
		}

		// If we still couldn't fetch the page after all retries
		if !success {
			return
		}

		// Process the HTML document to find links
		var f func(*html.Node)
		pageLinks := make([]string, 0)

		f = func(n *html.Node) {
			if g.stopRequested {
				return
			}

			if n.Type == html.ElementNode && n.Data == config.Selector {
				for _, attr := range n.Attr {
					if attr.Key == config.AttributeSelector {
						pageLinks = append(pageLinks, attr.Val)
						break
					}
				}
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}

		f(doc)

		// Add found links to the global list
		if len(pageLinks) > 0 {
			linksMutex.Lock()
			links = append(links, pageLinks...)
			linksMutex.Unlock()

			g.appendOutput(fmt.Sprintf("Found %d links on page %d", len(pageLinks), pageNum))
		}

		// Update progress for phase 1
		progress := float64(pageNum-config.StartPage+1) / float64(config.EndPage-config.StartPage+1) * 0.5
		g.updateProgress(progress)
	}

	// Start collecting links
	for page := config.StartPage; page <= config.EndPage; page++ {
		if g.stopRequested {
			break
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire a semaphore slot
		go fetchLinks(page)
	}

	// Wait for all link collection to complete
	wg.Wait()

	if g.stopRequested {
		g.appendOutput("Crawling stopped during phase 1")
		return
	}

	// Phase 2: Crawl each link
	g.appendOutput(fmt.Sprintf("\nPhase 2: Crawling %d individual links...", len(links)))

	// Reset wait group and progress
	wg = sync.WaitGroup{}

	// Function to crawl a single link
	crawlLink := func(url string, index int) {
		defer func() {
			wg.Done()
			<-semaphore // Release the semaphore slot
		}()

		if g.stopRequested {
			return
		}

		// Implement retry logic
		var doc *html.Node
		success := false

		for attempt := 0; attempt < config.MaxRetries; attempt++ {
			if g.stopRequested {
				return
			}

			if attempt > 0 {
				time.Sleep(config.RetryDelay * time.Duration(attempt)) // Exponential backoff
			}

			<-rl // Rate limiting

			resp, err := client.Get(url)
			if err != nil {
				continue // Try again
			}

			// Use a safe way to close the body
			func() {
				defer resp.Body.Close()

				// Check for non-successful status code
				if resp.StatusCode != http.StatusOK {
					return
				}

				// Try to parse the HTML
				var parseErr error
				doc, parseErr = html.Parse(resp.Body)
				if parseErr != nil {
					return
				}

				// If we reach here, we succeeded
				success = true
			}()

			if success {
				break // Exit retry loop on success
			}
		}

		// If we still couldn't fetch the page after all retries
		if !success {
			return
		}

		// Create a new crawl item
		item := CrawlItem{
			URL:        url,
			Timestamp:  time.Now(),
			Attributes: make(map[string]string),
			Content:    make([]string, 0),
			Links:      make([]string, 0),
		}

		// Extract title
		var title string
		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "title" {
				title = extractTextContent(n)
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(doc)
		item.Title = strings.TrimSpace(title)

		// Extract content based on content selector
		if config.ContentSelector != "" {
			var extractContentFunc func(*html.Node)
			extractContentFunc = func(n *html.Node) {
				if n.Type == html.ElementNode && hasClass(n, config.ContentSelector) {
					content := strings.TrimSpace(extractTextContent(n))
					if content != "" {
						item.Content = append(item.Content, content)
					}
				}

				for c := n.FirstChild; c != nil; c = c.NextSibling {
					extractContentFunc(c)
				}
			}
			extractContentFunc(doc)
		}

		// Apply custom filters if any
		if len(config.CustomFilters) > 0 {
			for _, filter := range config.CustomFilters {
				re, err := regexp.Compile(filter)
				if err == nil {
					for i, content := range item.Content {
						if re.MatchString(content) {
							item.Content[i] = re.ReplaceAllString(content, "")
						}
					}
				}
			}
		}

		// Add to results
		linksMutex.Lock()
		g.crawlResults = append(g.crawlResults, item)
		linksMutex.Unlock()

		g.appendOutput(fmt.Sprintf("Crawled: %s", item.Title))

		// Update progress for phase 2
		progress := 0.5 + (float64(index+1) / float64(len(links)) * 0.5)
		g.updateProgress(progress)
	}

	// Start crawling individual links
	for i, link := range links {
		if g.stopRequested {
			break
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire a semaphore slot
		go crawlLink(link, i)
	}

	// Wait for all crawling to complete
	wg.Wait()

	g.appendOutput(fmt.Sprintf("\nCrawling completed. Found %d items.", len(g.crawlResults)))
}

// saveResults saves the crawl results to a JSON file
func (g *WebCrawlerGUI) saveResults(filename string) {
	if len(g.crawlResults) == 0 {
		g.appendOutput("No results to save.")
		return
	}

	// Create output directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			g.appendOutput(fmt.Sprintf("Error creating directory: %v", err))
			return
		}
	}

	// Convert results to JSON
	data, err := json.MarshalIndent(g.crawlResults, "", "  ")
	if err != nil {
		g.appendOutput(fmt.Sprintf("Error marshalling data to JSON: %v", err))
		return
	}

	// Write to file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		g.appendOutput(fmt.Sprintf("Error writing JSON to file: %v", err))
		return
	}

	g.appendOutput(fmt.Sprintf("Data saved to %s", filename))
}

// previewURL fetches and displays a preview of the URL
func (g *WebCrawlerGUI) previewURL() {
	if g.urlEntry.Text == "" {
		dialog.ShowInformation("Error", "Please enter a URL to preview", g.window)
		return
	}

	// Replace page pattern with the start page
	startPage := 1
	fmt.Sscanf(g.startPageEntry.Text, "%d", &startPage)

	url := strings.Replace(g.urlEntry.Text, g.pagePatternEntry.Text, fmt.Sprintf("%d", startPage), 1)

	// Show a loading dialog
	loadingDialog := dialog.NewInformation("Loading", "Fetching URL preview...", g.window)
	loadingDialog.Show()

	// Fetch the URL in a goroutine
	go func() {
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Get(url)

		// Hide the loading dialog when we're done
		loadingDialog.Hide()

		if err != nil {
			dialog.ShowInformation("Error", fmt.Sprintf("Error fetching URL: %v", err), g.window)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			dialog.ShowInformation("Error", fmt.Sprintf("Error: Status code %d", resp.StatusCode), g.window)
			return
		}

		doc, err := html.Parse(resp.Body)
		if err != nil {
			dialog.ShowInformation("Error", fmt.Sprintf("Error parsing HTML: %v", err), g.window)
			return
		}

		// Extract title
		var title string
		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "title" {
				title = extractTextContent(n)
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(doc)

		// Extract elements matching the selector
		var elements []string
		if g.selectorEntry.Text != "" {
			var findElements func(*html.Node)
			findElements = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == g.selectorEntry.Text {
					elements = append(elements, extractTextContent(n))
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					findElements(c)
				}
			}
			findElements(doc)
		}

		// Create a new window to show the preview
		previewWindow := g.app.NewWindow("URL Preview")

		// Create content for the preview window
		titleLabel := widget.NewLabel("Title: " + title)
		urlLabel := widget.NewLabel("URL: " + url)

		var content *fyne.Container

		if len(elements) > 0 {
			elementsLabel := widget.NewLabel(fmt.Sprintf("Found %d matching elements:", len(elements)))

			// Create a list of elements
			elementsList := widget.NewList(
				func() int {
					return len(elements)
				},
				func() fyne.CanvasObject {
					return widget.NewLabel("Template")
				},
				func(id widget.ListItemID, obj fyne.CanvasObject) {
					label := obj.(*widget.Label)
					if id < len(elements) {
						text := elements[id]
						if len(text) > 100 {
							text = text[:97] + "..."
						}
						label.SetText(text)
					}
				},
			)

			// Add the list to a scroll container
			elementScroll := container.NewScroll(elementsList)
			elementScroll.SetMinSize(fyne.NewSize(500, 300))

			content = container.NewVBox(
				titleLabel,
				urlLabel,
				elementsLabel,
				elementScroll,
			)
		} else {
			content = container.NewVBox(
				titleLabel,
				urlLabel,
				widget.NewLabel("No matching elements found."),
			)
		}

		// Add a close button
		closeButton := widget.NewButton("Close", func() {
			previewWindow.Close()
		})

		// Set the window content
		previewWindow.SetContent(
			container.NewBorder(
				nil,
				container.NewHBox(layout.NewSpacer(), closeButton, layout.NewSpacer()),
				nil,
				nil,
				content,
			),
		)

		// Set window size and show it
		previewWindow.Resize(fyne.NewSize(600, 400))
		previewWindow.Show()
	}()
}

// saveCurrentConfig saves the current configuration as a template
func (g *WebCrawlerGUI) saveCurrentConfig() {
	// Create a dialog to get the template name
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Template Name")

	dialog.ShowCustomConfirm("Save Template", "Save", "Cancel",
		container.NewVBox(
			widget.NewLabel("Enter a name for this template:"),
			nameEntry,
		),
		func(save bool) {
			if save && nameEntry.Text != "" {
				config := g.getConfigFromUI()
				g.templates[nameEntry.Text] = config

				// Update template selector
				templateNames := make([]string, 0, len(g.templates)+1)
				templateNames = append(templateNames, "Custom")
				for name := range g.templates {
					templateNames = append(templateNames, name)
				}
				g.templateSelect.Options = templateNames

				dialog.ShowInformation("Template Saved",
					fmt.Sprintf("Template '%s' has been saved.", nameEntry.Text),
					g.window)
			}
		},
		g.window,
	)
}

// loadTemplate loads a saved template
func (g *WebCrawlerGUI) loadTemplate(name string) {
	if config, ok := g.templates[name]; ok {
		// Update UI with template values
		g.urlEntry.SetText(config.BaseURL)
		g.pagePatternEntry.SetText(config.PagePattern)
		g.startPageEntry.SetText(fmt.Sprintf("%d", config.StartPage))
		g.endPageEntry.SetText(fmt.Sprintf("%d", config.EndPage))
		g.selectorEntry.SetText(config.Selector)
		g.attrSelectorEntry.SetText(config.AttributeSelector)
		g.contentSelectorEntry.SetText(config.ContentSelector)
		g.maxConcurrentEntry.SetText(fmt.Sprintf("%d", config.MaxConcurrent))
		g.maxRetriesEntry.SetText(fmt.Sprintf("%d", config.MaxRetries))
		g.retryDelayEntry.SetText(fmt.Sprintf("%d", int(config.RetryDelay.Seconds())))
		g.rateLimitEntry.SetText(fmt.Sprintf("%d", int(config.RateLimit.Milliseconds())))
		g.outputFileEntry.SetText(config.OutputFile)
		g.maxDepthEntry.SetText(fmt.Sprintf("%d", config.MaxDepth))
		g.twoPhaseCrawlCheck.SetChecked(config.TwoPhaseCrawl)
		g.followLinksCheck.SetChecked(config.FollowLinks)

		// Set custom filters
		if len(config.CustomFilters) > 0 {
			g.customFiltersEntry.SetText(strings.Join(config.CustomFilters, "\n"))
		} else {
			g.customFiltersEntry.SetText("")
		}
	}
}

// loadTemplates loads predefined templates
func (g *WebCrawlerGUI) loadTemplates() {
	// News site template
	g.templates["News Site"] = CrawlConfig{
		BaseURL:           "https://example.com/news/page/{page}",
		StartPage:         1,
		EndPage:           10,
		PagePattern:       "{page}",
		Selector:          "article",
		AttributeSelector: "href",
		ContentSelector:   "div.content",
		MaxConcurrent:     10,
		MaxRetries:        3,
		RetryDelay:        2 * time.Second,
		RateLimit:         200 * time.Millisecond,
		OutputFile:        "news_articles.json",
		TwoPhaseCrawl:     true,
		FollowLinks:       false,
		MaxDepth:          1,
		CustomFilters:     []string{"\\d+\\s+comments", "advertisement"},
	}

	// E-commerce template
	g.templates["E-commerce"] = CrawlConfig{
		BaseURL:           "https://example.com/products/page/{page}",
		StartPage:         1,
		EndPage:           10,
		PagePattern:       "{page}",
		Selector:          "div.product",
		AttributeSelector: "href",
		ContentSelector:   "div.description",
		MaxConcurrent:     10,
		MaxRetries:        3,
		RetryDelay:        2 * time.Second,
		RateLimit:         300 * time.Millisecond,
		OutputFile:        "products.json",
		TwoPhaseCrawl:     true,
		FollowLinks:       false,
		MaxDepth:          1,
		CustomFilters:     []string{"Out of stock", "\\$\\d+\\.\\d+"},
	}

	// Blog template
	g.templates["Blog"] = CrawlConfig{
		BaseURL:           "https://example.com/blog/page/{page}",
		StartPage:         1,
		EndPage:           5,
		PagePattern:       "{page}",
		Selector:          "article",
		AttributeSelector: "href",
		ContentSelector:   "div.post-content",
		MaxConcurrent:     5,
		MaxRetries:        3,
		RetryDelay:        2 * time.Second,
		RateLimit:         500 * time.Millisecond,
		OutputFile:        "blog_posts.json",
		TwoPhaseCrawl:     true,
		FollowLinks:       false,
		MaxDepth:          1,
		CustomFilters:     []string{"Posted by", "\\d+ comments"},
	}

	// Forum template
	g.templates["Forum"] = CrawlConfig{
		BaseURL:           "https://example.com/forum/page/{page}",
		StartPage:         1,
		EndPage:           10,
		PagePattern:       "{page}",
		Selector:          "div.thread",
		AttributeSelector: "href",
		ContentSelector:   "div.post-content",
		MaxConcurrent:     8,
		MaxRetries:        3,
		RetryDelay:        2 * time.Second,
		RateLimit:         400 * time.Millisecond,
		OutputFile:        "forum_threads.json",
		TwoPhaseCrawl:     true,
		FollowLinks:       true,
		MaxDepth:          2,
		CustomFilters:     []string{"Posted by", "\\d+ replies", "\\d+ views"},
	}
}

// extractContent extracts content from nodes matching the given selector
func extractContent(n *html.Node, selector string, item *CrawlItem) {
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && hasClass(node, selector) {
			content := strings.TrimSpace(extractTextContent(node))
			if content != "" {
				item.Content = append(item.Content, content)
			}
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(n)
}

// extractTextContent extracts all text content from a node and its children
func extractTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var result string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += extractTextContent(c)
	}
	return result
}

// hasClass checks if an HTML element has a specific class
func hasClass(n *html.Node, className string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			classes := strings.Fields(attr.Val)
			for _, class := range classes {
				if class == className {
					return true
				}
			}
		}
	}
	return false
}

func main() {
	crawler := NewWebCrawlerGUI()
	crawler.Run()
}
