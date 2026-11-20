//go:build linux

package web

import (
	"unsafe"

	"github.com/bbmumford/mullion/internal/mainthread"
)

// fileDialogFrom runs a GtkFileChooserNative parented to window.
//
// The chooser iterates the GTK main context itself until the user answers, so
// it needs GTK loaded but no GTK main loop of its own running: a Go-drawn
// window, whose loop is Wayland or X11, opens it the same way a WebView window
// does. Loading GTK is what ensureDialogInit does, and on a machine with no
// WebKitGTK that is where this fails -- with the error that names the missing
// library, not a crash.
func fileDialogFrom(window unsafe.Pointer, kind fileDialogKind, opts FileDialogOptions) ([]string, error) {
	if err := ensureDialogInit(); err != nil {
		return nil, err
	}
	action, multi := gtkFileChooserActionOpen, false
	switch kind {
	case dialogOpenFiles:
		multi = true
	case dialogOpenDirectory:
		action = gtkFileChooserActionSelectFolder
	case dialogSaveFile:
		action = gtkFileChooserActionSave
	}
	parent := uintptr(window)

	// Already on the thread that owns GTK: run here. Queueing the work and then
	// waiting would block the only thread that could run it.
	if onUIThread() || mainthread.OnMainThread() {
		return runFileChooser(parent, action, multi, opts), nil
	}
	ch := make(chan []string, 1)
	mainthread.Do(func() { ch <- runFileChooser(parent, action, multi, opts) })
	return <-ch, nil
}
