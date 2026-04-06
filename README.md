# ide

A lightweight terminal-based IDE with Emacs-like appearance, designed for Go programming.

## Features

- **Emacs-like UI**: White background with black text
- **Vi integration**: Full vi editor support with status bar
- **Window management**: Split windows horizontally/vertically (up to 8)
- **Go language support**: Build, run, and navigate compilation errors
- **Keyboard-driven**: ALT-based commands for all IDE functions
- **Auto-save**: Files are automatically saved before build/run commands
- **Mouse scrolling**: Scroll through shell and output windows with mouse wheel

## Quick Start

```bash
# Build the IDE
go build -o ide

# Run the IDE
./ide

# In the IDE:
# - Open a file: vi test.go
# - Build: ALT+b
# - Run: ALT+r
# - Split window: ALT+h (horizontal) or ALT+v (vertical)
# - Quit: ALT+q
```

See [USAGE.md](USAGE.md) for detailed documentation.

## Requirements

- Linux or WSL
- Go 1.24 or higher
- Standard vi/vim editor

## Key Bindings

| Command | Function |
|---------|----------|
| ALT+h | Split window horizontally |
| ALT+v | Split window vertically |
| ALT+x | Close current window |
| ALT+t | Toggle to previous window |
| ALT+1-8 | Jump to window number |
| ALT+b | Build current Go file |
| ALT+r | Run current Go file |
| ALT+q | Save all and quit |

## License

See [LICENSE](LICENSE) file.
