package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/nutworker/ide/internal/window"
)

// Renderer handles rendering the IDE to the screen
type Renderer struct {
	screen    tcell.Screen
	theme     *Theme
	statusBar *StatusBar
	selection *Selection
}

// NewRenderer creates a new renderer
func NewRenderer(screen tcell.Screen, theme *Theme, selection *Selection) *Renderer {
	return &Renderer{
		screen:    screen,
		theme:     theme,
		statusBar: NewStatusBar(theme),
		selection: selection,
	}
}

// Render renders all windows and the UI
func (r *Renderer) Render(wm *window.Manager) {
	r.screen.Clear()

	windows := wm.GetWindows()
	activeWin := wm.GetActiveWindow()

	// Render each window
	for idx, win := range windows {
		r.renderWindow(win, idx+1, win == activeWin)
	}

	r.screen.Show()
}

// renderWindow renders a single window
func (r *Renderer) renderWindow(win *window.Window, windowNum int, isActive bool) {
	// Draw window content
	r.drawWindowContent(win)

	// Always draw status bar (with window number)
	r.statusBar.Render(r.screen, win, windowNum)

	// Draw borders between windows
	r.drawWindowBorder(win)
}

// drawWindowContent draws the window's terminal content
func (r *Renderer) drawWindowContent(win *window.Window) {
	lines := win.GetLines()

	startY := win.Rect.Y // Start from top of window
	maxHeight := win.Rect.Height - 1 // Reserve space for status bar (always present)

	for lineIdx, line := range lines {
		if lineIdx >= maxHeight {
			break
		}

		y := startY + lineIdx
		x := win.Rect.X

		for colIdx, cell := range line {
			if colIdx >= win.Rect.Width {
				break
			}

			style := cell.Style
			// Check if this position is in selection
			if r.selection != nil && r.selection.IsInSelection(x+colIdx, y) {
				// Highlight selected text with reverse video
				style = style.Reverse(true)
			}

			r.screen.SetContent(x+colIdx, y, cell.Rune, nil, style)
		}

		// Fill remaining space with default style
		for colIdx := len(line); colIdx < win.Rect.Width; colIdx++ {
			r.screen.SetContent(x+colIdx, y, ' ', nil, r.theme.Default())
		}
	}

	// Fill remaining lines with blank space
	for lineIdx := len(lines); lineIdx < maxHeight; lineIdx++ {
		y := startY + lineIdx
		x := win.Rect.X

		for colIdx := 0; colIdx < win.Rect.Width; colIdx++ {
			r.screen.SetContent(x+colIdx, y, ' ', nil, r.theme.Default())
		}
	}
}

// drawWindowBorder draws borders between windows
func (r *Renderer) drawWindowBorder(win *window.Window) {
	screenW, screenH := r.screen.Size()
	style := r.theme.Default().Reverse(true)

	// Draw right edge border (vertical line) at the rightmost column of this window
	rightX := win.Rect.X + win.Rect.Width - 1
	if rightX < screenW-1 { // Only if there's potentially another window to the right
		// Draw from top to bottom, excluding status bar
		for y := win.Rect.Y; y < win.Rect.Y+win.Rect.Height-1; y++ {
			r.screen.SetContent(rightX, y, '│', nil, style)
		}
	}

	// Draw bottom edge border (horizontal line) just below this window
	bottomY := win.Rect.Y + win.Rect.Height
	if bottomY < screenH { // Only if there's potentially another window below
		// Draw from left to right
		for x := win.Rect.X; x < win.Rect.X+win.Rect.Width; x++ {
			if x < screenW {
				r.screen.SetContent(x, bottomY, '─', nil, style)
			}
		}
	}
}
