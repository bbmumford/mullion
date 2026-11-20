// Package mullion runs a desktop application whose user interface may be drawn
// in Go, rendered by the system WebView, or be nothing but a tray icon -- with
// one event loop serving all of them.
//
// The three libraries this module is built from each owned the process: each
// pinned the thread the process started on and each called the platform's run
// function from its own Run. A program cannot do that three times, so a
// mullion application names its surfaces in an App and Run decides which one
// drives the loop and attaches the rest to it.
//
//	func main() {
//		err := mullion.Run(mullion.App{
//			Name:   "Notes",
//			Icon:   icon.PNG(iconPNG),
//			Window: &mullion.Window{Title: "Notes", Width: 900, Height: 600, Draw: ui},
//			Tray:   &shell.Tray{Tooltip: "Notes"},
//		})
//		if err != nil {
//			log.Fatal(err)
//		}
//	}
//
// Which surface drives the loop is not a setting: a Go-drawn window takes it,
// otherwise a WebView window, otherwise the tray. The reason is that the
// Go-drawn backend's loop is the one that cannot be entered a second time or
// attached to an existing one, while a WebView window and a tray icon can both
// be built on a loop that is already running. An application that wants a
// Go-drawn window therefore has to say so in App.Window rather than open one
// later, and Session.OpenWindow says so plainly if asked.
package mullion

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"unsafe"

	"github.com/bbmumford/mullion/icon"
	"github.com/bbmumford/mullion/internal/mainthread"
	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/app"
	"github.com/bbmumford/mullion/shell"
	"github.com/bbmumford/mullion/web"
	g "go.hasen.dev/generic"
)

// An App is a whole desktop application. Every field is optional on its own,
// but an App with no window and no tray has no user interface and Run says so.
type App struct {
	// Name identifies the application: the menu-bar title on macOS and the
	// sender a Linux notification daemon shows.
	Name string

	// Icon is the dock, taskbar and window icon, and the tray's icon when the
	// Tray does not carry one of its own.
	Icon icon.Icon

	// Window is the window opened at startup. Leave it nil for an application
	// that lives in the notification area, and open a WebView window later
	// with Session.OpenWindow.
	Window *Window

	// Tray is the notification-area icon. It is brought up once the loop is
	// running, which is the only time the platform will accept a status item.
	Tray *shell.Tray

	// MenuBar is the application menu. It is installed on macOS and Windows;
	// on Linux the shortcuts are installed without a bar being drawn, and
	// elsewhere it is skipped. A failure to install it is reported through
	// OnReady's error rather than silently.
	MenuBar shell.Menu

	// OnReady runs once, on the main thread, after the loop is up and the
	// surfaces are attached. It is where to open further windows, start
	// background work, or read what the shell actually managed to install. An
	// error from it stops the application.
	OnReady func(*Session) error

	// OnQuit runs as the process exits, whichever way it was asked to. It runs
	// on the goroutine that is tearing down, so it should be short: flush
	// state, do not start anything.
	OnQuit func()
}

// A Window is the application's window. Exactly one of Draw and Page says what
// is in it.
type Window struct {
	// Title is the window title.
	Title string

	// Width and Height are the initial content size in points. Zero means
	// 800x600.
	Width  int
	Height int

	// Draw is a Go-drawn user interface: a function called once per frame that
	// describes the window's contents, in the style of the native package.
	// A window with a Draw drives the application's event loop.
	Draw native.FrameFn

	// Page is a user interface rendered by the system WebView.
	Page *Page
}

// A Page is what a WebView window shows.
type Page struct {
	// HTML is the document to show. It is used when URL is empty.
	HTML string

	// URL is navigated to instead of HTML, and may use a scheme registered for
	// embedded assets.
	URL string

	// Debug opens the WebView with its inspector available.
	Debug bool

	// Bind exposes Go functions to the page's JavaScript under these names.
	// The signatures are the ones the web package accepts: no return, a value,
	// an error, or a value and an error.
	Bind map[string]any
}

// A Session is the running application, handed to OnReady and to the handlers
// an application installs. Its methods are safe to call from any goroutine.
type Session struct {
	app   App
	owner surface
	view  web.WebView
	bar   *shell.MenuBar

	// attachedViews counts the WebView windows living under another surface's
	// loop, which is what decides whether a frame has to service GTK. It is
	// read every frame on the main thread and written by OpenWindow, which may
	// be called from a goroutine.
	attachedViews atomic.Int32

	// shortcuts are the MenuBar accelerators this process has to match itself,
	// because no platform menu bar took them. Built once while attaching and
	// read every frame, both on the main thread.
	shortcuts []shell.Shortcut
}

// surface names which kind of thing is driving the event loop.
type surface string

const (
	surfaceNative surface = "the Go-drawn window"
	surfaceWeb    surface = "the WebView window"
	surfaceTray   surface = "the tray"
)

// Run starts the application and blocks until it quits.
//
// With a Go-drawn window it does not return: that backend ends the process
// through its own exit path so cleanup handlers run, and App.OnQuit is one of
// them. A failure while attaching the shell or in OnReady is written to stderr
// there, because there is no caller left to hand it to. With a WebView window
// or a tray, Run returns when that surface's loop ends, and returns such a
// failure instead of printing it.
//
// It must be called from the goroutine that ran main, which is the one pinned
// to the thread every platform here requires for windows.
func Run(a App) error {
	if err := a.validate(); err != nil {
		return err
	}
	if !mainthread.OnMainThread() {
		return errors.New("mullion: Run must be called from the goroutine that ran main; the window APIs on every platform here accept calls only on that thread")
	}

	s := &Session{app: a, owner: a.loopOwner()}
	if a.OnQuit != nil {
		g.AddExitCleanup(a.OnQuit)
	}

	// The surfaces that attach cannot be built yet: a status item made before
	// the application has finished launching crashes on macOS, and a WebView
	// has to see the loop running to attach to it instead of starting one.
	// Queueing the work here means it runs on the main thread at the loop's
	// first turn, whichever surface is driving.
	var attachErr error
	mainthread.Do(func() {
		attachErr = s.attach()
		if attachErr == nil {
			return
		}
		// The Go-drawn backend ends the process from inside its own loop, so
		// Run never returns there and returning this error would reach nobody.
		// Printing it is the only way it is seen; the other owners' loops do
		// return and Run hands it back as well.
		if s.owner == surfaceNative {
			fmt.Fprintf(os.Stderr, "mullion: %v\n", attachErr)
		}
		s.Quit()
	})

	switch s.owner {
	case surfaceNative:
		return s.runNative()
	case surfaceWeb:
		if err := s.runWeb(); err != nil {
			return err
		}
	case surfaceTray:
		if err := s.app.Tray.Run(); err != nil {
			return err
		}
	}
	return attachErr
}

// attach brings up everything that is not driving the loop. It runs on the main
// thread with the loop already going.
func (s *Session) attach() error {
	if s.app.Tray != nil {
		if s.app.Tray.Icon.Empty() {
			s.app.Tray.Icon = s.app.Icon
		}
		if s.app.Tray.Name == "" {
			s.app.Tray.Name = s.app.Name
		}
		if s.owner != surfaceTray {
			if err := s.app.Tray.Show(); err != nil {
				return err
			}
		}
	}
	if len(s.app.MenuBar) > 0 {
		bar, err := shell.SetMenuBar(s.app.MenuBar, s.NativeWindow())
		// No menu bar on this platform is a fact about the platform, not a
		// failure of the application: the rest of it still runs.
		if err != nil && !errors.Is(err, errors.ErrUnsupported) {
			return err
		}
		s.bar = bar
		// With no bar installed, nothing is matching the accelerators, so the
		// Go-drawn surface does it from its own input. Where a bar went up it
		// already answers them, and matching here as well would run each
		// handler twice.
		if bar == nil && s.owner == surfaceNative {
			s.shortcuts = s.app.MenuBar.Shortcuts()
		}
	}
	if s.app.OnReady != nil {
		return s.app.OnReady(s)
	}
	return nil
}

// runNative drives the loop with the Go-drawn backend.
func (s *Session) runNative() error {
	if !mainthread.Claim(string(surfaceNative)) {
		return fmt.Errorf("mullion: %s already drives the event loop", mainthread.Owner())
	}
	w := s.app.Window

	// The Go-drawn loop has no idle callback, but it does call frame-start
	// hooks every frame, and asking for a frame is how its backends are woken
	// from a goroutine already. Together those make the main-thread queue
	// arrive here the same way it does under a WebView or tray loop.
	native.RegisterFrameStart(mainthread.Drain)
	native.RegisterFrameStart(s.pumpAttachedWindows)
	native.RegisterFrameStart(s.matchMenuShortcuts)
	mainthread.SetWake(native.RequestNextFrame)

	app.SetupWindow(w.Title, w.width(), w.height())
	if !s.app.Icon.Empty() {
		app.SetupIconBytes(s.app.Icon.Data)
	}
	app.Run(w.Draw) // does not return
	return nil
}

// matchMenuShortcuts answers the MenuBar's accelerators from the Go-drawn
// surface's own input.
//
// A menu bar is not available everywhere the menu is: macOS installs a real one
// and matches the keys itself, Windows attaches one to the window, and Linux
// draws none at all -- and the bar that would have carried the shortcuts there
// wants a GtkWindow, which a Go-drawn window is not. Matching them here makes
// one MenuBar behave the same on all three.
//
// On macOS the bar has already answered the key, so this would run the handler
// a second time; the shortcuts are only collected where no bar took them.
func (s *Session) matchMenuShortcuts() {
	if len(s.shortcuts) == 0 {
		return
	}
	shell.Match(s.shortcuts)
}

// pumpAttachedWindows gives a WebView window opened under this loop its turn.
//
// A frame is the only recurring moment the Go-drawn backends offer, and on
// Linux it is the only thing that will service a GTK window: that loop is
// Wayland or X11 and never iterates GTK's main context, so a WebView window
// attached to it would not redraw or answer a click. The call costs nothing
// where the platform already services every window, and nothing at all until a
// WebView has been opened.
func (s *Session) pumpAttachedWindows() {
	if s.attachedViews.Load() == 0 {
		return
	}
	web.Pump()
}

// runWeb drives the loop with a WebView window.
func (s *Session) runWeb() error {
	if !mainthread.Claim(string(surfaceWeb)) {
		return fmt.Errorf("mullion: %s already drives the event loop", mainthread.Owner())
	}
	defer mainthread.Release()

	w := s.app.Window
	view, err := s.newView(*w)
	if err != nil {
		return err
	}
	s.view = view
	// The WebView's own dispatcher reaches the main thread without going
	// through libdispatch or GLib a second time, so prefer it while this
	// window is the one driving.
	mainthread.SetWake(func() { view.Dispatch(mainthread.Drain) })
	defer mainthread.SetWake(nil)

	view.Run()
	return nil
}

// newView builds a WebView window from a Window description.
func (s *Session) newView(w Window) (web.WebView, error) {
	view, err := web.New(w.Page.Debug)
	if err != nil {
		return nil, fmt.Errorf("mullion: opening a WebView window: %w", err)
	}
	title := w.Title
	if title == "" {
		title = s.app.Name
	}
	view.SetTitle(title)
	view.SetSize(w.width(), w.height(), web.HintNone)
	if !s.app.Icon.Empty() {
		// No application icon on Windows or Linux is normal and not worth
		// failing an otherwise working window for.
		_ = web.SetAppIcon(s.app.Icon.Data)
	}
	for name, fn := range w.Page.Bind {
		if err := view.Bind(name, fn); err != nil {
			view.Destroy()
			return nil, fmt.Errorf("mullion: binding %q to the page: %w", name, err)
		}
	}
	switch {
	case w.Page.URL != "":
		view.Navigate(w.Page.URL)
	case w.Page.HTML != "":
		view.SetHtml(w.Page.HTML)
	}
	if s.owner != surfaceWeb {
		s.attachedViews.Add(1)
	}
	return view, nil
}

// OpenWindow opens another WebView window and returns it. It may be called from
// any goroutine and from a handler on the main thread.
//
// It opens a WebView window and not a Go-drawn one whatever the Window says,
// and reports that rather than quietly ignoring Draw: the Go-drawn backends
// enter the platform's loop to show a window, which cannot be done a second
// time or on top of a loop that is already running. A Go-drawn window has to be
// the one in App.Window.
func (s *Session) OpenWindow(w Window) (web.WebView, error) {
	if w.Draw != nil {
		return nil, errors.New("mullion: a Go-drawn window can only be opened as App.Window, because its backend shows a window by entering the platform's event loop; use a Page here, or make the Go-drawn window the one App.Window names")
	}
	if w.Page == nil {
		return nil, errors.New("mullion: OpenWindow needs a Page to show")
	}
	type result struct {
		view web.WebView
		err  error
	}
	if mainthread.OnMainThread() {
		view, err := s.newView(w)
		return view, err
	}
	ch := make(chan result, 1)
	mainthread.Do(func() {
		view, err := s.newView(w)
		ch <- result{view, err}
	})
	r := <-ch
	return r.view, r.err
}

// Do runs fn on the main thread and returns before it has run. It is how a
// goroutine touches a window, a menu or the tray, all of which accept calls
// only there.
func (s *Session) Do(fn func()) { mainthread.Do(fn) }

// Tray is the application's tray, or nil if it has none.
func (s *Session) Tray() *shell.Tray { return s.app.Tray }

// View is the WebView window Run opened, or nil when the application's window
// is Go-drawn or it has none.
func (s *Session) View() web.WebView { return s.view }

// MenuBar is the installed menu bar, or nil where none was installed -- either
// because the App named none or because the platform has no bar.
func (s *Session) MenuBar() *shell.MenuBar { return s.bar }

// Notify shows a system notification. It needs a Tray, because on Windows a
// notification is a balloon belonging to a notification-area icon.
func (s *Session) Notify(title, body string) error {
	return shell.Notify(s.app.Tray, shell.Notification{Title: title, Body: body})
}

// NativeWindow is the platform handle of the window driving the application --
// an NSWindow on macOS, an HWND on Windows -- for the calls that need one, such
// as a menu bar on Windows or a file dialog's parent. It reads nil where the
// surface has no such handle to give: a Go-drawn window on Wayland or X11 is
// not a GtkWindow, and the dialogs and menus take nil to mean no parent.
func (s *Session) NativeWindow() unsafe.Pointer {
	if s.view != nil {
		return s.view.Window()
	}
	return nativeWindowHandle()
}

// Quit ends the application through the path its loop owner understands, so
// that OnQuit and the cleanup handlers run. Safe to call from any goroutine.
func (s *Session) Quit() {
	switch s.owner {
	case surfaceNative:
		app.Quit()
	case surfaceWeb:
		if s.view != nil {
			s.view.Terminate()
		}
	case surfaceTray:
		if s.app.Tray != nil {
			s.app.Tray.Remove()
		}
	}
}

// loopOwner picks the surface that drives the loop. The order is forced by what
// each backend can do, not by preference: see the package comment.
func (a App) loopOwner() surface {
	if a.Window != nil && a.Window.Draw != nil {
		return surfaceNative
	}
	if a.Window != nil && a.Window.Page != nil {
		return surfaceWeb
	}
	return surfaceTray
}

func (a App) validate() error {
	if a.Window == nil && a.Tray == nil {
		return errors.New("mullion: the App has no Window and no Tray, so it has no user interface")
	}
	if a.Window != nil {
		switch {
		case a.Window.Draw == nil && a.Window.Page == nil:
			return errors.New("mullion: the Window has neither a Draw nor a Page, so there is nothing to show in it")
		case a.Window.Draw != nil && a.Window.Page != nil:
			return errors.New("mullion: the Window has both a Draw and a Page; a window is drawn either in Go or by the WebView, so set one of them")
		}
	}
	return nil
}

func (w Window) width() int {
	if w.Width <= 0 {
		return 800
	}
	return w.Width
}

func (w Window) height() int {
	if w.Height <= 0 {
		return 600
	}
	return w.Height
}
