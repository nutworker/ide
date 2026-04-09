package ui

import "github.com/gdamore/tcell/v2"

// Theme defines the color scheme for the IDE
type Theme struct {
	defaultStyle   tcell.Style
	statusBarStyle tcell.Style
	windowNumStyle tcell.Style
}

// NewTheme creates a new theme with white background and black text
func NewTheme() *Theme {
	return &Theme{
		defaultStyle:   tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite),
		statusBarStyle: tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		windowNumStyle: tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue).Bold(true),
	}
}

// Default returns the default style (black on white)
func (t *Theme) Default() tcell.Style {
	return t.defaultStyle
}

// StatusBar returns the status bar style (white on black)
func (t *Theme) StatusBar() tcell.Style {
	return t.statusBarStyle
}

// WindowNum returns the window number style
func (t *Theme) WindowNum() tcell.Style {
	return t.windowNumStyle
}

// Syntax highlighting styles for Go code

// KeywordStyle returns the style for Go keywords (func, if, for, etc.)
// Bold navy blue - high contrast and professional
func (t *Theme) KeywordStyle() tcell.Style {
	return t.defaultStyle.Foreground(tcell.ColorNavy).Bold(true)
}

// StringStyle returns the style for string literals
// Bold green - bright and traditional for strings
func (t *Theme) StringStyle() tcell.Style {
	return t.defaultStyle.Foreground(tcell.ColorGreen).Bold(true)
}

// CommentStyle returns the style for comments
// Bold dark red/maroon - visible but distinguishable from code
func (t *Theme) CommentStyle() tcell.Style {
	return t.defaultStyle.Foreground(tcell.ColorMaroon).Bold(true)
}

// TypeStyle returns the style for type names
// Bold dark magenta - very distinct and highly visible
func (t *Theme) TypeStyle() tcell.Style {
	return t.defaultStyle.Foreground(tcell.ColorDarkMagenta).Bold(true)
}

// NumberStyle returns the style for numeric literals
// Bold dark cyan - bright and stands out
func (t *Theme) NumberStyle() tcell.Style {
	return t.defaultStyle.Foreground(tcell.ColorDarkCyan).Bold(true)
}

// FunctionStyle returns the style for function names (default black)
func (t *Theme) FunctionStyle() tcell.Style {
	return t.defaultStyle
}
