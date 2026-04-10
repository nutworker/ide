# Error Navigation in Build Window (Emacs-style)

## ✅ Implemented Features

### Navigation with Arrow Keys
When in a build output window, you can navigate through errors just like Emacs compilation mode:

- **Up Arrow** ↑ - Move to previous line
- **Down Arrow** ↓ - Move to next line
- **Enter** - Jump to the source file at the error location

### Visual Feedback
- The currently selected line is **highlighted** (reverse video)
- You can see exactly which error you're about to jump to
- Arrow keys move the highlight up and down

### Jump to Error
- Press **Enter** on any highlighted line
- If the line matches the error format (`file.go:line:col: message`):
  - Automatically switches to the source window
  - Jumps to the exact line number
  - Positions cursor at the column

## 🎯 Workflow Example

1. **Open a file with errors:**
   ```bash
   ./ide examples/test_errors.go
   ```

2. **Build:**
   - Press **ALT+b**
   - Build window opens below with errors
   - First line (line 0) is highlighted

3. **Navigate errors:**
   - Press **Down Arrow** to move to next error
   - Press **Up Arrow** to move to previous error
   - Watch the highlight move through the error list

4. **Jump to error:**
   - When on an error you want to fix, press **Enter**
   - IDE jumps to source window at that exact line
   - Cursor positioned at the error location

5. **Fix and rebuild:**
   - Fix the error in vi
   - Press **ALT+b** to rebuild
   - Navigate through remaining errors
   - Repeat until all errors are fixed

## 📋 Technical Details

### Error Line Format
The error parser recognizes standard Go error format:
```
filename.go:42:15: error message
```

Components:
- `filename.go` - Source file
- `42` - Line number
- `15` - Column number  
- `error message` - Description

Both `go vet` warnings and `go build` errors use this format.

### Window State
- `SelectedLine` tracks which line is highlighted
- Starts at 0 (first line) when build window opens
- Resets to 0 on each new build
- Bounds-checked (can't go above first or below last line)

### Visual Indication
- Selected line uses reverse video (background ↔ foreground swap)
- Easy to see which line you're on
- Works with any color scheme

### Integration with Mouse
- Mouse wheel still works for scrolling
- Arrow keys for precise line-by-line navigation
- Enter to jump - no mouse needed

## 🧪 Testing

Test with the error file:
```bash
./ide examples/test_errors.go
```

Test sequence:
1. Press **ALT+b** → build window opens
   - ✅ First error line highlighted

2. Press **Down Arrow** repeatedly
   - ✅ Highlight moves down through errors
   - ✅ Can see each error message

3. Press **Up Arrow** repeatedly  
   - ✅ Highlight moves back up
   - ✅ Stops at line 0

4. Navigate to an error and press **Enter**
   - ✅ Jumps to source window
   - ✅ Cursor at exact error line
   - ✅ Column positioned correctly

5. Press **ALT+t** to toggle back to build window
   - ✅ Highlight still on same line
   - ✅ Can continue navigating

## 💡 Tips

1. **Quick error fixing:**
   - Navigate to first error with Down Arrow
   - Press Enter to jump
   - Fix it
   - ALT+b to rebuild
   - Repeat

2. **Skip non-error lines:**
   - Some lines are just output, not errors
   - Press Enter on them - nothing happens
   - Move to actual error lines (with filename:line:col format)

3. **Work through systematically:**
   - Start at top error (line 0)
   - Fix each error from top to bottom
   - Rebuild after each fix to see progress

4. **Multiple errors in same file:**
   - Fix them in order from top to bottom
   - Line numbers stay valid
   - No need to re-navigate build window

## 🎨 Comparison with Emacs

This implementation mimics Emacs compilation mode:

| Feature | Emacs | This IDE |
|---------|-------|----------|
| Navigate errors | `n`/`p` or arrows | Arrow keys ↑↓ |
| Jump to error | `RET` (Enter) | Enter |
| Visual feedback | Highlight | Reverse video |
| Auto-parse errors | ✅ | ✅ |
| Column positioning | ✅ | ✅ |

The main difference: Emacs uses `n`/`p` keys while we use arrow keys (more intuitive for navigation).

## 🔧 Implementation Notes

- Works only in `ProcessBuildOutput` windows
- Arrow keys in regular bash windows still go to bash (command history)
- Arrow keys in vi windows still go to vi (cursor movement)
- Only build output windows get the navigation behavior

