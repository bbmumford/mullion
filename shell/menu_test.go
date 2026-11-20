package shell

import (
	"testing"

	"github.com/bbmumford/mullion/icon"
	"github.com/bbmumford/mullion/internal/tray"
)

// One Item has to become the right thing in two different menu models. These
// tests pin the mapping, because a field quietly landing nowhere looks like a
// platform quirk to whoever hits it.

func TestTrayConversionGivesEachItemKindItsType(t *testing.T) {
	m := Menu{
		{Label: "Plain"},
		Separator(),
		{Label: "Toggle", Checkbox: true, Checked: true},
		{Label: "With icon", Icon: icon.PNG([]byte("png"))},
		{Label: "More", Submenu: Menu{{Label: "Nested"}}},
	}
	got := m.trayMenu()
	if len(got.Items) != 5 {
		t.Fatalf("converted %d items, want 5", len(got.Items))
	}
	for i, want := range []tray.MenuItemType{
		tray.MenuItemNormal,
		tray.MenuItemSeparator,
		tray.MenuItemCheckbox,
		tray.MenuItemNormal,
		tray.MenuItemSubmenu,
	} {
		if got.Items[i].Type != want {
			t.Errorf("item %d (%q) has type %v, want %v", i, got.Items[i].Label, got.Items[i].Type, want)
		}
	}
	if string(got.Items[3].Icon) != "png" {
		t.Errorf("the item's icon bytes did not travel: %q", got.Items[3].Icon)
	}
	if !got.Items[2].Checked {
		t.Error("a checkbox that starts ticked arrived unticked")
	}
	sub := got.Items[4].Submenu
	if sub == nil || len(sub.Items) != 1 || sub.Items[0].Label != "Nested" {
		t.Errorf("the submenu did not convert: %+v", sub)
	}
}

func TestTrayConversionCarriesDisabledOnEveryKind(t *testing.T) {
	m := Menu{
		{Label: "Plain", Disabled: true},
		{Label: "Toggle", Checkbox: true, Disabled: true},
		{Label: "With icon", Icon: icon.PNG([]byte("png")), Disabled: true},
		{Label: "More", Submenu: Menu{{Label: "Nested"}}, Disabled: true},
	}
	for i, item := range m.trayMenu().Items {
		if !item.IsDisabled() {
			t.Errorf("item %d (%q) came through enabled", i, item.Label)
		}
	}
}

func TestTrayConversionKeepsHandlers(t *testing.T) {
	var ran string
	m := Menu{
		{Label: "Open", Do: func() { ran = "open" }},
		{Label: "Toggle", Checkbox: true, Do: func() { ran = "toggle" }},
	}
	items := m.trayMenu().Items
	items[0].OnClick()
	if ran != "open" {
		t.Errorf("a plain item's handler did not travel (ran=%q)", ran)
	}
	items[1].OnClick()
	if ran != "toggle" {
		t.Errorf("a checkbox's handler did not travel (ran=%q)", ran)
	}
}

func TestMenuBarConversionCarriesTheMenuBarOnlyFields(t *testing.T) {
	m := Menu{{
		Label: "App",
		Submenu: Menu{
			{Label: "About", Do: func() {}},
			Separator(),
			{Label: "Services", Services: true},
			{Label: "Copy", Selector: "copy:", Shortcut: "cmd+c"},
			{Label: "Quit", Shortcut: "cmd+q", Disabled: true},
		},
	}}
	bar := m.barItems()
	if len(bar) != 1 || bar[0].Title != "App" {
		t.Fatalf("top level = %+v", bar)
	}
	sub := bar[0].Submenu
	if len(sub) != 5 {
		t.Fatalf("submenu has %d items, want 5", len(sub))
	}
	if !sub[1].Separator {
		t.Error("the separator did not convert")
	}
	if !sub[2].Services {
		t.Error("Services did not reach the menu bar")
	}
	if sub[3].Selector != "copy:" || sub[3].Shortcut != "cmd+c" {
		t.Errorf("Selector/Shortcut did not reach the menu bar: %+v", sub[3])
	}
	if sub[4].Shortcut != "cmd+q" || !sub[4].Disabled {
		t.Errorf("Shortcut/Disabled did not reach the menu bar: %+v", sub[4])
	}
}

// Shortcut, Selector and Services have no place in a notification-area menu and
// Icon has none in a menu bar. The conversions must drop them rather than
// invent a place for them, and must not drop the item along with them.
func TestEachConversionDropsOnlyWhatItCannotShow(t *testing.T) {
	m := Menu{{
		Label:    "Copy",
		Icon:     icon.PNG([]byte("png")),
		Shortcut: "cmd+c",
		Selector: "copy:",
		Do:       func() {},
	}}

	trayItems := m.trayMenu().Items
	if len(trayItems) != 1 {
		t.Fatalf("the tray conversion dropped the item: %+v", trayItems)
	}
	if string(trayItems[0].Icon) != "png" {
		t.Errorf("the tray lost the icon it can show: %q", trayItems[0].Icon)
	}

	barItems := m.barItems()
	if len(barItems) != 1 {
		t.Fatalf("the menu-bar conversion dropped the item: %+v", barItems)
	}
	if barItems[0].Shortcut != "cmd+c" || barItems[0].Selector != "copy:" {
		t.Errorf("the menu bar lost what only it can show: %+v", barItems[0])
	}
}

func TestEmptyMenuConvertsToEmptyNotNil(t *testing.T) {
	if got := (Menu{}).trayMenu(); got == nil {
		t.Error("an empty Menu produced a nil tray menu, which reads as no menu at all")
	} else if len(got.Items) != 0 {
		t.Errorf("an empty Menu produced %d items", len(got.Items))
	}
	if got := (Menu{}).barItems(); len(got) != 0 {
		t.Errorf("an empty Menu produced %d menu-bar items", len(got))
	}
}
