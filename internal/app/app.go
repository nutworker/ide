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
	screen         tcell.Screen
	wm             *window.Manager
	keyHandler     *keyboard.Handler
	renderer       *ui.Renderer
	quitChan       chan struct{}
	ptyEvents      chan window.PTYEvent
	activeReaders  map[int]bool // Track which windows have active readers
	selection      *ui.Selection
	clipboard      string
	mousePressed   bool
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

		// Handle Ctrl-Shift-V (paste)
		if ev.Key() == tcell.KeyCtrlV && ev.Modifiers()&tcell.ModShift != 0 {
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
			// Start new selection
			a.selection.Start(targetWin.ID, x, y)
			a.mousePressed = true
		} else {
			// Update selection (dragging)
			a.selection.Update(x, y)
		}
		return
	}

	// Handle mouse button release
	if a.mousePressed && buttons == tcell.ButtonNone {
		a.mousePressed = false
		// Selection is now complete
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

	// Clear selection after copy
	a.selection.Clear()
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
	default:
		return []byte{}
	}
}
