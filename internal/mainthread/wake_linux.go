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
	// g_idle_add, and deliberately not g_main_context_invoke_full: that one
	// runs the callback immediately when the caller already owns the context,
	// which is the main thread in every case that matters here. A wake is
	// supposed to mean "drain at the loop's next turn", and running inline
	// instead would drain before the loop had started -- bringing the tray and
	// the menu bar up before there was a loop to attach them to, and running
	// work queued from the main thread on the spot rather than in order.
	// g_idle_add always queues.
	var idleAdd func(fn uintptr, data uintptr) uint32
	purego.RegisterLibFunc(&idleAdd, lib, "g_idle_add")

	// Returning G_SOURCE_REMOVE (0) makes GLib drop the source after it runs,
	// so each wake is one Drain.
	work := purego.NewCallback(func(uintptr) uintptr {
		Drain()
		return 0
	})
	return func() { idleAdd(work, 0) }
})
