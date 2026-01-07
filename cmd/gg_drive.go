package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Config struct {
	CredentialsPath string `json:"credentials_path"`
}

func main() {
	clearScreen()
	printBanner()

	ctx := context.Background()
	srv, err := ensureDriveService(ctx)
	if err != nil {
		color.Red("Failed to initialize Google Drive service: %v", err)
		return
	}

	for {
		clearScreen()
		choice, err := mainMenu()
		if err != nil {
			fmt.Println()
			return
		}

		switch choice {
		case "Search files/folders":
			clearScreen()
			handleSearchUI(ctx, srv)
		case "Show folder tree":
			clearScreen()
			handleShowTreeUI(ctx, srv)
		case "Upload file":
			clearScreen()
			handleUploadFileUI(ctx, srv)
		case "Upload folder":
			clearScreen()
			handleUploadFolderUI(ctx, srv)
		case "Create folder":
			clearScreen()
			handleMkdirUI(ctx, srv)
		case "Configure credentials":
			clearScreen()
			if err := configureCredentials(); err != nil {
				color.Red("Configuration failed: %v", err)
			} else {
				color.Green("✔ Credentials configured successfully!")
			}
			time.Sleep(2 * time.Second)
		case "Exit":
			color.Cyan("Goodbye 👋")
			return
		}
	}
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func printBanner() {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	magenta := color.New(color.FgMagenta, color.Bold).SprintFunc()

	fmt.Println()
	fmt.Println(cyan("  ╔═══════════════════════════════════════════════════════╗"))
	fmt.Println(cyan("  ║") + "                                                       " + cyan("║"))
	fmt.Println(cyan("  ║") + "     " + yellow("🚀  Google Drive CLI Manager  🚀") + "          " + cyan("║"))
	fmt.Println(cyan("  ║") + "                                                       " + cyan("║"))
	fmt.Println(cyan("  ║") + "     " + magenta("Upload • Search • Organize • Share") + "         " + cyan("║"))
	fmt.Println(cyan("  ║") + "                                                       " + cyan("║"))
	fmt.Println(cyan("  ╚═══════════════════════════════════════════════════════╝"))
	fmt.Println()
}

func mainMenu() (string, error) {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()

	fmt.Println(cyan("  ┌─────────────────────────────────────────────┐"))
	fmt.Println(cyan("  │") + "  " + yellow("📋  MAIN MENU") + "                           " + cyan("│"))
	fmt.Println(cyan("  └─────────────────────────────────────────────┘"))
	fmt.Println()

	prompt := promptui.Select{
		Label: "Select an action",
		Items: []string{
			"🔍  Search files/folders",
			"🌳  Show folder tree",
			"📤  Upload file",
			"📁  Upload folder",
			"➕  Create folder",
			"⚙️  Configure credentials",
			"🚪  Exit",
		},
		Size: 8,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . | cyan | bold }}",
			Active:   "▸ {{ . | yellow | bold }}",
			Inactive: "  {{ . | white }}",
			Selected: "✔ {{ . | green | bold }}",
		},
	}
	_, v, err := prompt.Run()
	// Remove emoji prefix from selection
	if v != "" && len(v) > 3 {
		parts := strings.SplitN(v, "  ", 2)
		if len(parts) == 2 {
			v = strings.TrimSpace(parts[1])
		}
	}
	return v, err
}

func ggdriveConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ggdrive"), nil
}

func loadConfig() (*Config, error) {
	dir, err := ggdriveConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	dir, err := ggdriveConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.json")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

func configureCredentials() error {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("\nGoogle Drive CLI setup — configure credentials.json path"))

	prompt := promptui.Prompt{
		Label: "Path to credentials.json",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("path is required")
			}
			if _, err := os.Stat(s); err != nil {
				return fmt.Errorf("file does not exist: %v", err)
			}
			return nil
		},
	}

	path, err := prompt.Run()
	if err != nil {
		return err
	}

	cfg := &Config{CredentialsPath: path}
	if err := saveConfig(cfg); err != nil {
		return err
	}

	color.Green("Configuration saved to ~/.ggdrive/config.json")
	return nil
}

func getCredentialsPath() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		// Try to configure
		if err := configureCredentials(); err != nil {
			return "", err
		}
		cfg, err = loadConfig()
		if err != nil {
			return "", err
		}
	}
	return cfg.CredentialsPath, nil
}

func getTokenPath() (string, error) {
	dir, err := ggdriveConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

func ensureDriveService(ctx context.Context) (*drive.Service, error) {
	credPath, err := getCredentialsPath()
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %v", err)
	}

	client, err := getClient(config)
	if err != nil {
		return nil, err
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Drive service: %v", err)
	}

	return srv, nil
}

func handleSearchUI(ctx context.Context, srv *drive.Service) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("🔍 Search Files/Folders"))
	fmt.Println()

	namePrompt := promptui.Prompt{
		Label: "Search term",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("search term is required")
			}
			return nil
		},
	}
	name, err := namePrompt.Run()
	if err != nil {
		return
	}

	parentPrompt := promptui.Prompt{
		Label:   "Parent folder ID (optional, press Enter to skip)",
		Default: "",
	}
	parentID, _ := parentPrompt.Run()

	color.Yellow("⏳ Searching...")
	results, err := searchFiles(ctx, srv, name, parentID)
	if err != nil {
		color.Red("Error searching: %v", err)
		waitForEnter()
		return
	}

	if len(results) == 0 {
		color.Yellow("No files found matching '%s'", name)
		waitForEnter()
		return
	}

	fmt.Println()
	color.Green("✔ Found %d result(s):", len(results))
	fmt.Println()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s  %-50s  %s\n", bold("Type"), bold("Name"), bold("ID"))
	fmt.Println(strings.Repeat("-", 100))

	for _, f := range results {
		fileType := "📄 File  "
		if f.MimeType == "application/vnd.google-apps.folder" {
			fileType = "📁 Folder"
		}
		fmt.Printf("%s  %-50s  %s\n", fileType, truncate(f.Name, 50), f.Id)
	}

	waitForEnter()
}

func handleShowTreeUI(ctx context.Context, srv *drive.Service) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("🌳 Show Folder Tree"))
	fmt.Println()

	parentPrompt := promptui.Prompt{
		Label:   "Folder ID (press Enter for root/My Drive)",
		Default: "",
	}
	parentID, _ := parentPrompt.Run()

	if parentID == "" {
		parentID = "root"
	}

	depthPrompt := promptui.Prompt{
		Label:   "Max depth (press Enter for 3 levels)",
		Default: "3",
	}
	depthStr, _ := depthPrompt.Run()
	maxDepth := 3
	if depthStr != "" {
		fmt.Sscanf(depthStr, "%d", &maxDepth)
	}

	color.Yellow("⏳ Loading folder tree...")
	fmt.Println()

	// Get root folder info
	rootName := "My Drive"
	if parentID != "root" {
		file, err := srv.Files.Get(parentID).Fields("name").Do()
		if err != nil {
			color.Red("Error getting folder info: %v", err)
			waitForEnter()
			return
		}
		rootName = file.Name
	}

	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	fmt.Println(green("📁 " + rootName))

	err := printTree(ctx, srv, parentID, "", maxDepth, 0)
	if err != nil {
		color.Red("\nError building tree: %v", err)
	}

	fmt.Println()
	waitForEnter()
}

func printTree(ctx context.Context, srv *drive.Service, folderID string, prefix string, maxDepth int, currentDepth int) error {
	if currentDepth >= maxDepth {
		return nil
	}

	// Get all items in this folder
	query := fmt.Sprintf("'%s' in parents and trashed = false", folderID)

	call := srv.Files.List().
		Q(query).
		Fields("files(id, name, mimeType)").
		OrderBy("folder,name").
		PageSize(100).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true)

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to list files: %v", err)
	}

	items := resp.Files

	for i, item := range items {
		isLast := i == len(items)-1

		// Determine the tree characters
		var branch string
		var newPrefix string
		if isLast {
			branch = "└── "
			newPrefix = prefix + "    "
		} else {
			branch = "├── "
			newPrefix = prefix + "│   "
		}

		// Determine icon and color
		isFolder := item.MimeType == "application/vnd.google-apps.folder"
		var itemStr string
		if isFolder {
			blue := color.New(color.FgBlue, color.Bold).SprintFunc()
			itemStr = blue("📁 " + item.Name)
		} else {
			itemStr = "📄 " + item.Name
		}

		fmt.Println(prefix + branch + itemStr)

		// Recursively print subdirectories
		if isFolder {
			err := printTree(ctx, srv, item.Id, newPrefix, maxDepth, currentDepth+1)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func handleUploadFileUI(ctx context.Context, srv *drive.Service) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("📤 Upload File"))
	fmt.Println()

	filePrompt := promptui.Prompt{
		Label: "File path",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("file path is required")
			}
			if _, err := os.Stat(s); err != nil {
				return fmt.Errorf("file does not exist: %v", err)
			}
			return nil
		},
	}
	filePath, err := filePrompt.Run()
	if err != nil {
		return
	}

	parentPrompt := promptui.Prompt{
		Label: "Parent folder ID",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("parent folder ID is required")
			}
			return nil
		},
	}
	parentID, err := parentPrompt.Run()
	if err != nil {
		return
	}

	fmt.Println()
	file, err := uploadFileWithProgress(ctx, srv, filePath, parentID)
	if err != nil {
		color.Red("✗ Upload failed: %v", err)
		waitForEnter()
		return
	}

	fmt.Println()
	color.Green("✔ File uploaded successfully!")
	fmt.Printf("  Name: %s\n", file.Name)
	fmt.Printf("  ID: %s\n", file.Id)
	waitForEnter()
}

func handleUploadFolderUI(ctx context.Context, srv *drive.Service) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("📤 Upload Folder"))
	fmt.Println()

	folderPrompt := promptui.Prompt{
		Label: "Folder path",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("folder path is required")
			}
			info, err := os.Stat(s)
			if err != nil {
				return fmt.Errorf("folder does not exist: %v", err)
			}
			if !info.IsDir() {
				return fmt.Errorf("path is not a folder")
			}
			return nil
		},
	}
	folderPath, err := folderPrompt.Run()
	if err != nil {
		return
	}

	parentPrompt := promptui.Prompt{
		Label: "Parent folder ID",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("parent folder ID is required")
			}
			return nil
		},
	}
	parentID, err := parentPrompt.Run()
	if err != nil {
		return
	}

	fmt.Println()
	color.Yellow("⏳ Uploading folder...")
	if err := uploadFolder(ctx, srv, folderPath, parentID); err != nil {
		color.Red("✗ Upload failed: %v", err)
		waitForEnter()
		return
	}

	fmt.Println()
	color.Green("✔ Folder uploaded successfully!")
	waitForEnter()
}

func handleMkdirUI(ctx context.Context, srv *drive.Service) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("📁 Create Folder"))
	fmt.Println()

	namePrompt := promptui.Prompt{
		Label: "Folder name",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("folder name is required")
			}
			return nil
		},
	}
	name, err := namePrompt.Run()
	if err != nil {
		return
	}

	parentPrompt := promptui.Prompt{
		Label: "Parent folder ID",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("parent folder ID is required")
			}
			return nil
		},
	}
	parentID, err := parentPrompt.Run()
	if err != nil {
		return
	}

	fmt.Println()
	color.Yellow("⏳ Creating folder...")
	folder, err := createFolder(ctx, srv, name, parentID)
	if err != nil {
		color.Red("✗ Failed to create folder: %v", err)
		waitForEnter()
		return
	}

	color.Green("✔ Folder created successfully!")
	fmt.Printf("  Name: %s\n", folder.Name)
	fmt.Printf("  ID: %s\n", folder.Id)
	waitForEnter()
}

func waitForEnter() {
	fmt.Println()
	color.New(color.Faint).Println("Press Enter to continue...")
	fmt.Scanln()
}

// searchFiles searches for files/folders by name in Google Drive
func searchFiles(ctx context.Context, srv *drive.Service, name string, parentID string) ([]*drive.File, error) {
	var query string
	if parentID != "" {
		query = fmt.Sprintf("name contains '%s' and '%s' in parents and trashed = false", name, parentID)
	} else {
		query = fmt.Sprintf("name contains '%s' and trashed = false", name)
	}

	var results []*drive.File
	pageToken := ""
	for {
		call := srv.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name, mimeType, createdTime, modifiedTime, size)").
			PageSize(100).
			SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to search files: %v", err)
		}

		results = append(results, resp.Files...)

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return results, nil
}

// uploadFile uploads a single file to Google Drive with progress bar
func uploadFileWithProgress(ctx context.Context, srv *drive.Service, filePath string, parentID string) (*drive.File, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	fileName := filepath.Base(filePath)
	file := &drive.File{
		Name:    fileName,
		Parents: []string{parentID},
	}

	// Create progress bar
	bar := progressbar.DefaultBytes(
		fileInfo.Size(),
		fmt.Sprintf("Uploading %s", fileName),
	)

	// Wrap the file reader with progress bar
	reader := progressbar.NewReader(f, bar)

	uploadedFile, err := srv.Files.Create(file).
		Media(&reader).
		Fields("id, name, mimeType, size").
		Context(ctx).
		Do()

	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %v", err)
	}

	return uploadedFile, nil
}

// uploadFile uploads a single file (without progress for folder uploads)
func uploadFile(ctx context.Context, srv *drive.Service, filePath string, parentID string) (*drive.File, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	fileName := filepath.Base(filePath)
	file := &drive.File{
		Name:    fileName,
		Parents: []string{parentID},
	}

	uploadedFile, err := srv.Files.Create(file).
		Media(f).
		Fields("id, name, mimeType, size").
		Context(ctx).
		Do()

	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %v", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("  %s %s (%.2f MB)\n", green("✔"), fileName, float64(fileInfo.Size())/1024/1024)
	return uploadedFile, nil
}

// uploadFolder recursively uploads a folder to Google Drive
func uploadFolder(ctx context.Context, srv *drive.Service, folderPath string, parentID string) error {
	folderName := filepath.Base(folderPath)

	// Create the folder in Google Drive
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("  %s Creating folder: %s\n", cyan("📁"), folderName)

	folder, err := createFolder(ctx, srv, folderName, parentID)
	if err != nil {
		return fmt.Errorf("failed to create folder %s: %v", folderName, err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("  %s Folder created (ID: %s)\n", green("✔"), folder.Id)

	// Read directory contents
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %v", folderPath, err)
	}

	// Upload each entry
	for _, entry := range entries {
		entryPath := filepath.Join(folderPath, entry.Name())

		if entry.IsDir() {
			// Recursively upload subdirectory
			if err := uploadFolder(ctx, srv, entryPath, folder.Id); err != nil {
				return err
			}
		} else {
			// Upload file
			_, err := uploadFile(ctx, srv, entryPath, folder.Id)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// createFolder creates a new folder in Google Drive
func createFolder(ctx context.Context, srv *drive.Service, name string, parentID string) (*drive.File, error) {
	folder := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentID},
	}

	createdFolder, err := srv.Files.Create(folder).
		Fields("id, name").
		SupportsAllDrives(true).
		Context(ctx).
		Do()

	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %v", err)
	}

	return createdFolder, nil
}

// detectMimeType detects the MIME type of a file based on extension
func detectMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
		".json": "application/json",
		".xml":  "application/xml",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func getClient(config *oauth2.Config) (*http.Client, error) {
	tokenPath, err := getTokenPath()
	if err != nil {
		return nil, err
	}

	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenPath, tok)
	}
	return config.Client(context.Background(), tok), nil
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println()
	fmt.Println(cyan("🔐 Google Drive Authorization Required"))
	fmt.Println()
	fmt.Println(yellow("Open this link in your browser:"))
	fmt.Println(authURL)
	fmt.Println()

	prompt := promptui.Prompt{
		Label: "Enter the authorization code",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("code is required")
			}
			return nil
		},
	}

	code, err := prompt.Run()
	if err != nil {
		color.Red("Authorization failed: %v", err)
		os.Exit(1)
	}

	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		color.Red("Unable to retrieve token: %v", err)
		os.Exit(1)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Token saved to: %s\n", green("✔"), path)
	f, err := os.Create(path)
	if err != nil {
		color.Red("Unable to cache token: %v", err)
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
