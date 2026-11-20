package shell

import (
	"errors"
	"fmt"

	"github.com/bbmumford/mullion/icon"
	"github.com/bbmumford/mullion/internal/mainthread"
	"github.com/bbmumford/mullion/internal/tray"
)

// A Tray is a notification-area icon and its menu. Fill the fields, then call
// Show; the setters change it afterwards while it is on screen.
//
// Show attaches the icon to whichever event loop the process is running and
// does not need one of its own, which is what lets a tray sit beside windows in
// the same process. Run is for the tray-only case, where nothing else would
// drive the loop.
type Tray struct {
	// Icon is the artwork in the notification area. On macOS set
	// icon.Icon.Template for a menu-bar silhouette that follows the system
	// theme; on Windows set Dark for a variant used with a dark taskbar.
	Icon icon.Icon

	// Tooltip is the hover text.
	Tooltip string

	// Menu opens on a right-click (on a left-click too, where the platform
	// does that). Replace it later with SetMenu.
	Menu Menu

	// Name identifies the application to the Linux notification daemon, which
	// shows it as the sender. macOS and Windows take the identity from the
	// running bundle or executable and ignore this.
	Name string

	// OnClick, OnDoubleClick and OnRightClick are the icon's own handlers; they
	// run on the main thread. Setting OnRightClick replaces the default, which
	// is to open Menu, so a tray that wants both has to open the menu itself.
	OnClick       func()
	OnDoubleClick func()
	OnRightClick  func()

	impl  *tray.Tray
	shown bool
}

// ErrUnsupported reports that this platform has no backend for the surface
// asked for. It wraps errors.ErrUnsupported, so one errors.Is covers the tray,
// the menu bar and notifications together.
var ErrUnsupported = fmt.Errorf("shell: not supported on this platform: %w", errors.ErrUnsupported)

// Show creates the icon if this is the first call and makes it visible. It must
// run on the main thread -- the thread internal/mainthread pinned -- because
// every platform builds its status item there.
func (t *Tray) Show() error {
	if err := t.create(); err != nil {
		return err
	}
	if err := t.apply(); err != nil {
		return err
	}
	if err := t.impl.Show(); err != nil {
		return fmt.Errorf("shell: showing the tray: %w", err)
	}
	t.shown = true
	return nil
}

// Hide takes the icon out of the notification area and keeps it, so a later
// Show brings the same icon and menu back. Remove destroys it instead.
func (t *Tray) Hide() error {
	if t.impl == nil {
		return nil
	}
	if err := t.impl.Hide(); err != nil {
		return fmt.Errorf("shell: hiding the tray: %w", err)
	}
	t.shown = false
	return nil
}

// Run drives the platform event loop with this tray and blocks until Remove.
// It is the entry point for an application that is only a tray: with a window
// in the process, that window's surface drives the loop and the tray just needs
// Show.
//
// Only one surface may drive the loop, so Run reports who has it rather than
// starting a second one that would fight the first.
func (t *Tray) Run() error {
	if !mainthread.Claim("shell.Tray") {
		return fmt.Errorf("shell: the event loop is already driven by %s; call Show instead", mainthread.Owner())
	}
	defer mainthread.Release()

	if !t.shown {
		if err := t.Show(); err != nil {
			return err
		}
	}
	mainthread.SetWake(mainthread.Wake)
	if err := t.impl.Run(); err != nil {
		return fmt.Errorf("shell: the tray event loop: %w", err)
	}
	return nil
}

// SetIcon replaces the artwork on a tray that is already up.
func (t *Tray) SetIcon(ic icon.Icon) error {
	t.Icon = ic
	if t.impl == nil {
		return nil
	}
	return t.installIcon()
}

// SetTooltip replaces the hover text on a tray that is already up.
func (t *Tray) SetTooltip(text string) error {
	t.Tooltip = text
	if t.impl == nil {
		return nil
	}
	if err := t.impl.SetTooltip(text); err != nil {
		return fmt.Errorf("shell: setting the tooltip: %w", err)
	}
	return nil
}

// SetMenu replaces the menu. Passing an empty Menu removes it, which leaves an
// icon that only reports clicks.
func (t *Tray) SetMenu(m Menu) error {
	t.Menu = m
	if t.impl == nil {
		return nil
	}
	var converted *tray.Menu
	if len(m) > 0 {
		converted = m.trayMenu()
	}
	if err := t.impl.SetMenu(converted); err != nil {
		return fmt.Errorf("shell: setting the tray menu: %w", err)
	}
	return nil
}

// Notify shows a system notification from this tray. On Windows the balloon
// belongs to the tray icon, so a tray is the only way to send one; Notify in
// notify.go is the surface-independent spelling and comes here when a tray
// exists.
func (t *Tray) Notify(title, body string) error {
	if t.impl == nil {
		return fmt.Errorf("shell: notify before the tray exists: call Show first")
	}
	if err := t.impl.ShowNotification(title, body); err != nil {
		return fmt.Errorf("shell: notifying: %w", err)
	}
	return nil
}

// Bounds is the icon's position and size on screen, for putting a popup window
// under it, in the platform's own screen coordinates -- on macOS that means the
// origin is the bottom left of the primary screen.
//
// Ask for it once the event loop is running. A notification area places its
// icons as part of running, so a read taken between Show and the loop can come
// back partly filled; from a click handler, or any other point where the loop
// is going, it is complete. Zero width or height means the platform would not
// say, rather than an icon of no size.
func (t *Tray) Bounds() (x, y, w, h int) {
	if t.impl == nil {
		return 0, 0, 0, 0
	}
	return t.impl.Bounds()
}

// Remove destroys the icon and releases the platform resources behind it. A
// Run blocked on this tray returns.
func (t *Tray) Remove() {
	if t.impl == nil {
		return
	}
	t.impl.Remove()
	t.shown = false
}

// newPlatform builds the backend a Tray drives. It is a variable so a test can
// put a recorder in front of the notification area and still exercise the
// conversion and ordering in this file, which is where the two menu models are
// reconciled and where a mistake would be invisible on the one platform the
// author happens to use.
var newPlatform = tray.NewPlatformTray

func (t *Tray) create() error {
	if t.impl != nil {
		return nil
	}
	impl := &tray.Tray{ID: tray.NewTrayID()}
	// The platform reads the callbacks through this pointer, so later writes to
	// OnClick and its siblings reach it without re-creating the tray.
	platform := newPlatform(&impl.Callbacks)
	impl.Platform = platform
	if err := platform.Create(); err != nil {
		return fmt.Errorf("shell: creating the tray: %w", err)
	}
	t.impl = impl
	return nil
}

func (t *Tray) apply() error {
	t.impl.Callbacks.OnClick = t.OnClick
	t.impl.Callbacks.OnDoubleClick = t.OnDoubleClick
	t.impl.Callbacks.OnRightClick = t.OnRightClick
	if t.Name != "" {
		t.impl.SetAppName(t.Name)
	}
	if err := t.installIcon(); err != nil {
		return err
	}
	if t.Tooltip != "" {
		if err := t.impl.SetTooltip(t.Tooltip); err != nil {
			return fmt.Errorf("shell: setting the tooltip: %w", err)
		}
	}
	if len(t.Menu) > 0 {
		if err := t.impl.SetMenu(t.Menu.trayMenu()); err != nil {
			return fmt.Errorf("shell: setting the tray menu: %w", err)
		}
	}
	return nil
}

// installIcon spreads one icon.Icon over the three calls the tray backend
// offers. The template variant goes in first because on macOS it decides how
// the icon is drawn, and the plain bytes are what the other platforms show.
func (t *Tray) installIcon() error {
	if t.Icon.Empty() {
		return nil
	}
	if t.Icon.Template {
		t.impl.SetTemplateIcon(t.Icon.Data)
	}
	if err := t.impl.SetIcon(t.Icon.Data); err != nil {
		return fmt.Errorf("shell: setting the tray icon: %w", err)
	}
	if len(t.Icon.Dark) > 0 {
		if err := t.impl.SetDarkModeIcon(t.Icon.Dark); err != nil {
			return fmt.Errorf("shell: setting the dark-mode tray icon: %w", err)
		}
	}
	return nil
}
