package ui

import (
	"github.com/nutworker/ide/internal/window"
)

// Selection represents a text selection in a window
type Selection struct {
	WindowID  int
	StartX    int
	StartY    int
	EndX      int
	EndY      int
	Active    bool
}

// NewSelection creates a new selection
func NewSelection() *Selection {
	return &Selection{
		Active: false,
	}
}

// Start begins a selection at the given position
func (s *Selection) Start(windowID, x, y int) {
	s.WindowID = windowID
	s.StartX = x
	s.StartY = y
	s.EndX = x
	s.EndY = y
	s.Active = true
}

// Update updates the selection end position
func (s *Selection) Update(x, y int) {
	if s.Active {
		s.EndX = x
		s.EndY = y
	}
}

// Clear clears the selection
func (s *Selection) Clear() {
	s.Active = false
}

// IsInSelection checks if a position is within the selection
func (s *Selection) IsInSelection(x, y int) bool {
	if !s.Active {
		return false
	}

	// Normalize start/end (handle both directions)
	minY, maxY := s.StartY, s.EndY
	minX, maxX := s.StartX, s.EndX

	if minY > maxY {
		minY, maxY = maxY, minY
		minX, maxX = maxX, minX
	}

	// Check if position is in selection
	if y < minY || y > maxY {
		return false
	}

	if y == minY && y == maxY {
		// Single line selection
		if minX > maxX {
			minX, maxX = maxX, minX
		}
		return x >= minX && x <= maxX
	}

	if y == minY {
		return x >= minX
	}

	if y == maxY {
		return x <= maxX
	}

	return true // Middle lines are fully selected
}

// GetSelectedText extracts the selected text from a window
func (s *Selection) GetSelectedText(win *window.Window) string {
	if !s.Active {
		return ""
	}

	lines := win.GetLines()
	if len(lines) == 0 {
		return ""
	}

	// Calculate which lines are selected
	startY := s.StartY - win.Rect.Y
	endY := s.EndY - win.Rect.Y
	startX := s.StartX - win.Rect.X
	endX := s.EndX - win.Rect.X

	// Normalize
	if startY > endY || (startY == endY && startX > endX) {
		startY, endY = endY, startY
		startX, endX = endX, startX
	}

	if startY < 0 {
		startY = 0
	}
	if endY >= len(lines) {
		endY = len(lines) - 1
	}

	var result string

	for lineIdx := startY; lineIdx <= endY; lineIdx++ {
		if lineIdx >= len(lines) {
			break
		}

		line := lines[lineIdx]
		lineStart := 0
		lineEnd := len(line)

		if lineIdx == startY {
			lineStart = startX
		}
		if lineIdx == endY {
			lineEnd = endX + 1
		}

		if lineStart < 0 {
			lineStart = 0
		}
		if lineEnd > len(line) {
			lineEnd = len(line)
		}

		for i := lineStart; i < lineEnd; i++ {
			if i < len(line) {
				result += string(line[i].Rune)
			}
		}

		if lineIdx < endY {
			result += "\n"
		}
	}

	return result
}
