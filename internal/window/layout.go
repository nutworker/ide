package window

// Layout manages the window layout using a binary tree
type Layout struct {
	Root *LayoutNode
}

// LayoutNode represents a node in the layout tree
type LayoutNode struct {
	Window   *Window
	Split    SplitType
	Children [2]*LayoutNode
	Parent   *LayoutNode
	Rect     Rect
}

// NewLayout creates a new layout with a single window
func NewLayout(window *Window) *Layout {
	return &Layout{
		Root: &LayoutNode{
			Window: window,
			Split:  SplitNone,
			Rect:   window.Rect,
		},
	}
}

// FindNode finds the node containing the given window ID
func (l *Layout) FindNode(windowID int) *LayoutNode {
	return l.findNodeRecursive(l.Root, windowID)
}

func (l *Layout) findNodeRecursive(node *LayoutNode, windowID int) *LayoutNode {
	if node == nil {
		return nil
	}

	if node.Window != nil && node.Window.ID == windowID {
		return node
	}

	if node.Split != SplitNone {
		if found := l.findNodeRecursive(node.Children[0], windowID); found != nil {
			return found
		}
		return l.findNodeRecursive(node.Children[1], windowID)
	}

	return nil
}

// SplitNode splits a node containing a window
func (l *Layout) SplitNode(node *LayoutNode, splitType SplitType, newWindow *Window) error {
	if node.Window == nil {
		return nil
	}

	// Calculate new rectangles
	rect1, rect2 := splitRect(node.Rect, splitType)

	// Update existing window rect
	node.Window.Rect = rect1
	node.Window.ResizePTY()

	// Set new window rect
	newWindow.Rect = rect2

	// Convert node to internal node
	node.Split = splitType
	node.Children[0] = &LayoutNode{
		Window: node.Window,
		Split:  SplitNone,
		Rect:   rect1,
		Parent: node,
	}
	node.Children[1] = &LayoutNode{
		Window: newWindow,
		Split:  SplitNone,
		Rect:   rect2,
		Parent: node,
	}
	node.Window = nil

	return nil
}

// splitRect splits a rectangle based on split type
func splitRect(rect Rect, splitType SplitType) (Rect, Rect) {
	switch splitType {
	case SplitHorizontal:
		// Split horizontally (top/bottom)
		height1 := rect.Height / 2
		height2 := rect.Height - height1
		return Rect{
				X:      rect.X,
				Y:      rect.Y,
				Width:  rect.Width,
				Height: height1,
			}, Rect{
				X:      rect.X,
				Y:      rect.Y + height1,
				Width:  rect.Width,
				Height: height2,
			}

	case SplitVertical:
		// Split vertically (left/right)
		width1 := rect.Width / 2
		width2 := rect.Width - width1
		return Rect{
				X:      rect.X,
				Y:      rect.Y,
				Width:  width1,
				Height: rect.Height,
			}, Rect{
				X:      rect.X + width1,
				Y:      rect.Y,
				Width:  width2,
				Height: rect.Height,
			}

	default:
		return rect, rect
	}
}

// ResizeAll recursively resizes all windows to fit new screen dimensions
func (l *Layout) ResizeAll(newRect Rect) {
	if l.Root != nil {
		l.Root.Rect = newRect
		l.resizeNodeRecursive(l.Root, newRect)
	}
}

func (l *Layout) resizeNodeRecursive(node *LayoutNode, rect Rect) {
	if node == nil {
		return
	}

	node.Rect = rect

	if node.Window != nil {
		// Leaf node - update window
		node.Window.Rect = rect
		node.Window.ResizePTY()
	} else if node.Split != SplitNone {
		// Internal node - split and recurse
		rect1, rect2 := splitRect(rect, node.Split)
		l.resizeNodeRecursive(node.Children[0], rect1)
		l.resizeNodeRecursive(node.Children[1], rect2)
	}
}

// RemoveWindow removes a window and reclaims its space
func (l *Layout) RemoveWindow(windowID int) *Window {
	node := l.FindNode(windowID)
	if node == nil || node.Window == nil {
		return nil
	}

	parent := node.Parent
	if parent == nil {
		// Cannot remove root window
		return nil
	}

	// Find sibling (the other child of parent)
	var sibling *LayoutNode
	if parent.Children[0] == node {
		sibling = parent.Children[1]
	} else {
		sibling = parent.Children[0]
	}

	// Get the window that will reclaim space (from sibling subtree)
	var siblingWindow *Window
	if sibling.Window != nil {
		siblingWindow = sibling.Window
	}

	// Replace parent with sibling (sibling takes parent's space)
	parent.Window = sibling.Window
	parent.Split = sibling.Split
	parent.Children = sibling.Children

	// Update parent references for children
	if parent.Split != SplitNone {
		if parent.Children[0] != nil {
			parent.Children[0].Parent = parent
		}
		if parent.Children[1] != nil {
			parent.Children[1].Parent = parent
		}
	}

	// Resize the sibling (or its subtree) to fill parent's space
	l.resizeNodeRecursive(parent, parent.Rect)

	return siblingWindow
}

// GetAllWindows returns all windows in the layout
func (l *Layout) GetAllWindows() []*Window {
	var windows []*Window
	l.collectWindows(l.Root, &windows)
	return windows
}

func (l *Layout) collectWindows(node *LayoutNode, windows *[]*Window) {
	if node == nil {
		return
	}

	if node.Window != nil {
		*windows = append(*windows, node.Window)
	}

	if node.Split != SplitNone {
		l.collectWindows(node.Children[0], windows)
		l.collectWindows(node.Children[1], windows)
	}
}
