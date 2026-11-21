//go:build darwin

package web

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/bbmumford/mullion/internal/mainthread"
	"github.com/ebitengine/purego/objc"
)

// NSAlertFirstButtonReturn. Buttons are numbered from the first one added, so
// the affirmative answer is the first and this is the only value to compare.
const nsAlertFirstButtonReturn = 1000

// NSAlertStyle. Informational is right for a question: warning draws a caution
// badge, which overstates a consent prompt.
const nsAlertStyleInformational = 1

// askQuestion shows an NSAlert and reports whether the first button was chosen.
//
// window is unused: runModal presents application-modally, as the file panels
// here do, so there is no parent to attach. Running it as a sheet on a parent
// would need a completion handler and a nested loop, and a tray-only process
// has no window to sheet onto.
func askQuestion(window unsafe.Pointer, opts QuestionOptions) (bool, error) {
	// AppKit has to be in the process before NSAlert can be looked up. Nothing
	// else will have loaded it when the asker is a tray-only application or a
	// Go-drawn window, both of which reach AppKit by their own route.
	if err := ensureInit(); err != nil {
		return false, err
	}

	// Already on the main thread -- a frame function, a tray menu handler, a
	// WebKit callback -- so run the alert here. Queueing it and waiting would
	// block the only thread that could run it.
	if mainthread.OnMainThread() {
		return runAlert(opts)
	}
	type answer struct {
		ok  bool
		err error
	}
	ch := make(chan answer, 1)
	mainthread.Do(func() {
		ok, err := runAlert(opts)
		ch <- answer{ok, err}
	})
	r := <-ch
	return r.ok, r.err
}

// runAlert builds and runs the alert. It must be on the main thread.
//
// The classes are fetched with objc.GetClass rather than this package's class
// helper, which panics when a class is missing: an alert that cannot be built
// should be an error the caller can report, not the end of the process.
func runAlert(opts QuestionOptions) (bool, error) {
	alertClass := objc.GetClass("NSAlert")
	if alertClass == 0 {
		return false, errors.New("webview: NSAlert not available")
	}
	// runModal needs an NSApplication. A tray or a window will have made one
	// already; a process that has made neither gets one here, and asking for
	// the shared instance is enough -- the alert runs its own modal loop and
	// does not need the application's.
	if appClass := objc.GetClass("NSApplication"); appClass != 0 {
		objc.ID(appClass).Send(sel("sharedApplication"))
	}

	var (
		answered bool
		failed   error
	)
	autorelease(func() {
		alert := objc.ID(alertClass).Send(sel("alloc")).Send(sel("init"))
		if alert == 0 {
			failed = errors.New("webview: could not create an NSAlert")
			return
		}
		defer alert.Send(sel("release"))

		alert.Send(sel("setAlertStyle:"), nsAlertStyleInformational)
		alert.Send(sel("setMessageText:"), nsstr(opts.Title))
		if opts.Message != "" {
			alert.Send(sel("setInformativeText:"), nsstr(opts.Message))
		}
		// Order matters: the first button added is the default, the one Return
		// activates, and the one NSAlertFirstButtonReturn names.
		confirm := alert.Send(sel("addButtonWithTitle:"), nsstr(opts.Confirm))
		alert.Send(sel("addButtonWithTitle:"), nsstr(opts.Cancel))
		if opts.Destructive && confirm != 0 {
			// macOS 11 and later draw a destructive button in red; older
			// systems do not know the selector and are left alone.
			if confirm.Send(sel("respondsToSelector:"), sel("setHasDestructiveAction:")) != 0 {
				confirm.Send(sel("setHasDestructiveAction:"), true)
			}
		}
		answered = int(alert.Send(sel("runModal"))) == nsAlertFirstButtonReturn
	})
	if failed != nil {
		return false, fmt.Errorf("webview: asking the question: %w", failed)
	}
	return answered, nil
}
