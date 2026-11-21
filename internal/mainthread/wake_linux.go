//go:build linux && !android

package mainthread

import (
	"sync"

	"github.com/ebitengine/purego"
)

// G_PRIORITY_HIGH_IDLE. Higher than g_idle_add's own default, and chosen to
// match what the WebView surface's dispatcher used before it was routed
// through here: lowering it would have changed when that work runs relative to
// other idle sources, on a platform neither of us can test. It is still an
// idle source, so it never runs ahead of GTK's own event handling.
const gPriorityHighIdle = 100

// Wake asks the main thread to call Drain, through GLib's idle source.
//
// GLib is the right channel when the loop belongs to a webview window or to the
// StatusNotifierItem tray, because both run a GLib main loop. The Go-drawn
// surface runs a Wayland or X11 loop with no GLib in it, and there the source
// never fires: that surface drains from its own frame hook instead, so a wake
// that goes nowhere costs nothing. This is also why a missing libglib is not an
// error here.
//
// g_idle_add_full, and deliberately not g_main_context_invoke_full: that one
// runs the callback immediately when the caller already owns the context, which
// is the main thread in every case that matters. A wake is supposed to mean
// "drain at the loop's next turn", and running inline instead would drain
// before the loop had started -- bringing the tray and the menu bar up before
// there was a loop to attach them to.
func Wake() {
	if g := linuxIdle(); g.ok {
		g.idleAdd(gPriorityHighIdle, g.drainWork, 0, 0)
	}
}

// postToPlatform hands a posted call to GLib, and reports whether it could.
func postToPlatform(id uintptr) bool {
	g := linuxIdle()
	if !g.ok {
		return false
	}
	g.idleAdd(gPriorityHighIdle, g.postWork, id, 0)
	return true
}

type linuxQueue struct {
	ok        bool
	idleAdd   func(priority int32, fn uintptr, data uintptr, notify uintptr) uint32
	drainWork uintptr
	postWork  uintptr
}

var linuxIdle = sync.OnceValue(func() linuxQueue {
	lib, err := purego.Dlopen("libglib-2.0.so.0", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return linuxQueue{}
	}
	var idleAdd func(priority int32, fn uintptr, data uintptr, notify uintptr) uint32
	purego.RegisterLibFunc(&idleAdd, lib, "g_idle_add_full")

	// Returning G_SOURCE_REMOVE (0) makes GLib drop the source after it runs,
	// so each wake is one Drain and each post is one call.
	return linuxQueue{
		ok:      true,
		idleAdd: idleAdd,
		drainWork: purego.NewCallback(func(uintptr) uintptr {
			Drain()
			return 0
		}),
		postWork: purego.NewCallback(func(id uintptr) uintptr {
			runPosted(id)
			return 0
		}),
	}
})
