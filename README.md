# mullion

mullion is a Go module for desktop applications. Under one event loop it runs a
window whose interface is drawn in Go, windows rendered by the operating
system's WebView, a tray icon with its menu, an application menu bar, file
dialogs, a yes/no prompt and system notifications. It builds without cgo for
macOS, Windows, Linux and the browser (js/wasm).

It is built from three open-source projects, altered so that they share one
loop: [go-shirei](https://github.com/hasenj/go-shirei) (the Go-drawn interface),
[glaze](https://github.com/crgimenes/glaze) (the WebView window and the menu
bar) and [systray](https://github.com/gogpu/systray) (the tray). Each of them
expected to own the process, pinning the main thread and running the
platform's loop from its own `Run`; mullion pins the thread once and decides
which surface runs the loop. Their code stays under their licences: see
[Licences](#licences).

In ORBTR, mullion is the Go tray and shell for Windows and Linux: the device
tray is built on it and reaches the orbtrd daemon over orbtrd's local IPC. On
macOS the ORBTR tray is a separate Swift application.

## Use

```go
package main

import (
	_ "embed"
	"log"

	"github.com/bbmumford/mullion"
	"github.com/bbmumford/mullion/icon"
	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/widgets"
	"github.com/bbmumford/mullion/shell"
)

//go:embed icon.png
var iconPNG []byte

func main() {
	var name string
	var session *mullion.Session

	err := mullion.Run(mullion.App{
		Name: "Notes",
		Icon: icon.PNG(iconPNG),
		Window: &mullion.Window{
			Title: "Notes", Width: 900, Height: 600,
			Draw: func() {
				native.Container(native.Attrs(native.Pad(20), native.Gap(12)), func() {
					native.Label("Name")
					widgets.TextInput(&name)
				})
			},
		},
		Tray: &shell.Tray{Tooltip: "Notes"},
		MenuBar: shell.Menu{
			{Label: "Notes", Submenu: shell.Menu{
				{Label: "Quit", Shortcut: "cmd+q", Do: func() { session.Quit() }},
			}},
		},
		OnReady: func(s *mullion.Session) error {
			session = s
			return nil
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

`mullion.Run` takes an `App` naming the surfaces, and which one drives the
event loop follows from them, not from a setting: a Go-drawn window
(`Window.Draw`) if there is one, otherwise a WebView window (`Window.Page`),
otherwise the tray. The Go-drawn backends show their window by entering the
platform's loop, which a process can do once; a WebView window and a tray icon
can be built on a loop that is already running, so they attach to whichever
surface drives it. The tray and the menu bar are put up once the loop is
running, which is the only time the platform accepts a status item.

`OnReady` runs once on the main thread after everything is attached. The
`Session` it receives opens further WebView windows (`OpenWindow`) and small
Go-drawn popup windows (`OpenPopup`), shows notifications (`Notify`), moves,
resizes and activates the window, starts audio output, and ends the
application (`Quit`, after which `OnQuit` runs). Window, menu and tray calls
are valid only on the main thread: `Session.Do` carries work there from any
goroutine and returns before it runs, so it is safe from a frame function or a
menu handler too.

A tray-only application leaves `Window` out:

```go
err := mullion.Run(mullion.App{
	Name: "Helper",
	Tray: &shell.Tray{
		Icon:    icon.PNG(trayPNG),
		Tooltip: "Helper",
		Menu: shell.Menu{
			{Label: "Open the site", Do: func() { _ = mullion.OpenURL("https://example.com") }},
			{Label: "Share diagnostics…", Do: func() {
				go func() {
					ok, err := dialog.Ask(dialog.Question{
						Title:   "Share diagnostics?",
						Confirm: "Share",
						Cancel:  "Not now",
					})
					switch {
					case errors.Is(err, dialog.ErrNotShown):
						log.Print("the question could not be shown")
					case ok:
						log.Print("shared")
					}
				}()
			}},
			shell.Separator(),
			{Label: "Quit", Do: func() { session.Quit() }},
		},
	},
	OnReady: func(s *mullion.Session) error {
		session = s
		return nil
	},
})
```

`dialog.Ask` blocks until the person answers, and its error separates "the
person said no" (`false, nil`) from "nobody was asked" (an error wrapping
`dialog.ErrNotShown`).

### Examples

```sh
go run ./examples/native-window    # a window drawn in Go
go run ./examples/web-window       # a WebView window calling Go functions from the page
go run ./examples/tray-only        # no window, only a tray icon and its menu
go run ./examples/full-shell       # a Go-drawn window, menu bar, tray, file dialog and a WebView window
go run ./examples/widget-gallery   # every widget of the Go-drawn surface, live
```

## Packages

| package | what it does |
|---|---|
| `mullion` | `App`, `Run`, `Session`, `Window`, `Page`, `OpenURL`: the one event loop and the window that drives it |
| `native`, `native/widgets`, `native/app`, `native/ext/...` | the Go-drawn surface: a frame function describes the window every frame, with layout, text, widgets, GPU rendering and a backend per platform |
| `web` | windows rendered by the system WebView (WKWebView, WebView2, WebKitGTK), with Go functions bound into the page's JavaScript |
| `shell` | the tray (`Tray`), one menu model (`Menu`, `Item`) for both the tray and the menu bar, `SetMenuBar`, notifications, menu-bar shortcuts |
| `dialog` | the platform's open, save and folder dialogs, and a yes/no prompt (`Ask`), for either surface |
| `icon` | one icon value, from PNG bytes, an image or a file, with a dark variant and the macOS template flag |
| `bind` | change notification for values a frame function reads, so a change made from a goroutine repaints |
| `cmd/mullion-web`, `cmd/mullion-mobilerun`, `cmd/mullion-bundle` | build tools for the Go-drawn surface: a static site for js/wasm, development installs on iOS and Android, release bundles |

## Platforms

| | macOS | Windows | Linux | js/wasm | iOS, Android |
|---|---|---|---|---|---|
| Go-drawn window | yes | yes | Wayland and X11 | yes | yes |
| WebView window | WKWebView | WebView2 | WebKitGTK | no | no |
| Tray | status item | notification area | StatusNotifierItem over D-Bus | no | no |
| Menu bar | yes | yes, on a window | shortcuts only | no | no |
| File dialogs, yes/no prompt | yes | yes | yes | no | no |
| Notifications | yes, from an `.app` bundle | yes | yes | no | no |
| Needs cgo | no | no | no | no | yes |

On macOS, Windows, Linux and js/wasm the system libraries are reached through
[purego](https://github.com/ebitengine/purego) and `golang.org/x/sys`, so a
build needs no C toolchain and cross-compiles from any host:

```sh
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
```

What the platform must provide at run time:

- **Windows**: the WebView2 runtime, for WebView windows.
- **Linux**: WebKitGTK (GTK 4 with webkitgtk-6.0, or GTK 3 with webkit2gtk-4.1
  or 4.0) for WebView windows. The file dialogs and the yes/no prompt load GTK
  through the same path, so they need WebKitGTK too, even in an application
  that opens no WebView. The tray needs a StatusNotifierItem host: KDE, XFCE,
  Cinnamon and MATE have one, and GNOME has one with the AppIndicator
  extension.
- **iOS and Android**: cgo and the platform's toolchain (Xcode, the Android
  NDK). Only the Go-drawn surface has a backend written for them. Go's `ios`
  target also carries the `darwin` build tag and `android` the `linux` tag, so
  the macOS and Linux code of the other surfaces compiles into those builds,
  but it was not written for them and has not been run there.

On js/wasm, and on any target built with none of the `darwin`, `linux` and
`windows` tags, the WebView, tray, menu bar and dialogs return an error
wrapping `errors.ErrUnsupported`, so one `errors.Is` covers them all and the
module still compiles.

The module's minimum Go version is the `go` line in `go.mod`.

## Limits

- **One Go-drawn window per process.** The Go-drawn window has to be the one
  `App.Window` names. `Session.OpenWindow` opens WebView windows, and says so
  rather than ignore a `Draw` it was given. `Session.OpenPopup` opens the small
  borderless windows the Go-drawn window needs beyond its bounds (a completion
  list, a tooltip), which it draws and owns.
- **No menu bar is drawn on Linux.** GTK 3 and GTK 4 differ too much for one
  bar. The menu's shortcuts are installed on the window instead, with `cmd`
  read as `ctrl`, and when the Go-drawn surface drives the loop the runtime
  matches them from its own keyboard input, so one `MenuBar` answers the same
  keys on all three desktops.
- **On Windows a menu bar belongs to a window.** An application with a
  `MenuBar` and no `Window` has nothing to attach it to there, and `Run` stops
  with that error.
- **A Go-drawn window has no native handle on Linux.** `Session.NativeWindow`
  is an `NSWindow` on macOS and an `HWND` on Windows; a Wayland or X11 window is
  not what GTK's dialogs take as a parent, so it reads nil there and dialogs
  open without a parent.
- **Notifications need a tray**, because on Windows a notification is a balloon
  belonging to a notification-area icon; `Session.Notify` refuses without one.
  On macOS they also need the process to run from an `.app` bundle, and fail
  with an error saying so otherwise.
- **`Tray.Bounds` is complete only once the loop runs**: read between `Show` and
  the loop it can come back partly filled.
- **The snapshot tests in `native/widgets`** compare rendered pixels with
  go-shirei's reference images, so their result depends on the fonts installed.

## Licences

The code written for mullion is Copyright (c) 2026 Brandon Miller-Mumford,
under the MIT licence in [LICENSE](LICENSE). The parts derived from go-shirei
(`native/`, `cmd/`, under the zlib licence), glaze (`web/`,
`internal/menubar/`, MIT) and systray (`shell/`, `internal/tray/`,
`internal/ffiaudit/`, MIT) stay under their own licences and their authors'
copyright. Each of those directories carries its upstream's licence file
(`COPYING` or `LICENSE`), so a package's licence can be read from its own
directory, and an `ALTERED.md` saying how it differs from the commit merged;
[licenses/](licenses) holds every licence text in one place.
[NOTICE](NOTICE) names each project, the commit merged and where its code
lives, and lists the third-party parts that came inside go-shirei with their
own terms: the Android native-app glue (Apache 2.0), the Wayland bindings (MIT),
Go's `wasm_exec.js` (BSD) and three fonts (SIL Open Font License).

## Contributing

[AGENTS.md](AGENTS.md) holds the repository's two rules, for people and coding
agents alike: no temporary documents in the repository, and conventional
commit messages that stand on their own. [CLAUDE.md](CLAUDE.md) imports it for
Claude Code.
