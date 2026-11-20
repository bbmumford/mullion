//go:build linux && !android

package mainthread

import "golang.org/x/sys/unix"

// mainTID is the thread the process started on, read in an init so it is
// captured on that thread before the application can move off it.
var mainTID = unix.Gettid()

// OnMainThread reports whether the caller is running on the thread this package
// pinned. See the darwin file for why work that waits for a result has to ask.
func OnMainThread() bool { return unix.Gettid() == mainTID }
