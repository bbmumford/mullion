package widgets

// Added for mullion; not part of the merged go-mullion snapshot. Kept in a file
// of its own so a later re-sync with upstream can see at a glance what is
// local.

import (
	. "github.com/bbmumford/mullion/native"
)

// A TreeNode is one row of a Tree.
//
// Children is read only for a node that is open, so a large tree can be built
// lazily: return the children of the node being asked about and nothing else.
type TreeNode[T any] struct {
	// Key identifies the node across frames, and is what the open set and the
	// selection are keyed by. It has to be stable and unique within the tree;
	// a path is the usual thing.
	Key string

	// Label is the text shown for the row.
	Label string

	// Icon is drawn before the label. NoIcon for none.
	Icon IconGlyph

	// Value is whatever the caller wants back when the row is chosen.
	Value T

	// HasChildren says whether to draw a disclosure triangle. It is separate
	// from the children themselves so a directory can show as expandable
	// without being read first.
	HasChildren bool
}

// TreeAttrs tunes a tree. The zero value is the ordinary one.
type TreeAttrs struct {
	// Indent is how far each level is inset, in points. Zero takes 16.
	Indent float32

	// RowHeight is the height of a row, in points. Zero takes 22.
	RowHeight float32
}

// TreeState is which rows are open and which is selected. The caller owns it,
// so an expanded tree survives a rebuild and can be saved with the window.
//
// The zero value works: nothing open, nothing selected.
type TreeState struct {
	// Open holds the keys of the expanded nodes. Nil is treated as empty and
	// filled in on the first expansion.
	Open map[string]bool

	// Selected is the key of the chosen row, empty for none.
	Selected string
}

// IsOpen reports whether the node with this key is expanded.
func (s *TreeState) IsOpen(key string) bool { return s.Open[key] }

// SetOpen expands or collapses a node.
func (s *TreeState) SetOpen(key string, open bool) {
	if s.Open == nil {
		if !open {
			return
		}
		s.Open = map[string]bool{}
	}
	if open {
		s.Open[key] = true
		return
	}
	delete(s.Open, key)
}

// Toggle flips a node between open and closed and reports the new state.
func (s *TreeState) Toggle(key string) bool {
	open := !s.IsOpen(key)
	s.SetOpen(key, open)
	return open
}

// Tree draws a hierarchy with disclosure triangles, and reports the value of
// the row chosen this frame.
//
// children is asked for the children of a key, and called only for the root
// and for nodes that are open, so a tree over a filesystem reads the
// directories that are showing and no others. The root's children are asked
// for with an empty key.
//
// Clicking a row selects it; clicking the triangle, or pressing Space or Enter
// on a focused row, expands or collapses it.
//
//	chosen, ok := widgets.Tree(&state, widgets.TreeAttrs{}, func(key string) []widgets.TreeNode[string] {
//	    return entriesUnder(key)
//	})
func Tree[T any](state *TreeState, attrs TreeAttrs, children func(key string) []TreeNode[T]) (chosen T, ok bool) {
	var zero T
	if state == nil || children == nil {
		return zero, false
	}
	if attrs.Indent == 0 {
		attrs.Indent = 16
	}
	if attrs.RowHeight == 0 {
		attrs.RowHeight = 22
	}
	scheme := CurrentColorScheme
	Container(Attrs(Expand), func() {
		chosen, ok = treeLevel(state, attrs, children, "", 0, scheme)
	})
	return chosen, ok
}

// treeLevel draws the children of one key and recurses into the open ones.
func treeLevel[T any](state *TreeState, attrs TreeAttrs, children func(string) []TreeNode[T],
	key string, depth int, scheme ColorScheme) (chosen T, ok bool) {
	var zero T
	chosen = zero

	for _, node := range children(key) {
		open := state.IsOpen(node.Key)
		if hit, got := treeRow(state, attrs, node, depth, open, scheme); got {
			chosen, ok = hit, true
		}
		if node.HasChildren && state.IsOpen(node.Key) {
			if hit, got := treeLevel(state, attrs, children, node.Key, depth+1, scheme); got {
				chosen, ok = hit, true
			}
		}
	}
	return chosen, ok
}

// treeRow draws one row and reports whether it was chosen.
func treeRow[T any](state *TreeState, attrs TreeAttrs, node TreeNode[T], depth int,
	open bool, scheme ColorScheme) (chosen T, ok bool) {
	selected := state.Selected == node.Key
	indent := comfort(attrs.Indent) * float32(depth)

	Container(Attrs(Row, Expand, CrossMid, Gap(comfort(4)),
		FixHeight(comfort(attrs.RowHeight)), Pad4(0, comfort(6), 0, comfort(6)+indent),
		Corners(comfort(3))), func() {
		st := ProcessButtonEvents(false)
		NextAccessRole("treeitem")
		NextAccessName(node.Label)
		if node.HasChildren {
			NextAccessChecked(open)
		}
		AssignAccess()

		// The list style already says what a selected and a hovered row look
		// like, so a tree row and a list row agree without either being told
		// about the other.
		switch {
		case selected:
			ModAttrs(BackgroundVec(scheme.List.Selected.Background),
				AmendTextStyle(TextColorVec(scheme.List.Selected.Text)))
		case st.Hovered || st.FocusVisible:
			ModAttrs(BackgroundVec(scheme.List.Hovered.Background),
				AmendTextStyle(TextColorVec(scheme.List.Hovered.Text)))
		default:
			ModAttrs(AmendTextStyle(TextColorVec(scheme.List.Surface.Text)))
		}

		if st.Clicked {
			// A click on the triangle expands; anywhere else selects. The
			// triangle's width is the indent, which is where it is drawn.
			triangleEdge := comfort(6) + indent + comfort(attrs.Indent)
			if node.HasChildren && st.Local[0] <= triangleEdge {
				state.Toggle(node.Key)
			} else {
				state.Selected = node.Key
				chosen, ok = node.Value, true
			}
			RequestNextFrame()
		}

		// The triangle occupies its slot whether or not the node has children,
		// so labels at one depth line up rather than stepping in and out with
		// whether each node happens to be expandable.
		Container(Attrs(FixWidth(comfort(attrs.Indent)), CrossMid), func() {
			if node.HasChildren {
				// A solid triangle both ways, rotated: a long arrow beside a
				// filled triangle reads as two different controls.
				glyph := TypMediaPlay
				if open {
					glyph = TypArrowSortedDown
				}
				Icon(glyph, FontSize(comfort(10)))
			}
		})
		if node.Icon != NoIcon {
			Icon(node.Icon, FontSize(comfort(12)))
		}
		Label(node.Label, FontSize(comfort(12)))
	})
	return chosen, ok
}
