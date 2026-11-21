package mainthread

import "sync"

// Post runs fn on the main thread, through the platform's own queue, and
// returns before it has run.
//
// It differs from Do in where the work waits. Do holds it in this package's
// queue until the loop owner drains it, which keeps it ordered with everything
// else queued here. Post hands it to the platform -- libdispatch's main queue,
// GLib's idle source -- so it arrives when the platform gets to it, ahead of a
// drain that may be a frame away.
//
// Both reach the same thread, and a caller wanting one main-thread queue should
// use Do. Post exists because the WebView surface already worked this way: its
// own dispatcher posted straight to the platform, so routing it through Do would
// have changed when its work ran relative to the platform's own callbacks.
// Sharing the binding rather than the semantics was the safe half of that
// collapse.
//
// Where a platform has no such queue, Post falls back to Do.
func Post(fn func()) {
	if fn == nil {
		return
	}
	postMu.Lock()
	postSeq++
	id := postSeq
	postFns[id] = fn
	postMu.Unlock()

	if postToPlatform(id) {
		return
	}
	// No platform queue here, so the work goes where everything else does.
	postMu.Lock()
	delete(postFns, id)
	postMu.Unlock()
	Do(fn)
}

// runPosted is what each platform's callback calls, with the id it was given.
// It is exported to the package only; the platform files are its only callers.
func runPosted(id uintptr) {
	postMu.Lock()
	fn := postFns[id]
	delete(postFns, id)
	postMu.Unlock()
	if fn != nil {
		fn()
	}
}

// The platform callback is one C function pointer for every posted call, so the
// closure cannot travel with it: an id does, and this is where the id is looked
// back up.
var (
	postMu  sync.Mutex
	postSeq uintptr
	postFns = map[uintptr]func(){}
)
