package ui

import "github.com/gdamore/tcell/v2"

// StyleMap represents a map of cell positions to styles
type StyleMap map[int]map[int]tcell.Style // [row][col] -> style

// Highlighter is an interface for syntax highlighting
type Highlighter interface {
	// GetStyles returns syntax highlighting styles for the given source
	GetStyles(filename string, source []byte) StyleMap
}
