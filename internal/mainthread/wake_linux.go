//go:build linux && !android

package mainthread

import (
	"sync"

	"github.com/ebitengine/purego"
)

// Wake asks the main thread to call Drain, through GLib's default main context.
//
// GLib is the right channel when the loop belongs to a webview window or to the
// StatusNotifierItem tray, because both run a GLib main loop. The Go-drawn
// surface runs a Wayland or X11 loop with no GLib in it, and there the invoke
// never fires: that surface drains from its own frame hook instead, so a wake
// that goes nowhere costs nothing. This is also why a missing libglib is not an
// error here.
func Wake() { linuxWake()() }

var linuxWake = sync.OnceValue(func() func() {
	lib, err := purego.Dlopen("libglib-2.0.so.0", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return func() {}
	}
	// g_main_context_invoke_full(NULL, ...) runs the callback on the thread that
	// owns the default context, which under GTK is the thread that called
	// gtk_init -- the one pinned here.
	var invokeFull func(context uintptr, priority int32, fn uintptr, data uintptr, notify uintptr)
	purego.RegisterLibFunc(&invokeFull, lib, "g_main_context_invoke_full")

	// Returning G_SOURCE_REMOVE (0) makes GLib drop the callback after it runs,
	// so each wake is one Drain.
	work := purego.NewCallback(func(uintptr) uintptr {
		Drain()
		return 0
	})
	const gPriorityDefault = 0
	return func() { invokeFull(0, gPriorityDefault, work, 0, 0) }
})
