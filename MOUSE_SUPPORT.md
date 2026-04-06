# Mouse Support

## Overview

The IDE supports **mouse wheel scrolling** to navigate through shell output and build/run output windows.

## Scrolling Behavior

### Where Scrolling Works ✅

- **Shell windows** (bash prompt)
- **Build output windows** (ALT+b)
- **Run output windows** (ALT+r)

### Where Scrolling is Disabled ❌

- **Vi editing windows** (use vi's native scrolling: Ctrl+U, Ctrl+D, etc.)

## How to Use

### Basic Scrolling

1. Position mouse cursor over the window you want to scroll
2. Use mouse wheel:
   - **Scroll Up** (wheel up) - View older output (3 lines per scroll)
   - **Scroll Down** (wheel down) - View newer output (3 lines per scroll)

### Examples

**Example 1: Scrolling through command output**
```bash
./ide
find /usr/bin -name "*.so"  # Long output
# Scroll up to see earlier results
# Scroll down to return to latest output
```

**Example 2: Reviewing build errors**
```bash
./ide
vi examples/test.go
ALT+b                       # Build (if errors)
# Move mouse to build output window
# Scroll up to see all errors
# Click on error line and press Enter to jump to source
```

**Example 3: Multi-window scrolling**
```bash
./ide
ALT+h                       # Split horizontally
# In window 1: ls -la /usr/bin
# In window 2: ls -la /etc
# Move mouse to window 1, scroll to see different parts
# Move mouse to window 2, scroll independently
```

## Technical Details

### Scroll Speed
- Each wheel notch scrolls **3 lines**
- Smooth and responsive scrolling experience

### Scroll Limits
- **Maximum scroll up**: Limited by buffer size (1MB)
- **Maximum scroll down**: Always returns to latest output (scrollback = 0)

### Window Detection
- IDE automatically detects which window the mouse is over
- Scrolling only affects the window under the mouse cursor
- No need to click or change focus - just hover and scroll

### Vi Mode Behavior
When editing a file in vi:
- Mouse scrolling is **disabled**
- Use vi's native scrolling commands:
  - `Ctrl+U` - Scroll up half page
  - `Ctrl+D` - Scroll down half page
  - `Ctrl+B` - Scroll up full page
  - `Ctrl+F` - Scroll down full page
  - `gg` - Go to top
  - `G` - Go to bottom

This ensures vi's normal behavior is preserved.

## Keyboard Alternatives

If you prefer keyboard-only navigation:
- Currently: Use `less` or pipe commands through pagers
- Example: `ls -la | less` then use arrow keys

## Use Cases

### 1. Long Command Output
```bash
ls -lR /                    # Recursive listing
# Scroll up to see earlier directories
```

### 2. Log File Review
```bash
cat /var/log/syslog         # View log
# Scroll to review different parts
```

### 3. Build Error Investigation
```bash
vi large_project.go
ALT+b                       # Build with many errors
# Scroll through all errors
# Jump to each error location
```

### 4. Comparing Output Across Windows
```bash
./ide
ALT+v                       # Split vertically
# Window 1: ls -la /home
# Window 2: ls -la /tmp
# Scroll each independently to compare
```

## Benefits

✅ **Natural interaction** - Use mouse wheel as expected
✅ **Context-aware** - Automatically detects target window
✅ **Vi-compatible** - Doesn't interfere with vi editing
✅ **Multi-window** - Each window scrolls independently
✅ **Responsive** - Smooth 3-line scrolling increments
