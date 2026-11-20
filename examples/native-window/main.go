// A window whose interface is drawn in Go.
//
// The Go-drawn surface takes the application's event loop, which is why this
// window is the one App.Window names rather than something opened later. Draw
// runs once per frame and describes the window from the state above it; there
// are no widget objects to create or keep in step.
//
// Run: go run ./examples/native-window
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bbmumford/mullion"
	"github.com/bbmumford/mullion/examples/internal/appicon"
	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/widgets"
)

// state is the whole application. A frame reads it and the handlers write it,
// both on the main thread, so it needs no lock.
type state struct {
	name    string
	note    string
	entries []string
}

func main() {
	s := &state{name: "", note: ""}

	err := mullion.Run(mullion.App{
		Name: "Notes",
		Icon: appicon.Colour(),
		Window: &mullion.Window{
			Title:  "Notes",
			Width:  720,
			Height: 520,
			Draw:   func() { s.draw() },
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (s *state) draw() {
	native.Container(native.Attrs(native.Expand, native.Pad(20), native.Gap(12)), func() {
		native.Label("Notes", native.FontSize(24), native.FontWeight(native.WeightBold))

		native.Label("Name")
		widgets.TextInput(&s.name)

		native.Label("Note")
		widgets.TextInput(&s.note)

		native.Container(native.Attrs(native.Row, native.CrossMid, native.Gap(10)), func() {
			if widgets.Button(widgets.SymIPlus, "Add") {
				s.add()
			}
			native.Label(fmt.Sprintf("%d saved", len(s.entries)), native.FontSize(12))
		})

		for _, e := range s.entries {
			native.Label(e)
		}
	})
}

// add records the note and clears the fields. It runs inside the frame, on the
// main thread, so it writes the state directly; background work would go
// through Session.Do instead.
func (s *state) add() {
	name := strings.TrimSpace(s.name)
	note := strings.TrimSpace(s.note)
	if name == "" && note == "" {
		return
	}
	s.entries = append(s.entries, fmt.Sprintf("%s — %s", name, note))
	s.name, s.note = "", ""
}
