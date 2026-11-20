//go:build darwin

package web

import (
	"unsafe"

	"github.com/bbmumford/mullion/internal/mainthread"
)

// fileDialogFrom runs an NSOpenPanel or NSSavePanel and returns the paths
// chosen.
//
// It routes through mainthread rather than leaving the thread to this package's
// own performOnMain. performOnMain decides from uiIsMain, which is recorded
// when a WebView is first created: in a process that never creates one -- a
// Go-drawn window with no web surface at all -- it stays false and
// performOnMain runs its argument inline, on whatever goroutine called. That
// would drive a modal AppKit panel off the main thread, which is exactly what
// AppKit does not allow. Going through mainthread puts the panel on the thread
// this module pinned, whether or not a WebView exists.
//
// window is unused: these panels are application-modal, so there is no parent
// to pass. The parameter keeps one signature across the platforms where Windows
// and Linux do need it.
func fileDialogFrom(window unsafe.Pointer, kind fileDialogKind, opts FileDialogOptions) ([]string, error) {
	// Already on the main thread -- a frame function, a menu handler, a
	// WebKit callback -- so run the panel here. Queueing it and waiting would
	// block the only thread that could run it.
	if mainthread.OnMainThread() {
		return runFilePanel(kind, opts), nil
	}
	ch := make(chan []string, 1)
	mainthread.Do(func() { ch <- runFilePanel(kind, opts) })
	return <-ch, nil
}

// runFilePanel presents the panel for kind. It must run on the main thread;
// performOnMain inside the presenters is a passthrough once it does.
func runFilePanel(kind fileDialogKind, opts FileDialogOptions) []string {
	if kind == dialogSaveFile {
		if path := presentSavePanelFor(opts); path != "" {
			return []string{path}
		}
		return nil
	}
	canFiles := kind != dialogOpenDirectory
	canDirs := kind == dialogOpenDirectory
	multiple := kind == dialogOpenFiles
	return presentOpenPanelFor(opts, canFiles, canDirs, multiple)
}
