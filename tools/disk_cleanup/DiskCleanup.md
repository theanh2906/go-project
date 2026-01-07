# 🧹 Windows Disk Cleanup Tool

A fast, interactive command-line tool for cleaning up Windows disk space, written in Go.

## Features

- 🔍 **Smart Scanning**: Automatically detects cleanup opportunities
- ✅ **Safe Operations**: Distinguishes between safe and potentially risky cleanups
- 🎯 **Targeted Cleanup**: Cleans temp files, caches, crash dumps, and more
- 📊 **Real-time Reporting**: Shows disk space before/after with human-readable sizes
- 🔄 **Interactive Menu**: Choose what to clean or let it auto-clean safe items
- 🧪 **Dry Run Mode**: Preview what would be cleaned without making changes
- ⚡ **Fast & Lightweight**: Single executable, no dependencies

## What It Cleans

### Safe to Clean Automatically ✅
- **Temporary files** (Windows temp, user temp)
- **Browser caches** (Edge, Chrome, Puppeteer, Codeium)
- **Java crash dumps** (.hprof files)
- **Windows Update cache** (re-downloaded when needed)
- **Recycle Bin contents**
- **Windows prefetch files**
- **Thumbnail cache**
- **Old log files** (keeps recent ones)

### User Review Required ⚠️
- **Large downloads** (installers, archives >100MB)
- **Duplicate files** in Downloads folder
- **Development caches** (Maven, npm, etc.)

## Installation

### Option 1: Build from Source
```bash
# Clone or download the code
cd disk-cleanup-tool

# Download dependencies
go mod tidy

# Build executable
go build -o disk-cleanup.exe main.go
```

### Option 2: Direct Run
```bash
go run main.go
```

## Usage

### Interactive Mode (Recommended)
```bash
./disk-cleanup.exe
```

The tool will:
1. Show current disk space information
2. Scan for cleanup opportunities 
3. Present an interactive menu with options:
   - Clean all safe items automatically
   - Select specific items to clean
   - View disk space info
   - Dry run (preview only)

### Example Output
```
🧹 Windows Disk Cleanup Tool
=====================================

💾 Disk Space Information for C:
--------------------------------------------------
Total Space: 299 GB
Used Space:  264 GB  
Free Space:  35 GB
Percent Free: 11.63%

🔍 Scanning for cleanup opportunities...

📊 Cleanup Opportunities Found:
================================================================================

📁 Crash Dumps:
  ✅ Java Crash Dump: java_error_in_idea.hprof       6.8 GB
     Java application crash dump file

📁 Temporary Files:
  ✅ User Temp Files                                  245 MB
     Temporary files created by applications
  ✅ Windows Temp Files                              298 MB
     System temporary files

💾 Total potential cleanup: 7.3 GB
✅ Safe to clean automatically: 7.3 GB
```

## Command Line Options

The tool currently runs in interactive mode. Future versions may add CLI flags for:
- `--auto-clean` - Clean all safe items without prompts
- `--dry-run` - Preview mode only
- `--target <category>` - Clean specific categories only

## Safety Features

- **Administrator Check**: Warns if not running as admin (some operations may be limited)
- **Safety Classification**: Each cleanup item is marked as safe or requiring user review
- **Dry Run Mode**: Preview changes before applying them
- **Selective Cleaning**: Choose exactly what to clean
- **Real-time Feedback**: Shows success/failure for each cleanup operation

## Performance

- **Fast Scanning**: Efficiently walks directory trees
- **Memory Efficient**: Processes large directories without excessive memory usage
- **Native Performance**: Compiled Go binary with no runtime dependencies
- **Windows Optimized**: Uses Windows APIs for disk space and file operations

## Requirements

- Windows 10/11
- Go 1.21+ (if building from source)
- Recommended: Run as Administrator for full functionality

## Limitations

- Windows only (by design)
- Some operations require administrator privileges
- Large directory scans may take time on slow drives

## Contributing

Feel free to submit issues or pull requests. Areas for enhancement:
- Additional cleanup categories
- Scheduled cleanup options  
- GUI interface
- More granular safety controls
- Cross-drive support

## License

MIT License - feel free to use and modify as needed.

## Safety Disclaimer

While this tool is designed to be safe, always:
- Run a dry run first on important systems
- Ensure you have backups of critical data
- Review what will be cleaned before proceeding
- Test on a non-production system first

The tool focuses on cleaning temporary files, caches, and clearly unnecessary data, but use at your own discretion.
