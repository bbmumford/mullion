//go:build darwin

package mainthread

import (
	"sync"

	"github.com/ebitengine/purego"
)

// Wake asks the main thread to call Drain, by posting a block to libdispatch's
// main queue.
//
// The main queue is the one primitive every loop in this module services: AppKit
// drains it from [NSApp run], from the nextEventMatchingMask pump a webview uses
// when it is not the loop owner, and from CFRunLoopRun. Binding it here rather
// than reusing one of the surfaces' own dispatchers is what lets queued work
// arrive whichever surface is driving.
func Wake() { darwinWake()() }

// darwinWake resolves libdispatch once. A resolution failure leaves a wake that
// does nothing, which costs latency and not correctness: queued work still runs
// at the loop owner's next Drain.
var darwinWake = sync.OnceValue(func() func() {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return func() {}
	}
	// dispatch_get_main_queue() is &_dispatch_main_q, so the symbol's address is
	// the queue; there is no function to call for it.
	queue, err := purego.Dlsym(lib, "_dispatch_main_q")
	if err != nil || queue == 0 {
		return func() {}
	}
	var asyncF func(queue, context, work uintptr)
	purego.RegisterLibFunc(&asyncF, lib, "dispatch_async_f")

	// One callback for the life of the process: purego's callback table is a
	// fixed size, and a new one per wake would exhaust it.
	work := purego.NewCallback(func(uintptr) uintptr {
		Drain()
		return 0
	})
	return func() { asyncF(queue, 0, work) }
})
