# IDE Features

Complete feature list for the terminal-based IDE.

## Core Features

### Terminal Emulation
- Full ANSI escape sequence support
- 256-color support
- Scrolling regions (CSI r)
- Character insert/delete (CSI @, P, X)
- Reverse index (ESC M) and index (ESC D)
- Proper newline handling (CRLF, LF)

### Vi Integration
- Automatic vi detection via status line parsing
- Real-time status bar showing:
  - Current filename
  - Cursor position (line, column)
  - Vi mode (INSERT, COMMAND)
- Auto-save before ALT commands
- Swap file disabled (vi -n flag) to avoid conflicts
- Full vi scrolling support:
  - Ctrl-F (page down)
  - Ctrl-B (page up)
  - Ctrl-D (half page down)
  - Ctrl-U (half page up)
  - Arrow keys for line-by-line navigation

### Window Management
- Split windows horizontally (ALT+h) or vertically (ALT+v)
- Up to 8 windows supported
- Binary tree layout algorithm
- Smart window closing with space reclamation
- Window numbering (1-8) displayed in top-left corner
- Active window indicated by brighter number
- Jump to window by number (ALT+1 through ALT+8)
- Toggle between current and previous window (ALT+t)
- When splitting vi, new window opens same file

### File Navigation (Emacs-style)
- **ALT+f** - Find/open file command
- Minibuffer-style prompt at bottom of window
- **Tab completion** with smart features:
  - Shows all matching files/directories
  - Completes to common prefix automatically
  - Adds trailing `/` for directories
  - Supports up to 8 visible completion candidates
  - Works with relative paths (internal/app/)
  - Works with absolute paths (/home/user/)
  - Supports home directory expansion (~/)
  - Hidden files shown only when typing `.`
- **File history stack**:
  - Opening a file pushes current file to stack
  - Quitting vi (`:q`) returns to previous file
  - Unlimited stack depth
  - Perfect for quick file peeks

### Mouse Support

#### Text Selection
- **Click and drag** - Select text character-by-character
- **Double-click** - Select word under cursor
  - Drag after double-click extends selection word-by-word
- **Triple-click** - Select entire line
- Auto-copy to clipboard on mouse release
- Visual feedback with reverse video highlighting
- Works across line boundaries

#### Copy and Paste
- **Ctrl+Shift+C** - Copy selection to system clipboard
- **Ctrl+P** - Paste from clipboard (non-vi windows)
- **Middle-click** - Paste (X11/Linux style)
- Clipboard integration with external applications

#### Scrolling
- **Mouse wheel up** - Scroll up 3 lines
- **Mouse wheel down** - Scroll down 3 lines
- Works in shell windows, build output, run output
- Disabled in vi mode (use vi's native scrolling)

### Go Language Support

#### Building
- **ALT+b** - Build current Go file
- Auto-saves file before building
- Spawns dedicated build output window
- Real-time compilation output
- Error parsing with regex: `filename.go:line:col: message`

#### Running
- **ALT+r** - Run current Go file
- Auto-saves file before running
- Spawns dedicated run output window
- Shows program output in real-time
- Supports interactive programs

#### Error Navigation
- In build output window, press **Enter** on error line
- Automatically jumps to source file and line
- Sends vi command: `ESC :<line>CR`
- Works with multi-file projects

### Keyboard Shortcuts

#### Window Management
- `ALT+h` - Split window horizontally
- `ALT+v` - Split window vertically
- `ALT+x` - Close current window
- `ALT+t` - Toggle to previous window
- `ALT+1-8` - Jump to window number

#### File Operations
- `ALT+f` - Find/open file (with Tab completion)
- `ALT+q` - Save all files and quit IDE

#### Go Development
- `ALT+b` - Build current Go file
- `ALT+r` - Run current Go file

#### Clipboard
- `Ctrl+Shift+C` - Copy selection
- `Ctrl+P` - Paste from clipboard

#### General
- `Ctrl+C` - Send interrupt (in shell/programs)
- `Tab` - Trigger completion (in prompts)
- `Esc` - Cancel prompt/command
- `Enter` - Accept/execute

### Theme
- White background (Emacs-like)
- Black foreground text
- Reverse video for status bars
- Blinking cursor in active window
- Reverse video for text selection
- High contrast for readability

### PTY Management
- Pseudo-terminal for each window
- Generation-based event tracking (prevents stale events)
- Automatic bash restart when processes exit (unless replaced manually)
- Clean process lifecycle management
- Proper terminal size updates (TIOCSWINSZ)
- Non-canonical mode for readline support

### Rendering
- tcell-based terminal UI
- Efficient dirty-region rendering
- Proper cursor positioning
- Window border drawing (│ and ─)
- Status bar rendering with window info
- Completion candidate display
- Selection overlay rendering

## Implementation Details

### Architecture
- **Language**: Go 1.24
- **Terminal UI**: tcell v2
- **PTY**: creack/pty
- **Key bindings**: cbind

### Key Algorithms
- Binary tree layout for window splits
- Generation counter for PTY event disambiguation
- Common prefix finding for Tab completion
- Word boundary detection for double-click selection
- ANSI escape sequence parser
- Vi state detection via heuristics

### Performance
- Supports 8 simultaneous windows
- Efficient goroutine management
- Minimal memory overhead
- Fast rendering with tcell
- No significant input lag

## Known Limitations

1. Only vi/vim editor supported (no nano, emacs, etc.)
2. Only Go language build/run support
3. Maximum 8 windows
4. No configuration file
5. No session persistence
6. No syntax highlighting
7. No plugin system
8. Linux/WSL only (no Windows native, macOS)

## Future Enhancement Ideas

1. Support for more languages (Python, Rust, C, etc.)
2. Configuration file for customization
3. Session save/restore
4. Syntax highlighting
5. Multiple clipboards/registers
6. Search in files
7. File tree browser
8. Git integration
9. Debugger integration
10. Custom key bindings
