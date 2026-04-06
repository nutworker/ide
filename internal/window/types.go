package window

// Rect represents a rectangular area
type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// NewRect creates a new rectangle
func NewRect(x, y, width, height int) Rect {
	return Rect{X: x, Y: y, Width: width, Height: height}
}

// SplitType defines how a window is split
type SplitType int

const (
	SplitNone SplitType = iota
	SplitHorizontal
	SplitVertical
)

// ProcessType defines the type of process running in a window
type ProcessType int

const (
	ProcessShell ProcessType = iota
	ProcessEditor
	ProcessBuildOutput
	ProcessRunOutput
)

// ViMode represents vi editor mode
type ViMode int

const (
	ViModeCommand ViMode = iota
	ViModeInsert
)

func (m ViMode) String() string {
	switch m {
	case ViModeCommand:
		return "COMMAND"
	case ViModeInsert:
		return "INSERT"
	default:
		return "UNKNOWN"
	}
}

// WindowState tracks the state of a window
type WindowState struct {
	IsVi      bool
	ViMode    ViMode
	Filename  string
	CursorRow int
	CursorCol int
	IsDirty   bool
}

// NewWindowState creates a new window state
func NewWindowState() *WindowState {
	return &WindowState{
		IsVi:      false,
		ViMode:    ViModeCommand,
		CursorRow: 1,
		CursorCol: 1,
	}
}
