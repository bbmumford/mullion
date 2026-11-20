//go:build windows

package web

import (
	"unsafe"

	"github.com/bbmumford/mullion/internal/mainthread"
)

// fileDialogFrom runs an IFileOpenDialog or IFileSaveDialog owned by window.
//
// The COM dialog pumps its own messages while it is up, so it needs no loop of
// its own from the caller; what it needs is the thread that owns the parent
// window, because Win32 delivers that window's messages nowhere else.
func fileDialogFrom(window unsafe.Pointer, kind fileDialogKind, opts FileDialogOptions) ([]string, error) {
	if err := ensureDialogInit(); err != nil {
		return nil, err
	}
	save := kind == dialogSaveFile
	pickFolders := kind == dialogOpenDirectory
	multi := kind == dialogOpenFiles
	parent := uintptr(window)

	// Already on the UI thread: run here. Queueing the work and then waiting
	// would block the only thread that could run it.
	if onUIThread() || mainthread.OnMainThread() {
		return runFileDialog(parent, save, pickFolders, multi, opts)
	}
	type outcome struct {
		paths []string
		err   error
	}
	ch := make(chan outcome, 1)
	mainthread.Do(func() {
		paths, err := runFileDialog(parent, save, pickFolders, multi, opts)
		ch <- outcome{paths, err}
	})
	r := <-ch
	return r.paths, r.err
}
