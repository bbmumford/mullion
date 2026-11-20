// Package mainthread owns the thread the process started on and decides which
// surface drives the platform event loop.
//
// Every windowing API in this module is bound to that one thread: AppKit
// instantiates NSWindow and NSStatusItem only there, Win32 delivers a window's
// messages only to the thread that created the window, and GTK is initialised
// on it. Each of the three trees mullion carries used to pin that thread in its
// own init and then call the platform's run function from its own Run. A
// process has one such thread and one such loop, so a build that contains all
// three has to put both in one place: pinning happens in this package's init,
// and a surface that wants to drive the loop asks Claim first.
//
// Work that has to happen on that thread but arrives on a goroutine goes
// through Do. Do only enqueues and wakes the loop, so it is safe to call from
// anywhere including the loop itself; whoever owns the loop calls Drain to run
// what has queued up.
package mainthread

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// init pins the main goroutine to the thread the process started on. It has to
// happen in an init rather than in Run: package init runs on that thread before
// main and before any goroutine exists, whereas by the time an application has
// started a few goroutines and called Run the scheduler may already have moved
// the main goroutine to another thread, and AppKit then refuses to make a
// window at all.
func init() {
	runtime.LockOSThread()
}

var (
	owner atomic.Pointer[string]

	wakeMu sync.Mutex
	wake   func()

	queueMu sync.Mutex
	queue   []func()
)

// Claim records name as the owner of the platform event loop and reports
// whether it got it. The first caller wins; later callers get false and must
// attach to the running loop rather than start one of their own. A surface that
// gets false still works -- a webview window built while the tray's loop runs
// is a normal window -- it just must not call the platform's run function.
func Claim(name string) bool {
	return owner.CompareAndSwap(nil, &name)
}

// Owner names the surface driving the loop, or "" while nothing does.
func Owner() string {
	if p := owner.Load(); p != nil {
		return *p
	}
	return ""
}

// Claimed reports whether any surface is driving the loop. A surface that is
// asked to show a window consults this to choose between running the loop and
// attaching to the one already going.
func Claimed() bool { return owner.Load() != nil }

// Release gives the loop up. The owner calls it when its loop returns, so a
// later surface can take over -- a tray-only application that opens its first
// window after the tray has stopped, for instance.
func Release() { owner.Store(nil) }

// SetWake replaces the platform wake with one of the loop owner's own. The
// per-platform Wake in this package already reaches every loop here, so an
// owner only needs this when it has something cheaper -- the Go-drawn surface
// passes its request-a-frame call, which is the same thing it already does for
// a repaint. Passing nil goes back to Wake.
func SetWake(fn func()) {
	wakeMu.Lock()
	wake = fn
	wakeMu.Unlock()
	if fn != nil && pending() {
		fn()
	}
}

// Do queues fn to run on the main thread and wakes the loop. It returns before
// fn runs, which is what makes it safe from the main thread as well: a call
// made from inside a frame function or a menu handler lands on the next Drain
// instead of deadlocking on a thread that is already busy.
func Do(fn func()) {
	if fn == nil {
		return
	}
	queueMu.Lock()
	queue = append(queue, fn)
	queueMu.Unlock()

	wakeMu.Lock()
	w := wake
	wakeMu.Unlock()
	if w == nil {
		w = Wake
	}
	w()
}

// Drain runs everything Do has queued. The loop owner calls it once per turn of
// the loop -- for the Go-drawn surface that is a frame start hook, for a
// webview or tray loop it is the wake callback. It must run on the main thread;
// calling it anywhere else defeats the point of the queue.
//
// Each queued function runs outside the lock, so a function that calls Do
// queues onto the next Drain rather than deadlocking on this one.
func Drain() {
	for {
		queueMu.Lock()
		if len(queue) == 0 {
			queueMu.Unlock()
			return
		}
		batch := queue
		queue = nil
		queueMu.Unlock()

		for _, fn := range batch {
			fn()
		}
	}
}

func pending() bool {
	queueMu.Lock()
	defer queueMu.Unlock()
	return len(queue) > 0
}
