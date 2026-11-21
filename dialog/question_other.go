//go:build !darwin && !linux && !windows

package dialog

import (
	"errors"
	"fmt"
)

// askQuestion reports that this platform has no alert to show.
//
// iOS and Android do have one, but it is presented into a view controller or an
// activity rather than shown from a blocking call, so it does not fit this
// call's shape; a wasm build would have to ask the browser.
func askQuestion(q Question) (bool, error) {
	return false, fmt.Errorf("dialog: no native question prompt on this platform: %w", errors.ErrUnsupported)
}
