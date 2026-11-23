# web

Package `web` opens windows rendered by the WebView the operating system
already ships (WKWebView on macOS, WebView2 on Windows, WebKitGTK on Linux) and
lets the page call Go functions. No browser engine is bundled and no cgo is
needed: the system libraries are reached through
[purego](https://github.com/ebitengine/purego).

It is an altered version of [glaze](https://github.com/crgimenes/glaze);
[ALTERED.md](ALTERED.md) says how it differs, and [LICENSE](LICENSE) is glaze's
licence.

## Use

Most applications use it through mullion, which runs it under the module's one
event loop beside a tray, a menu bar or a Go-drawn window:

```go
err := mullion.Run(mullion.App{
	Name: "Hello",
	Window: &mullion.Window{
		Title: "Hello", Width: 640, Height: 400,
		Page: &mullion.Page{
			HTML: `<button onclick="greet('world').then(t => document.body.append(t))">Greet</button>`,
			Bind: map[string]any{
				"greet": func(name string) string { return "Hello, " + name },
			},
		},
	},
})
```

A bound function runs on a goroutine, not the main thread, and its result
reaches the page as the resolution of a promise. It may return nothing, a
value, an error, or a value and an error. `Session.OpenWindow` opens further
WebView windows at any time, and `examples/web-window` is a fuller program.

The package also stands on its own: `New`, `NewWindow` and `NewWithOptions`
create a `WebView` directly, and `AppWindow`, `Events`, `BindMethods`,
`RenderHTML` and custom schemes (`Options`, `SchemeHandler`) are here for an
application that drives the WebView itself.

## Requirements

- **macOS**: nothing extra; WebKit ships with the system.
- **Windows**: the Microsoft Edge WebView2 Runtime. It is found through the
  registry, and `New` returns an error when it is missing. No
  `WebView2Loader.dll` is shipped: the package calls the runtime's own
  environment-creation export, which is undocumented, so a future runtime that
  changes it makes `New` fail with an error rather than crash.
- **Linux**: a system WebKitGTK, on GTK 4 or GTK 3, chosen at run time.

### Linux shared libraries

These are the libraries loaded with `dlopen` at startup. They must be on the
dynamic linker's path and of the same architecture as the binary.

- Always: `libglib-2.0.so.0`, `libgobject-2.0.so.0`.
- `libwebkitgtk-6.0.so.4` decides the stack. If it loads, GTK 4:
  `libgtk-4.so.1`, `libwebkitgtk-6.0.so.4`, `libjavascriptcoregtk-6.0.so.1`.
  Otherwise GTK 3: `libgtk-3.so.0`, `libwebkit2gtk-4.1.so.0` (or
  `libwebkit2gtk-4.0.so.37`), `libjavascriptcoregtk-4.1.so.0` (or
  `libjavascriptcoregtk-4.0.so.18`). Only one stack is ever loaded: GTK 3 and
  GTK 4 in one process corrupt GTK's type system.
- On GTK 4 the file dialogs also load `libgio-2.0.so.0`, which comes with GLib.

Installing WebKitGTK brings GTK and GLib with it:

- Debian and Ubuntu: `apt install libwebkit2gtk-4.1-0` (GTK 3) or
  `libwebkitgtk-6.0-4` (GTK 4)
- Fedora: `dnf install webkit2gtk4.1` or `webkitgtk6.0`
- Arch: `pacman -S webkit2gtk-4.1` or `webkitgtk-6.0`
- Nix: add `webkitgtk_4_1` or `webkitgtk_6_0` to the shell's inputs, since the
  libraries are not on the default loader path.

When `New` reports that none of the libraries could be loaded, see what the
linker can find:

```sh
ldconfig -p | grep -E 'libwebkit(2)?gtk|libjavascriptcoregtk|libgtk-[34]'
```

The file dialogs and the yes/no prompt in mullion's `dialog` package load GTK
through the same path, so they need these libraries too.
