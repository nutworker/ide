package golang

import (
	"crypto/sha256"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/nutworker/ide/internal/ui"
)

// CacheEntry represents a cached parse result
type CacheEntry struct {
	Styles    ui.StyleMap
	Hash      string
	Timestamp time.Time
}

// Highlighter provides syntax highlighting for Go source code
type Highlighter struct {
	theme   *ui.Theme
	styler  *TokenStyler
	cache   map[string]*CacheEntry
	mutex   sync.RWMutex
	maxAge  time.Duration
	enabled bool
}

// NewHighlighter creates a new syntax highlighter
func NewHighlighter(theme *ui.Theme) *Highlighter {
	return &Highlighter{
		theme:   theme,
		styler:  NewTokenStyler(theme),
		cache:   make(map[string]*CacheEntry),
		maxAge:  5 * time.Second,
		enabled: true,
	}
}

// SetEnabled enables or disables the highlighter
func (h *Highlighter) SetEnabled(enabled bool) {
	h.enabled = enabled
}

// GetStyles returns syntax highlighting styles for the given source text
// Returns a map of [row][col] -> tcell.Style
func (h *Highlighter) GetStyles(filename string, source []byte) ui.StyleMap {
	if !h.enabled {
		return nil
	}

	// Compute hash for cache key
	hash := fmt.Sprintf("%x", sha256.Sum256(source))

	// Check cache
	h.mutex.RLock()
	if entry, ok := h.cache[filename]; ok {
		if entry.Hash == hash && time.Since(entry.Timestamp) < h.maxAge {
			h.mutex.RUnlock()
			return entry.Styles
		}
	}
	h.mutex.RUnlock()

	// Parse and highlight
	styles := h.parseAndHighlight(source)

	// Update cache
	h.mutex.Lock()
	h.cache[filename] = &CacheEntry{
		Styles:    styles,
		Hash:      hash,
		Timestamp: time.Now(),
	}
	h.mutex.Unlock()

	return styles
}

// parseAndHighlight parses Go source and returns style mappings
func (h *Highlighter) parseAndHighlight(source []byte) ui.StyleMap {
	styles := make(ui.StyleMap)

	// Create file set and file
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(source))

	// Create scanner
	var s scanner.Scanner
	s.Init(file, source, nil, scanner.ScanComments)

	// Scan tokens
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}

		// Get position
		position := fset.Position(pos)
		line := position.Line - 1 // Convert to 0-based
		col := position.Column - 1 // Convert to 0-based

		// Get style for this token
		style := h.styler.GetStyle(tok)

		// Map each character of the token
		litLen := len(lit)
		if litLen == 0 {
			// For operators and other tokens, use token string
			litLen = len(tok.String())
		}

		// Create row map if needed
		if styles[line] == nil {
			styles[line] = make(map[int]tcell.Style)
		}

		// Apply style to all characters in the token
		for i := 0; i < litLen; i++ {
			styles[line][col+i] = style
		}
	}

	return styles
}

// GetStylesFromFile loads and highlights a file
func (h *Highlighter) GetStylesFromFile(filename string) (ui.StyleMap, error) {
	if !h.enabled {
		return nil, nil
	}

	// Read file
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return h.GetStyles(filename, source), nil
}

// InvalidateCache clears the cache for a specific file
func (h *Highlighter) InvalidateCache(filename string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.cache, filename)
}

// ClearCache clears all cached entries
func (h *Highlighter) ClearCache() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.cache = make(map[string]*CacheEntry)
}
