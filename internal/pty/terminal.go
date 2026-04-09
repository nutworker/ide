package pty

import (
	"github.com/gdamore/tcell/v2"
)

// Terminal represents a terminal emulator with a 2D screen buffer
type Terminal struct {
	width        int
	height       int
	buffer       [][]Cell // 2D screen buffer
	cursorRow    int
	cursorCol    int
	savedRow     int
	savedCol     int
	style        tcell.Style
	parser       *ANSIParser
	scrollTop    int // Top line of scrolling region (0-indexed)
	scrollBottom int // Bottom line of scrolling region (0-indexed)
}

// NewTerminal creates a new terminal emulator
func NewTerminal(width, height int, defaultStyle tcell.Style) *Terminal {
	t := &Terminal{
		width:        width,
		height:       height,
		buffer:       make([][]Cell, height),
		style:        defaultStyle,
		parser:       NewANSIParser(defaultStyle),
		scrollTop:    0,
		scrollBottom: height - 1,
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
			} else if data[i+1] == 'M' {
				// RI - Reverse Index (scroll down or move cursor up)
				if t.cursorRow > t.scrollTop {
					t.cursorRow--
				} else {
					// At top of scrolling region, scroll down
					t.scrollDownRegion()
				}
				i++ // Skip the 'M'
				continue
			} else if data[i+1] == 'D' {
				// IND - Index (scroll up or move cursor down)
				if t.cursorRow < t.scrollBottom {
					t.cursorRow++
				} else {
					// At bottom of scrolling region, scroll up
					t.scrollUpRegion()
					// Cursor stays at scrollBottom after scrolling
				}
				i++ // Skip the 'D'
				continue
			} else if data[i+1] == '7' {
				// Save cursor (DECSC)
				t.savedRow = t.cursorRow
				t.savedCol = t.cursorCol
				i++ // Skip the '7'
				continue
			} else if data[i+1] == '8' {
				// Restore cursor (DECRC)
				t.cursorRow = t.savedRow
				t.cursorCol = t.savedCol
				i++ // Skip the '8'
				continue
			}
		}

		// Handle control characters
		switch b {
		case '\r':
			// Carriage return
			t.cursorCol = 0
		case '\n':
			// Line feed - move cursor down, scroll if needed
			if t.cursorRow < t.scrollBottom {
				t.cursorRow++
			} else if t.cursorRow == t.scrollBottom {
				// At bottom of scrolling region - scroll up and stay at bottom
				t.scrollUpRegion()
				// Cursor stays at scrollBottom
			} else {
				// Beyond scrolling region - just move down
				t.cursorRow++
				if t.cursorRow >= t.height {
					t.cursorRow = t.height - 1
				}
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
			oldRow := t.cursorRow
			t.cursorRow++

			// Only scroll if we WERE within the scrolling region before wrapping
			if oldRow >= t.scrollTop && oldRow <= t.scrollBottom {
				// We were in the scrolling region, now check if we've gone past it
				if t.cursorRow > t.scrollBottom {
					t.scrollUpRegion()
					t.cursorRow = t.scrollBottom
				}
			} else {
				// We were outside the scrolling region (e.g., on vi's status line)
				// Just clamp to screen height, don't scroll
				if t.cursorRow >= t.height {
					t.cursorRow = t.height - 1
				}
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
		if len(params) >= 1 && params[0] > 0 {
			row = params[0]
		}
		if len(params) >= 2 && params[1] > 0 {
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

	case 'L': // Insert lines
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		t.insertLines(n)

	case 'M': // Delete lines
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		t.deleteLines(n)

	case 'S': // Scroll up
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		for i := 0; i < n; i++ {
			t.scrollUp()
		}

	case 'T': // Scroll down
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		for i := 0; i < n; i++ {
			t.scrollDown()
		}

	case 'r': // Set scrolling region
		// ESC[<top>;<bottom>r
		// When no params, reset to full screen
		oldBottom := t.scrollBottom

		if len(params) == 0 {
			t.scrollTop = 0
			t.scrollBottom = t.height - 1
		} else {
			top := 1
			bottom := t.height
			if len(params) >= 1 && params[0] > 0 {
				top = params[0]
			}
			if len(params) >= 2 && params[1] > 0 {
				bottom = params[1]
			}
			t.scrollTop = top - 1    // Convert to 0-indexed
			t.scrollBottom = bottom - 1 // Convert to 0-indexed
			if t.scrollTop < 0 {
				t.scrollTop = 0
			}
			if t.scrollBottom >= t.height {
				t.scrollBottom = t.height - 1
			}
			if t.scrollTop > t.scrollBottom {
				t.scrollTop = 0
				t.scrollBottom = t.height - 1
			}
		}

		// CRITICAL FIX: If the region shrunk (bottom decreased), clear the lines
		// that are now outside the region to prevent them from being copied in
		if t.scrollBottom < oldBottom {
			for row := t.scrollBottom + 1; row <= oldBottom && row < t.height; row++ {
				t.buffer[row] = make([]Cell, t.width)
				for j := 0; j < t.width; j++ {
					t.buffer[row][j] = Cell{Rune: ' ', Style: t.style}
				}
			}
		}

		// Move cursor to home position after setting scroll region
		t.cursorRow = 0
		t.cursorCol = 0

	case 'P': // Delete characters
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		t.deleteChars(n)

	case '@': // Insert characters
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		t.insertChars(n)

	case 'X': // Erase characters
		n := 1
		if len(params) > 0 {
			n = params[0]
		}
		t.eraseChars(n)

	case 'h': // Set mode
		// Ignore for now

	case 'l': // Reset mode
		// Ignore for now
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

// scrollUp scrolls the entire screen buffer up by one line
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

// scrollDown scrolls the entire screen buffer down by one line
func (t *Terminal) scrollDown() {
	// Move all lines down
	for i := t.height - 1; i > 0; i-- {
		t.buffer[i] = t.buffer[i-1]
	}

	// Clear the first line
	t.buffer[0] = make([]Cell, t.width)
	for j := 0; j < t.width; j++ {
		t.buffer[0][j] = Cell{Rune: ' ', Style: t.style}
	}
}

// scrollUpRegion scrolls the scrolling region up by one line
func (t *Terminal) scrollUpRegion() {
	if t.scrollTop > t.scrollBottom || t.scrollBottom >= t.height {
		return // Invalid region
	}

	// CRITICAL: Save ONLY the content within the scrolling region
	savedLines := make([][]Cell, t.scrollBottom - t.scrollTop + 1)
	for i := t.scrollTop; i <= t.scrollBottom; i++ {
		savedLines[i-t.scrollTop] = make([]Cell, t.width)
		copy(savedLines[i-t.scrollTop], t.buffer[i])
	}

	// Move lines up within the scrolling region ONLY
	// Copy from saved lines, ensuring we NEVER read outside the region
	for i := t.scrollTop; i <= t.scrollBottom; i++ {
		if i == t.scrollBottom {
			// Last line of region - clear it
			t.buffer[i] = make([]Cell, t.width)
			for j := 0; j < t.width; j++ {
				t.buffer[i][j] = Cell{Rune: ' ', Style: t.style}
			}
		} else {
			// Copy from the next line in saved data
			t.buffer[i] = make([]Cell, t.width)
			copy(t.buffer[i], savedLines[i-t.scrollTop+1])
		}
	}
}

// scrollDownRegion scrolls the scrolling region down by one line
func (t *Terminal) scrollDownRegion() {
	if t.scrollTop > t.scrollBottom || t.scrollBottom >= t.height {
		return // Invalid region
	}

	// CRITICAL: Save content from the region BEFORE we start moving it
	// This prevents any possibility of copying from outside the region
	savedLines := make([][]Cell, t.scrollBottom - t.scrollTop + 1)
	for i := t.scrollTop; i <= t.scrollBottom; i++ {
		savedLines[i-t.scrollTop] = make([]Cell, t.width)
		copy(savedLines[i-t.scrollTop], t.buffer[i])
	}

	// Now move lines down within the scrolling region ONLY
	// Copy from saved lines, not from buffer (to avoid any aliasing)
	for i := t.scrollTop; i <= t.scrollBottom; i++ {
		if i == t.scrollTop {
			// First line of region - clear it
			t.buffer[i] = make([]Cell, t.width)
			for j := 0; j < t.width; j++ {
				t.buffer[i][j] = Cell{Rune: ' ', Style: t.style}
			}
		} else {
			// Copy from the previous line in saved data
			t.buffer[i] = make([]Cell, t.width)
			copy(t.buffer[i], savedLines[i-t.scrollTop-1])
		}
	}
}

// insertLines inserts n blank lines at cursor position
func (t *Terminal) insertLines(n int) {
	if t.cursorRow < 0 || t.cursorRow >= t.height || n <= 0 {
		return
	}

	if n > t.height - t.cursorRow {
		n = t.height - t.cursorRow
	}

	// Shift lines down from cursor position
	for row := t.height - 1; row >= t.cursorRow+n; row-- {
		t.buffer[row] = t.buffer[row-n]
	}

	// Clear the inserted lines
	for i := 0; i < n; i++ {
		t.buffer[t.cursorRow+i] = make([]Cell, t.width)
		for j := 0; j < t.width; j++ {
			t.buffer[t.cursorRow+i][j] = Cell{Rune: ' ', Style: t.style}
		}
	}
}

// deleteLines deletes n lines at cursor position
func (t *Terminal) deleteLines(n int) {
	if t.cursorRow < 0 || t.cursorRow >= t.height || n <= 0 {
		return
	}

	if n > t.height - t.cursorRow {
		n = t.height - t.cursorRow
	}

	// Shift lines up from cursor position
	for row := t.cursorRow; row < t.height-n; row++ {
		t.buffer[row] = t.buffer[row+n]
	}

	// Clear the lines at the bottom
	for i := 0; i < n; i++ {
		row := t.height - n + i
		t.buffer[row] = make([]Cell, t.width)
		for j := 0; j < t.width; j++ {
			t.buffer[row][j] = Cell{Rune: ' ', Style: t.style}
		}
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

	// Reset scrolling region to full screen
	t.scrollTop = 0
	t.scrollBottom = height - 1

	// Adjust cursor position
	if t.cursorRow >= height {
		t.cursorRow = height - 1
	}
	if t.cursorCol >= width {
		t.cursorCol = width - 1
	}
}

// deleteChars deletes n characters at cursor position
func (t *Terminal) deleteChars(n int) {
	if t.cursorRow < 0 || t.cursorRow >= t.height || n <= 0 {
		return
	}

	line := t.buffer[t.cursorRow]
	// Shift characters left
	for col := t.cursorCol; col < t.width-n; col++ {
		if col+n < t.width {
			line[col] = line[col+n]
		}
	}
	// Clear the rightmost characters
	for col := t.width - n; col < t.width; col++ {
		if col >= 0 && col < t.width {
			line[col] = Cell{Rune: ' ', Style: t.style}
		}
	}
}

// insertChars inserts n blank characters at cursor position
func (t *Terminal) insertChars(n int) {
	if t.cursorRow < 0 || t.cursorRow >= t.height || n <= 0 {
		return
	}

	line := t.buffer[t.cursorRow]
	// Shift characters right
	for col := t.width - 1; col >= t.cursorCol+n; col-- {
		if col-n >= 0 {
			line[col] = line[col-n]
		}
	}
	// Clear the inserted characters
	for col := t.cursorCol; col < t.cursorCol+n && col < t.width; col++ {
		line[col] = Cell{Rune: ' ', Style: t.style}
	}
}

// eraseChars erases n characters at cursor position
func (t *Terminal) eraseChars(n int) {
	if t.cursorRow < 0 || t.cursorRow >= t.height || n <= 0 {
		return
	}

	line := t.buffer[t.cursorRow]
	for col := t.cursorCol; col < t.cursorCol+n && col < t.width; col++ {
		line[col] = Cell{Rune: ' ', Style: t.style}
	}
}

// GetCursorPosition returns the current cursor position
func (t *Terminal) GetCursorPosition() (row, col int) {
	return t.cursorRow, t.cursorCol
}
