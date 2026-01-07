package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"golang.org/x/sys/windows/registry"
)

const (
	DefaultWSPort = "54675"
	RegistryKey   = `SOFTWARE\WOW6432Node\OPSWAT\MD4M`
	RegistryValue = "ws_port"
)

type Backend struct {
	Conn        *websocket.Conn
	Server      *http.Server
	LogFunc     func(string)
	ProtoFolder string
	FileDescs   map[string]*desc.FileDescriptor
	mu          sync.Mutex
	upgrader    websocket.Upgrader
}

func NewBackend(logFunc func(string)) *Backend {
	return &Backend{
		LogFunc:   logFunc,
		FileDescs: make(map[string]*desc.FileDescriptor),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (b *Backend) Log(format string, args ...interface{}) {
	if b.LogFunc != nil {
		b.LogFunc(fmt.Sprintf(format, args...))
	}
}

func (b *Backend) GetWSPort() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, RegistryKey, registry.QUERY_VALUE)
	if err != nil {
		b.Log("Registry key not found, using default port: %s", DefaultWSPort)
		return DefaultWSPort
	}
	defer k.Close()

	port, _, err := k.GetStringValue(RegistryValue)
	if err != nil {
		b.Log("Registry value not found, using default port: %s", DefaultWSPort)
		return DefaultWSPort
	}
	b.Log("Found port in registry: %s", port)
	return port
}

func (b *Backend) ScanProtoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".proto") {
			rel, _ := filepath.Rel(root, path)
			// Convert to Unix-style path (forward slashes) for protoparse
			rel = filepath.ToSlash(rel)
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

func (b *Backend) ParseProto(root string, filename string) (*desc.FileDescriptor, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check cache
	if fd, ok := b.FileDescs[filename]; ok {
		return fd, nil
	}

	b.Log("Parsing proto file: %s", filename)
	b.Log("Root directory: %s", root)

	// Ensure filename uses forward slashes
	filename = filepath.ToSlash(filename)

	parser := protoparse.Parser{
		ImportPaths: []string{root},
	}

	fds, err := parser.ParseFiles(filename)
	if err != nil {
		b.Log("Parse error: %v", err)
		return nil, err
	}

	fd := fds[0]
	b.FileDescs[filename] = fd
	b.Log("Successfully parsed: %s", filename)
	return fd, nil
}

func (b *Backend) StartServer(port string) error {
	addr := fmt.Sprintf(":%s", port)
	b.Log("Starting WebSocket server on %s...", addr)

	mux := http.NewServeMux()
	mux.HandleFunc("/", b.handleWebSocket)

	b.Server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := b.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			b.Log("Server error: %v", err)
		}
	}()

	b.Log("WebSocket server started on ws://localhost%s", addr)
	return nil
}

func (b *Backend) StopServer() {
	if b.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		b.Server.Shutdown(ctx)
		b.Log("Server stopped.")
	}
}

func (b *Backend) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		b.Log("Upgrade error: %v", err)
		return
	}

	b.mu.Lock()
	b.Conn = conn
	b.mu.Unlock()

	b.Log("Client connected from %s", r.RemoteAddr)

	// Start listening
	go b.listen()
}

func (b *Backend) listen() {
	conn := b.Conn
	defer func() {
		b.Log("Client disconnected.")
		if conn != nil {
			conn.Close()
		}
		b.mu.Lock()
		if b.Conn == conn {
			b.Conn = nil
		}
		b.mu.Unlock()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			b.Log("Read error: %v", err)
			return
		}

		// Parse message
		// Format: 4 bytes length (Little Endian) + Payload
		if len(message) < 4 {
			b.Log("Received message too short")
			continue
		}

		length := binary.LittleEndian.Uint32(message[0:4])
		if int(length) != len(message)-4 {
			b.Log("Warning: Message length mismatch. Header: %d, Actual: %d", length, len(message)-4)
		}

		payloadData := message[4:]
		b.Log("Received %d bytes payload", len(payloadData))

		// Here we should deserialize the payload
		// We need the Payload descriptor.
		// For now, just log the raw bytes or try to decode if we have the descriptor loaded.
		// In a real app, we would decode Kiosk.Payload.
	}
}

func (b *Backend) Send(msg *dynamic.Message, payloadDesc *desc.MessageDescriptor) error {
	b.mu.Lock()
	conn := b.Conn
	b.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("no client connected")
	}

	// 1. Serialize the inner message
	innerBytes, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal inner message: %w", err)
	}

	// 2. Create Any
	// We need to construct google.protobuf.Any manually or using dynamic message
	// Since we are using dynamic messages, we should create a dynamic message for Any.
	// However, creating Any manually is easier if we just set type_url and value.

	// We need the descriptor for Kiosk.Payload and Kiosk.Metadata.
	// Assuming payloadDesc is Kiosk.Payload.

	payloadMsg := dynamic.NewMessage(payloadDesc)

	// Create Metadata
	metaField := payloadDesc.FindFieldByName("metadata")
	if metaField != nil {
		metaMsg := dynamic.NewMessage(metaField.GetMessageType())
		dataField := metaMsg.GetMessageDescriptor().FindFieldByName("data")
		if dataField != nil {
			// Set transactionID
			metaMsg.PutMapField(dataField, "transactionID", fmt.Sprintf("%d", time.Now().UnixMilli()))
		}
		payloadMsg.SetField(metaField, metaMsg)
	}

	// Create Any message
	messageField := payloadDesc.FindFieldByName("message")
	if messageField != nil {
		anyMsg := dynamic.NewMessage(messageField.GetMessageType())

		// Set type_url
		// The format in the TS code is just the message name (e.g. "AuthRequest"),
		// but standard Any uses "type.googleapis.com/packagename.MessageName".
		// The TS code: `anyProto.setTypeUrl(caseType);` where caseType is `message.getTypeUrl().replace(/.*(?=\/)/g, "")` which seems to strip the prefix?
		// Wait, `message.getTypeUrl()` usually returns the full URL.
		// In `constructProtobuf`: `anyProto.setTypeUrl(caseType);`
		// And `caseType` comes from the input JSON.
		// In `getMessage`: `caseType: message.getTypeUrl().replace(/.*(?=\/)/g, "")` -> this REMOVES the prefix.
		// So the TS code expects just the message name?
		// Let's check `constructProtobuf` again.
		// `anyProto.setTypeUrl(caseType);`
		// If `caseType` is "AuthRequest", then it sets "AuthRequest".

		// However, `google.protobuf.Any` usually expects a URL.
		// Let's look at `unpack`:
		// `const typeurl = message.getTypeUrl();`
		// `const dotIndex = typeurl.lastIndexOf(".");`
		// `const normalized = typeurl.substring(dotIndex + 1);`
		// `if (proto.Kiosk[normalized]) ...`

		// So it seems the system is quite loose with the Type URL.
		// I will use the full name of the message.

		anyMsg.SetFieldByName("type_url", "type.googleapis.com/"+msg.GetMessageDescriptor().GetFullyQualifiedName())
		anyMsg.SetFieldByName("value", innerBytes)

		payloadMsg.SetField(messageField, anyMsg)
	}

	// 3. Serialize Payload
	payloadBytes, err := payloadMsg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 4. Add Length Prefix (2 bytes, Little Endian)
	buf := make([]byte, 2+len(payloadBytes))
	binary.LittleEndian.PutUint16(buf[0:2], uint16(len(payloadBytes)))
	copy(buf[2:], payloadBytes)

	// 5. Send
	err = conn.WriteMessage(websocket.BinaryMessage, buf)
	if err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	b.Log("Sent message: %s (%d bytes)", msg.GetMessageDescriptor().GetName(), len(payloadBytes))
	return nil
}
