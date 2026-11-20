//go:build darwin

package mainthread

import (
	"sync"

	"github.com/ebitengine/purego"
)

// OnMainThread reports whether the caller is running on the thread this package
// pinned.
//
// Work that has to happen there and then wait for a result -- a modal file
// panel is the case that matters -- must know this. Called from a goroutine it
// can queue the work with Do and block on the answer; called from the main
// thread itself, queueing and then blocking would stop the only thread that
// could ever run the queue, and the application would hang with no window and
// no error. A button handler inside a frame function is on the main thread, so
// that is not a rare path.
//
// It reads false if libSystem cannot be resolved, which routes such work
// through the queue: slower to arrive, and never a deadlock.
func OnMainThread() bool { return pthreadMainNp()() }

var pthreadMainNp = sync.OnceValue(func() func() bool {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return func() bool { return false }
	}
	var mainNp func() int32
	purego.RegisterLibFunc(&mainNp, lib, "pthread_main_np")
	return func() bool { return mainNp() != 0 }
})
