# Border and Status Bar Improvements

## Changes Made

### 1. Fixed Vertical Borders
**Issue**: Borders not showing between vertically split windows

**Fix**:
- Border now drawn at rightmost column of window (`X + Width - 1`)
- Checks if there's space for another window to the right
- Properly excludes status bar area

### 2. Compact Status Bar
**Before**:
```
 [1]                              (shell mode)
 [1] test.go [12,45] -- INSERT -- (vi mode)
```

**After**:
```
[1]                               (shell mode - more compact)
[1] test.go [12,45] --INSERT--    (vi mode - tighter spacing)
```

## Visual Examples

### Horizontal Split
```
┌─────────────────────────────────┐
│ Window 1 content                │
│ $ ls                            │
├─────────────────────────────────┤  ← Border
│[1]                              │  ← Compact status bar
├─────────────────────────────────┤  ← Border
│ Window 2 content                │
│ $ pwd                           │
├─────────────────────────────────┤
│[2]                              │
└─────────────────────────────────┘
```

### Vertical Split
```
┌────────────────┬────────────────┐
│ Window 1       │ Window 2       │
│ $ ls           │ $ pwd          │
│                │                │
│                │                │
├────────────────┼────────────────┤
│[1]             │[2]             │  ← Compact status bars
└────────────────┴────────────────┘
         ↑
    Vertical border now visible
```

### Mixed Split (Horizontal then Vertical)
```
┌────────────────┬────────────────┐
│ Window 1       │ Window 2       │
│                │                │
├────────────────┼────────────────┤
│[1]             │[2]             │
├────────────────┴────────────────┤  ← Horizontal border
│ Window 3                        │
│                                 │
├─────────────────────────────────┤
│[3]                              │
└─────────────────────────────────┘
```

### Vi Mode with Compact Status
```
┌─────────────────────────────────┐
│ package main                    │
│                                 │
│ func main() {                   │
│     fmt.Println("Hello")        │
│ }                               │
├─────────────────────────────────┤
│[1] test.go [3,5] --INSERT--     │  ← More compact
└─────────────────────────────────┘
```

## Test Commands

```bash
# Test vertical borders
./ide
ALT+v           # Split vertically
# Expected: See │ border between windows

# Test horizontal borders
./ide
ALT+h           # Split horizontally
# Expected: See ─ border between windows

# Test compact status bar
./ide
# Expected: See [1] at bottom (no extra spaces)

vi examples/test.go
# Expected: See [1] test.go [1,1] --COMMAND-- (compact)

# Test complex layout
./ide
ALT+h           # Split horizontal
ALT+v           # Split vertical
# Expected: See both │ and ─ borders clearly
```

## Benefits

✅ **Clear window separation** - borders visible in all split configurations
✅ **Compact status bar** - more space for content
✅ **Consistent appearance** - borders don't overlap content
✅ **Better visual hierarchy** - easier to see window boundaries
