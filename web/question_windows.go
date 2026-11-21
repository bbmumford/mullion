//go:build windows

package web

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/bbmumford/mullion/internal/mainthread"
)

// MessageBox flags and returns. The task-dialog API would give custom button
// labels, but it lives in comctl32 and needs a manifest naming version 6 of it:
// without one the call fails at runtime, and a manifest is the application's to
// ship, not a library's to require. So the buttons here are the system's Yes
// and No, and QuestionOptions.Confirm and Cancel are not used.
const (
	mbYesNo         = 0x00000004
	mbIconQuestion  = 0x00000020
	mbTaskModal     = 0x00002000
	mbSetForeground = 0x00010000
	idYes           = 6
)

var (
	user32MessageBox = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
)

// askQuestion shows a MessageBox and reports whether Yes was chosen.
//
// MessageBoxW pumps its own messages while it is up, so it needs no loop from
// the caller -- but it must run on the thread that owns the parent window,
// because Win32 delivers that window's messages nowhere else. Passing no parent
// is why mbTaskModal is set: it disables the calling thread's other windows,
// which is what a parentless modal prompt should do.
func askQuestion(window unsafe.Pointer, opts QuestionOptions) (bool, error) {
	title, err := syscall.UTF16PtrFromString(opts.Title)
	if err != nil {
		return false, fmt.Errorf("webview: question title: %w", err)
	}
	// The message box shows one block of text; the detail reads as a second
	// paragraph under the question.
	body := opts.Title
	if opts.Message != "" {
		body = opts.Title + "\n\n" + opts.Message
	}
	text, err := syscall.UTF16PtrFromString(body)
	if err != nil {
		return false, fmt.Errorf("webview: question message: %w", err)
	}

	ask := func() uintptr {
		// The conversions are written in the argument list of a LazyProc.Call,
		// which carries //go:uintptrescapes and so keeps both strings alive and
		// unmoved for the duration.
		r, _, _ := user32MessageBox.Call(
			uintptr(window),
			uintptr(unsafe.Pointer(text)),
			uintptr(unsafe.Pointer(title)),
			mbYesNo|mbIconQuestion|mbTaskModal|mbSetForeground)
		return r
	}

	if onUIThread() || mainthread.OnMainThread() {
		return ask() == idYes, nil
	}
	ch := make(chan uintptr, 1)
	mainthread.Do(func() { ch <- ask() })
	return <-ch == idYes, nil
}
