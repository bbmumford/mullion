package shell

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/bbmumford/mullion/internal/menubar"
)

// A MenuBar is an installed application menu. Release takes it down and
// restores what was there before; it is optional and may be called once.
type MenuBar struct {
	impl *menubar.Menu
}

// SetMenuBar installs m as the application's menu bar, replacing any previous
// one. The window handle is what the bar attaches to on Windows, where a menu
// bar belongs to a window; macOS has one application-wide bar and ignores it.
// Pass what the surface hands you -- a webview window's Window(), or the Go-drawn
// surface's native handle -- or nil on macOS.
//
// Call it on the main thread. Doing it from a goroutine while the loop runs
// needs the work marshalled, which mainthread.Do is for:
//
//	mainthread.Do(func() { shell.SetMenuBar(m, w.Window()) })
//
// On Linux no bar is drawn -- the GTK3-versus-GTK4 menu-bar story is too
// fragmented to do cheaply -- but the items' shortcuts are installed on the
// window, with cmd read as ctrl. That is a partial success and reported as one.
func SetMenuBar(m Menu, window unsafe.Pointer) (*MenuBar, error) {
	installed, err := menubar.Set(m.barItems(), menubar.Options{Window: window})
	if err != nil {
		if errors.Is(err, errors.ErrUnsupported) {
			return nil, fmt.Errorf("shell: no menu bar on this platform: %w", err)
		}
		return nil, fmt.Errorf("shell: installing the menu bar: %w", err)
	}
	return &MenuBar{impl: installed}, nil
}

// Release removes the bar and frees what installing it retained.
func (b *MenuBar) Release() {
	if b == nil {
		return
	}
	b.impl.Release()
}
