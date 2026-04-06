# Shell Prompt Design

## Overview

The IDE uses a **simplified shell prompt** to ensure a clean appearance in windows of all sizes, especially after splitting.

## Prompt Format

### Smart Path Prompt

The prompt shows the **current directory** but intelligently shortens it to ~20 characters:

**Short paths (≤20 chars):**
```
~/projects$
/tmp$
~/code/myapp$
```

**Long paths (>20 chars):**
```
~/.../config$
/usr/.../share$
/home/.../very/.../dir$
```

The format is: `first/.../last$` where:
- `first` = first directory (or `~` for home)
- `...` = intermediate directories collapsed
- `last` = current directory name

## Why a Simple Prompt?

### The Problem with Default Prompts

Most Linux distributions use complex prompts like:
```
user@hostname:/very/long/current/directory$
```

**Issues with complex prompts:**
- Takes up 40+ characters
- Gets truncated in narrow windows (after vertical split)
- Looks messy when wrapped
- Reduces usable command space

### Example: Window Size Impact

**Full Screen (80 columns):**
```
user@hostname:/home/user/projects$ ls -la
[plenty of space for command]
```

**After Vertical Split (40 columns):**
```
user@hostname:/home/user/projec  [truncated]
ts$ ls -la  [wrapped, messy]
```

**With Simple Prompt (40 columns):**
```
user$ ls -la
[clean and readable]
```

## Design Principles

### 1. Minimal Width
- Username only (typically 4-8 characters)
- No hostname (unnecessary in local development)
- No path (use `pwd` when needed)
- Total: ~10 characters maximum

### 2. Clear User Indication
- `$` for regular user
- `#` for root user
- Industry standard convention

### 3. Readable at Any Size
- Works in full screen: `user$ `
- Works in narrow split: `user$ `
- Works in multiple splits: `user$ `

### 4. Emacs-Inspired
Similar to Emacs shell-mode which uses simple prompts for the same reasons:
- Consistent behavior across window sizes
- Maximizes usable space
- Clean, professional appearance

## Comparison with Other Approaches

### Default Bash
```
user@host:/long/path$ command
```
- ❌ Too wide for small windows
- ❌ Gets truncated and wrapped
- ✅ Shows current directory

### Zsh (Oh-My-Zsh)
```
➜ ~/projects/myproject git:(main) ✗
```
- ❌ Even wider than bash
- ❌ Special characters may not render in all terminals
- ✅ Rich information display

### Emacs Shell Mode
```
user$
```
- ✅ Minimal width
- ✅ Clean appearance
- ✅ Works at any size
- ❌ Less information

### Our IDE Prompt
```
user$
```
- ✅ Minimal width (same as Emacs)
- ✅ Clean appearance
- ✅ Works at any size
- ✅ Standard user indication ($/#)
- ✅ Easy to customize if needed

## Advanced Usage

### Get Current Directory
```bash
user$ pwd
/home/user/current/directory
```

### Show Full Path in Prompt (Override)
If you prefer the full path, you can override:
```bash
user$ PS1='\u@\h:\w\$ '
user@hostname:/path$
```

### Restore Original Prompt
```bash
user$ source ~/.bashrc
[your normal prompt returns]
```

## Technical Implementation

The IDE overrides the prompt **after** bash starts:

```go
// Wait for bash to initialize and load ~/.bashrc
time.Sleep(100 * time.Millisecond)

// Override the prompt with a simple one
w.PTY.Write([]byte("PS1='$ '\n"))
w.PTY.Write([]byte("clear\n"))
```

**Why after startup?**
- Bash loads ~/.bashrc which sets its own PS1
- By sending the command after, we override any .bashrc settings
- The `clear` command removes startup messages for a clean look
- Your aliases and functions from .bashrc are still loaded

## Benefits

✅ **Clean appearance** - No truncation or wrapping
✅ **Space efficient** - More room for commands and output
✅ **Consistent** - Looks good at any window size
✅ **Professional** - Similar to Emacs and other IDEs
✅ **Predictable** - Always the same width
✅ **Customizable** - Can be changed within the shell if needed

## Examples

### Single Window
```
┌─────────────────────────────────┐
│ user$ ls                        │
│ file1  file2  file3             │
│ user$ pwd                       │
│ /home/user/projects             │
├─────────────────────────────────┤
│[1]                              │
└─────────────────────────────────┘
```

### Vertical Split (Narrow Windows)
```
┌───────────────┬───────────────┐
│ user$ ls      │ user$ pwd     │
│ file1  file2  │ /home/user    │
│ user$ cd src  │ user$ ls      │
│ user$         │ dir1  dir2    │
├───────────────┼───────────────┤
│[1]            │[2]            │
└───────────────┴───────────────┘
```

### Multiple Splits (Very Narrow)
```
┌──────┬──────┬──────┬──────┐
│user$ │user$ │user$ │user$ │
│ls    │pwd   │cd    │make  │
├──────┼──────┼──────┼──────┤
│[1]   │[2]   │[3]   │[4]   │
└──────┴──────┴──────┴──────┘
```

Even in very narrow windows (15 columns), the prompt remains clean and usable!
