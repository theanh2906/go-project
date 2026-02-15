# gRPC Tool (WebSocket)

This tool allows you to send Protobuf messages over WebSocket, similar to the existing Kiosk UI logic.

## Prerequisites

- Go 1.21 or later
- Windows (uses native Windows controls via `lxn/walk`)

## Setup

1.  Open a terminal in this folder (`gRPC Tool`).
2.  Initialize the module (if not already done):
    ```bash
    go mod tidy
    ```
3.  Run the tool:
    ```bash
    go run .
    ```

## Usage

1.  **Select Folder**: Choose the `Protobuf` folder in your workspace.
    *   The tool tries to default to `Protobuf` in the current working directory.
2.  **Scan**: Click "Scan" to list all `.proto` files.
3.  **Select File**: Choose a `.proto` file (e.g., `Auth.proto`).
4.  **Select Message**: Choose a message type (e.g., `AuthRequest`).
5.  **Fill Form**: Enter the data.
6.  **Connect**: Click "Connect" to establish WebSocket connection to the Kiosk Service.
    *   Port is read from Registry or defaults to `54675`.
7.  **Send**: Click "Send".

## Notes

- The tool uses `Payload.proto` and `Metadata.proto` to wrap messages, mimicking the Kiosk protocol.
- Ensure `Payload.proto` is present in the selected folder (or root of scan).
- Logs are displayed in the bottom panel.
