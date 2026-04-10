package window

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

// Manager manages all windows in the IDE
type Manager struct {
	windows      []*Window
	activeIdx    int
	prevIdx      int
	layout       *Layout
	maxWindows   int
	nextID       int
	defaultStyle tcell.Style
}

// NewManager creates a new window manager
func NewManager(maxWindows int, defaultStyle tcell.Style) *Manager {
	return &Manager{
		windows:      make([]*Window, 0, maxWindows),
		maxWindows:   maxWindows,
		nextID:       1,
		defaultStyle: defaultStyle,
	}
}

// CreateWindow creates a new window with a PTY running the given command
func (m *Manager) CreateWindow(rect Rect, command string, args ...string) (*Window, error) {
	if len(m.windows) >= m.maxWindows {
		return nil, fmt.Errorf("maximum windows (%d) reached", m.maxWindows)
	}

	window := NewWindow(m.nextID, rect, m.defaultStyle)
	m.nextID++

	if err := window.StartPTY(command, args...); err != nil {
		return nil, err
	}

	m.windows = append(m.windows, window)

	// Initialize layout if this is the first window
	if m.layout == nil {
		m.layout = NewLayout(window)
		m.activeIdx = 0
		m.prevIdx = 0
	}

	return window, nil
}

// GetActiveWindow returns the currently active window
func (m *Manager) GetActiveWindow() *Window {
	if m.activeIdx >= 0 && m.activeIdx < len(m.windows) {
		return m.windows[m.activeIdx]
	}
	return nil
}

// SetActive sets the active window by index
func (m *Manager) SetActive(idx int) {
	if idx >= 0 && idx < len(m.windows) {
		m.prevIdx = m.activeIdx
		m.activeIdx = idx
	}
}

// SetActiveByID sets the active window by window ID
func (m *Manager) SetActiveByID(windowID int) {
	for i, w := range m.windows {
		if w.ID == windowID {
			m.SetActive(i)
			return
		}
	}
}

// TogglePrevious toggles to the previously active window
func (m *Manager) TogglePrevious() {
	m.activeIdx, m.prevIdx = m.prevIdx, m.activeIdx
}

// GetWindowByID returns a window by its ID
func (m *Manager) GetWindowByID(id int) *Window {
	for _, w := range m.windows {
		if w.ID == id {
			return w
		}
	}
	return nil
}

// GetWindowByIndex returns a window by its index (1-based for user display)
func (m *Manager) GetWindowByIndex(idx int) *Window {
	if idx >= 1 && idx <= len(m.windows) {
		return m.windows[idx-1]
	}
	return nil
}

// SplitActive splits the active window
func (m *Manager) SplitActive(splitType SplitType, command string, args ...string) error {
	return m.splitActiveInternal(splitType, command, false, args...)
}

// SplitActiveForced splits the active window, forcing the command type (no vi auto-detection)
func (m *Manager) SplitActiveForced(splitType SplitType, command string, args ...string) error {
	return m.splitActiveInternal(splitType, command, true, args...)
}

// splitActiveInternal is the internal implementation for splitting
func (m *Manager) splitActiveInternal(splitType SplitType, command string, forceCommand bool, args ...string) error {
	if len(m.windows) >= m.maxWindows {
		return fmt.Errorf("maximum windows reached")
	}

	activeWin := m.GetActiveWindow()
	if activeWin == nil {
		return fmt.Errorf("no active window")
	}

	// Find the node for the active window
	node := m.layout.FindNode(activeWin.ID)
	if node == nil {
		return fmt.Errorf("active window not found in layout")
	}

	// Create new window (rect will be set by SplitNode)
	newWindow := NewWindow(m.nextID, activeWin.Rect, m.defaultStyle)
	m.nextID++

	// If splitting a vi window, use the same file in the new window
	// (unless forceCommand is true, e.g., for build/run output windows)

	// Debug logging
	if f, err := os.OpenFile("/tmp/split-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "Split: IsVi=%v Filename=%s Command=%s Force=%v\n", activeWin.State.IsVi, activeWin.State.Filename, command, forceCommand)
		if activeWin.Cmd != nil {
			fmt.Fprintf(f, "  Cmd.Path=%s Args=%v\n", activeWin.Cmd.Path, activeWin.Cmd.Args)
		}
		f.Close()
	}

	// Check if vi is running based on State (only if not forcing command)
	if !forceCommand && activeWin.State.IsVi && activeWin.State.Filename != "" && command == "/bin/bash" {
		command = "vi"
		// Use -n flag to disable swap file (no swap file warning when opening same file)
		args = []string{"-n", activeWin.State.Filename}
	}

	if err := newWindow.StartPTY(command, args...); err != nil {
		return err
	}

	// Split the layout node
	if err := m.layout.SplitNode(node, splitType, newWindow); err != nil {
		return err
	}

	// Add to windows list
	m.windows = append(m.windows, newWindow)

	// Make the new window active
	m.SetActive(len(m.windows) - 1)

	return nil
}

// GetWindows returns all windows
func (m *Manager) GetWindows() []*Window {
	return m.windows
}

// ResizeAll resizes all windows to fit new screen dimensions
func (m *Manager) ResizeAll(width, height int) {
	if m.layout != nil {
		m.layout.ResizeAll(NewRect(0, 0, width, height-1))
	}
}

// CloseActive closes the active window and gives space to parent
func (m *Manager) CloseActive() error {
	if len(m.windows) <= 1 {
		return fmt.Errorf("cannot close the last window")
	}

	activeWin := m.GetActiveWindow()
	if activeWin == nil {
		return fmt.Errorf("no active window")
	}

	// Remove from layout and get the window that reclaims the space
	siblingWin := m.layout.RemoveWindow(activeWin.ID)

	// Close the window
	activeWin.Close()

	// Remove from windows list
	newWindows := make([]*Window, 0, len(m.windows)-1)
	for i, w := range m.windows {
		if w.ID != activeWin.ID {
			newWindows = append(newWindows, w)
		} else if i < m.activeIdx {
			m.activeIdx--
		}
	}
	m.windows = newWindows

	// Set focus to the sibling window (the one that reclaimed space)
	if siblingWin != nil {
		m.SetActiveByID(siblingWin.ID)
	} else if m.activeIdx >= len(m.windows) {
		m.activeIdx = len(m.windows) - 1
	}

	// Adjust prevIdx
	if m.prevIdx >= len(m.windows) {
		m.prevIdx = m.activeIdx
	}

	return nil
}

// GetWindowAtPosition returns the window at the given screen coordinates
func (m *Manager) GetWindowAtPosition(x, y int) *Window {
	for _, w := range m.windows {
		if w.IsPointInside(x, y) {
			return w
		}
	}
	return nil
}

// CloseAll closes all windows
func (m *Manager) CloseAll() {
	for _, w := range m.windows {
		w.Close()
	}
}
