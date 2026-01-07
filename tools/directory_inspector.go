package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type FileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
	IsDir   bool
}

type SortBy int

// Helper function to format file size in human-readable format
func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}

// Helper function to get appropriate icon for file or directory
func getFileIcon(isDir bool) fyne.Resource {
	if isDir {
		return theme.FolderIcon()
	}
	return theme.FileIcon()
}

func main() {
	a := app.New()
	w := a.NewWindow("Directory Inspector GUI")
	w.Resize(fyne.NewSize(800, 600))

	// Initialize data
	var fileInfos []FileInfo

	// Create the file list table first (it's referenced in other functions)
	var list *widget.Table

	// Create status bar components
	statusLabel := widget.NewLabel("Ready")
	statusContainer := container.NewHBox(statusLabel, widget.NewLabel(""))

	// Create search and sort components
	searchEntry := widget.NewEntryWithData(binding.NewString())
	searchEntry.SetPlaceHolder("Search files...")
	searchIcon := widget.NewIcon(theme.SearchIcon())

	sortIcon := widget.NewIcon(theme.ListIcon())
	sortSelect := widget.NewSelect([]string{"Name ↑", "Name ↓", "Size ↑", "Size ↓"}, nil) // We'll set the callback later

	// Define the filter and sort function
	applyFilterAndSort := func(searchText, sortOption string) {
		// First filter
		filteredInfos := []FileInfo{}
		if searchText != "" {
			searchLower := strings.ToLower(searchText)
			for _, info := range fileInfos {
				if strings.Contains(strings.ToLower(info.Name), searchLower) {
					filteredInfos = append(filteredInfos, info)
				}
			}
		} else {
			filteredInfos = fileInfos
		}

		// Then sort
		switch sortOption {
		case "Name ↑":
			sort.Slice(filteredInfos, func(i, j int) bool { return filteredInfos[i].Name < filteredInfos[j].Name })
		case "Name ↓":
			sort.Slice(filteredInfos, func(i, j int) bool { return filteredInfos[i].Name > filteredInfos[j].Name })
		case "Size ↑":
			sort.Slice(filteredInfos, func(i, j int) bool { return filteredInfos[i].Size < filteredInfos[j].Size })
		case "Size ↓":
			sort.Slice(filteredInfos, func(i, j int) bool { return filteredInfos[i].Size > filteredInfos[j].Size })
		}

		fileInfos = filteredInfos
		list.Refresh()

		// Update status
		statusLabel.SetText(fmt.Sprintf("%d items found", len(filteredInfos)))
	}

	// Now set the sort select callback
	sortSelect.OnChanged = func(option string) {
		applyFilterAndSort(searchEntry.Text, option)
	}

	// Connect search entry to filter function
	searchEntry.OnChanged = func(s string) {
		applyFilterAndSort(s, sortSelect.Selected)
	}

	// Initialize the file list table
	list = widget.NewTable(
		func() (int, int) { return len(fileInfos) + 1, 3 },
		func() fyne.CanvasObject {
			return container.NewHBox(widget.NewIcon(theme.DocumentIcon()), widget.NewLabel(""))
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			container := o.(*fyne.Container)
			icon := container.Objects[0].(*widget.Icon)
			label := container.Objects[1].(*widget.Label)

			// Apply alternating row colors for better readability
			if id.Row%2 == 0 {
				bg := canvas.NewRectangle(theme.BackgroundColor())
				container.Objects = append([]fyne.CanvasObject{bg}, container.Objects...)
			}

			if id.Row == 0 {
				// Header row styling
				label.TextStyle = fyne.TextStyle{Bold: true}
				switch id.Col {
				case 0:
					icon.SetResource(theme.ListIcon())
					label.SetText("Name")
				case 1:
					icon.SetResource(theme.StorageIcon())
					label.SetText("Size")
				case 2:
					icon.SetResource(theme.HistoryIcon())
					label.SetText("Date Modified")
				}
			} else {
				info := fileInfos[id.Row-1]
				if id.Col == 0 {
					// Set appropriate icon for file or directory
					icon.SetResource(getFileIcon(info.IsDir))
					label.SetText(info.Name)
				} else if id.Col == 1 {
					icon.SetResource(nil) // No icon for size column except header
					if info.IsDir {
						label.SetText("<Directory>")
					} else {
						label.SetText(formatFileSize(info.Size))
					}
				} else if id.Col == 2 {
					icon.SetResource(nil) // No icon for date column except header
					label.SetText(info.ModTime.Format("2006-01-02 15:04:05"))
				}
			}
		},
	)

	// Set minimum size for better readability
	list.SetColumnWidth(0, 300)
	list.SetColumnWidth(1, 100)
	list.SetColumnWidth(2, 200)

	// Now it's safe to set the initial sort option
	sortSelect.SetSelected("Name ↑")

	// Create directory selection components
	dirLabel := widget.NewLabelWithStyle("No directory selected", fyne.TextAlignLeading, fyne.TextStyle{})

	// Create a styled button with icon
	selectBtn := widget.NewButtonWithIcon("Select Directory", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri == nil || err != nil {
				return
			}
			path := uri.Path()
			dirLabel.SetText(path)

			// Show loading indicator
			progress := widget.NewProgressBarInfinite()
			statusLabel.SetText("Loading directory contents...")
			statusContainer.Objects[1] = progress

			// Read directory contents
			infos, err := readDirInfo(path)
			if err != nil {
				dialog.ShowError(err, w)
				statusLabel.SetText("Error loading directory")
				statusContainer.Objects[1] = widget.NewLabel("")
				return
			}

			fileInfos = infos
			list.Refresh()

			// Update status bar
			fileCount := len(fileInfos)
			dirCount := 0
			for _, info := range fileInfos {
				if info.IsDir {
					dirCount++
				}
			}

			statusLabel.SetText(fmt.Sprintf("%d items (%d directories, %d files)",
				fileCount, dirCount, fileCount-dirCount))
			statusContainer.Objects[1] = widget.NewLabel("")

			// Apply initial filter and sort
			applyFilterAndSort(searchEntry.Text, sortSelect.Selected)
		}, w)
	})

	// Create the header with search and sort
	headerContainer := container.NewHBox(
		selectBtn,
		dirLabel,
		container.NewHBox(searchIcon, searchEntry),
		container.NewHBox(sortIcon, widget.NewLabel("Sort by:"), sortSelect),
	)

	// Set up the main layout
	content := container.NewBorder(
		headerContainer, // top
		statusContainer, // bottom
		nil,             // left
		nil,             // right
		list,            // center
	)

	w.SetContent(content)
	w.ShowAndRun()
}

func readDirInfo(path string) ([]FileInfo, error) {
	var infos []FileInfo
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if p == path {
			return nil // skip root
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		infos = append(infos, FileInfo{
			Name:    d.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   d.IsDir(),
		})
		return nil
	})
	return infos, err
}
