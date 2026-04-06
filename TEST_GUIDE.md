# Testing Guide for IDE Fixes

## Test 1: Shell Command Output
**What to test:** Shell commands display correctly with proper line wrapping

```
1. Run: ./ide
2. Type: ls -l
3. Expected: Full output displayed, properly formatted
4. Type: ls
5. Expected: Files listed cleanly, no broken edges
```

## Test 2: Window Split and Shell Resizing
**What to test:** Shell adapts to new window size after split

```
1. Run: ./ide
2. Type: ls -l
   - Note the output format
3. Press: ALT+h (split horizontally)
4. Type: ls -l (in new window)
   - Expected: Output adjusts to narrower width
5. Press: ALT+v (split vertically)
6. Type: ls -l
   - Expected: Output adjusts to even narrower width
7. Press: ALT+x (close window)
8. Type: ls -l
   - Expected: Output adjusts back to wider width
```

## Test 3: Long Command Output
**What to test:** No pause/hang when displaying long output

```
1. Run: ./ide
2. Type: find /usr -name "*.so" 2>/dev/null | head -50
3. Expected: Output flows smoothly, no need to press ENTER
4. Type: cat /etc/services
5. Expected: Full file displays (may need to scroll)
```

## Test 4: Vi Window Splitting
**What to test:** Vi properly sized in split windows

```
1. Run: ./ide
2. Type: vi examples/test.go
3. Press: ALT+h
4. Expected: Both vi windows properly sized
5. Edit in both windows
6. Expected: Status bar shows correct position
```

## Test 5: Window Closing and Space Reclamation
**What to test:** Closed windows' space is reclaimed

```
1. Run: ./ide
2. Press: ALT+h (creates Window 2)
3. In Window 2, Press: ALT+h (creates Window 3)
4. Press: ALT+x (close Window 3)
5. Expected: Window 2 reclaims full space, commands display full width
6. Type: ls -l
7. Expected: Output uses full available width
```

## Expected Behaviors

✅ **Shell commands** should display cleanly without artifacts
✅ **Window splits** should immediately resize shell to new dimensions
✅ **Long output** should flow continuously without pausing
✅ **Line wrapping** should match window width
✅ **No strange characters** when pressing ENTER
✅ **Space reclamation** works correctly when closing windows
