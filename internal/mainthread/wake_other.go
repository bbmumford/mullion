//go:build android || js || (!darwin && !windows && !linux)

package mainthread

// Wake does nothing on the platforms with no main-queue primitive to post to.
//
// Android runs the Go-drawn surface over a JNI-driven GLES loop and the web
// build is single-threaded; in both, the only loop in the process is that
// surface's own, and it calls Drain from its frame hook. Queued work therefore
// still runs -- at the next frame rather than immediately -- and nothing here
// needs a platform call. A platform that gains a second loop owner needs a real
// wake in a file of its own.
func Wake() {}
