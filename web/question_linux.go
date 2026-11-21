//go:build linux

package web

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/bbmumford/mullion/internal/mainthread"
	"github.com/ebitengine/purego"
)

// GtkMessageType, GtkButtonsType and GtkDialogFlags.
const (
	gtkMessageQuestion    = 2
	gtkButtonsNone        = 0
	gtkDialogModal        = 1
	gtkDialogDestroyChild = 2
)

// The response ids this dialog adds its buttons with. GTK's own ids are
// negative, so positive ones cannot collide with a GTK_RESPONSE_DELETE_EVENT
// raised when the person closes the window instead of answering.
const (
	questionConfirmResponse = 1
	questionCancelResponse  = 2
)

var (
	questionOnce sync.Once
	questionErr  error

	// gtk_message_dialog_new is variadic, with the message format last. It is
	// called with a NULL format and no variadic arguments, which is why a fixed
	// five-argument signature is right, and the text is set afterwards with
	// set_markup over escaped text -- a format string built from the caller's
	// text would read any percent in it as a directive.
	gtkMessageDialogNew       func(parent uintptr, flags, msgType, buttons int, format uintptr) uintptr
	gtkMessageDialogSetMarkup func(dialog uintptr, markup string)
	gtkDialogAddButton        func(dialog uintptr, text string, responseID int) uintptr
	gtkWindowSetModalFn       func(window uintptr, modal bool)
	gtkWindowSetTransientFor  func(window, parent uintptr)
	gtkWindowDestroyFn        func(window uintptr)
	gtkWidgetDestroyFn        func(widget uintptr)
	gMarkupEscapeText         func(text string, length int) uintptr
	gFreeFn                   func(mem uintptr)
)

// ensureQuestionInit resolves the alert symbols once, on top of the GTK stack
// ensureInit already chose.
func ensureQuestionInit() error {
	if err := ensureInit(); err != nil {
		return err
	}
	questionOnce.Do(func() {
		defer func() {
			// A missing symbol makes purego panic rather than return, and a
			// desktop without one of these should get an error, not a dead
			// process.
			if r := recover(); r != nil {
				questionErr = fmt.Errorf("webview: no message dialog in this GTK: %v", r)
			}
		}()
		purego.RegisterLibFunc(&gtkMessageDialogNew, gtkLib, "gtk_message_dialog_new")
		purego.RegisterLibFunc(&gtkMessageDialogSetMarkup, gtkLib, "gtk_message_dialog_set_markup")
		purego.RegisterLibFunc(&gtkDialogAddButton, gtkLib, "gtk_dialog_add_button")
		purego.RegisterLibFunc(&gtkWindowSetModalFn, gtkLib, "gtk_window_set_modal")
		purego.RegisterLibFunc(&gtkWindowSetTransientFor, gtkLib, "gtk_window_set_transient_for")
		purego.RegisterLibFunc(&gMarkupEscapeText, glibLib, "g_markup_escape_text")
		purego.RegisterLibFunc(&gFreeFn, glibLib, "g_free")
		if gtk4 {
			purego.RegisterLibFunc(&gtkWindowDestroyFn, gtkLib, "gtk_window_destroy")
			return
		}
		purego.RegisterLibFunc(&gtkWidgetDestroyFn, gtkLib, "gtk_widget_destroy")
	})
	return questionErr
}

// askQuestion shows a GtkMessageDialog and reports whether the affirmative
// button was chosen.
//
// It runs the dialog with the show-plus-response-plus-iteration pattern the
// file chooser here already uses, rather than gtk_dialog_run, which GTK4
// removed. That also means it needs no GTK main loop of its own to be running,
// so a Go-drawn window whose loop is Wayland or X11 can ask a question.
func askQuestion(window unsafe.Pointer, opts QuestionOptions) (bool, error) {
	if err := ensureQuestionInit(); err != nil {
		return false, err
	}
	parent := uintptr(window)

	// Already on the thread that owns GTK: run it here. Queueing the dialog and
	// then waiting would block the only thread that could run it.
	if onUIThread() || mainthread.OnMainThread() {
		return runQuestion(parent, opts), nil
	}
	ch := make(chan bool, 1)
	mainthread.Do(func() { ch <- runQuestion(parent, opts) })
	return <-ch, nil
}

func runQuestion(parent uintptr, opts QuestionOptions) bool {
	flags := gtkDialogModal
	if parent != 0 {
		flags |= gtkDialogDestroyChild
	}
	dlg := gtkMessageDialogNew(parent, flags, gtkMessageQuestion, gtkButtonsNone, 0)
	if dlg == 0 {
		return false
	}
	defer func() {
		if gtk4 {
			gtkWindowDestroyFn(dlg)
			return
		}
		gtkWidgetDestroyFn(dlg)
	}()

	// The question is the primary text and the detail a second paragraph, which
	// is how GTK's own two-part message dialogs read. Both are escaped: markup
	// is what set_markup expects, and a stray ampersand in a caller's string
	// would otherwise be a parse error that shows no text at all.
	text := escapeMarkup(opts.Title)
	if opts.Message != "" {
		text += "\n\n" + escapeMarkup(opts.Message)
	}
	gtkMessageDialogSetMarkup(dlg, text)

	// Cancel first so it sits left of the affirmative, which is the order GNOME
	// asks for; the response id, not the position, decides the answer.
	gtkDialogAddButton(dlg, opts.Cancel, questionCancelResponse)
	gtkDialogAddButton(dlg, opts.Confirm, questionConfirmResponse)

	if parent != 0 {
		gtkWindowSetTransientFor(dlg, parent)
	}
	gtkWindowSetModalFn(dlg, true)

	return runWidgetDialog(dlg) == questionConfirmResponse
}

// runWidgetDialog shows a GtkDialog and pumps the main context until the person
// responds, returning the response id. It is runNativeDialog for a widget
// dialog rather than a GtkNativeDialog, which take different show calls.
func runWidgetDialog(dlg uintptr) int {
	dialogRespMu.Lock()
	dialogRespSeq++
	token := dialogRespSeq
	st := &dialogResp{}
	dialogRespStates[token] = st
	dialogRespMu.Unlock()
	defer func() {
		dialogRespMu.Lock()
		delete(dialogRespStates, token)
		dialogRespMu.Unlock()
	}()

	gSignalConnectData(dlg, "response", dialogResponseFn, token, 0, 0)
	if gtk4 {
		gtkWidgetSetVisible(dlg, true)
	} else {
		gtkWidgetShow(dlg)
	}
	for {
		dialogRespMu.Lock()
		done := st.done
		dialogRespMu.Unlock()
		if done {
			break
		}
		gMainContextIteration(0, true)
	}
	return st.response
}

// escapeMarkup escapes a Go string for Pango markup, through GLib so the rules
// are GLib's own.
func escapeMarkup(s string) string {
	if s == "" {
		return ""
	}
	out := gMarkupEscapeText(s, -1)
	if out == 0 {
		return ""
	}
	defer gFreeFn(out)
	return goStringFromC(out)
}

// goStringFromC copies a NUL-terminated C string into Go.
func goStringFromC(p uintptr) string {
	if p == 0 {
		return ""
	}
	var n int
	for ; *(*byte)(unsafe.Pointer(p + uintptr(n))) != 0; n++ {
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(p)), n))
}
