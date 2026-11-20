// A window rendered by the system WebView, with Go functions called from the
// page.
//
// There is no browser engine in this binary: the window binds the WebView the
// operating system already ships. With no Go-drawn window in the App, this
// window drives the event loop.
//
// Run: go run ./examples/web-window
package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/bbmumford/mullion"
	"github.com/bbmumford/mullion/dialog"
	"github.com/bbmumford/mullion/examples/internal/appicon"
	"github.com/bbmumford/mullion/shell"
)

const page = `<!doctype html>
<meta charset="utf-8">
<title>Host</title>
<style>
  :root { color-scheme: light dark; font: 15px/1.5 system-ui, sans-serif }
  body { margin: 0; padding: 28px; max-width: 34rem }
  h1 { font-size: 1.4rem; margin: 0 0 1rem }
  button { font: inherit; padding: .4rem .9rem; border-radius: .4rem; cursor: pointer }
  pre { background: color-mix(in srgb, currentColor 8%, transparent);
        padding: .8rem; border-radius: .4rem; white-space: pre-wrap }
</style>
<h1>Host</h1>
<p>These buttons call Go.</p>
<p>
  <button onclick="ask()">Describe the host</button>
  <button onclick="pick()">Pick a file</button>
</p>
<pre id="out">...</pre>
<script>
  const out = document.getElementById('out')
  async function ask()  { out.textContent = await describeHost() }
  async function pick() {
    const path = await chooseFile()
    out.textContent = path ? path : 'cancelled'
  }
</script>`

func main() {
	var session *mullion.Session

	app := mullion.App{
		Name: "Host",
		Icon: appicon.Colour(),
		Window: &mullion.Window{
			Title:  "Host",
			Width:  760,
			Height: 560,
			Page: &mullion.Page{
				HTML: page,
				Bind: map[string]any{
					// A bound function runs on a goroutine, not the main
					// thread, and its result is handed back to the page as the
					// resolution of a promise.
					"describeHost": func() string {
						return fmt.Sprintf("%s/%s, Go %s",
							runtime.GOOS, runtime.GOARCH,
							strings.TrimPrefix(runtime.Version(), "go"))
					},
					// A dialog is modal and blocks this goroutine until the
					// user answers; it runs the panel on the main thread
					// itself, so nothing here has to marshal the call.
					"chooseFile": func() (string, error) {
						return dialog.OpenFile(dialog.Options{
							Title:   "Choose a file",
							Parent:  session.NativeWindow(),
							Filters: []dialog.Filter{{Name: "Text", Extensions: []string{"txt", "md"}}},
						})
					},
				},
			},
		},
		MenuBar: shell.Menu{
			{Label: "Host", Submenu: shell.Menu{
				{Label: "Reload", Shortcut: "cmd+r", Do: func() { session.View().Reload() }},
				shell.Separator(),
				{Label: "Quit", Shortcut: "cmd+q", Do: func() { session.Quit() }},
			}},
			// The Edit menu is wired to the responder chain rather than to Go
			// callbacks, which is the only way Cmd+C and Cmd+V reach a text
			// field inside the page.
			{Label: "Edit", Submenu: shell.Menu{
				{Label: "Undo", Shortcut: "cmd+z", Selector: "undo:"},
				shell.Separator(),
				{Label: "Cut", Shortcut: "cmd+x", Selector: "cut:"},
				{Label: "Copy", Shortcut: "cmd+c", Selector: "copy:"},
				{Label: "Paste", Shortcut: "cmd+v", Selector: "paste:"},
				{Label: "Select All", Shortcut: "cmd+a", Selector: "selectAll:"},
			}},
		},
		OnReady: func(s *mullion.Session) error {
			session = s
			return nil
		},
	}

	if err := mullion.Run(app); err != nil {
		log.Fatal(err)
	}
}
