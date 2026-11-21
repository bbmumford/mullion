//go:build !darwin && !linux && !windows

package web

import "unsafe"

// askQuestion reports that this platform has no alert to show.
//
// iOS and Android both have one, but it is presented into a view controller or
// an activity rather than shown from a call that blocks until answered, so it
// does not fit this shape; a wasm build would have to ask the browser.
func askQuestion(window unsafe.Pointer, opts QuestionOptions) (bool, error) {
	return false, ErrUnsupported
}
