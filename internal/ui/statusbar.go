package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/nutworker/ide/internal/window"
)

// StatusBar renders the status bar for vi windows
type StatusBar struct {
	theme *Theme
}

// NewStatusBar creates a new status bar
func NewStatusBar(theme *Theme) *StatusBar {
	return &StatusBar{
		theme: theme,
	}
}

// Render renders the status bar for a window
func (sb *StatusBar) Render(screen tcell.Screen, win *window.Window) {
	if !win.State.IsVi {
		return
	}

	// Status bar at bottom of window
	y := win.Rect.Y + win.Rect.Height - 1
	x := win.Rect.X

	// Format: "filename.go [12,45] -- INSERT --"
	status := fmt.Sprintf(" %s [%d,%d] -- %s -- ",
		win.State.Filename,
		win.State.CursorRow,
		win.State.CursorCol,
		win.State.ViMode)

	// Render with status bar style
	style := sb.theme.StatusBar()

	// Fill the entire line
	for i := 0; i < win.Rect.Width; i++ {
		if i < len(status) {
			screen.SetContent(x+i, y, rune(status[i]), nil, style)
		} else {
			screen.SetContent(x+i, y, ' ', nil, style)
		}
	}
}
