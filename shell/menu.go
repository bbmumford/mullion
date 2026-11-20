package shell

import (
	"github.com/bbmumford/mullion/icon"
	"github.com/bbmumford/mullion/internal/menubar"
	"github.com/bbmumford/mullion/internal/tray"
)

// A Menu is an ordered list of commands. The same Menu can be given to a Tray
// and to SetMenuBar; each surface reads the fields it can honour.
type Menu []Item

// An Item is one command. Separator makes a divider and ignores every other
// field; a non-empty Submenu makes the item open a sub-menu and ignores Do.
//
// Three fields are honoured by one surface only, because the other platform
// surface has nowhere to put them. They are kept on the one Item rather than
// split across two types so that a command is still described once.
type Item struct {
	// Label is the visible text. In a menu bar on macOS the first top-level
	// item is the application menu and the system replaces its label with the
	// application's name.
	Label string

	// Do runs when the item is chosen. It runs on the main thread, so it may
	// touch window state directly; work that blocks belongs on a goroutine.
	Do func()

	// Icon is drawn beside the label in a tray menu. A menu bar ignores it:
	// AppKit's main menu takes no per-item image, and the Win32 menu bar this
	// package installs is text-only.
	Icon icon.Icon

	// Shortcut is a menu-bar accelerator such as "cmd+q" or "cmd+shift+z"; the
	// modifiers are cmd, ctrl, alt (opt) and shift, and the last token is the
	// key. A tray menu ignores it -- a notification-area menu has no
	// accelerator in any of the three platform APIs.
	Shortcut string

	// Checked draws the item with a tick. Set Checkbox as well: an item that is
	// merely Checked but not a checkbox has nothing to toggle.
	Checked bool

	// Checkbox makes the item a toggle. Its Do runs after the tick changes, and
	// Checked is the state it starts in.
	Checkbox bool

	// Disabled greys the item out and stops Do from running.
	Disabled bool

	// Separator makes this item a divider.
	Separator bool

	// Submenu, when non-empty, makes this item open a sub-menu.
	Submenu Menu

	// Selector wires a macOS menu-bar item to an Objective-C action sent down
	// the responder chain instead of to Do: "undo:", "redo:", "cut:", "copy:",
	// "paste:" and "selectAll:" give an Edit menu whose shortcuts reach the
	// focused view, including a WebView's text fields, which a Go callback
	// cannot do. An item with a Selector ignores Do. Elsewhere it is skipped,
	// since there the focused control already handles those keys.
	Selector string

	// Services makes this macOS menu-bar item the system Services menu, filled
	// by AppKit from the current selection. Submenu and Do are ignored, and the
	// item is skipped on other platforms.
	Services bool
}

// Separator is a divider, for writing a Menu as a literal.
func Separator() Item { return Item{Separator: true} }

// trayMenu converts to the notification-area menu model. Shortcut, Selector and
// Services have no place there and are dropped; see the fields for why.
func (m Menu) trayMenu() *tray.Menu {
	out := tray.NewMenu()
	for _, it := range m {
		switch {
		case it.Separator:
			out.AddSeparator()
		case len(it.Submenu) > 0:
			sub := out.AddSubmenu(it.Label, it.Submenu.trayMenu())
			sub.SetDisabled(it.Disabled)
		case it.Checkbox:
			cb := out.AddCheckbox(it.Label, it.Checked, it.Do)
			cb.SetDisabled(it.Disabled)
		case !it.Icon.Empty():
			added := out.AddWithIcon(it.Label, it.Icon.Data, it.Do)
			added.SetDisabled(it.Disabled)
		default:
			added := out.Add(it.Label, it.Do)
			added.SetChecked(it.Checked)
			added.SetDisabled(it.Disabled)
		}
	}
	return out
}

// barItems converts to the menu-bar model. Icon and Checkbox are dropped; see
// the fields for why.
func (m Menu) barItems() []menubar.Item {
	out := make([]menubar.Item, 0, len(m))
	for _, it := range m {
		out = append(out, menubar.Item{
			Title:     it.Label,
			Shortcut:  it.Shortcut,
			OnClick:   it.Do,
			Selector:  it.Selector,
			Services:  it.Services,
			Submenu:   it.Submenu.barItems(),
			Separator: it.Separator,
			Disabled:  it.Disabled,
		})
	}
	return out
}
