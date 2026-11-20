package shell

import (
	"strings"

	"github.com/bbmumford/mullion/native"
)

// A Shortcut is a menu accelerator resolved against the Go-drawn surface's
// keyboard input.
//
// It exists because a menu bar is not available everywhere the menu is. macOS
// installs a real bar and AppKit matches the accelerators itself; Windows
// attaches a bar to a window; Linux draws no bar at all, and the bar that would
// have carried the shortcuts needs a GtkWindow, which a Go-drawn window is not.
// Without this, a Go-drawn application on Linux had a Menu whose Shortcut
// fields did nothing, and the same application on macOS answered them -- the
// kind of difference that looks like a platform bug to whoever hits it.
//
// Matching here makes the accelerators work on every platform the Go-drawn
// surface reaches, including the ones with no menu bar in them at all.
type Shortcut struct {
	// Combo is what has to be pressed.
	Combo native.KeyCombo

	// Do is the item's handler.
	Do func()
}

// Shortcuts flattens a Menu into the accelerators it declares, sub-menus
// included. An item with no Shortcut, no Do, or one the platform's own menu
// will answer is left out.
//
// The order is the menu's own, depth first, so when two items declare the same
// combo the first one in the menu wins -- the same rule a menu bar applies.
func (m Menu) Shortcuts() []Shortcut {
	var out []Shortcut
	m.collectShortcuts(&out)
	return out
}

func (m Menu) collectShortcuts(out *[]Shortcut) {
	for _, it := range m {
		if len(it.Submenu) > 0 {
			it.Submenu.collectShortcuts(out)
			continue
		}
		// A Selector item is wired to the responder chain and answered by the
		// platform; matching it here as well would fire it twice.
		if it.Separator || it.Disabled || it.Shortcut == "" || it.Do == nil || it.Selector != "" {
			continue
		}
		combo, ok := ParseShortcut(it.Shortcut)
		if !ok {
			continue
		}
		*out = append(*out, Shortcut{Combo: combo, Do: it.Do})
	}
}

// ParseShortcut reads an accelerator such as "cmd+q" or "cmd+shift+z" into a
// combo, and reports whether it understood it.
//
// The modifiers are cmd, ctrl, alt (or opt), shift and super, and the last
// token is the key. "cmd" means the host's primary shortcut modifier, so one
// spelling follows the platform: Command on macOS and Control elsewhere, which
// is what the menu bar backends already do with it.
func ParseShortcut(s string) (native.KeyCombo, bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(s)), "+")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return native.KeyCombo{}, false
	}
	var mod native.Modifiers
	for _, p := range parts[:len(parts)-1] {
		switch strings.TrimSpace(p) {
		case "cmd", "command":
			mod |= native.PrimaryMod()
		case "ctrl", "control":
			mod |= native.ModCtrl
		case "alt", "opt", "option":
			mod |= native.ModAlt
		case "shift":
			mod |= native.ModShift
		case "super", "win", "meta":
			mod |= native.ModSuper
		default:
			return native.KeyCombo{}, false
		}
	}
	key, ok := parseKey(strings.TrimSpace(parts[len(parts)-1]))
	if !ok {
		return native.KeyCombo{}, false
	}
	return native.Combo(key, mod), true
}

// parseKey reads the key half of an accelerator.
func parseKey(s string) (native.KeyCode, bool) {
	if len(s) == 1 {
		c := s[0]
		switch {
		case c >= 'a' && c <= 'z':
			// The key codes follow the ASCII table from upper case.
			return native.KeyA + native.KeyCode(c-'a'), true
		case c >= '0' && c <= '9':
			return native.Key0 + native.KeyCode(c-'0'), true
		}
	}
	named := map[string]native.KeyCode{
		"enter":     native.KeyEnter,
		"return":    native.KeyEnter,
		"escape":    native.KeyEscape,
		"esc":       native.KeyEscape,
		"tab":       native.KeyTab,
		"space":     native.KeySpace,
		"backspace": native.KeyDeleteBackward,
		"delete":    native.KeyDeleteForward,
		"home":      native.KeyHome,
		"end":       native.KeyEnd,
		"insert":    native.KeyInsert,
		"up":        native.KeyUp,
		"down":      native.KeyDown,
		"left":      native.KeyLeft,
		"right":     native.KeyRight,
		"f1":        native.KeyF1,
		"f2":        native.KeyF2,
		"f3":        native.KeyF3,
		"f4":        native.KeyF4,
		"f5":        native.KeyF5,
		"f6":        native.KeyF6,
		"f7":        native.KeyF7,
		"f8":        native.KeyF8,
		"f9":        native.KeyF9,
		"f10":       native.KeyF10,
		"f11":       native.KeyF11,
		"f12":       native.KeyF12,
	}
	k, ok := named[s]
	return k, ok
}

// Match fires the first shortcut whose combo is the one pressed this frame, and
// reports whether it did. Call it from a frame function; it reads the Go-drawn
// surface's input for the frame and nothing else.
func Match(shortcuts []Shortcut) bool {
	active := native.ActiveCombo()
	if active.Key == native.KeyCodeNone {
		return false
	}
	for _, s := range shortcuts {
		if s.Combo == active && s.Do != nil {
			s.Do()
			return true
		}
	}
	return false
}
