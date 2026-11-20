package tray

// The tray's platform calls -- NSStatusItem, Shell_NotifyIcon, the
// StatusNotifierItem registration -- all have to happen on the thread the
// process started on. That pinning used to happen in this package's own init,
// which is correct for a build that contains nothing else but fights the Go-drawn
// and webview surfaces here, each of which wanted the same thread for itself.
// mainthread's init pins it once for all of them, so importing it is all this
// package needs; see internal/mainthread for why pinning cannot wait for Run.
import _ "github.com/bbmumford/mullion/internal/mainthread"
