package shell

import (
	"testing"

	"github.com/bbmumford/mullion/native"
)

func TestParseShortcutReadsTheFormsAMenuUses(t *testing.T) {
	for _, tc := range []struct {
		in   string
		key  native.KeyCode
		mod  native.Modifiers
		good bool
	}{
		{"cmd+q", native.KeyQ, native.PrimaryMod(), true},
		{"cmd+shift+z", native.KeyZ, native.PrimaryMod() | native.ModShift, true},
		{"ctrl+c", native.KeyC, native.ModCtrl, true},
		{"alt+f4", native.KeyF4, native.ModAlt, true},
		{"opt+left", native.KeyLeft, native.ModAlt, true},
		{"cmd+1", native.Key1, native.PrimaryMod(), true},
		{"CMD+Q", native.KeyQ, native.PrimaryMod(), true},
		{"  cmd+q  ", native.KeyQ, native.PrimaryMod(), true},
		{"escape", native.KeyEscape, 0, true},
		{"cmd+alt+shift+s", native.KeyS, native.PrimaryMod() | native.ModAlt | native.ModShift, true},
		// Rejected rather than silently turned into something else.
		{"", 0, 0, false},
		{"cmd+", 0, 0, false},
		{"hyper+q", 0, 0, false},
		{"cmd+nosuchkey", 0, 0, false},
	} {
		got, ok := ParseShortcut(tc.in)
		if ok != tc.good {
			t.Errorf("ParseShortcut(%q) ok = %v, want %v", tc.in, ok, tc.good)
			continue
		}
		if !tc.good {
			continue
		}
		if got.Key != tc.key || got.Mod != tc.mod {
			t.Errorf("ParseShortcut(%q) = key %d mod %d, want key %d mod %d",
				tc.in, got.Key, got.Mod, tc.key, tc.mod)
		}
	}
}

// "cmd" has to mean the host's primary modifier, or one menu would need a
// different spelling per platform.
func TestCmdMeansTheHostsPrimaryModifier(t *testing.T) {
	got, ok := ParseShortcut("cmd+q")
	if !ok {
		t.Fatal("cmd+q was not understood")
	}
	if got.Mod != native.PrimaryMod() {
		t.Errorf("cmd gave %d, want PrimaryMod() = %d", got.Mod, native.PrimaryMod())
	}
}

func TestShortcutsFlattensSubmenusAndSkipsWhatCannotFire(t *testing.T) {
	var fired string
	m := Menu{
		{Label: "File", Submenu: Menu{
			{Label: "Open", Shortcut: "cmd+o", Do: func() { fired = "open" }},
			{Label: "Close", Shortcut: "cmd+w", Do: func() { fired = "close" }},
			Separator(),
			{Label: "No handler", Shortcut: "cmd+n"},
			{Label: "No shortcut", Do: func() { fired = "nope" }},
			{Label: "Disabled", Shortcut: "cmd+d", Disabled: true, Do: func() { fired = "nope" }},
			{Label: "Unparseable", Shortcut: "hyper+x", Do: func() { fired = "nope" }},
		}},
		// A responder-chain item is answered by the platform; matching it here
		// as well would run the command twice on macOS.
		{Label: "Edit", Submenu: Menu{
			{Label: "Copy", Shortcut: "cmd+c", Selector: "copy:"},
		}},
	}
	got := m.Shortcuts()
	if len(got) != 2 {
		t.Fatalf("collected %d shortcuts, want 2 (open and close): %+v", len(got), got)
	}
	got[0].Do()
	if fired != "open" {
		t.Errorf("the first shortcut ran %q, want open", fired)
	}
	got[1].Do()
	if fired != "close" {
		t.Errorf("the second shortcut ran %q, want close", fired)
	}
}

// Two items on one combo resolve the way a menu bar resolves them: the first in
// the menu wins, rather than both firing.
func TestTheFirstItemOnACombloWins(t *testing.T) {
	var fired []string
	m := Menu{
		{Label: "First", Shortcut: "cmd+k", Do: func() { fired = append(fired, "first") }},
		{Label: "Second", Shortcut: "cmd+k", Do: func() { fired = append(fired, "second") }},
	}
	got := m.Shortcuts()
	if len(got) != 2 {
		t.Fatalf("collected %d, want both recorded so the order is visible", len(got))
	}
	if got[0].Combo != got[1].Combo {
		t.Fatal("the fixture no longer has two items on one combo")
	}
	// Match fires at most one, and it is the first.
	for _, s := range got {
		if s.Combo == got[0].Combo {
			s.Do()
			break
		}
	}
	if len(fired) != 1 || fired[0] != "first" {
		t.Errorf("fired %v, want only first", fired)
	}
}

func TestShortcutsOnAnEmptyMenuIsEmpty(t *testing.T) {
	if got := (Menu{}).Shortcuts(); len(got) != 0 {
		t.Errorf("collected %d from an empty Menu", len(got))
	}
}

// Match reads the frame's input. With no key pressed there is nothing to match,
// and it must not fire anything.
func TestMatchFiresNothingWithoutAKeyThisFrame(t *testing.T) {
	fired := false
	shortcuts := []Shortcut{{
		Combo: native.Combo(native.KeyQ, native.PrimaryMod()),
		Do:    func() { fired = true },
	}}
	if Match(shortcuts) {
		t.Error("Match reported a hit with no key pressed")
	}
	if fired {
		t.Error("Match ran a handler with no key pressed")
	}
}
