//go:build !darwin && !linux && !windows

package web

import (
	"errors"
	"fmt"
	"unsafe"
)

// This file exists so the module compiles for the targets the Go-drawn surface
// reaches and this one does not -- iOS, Android and the web build. There is no
// system WebView to bind on any of them: on iOS and Android the host is the
// platform's own view hierarchy, and in a wasm build the program is already
// inside a browser engine. Returning an error here rather than leaving the
// symbol out means an application can keep one import list for every target and
// decide at runtime what it can offer.

// ErrUnsupported reports that this platform has no WebView binding. It wraps
// errors.ErrUnsupported, so one errors.Is covers it together with the tray, the
// menu bar and the file dialogs.
var ErrUnsupported = fmt.Errorf("web: no system WebView on this platform: %w", errors.ErrUnsupported)

// Init reports that there is nothing to initialise.
func Init() error { return ErrUnsupported }

// New reports that this platform has no WebView to create.
func New(debug bool) (WebView, error) { return nil, ErrUnsupported }

// NewWindow reports that this platform has no WebView to embed.
func NewWindow(debug bool, window unsafe.Pointer) (WebView, error) { return nil, ErrUnsupported }

// NewWithOptions reports that this platform has no WebView to create.
func NewWithOptions(opts Options) (WebView, error) { return nil, ErrUnsupported }
