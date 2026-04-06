# Copy-Paste Support

## Overview

The IDE supports copy-paste operations using both keyboard shortcuts and mouse gestures.

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| **Ctrl-Shift-C** | Copy selected text to clipboard |
| **Ctrl-Shift-V** | Paste clipboard to active window |
| **Ctrl-C** | Send interrupt to shell (traditional behavior) |

## Mouse Operations

| Action | Function |
|--------|----------|
| **Click and drag** | Select text (highlighted in reverse video) |
| **Middle-click** | Paste clipboard (X11 style) |
| **Mouse wheel** | Scroll up/down (in non-vi windows) |

## How to Use

### Selecting Text with Mouse

1. **Click** at the start of the text you want to select
2. **Hold and drag** to the end of the text
3. **Release** to complete selection
4. Selected text is **highlighted** with reverse video (white on black)

### Copying Text

**Method 1: Keyboard**
1. Select text with mouse
2. Press **Ctrl-Shift-C**
3. Text is copied to clipboard
4. Selection is cleared

**Method 2: Automatic**
- Selected text is automatically available for middle-click paste
- No need to explicitly copy

### Pasting Text

**Method 1: Keyboard**
1. Press **Ctrl-Shift-V**
2. Clipboard content is pasted to active window
3. Works in shell and output windows (not vi)

**Method 2: Middle-Click**
1. Click middle mouse button in target window
2. Clipboard content is pasted
3. X11/Linux standard behavior

## Examples

### Example 1: Copy Command Output
```
$ ls -la
total 48
drwxr-xr-x  5 user  staff   160
-rw-r--r--  1 user  staff  1066

[Click and drag over "drwxr-xr-x  5 user  staff   160"]
[Press Ctrl-Shift-C]
[Copied to clipboard]
```

### Example 2: Paste Between Windows
```
Window 1:
$ echo "some long command with parameters"
[Select the command]
[Ctrl-Shift-C to copy]

Window 2:
$ [Ctrl-Shift-V to paste]
$ echo "some long command with parameters"
```

### Example 3: Copy Error Message
```
Build Output Window:
./main.go:42:15: undefined: fmt.Printl
[Select error message]
[Ctrl-Shift-C]

Browser/Editor:
[Paste to search or document]
```

### Example 4: Middle-Click Paste
```
$ pwd
/home/user/projects
[Select "/home/user/projects"]

$ cd [Middle-click]
$ cd /home/user/projects
```

## Selection Behavior

### Visual Feedback
- Selected text has **reverse video** (white background on black text)
- Clear visual indication of what will be copied
- Selection spans across lines if needed

### Multi-line Selection
```
$ cat file.txt
line 1
line 2
line 3

[Select from "line 1" to "line 3"]
[Copied text includes all three lines with newlines]
```

### Selection Clearing
- Pressing **Ctrl-Shift-C** clears selection after copy
- Clicking elsewhere starts a new selection
- Starting a new drag operation replaces previous selection

## Restrictions

### Vi Mode
**Copy**: ✅ Works - select text with mouse, copy with Ctrl-Shift-C
**Paste**: ❌ Disabled - use vi's native paste commands instead
- In vi: Use `p` or `P` after yanking text with `y`
- Prevents conflicts with vi's own clipboard

### Shell Mode
**Copy**: ✅ Works - select and copy any output
**Paste**: ✅ Works - paste commands or text

### Build/Run Output
**Copy**: ✅ Works - copy errors, warnings, output
**Paste**: ✅ Works - paste into command prompt if window has shell

## Technical Details

### Clipboard Storage
- Clipboard is **in-memory** within the IDE process
- Not synchronized with system clipboard (X11/Wayland)
- Persists across windows within the IDE
- Cleared when IDE exits

### Selection Coordinates
- Selection tracks absolute screen coordinates
- Converts to window-relative positions for text extraction
- Handles window boundaries correctly

### Text Extraction
- Extracts actual text characters (not ANSI codes)
- Preserves newlines for multi-line selections
- Strips formatting and color codes

## Tips

### Copying Long Output
```bash
$ find /usr -name "*.so" > /tmp/output.txt
$ less /tmp/output.txt
[Select and copy specific parts]
```

### Copying File Paths
```bash
$ pwd
/very/long/path/to/project
[Select path]
[Ctrl-Shift-C]
[Paste elsewhere: Ctrl-Shift-V]
```

### Copying Error Messages for Search
```bash
[Build fails with error]
./main.go:42:15: undefined: fmt.Printl
[Select error message]
[Ctrl-Shift-C]
[Open browser and paste to search]
```

### Quick Command Repetition
```bash
$ some-long-command --with --many --flags
[Select command]
[Middle-click to paste again]
$ some-long-command --with --many --flags
```

## Comparison with Standard Terminals

| Feature | Standard Terminal | IDE |
|---------|------------------|-----|
| Copy shortcut | Ctrl-Shift-C | ✅ Ctrl-Shift-C |
| Paste shortcut | Ctrl-Shift-V | ✅ Ctrl-Shift-V |
| Mouse selection | Click-drag | ✅ Click-drag |
| Middle-click paste | Yes (X11) | ✅ Yes |
| Visual highlight | Usually | ✅ Reverse video |
| System clipboard | Yes | ❌ Internal only |

## Future Enhancements

Potential future improvements:
- System clipboard integration (xclip/xsel on Linux)
- Keyboard-only selection (Shift+arrows)
- Copy without clearing selection
- Paste history
- Smart paste (auto-quote spaces)

## Benefits

✅ **Familiar**: Standard Ctrl-Shift-C/V shortcuts
✅ **Visual**: Clear selection highlighting
✅ **Efficient**: Mouse drag for quick selection
✅ **Compatible**: X11-style middle-click paste
✅ **Safe**: Ctrl-C still works for interrupt
✅ **Flexible**: Works across all IDE windows
