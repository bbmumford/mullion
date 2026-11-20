// Everything at once: a Go-drawn main window, a menu bar, a tray icon, a
// WebView window opened from the tray, and a native file dialog.
//
// This is the arrangement the merge exists for. The Go-drawn window drives the
// one event loop; the tray and the menu bar attach to it once it is running,
// and the WebView window opened from the tray menu attaches to it too. Three
// libraries that each used to take the process now share it.
//
// Run: go run ./examples/full-shell
package main

import (
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bbmumford/mullion"
	"github.com/bbmumford/mullion/dialog"
	"github.com/bbmumford/mullion/examples/internal/appicon"
	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/widgets"
	"github.com/bbmumford/mullion/shell"
)

// state is read by the frame function and written by the handlers, all on the
// main thread, so it needs no lock. Anything a goroutine wants to change goes
// through Session.Do.
type state struct {
	session *mullion.Session
	opened  string
	preview string
	log     []string
}

func main() {
	s := &state{}

	tray := &shell.Tray{
		Tooltip: "Workbench",
		Menu: shell.Menu{
			{Label: "Open a file...", Do: s.openFile},
			{Label: "Show the page", Do: s.openPage},
			shell.Separator(),
			{Label: "Quit", Do: func() { s.session.Quit() }},
		},
	}
	if isMac() {
		tray.Icon = appicon.Template()
	}

	err := mullion.Run(mullion.App{
		Name: "Workbench",
		Icon: appicon.Colour(),
		Window: &mullion.Window{
			Title:  "Workbench",
			Width:  820,
			Height: 580,
			Draw:   func() { s.draw() },
		},
		Tray: tray,
		MenuBar: shell.Menu{
			{Label: "Workbench", Submenu: shell.Menu{
				{Label: "Open a file...", Shortcut: "cmd+o", Do: s.openFile},
				{Label: "Show the page", Shortcut: "cmd+p", Do: s.openPage},
				shell.Separator(),
				{Label: "Quit", Shortcut: "cmd+q", Do: func() { s.session.Quit() }},
			}},
		},
		OnReady: func(sess *mullion.Session) error {
			s.session = sess
			s.note("ready; the loop belongs to the Go-drawn window")

			// A goroutine cannot touch the tray or the window directly. Do
			// queues the work onto the main thread and the frame hook runs it.
			go func() {
				for t := range time.Tick(10 * time.Second) {
					stamp := t.Format("15:04:05")
					sess.Do(func() {
						if err := sess.Tray().SetTooltip("Workbench " + stamp); err != nil {
							s.note("tooltip: " + err.Error())
						}
					})
				}
			}()
			return nil
		},
		OnQuit: func() { log.Printf("closed with %d entries in the log", len(s.log)) },
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (s *state) draw() {
	native.Container(native.Attrs(native.Expand, native.Pad(20), native.Gap(12)), func() {
		native.Label("Workbench", native.FontSize(24), native.FontWeight(native.WeightBold))
		native.Label("Drawn in Go. The tray and the menu bar share this window's event loop.",
			native.FontSize(13))

		native.Container(native.Attrs(native.Row, native.CrossMid, native.Gap(10)), func() {
			if widgets.Button(widgets.SymIndent, "Open a file") {
				s.openFile()
			}
			if widgets.Button(widgets.SymInternal, "Show the page") {
				s.openPage()
			}
		})

		if s.opened != "" {
			native.Label("Opened: "+s.opened, native.FontSize(13))
		}
		if s.preview != "" {
			native.Label(s.preview, native.FontSize(12))
		}

		native.Label("Log", native.FontSize(16), native.FontWeight(native.WeightBold))
		for _, line := range s.log {
			native.Label(line, native.FontSize(12))
		}
	})
}

// openFile shows the platform's file dialog. It runs on the main thread, from a
// button or a menu, and the dialog runs the panel there in place rather than
// queueing it behind a thread it would be blocking.
func (s *state) openFile() {
	path, err := dialog.OpenFile(dialog.Options{
		Title:  "Open a file",
		Parent: s.session.NativeWindow(),
		Filters: []dialog.Filter{
			{Name: "Text", Extensions: []string{"txt", "md", "go"}},
			{Name: "Any file", Extensions: []string{"*"}},
		},
	})
	switch {
	case err != nil:
		s.note("dialog: " + err.Error())
	case path == "":
		s.note("dialog cancelled")
	default:
		s.opened = filepath.Base(path)
		s.preview = firstLines(path, 3)
		s.note("opened " + s.opened)
	}
}

// openPage opens a WebView window on the loop the Go-drawn window is driving.
// On Linux that window is serviced by this process's frame hook, which is what
// lets the two surfaces live together there.
func (s *state) openPage() {
	page := fmt.Sprintf(`<!doctype html>
<meta charset="utf-8"><title>Page</title>
<style>:root{color-scheme:light dark;font:15px/1.6 system-ui,sans-serif}
body{margin:0;padding:28px}code{font-family:ui-monospace,monospace}</style>
<h1>A WebView window</h1>
<p>Rendered by the system WebView, in the same process and on the same event
loop as the Go-drawn window behind it.</p>
<p>Last file opened: <code>%s</code></p>`, html.EscapeString(orDash(s.opened)))

	view, err := s.session.OpenWindow(mullion.Window{
		Title:  "Page",
		Width:  640,
		Height: 420,
		Page:   &mullion.Page{HTML: page},
	})
	if err != nil {
		s.note("page: " + err.Error())
		return
	}
	s.note("opened a WebView window")
	_ = view // the window lives on the shared loop; nothing here has to pump it
}

func (s *state) note(line string) {
	s.log = append(s.log, time.Now().Format("15:04:05")+"  "+line)
	if len(s.log) > 8 {
		s.log = s.log[len(s.log)-8:]
	}
	// The Go-drawn surface redraws on input; a change made from anywhere else
	// has to ask for the frame that will show it.
	native.RequestNextFrame()
}

func firstLines(path string, n int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "could not read it: " + err.Error()
	}
	lines := strings.SplitN(string(data), "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func isMac() bool { return runtime.GOOS == "darwin" }
