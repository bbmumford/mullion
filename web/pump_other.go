//go:build !linux

package web

// Pump does nothing where a WebView window needs no servicing of its own.
//
// NSApp services every window in the application and a Win32 message pump
// retrieves every message for its thread, so a WebView window attached to
// another surface's loop is already being serviced. See pump_linux.go for the
// platform where that is not true.
func Pump() {}
