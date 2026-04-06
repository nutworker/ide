# IDE UI Design

## Layout

```
┌─────────────────────────────────────┐
│                                     │
│   Terminal Content Area             │
│   (Shell or Vi output)              │
│                                     │
├─────────────────────────────────────┤
│ [1]                                 │  ← Status Bar (always present)
└─────────────────────────────────────┘

Window Split (Horizontal):
┌─────────────────────────────────────┐
│   Window 1 Content                  │
├─────────────────────────────────────┤
│ [1]                                 │  ← Status Bar
├─────────────────────────────────────┤  ← Border
│   Window 2 Content                  │
├─────────────────────────────────────┤
│ [2]                                 │  ← Status Bar
└─────────────────────────────────────┘

Window Split (Vertical):
┌──────────────────┬──────────────────┐
│   Window 1       │   Window 2       │
│   Content        │   Content        │
│                  │                  │
├──────────────────┼──────────────────┤
│ [1]              │ [2]              │  ← Status Bars
└──────────────────┴──────────────────┘
```

## Status Bar

The status bar is **always present** at the bottom of each window.

### Shell Mode
```
[1]
```
Shows: Window number only

### Vi Mode
```
[1] test.go [12,45] -- INSERT --
```
Shows: Window number, filename, cursor position, mode (INSERT/COMMAND)

## Window Borders

- **Vertical borders** (`│`) separate windows side-by-side
- **Horizontal borders** (`─`) separate windows top-to-bottom
- Borders do not overlap with content or status bar

## Color Scheme

- **Background**: White
- **Text**: Black
- **Status Bar**: White text on black background
- **Borders**: Reverse video (white on black)

## Space Allocation

For a window with dimensions `Width x Height`:
- **Content area**: `Width x (Height - 1)`
- **Status bar**: `Width x 1` (bottom line)
- **PTY size**: `Width x (Height - 1)` (shell sees this size)

This ensures:
✅ Terminal programs get accurate dimensions
✅ No overlap between content and status bar
✅ Borders don't hide content
