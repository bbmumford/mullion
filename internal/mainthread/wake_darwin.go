//go:build darwin

package mainthread

import (
	"sync"

	"github.com/ebitengine/purego"
)

// Wake asks the main thread to call Drain, by posting to libdispatch's main
// queue.
//
// The main queue is the one primitive every loop in this module services:
// AppKit drains it from [NSApp run], from the nextEventMatchingMask pump a
// webview uses when it is not the loop owner, and from CFRunLoopRun. That is
// why the binding lives here rather than in one of the surfaces -- queued work
// has to arrive whichever surface is driving, and before this there were two
// bindings of the same two symbols with a callback each.
func Wake() {
	if d := darwinDispatch(); d.ok {
		d.async(d.queue, 0, d.drainWork)
	}
}

// postToPlatform hands a posted call to the main queue, and reports whether it
// could.
func postToPlatform(id uintptr) bool {
	d := darwinDispatch()
	if !d.ok {
		return false
	}
	d.async(d.queue, id, d.postWork)
	return true
}

type darwinQueue struct {
	ok        bool
	queue     uintptr
	async     func(queue, context, work uintptr)
	drainWork uintptr
	postWork  uintptr
}

// darwinDispatch resolves libdispatch once. A failure leaves Wake doing
// nothing, which costs latency and not correctness -- queued work still runs at
// the loop owner's next Drain -- and makes Post fall back to Do.
var darwinDispatch = sync.OnceValue(func() darwinQueue {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return darwinQueue{}
	}
	// dispatch_get_main_queue() is &_dispatch_main_q, so the symbol's address is
	// the queue; there is no function to call for it.
	queue, err := purego.Dlsym(lib, "_dispatch_main_q")
	if err != nil || queue == 0 {
		return darwinQueue{}
	}
	var async func(queue, context, work uintptr)
	purego.RegisterLibFunc(&async, lib, "dispatch_async_f")

	// Two callbacks for the life of the process, not one per call: purego's
	// callback table is a fixed size. The context argument carries a drain
	// signal in one and a posted call's id in the other.
	return darwinQueue{
		ok:    true,
		queue: queue,
		async: async,
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
