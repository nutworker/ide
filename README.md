# ide

A lightweight terminal-based IDE with Emacs-like appearance, designed for Go programming.

## Features

- **Emacs-like UI**: White background with black text
- **Vi integration**: Full vi editor support with live status bar (cursor position, mode)
- **Window management**: Split windows horizontally/vertically (up to 8)
- **File navigation**: Emacs-style find-file (`ALT+F`) with Tab completion and history
- **Go language support**: Build, run, and navigate compilation errors
- **Keyboard-driven**: ALT-based commands for all IDE functions
- **Auto-save**: Files are automatically saved before build/run commands
- **Mouse support**: Text selection, copy/paste, and scrolling
- **Smart scrolling**: Full vi scrolling support (Ctrl-F, Ctrl-B, arrows)

## Quick Start

```bash
# Build the IDE
go build -o ide

# Run the IDE
./ide

# In the IDE:
# - Open a file: ALT+f (then type filename with Tab completion)
# - Or: vi test.go
# - Build: ALT+b
# - Run: ALT+r
# - Split window: ALT+h (horizontal) or ALT+v (vertical)
# - Quit: ALT+q
```

## Documentation

- [USAGE.md](USAGE.md) - Detailed usage guide with examples
- [FEATURES.md](FEATURES.md) - Complete feature list and technical details

## Requirements

- Linux or WSL
- Go 1.24 or higher
- Standard vi/vim editor

## Key Bindings

| Command | Function |
|---------|----------|
| ALT+f | Find/open file (with Tab completion) |
| ALT+h | Split window horizontally |
| ALT+v | Split window vertically |
| ALT+x | Close current window |
| ALT+t | Toggle to previous window |
| ALT+1-8 | Jump to window number |
| ALT+b | Build current Go file |
| ALT+r | Run current Go file |
| ALT+q | Save all and quit |
| Ctrl+Shift+C | Copy selection to clipboard |
| Ctrl+P | Paste from clipboard |

## License

See [LICENSE](LICENSE) file.
