package keyboard

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nutworker/ide/internal/window"
)

// GoModeHandler handles Go-specific functionality
type GoModeHandler struct {
	wm           *window.Manager
	errorParser  *GoErrorParser
	buildWindows map[int]int // Maps build output window ID to source window ID
}

// NewGoModeHandler creates a new Go mode handler
func NewGoModeHandler(wm *window.Manager) *GoModeHandler {
	return &GoModeHandler{
		wm:           wm,
		errorParser:  NewGoErrorParser(),
		buildWindows: make(map[int]int),
	}
}

// Build builds the Go file in the active window
func (gh *GoModeHandler) Build(sourceWin *window.Window) error {
	if !strings.HasSuffix(sourceWin.State.Filename, ".go") {
		return fmt.Errorf("not a Go file")
	}

	// Determine available space for output window
	screenW, screenH := 80, 24 // Default, will be updated by window manager
	if len(gh.wm.GetWindows()) > 0 {
		// Use a portion of screen for output
		screenW = sourceWin.Rect.Width
		screenH = sourceWin.Rect.Height / 3
		if screenH < 5 {
			screenH = 5
		}
	}

	outputRect := window.NewRect(0, 0, screenW, screenH)

	// Create output window with bash
	outputWin, err := gh.wm.CreateWindow(outputRect, "/bin/bash")
	if err != nil {
		return err
	}

	outputWin.ProcessType = window.ProcessBuildOutput
	outputWin.SourceFile = sourceWin.State.Filename

	// Track the relationship
	gh.buildWindows[outputWin.ID] = sourceWin.ID

	// Give it a moment to initialize
	time.Sleep(100 * time.Millisecond)

	// Execute build command
	buildCmd := fmt.Sprintf("go build %s\n", sourceWin.State.Filename)
	outputWin.WriteToPTY([]byte(buildCmd))

	return nil
}

// Run runs the Go file in the active window
func (gh *GoModeHandler) Run(sourceWin *window.Window) error {
	if !strings.HasSuffix(sourceWin.State.Filename, ".go") {
		return fmt.Errorf("not a Go file")
	}

	// Determine available space for output window
	screenW, screenH := 80, 24
	if len(gh.wm.GetWindows()) > 0 {
		screenW = sourceWin.Rect.Width
		screenH = sourceWin.Rect.Height / 3
		if screenH < 5 {
			screenH = 5
		}
	}

	outputRect := window.NewRect(0, 0, screenW, screenH)

	// Create output window with bash
	outputWin, err := gh.wm.CreateWindow(outputRect, "/bin/bash")
	if err != nil {
		return err
	}

	outputWin.ProcessType = window.ProcessRunOutput
	outputWin.SourceFile = sourceWin.State.Filename

	// Give it a moment to initialize
	time.Sleep(100 * time.Millisecond)

	// Execute run command
	runCmd := fmt.Sprintf("go run %s\n", sourceWin.State.Filename)
	outputWin.WriteToPTY([]byte(runCmd))

	return nil
}

// HandleEnterInBuildOutput handles Enter key in build output window
func (gh *GoModeHandler) HandleEnterInBuildOutput(outputWin *window.Window) {
	// Get current line
	line := outputWin.GetCurrentLine()

	// Parse error
	compileErr := gh.errorParser.ParseLine(line)
	if compileErr == nil {
		return // Not an error line
	}

	// Find source window
	sourceWinID, ok := gh.buildWindows[outputWin.ID]
	if !ok {
		return
	}

	sourceWin := gh.wm.GetWindowByID(sourceWinID)
	if sourceWin == nil {
		return
	}

	// Jump to error line
	gh.jumpToLine(sourceWin, compileErr.Line, compileErr.Column)

	// Switch focus to source window
	gh.wm.SetActiveByID(sourceWinID)
}

// jumpToLine jumps to a specific line and column in vi
func (gh *GoModeHandler) jumpToLine(win *window.Window, line, col int) {
	if !win.State.IsVi {
		return
	}

	// Send ESC to ensure command mode
	win.WriteToPTY([]byte{27})
	time.Sleep(50 * time.Millisecond)

	// Send goto line command
	cmd := fmt.Sprintf(":%d\r", line)
	win.WriteToPTY([]byte(cmd))
	time.Sleep(50 * time.Millisecond)

	// Send move to column if specified
	if col > 0 {
		cmd = fmt.Sprintf("%d|", col)
		win.WriteToPTY([]byte(cmd))
	}
}

// GoErrorParser parses Go compiler errors
type GoErrorParser struct {
	errorRegex *regexp.Regexp
}

// NewGoErrorParser creates a new error parser
func NewGoErrorParser() *GoErrorParser {
	return &GoErrorParser{
		// Matches: filename.go:42:15: error message
		errorRegex: regexp.MustCompile(`^([^:]+):(\d+):(\d+):\s+(.+)$`),
	}
}

// CompileError represents a compilation error
type CompileError struct {
	File    string
	Line    int
	Column  int
	Message string
}

// ParseLine parses a line and extracts error information
func (gep *GoErrorParser) ParseLine(line string) *CompileError {
	line = strings.TrimSpace(line)
	matches := gep.errorRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	lineNum, _ := strconv.Atoi(matches[2])
	colNum, _ := strconv.Atoi(matches[3])

	return &CompileError{
		File:    matches[1],
		Line:    lineNum,
		Column:  colNum,
		Message: matches[4],
	}
}
