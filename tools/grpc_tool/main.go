package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	var mw *walk.MainWindow
	var logTE *walk.TextEdit
	var folderLE *walk.LineEdit
	var fileCB *walk.ComboBox
	var msgCB *walk.ComboBox
	var sendBtn *walk.PushButton
	var formComp *walk.Composite
	var useDefaultCB *walk.CheckBox

	// State
	backend := NewBackend(func(msg string) {
		if logTE != nil {
			// Append text safely on UI thread
			logTE.Synchronize(func() {
				timestamp := time.Now().Format("15:04:05")
				logTE.AppendText(fmt.Sprintf("[%s] %s\r\n", timestamp, msg))
			})
		}
	})

	var currentProtoFolder string
	var currentFileDesc *desc.FileDescriptor
	var currentMessageDesc *desc.MessageDescriptor
	var payloadDesc *desc.MessageDescriptor
	var currentDynamicMsg *dynamic.Message

	// Rebuild/clear re-entrancy guard
	isBuildingForm := false

	// Models for ComboBoxes
	filesModel := NewStringListModel()
	msgsModel := NewStringListModel()

	// Cache for parsed proto files and their messages
	type ProtoFileInfo struct {
		Descriptor *desc.FileDescriptor
		Messages   []string
	}
	protoCache := make(map[string]*ProtoFileInfo)

	// Start WebSocket server
	port := backend.GetWSPort()
	if err := backend.StartServer(port); err != nil {
		backend.Log("Failed to start server: %v", err)
	}

	// Logic
	refreshFiles := func() {
		path := folderLE.Text()
		if path == "" {
			backend.Log("Please select a folder first.")
			return
		}
		currentProtoFolder = path
		files, err := backend.ScanProtoFiles(path)
		if err != nil {
			backend.Log("Error scanning files: %v", err)
			return
		}
		sort.Strings(files)

		backend.Log("Found %d proto files. Parsing all files...", len(files))

		// Clear cache
		protoCache = make(map[string]*ProtoFileInfo)

		// Parse all files in parallel
		var wg sync.WaitGroup
		var mu sync.Mutex
		successCount := 0

		for _, file := range files {
			wg.Add(1)
			go func(filename string) {
				defer wg.Done()

				fd, err := backend.ParseProto(currentProtoFolder, filename)
				if err != nil {
					backend.Log("Failed to parse %s: %v", filename, err)
					return
				}

				// Extract message names
				var msgs []string
				for _, msg := range fd.GetMessageTypes() {
					msgs = append(msgs, msg.GetName())
				}
				sort.Strings(msgs)

				// Store in cache
				mu.Lock()
				protoCache[filename] = &ProtoFileInfo{
					Descriptor: fd,
					Messages:   msgs,
				}
				successCount++
				mu.Unlock()
			}(file)
		}

		wg.Wait()
		backend.Log("Successfully parsed %d/%d proto files.", successCount, len(files))

		filesModel.Items = files
		filesModel.PublishItemsReset()

		// Reset selection
		fileCB.SetCurrentIndex(-1)
		msgsModel.Items = []string{}
		msgsModel.PublishItemsReset()

		// Load Payload.proto for wrapping messages
		if info, ok := protoCache["Payload.proto"]; ok {
			payloadDesc = info.Descriptor.FindMessage("Kiosk.Payload")
			if payloadDesc != nil {
				backend.Log("Loaded Kiosk.Payload definition.")
			} else {
				backend.Log("Error: Kiosk.Payload message not found in Payload.proto")
			}
		} else {
			backend.Log("Warning: Could not load Payload.proto")
		}
	}

	// Safely clear the dynamic input form
	clearForm := func() {
		if formComp == nil {
			return
		}
		// Ensure this runs on the UI thread, and avoid re-entrancy
		formComp.Synchronize(func() {
			if isBuildingForm {
				return
			}
			isBuildingForm = true
			defer func() { isBuildingForm = false }()
			formComp.SetSuspended(true)
			defer formComp.SetSuspended(false)
			// Dispose all dynamically added controls
			formComp.Children().Clear()
		})
	}

	onFileSelected := func() {
		idx := fileCB.CurrentIndex()
		if idx < 0 || idx >= len(filesModel.Items) {
			return
		}
		selected := filesModel.Items[idx]

		// Get from cache
		info, ok := protoCache[selected]
		if !ok {
			backend.Log("Error: %s not found in cache", selected)
			return
		}

		backend.Log("Selected %s (%d messages)", selected, len(info.Messages))
		currentFileDesc = info.Descriptor

		// Reset current message selection and clear the dynamic form
		currentMessageDesc = nil
		currentDynamicMsg = nil
		sendBtn.SetEnabled(false)
		clearForm()

		// Update messages model after clearing to avoid re-entrancy issues
		msgsModel.Items = info.Messages
		msgsModel.PublishItemsReset()
		// Ensure no selection
		msgCB.SetCurrentIndex(-1)
	}

	// Imperative Form Generation
	generateForm := func() {
		// Guard against nested calls that could corrupt the widget list
		if isBuildingForm {
			return
		}
		isBuildingForm = true
		defer func() { isBuildingForm = false }()

		formComp.SetSuspended(true)
		defer formComp.SetSuspended(false)

		// Dispose all children safely
		formComp.Children().Clear()

		if currentMessageDesc == nil {
			return
		}

		currentDynamicMsg = dynamic.NewMessage(currentMessageDesc)

		for _, field := range currentMessageDesc.GetFields() {
			f := field

			// Label
			lbl, _ := walk.NewLabel(formComp)
			lbl.SetText(f.GetName() + " (" + f.GetType().String() + ")")

			// Input
			if f.IsRepeated() {
				_, _ = walk.NewLineEdit(formComp)
				// le.SetToolTipText("Comma separated values")
			} else {
				switch f.GetType() {
				case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
					chk, _ := walk.NewCheckBox(formComp)
					chk.SetText("True")
					chk.CheckedChanged().Attach(func() {
						currentDynamicMsg.SetField(f, chk.Checked())
					})
				case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
					cmb, _ := walk.NewComboBox(formComp)
					var items []string
					for _, v := range f.GetEnumType().GetValues() {
						items = append(items, v.GetName())
					}
					cmb.SetModel(items)
					cmb.CurrentIndexChanged().Attach(func() {
						idx := cmb.CurrentIndex()
						if idx >= 0 {
							valDesc := f.GetEnumType().FindValueByName(items[idx])
							if valDesc != nil {
								currentDynamicMsg.SetField(f, valDesc.GetNumber())
							}
						}
					})
				default:
					// String, Int, etc.
					le, _ := walk.NewLineEdit(formComp)
					le.TextChanged().Attach(func() {
						txt := le.Text()
						if f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_STRING {
							currentDynamicMsg.SetField(f, txt)
						} else {
							// Int parsing
							var val int64
							fmt.Sscanf(txt, "%d", &val)
							if f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_INT32 {
								currentDynamicMsg.SetField(f, int32(val))
							} else {
								currentDynamicMsg.SetField(f, val)
							}
						}
					})
				}
			}
		}
	}

	onMessageSelected := func() {
		idx := msgCB.CurrentIndex()
		if idx < 0 || idx >= len(msgsModel.Items) {
			return
		}
		selected := msgsModel.Items[idx]

		if currentFileDesc == nil {
			return
		}
		currentMessageDesc = currentFileDesc.FindMessage(currentFileDesc.GetPackage() + "." + selected)
		if currentMessageDesc == nil {
			currentMessageDesc = currentFileDesc.FindMessage(selected)
		}

		if currentMessageDesc == nil {
			backend.Log("Error: Could not find message descriptor.")
			return
		}

		// Defer form generation until after current event completes to avoid re-entrancy
		mw.Synchronize(func() {
			generateForm()
			sendBtn.SetEnabled(true)
		})
	}

	// Main Window
	if _, err := (MainWindow{
		AssignTo: &mw,
		Title:    "gRPC Tool - WebSocket Server",
		MinSize:  Size{Width: 900, Height: 700},
		Layout:   VBox{MarginsZero: false, SpacingZero: false},
		Font:     Font{PointSize: 10, Family: "Segoe UI"},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 4, Spacing: 10, Margins: Margins{Top: 10, Bottom: 10, Left: 10, Right: 10}},
				Children: []Widget{
					CheckBox{
						AssignTo:   &useDefaultCB,
						Text:       "Use default path",
						Checked:    true,
						ColumnSpan: 4,
						Font:       Font{PointSize: 10},
						OnCheckedChanged: func() {
							folderLE.SetEnabled(!useDefaultCB.Checked())
							if useDefaultCB.Checked() {
								cwd, _ := os.Getwd()
								folderLE.SetText(filepath.Join(cwd, "..", "Protobuf"))
							}
						},
					},
					LineEdit{
						AssignTo:   &folderLE,
						Text:       filepath.Join(func() string { d, _ := os.Getwd(); return d }(), "..", "Protobuf"),
						Enabled:    false,
						ColumnSpan: 2,
						MinSize:    Size{Width: 400, Height: 28},
						Font:       Font{PointSize: 10},
					},
					PushButton{
						Text:    "Browse...",
						MinSize: Size{Width: 100, Height: 28},
						Font:    Font{PointSize: 10},
						OnClicked: func() {
							dlg := new(walk.FileDialog)
							dlg.FilePath = folderLE.Text()
							if ok, _ := dlg.ShowBrowseFolder(mw); ok {
								folderLE.SetText(dlg.FilePath)
								useDefaultCB.SetChecked(false)
							}
						},
					},
					PushButton{
						Text:      "Scan Files",
						MinSize:   Size{Width: 100, Height: 28},
						Font:      Font{PointSize: 10},
						OnClicked: refreshFiles,
					},
				},
			},
			Composite{
				Layout: Grid{Columns: 4, Spacing: 10, Margins: Margins{Top: 5, Bottom: 10, Left: 10, Right: 10}},
				Children: []Widget{
					Label{
						Text:    "Proto File:",
						Font:    Font{PointSize: 10, Bold: true},
						MinSize: Size{Width: 80, Height: 25},
					},
					ComboBox{
						AssignTo:              &fileCB,
						Model:                 filesModel,
						MinSize:               Size{Width: 300, Height: 28},
						Font:                  Font{PointSize: 10},
						OnCurrentIndexChanged: onFileSelected,
					},
					Label{
						Text:    "Message:",
						Font:    Font{PointSize: 10, Bold: true},
						MinSize: Size{Width: 80, Height: 25},
					},
					ComboBox{
						AssignTo:              &msgCB,
						Model:                 msgsModel,
						MinSize:               Size{Width: 300, Height: 28},
						Font:                  Font{PointSize: 10},
						OnCurrentIndexChanged: onMessageSelected,
					},
				},
			},
			Composite{
				Layout: HBox{Margins: Margins{Top: 10, Bottom: 10, Left: 10, Right: 10}},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &sendBtn,
						Text:     "📤 Send Message",
						Enabled:  false,
						MinSize:  Size{Width: 150, Height: 40},
						Font:     Font{PointSize: 11, Bold: true},
						OnClicked: func() {
							if currentDynamicMsg == nil {
								return
							}
							if payloadDesc == nil {
								backend.Log("Error: Payload descriptor not loaded.")
								return
							}
							err := backend.Send(currentDynamicMsg, payloadDesc)
							if err != nil {
								backend.Log("Send error: %v", err)
							}
						},
					},
					HSpacer{},
				},
			},
			GroupBox{
				Title:   "Message Fields",
				Layout:  VBox{Margins: Margins{Top: 5, Bottom: 5, Left: 5, Right: 5}},
				MinSize: Size{Width: 0, Height: 200},
				Font:    Font{PointSize: 10, Bold: true},
				Children: []Widget{
					ScrollView{
						Layout: VBox{},
						Children: []Widget{
							Composite{
								AssignTo: &formComp,
								Layout:   VBox{Spacing: 5},
							},
						},
					},
				},
			},
			GroupBox{
				Title:  "Application Log",
				Layout: VBox{Margins: Margins{Top: 5, Bottom: 5, Left: 5, Right: 5}},
				Font:   Font{PointSize: 10, Bold: true},
				Children: []Widget{
					TextEdit{
						AssignTo: &logTE,
						ReadOnly: true,
						VScroll:  true,
						Font:     Font{PointSize: 9, Family: "Consolas"},
					},
				},
			},
		},
	}.Run()); err != nil {
		fmt.Fprintln(os.Stderr, "Error running application:", err)
		os.Exit(1)
	}
}

// StringListModel helper
type StringListModel struct {
	walk.ListModelBase
	Items []string
}

func NewStringListModel() *StringListModel {
	return &StringListModel{}
}

func (m *StringListModel) ItemCount() int {
	return len(m.Items)
}

func (m *StringListModel) Value(index int) interface{} {
	return m.Items[index]
}
