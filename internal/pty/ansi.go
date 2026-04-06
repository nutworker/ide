package pty

import (
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

// ANSIParser parses ANSI escape sequences and converts them to tcell content
type ANSIParser struct {
	cursorPosRegex  *regexp.Regexp
	termTitleRegex  *regexp.Regexp
	currentStyle    tcell.Style
	defaultStyle    tcell.Style
}

// NewANSIParser creates a new ANSI parser
func NewANSIParser(defaultStyle tcell.Style) *ANSIParser {
	return &ANSIParser{
		// Matches cursor position report: ESC[row;colR
		cursorPosRegex: regexp.MustCompile(`\x1b\[(\d+);(\d+)R`),
		// Matches terminal title: ESC]0;title BEL or ESC]0;title ESC\
		termTitleRegex: regexp.MustCompile(`\x1b\]0;([^\x07\x1b]+)(?:\x07|\x1b\\)`),
		currentStyle:   defaultStyle,
		defaultStyle:   defaultStyle,
	}
}

// ParseCursorPosition extracts cursor position from ANSI output
func (p *ANSIParser) ParseCursorPosition(data []byte) (row, col int, found bool) {
	matches := p.cursorPosRegex.FindSubmatch(data)
	if matches == nil {
		return 0, 0, false
	}

	row, _ = strconv.Atoi(string(matches[1]))
	col, _ = strconv.Atoi(string(matches[2]))
	return row, col, true
}

// ParseTerminalTitle extracts terminal title from ANSI output
func (p *ANSIParser) ParseTerminalTitle(data []byte) (title string, found bool) {
	matches := p.termTitleRegex.FindSubmatch(data)
	if matches == nil {
		return "", false
	}
	return string(matches[1]), true
}

// Cell represents a terminal cell with character and style
type Cell struct {
	Rune  rune
	Style tcell.Style
}

// Line represents a line of terminal cells
type Line []Cell

// ParseLine converts raw bytes to a line of cells (simplified version)
// This is a basic implementation that strips most ANSI codes
func (p *ANSIParser) ParseLine(data []byte) Line {
	var cells Line

	// Strip common ANSI escape sequences for now
	// This is a simplified parser - a full implementation would handle all ANSI codes
	cleaned := p.stripANSI(data)

	for len(cleaned) > 0 {
		r, size := utf8.DecodeRune(cleaned)
		if r == utf8.RuneError {
			cleaned = cleaned[1:]
			continue
		}

		cells = append(cells, Cell{
			Rune:  r,
			Style: p.defaultStyle,
		})
		cleaned = cleaned[size:]
	}

	return cells
}

// stripANSI removes ANSI escape sequences (simplified)
func (p *ANSIParser) stripANSI(data []byte) []byte {
	// Remove color/style sequences (ESC[...m)
	colorRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	data = colorRegex.ReplaceAll(data, nil)

	// Remove terminal title sequences
	titleRegex := regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)
	data = titleRegex.ReplaceAll(data, nil)

	// Remove cursor movement and positioning sequences
	// ESC[nA (up), ESC[nB (down), ESC[nC (forward), ESC[nD (backward)
	// ESC[n;nH (position), ESC[nJ (clear), ESC[nK (clear line)
	cursorRegex := regexp.MustCompile(`\x1b\[[0-9;]*[ABCDHJKfhlm]`)
	data = cursorRegex.ReplaceAll(data, nil)

	// Remove other control sequences
	otherRegex := regexp.MustCompile(`\x1b\[[?0-9;]*[a-zA-Z]`)
	data = otherRegex.ReplaceAll(data, nil)

	// Don't trim - preserve content
	return data
}
