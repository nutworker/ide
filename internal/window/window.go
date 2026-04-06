package window

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	ptyparser "github.com/nutworker/ide/internal/pty"
)

// Window represents a single window in the IDE
type Window struct {
	ID          int
	PTY         *os.File
	Cmd         *exec.Cmd
	State       *WindowState
	Rect        Rect
	ProcessType ProcessType
	SourceFile  string // For build/run output windows

	// Terminal buffer - using simple line-based approach
	rawBuffer   []byte // Raw output buffer
	mutex       sync.RWMutex
	parser      *ptyparser.ANSIParser
	scrollback  int // How many lines scrolled back
	maxBytes    int // Maximum buffer size
}

// NewWindow creates a new window
func NewWindow(id int, rect Rect, defaultStyle tcell.Style) *Window {
	return &Window{
		ID:          id,
		Rect:        rect,
		State:       NewWindowState(),
		ProcessType: ProcessShell,
		rawBuffer:   make([]byte, 0, 65536),
		parser:      ptyparser.NewANSIParser(defaultStyle),
		maxBytes:    1048576, // 1MB buffer
	}
}

// StartPTY starts a PTY with the given command
func (w *Window) StartPTY(command string, args ...string) error {
	// For bash, create a custom rcfile to set up our prompt cleanly
	if command == "/bin/bash" || command == "bash" {
		// Create a temporary rcfile
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
		tmpfile.Close()

		// Start bash with our custom rcfile
		args = append([]string{"--rcfile", tmpfile.Name()}, args...)

		// Clean up the temp file after a delay (bash will have read it by then)
		go func() {
			time.Sleep(2 * time.Second)
			os.Remove(tmpfile.Name())
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

	err := pty.Setsize(w.PTY, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})

	return err
}

// PTYEvent represents output from a PTY
type PTYEvent struct {
	WindowID int
	Data     []byte
	Err      error
}

// ReadPTY reads from the PTY and sends events to the channel
func (w *Window) ReadPTY(events chan<- PTYEvent) {
	if w.PTY == nil {
		return
	}

	reader := bufio.NewReader(w.PTY)
	buf := make([]byte, 4096)

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			// Send event
			events <- PTYEvent{
				WindowID: w.ID,
				Data:     data,
			}

			// Process the data internally
			w.processOutput(data)
		}

		if err != nil {
			if err != io.EOF {
				events <- PTYEvent{
					WindowID: w.ID,
					Err:      err,
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

	// Append data to raw buffer
	w.rawBuffer = append(w.rawBuffer, data...)

	// Limit buffer size
	if len(w.rawBuffer) > w.maxBytes {
		// Keep only the last maxBytes/2 to avoid frequent trimming
		w.rawBuffer = w.rawBuffer[len(w.rawBuffer)-w.maxBytes/2:]
	}
}

// GetLines returns the visible lines for rendering
func (w *Window) GetLines() []ptyparser.Line {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	// Split buffer into lines
	lines := w.splitIntoLines(w.rawBuffer)

	// Calculate visible height (always reserve space for status bar)
	height := w.Rect.Height - 1

	// If we have fewer lines than height, return all lines
	if len(lines) <= height {
		return lines
	}

	// Get last N lines (scrollback-aware)
	start := len(lines) - height - w.scrollback
	if start < 0 {
		start = 0
	}

	end := len(lines) - w.scrollback
	if end > len(lines) {
		end = len(lines)
	}
	if end < 0 {
		end = 0
	}

	if start >= end {
		return []ptyparser.Line{}
	}

	return lines[start:end]
}

// splitIntoLines splits raw buffer into lines, handling wrapping
func (w *Window) splitIntoLines(data []byte) []ptyparser.Line {
	var lines []ptyparser.Line
	var currentLine []byte

	for i := 0; i < len(data); i++ {
		b := data[i]

		switch b {
		case '\n':
			// Add current line
			line := w.parser.ParseLine(currentLine)
			lines = append(lines, line)
			currentLine = nil

		case '\r':
			// Carriage return - typically followed by \n, ignore

		default:
			currentLine = append(currentLine, b)
		}
	}

	// Add remaining line if any
	if len(currentLine) > 0 {
		line := w.parser.ParseLine(currentLine)
		lines = append(lines, line)
	}

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

	lines := w.splitIntoLines(w.rawBuffer)
	if len(lines) == 0 {
		return ""
	}

	// Get the last line (most recent)
	lineIdx := len(lines) - w.scrollback - 1
	if lineIdx < 0 || lineIdx >= len(lines) {
		return ""
	}

	line := lines[lineIdx]
	result := ""
	for _, cell := range line {
		result += string(cell.Rune)
	}

	return result
}

// ScrollUp scrolls the window content up by the given number of lines
func (w *Window) ScrollUp(lines int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.scrollback += lines

	// Limit scrollback to available lines
	allLines := w.splitIntoLines(w.rawBuffer)
	maxScrollback := len(allLines) - w.Rect.Height + 1
	if maxScrollback < 0 {
		maxScrollback = 0
	}

	if w.scrollback > maxScrollback {
		w.scrollback = maxScrollback
	}
}

// ScrollDown scrolls the window content down by the given number of lines
func (w *Window) ScrollDown(lines int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.scrollback -= lines
	if w.scrollback < 0 {
		w.scrollback = 0
	}
}

// IsPointInside checks if a point (x, y) is inside this window
func (w *Window) IsPointInside(x, y int) bool {
	return x >= w.Rect.X && x < w.Rect.X+w.Rect.Width &&
		y >= w.Rect.Y && y < w.Rect.Y+w.Rect.Height
}
