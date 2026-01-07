# 🚀 Google Drive CLI - Terminal UI

A beautiful, interactive Terminal User Interface (TUI) for managing Google Drive operations.

## ✨ Features

- 🔍 **Search** - Search for files and folders
- 📤 **Upload File** - Upload individual files with progress bar
- 📁 **Upload Folder** - Recursively upload entire folders
- 🆕 **Create Folder** - Create new folders in Google Drive
- ⚙️ **Configure** - Easy credential management
- 🎨 **Beautiful UI** - Colorful, interactive menu system
- 📊 **Progress Bars** - Visual feedback during file uploads

## 📋 Prerequisites

1. **Google Cloud Project** with Drive API enabled
2. **credentials.json** from Google Cloud Console
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
   - Enable Google Drive API
   - Create OAuth 2.0 credentials (Desktop app)
   - Download as `credentials.json`

## 🚀 Quick Start

### First Time Setup

1. **Build the application** (if not already built):
   ```powershell
   go build -o gg_drive.exe gg_drive.go
   ```

2. **Run the application**:
   ```powershell
   .\gg_drive.exe
   ```

3. **Configure credentials**:
   - On first run, you'll be prompted to configure
   - Select "Configure credentials" from menu
   - Enter the full path to your `credentials.json` file
   - Example: `C:\Users\YourName\Downloads\credentials.json`

4. **Authorize the app**:
   - When prompted, open the authorization URL in your browser
   - Log in to your Google account
   - Grant permissions
   - Copy the authorization code
   - Paste it back into the terminal

## 📁 Configuration Files

All configuration files are stored in your home directory under `.ggdrive`:

```
~/.ggdrive/
  ├── config.json    # Stores path to credentials.json
  └── token.json     # OAuth token (auto-generated)
```

**Windows**: `C:\Users\YourName\.ggdrive\`  
**Linux/Mac**: `~/.ggdrive/`

## 🎯 Usage

### Main Menu

When you run `gg_drive.exe`, you'll see an interactive menu:

```
Google Drive CLI — select action
  Search files/folders
  Upload file
  Upload folder
  Create folder
  Configure credentials
  Exit
```

### 1️⃣ Search Files/Folders

- Enter search term (e.g., "report.pdf", "Documents")
- Optionally enter parent folder ID to narrow search
- Results display with file type (📄 File or 📁 Folder), name, and ID

### 2️⃣ Upload File

- Enter the full path to your file
- Example: `C:\Users\YourName\Documents\report.pdf`
- Enter the parent folder ID in Google Drive
- Watch the progress bar as your file uploads!

### 3️⃣ Upload Folder

- Enter the full path to your folder
- Example: `C:\Users\YourName\Projects\MyProject`
- Enter the parent folder ID in Google Drive
- All files and subfolders will be uploaded recursively
- Progress shown for each file

### 4️⃣ Create Folder

- Enter the new folder name
- Enter the parent folder ID
- Folder created instantly with confirmation

### 5️⃣ Configure Credentials

- Update your credentials.json path
- Useful if you move the file or want to use different credentials

## 💡 Tips & Tricks

### Getting Folder IDs

1. Open Google Drive in your browser
2. Navigate to the folder
3. Look at the URL: `https://drive.google.com/drive/folders/FOLDER_ID_HERE`
4. Copy the ID after `/folders/`

### Root Folder

- To upload to "My Drive" root, use folder ID from any folder's parent
- Or create a test folder first and use its ID

### Bulk Operations

- Upload entire project folders in one go
- Maintains folder structure
- All files uploaded with progress tracking

### Re-authentication

If you get permission errors:
1. Delete `~/.ggdrive/token.json`
2. Run the app again
3. Re-authorize with your Google account

## 🎨 UI Features

- **🎨 Color-coded output**:
  - 🔵 Cyan - Headers and prompts
  - 🟢 Green - Success messages
  - 🔴 Red - Errors
  - 🟡 Yellow - Warnings and progress

- **📊 Progress bars** for file uploads
- **📁 Icons** for files and folders
- **✨ Clean screen** between operations
- **⌨️ Keyboard navigation** with arrow keys

## 🔧 Troubleshooting

### "Failed to initialize Google Drive service"
- Ensure `credentials.json` path is correct
- Run "Configure credentials" from menu
- Check file exists and is readable

### "Insufficient permissions" error
- Delete `~/.ggdrive/token.json`
- Re-run and re-authorize
- Make sure you're granting full Drive permissions

### "File not found" during upload
- Use absolute paths (full path from drive root)
- Check for typos
- Ensure file/folder exists

### Progress bar not showing
- This is normal for very small files
- Progress shows for files > 1MB

## 🏗️ Building from Source

```powershell
# Install dependencies
go get github.com/fatih/color
go get github.com/manifoldco/promptui
go get github.com/schollz/progressbar/v3
go get golang.org/x/oauth2/google
go get google.golang.org/api/drive/v3

# Build
go build -o gg_drive.exe gg_drive.go
```

## 📝 Examples

### Example 1: Upload a single PDF
```
1. Select "Upload file"
2. Enter path: C:\Documents\report.pdf
3. Enter parent ID: 1n4ADRIJC3qBNAcqj1eKPjYQDLRvj05yj
4. Watch progress bar!
```

### Example 2: Upload entire project folder
```
1. Select "Upload folder"
2. Enter path: C:\Projects\MyApp
3. Enter parent ID: 1n4ADRIJC3qBNAcqj1eKPjYQDLRvj05yj
4. All files and folders uploaded with structure preserved
```

### Example 3: Search for a file
```
1. Select "Search files/folders"
2. Enter search term: budget
3. Enter parent ID: (or press Enter to search everywhere)
4. View results with file IDs
```

## 🌟 Features Comparison

| Feature | Old CLI | New TUI |
|---------|---------|---------|
| Interactive Menu | ❌ | ✅ |
| Progress Bars | ❌ | ✅ |
| Color Output | ❌ | ✅ |
| Config Management | ❌ | ✅ |
| Validation | ❌ | ✅ |
| Visual Feedback | ❌ | ✅ |

## 📄 License

MIT License - Feel free to use and modify!

## 🤝 Contributing

Found a bug or want to add a feature? Feel free to submit a PR!

---

**Made with ❤️ using Go, promptui, and Google Drive API**
