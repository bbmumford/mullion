// Package shell drives the parts of a desktop application that sit outside its
// windows: the notification-area icon, the application menu bar, and system
// notifications.
//
// The two trees these came from each had their own menu type -- the tray's was
// a builder that returned mutable item handles, the menu bar's was a slice of
// value structs with keyboard shortcuts -- and an application that wanted both
// described its commands twice. Here one Item describes a command once and
// either surface takes it; a field that one surface cannot honour is documented
// on the field, and ignored rather than refused, so the same Menu can be used
// in both places.
//
// Nothing in this package starts an event loop. A Tray attaches to whichever
// loop the process is running (see internal/mainthread), which is what lets a
// tray, a Go-drawn window and a webview window coexist in one process.
//
// Platform support follows the underlying backends: the tray works on macOS,
// Windows and Linux (StatusNotifierItem over D-Bus); the menu bar works on
// macOS and Windows, and on Linux installs the shortcuts without drawing a bar.
// Everywhere else both return an error wrapping errors.ErrUnsupported.
package shell
