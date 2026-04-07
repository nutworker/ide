package pty

import (
	"github.com/gdamore/tcell/v2"
)

// Terminal represents a terminal emulator with a 2D screen buffer
type Terminal struct {
	width      int
	height     int
	buffer     [][]Cell // 2D screen buffer
	cursorRow  int
	cursorCol  int
	savedRow   int
	savedCol   int
	style      tcell.Style
	parser     *ANSIParser
}

// NewTerminal creates a new terminal emulator
func NewTerminal(width, height int, defaultStyle tcell.Style) *Terminal {
	t := &Terminal{
		width:  width,
		height: height,
		buffer: make([][]Cell, height),
		style:  defaultStyle,
		parser: NewANSIParser(defaultStyle),
	}

	// Initialize buffer with empty cells
	for i := 0; i < height; i++ {
		t.buffer[i] = make([]Cell, width)
		for j := 0; j < width; j++ {
			t.buffer[i][j] = Cell{Rune: ' ', Style: defaultStyle}
		}
	}

	return t
}

// Write processes output data and updates the screen buffer
func (t *Terminal) Write(data []byte) {
	for i := 0; i < len(data); i++ {
		b := data[i]

		// Handle escape sequences
		if b == 0x1b && i+1 < len(data) {
			if data[i+1] == '[' {
				// CSI sequence
				seqLen := t.handleCSI(data[i:])
				i += seqLen - 1
				continue
			} else if data[i+1] == ']' {
				// OSC sequence (terminal title, etc.)
				seqLen := t.handleOSC(data[i:])
				i += seqLen - 1
				continue
			}
		}

		// Handle control characters
		switch b {
		case '\r':
			// Carriage return
			t.cursorCol = 0
		case '\n':
			// Line feed
			t.cursorRow++
			if t.cursorRow >= t.height {
				t.scrollUp()
				t.cursorRow = t.height - 1
			}
		case '\b':
			// Backspace
			if t.cursorCol > 0 {
				t.cursorCol--
			}
		case '\t':
			// Tab - move to next tab stop (every 8 columns)
			t.cursorCol = ((t.cursorCol / 8) + 1) * 8
			if t.cursorCol >= t.width {
				t.cursorCol = t.width - 1
			}
		case 0x07:
			// Bell - ignore
		default:
			// Printable character
			if b >= 32 || b == 0x0a || b == 0x0d {
				t.putChar(rune(b))
			}
		}
	}
}

// putChar places a character at the current cursor position
func (t *Terminal) putChar(r rune) {
	if t.cursorRow >= 0 && t.cursorRow < t.height &&
		t.cursorCol >= 0 && t.cursorCol < t.width {
		t.buffer[t.cursorRow][t.cursorCol] = Cell{Rune: r, Style: t.style}
		t.cursorCol++

		// Wrap to next line if needed
		if t.cursorCol >= t.width {
			t.cursorCol = 0
			t.cursorRow++
			if t.cursorRow >= t.height {
				t.scrollUp()
				t.cursorRow = t.height - 1
			}
		}
	}
}

// handleCSI handles CSI (Control Sequence Introducer) escape sequences
func (t *Terminal) handleCSI(data []byte) int {
	// Find the end of the CSI sequence
	end := 2 // Start after ESC[
	for end < len(data) && data[end] >= 0x20 && data[end] <= 0x3f {
		end++
	}
	if end >= len(data) {
		return len(data)
	}

	cmd := data[end]
	end++

	// Parse parameters
	params := t.parseParams(data[2 : end-1])

	switch cmd {
	case 'A': // Cursor up
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		t.cursorRow -= n
		if t.cursorRow < 0 {
			t.cursorRow = 0
		}

	case 'B': // Cursor down
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		t.cursorRow += n
		if t.cursorRow >= t.height {
			t.cursorRow = t.height - 1
		}

	case 'C': // Cursor forward
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		t.cursorCol += n
		if t.cursorCol >= t.width {
			t.cursorCol = t.width - 1
		}

	case 'D': // Cursor backward
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		t.cursorCol -= n
		if t.cursorCol < 0 {
			t.cursorCol = 0
		}

	case 'H', 'f': // Cursor position
		row := 1
		col := 1
		if len(params) >= 1 {
			row = params[0]
		}
		if len(params) >= 2 {
			col = params[1]
		}
		t.cursorRow = row - 1
		t.cursorCol = col - 1
		if t.cursorRow < 0 {
			t.cursorRow = 0
		}
		if t.cursorRow >= t.height {
			t.cursorRow = t.height - 1
		}
		if t.cursorCol < 0 {
			t.cursorCol = 0
		}
		if t.cursorCol >= t.width {
			t.cursorCol = t.width - 1
		}

	case 'J': // Erase in display
		n := 0
		if len(params) > 0 {
			n = params[0]
		}
		t.eraseDisplay(n)

	case 'K': // Erase in line
		n := 0
		if len(params) > 0 {
			n = params[0]
		}
		t.eraseLine(n)

	case 'm': // Set graphics mode (colors, etc.)
		// Ignore for now - we use default style

	case 's': // Save cursor position
		t.savedRow = t.cursorRow
		t.savedCol = t.cursorCol

	case 'u': // Restore cursor position
		t.cursorRow = t.savedRow
		t.cursorCol = t.savedCol
	}

	return end
}

// handleOSC handles OSC (Operating System Command) sequences
func (t *Terminal) handleOSC(data []byte) int {
	// Find the terminator (BEL or ESC\)
	end := 2
	for end < len(data) {
		if data[end] == 0x07 {
			return end + 1
		}
		if data[end] == 0x1b && end+1 < len(data) && data[end+1] == '\\' {
			return end + 2
		}
		end++
	}
	return len(data)
}

// parseParams parses CSI parameters
func (t *Terminal) parseParams(data []byte) []int {
	var params []int
	current := 0
	hasDigit := false

	for _, b := range data {
		if b >= '0' && b <= '9' {
			current = current*10 + int(b-'0')
			hasDigit = true
		} else if b == ';' {
			if hasDigit {
				params = append(params, current)
			} else {
				params = append(params, 0)
			}
			current = 0
			hasDigit = false
		}
	}

	if hasDigit {
		params = append(params, current)
	}

	return params
}

// eraseDisplay clears parts of the display
func (t *Terminal) eraseDisplay(mode int) {
	switch mode {
	case 0: // Clear from cursor to end of screen
		t.eraseLine(0)
		for row := t.cursorRow + 1; row < t.height; row++ {
			for col := 0; col < t.width; col++ {
				t.buffer[row][col] = Cell{Rune: ' ', Style: t.style}
			}
		}
	case 1: // Clear from cursor to beginning of screen
		t.eraseLine(1)
		for row := 0; row < t.cursorRow; row++ {
			for col := 0; col < t.width; col++ {
				t.buffer[row][col] = Cell{Rune: ' ', Style: t.style}
			}
		}
	case 2: // Clear entire screen
		for row := 0; row < t.height; row++ {
			for col := 0; col < t.width; col++ {
				t.buffer[row][col] = Cell{Rune: ' ', Style: t.style}
			}
		}
	}
}

// eraseLine clears parts of the current line
func (t *Terminal) eraseLine(mode int) {
	if t.cursorRow < 0 || t.cursorRow >= t.height {
		return
	}

	switch mode {
	case 0: // Clear from cursor to end of line
		for col := t.cursorCol; col < t.width; col++ {
			t.buffer[t.cursorRow][col] = Cell{Rune: ' ', Style: t.style}
		}
	case 1: // Clear from cursor to beginning of line
		for col := 0; col <= t.cursorCol && col < t.width; col++ {
			t.buffer[t.cursorRow][col] = Cell{Rune: ' ', Style: t.style}
		}
	case 2: // Clear entire line
		for col := 0; col < t.width; col++ {
			t.buffer[t.cursorRow][col] = Cell{Rune: ' ', Style: t.style}
		}
	}
}

// scrollUp scrolls the screen buffer up by one line
func (t *Terminal) scrollUp() {
	// Move all lines up
	for i := 0; i < t.height-1; i++ {
		t.buffer[i] = t.buffer[i+1]
	}

	// Clear the last line
	t.buffer[t.height-1] = make([]Cell, t.width)
	for j := 0; j < t.width; j++ {
		t.buffer[t.height-1][j] = Cell{Rune: ' ', Style: t.style}
	}
}

// GetLines returns the screen buffer as lines
func (t *Terminal) GetLines() []Line {
	lines := make([]Line, t.height)
	for i := 0; i < t.height; i++ {
		lines[i] = t.buffer[i]
	}
	return lines
}

// Resize resizes the terminal buffer
func (t *Terminal) Resize(width, height int) {
	newBuffer := make([][]Cell, height)
	for i := 0; i < height; i++ {
		newBuffer[i] = make([]Cell, width)
		for j := 0; j < width; j++ {
			newBuffer[i][j] = Cell{Rune: ' ', Style: t.style}
		}
	}

	// Copy old content
	copyRows := t.height
	if height < copyRows {
		copyRows = height
	}
	for i := 0; i < copyRows; i++ {
		copyCols := t.width
		if width < copyCols {
			copyCols = width
		}
		for j := 0; j < copyCols; j++ {
			newBuffer[i][j] = t.buffer[i][j]
		}
	}

	t.buffer = newBuffer
	t.width = width
	t.height = height

	// Adjust cursor position
	if t.cursorRow >= height {
		t.cursorRow = height - 1
	}
	if t.cursorCol >= width {
		t.cursorCol = width - 1
	}
}

// GetCursorPosition returns the current cursor position
func (t *Terminal) GetCursorPosition() (row, col int) {
	return t.cursorRow, t.cursorCol
}
