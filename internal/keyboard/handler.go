package keyboard

import (
	"fmt"
	"time"

	"code.rocketnine.space/tslocum/cbind"
	"github.com/gdamore/tcell/v2"
	"github.com/nutworker/ide/internal/window"
)

// Handler handles keyboard input and ALT commands
type Handler struct {
	cbind       *cbind.Configuration
	wm          *window.Manager
	quitChan    chan struct{}
	goHandler   *GoModeHandler
}

// NewHandler creates a new keyboard handler
func NewHandler(wm *window.Manager, quitChan chan struct{}) *Handler {
	h := &Handler{
		cbind:    cbind.NewConfiguration(),
		wm:       wm,
		quitChan: quitChan,
	}
	h.goHandler = NewGoModeHandler(wm)
	h.setupBindings()
	return h
}

// setupBindings sets up all ALT key bindings
func (h *Handler) setupBindings() {
	// Window splitting
	h.cbind.Set("Alt+h", func(ev *tcell.EventKey) *tcell.EventKey {
		h.handleSplitHorizontal()
		return nil
	})

	h.cbind.Set("Alt+v", func(ev *tcell.EventKey) *tcell.EventKey {
		h.handleSplitVertical()
		return nil
	})

	// Window navigation
	h.cbind.Set("Alt+t", func(ev *tcell.EventKey) *tcell.EventKey {
		h.handleToggle()
		return nil
	})

	// Window number navigation (Alt+1 through Alt+8)
	for i := 1; i <= 8; i++ {
		num := i
		key := fmt.Sprintf("Alt+%d", num)
		h.cbind.Set(key, func(ev *tcell.EventKey) *tcell.EventKey {
			h.handleJumpToWindow(num)
			return nil
		})
	}

	// Go mode commands
	h.cbind.Set("Alt+b", func(ev *tcell.EventKey) *tcell.EventKey {
		h.handleBuild()
		return nil
	})

	h.cbind.Set("Alt+r", func(ev *tcell.EventKey) *tcell.EventKey {
		h.handleRun()
		return nil
	})

	// Close current window
	h.cbind.Set("Alt+x", func(ev *tcell.EventKey) *tcell.EventKey {
		h.handleCloseWindow()
		return nil
	})

	// Quit
	h.cbind.Set("Alt+q", func(ev *tcell.EventKey) *tcell.EventKey {
		h.handleQuit()
		return nil
	})
}

// HandleEvent processes keyboard events
func (h *Handler) HandleEvent(ev *tcell.EventKey) *tcell.EventKey {
	// Check if this is an ALT command
	if ev.Modifiers()&tcell.ModAlt != 0 {
		// Auto-save if in vi mode
		activeWin := h.wm.GetActiveWindow()
		if activeWin != nil && activeWin.State.IsVi {
			h.autoSave(activeWin)
		}

		// Process ALT command (returns nil to consume event)
		return h.cbind.Capture(ev)
	}

	// Special handling for Enter key in build/run output windows
	if ev.Key() == tcell.KeyEnter {
		activeWin := h.wm.GetActiveWindow()
		if activeWin != nil {
			if activeWin.ProcessType == window.ProcessBuildOutput {
				h.goHandler.HandleEnterInBuildOutput(activeWin)
				return nil
			}
		}
	}

	// Not an ALT command - pass through to PTY
	return ev
}

// autoSave auto-saves the current file in vi
func (h *Handler) autoSave(win *window.Window) {
	if !win.State.IsVi || win.State.Filename == "" {
		return
	}

	// Send ESC to ensure command mode
	win.WriteToPTY([]byte{27})
	time.Sleep(50 * time.Millisecond)

	// Send :w command
	win.WriteToPTY([]byte(":w\r"))
	time.Sleep(100 * time.Millisecond)

	// Mark as saved
	win.State.IsDirty = false
}

// Command handlers

func (h *Handler) handleSplitHorizontal() {
	if err := h.wm.SplitActive(window.SplitHorizontal, "/bin/bash"); err != nil {
		// Error - could show in status line
	}
}

func (h *Handler) handleSplitVertical() {
	if err := h.wm.SplitActive(window.SplitVertical, "/bin/bash"); err != nil {
		// Error - could show in status line
	}
}

func (h *Handler) handleToggle() {
	h.wm.TogglePrevious()
}

func (h *Handler) handleJumpToWindow(num int) {
	win := h.wm.GetWindowByIndex(num)
	if win != nil {
		h.wm.SetActiveByID(win.ID)
	}
}

func (h *Handler) handleBuild() {
	activeWin := h.wm.GetActiveWindow()
	if activeWin == nil {
		return
	}

	if err := h.goHandler.Build(activeWin); err != nil {
		// Show error - for now just ignore
	}
}

func (h *Handler) handleRun() {
	activeWin := h.wm.GetActiveWindow()
	if activeWin == nil {
		return
	}

	if err := h.goHandler.Run(activeWin); err != nil {
		// Show error - for now just ignore
	}
}

func (h *Handler) handleCloseWindow() {
	if err := h.wm.CloseActive(); err != nil {
		// Error - cannot close (probably last window)
		// Could show error in status line
	}
}

func (h *Handler) handleQuit() {
	// Save all vi windows
	for _, win := range h.wm.GetWindows() {
		if win.State.IsVi && win.State.Filename != "" {
			h.autoSave(win)
		}
	}

	// Signal quit
	close(h.quitChan)
}

// ParseKeyString converts a key string like "Alt+h" to tcell.EventKey
func ParseKeyString(s string) *tcell.EventKey {
	// This is handled by cbind internally
	return nil
}

// GetGoHandler returns the Go mode handler
func (h *Handler) GetGoHandler() *GoModeHandler {
	return h.goHandler
}
