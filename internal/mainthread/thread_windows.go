//go:build windows

package mainthread

import "golang.org/x/sys/windows"

// mainTID is the thread the process started on, read in an init so it is
// captured on that thread before the application can move off it. It is also
// the thread that owns every window made here, because Win32 delivers a
// window's messages only to its creating thread.
var mainTID = windows.GetCurrentThreadId()

// OnMainThread reports whether the caller is running on the thread this package
// pinned. See the darwin file for why work that waits for a result has to ask.
func OnMainThread() bool { return windows.GetCurrentThreadId() == mainTID }
