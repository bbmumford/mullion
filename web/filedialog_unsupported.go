//go:build !darwin && !linux && !windows

package web

import "unsafe"

// fileDialogFrom reports that this platform has no native file dialog to show.
//
// iOS and Android present a document picker through their own view hierarchy
// rather than a modal panel, and a wasm build can only ask the browser; none of
// them fits this call's shape, which returns filesystem paths.
func fileDialogFrom(window unsafe.Pointer, kind fileDialogKind, opts FileDialogOptions) ([]string, error) {
	return nil, ErrUnsupported
}
