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
	screen      tcell.Screen
	wm          *window.Manager
	keyHandler  *keyboard.Handler
	renderer    *ui.Renderer
	quitChan    chan struct{}
	ptyEvents   chan window.PTYEvent
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

	// Create key handler
	keyHandler := keyboard.NewHandler(wm, quitChan)

	// Create renderer
	renderer := ui.NewRenderer(screen, theme)

	app := &App{
		screen:     screen,
		wm:         wm,
		keyHandler: keyHandler,
		renderer:   renderer,
		quitChan:   quitChan,
		ptyEvents:  make(chan window.PTYEvent, 100),
	}

	return app, nil
}

// Run starts the main event loop
func (a *App) Run() error {
	defer a.cleanup()

	// Start PTY readers for all windows
	a.startPTYReaders()

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

// handleScreenEvent handles screen events (keyboard, resize)
func (a *App) handleScreenEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyCtrlC {
			// Emergency quit
			close(a.quitChan)
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

	case *tcell.EventResize:
		width, height := a.screen.Size()
		a.wm.ResizeAll(width, height)
		a.screen.Sync()
	}
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

// startPTYReaders starts PTY readers for all windows
func (a *App) startPTYReaders() {
	for _, win := range a.wm.GetWindows() {
		go win.ReadPTY(a.ptyEvents)
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
