// Package web is the WebView surface: a window whose contents are a page
// rendered by the system's own browser engine.
//
// It is glaze, merged into mullion. The import path and the package name
// changed; the design did not. See ALTERED.md beside this file.
//
// # Which engine
//
// The platform's, never a bundled one: WKWebView on macOS and iOS, WebKitGTK
// on Linux, WebView2 on Windows. The binary therefore carries no engine and
// stays the size of a Go program, and the pages it shows are rendered by
// something the operating system already updates. The cost is that Linux needs
// WebKitGTK present and Windows needs the WebView2 runtime, which recent
// Windows installs have and older ones may not.
//
// # Calling between Go and the page
//
// Bind exposes a Go function to JavaScript under a name, and the page calls it
// and gets a promise. The bound function runs on a background goroutine, not
// the UI thread, so it may do work; what it must not do is touch the WebView
// directly. Dispatch puts that back on the UI thread.
//
// # Threads
//
// Everything that touches a window belongs on the thread that created it.
// Within mullion that is the thread internal/mainthread pinned, and
// Session.Do is the way to reach it; this package's own dispatcher is on top
// of the same one.
//
// # Where this sits
//
// A program using mullion does not usually import this package. mullion.Page
// describes a WebView window, mullion.Session opens one, and the dialog
// package reaches the file dialogs and the yes/no prompt that live here --
// those are available to both surfaces, not only to a page.
package web
