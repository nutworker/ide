package window

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

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

	// Terminal buffer
	lines       []ptyparser.Line
	mutex       sync.RWMutex
	parser      *ptyparser.ANSIParser
	scrollback  int // How many lines scrolled back
	maxLines    int // Maximum scrollback buffer
}

// NewWindow creates a new window
func NewWindow(id int, rect Rect, defaultStyle tcell.Style) *Window {
	return &Window{
		ID:          id,
		Rect:        rect,
		State:       NewWindowState(),
		ProcessType: ProcessShell,
		lines:       make([]ptyparser.Line, 0, 100),
		parser:      ptyparser.NewANSIParser(defaultStyle),
		maxLines:    10000,
	}
}

// StartPTY starts a PTY with the given command
func (w *Window) StartPTY(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

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

	// Reserve one line for status bar if in vi mode
	if w.State.IsVi {
		rows--
	}

	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}

	return pty.Setsize(w.PTY, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
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

	// Parse and add lines to buffer
	// For now, just add raw lines - we'll improve this later
	line := w.parser.ParseLine(data)
	if len(line) > 0 {
		w.lines = append(w.lines, line)

		// Limit scrollback
		if len(w.lines) > w.maxLines {
			w.lines = w.lines[len(w.lines)-w.maxLines:]
		}
	}
}

// GetLines returns the visible lines for rendering
func (w *Window) GetLines() []ptyparser.Line {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	height := w.Rect.Height
	if w.State.IsVi {
		height-- // Reserve space for status bar
	}

	start := len(w.lines) - height - w.scrollback
	if start < 0 {
		start = 0
	}

	end := len(w.lines) - w.scrollback
	if end > len(w.lines) {
		end = len(w.lines)
	}

	if start >= end {
		return []ptyparser.Line{}
	}

	return w.lines[start:end]
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

	if len(w.lines) == 0 {
		return ""
	}

	// Get the line at current scroll position
	lineIdx := len(w.lines) - w.scrollback - 1
	if lineIdx < 0 || lineIdx >= len(w.lines) {
		return ""
	}

	line := w.lines[lineIdx]
	result := ""
	for _, cell := range line {
		result += string(cell.Rune)
	}

	return result
}
