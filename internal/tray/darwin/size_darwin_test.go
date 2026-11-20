//go:build darwin

package darwin

import "github.com/ebitengine/purego/objc"

// sendForSize reads an NSSize return value, which comes back in floating-point
// registers rather than the integer one the Send methods here use.
func sendForSize(id ID, sel SEL) NSSize {
	return objc.Send[NSSize](objc.ID(id), objc.SEL(sel))
}
