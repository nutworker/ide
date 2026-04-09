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

### Text Selection and Copy/Paste
- **Click and drag** - Select text (auto-copies to clipboard on release)
- **Double-click** - Select word, then drag to extend selection word-by-word
- **Triple-click** - Select entire line
- **Middle-click** - Paste from clipboard (X11 style)
- **Ctrl+Shift+C** - Copy selection to clipboard
- **Ctrl+P** - Paste from clipboard (in non-vi windows)

### Scrolling
- **Mouse Wheel Up** - Scroll up through output (3 lines)
- **Mouse Wheel Down** - Scroll down through output (3 lines)
- Works in: Shell windows, build output, run output
- Disabled in: Vi editing mode (use vi's native scrolling)

## Key Bindings

### File Navigation
- `ALT+f` - Find/open file (Emacs-style)
  - Type filename or path
  - Press `Tab` for auto-completion
  - Supports directory navigation with `/`
  - Supports home directory with `~`
  - Press `Enter` to open, `Esc` to cancel
  - When you quit vi, automatically returns to previous file (file history stack)

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
- `CTRL+C` - Send interrupt signal to active window

## Workflow Examples

### Basic File Editing
1. Launch IDE: `./ide`
2. Open a file: `ALT+f` → type `test.go` → `Enter`
3. Edit your code (status bar shows: filename, line:column, INSERT/COMMAND mode)
4. Open another file: `ALT+f` → type `utils.go` (use Tab to autocomplete) → `Enter`
5. Quit back to first file: `:q` (automatically returns to test.go)
6. Press `ALT+q` to save and quit

### Go Development Workflow
1. Launch IDE: `./ide`
2. Open a Go file: `ALT+f` → type `main.go` → `Enter`
3. Edit your code (status bar shows live cursor position)
4. Press `ALT+b` to build (auto-saves first)
5. If there are errors, press `Enter` on the error line to jump to that line in source
6. Fix errors and build again
7. Press `ALT+r` to run your program
8. Use `ALT+h` or `ALT+v` to split windows for multi-file editing
9. Press `ALT+q` to save all and quit

### Multi-File Editing
1. Open first file: `ALT+f` → `app.go`
2. Split window: `ALT+v`
3. In new window, open second file: `ALT+f` → `internal/` (Tab to see completions)
4. Navigate completions with Tab, select `internal/ui/renderer.go`
5. Jump between windows with `ALT+t` or `ALT+1`, `ALT+2`

## Features

- **White background** Emacs-like appearance
- **Vi integration** with live status bar showing mode, cursor position, and filename
- **File history stack** - quitting a file returns to the previous one
- **Tab completion** - smart file path completion with directory support
- **Auto-save** before ALT commands
- **Window splitting** up to 8 windows
- **Window numbers** displayed in top-left corner
- **Mouse support** - text selection, copy/paste, scrolling
- **Go build/run** integration with error navigation
- **Terminal scrolling** - full vi scrolling support (Ctrl-F, Ctrl-B, etc.)
- **Keyboard-driven** workflow (mouse optional)

## Tips

### General
- Windows are numbered 1-8 in the order they are created
- The active window has a brighter window number
- Vi status bar appears automatically when editing files
- Build/run output windows stay open until you close them manually
- All vi files are auto-saved before ALT commands execute

### File Navigation
- Use Tab liberally while typing filenames - it completes or shows matches
- Start typing a filename and press Tab to see all matches
- Directory names get a trailing `/` automatically
- The file history stack means you can quickly peek at other files with `ALT+f` and return with `:q`

### Vi Scrolling
- All vi scroll commands work: Ctrl-F (page down), Ctrl-B (page up), Ctrl-D (half page down), Ctrl-U (half page up)
- Vi's scrolling region support is fully implemented
- Status line remains at bottom even during scrolling

### Text Selection
- Double-click to select a word, then drag to extend selection word-by-word
- Triple-click to select an entire line
- Selection is automatically copied to clipboard when you release the mouse
- Middle-click to paste (X11/Linux style)
