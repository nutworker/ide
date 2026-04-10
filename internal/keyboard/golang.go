package keyboard

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nutworker/ide/internal/window"
)

// GoModeHandler handles Go-specific functionality
type GoModeHandler struct {
	wm              *window.Manager
	errorParser     *GoErrorParser
	buildWindows    map[int]int // Maps build output window ID to source window ID
	sourceBuildWins map[int]int // Maps source window ID to build output window ID
}

// NewGoModeHandler creates a new Go mode handler
func NewGoModeHandler(wm *window.Manager) *GoModeHandler {
	return &GoModeHandler{
		wm:              wm,
		errorParser:     NewGoErrorParser(),
		buildWindows:    make(map[int]int),
		sourceBuildWins: make(map[int]int),
	}
}

// Build builds the Go file in the active window
func (gh *GoModeHandler) Build(sourceWin *window.Window) error {
	if !strings.HasSuffix(sourceWin.State.Filename, ".go") {
		return fmt.Errorf("not a Go file")
	}

	var outputWin *window.Window
	var err error

	// Check if we already have a build window for this source
	if existingBuildWinID, exists := gh.sourceBuildWins[sourceWin.ID]; exists {
		outputWin = gh.wm.GetWindowByID(existingBuildWinID)
		if outputWin != nil {
			// Reuse existing build window - clear it first
			outputWin.WriteToPTY([]byte("clear\n"))
			time.Sleep(50 * time.Millisecond)
			outputWin.State.SelectedLine = 0 // Reset to first line
		}
	}

	// If no existing build window, create a new one below the source window
	if outputWin == nil {
		// Split the source window horizontally (build window below)
		// Use SplitActiveForced to ensure we get bash, not vi
		err = gh.wm.SplitActiveForced(window.SplitHorizontal, "/bin/bash")
		if err != nil {
			return err
		}

		// The new window is now active and is the last window
		outputWin = gh.wm.GetActiveWindow()
		if outputWin == nil {
			return fmt.Errorf("failed to get build output window")
		}

		outputWin.ProcessType = window.ProcessBuildOutput
		outputWin.SourceFile = sourceWin.State.Filename
		outputWin.State.SelectedLine = 0 // Start at first line

		// Track the relationship (bidirectional)
		gh.buildWindows[outputWin.ID] = sourceWin.ID
		gh.sourceBuildWins[sourceWin.ID] = outputWin.ID

		// Switch back to source window initially
		gh.wm.SetActiveByID(sourceWin.ID)

		// Give bash a moment to initialize
		time.Sleep(100 * time.Millisecond)
	}

	// Execute build command with vet (use ; to run both even if first fails)
	buildCmd := fmt.Sprintf("go vet %s 2>&1; go build %s 2>&1\n", sourceWin.State.Filename, sourceWin.State.Filename)
	outputWin.WriteToPTY([]byte(buildCmd))

	// Wait a moment for errors to appear, then switch to build window
	time.Sleep(200 * time.Millisecond)
	gh.wm.SetActiveByID(outputWin.ID)

	return nil
}

// Run runs the Go file in the active window
func (gh *GoModeHandler) Run(sourceWin *window.Window) error {
	if !strings.HasSuffix(sourceWin.State.Filename, ".go") {
		return fmt.Errorf("not a Go file")
	}

	// Split the source window horizontally (run output below)
	// Use SplitActiveForced to ensure we get bash, not vi
	err := gh.wm.SplitActiveForced(window.SplitHorizontal, "/bin/bash")
	if err != nil {
		return err
	}

	// The new window is now active and is the last window
	outputWin := gh.wm.GetActiveWindow()
	if outputWin == nil {
		return fmt.Errorf("failed to get run output window")
	}

	outputWin.ProcessType = window.ProcessRunOutput
	outputWin.SourceFile = sourceWin.State.Filename

	// Give bash a moment to initialize
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

	// Debug logging
	if f, err := os.OpenFile("/tmp/enter-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "HandleEnter: line='%s' selectedLine=%d\n", line, outputWin.State.SelectedLine)
		f.Close()
	}

	// Parse error
	compileErr := gh.errorParser.ParseLine(line)
	if compileErr == nil {
		// Debug: not an error line
		if f, err := os.OpenFile("/tmp/enter-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(f, "  -> Not an error line\n")
			f.Close()
		}
		return // Not an error line
	}

	// Debug: found error
	if f, err := os.OpenFile("/tmp/enter-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "  -> Error: file=%s line=%d col=%d\n", compileErr.File, compileErr.Line, compileErr.Column)
		f.Close()
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
		// Also handles: vet: filename.go:42:15: error message
		// Also handles: # command-line-arguments (ignored)
		errorRegex: regexp.MustCompile(`(?:^|.*:\s+)([^:]+\.go):(\d+):(\d+):\s+(.+)$`),
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

// NotifyWindowClosed cleans up tracking when a window is closed
func (gh *GoModeHandler) NotifyWindowClosed(windowID int) {
	// If it's a build window, remove from both maps
	if sourceWinID, isBuildWin := gh.buildWindows[windowID]; isBuildWin {
		delete(gh.buildWindows, windowID)
		delete(gh.sourceBuildWins, sourceWinID)
	}

	// If it's a source window, remove its build window mapping
	if buildWinID, hasBuiltWin := gh.sourceBuildWins[windowID]; hasBuiltWin {
		delete(gh.sourceBuildWins, windowID)
		delete(gh.buildWindows, buildWinID)
	}
}
