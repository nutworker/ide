# Build Window Improvements

## ✅ Implemented Features

### 1. Build Window Positioned Below Source
- Build output window now opens **below** the source window (horizontal split)
- Source window remains visible above while errors show below
- Natural workflow: code above, errors below

### 2. Auto-Focus Build Window
- After running build (ALT+b), cursor automatically moves to build window
- Errors are immediately visible and ready for navigation
- No need to manually switch windows

### 3. Scrolling Support
- **Mouse wheel** scrolling works in build output window
- **Scroll up**: View older output (3 lines per wheel notch)
- **Scroll down**: View newer output (3 lines per wheel notch)
- Navigate through long error lists easily

### 4. Close Build Window with ALT+x
- Press **ALT+x** in build window to close it
- Source window reclaims the space
- All window closing operations work normally

### 5. Build Window Reuse
- Building the same file multiple times **reuses the same build window**
- No window proliferation - clean workspace
- Build window is cleared and reused for each build
- Mapping tracked: one build window per source file

## 🎯 Workflow

### First Build
1. Open a Go file: `./ide examples/test_errors.go`
2. Press **ALT+b**
3. Window splits horizontally:
   - **Top**: Source code (vi)
   - **Bottom**: Build output (auto-focused)
4. See errors in bottom window
5. Press **Enter** on error line → jumps to source

### Subsequent Builds
1. Fix error in source window
2. Press **ALT+b** again
3. **Same build window** is reused (cleared first)
4. Cursor auto-moves to build window
5. View new errors (if any)

### Navigation
- **In build window**:
  - Use **arrow keys** or **mouse** to select error line
  - **Mouse wheel** to scroll through errors
  - Press **Enter** → jumps to source at exact line/column
  
- **In source window**:
  - Fix the error
  - Press **ALT+b** → rebuild

### Cleanup
- Press **ALT+x** in build window to close it
- Source window expands to full size
- Can rebuild later, new window will be created

## 📋 Technical Details

### Window Tracking
- `buildWindows` map: build window ID → source window ID
- `sourceBuildWins` map: source window ID → build window ID
- Bidirectional mapping for efficient lookup

### Window Lifecycle
- Created on first build (horizontal split below source)
- Reused on subsequent builds of same file
- Cleared before each build (`clear` command)
- Auto-removed from tracking maps when closed

### Focus Management
- After build command sent, wait 200ms for errors
- Auto-switch to build window to show results
- User can navigate errors or switch back to source

## 🧪 Testing

Test with the error file:
```bash
./ide examples/test_errors.go
```

Test sequence:
1. **First build**: Press ALT+b
   - ✅ Window splits (source above, build below)
   - ✅ Cursor in build window
   - ✅ See multiple errors

2. **Navigate**: Press Enter on first error
   - ✅ Jumps to source at error line
   - ✅ Cursor positioned correctly

3. **Fix error**: Remove or fix the error
   - ✅ Edit in vi normally

4. **Rebuild**: Press ALT+b
   - ✅ Same build window reused
   - ✅ Old output cleared
   - ✅ New errors shown
   - ✅ Auto-focused

5. **Scroll**: Use mouse wheel in build window
   - ✅ Can scroll up/down through errors

6. **Close**: Press ALT+x in build window
   - ✅ Build window closes
   - ✅ Source window expands

7. **Rebuild**: Press ALT+b again
   - ✅ New build window created below

## 💡 Benefits

- **Better workflow**: Code and errors visible simultaneously
- **Less clutter**: Same build window reused, no window spam
- **Auto-focus**: Immediately see build results
- **Easy navigation**: Scroll through errors, jump to source
- **Clean cleanup**: Close with ALT+x like any window
- **Intuitive**: Build window behaves like a standard IDE output panel

