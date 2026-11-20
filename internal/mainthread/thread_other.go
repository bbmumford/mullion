//go:build android || js || (!darwin && !windows && !linux)

package mainthread

// OnMainThread reports true on the targets with one thread to be on.
//
// A wasm build is single-threaded. On Android the Go-drawn surface is the only
// loop in the process and its frame function is the only place application code
// runs, which is that thread. Answering true keeps work that waits for a result
// running inline instead of queueing it behind a drain that the caller is
// blocking.
func OnMainThread() bool { return true }
