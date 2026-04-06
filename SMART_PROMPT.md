# Smart Prompt System

## Overview

The IDE uses a **smart prompt** that shows your current directory while ensuring it never gets truncated or wrapped, even in narrow windows.

## How It Works

### Path Shortening Algorithm

1. **Replace home with ~**: `/home/username` → `~`
2. **Check length**: If ≤ 20 characters, show full path
3. **If too long**: Show `first/.../last`
   - `first` = root or ~
   - `...` = collapsed middle directories
   - `last` = current directory name

### Examples

| Full Path | Shortened (20 char limit) | Length |
|-----------|---------------------------|--------|
| `/tmp` | `/tmp$ ` | 6 |
| `~/projects` | `~/projects$ ` | 13 |
| `~/code/myapp` | `~/code/myapp$ ` | 15 |
| `~/projects/work/client` | `~/.../client$ ` | 15 |
| `/home/user/very/long/path` | `/home/.../path$ ` | 17 |
| `/usr/local/share/applications` | `/usr/.../applications$ ` | 24 |
| `~/.config/systemd/user/default.target.wants` | `~/.../default.target.wants$ ` | 31 |

## Visual Examples

### Full Screen
```
┌────────────────────────────────────┐
│ ~/projects$ ls                     │
│ app1  app2  app3                   │
│ ~/projects$ cd app1                │
│ ~/projects/app1$ make              │
├────────────────────────────────────┤
│[1]                                 │
└────────────────────────────────────┘
```

### After Vertical Split (Narrow)
```
┌──────────────┬──────────────┐
│ ~$ ls        │ ~$ cd code   │
│ code  docs   │ ~/code$ ls   │
│ ~$ cd docs   │ proj1  proj2 │
│ ~/docs$ pwd  │ ~/code$ cd   │
├──────────────┼──────────────┤
│[1]           │[2]           │
└──────────────┴──────────────┘
```

### Long Path Handling
```
┌──────────────────────────────┐
│ ~/code/project$ cd src/main  │
│ ~/.../main$ ls               │
│ file1.go  file2.go           │
│ ~/.../main$ cd ../lib        │
│ ~/.../lib$ pwd               │
│ /home/user/code/project/lib  │
├──────────────────────────────┤
│[1]                           │
└──────────────────────────────┘
```

## Benefits of Smart Shortening

### ✅ Context Awareness
- Always shows **where you are** (current directory)
- Shows **where you started** (root or home)
- Collapsed middle is usually less important

### ✅ Predictable Width
- Never exceeds ~25 characters
- No wrapping in narrow windows
- Clean appearance at any size

### ✅ More Informative than "$ "
- Compare: `$ ` (no context)
- To: `~/.../config$ ` (clear location)

### ✅ Better than Full Path
- Compare: `/home/username/projects/work/client/src/components$ ` (60+ chars, truncated)
- To: `~/.../components$ ` (18 chars, clean)

## Technical Implementation

The IDE injects a bash function on startup:

```bash
_ide_prompt() {
  local dir="$PWD"
  # Replace home with ~
  dir="${dir/#$HOME/~}"

  # If longer than 20 chars, shorten it
  if [ ${#dir} -gt 20 ]; then
    local first="${dir%%/*}"  # First directory
    local last="${dir##*/}"   # Current directory

    # Keep ~ for home
    if [[ "$dir" == ~* ]]; then
      first="~"
    fi

    # Build: first/.../last
    if [ "$first" = "$last" ]; then
      dir="$first"
    else
      dir="$first/.../$last"
    fi
  fi

  PS1="$dir\$ "
}

# Run before each prompt
PROMPT_COMMAND=_ide_prompt
```

## Silent Setup

The setup commands are **not echoed** to the terminal:
- No "PS1=..." messages visible
- No "clear" message visible
- Clean startup experience
- Function definitions are silent

## Comparison with Other Approaches

### Default Bash
```
user@host:/very/long/path/to/current/directory$
```
❌ 50+ characters
❌ Truncates in narrow windows
❌ Includes unnecessary hostname

### Just "$"
```
$
```
✅ Never truncates
❌ No context about location
❌ Need to run `pwd` frequently

### Our Smart Prompt
```
~/.../directory$
```
✅ Shows location context
✅ Never truncates
✅ Predictable width
✅ Clean in narrow windows

## Edge Cases Handled

### Root Directory
```
/$
```
Clean and simple.

### Home Directory
```
~$
```
Standard convention.

### One Level Deep
```
~/code$
/tmp$
```
Shows full path (under 20 chars).

### Very Deep Nesting
```
~/.../final-dir$
```
Always shows first and last, collapses middle.

### Directory Names with Spaces
```
~/.../My Documents$
```
Properly handled with quotes if needed.

## User Customization

To change the prompt within a session:
```bash
~/.../dir$ PS1='custom> '
custom>
```

To disable smart shortening:
```bash
~$ PROMPT_COMMAND=''
~$ PS1='\w\$ '
/full/path/shown$
```

To restore:
```bash
$ PROMPT_COMMAND=_ide_prompt
~/.../shown$
```

## Why 20 Characters?

- ✅ Fits in 30-column windows (after split)
- ✅ Leaves room for commands
- ✅ Shows enough context (first + last dir)
- ✅ Industry standard for compact prompts

Similar to:
- Emacs shell-mode default
- VS Code integrated terminal in narrow mode
- Vim terminal mode default
