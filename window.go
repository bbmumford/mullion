package mullion

import (
	"errors"
	"fmt"

	"github.com/bbmumford/mullion/internal/mainthread"
	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/app"
	extwindow "github.com/bbmumford/mullion/native/ext/window"
)

// The calls below move and measure the Go-drawn window while the application
// runs, where the fields on Window describe it before it opens.
//
// Each is best-effort by nature: a tiling compositor decides where windows go
// and a maximised window will not be resized, so a platform that declines is
// behaving correctly rather than failing. None of them reports what the
// platform did, because the platforms do not say.
//
// They act on the Go-drawn window and need it to be the surface driving the
// loop. Asked of a WebView or tray application they report an error wrapping
// errors.ErrUnsupported rather than doing nothing quietly, so the caller finds
// out at the call instead of wondering why the window did not move.
//
// Any of them may be called from a goroutine: the work is put on the main
// thread, since these are UI calls and the platforms accept them nowhere else.

// PopupOptions is how a popup window is described, re-exported so a caller
// configuring one does not have to import the native package to name the type.
type PopupOptions = native.PopupWindowOptions

// Resize asks for the window's content area to become this size, in logical
// points.
func (s *Session) Resize(width, height int) error {
	return s.onWindow("Resize", func() {
		extwindow.SetSize(float32(width), float32(height))
	})
}

// SetMinSize stops the person shrinking the window below this size. Zero for
// either dimension removes the minimum on that axis.
func (s *Session) SetMinSize(width, height int) error {
	return s.onWindow("SetMinSize", func() {
		extwindow.SetMinSize(float32(width), float32(height))
	})
}

// Center puts the window in the middle of the primary display.
func (s *Session) Center() error {
	return s.onWindow("Center", extwindow.Center)
}

// MoveTo puts the window's top-left corner at this screen position, in the
// coordinates Position reports.
func (s *Session) MoveTo(x, y int) error {
	return s.onWindow("MoveTo", func() { extwindow.Position(x, y) })
}

// Position reads the window's top-left corner on screen. The second result is
// false where the platform will not say, which is every mobile backend and
// Wayland for an ordinary toplevel.
func (s *Session) Position() (x, y int, ok bool) {
	if s.owner != surfaceNative {
		return 0, 0, false
	}
	var pos native.Vec2
	var got bool
	if err := s.onWindow("Position", func() { pos, got = extwindow.CurrentPosition() }); err != nil {
		return 0, 0, false
	}
	return int(pos[0]), int(pos[1]), got
}

// PointerInside reports whether the pointer is over the window's content. It
// is a geometric check on X11, macOS and Windows; on Wayland it follows pointer
// focus, so a window that never takes keyboard focus still reports true.
func (s *Session) PointerInside() bool {
	if s.owner != surfaceNative {
		return false
	}
	var inside bool
	if err := s.onWindow("PointerInside", func() { inside = extwindow.Hovered() }); err != nil {
		return false
	}
	return inside
}

// Activate brings the window forward and gives it keyboard focus.
//
// Desktops increasingly refuse this from an application that the person is not
// already interacting with, which is deliberate on their part: it is how a
// window is stopped from stealing what someone is typing. Expect it to be
// ignored unless the request follows something they did.
func (s *Session) Activate() error {
	return s.onWindow("Activate", extwindow.Activate)
}

// StartAudio opens an audio output stream and calls fill whenever the device
// wants more samples.
//
// fill runs on an audio thread, not the main one, and must return before the
// buffer is due: no allocation, no locks held by the UI, no blocking. Writing
// nothing leaves silence rather than a glitch.
func (s *Session) StartAudio(sampleRate int, fill func(out []float32)) error {
	if fill == nil {
		return errors.New("mullion: StartAudio needs a fill function")
	}
	return app.StartAudio(sampleRate, fill)
}

// OpenPopup opens a small borderless window drawn by root, for a completion
// list, a tooltip or a colour picker -- the things that have to escape the
// window's bounds.
//
// It belongs to the Go-drawn surface, and the returned handle is the only way
// to close it again.
func (s *Session) OpenPopup(opts PopupOptions, root native.FrameFn) (app.PopupWindow, error) {
	if s.owner != surfaceNative {
		return 0, fmt.Errorf("mullion: popups need the Go-drawn surface, and %s is driving: %w",
			s.owner, errors.ErrUnsupported)
	}
	if root == nil {
		return 0, errors.New("mullion: OpenPopup needs a draw function")
	}
	return app.OpenPopupWindow(opts, root)
}

// onWindow runs fn on the main thread, once it is established that there is a
// Go-drawn window for it to act on.
func (s *Session) onWindow(what string, fn func()) error {
	if s.owner != surfaceNative {
		return fmt.Errorf("mullion: %s needs the Go-drawn window, and %s is driving the loop: %w",
			what, s.owner, errors.ErrUnsupported)
	}
	if mainthread.OnMainThread() {
		fn()
		native.RequestNextFrame()
		return nil
	}
	done := make(chan struct{})
	mainthread.Do(func() {
		fn()
		native.RequestNextFrame()
		close(done)
	})
	<-done
	return nil
}
