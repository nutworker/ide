package app

import (
	"fmt"
	"os"

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
	promptActive      bool
	promptText        string
	promptInput       string
	promptCallback    func(string) // Called when user presses Enter
	completionMatches []string     // Current completion candidates
	completionPrefix  string       // Common prefix for completions
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

	// Set prompt function for keyboard handler
	keyHandler.SetPromptFunc(app.showPrompt)
	keyHandler.SetRestartFunc(app.restartWindowPTY)

	return app, nil
}

// Run starts the main event loop
func (a *App) Run() error {
	defer a.cleanup()

	// Initial render
	a.renderer.Render(a.wm, a.promptActive, a.promptText, a.promptInput, a.completionMatches)

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
		a.renderer.Render(a.wm, a.promptActive, a.promptText, a.promptInput, a.completionMatches)
	}
}

// handleScreenEvent handles screen events (keyboard, resize, mouse)
func (a *App) handleScreenEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		// If prompt is active, handle prompt input
		if a.promptActive {
			a.handlePromptKey(ev)
			return
		}

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

		// Handle Ctrl-P (paste) - but not in vi mode
		if ev.Key() == tcell.KeyCtrlP {
			activeWin := a.wm.GetActiveWindow()
			if activeWin == nil || !activeWin.State.IsVi {
				a.handlePaste()
				return
			}
			// In vi mode, pass Ctrl-P through to vi
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
				// Track vi mode changes for status bar
				if activeWin.State.IsVi {
					a.trackViMode(activeWin, processedEv)
				}

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

		// Auto-copy selection to clipboard on mouse release
		if a.selection.Active {
			win := a.wm.GetWindowByID(a.selection.WindowID)
			if win != nil {
				a.clipboard = a.selection.GetSelectedText(win)
			}
		}

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
	win := a.wm.GetWindowByID(pev.WindowID)
	if win == nil {
		return
	}

	// Ignore events from old PTY generations (stale reader goroutines)
	if pev.Generation != win.Generation {
		return
	}

	if pev.Err != nil {
		// PTY closed or error - only auto-restart if reader is still active
		// (prevents race with manual restart from keyboard handler)
		if a.activeReaders[pev.WindowID] {
			// Increment generation
			win.Generation++

			// Check if we should pop from file stack (vi exited)
			if win.State.IsVi && len(win.FileStack) > 0 {
				// Pop the previous file from stack
				prevFile := win.FileStack[len(win.FileStack)-1]
				win.FileStack = win.FileStack[:len(win.FileStack)-1]

				// Reset state
				win.State = window.NewWindowState()

				// Reopen previous file
				win.StartPTY("vi", "-n", prevFile)

				// Set state immediately
				win.State.IsVi = true
				win.State.Filename = prevFile
				win.State.ViMode = window.ViModeCommand
			} else {
				// No file history - restart bash
				win.StartPTY("/bin/bash")
			}

			// Restart the reader for this window
			go win.ReadPTY(a.ptyEvents)
		}
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

// trackViMode tracks vi mode changes based on keyboard input
func (a *App) trackViMode(win *window.Window, ev *tcell.EventKey) {
	// ESC key switches to command mode
	if ev.Key() == tcell.KeyEscape {
		win.State.ViMode = window.ViModeCommand
		return
	}

	// In command mode, check for keys that enter insert mode
	if win.State.ViMode == window.ViModeCommand && ev.Key() == tcell.KeyRune {
		r := ev.Rune()
		// i, a, o, O, I, A, s, S, C, c enter insert mode
		if r == 'i' || r == 'a' || r == 'o' || r == 'O' ||
		   r == 'I' || r == 'A' || r == 's' || r == 'S' ||
		   r == 'C' || r == 'c' {
			win.State.ViMode = window.ViModeInsert
		}
	}
}

// showPrompt shows a prompt at the bottom of the screen for user input
func (a *App) showPrompt(promptText string, callback func(string)) {
	a.promptActive = true
	a.promptText = promptText
	a.promptInput = ""
	a.promptCallback = callback
}

// handlePromptKey handles keyboard input when prompt is active
func (a *App) handlePromptKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		// Cancel prompt
		a.promptActive = false
		a.promptInput = ""
		a.promptCallback = nil
		a.completionMatches = nil

	case tcell.KeyEnter:
		// Accept input
		if a.promptCallback != nil {
			a.promptCallback(a.promptInput)
		}
		a.promptActive = false
		a.promptInput = ""
		a.promptCallback = nil
		a.completionMatches = nil

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		// Delete last character
		if len(a.promptInput) > 0 {
			a.promptInput = a.promptInput[:len(a.promptInput)-1]
			// Clear completion matches on edit
			a.completionMatches = nil
		}

	case tcell.KeyTab:
		// Trigger completion
		a.handleCompletion()

	case tcell.KeyRune:
		// Add character to input
		a.promptInput += string(ev.Rune())
		// Clear completion matches on edit
		a.completionMatches = nil
	}
}

// GetPromptState returns the current prompt state for rendering
func (a *App) GetPromptState() (active bool, text string, input string) {
	return a.promptActive, a.promptText, a.promptInput
}

// handleCompletion handles Tab completion for file paths
func (a *App) handleCompletion() {
	input := a.promptInput

	// Parse the input to get directory and prefix
	dir := "."
	prefix := input

	// Find the last slash to split directory and filename
	lastSlash := -1
	for i := len(input) - 1; i >= 0; i-- {
		if input[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash >= 0 {
		dir = input[:lastSlash+1]
		prefix = input[lastSlash+1:]
		// Handle empty directory (just "/")
		if dir == "" {
			dir = "/"
		}
	}

	// Expand ~ to home directory
	if len(dir) > 0 && dir[0] == '~' {
		home := os.Getenv("HOME")
		if home != "" {
			dir = home + dir[1:]
		}
	}

	// Read directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Find matches
	var matches []string
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files unless prefix starts with .
		if len(prefix) == 0 && len(name) > 0 && name[0] == '.' {
			continue
		}
		// Check if name starts with prefix
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			fullName := name
			// Add trailing slash for directories
			if entry.IsDir() {
				fullName += "/"
			}
			matches = append(matches, fullName)
		}
	}

	if len(matches) == 0 {
		// No matches - do nothing
		return
	}

	if len(matches) == 1 {
		// Single match - complete it
		if lastSlash >= 0 {
			a.promptInput = input[:lastSlash+1] + matches[0]
		} else {
			a.promptInput = matches[0]
		}
		a.completionMatches = nil
	} else {
		// Multiple matches - find common prefix and show candidates
		commonPrefix := findCommonPrefix(matches)

		if commonPrefix != prefix {
			// We can complete to common prefix
			if lastSlash >= 0 {
				a.promptInput = input[:lastSlash+1] + commonPrefix
			} else {
				a.promptInput = commonPrefix
			}
		}

		// Store matches to display
		a.completionMatches = matches
		a.completionPrefix = commonPrefix
	}
}

// findCommonPrefix finds the longest common prefix of a slice of strings
func findCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		// Find common prefix between current prefix and s
		i := 0
		for i < len(prefix) && i < len(s) && prefix[i] == s[i] {
			i++
		}
		prefix = prefix[:i]
		if prefix == "" {
			break
		}
	}
	return prefix
}

// restartWindowPTY restarts a window's PTY with a new command
func (a *App) restartWindowPTY(win *window.Window, command string, args ...string) error {
	// Increment generation to invalidate old PTY reader events
	win.Generation++

	// Reset window state
	win.State = window.NewWindowState()

	// Close current PTY
	if win.PTY != nil {
		win.PTY.Close()
	}
	if win.Cmd != nil && win.Cmd.Process != nil {
		win.Cmd.Process.Kill()
	}

	// Start new PTY
	if err := win.StartPTY(command, args...); err != nil {
		return err
	}

	// If opening vi with a file, set state immediately
	if command == "vi" && len(args) > 0 {
		// Find the filename (skip flags like -n)
		for _, arg := range args {
			if arg != "" && arg[0] != '-' {
				win.State.IsVi = true
				win.State.Filename = arg
				win.State.ViMode = window.ViModeCommand
				break
			}
		}
	}

	// Restart PTY reader
	a.activeReaders[win.ID] = true
	go win.ReadPTY(a.ptyEvents)

	return nil
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
	case tcell.KeyCtrlB:
		return []byte{2}
	case tcell.KeyCtrlD:
		return []byte{4}
	case tcell.KeyCtrlE:
		return []byte{5}
	case tcell.KeyCtrlF:
		return []byte{6}
	case tcell.KeyCtrlK:
		return []byte{11}
	case tcell.KeyCtrlL:
		return []byte{12}
	case tcell.KeyCtrlN:
		return []byte{14}
	case tcell.KeyCtrlU:
		return []byte{21}
	case tcell.KeyCtrlW:
		return []byte{23}
	case tcell.KeyF1:
		return []byte{27, 'O', 'P'}
	case tcell.KeyF2:
		return []byte{27, 'O', 'Q'}
	case tcell.KeyF3:
		return []byte{27, 'O', 'R'}
	case tcell.KeyF4:
		return []byte{27, 'O', 'S'}
	case tcell.KeyF5:
		return []byte{27, '[', '1', '5', '~'}
	case tcell.KeyF6:
		return []byte{27, '[', '1', '7', '~'}
	case tcell.KeyF7:
		return []byte{27, '[', '1', '8', '~'}
	case tcell.KeyF8:
		return []byte{27, '[', '1', '9', '~'}
	case tcell.KeyF9:
		return []byte{27, '[', '2', '0', '~'}
	case tcell.KeyF10:
		return []byte{27, '[', '2', '1', '~'}
	case tcell.KeyF11:
		return []byte{27, '[', '2', '3', '~'}
	case tcell.KeyF12:
		return []byte{27, '[', '2', '4', '~'}
	default:
		return []byte{}
	}
}
