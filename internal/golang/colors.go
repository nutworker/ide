package golang

import (
	"go/token"

	"github.com/gdamore/tcell/v2"
	"github.com/nutworker/ide/internal/ui"
)

// TokenStyler maps Go tokens to tcell styles
type TokenStyler struct {
	theme *ui.Theme
}

// NewTokenStyler creates a new token styler
func NewTokenStyler(theme *ui.Theme) *TokenStyler {
	return &TokenStyler{theme: theme}
}

// GetStyle returns the appropriate style for a given token type
func (ts *TokenStyler) GetStyle(tok token.Token) tcell.Style {
	switch {
	case tok.IsKeyword():
		return ts.theme.KeywordStyle()
	case tok == token.STRING, tok == token.CHAR:
		return ts.theme.StringStyle()
	case tok == token.COMMENT:
		return ts.theme.CommentStyle()
	case tok == token.INT, tok == token.FLOAT, tok == token.IMAG:
		return ts.theme.NumberStyle()
	case tok == token.IDENT:
		// Could be a type or function - for now use default
		return ts.theme.FunctionStyle()
	default:
		return ts.theme.Default()
	}
}
