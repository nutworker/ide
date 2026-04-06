# IDE Usage Guide

## Installation

To make `ide` available in your PATH:

```bash
sudo cp ide /usr/local/bin/
# or
cp ide ~/bin/  # if ~/bin is in your PATH
```

## Running the IDE

Simply run:
```bash
./ide
```

This will launch the IDE with a bash shell in a white background terminal window.

## Mouse Support

### Scrolling
- **Mouse Wheel Up** - Scroll up through output (3 lines)
- **Mouse Wheel Down** - Scroll down through output (3 lines)
- Works in: Shell windows, build output, run output
- Disabled in: Vi editing mode (use vi's native scrolling)

## Key Bindings

### Window Management
- `ALT+h` - Split current window horizontally (bash→bash, vi→vi with same file)
- `ALT+v` - Split current window vertically (bash→bash, vi→vi with same file)
- `ALT+x` - Close current window (space reclaimed by parent window)
- `ALT+t` - Toggle to previously active window
- `ALT+1` through `ALT+8` - Jump to window number

### Go Programming
- `ALT+b` - Build current Go file (spawns build output window)
- `ALT+r` - Run current Go file (spawns run output window)
- In build output window, press `Enter` on an error line to jump to that line in the source

### General
- `ALT+q` - Save all open files and quit the IDE
- `CTRL+C` - Emergency quit

## Workflow Example

1. Launch IDE: `./ide`
2. In the bash shell, open a Go file: `vi test.go`
3. Edit your code (status bar shows filename, cursor position, and mode)
4. Press `ALT+b` to build (auto-saves first)
5. If there are errors, press `Enter` on the error line to jump to it
6. Fix errors and build again
7. Press `ALT+r` to run your program
8. Use `ALT+h` or `ALT+v` to split windows for multi-file editing
9. Press `ALT+q` to save and quit

## Features

- **White background** Emacs-like appearance
- **Vi integration** with status bar showing mode and cursor position
- **Auto-save** before ALT commands
- **Window splitting** up to 8 windows
- **Window numbers** displayed in top-left corner
- **Go build/run** integration with error navigation
- **Keyboard-driven** workflow (no mouse needed)

## Tips

- Windows are numbered 1-8 in the order they are created
- The active window has a brighter window number
- Vi status bar appears automatically when editing files
- Build/run output windows stay open until you close them manually
- All vi files are auto-saved before ALT commands execute
