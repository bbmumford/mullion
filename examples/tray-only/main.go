// An application that lives in the notification area and has no window.
//
// With no window in the App, the tray drives the event loop. On macOS that also
// means no dock icon: a tray-only process asks for the accessory activation
// policy, which the tray backend does when it runs the loop.
//
// Run: go run ./examples/tray-only
package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/bbmumford/mullion"
	"github.com/bbmumford/mullion/examples/internal/appicon"
	"github.com/bbmumford/mullion/shell"
)

func main() {
	var session *mullion.Session
	started := time.Now()
	quiet := false

	tray := &shell.Tray{
		Tooltip: "Uptime",
		Menu: shell.Menu{
			{Label: "How long have I been up?", Do: func() {
				if quiet {
					return
				}
				// A menu handler runs on the main thread. Notifying is cheap
				// and stays here; anything slow would go to a goroutine so the
				// menu does not hang open.
				err := session.Notify("Uptime",
					fmt.Sprintf("%s on %s", time.Since(started).Round(time.Second), runtime.GOOS))
				if err != nil {
					log.Printf("notify: %v", err)
				}
			}},
			{Label: "Stay quiet", Checkbox: true, Do: func() { quiet = !quiet }},
			shell.Separator(),
			{Label: "Quit", Do: func() { session.Quit() }},
		},
	}

	// A template icon is a silhouette the macOS menu bar tints to match the
	// system theme. Elsewhere the flag is ignored, so the examples hand the
	// colour icon to the App and the silhouette to the tray only on macOS.
	if runtime.GOOS == "darwin" {
		tray.Icon = appicon.Template()
	}

	err := mullion.Run(mullion.App{
		Name: "Uptime",
		Icon: appicon.Colour(),
		Tray: tray,
		OnReady: func(s *mullion.Session) error {
			session = s
			log.Printf("in the notification area; the loop belongs to the tray")

			// Background work reaches the tray through Do, because every
			// platform here accepts tray calls on the main thread only.
			go func() {
				for range time.Tick(30 * time.Second) {
					up := time.Since(started).Round(time.Second).String()
					s.Do(func() {
						if err := s.Tray().SetTooltip("Up " + up); err != nil {
							log.Printf("tooltip: %v", err)
						}
					})
				}
			}()
			return nil
		},
		OnQuit: func() { log.Printf("up for %s", time.Since(started).Round(time.Second)) },
	})
	if err != nil {
		log.Fatal(err)
	}
}
