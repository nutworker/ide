# Quick Test for Latest Fixes

## Issue 1: First files in `ls` output
```bash
./ide
ls
# Expected: ALL files visible, including first ones
# Check: Can you see .git, LICENSE, README.md at the top?
```

## Issue 2: `ls -l` overlapping at bottom
```bash
./ide
ls -l
# Expected: Last line fully visible, no overlap
# Check: Can you see the complete last line without it being cut off?
```

## Additional Tests
```bash
# Test 1: Clear screen behavior
./ide
clear
ls
# Should show all files from top

# Test 2: Multiple commands
./ide
ls
pwd
ls -la
# All output should be visible and properly formatted

# Test 3: Long filename display
./ide
ls -1
# Each filename on its own line, all visible
```

## What Changed
1. **Removed window borders** - they were overlapping with content
2. **Fixed ANSI escape handling** - less aggressive stripping preserves content
3. **Improved line rendering** - shows all available lines when buffer is small
4. **PTY sizing** - correctly accounts for window number line

## Expected Behavior
✅ All `ls` output visible from first to last file
✅ No overlap at bottom of window
✅ Clean rendering with no cut-off text
✅ Proper spacing and formatting
