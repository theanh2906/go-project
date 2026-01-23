# Unit Converter TUI

A beautiful Terminal User Interface (TUI) application for unit conversions, starting with Pixel to REM conversion.

## Features

- 🎨 Beautiful TUI interface with colors and styling
- 🔄 Continuous conversion mode (loop mode)
- 📝 Conversion history (last 5 conversions)
- ⌨️ Keyboard shortcuts for easy navigation
- 🔌 Extensible architecture - easy to add new converters

## Installation

```bash
cd cmd
go build -o unit_converter.exe unit_converter.go
```

## Usage

```bash
./unit_converter.exe
```

### Keyboard Shortcuts

#### Main Menu
- `↑/↓` or `k/j` - Navigate between converters
- `Enter` - Select a converter
- `q` - Quit application

#### Convert Mode
- Type numbers directly to input
- `Enter` - Perform conversion
- `Esc` - Return to main menu
- `Ctrl+C` - Quit application
- `Backspace` - Delete last character

## Current Converters

### Pixel to REM
Converts pixel values to rem units based on a 16px base font size.

**Examples:**
- `16` → `1.0000 rem`
- `24` → `1.5000 rem`
- `32` → `2.0000 rem`

## Adding New Converters

The application is designed to be easily extensible. To add a new converter:

1. Create a new struct that implements the `Converter` interface:

```go
type Converter interface {
    GetName() string
    GetDescription() string
    Convert(input string) (string, error)
    GetInputUnit() string
    GetOutputUnit() string
}
```

2. Add your converter to the `initialModel()` function:

```go
func initialModel() model {
    converters := []Converter{
        NewPxToRemConverter(),
        NewYourNewConverter(), // Add here
    }
    // ...
}
```

### Example: REM to Pixel Converter

```go
type RemToPxConverter struct {
    baseFontSize float64
}

func NewRemToPxConverter() *RemToPxConverter {
    return &RemToPxConverter{baseFontSize: 16.0}
}

func (c *RemToPxConverter) GetName() string {
    return "REM to Pixel"
}

func (c *RemToPxConverter) GetDescription() string {
    return fmt.Sprintf("Convert rem to pixels (base: %.0fpx)", c.baseFontSize)
}

func (c *RemToPxConverter) GetInputUnit() string {
    return "rem"
}

func (c *RemToPxConverter) GetOutputUnit() string {
    return "px"
}

func (c *RemToPxConverter) Convert(input string) (string, error) {
    input = strings.TrimSpace(input)
    input = strings.TrimSuffix(input, "rem")
    
    rem, err := strconv.ParseFloat(input, 64)
    if err != nil {
        return "", fmt.Errorf("invalid number: %v", err)
    }
    
    px := rem * c.baseFontSize
    return fmt.Sprintf("%.2f", px), nil
}
```

## Architecture

```
Converter Interface
    ↓
┌─────────────────┬─────────────────┬─────────────────┐
│  PxToRemConverter │ RemToPxConverter │  YourConverter  │
└─────────────────┴─────────────────┴─────────────────┘
```

Each converter is self-contained and implements the same interface, making the codebase:
- **Maintainable**: Each converter is independent
- **Testable**: Easy to unit test each converter
- **Extensible**: Just implement the interface to add new converters

## Dependencies

- [bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Styling and layout

## Tips

- The converter stays in loop mode, so you can convert multiple values quickly
- Input automatically cleans "px" suffix (you can type "16px" or just "16")
- History shows your last 5 conversions for reference
- Press `Esc` to go back to menu and switch to a different converter
