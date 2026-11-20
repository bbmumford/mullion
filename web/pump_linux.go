//go:build linux

package web

// Pump services the GTK main context without blocking, for a process whose
// event loop belongs to another surface.
//
// On macOS and Windows a WebView window needs nothing of the sort: NSApp
// services every window in the application, and a Win32 message pump retrieves
// the messages of every window on its thread, so a WebView window attached to
// the Go-drawn surface's loop is serviced by it. GTK is the exception -- it has
// a main context of its own that a Wayland or X11 loop never iterates, and a
// WebView window under that loop would never redraw or answer a click.
//
// The runtime calls this once per frame while a WebView window is attached and
// the Go-drawn surface is driving. It returns at once when GTK has nothing
// waiting, and does nothing before a WebView has loaded GTK at all.
func Pump() {
	if gMainContextIteration == nil {
		return
	}
	// g_main_context_iteration with may_block FALSE dispatches at most one
	// pending event and says whether it found one, so looping on that drains
	// what is waiting without ever blocking the frame.
	//
	// The cap is there because a page that keeps the context busy -- an
	// animation, a running fetch -- would otherwise hold this frame open for
	// as long as it has work. Whatever is left is dispatched next frame.
	for range 64 {
		if !gMainContextIteration(0, false) {
			return
		}
	}
}
