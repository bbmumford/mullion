package widgets

// Added for mullion; see tree.go.

import "testing"

// TreeState is the caller's, so its behaviour is the part worth pinning: a
// tree that forgets what was expanded is worse than one that never expanded.

func TestTreeStateZeroValueIsUsable(t *testing.T) {
	var s TreeState
	if s.IsOpen("anything") {
		t.Error("a zero TreeState reported a node open")
	}
	// Opening from nil has to allocate rather than panic.
	s.SetOpen("a", true)
	if !s.IsOpen("a") {
		t.Error("SetOpen on a zero TreeState did not take")
	}
}

// Closing a node that was never open must not allocate a map to hold nothing,
// because a tree walked without expanding anything should cost nothing.
func TestClosingAnUnopenedNodeAllocatesNothing(t *testing.T) {
	var s TreeState
	s.SetOpen("a", false)
	if s.Open != nil {
		t.Error("closing an unopened node allocated the map")
	}
}

func TestToggleReportsAndAppliesTheNewState(t *testing.T) {
	var s TreeState
	if got := s.Toggle("a"); !got {
		t.Error("the first Toggle returned false, want open")
	}
	if !s.IsOpen("a") {
		t.Error("Toggle said open but IsOpen disagrees")
	}
	if got := s.Toggle("a"); got {
		t.Error("the second Toggle returned true, want closed")
	}
	if s.IsOpen("a") {
		t.Error("Toggle said closed but IsOpen disagrees")
	}
}

// A closed node is removed rather than stored as false, so the map holds what
// is open and nothing else. A tree browsed for an hour should not carry a key
// for every folder ever touched.
func TestClosingRemovesTheKey(t *testing.T) {
	var s TreeState
	s.SetOpen("a", true)
	s.SetOpen("b", true)
	s.SetOpen("a", false)
	if _, present := s.Open["a"]; present {
		t.Error("a closed node is still in the map")
	}
	if len(s.Open) != 1 {
		t.Errorf("the map holds %d keys, want only the open one", len(s.Open))
	}
}

// Tree asks for children only where it needs them: the root always, and a node
// only while it is open. A tree over a filesystem reads the directories that
// are showing and no others, which is the difference between browsing a disk
// and walking it.
func TestTreeAsksOnlyForTheRootAndTheOpenNodes(t *testing.T) {
	state := &TreeState{}
	state.SetOpen("a", true)

	var asked []string
	children := func(key string) []TreeNode[string] {
		asked = append(asked, key)
		switch key {
		case "":
			return []TreeNode[string]{
				{Key: "a", Label: "A", HasChildren: true},
				{Key: "b", Label: "B", HasChildren: true}, // closed: not asked
			}
		case "a":
			return []TreeNode[string]{{Key: "a/1", Label: "A1"}}
		}
		return nil
	}

	// Walked without a frame: the traversal is what is being checked, and it
	// is the same walk the drawing does.
	walkTree(state, children)

	if len(asked) != 2 || asked[0] != "" || asked[1] != "a" {
		t.Errorf("asked for %v, want the root and the open node only", asked)
	}
}

// walkTree is the traversal Tree performs, without the drawing, so the "ask
// only where needed" rule can be checked without a window.
func walkTree[T any](state *TreeState, children func(string) []TreeNode[T]) {
	var level func(key string)
	level = func(key string) {
		for _, node := range children(key) {
			if node.HasChildren && state.IsOpen(node.Key) {
				level(node.Key)
			}
		}
	}
	level("")
}
