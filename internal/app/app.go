package app

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/nutworker/ide/internal/keyboard"
	"github.com/nutworker/ide/internal/ui"
	"github.com/nutworker/ide/internal/window"
)

// App represents the main IDE application
type App struct {
	screen            tcell.Screen
	wm                *window.Manager
	keyHandler        *keyboard.Handler
	renderer          *ui.Renderer
	quitChan          chan struct{}
	ptyEvents         chan window.PTYEvent
	activeReaders     map[int]bool // Track which windows have active readers
	selection         *ui.Selection
	clipboard         string
	mousePressed      bool
	lastClickTime     int64 // Unix timestamp in milliseconds
	lastClickX        int
	lastClickY        int
	clickCount        int
	wordSelectMode    bool // True when dragging after double-click
	wordAnchorStart   int  // Original word start position (screen coords)
	wordAnchorEnd     int  // Original word end position (screen coords)
	wordAnchorY       int  // Original word line position
	wordAnchorWinID   int  // Window ID for anchor
}

// New creates a new application
func New() (*App, error) {
	// Initialize screen
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize screen: %w", err)
	}

	// Enable mouse support
	screen.EnableMouse()

	// Enable paste mode to allow better terminal interaction
	screen.EnablePaste()

	// Create theme
	theme := ui.NewTheme()

	// Set default style
	screen.SetStyle(theme.Default())
	screen.Clear()

	// Get initial screen size
	width, height := screen.Size()

	// Create window manager
	wm := window.NewManager(8, theme.Default())

	// Create first window with bash
	rect := window.NewRect(0, 0, width, height)
	_, err = wm.CreateWindow(rect, "/bin/bash")
	if err != nil {
		screen.Fini()
		return nil, fmt.Errorf("failed to create initial window: %w", err)
	}

	// Create quit channel
	quitChan := make(chan struct{})

	// Create selection
	selection := ui.NewSelection()

	// Create key handler
	keyHandler := keyboard.NewHandler(wm, quitChan)

	// Create renderer with selection support
	renderer := ui.NewRenderer(screen, theme, selection)

	app := &App{
		screen:        screen,
		wm:            wm,
		keyHandler:    keyHandler,
		renderer:      renderer,
		quitChan:      quitChan,
		ptyEvents:     make(chan window.PTYEvent, 100),
		activeReaders: make(map[int]bool),
		selection:     selection,
		clipboard:     "",
		mousePressed:  false,
	}

	return app, nil
}

// Run starts the main event loop
func (a *App) Run() error {
	defer a.cleanup()

	// Initial render
	a.renderer.Render(a.wm)

	// Event channels
	screenEvents := make(chan tcell.Event)

	// Start screen event poller
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev != nil {
				screenEvents <- ev
			}
		}
	}()

	// Main event loop
	for {
		// Start readers for any new windows
		a.ensurePTYReaders()

		select {
		case <-a.quitChan:
			return nil

		case ev := <-screenEvents:
			a.handleScreenEvent(ev)

		case pev := <-a.ptyEvents:
			a.handlePTYEvent(pev)
		}

		// Render after each event
		a.renderer.Render(a.wm)
	}
}

// handleScreenEvent handles screen events (keyboard, resize, mouse)
func (a *App) handleScreenEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		// Handle Ctrl-Shift-C (copy)
		if ev.Key() == tcell.KeyCtrlC && ev.Modifiers()&tcell.ModShift != 0 {
			a.handleCopy()
			return
		}
		// Also check for Rune 'C' with Ctrl+Shift
		if ev.Key() == tcell.KeyRune && (ev.Rune() == 'C' || ev.Rune() == 'c') &&
			ev.Modifiers()&tcell.ModCtrl != 0 && ev.Modifiers()&tcell.ModShift != 0 {
			a.handleCopy()
			return
		}

		// Handle Ctrl-P (paste)
		if ev.Key() == tcell.KeyCtrlP {
			a.handlePaste()
			return
		}

		if ev.Key() == tcell.KeyCtrlC {
			// Regular Ctrl-C - pass to PTY (interrupt)
			activeWin := a.wm.GetActiveWindow()
			if activeWin != nil {
				activeWin.WriteToPTY([]byte{3}) // Ctrl-C
			}
			return
		}

		// Process through key handler
		processedEv := a.keyHandler.HandleEvent(ev)

		// If not consumed, pass to active window PTY
		if processedEv != nil {
			activeWin := a.wm.GetActiveWindow()
			if activeWin != nil {
				// Convert key event to bytes
				data := keyEventToBytes(processedEv)
				if len(data) > 0 {
					activeWin.WriteToPTY(data)
				}
			}
		}

	case *tcell.EventMouse:
		a.handleMouseEvent(ev)

	case *tcell.EventResize:
		width, height := a.screen.Size()
		a.wm.ResizeAll(width, height)
		a.screen.Sync()
	}
}

// handleMouseEvent handles mouse events (scrolling, selection)
func (a *App) handleMouseEvent(ev *tcell.EventMouse) {
	x, y := ev.Position()

	// Find which window the mouse is over
	targetWin := a.wm.GetWindowAtPosition(x, y)
	if targetWin == nil {
		return
	}

	buttons := ev.Buttons()

	// Handle mouse button press (start selection)
	if buttons&tcell.Button1 != 0 {
		if !a.mousePressed {
			// Detect double/triple click
			now := ev.When().UnixMilli()
			timeDiff := now - a.lastClickTime

			// Double-click threshold: 500ms
			if timeDiff < 500 && x == a.lastClickX && y == a.lastClickY {
				a.clickCount++
			} else {
				a.clickCount = 1
			}

			a.lastClickTime = now
			a.lastClickX = x
			a.lastClickY = y

			if a.clickCount == 2 {
				// Double-click: select word
				a.selectWord(targetWin, x, y)
				// Save anchor points for word-by-word dragging
				a.wordAnchorStart = a.selection.StartX
				a.wordAnchorEnd = a.selection.EndX
				a.wordAnchorY = a.selection.StartY
				a.wordAnchorWinID = targetWin.ID
				// Copy without clearing selection
				a.clipboard = a.selection.GetSelectedText(targetWin)
				// Enable word selection mode for dragging
				a.wordSelectMode = true
			} else if a.clickCount >= 3 {
				// Triple-click: select line
				a.selectLine(targetWin, x, y)
				// Copy without clearing selection
				a.clipboard = a.selection.GetSelectedText(targetWin)
				a.clickCount = 0 // Reset after triple-click
				a.wordSelectMode = false
			} else {
				// Single click: clear any existing selection and start new one
				a.selection.Clear()
				a.selection.Start(targetWin.ID, x, y)
				a.wordSelectMode = false
			}

			a.mousePressed = true
		} else {
			// Update selection (dragging)
			if a.wordSelectMode {
				// Extend selection word-by-word
				a.extendSelectionByWord(targetWin, x, y)
				// Update clipboard with extended selection
				a.clipboard = a.selection.GetSelectedText(targetWin)
			} else {
				// Normal character-by-character selection
				a.selection.Update(x, y)
			}
		}
		return
	}

	// Handle mouse button release
	if a.mousePressed && buttons == tcell.ButtonNone {
		a.mousePressed = false
		a.wordSelectMode = false // Reset word selection mode
		// Selection is now complete - keep it visible
		return
	}

	// Handle middle-click paste (X11 style)
	if buttons&tcell.Button2 != 0 {
		a.handlePaste()
		return
	}

	// Only allow scrolling in non-vi windows
	if targetWin.State.IsVi {
		return
	}

	// Handle mouse wheel scrolling
	switch buttons {
	case tcell.WheelUp:
		targetWin.ScrollUp(3)
	case tcell.WheelDown:
		targetWin.ScrollDown(3)
	}
}

// selectWord selects the word at the given position
func (a *App) selectWord(win *window.Window, x, y int) {
	// Convert screen coordinates to window-relative
	relX := x - win.Rect.X
	relY := y - win.Rect.Y

	lines := win.GetLines()
	if relY < 0 || relY >= len(lines) {
		return
	}

	line := lines[relY]
	if relX < 0 || relX >= len(line) {
		return
	}

	// Find word boundaries
	start := relX
	end := relX

	// Move start back to beginning of word
	for start > 0 && isWordChar(line[start-1].Rune) {
		start--
	}

	// Move end forward to end of word
	for end < len(line) && isWordChar(line[end].Rune) {
		end++
	}

	// Convert back to screen coordinates
	startX := win.Rect.X + start
	endX := win.Rect.X + end - 1
	screenY := win.Rect.Y + relY

	// Set selection
	a.selection.Start(win.ID, startX, screenY)
	a.selection.Update(endX, screenY)
}

// selectLine selects the entire line at the given position
func (a *App) selectLine(win *window.Window, x, y int) {
	// Convert screen coordinates to window-relative
	relY := y - win.Rect.Y

	lines := win.GetLines()
	if relY < 0 || relY >= len(lines) {
		return
	}

	// Select from start to end of line
	startX := win.Rect.X
	endX := win.Rect.X + win.Rect.Width - 1
	screenY := win.Rect.Y + relY

	a.selection.Start(win.ID, startX, screenY)
	a.selection.Update(endX, screenY)
}

// extendSelectionByWord extends the selection word-by-word when dragging
func (a *App) extendSelectionByWord(win *window.Window, x, y int) {
	// Convert screen coordinates to window-relative
	relX := x - win.Rect.X
	relY := y - win.Rect.Y

	lines := win.GetLines()
	if relY < 0 || relY >= len(lines) {
		return
	}

	line := lines[relY]
	if relX < 0 {
		relX = 0
	}
	if relX >= len(line) {
		relX = len(line) - 1
	}

	// Find word boundaries at the current mouse position
	wordStart := relX
	wordEnd := relX

	// If we're on a word character, extend to word boundaries
	if relX < len(line) && isWordChar(line[relX].Rune) {
		// Move start back to beginning of word
		for wordStart > 0 && isWordChar(line[wordStart-1].Rune) {
			wordStart--
		}
		// Move end forward to end of word
		for wordEnd < len(line) && isWordChar(line[wordEnd].Rune) {
			wordEnd++
		}
	} else {
		// On whitespace - find the nearest word
		// Look left first
		if relX > 0 {
			for wordStart > 0 && !isWordChar(line[wordStart].Rune) {
				wordStart--
			}
			if wordStart >= 0 && isWordChar(line[wordStart].Rune) {
				// Found word on left, get its boundaries
				wordEnd = wordStart + 1
				for wordStart > 0 && isWordChar(line[wordStart-1].Rune) {
					wordStart--
				}
				for wordEnd < len(line) && isWordChar(line[wordEnd].Rune) {
					wordEnd++
				}
			}
		}
	}

	screenY := win.Rect.Y + relY

	// Determine if we're dragging left or right from the original anchor
	if x < a.wordAnchorStart {
		// Dragging left - extend selection to the left
		a.selection.StartX = win.Rect.X + wordStart
		a.selection.StartY = screenY
		a.selection.EndX = a.wordAnchorEnd
		a.selection.EndY = a.wordAnchorY
	} else if x > a.wordAnchorEnd {
		// Dragging right - extend selection to the right
		a.selection.StartX = a.wordAnchorStart
		a.selection.StartY = a.wordAnchorY
		a.selection.EndX = win.Rect.X + wordEnd - 1
		a.selection.EndY = screenY
	} else {
		// Within the original word - keep original selection
		a.selection.StartX = a.wordAnchorStart
		a.selection.StartY = a.wordAnchorY
		a.selection.EndX = a.wordAnchorEnd
		a.selection.EndY = a.wordAnchorY
	}
}

// isWordChar returns true if the rune is part of a word
func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_' || r == '-' || r == '.'
}

// handleCopy copies the selected text to clipboard
func (a *App) handleCopy() {
	if !a.selection.Active {
		return
	}

	// Get the window with the selection
	win := a.wm.GetWindowByID(a.selection.WindowID)
	if win == nil {
		return
	}

	// Extract selected text
	a.clipboard = a.selection.GetSelectedText(win)

	// Keep selection visible after copy (don't clear it)
	// User can click elsewhere to clear it
}

// handlePaste pastes clipboard content to active window
func (a *App) handlePaste() {
	if a.clipboard == "" {
		return
	}

	activeWin := a.wm.GetActiveWindow()
	if activeWin == nil {
		return
	}

	// Don't paste in vi mode - it has its own paste
	if activeWin.State.IsVi {
		return
	}

	// Send clipboard content to PTY
	activeWin.WriteToPTY([]byte(a.clipboard))
}

// handlePTYEvent handles PTY output events
func (a *App) handlePTYEvent(pev window.PTYEvent) {
	if pev.Err != nil {
		// PTY closed or error - could handle window cleanup here
		return
	}

	// PTY output is already processed by the window itself
	// We just need to trigger a re-render, which happens in the main loop
}

// ensurePTYReaders ensures all windows have active PTY readers
func (a *App) ensurePTYReaders() {
	for _, win := range a.wm.GetWindows() {
		if !a.activeReaders[win.ID] {
			a.activeReaders[win.ID] = true
			go win.ReadPTY(a.ptyEvents)
		}
	}
}

// cleanup cleans up resources
func (a *App) cleanup() {
	a.wm.CloseAll()
	a.screen.Fini()
}

// keyEventToBytes converts a tcell key event to bytes for PTY input
func keyEventToBytes(ev *tcell.EventKey) []byte {
	// Handle Ctrl key combinations (Ctrl-A through Ctrl-Z)
	if ev.Key() == tcell.KeyRune && ev.Modifiers()&tcell.ModCtrl != 0 {
		r := ev.Rune()
		if r >= 'a' && r <= 'z' {
			// Ctrl-A is 1, Ctrl-B is 2, ..., Ctrl-Z is 26
			return []byte{byte(r - 'a' + 1)}
		}
		if r >= 'A' && r <= 'Z' {
			return []byte{byte(r - 'A' + 1)}
		}
	}

	switch ev.Key() {
	case tcell.KeyRune:
		return []byte(string(ev.Rune()))
	case tcell.KeyEnter:
		return []byte{'\r'}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		return []byte{127}
	case tcell.KeyTab:
		return []byte{'\t'}
	case tcell.KeyEscape:
		return []byte{27}
	case tcell.KeyUp:
		return []byte{27, '[', 'A'}
	case tcell.KeyDown:
		return []byte{27, '[', 'B'}
	case tcell.KeyRight:
		return []byte{27, '[', 'C'}
	case tcell.KeyLeft:
		return []byte{27, '[', 'D'}
	case tcell.KeyHome:
		return []byte{27, '[', 'H'}
	case tcell.KeyEnd:
		return []byte{27, '[', 'F'}
	case tcell.KeyPgUp:
		return []byte{27, '[', '5', '~'}
	case tcell.KeyPgDn:
		return []byte{27, '[', '6', '~'}
	case tcell.KeyDelete:
		return []byte{27, '[', '3', '~'}
	case tcell.KeyInsert:
		return []byte{27, '[', '2', '~'}
	case tcell.KeyCtrlA:
		return []byte{1}
	case tcell.KeyCtrlE:
		return []byte{5}
	case tcell.KeyCtrlU:
		return []byte{21}
	case tcell.KeyCtrlK:
		return []byte{11}
	case tcell.KeyCtrlW:
		return []byte{23}
	case tcell.KeyCtrlL:
		return []byte{12}
	default:
		return []byte{}
	}
}
