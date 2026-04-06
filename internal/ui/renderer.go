package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/nutworker/ide/internal/window"
)

// Renderer handles rendering the IDE to the screen
type Renderer struct {
	screen    tcell.Screen
	theme     *Theme
	statusBar *StatusBar
}

// NewRenderer creates a new renderer
func NewRenderer(screen tcell.Screen, theme *Theme) *Renderer {
	return &Renderer{
		screen:    screen,
		theme:     theme,
		statusBar: NewStatusBar(theme),
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
	// Draw window number in top-left corner
	r.drawWindowNumber(win, windowNum, isActive)

	// Draw window content
	r.drawWindowContent(win)

	// Draw status bar if this is a vi window
	if win.State.IsVi {
		r.statusBar.Render(r.screen, win)
	}

	// Draw border (simple line separators)
	r.drawWindowBorder(win)
}

// drawWindowNumber draws the window number
func (r *Renderer) drawWindowNumber(win *window.Window, num int, isActive bool) {
	numStr := fmt.Sprintf(" %d ", num)
	style := r.theme.WindowNum()

	if !isActive {
		// Dim the style for inactive windows
		style = style.Dim(true)
	}

	x := win.Rect.X
	y := win.Rect.Y

	for i, ch := range numStr {
		if x+i < win.Rect.X+win.Rect.Width {
			r.screen.SetContent(x+i, y, ch, nil, style)
		}
	}
}

// drawWindowContent draws the window's terminal content
func (r *Renderer) drawWindowContent(win *window.Window) {
	lines := win.GetLines()

	startY := win.Rect.Y + 1 // Start below window number
	maxHeight := win.Rect.Height - 1

	if win.State.IsVi {
		maxHeight-- // Reserve space for status bar
	}

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

			r.screen.SetContent(x+colIdx, y, cell.Rune, nil, cell.Style)
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

// drawWindowBorder draws simple borders between windows
func (r *Renderer) drawWindowBorder(win *window.Window) {
	// Draw a simple line at the right edge if not at screen edge
	// This is a minimal border - we can enhance later
	style := r.theme.Default().Reverse(true)

	// Right edge
	screenW, screenH := r.screen.Size()
	if win.Rect.X+win.Rect.Width < screenW {
		x := win.Rect.X + win.Rect.Width - 1
		for y := win.Rect.Y; y < win.Rect.Y+win.Rect.Height; y++ {
			r.screen.SetContent(x, y, '│', nil, style)
		}
	}

	// Bottom edge
	y := win.Rect.Y + win.Rect.Height - 1
	if y < screenH-1 {
		for x := win.Rect.X; x < win.Rect.X+win.Rect.Width; x++ {
			if x < screenW {
				r.screen.SetContent(x, y, '─', nil, style)
			}
		}
	}
}
