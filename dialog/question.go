package dialog

import "unsafe"

// A Question is a yes/no prompt: a short question the person answers before
// anything else happens.
//
// It is the platform's own alert, not a window of your own. That matters for a
// consent prompt, which is the case this exists for: an alert needs no WebView
// runtime present, cannot be mistaken for page content, and is placed and
// focused by the window manager as a prompt rather than as an ordinary window.
type Question struct {
	// Title is the bold first line. Keep it a question.
	Title string

	// Message is the detail under the title, and may be empty.
	Message string

	// Confirm labels the affirmative button. Empty means "Yes".
	//
	// A verb naming the action reads better than Yes on every platform, and on
	// macOS it is what the guidelines ask for: "Allow" rather than "Yes".
	Confirm string

	// Cancel labels the negative button. Empty means "No".
	Cancel string

	// Destructive marks the affirmative answer as one that destroys something,
	// where the platform draws such a button differently. Elsewhere it is
	// ignored.
	Destructive bool

	// Parent is the window the prompt belongs to -- a Go-drawn window's native
	// handle, or a WebView's Window(). Nil asks without a parent, which is
	// right for a tray, whose menu belongs to no window.
	Parent unsafe.Pointer
}

// Ask shows the question and returns what the person answered: true for the
// affirmative button, false for the negative one.
//
// Dismissing the prompt without answering -- Escape, the close button, the
// window menu -- is false. A question therefore always has an answer, and a
// caller never has to tell "no" from "never replied"; there is no third state
// to handle and no deadline to run.
//
// It blocks the calling goroutine until the person answers, and may be called
// from the main thread as well, where it runs the prompt in place rather than
// waiting on a thread it would be blocking.
//
// Platform support is macOS, Windows and Linux. Elsewhere it returns false with
// an error wrapping errors.ErrUnsupported, which a caller should treat as "I
// could not ask" and not as a no.
func Ask(q Question) (bool, error) {
	return askQuestion(q)
}
