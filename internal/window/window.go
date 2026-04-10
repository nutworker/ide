package window

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	ptyparser "github.com/nutworker/ide/internal/pty"
	"golang.org/x/sys/unix"
)

// Window represents a single window in the IDE
type Window struct {
	ID          int
	PTY         *os.File
	Cmd         *exec.Cmd
	State       *WindowState
	Rect        Rect
	ProcessType ProcessType
	SourceFile  string   // For build/run output windows
	Generation  int      // PTY session generation (incremented on restart)
	FileStack   []string // Stack of previously opened files (for Alt-F history)

	// Terminal emulator
	terminal   *ptyparser.Terminal
	mutex      sync.RWMutex
	parser     *ptyparser.ANSIParser
	scrollback int // How many lines scrolled back
}

// NewWindow creates a new window
func NewWindow(id int, rect Rect, defaultStyle tcell.Style) *Window {
	// Reserve space for status bar
	termHeight := rect.Height - 1
	if termHeight < 1 {
		termHeight = 1
	}

	return &Window{
		ID:          id,
		Rect:        rect,
		State:       NewWindowState(),
		ProcessType: ProcessShell,
		terminal:    ptyparser.NewTerminal(rect.Width, termHeight, defaultStyle),
		parser:      ptyparser.NewANSIParser(defaultStyle),
	}
}

// StartPTY starts a PTY with the given command
func (w *Window) StartPTY(command string, args ...string) error {
	var bashRcFile string

	// For bash, create a custom rcfile to set up our prompt cleanly
	if (command == "/bin/bash" || command == "bash") && len(args) == 0 {
		rcContent := `# Source user's bashrc if it exists
if [ -f ~/.bashrc ]; then
    source ~/.bashrc
fi

# Set up IDE prompt function
_ide_prompt() {
    local dir="$PWD"
    dir="${dir/#$HOME/~}"
    if [ ${#dir} -gt 20 ]; then
        local first="${dir%%/*}"
        local last="${dir##*/}"
        if [[ "$dir" == ~* ]]; then
            first="~"
        fi
        if [ "$first" = "$last" ]; then
            dir="$first"
        else
            dir="$first/.../$last"
        fi
    fi
    PS1="$dir\$ "
}

PROMPT_COMMAND=_ide_prompt
clear
`
		tmpfile, err := os.CreateTemp("", "ide-bashrc-*")
		if err != nil {
			return fmt.Errorf("failed to create temp rcfile: %w", err)
		}

		if _, err := tmpfile.WriteString(rcContent); err != nil {
			tmpfile.Close()
			os.Remove(tmpfile.Name())
			return fmt.Errorf("failed to write temp rcfile: %w", err)
		}
		bashRcFile = tmpfile.Name()
		tmpfile.Close()

		// Use --rcfile to load our custom rc
		args = []string{"--rcfile", bashRcFile}

		// Clean up temp file after delay
		go func() {
			time.Sleep(2 * time.Second)
			os.Remove(bashRcFile)
		}()
	}

	cmd := exec.Command(command, args...)

	// Set environment variables
	env := os.Environ()
	env = append(env, "TERM=xterm-256color")
	cmd.Env = env

	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start PTY: %w", err)
	}

	// Set terminal to raw mode for readline support
	termios, err := unix.IoctlGetTermios(int(ptyFile.Fd()), unix.TCGETS)
	if err == nil {
		// Disable canonical mode - readline needs raw mode
		termios.Lflag &^= unix.ICANON
		// Keep echo on and signal handling
		termios.Lflag |= unix.ECHO | unix.ISIG
		// Enable CR-NL mapping for proper newlines
		termios.Iflag |= unix.ICRNL
		termios.Oflag |= unix.OPOST | unix.ONLCR
		// Set minimum characters to return immediately
		termios.Cc[unix.VMIN] = 1
		termios.Cc[unix.VTIME] = 0

		unix.IoctlSetTermios(int(ptyFile.Fd()), unix.TCSETS, termios)
	}

	w.PTY = ptyFile
	w.Cmd = cmd

	// Set initial PTY size
	if err := w.ResizePTY(); err != nil {
		return err
	}

	return nil
}
// ResizePTY resizes the PTY to match the window rectangle
func (w *Window) ResizePTY() error {
	if w.PTY == nil {
		return nil
	}

	rows := w.Rect.Height
	cols := w.Rect.Width

	// Always reserve one line for status bar (bottom)
	rows--

	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}

	// Resize terminal emulator buffer
	w.mutex.Lock()
	w.terminal.Resize(cols, rows)
	w.mutex.Unlock()

	// Resize PTY
	err := pty.Setsize(w.PTY, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})

	return err
}

// PTYEvent represents output from a PTY
type PTYEvent struct {
	WindowID   int
	Generation int
	Data       []byte
	Err        error
}

// ReadPTY reads from the PTY and sends events to the channel
func (w *Window) ReadPTY(events chan<- PTYEvent) {
	if w.PTY == nil {
		return
	}

	// Capture the generation at the start of this reader
	generation := w.Generation

	reader := bufio.NewReader(w.PTY)
	buf := make([]byte, 4096)

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			// Send event
			events <- PTYEvent{
				WindowID:   w.ID,
				Generation: generation,
				Data:       data,
			}

			// Process the data internally
			w.processOutput(data)
		}

		if err != nil {
			if err != io.EOF {
				events <- PTYEvent{
					WindowID:   w.ID,
					Generation: generation,
					Err:        err,
				}
			}
			return
		}
	}
}

// processOutput processes PTY output and updates window state
func (w *Window) processOutput(data []byte) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Check for cursor position
	if row, col, found := w.parser.ParseCursorPosition(data); found {
		w.State.CursorRow = row
		w.State.CursorCol = col
	}

	// Check for terminal title (indicates vi with filename)
	if title, found := w.parser.ParseTerminalTitle(data); found {
		if title != "" {
			w.State.Filename = title
			w.State.IsVi = true
		}
	}

	// Alternative vi detection: check if the last line looks like a vi status line
	// Vi status lines typically contain filename and line/column info
	// This is a heuristic approach since vi doesn't always set terminal title
	lines := w.terminal.GetLines()
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		lastLineText := ""
		for _, cell := range lastLine {
			lastLineText += string(cell.Rune)
		}

		// Detect if we've returned to shell (prompt with $ or other shell indicators)
		if w.State.IsVi {
			// If we see a shell prompt, we've exited vi
			if strings.Contains(lastLineText, "$ ") ||
			   strings.HasSuffix(strings.TrimSpace(lastLineText), "$") ||
			   strings.Contains(lastLineText, "# ") ||
			   strings.HasSuffix(strings.TrimSpace(lastLineText), "#") {
				// We're back in the shell, clear vi state
				w.State.IsVi = false
				w.State.Filename = ""
				w.State.ViMode = ViModeCommand
			}
		}

		// Check for vi status line patterns like "filename.txt" or "filename.txt [Modified]"
		// or line/column indicators
		if !w.State.IsVi {
			// Simple heuristic: if last line contains quotes and "lines" or has format indicators
			if (strings.Contains(lastLineText, "\"") && (strings.Contains(lastLineText, "lines") || strings.Contains(lastLineText, "L,") || strings.Contains(lastLineText, "C"))) {
				// Extract filename from status line (between quotes)
				if start := strings.Index(lastLineText, "\""); start >= 0 {
					if end := strings.Index(lastLineText[start+1:], "\""); end >= 0 {
						filename := lastLineText[start+1 : start+1+end]
						if filename != "" && !strings.Contains(filename, " ") {
							w.State.Filename = filename
							w.State.IsVi = true
						}
					}
				}
			}
		}
	}

	// Write data to terminal emulator
	w.terminal.Write(data)

	// Sync cursor position from terminal emulator
	// Vi doesn't always send cursor position reports, so we use the terminal's tracked position
	if w.State.IsVi {
		row, col := w.terminal.GetCursorPosition()
		// Terminal uses 0-based, vi status shows 1-based
		w.State.CursorRow = row + 1
		w.State.CursorCol = col + 1
	}
}

// GetLines returns the visible lines for rendering
func (w *Window) GetLines() []ptyparser.Line {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	// Get all lines from terminal emulator
	lines := w.terminal.GetLines()

	// For now, return all lines (scrollback can be added later if needed)
	return lines
}

// WriteToPTY writes data to the PTY
func (w *Window) WriteToPTY(data []byte) error {
	if w.PTY == nil {
		return fmt.Errorf("PTY not initialized")
	}

	_, err := w.PTY.Write(data)
	return err
}

// Close closes the window and cleans up resources
func (w *Window) Close() error {
	if w.Cmd != nil && w.Cmd.Process != nil {
		w.Cmd.Process.Kill()
	}

	if w.PTY != nil {
		return w.PTY.Close()
	}

	return nil
}

// GetCurrentLine returns the current line content (for error navigation)
func (w *Window) GetCurrentLine() string {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	lines := w.terminal.GetLines()
	if len(lines) == 0 {
		return ""
	}

	// For build output windows, use the selected line
	// Otherwise, use the last line (most recent)
	lineIdx := w.State.SelectedLine

	// Clamp to valid range
	if lineIdx < 0 {
		lineIdx = 0
	}
	if lineIdx >= len(lines) {
		lineIdx = len(lines) - 1
	}

	line := lines[lineIdx]
	result := ""
	for _, cell := range line {
		result += string(cell.Rune)
	}

	return result
}

// MoveSelectedLineUp moves the selected line up (for build output navigation)
func (w *Window) MoveSelectedLineUp() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.State.SelectedLine > 0 {
		w.State.SelectedLine--
	}
}

// MoveSelectedLineDown moves the selected line down (for build output navigation)
func (w *Window) MoveSelectedLineDown() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	lines := w.terminal.GetLines()
	if len(lines) == 0 {
		return
	}

	// Find the last non-empty line (search backwards)
	maxLine := len(lines) - 1
	for maxLine >= 0 {
		// Check if line has any non-space content
		hasContent := false
		for _, cell := range lines[maxLine] {
			if cell.Rune != ' ' && cell.Rune != 0 {
				hasContent = true
				break
			}
		}
		if hasContent {
			break
		}
		maxLine--
	}

	// If all lines are empty, stay at 0
	if maxLine < 0 {
		maxLine = 0
	}

	// Only move down if we haven't reached the last non-empty line
	if w.State.SelectedLine < maxLine {
		w.State.SelectedLine++
	}
}

// ScrollUp scrolls the window content up by the given number of lines
func (w *Window) ScrollUp(lines int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Scrollback disabled for now with terminal emulator
	// TODO: Add scrollback buffer to terminal emulator
}

// ScrollDown scrolls the window content down by the given number of lines
func (w *Window) ScrollDown(lines int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Scrollback disabled for now with terminal emulator
	// TODO: Add scrollback buffer to terminal emulator
}

// IsPointInside checks if a point (x, y) is inside this window
func (w *Window) IsPointInside(x, y int) bool {
	return x >= w.Rect.X && x < w.Rect.X+w.Rect.Width &&
		y >= w.Rect.Y && y < w.Rect.Y+w.Rect.Height
}

// GetCursorPosition returns the terminal cursor position (relative to window)
func (w *Window) GetCursorPosition() (row, col int) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.terminal.GetCursorPosition()
}
