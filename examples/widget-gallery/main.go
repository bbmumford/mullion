// Every widget the Go-drawn surface offers, in one window, with the state live
// so each control does something.
//
// The gallery existed only as a snapshot fixture inside the widgets package's
// tests, where it rendered to a PNG and nothing could be clicked. It is the
// densest piece of UI in the tree -- three type styles, mixed scripts, four
// kinds of button state, a slider, an editable field and a sortable table --
// which makes it the thing to run when the question is whether the renderer and
// the widget set came through the merge intact.
//
// Nothing here is HTML. Every pixel is drawn by the Go-drawn surface, and the
// colours come from the scheme, so the window follows the desktop between light
// and dark without the application doing anything.
//
// Run: go run ./examples/widget-gallery
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bbmumford/mullion"
	"github.com/bbmumford/mullion/examples/internal/appicon"
	. "github.com/bbmumford/mullion/native"
	. "github.com/bbmumford/mullion/native/widgets"
)

// row is one line of the table.
type row struct {
	Name string
	Size int64
	Hits int
}

// state is the whole gallery. A frame reads it and the widgets write to it
// through the pointers they are given, so there is nothing to wire up and
// nothing to keep in step.
type state struct {
	checkOn  bool
	checkOff bool
	option   string
	slider   float32
	input    string
	unit     string
	clicks   int
	tree     TreeState
	sections [2]bool
	picked   string
	rows     []*row
}

func main() {
	s := &state{
		checkOn: true,
		option:  "b",
		slider:  0.6,
		input:   "hello 世界",
		rows:    makeRows(40),
	}
	// Opened so the first frame shows the tree expanded rather than a
	// single collapsed row.
	s.sections[0] = true
	s.tree.SetOpen("src", true)

	err := mullion.Run(mullion.App{
		Name: "Widget Gallery",
		Icon: appicon.Colour(),
		Window: &mullion.Window{
			Title:  "Widget Gallery",
			Width:  900,
			Height: 640,
			Draw:   func() { s.draw() },
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (s *state) draw() {
	// The surfaces come from the colour scheme rather than being written as
	// fixed colours. The widgets below follow the scheme on their own; a
	// container does not, so hard-coding its background is how a window ends
	// up with dark controls on a white panel when the desktop turns dark.
	scheme := CurrentColorScheme
	canvas := scheme.Surfaces.Canvas
	panel := scheme.Surfaces.Panel

	Container(Attrs(Row, Viewport, BackgroundVec(canvas.Background)), func() {
		// The controls, down the left in a fixed column.
		Container(Attrs(FixWidth(300), Expand, Clip, Pad(14), Gap(10), BackgroundVec(panel.Background)), func() {
			Label("Widget Gallery", FontWeight(WeightBold), FontSize(16), TextColorVec(panel.Text))
			Label("mixed 日本語 and English 123", FontSize(12), TextColorVec(panel.Text))
			Label("subtle italic caption", FontStyle(StyleItalic), FontSize(11), TextColorVec(panel.Text))

			NextButtonType(ButtonPrimary)
			if Button(NoIcon, "Primary Button") {
				s.clicks++
			}
			Container(Attrs(Row, CrossMid, Gap(8)), func() {
				if CtrlButton(NoIcon, "Ctrl Button", true) {
					s.clicks++
				}
				// Passing false leaves the button disabled, which is what the
				// greyed one beside it is for.
				CtrlButton(NoIcon, "Disabled", false)
			})
			Label(fmt.Sprintf("%d button presses", s.clicks), FontSize(11), TextColorVec(panel.Text))

			CheckBox(&s.checkOn, "Enabled option")
			CheckBox(&s.checkOff, "Disabled option")
			OptionGroup(&s.option, func() {
				ModAttrs(Gap(10))
				OptionButton("Choice A", "a")
				OptionButton("Choice B", "b")
			})

			Select(&s.unit, "Choose a unit", "bytes", "lines", "pages")
			Label(unitLabel(s.unit), FontSize(11), TextColorVec(panel.Text))

			Slider(&s.slider, SliderAttrs{Min: 0, Max: 1, Width: 200})
			Label(fmt.Sprintf("slider at %.2f", s.slider), FontSize(11), TextColorVec(panel.Text))

			TextInput(&s.input)
			Label(fmt.Sprintf("%d runes typed", len([]rune(s.input))), FontSize(11), TextColorVec(panel.Text))
		})

		// A middle column of the composite widgets.
		Container(Attrs(FixWidth(260), Expand, Clip, Pad(10), Gap(10)), func() {
			Card(CardAttrs{Title: "Card", Subtitle: "a titled panel"}, func() {
				Form(FormAttrs{LabelWidth: 70}, func() {
					FormRow("Unit", func() { Select(&s.unit, "choose", "bytes", "lines", "pages") })
					FormRow("Enabled", func() { CheckBox(&s.checkOn, "") })
				})
			})
			AccordionSection(&s.sections[0], "Tree", func() {
				if v, ok := Tree(&s.tree, TreeAttrs{}, treeChildren); ok {
					s.picked = v
				}
				Label("picked: "+pickedOr(s.picked), FontSize(11))
			})
			AccordionSection(&s.sections[1], "Collapsed by default", func() {
				Label("only drawn while open", FontSize(11))
			})
		})

		// The table fills what is left, and sorts on a header click.
		Container(Attrs(Grow(1), Expand, Clip), func() {
			Table(nil, 28, columns(), s.visibleRows(), func(r *row) any { return r }, 2)
		})
	})
}

// visibleRows filters by whatever is in the text field, so typing in it changes
// the table beside it and the two widgets are visibly one program.
func (s *state) visibleRows() []*row {
	needle := strings.ToLower(strings.TrimSpace(s.input))
	// The default text is a greeting rather than a filter; an empty field and
	// one that matches nothing should both show everything.
	if needle == "" || needle == "hello 世界" {
		return s.rows
	}
	var out []*row
	for _, r := range s.rows {
		if strings.Contains(strings.ToLower(r.Name), needle) {
			out = append(out, r)
		}
	}
	return out
}

func columns() []TableColumn[*row] {
	return []TableColumn[*row]{
		{
			Label: "Name",
			Cell:  func(r *row) { Label(r.Name) },
			Less:  func(a, b *row) bool { return a.Name < b.Name },
		},
		{
			Label: "Size", Width: 90, DefaultDesc: true,
			Cell: func(r *row) { Label(fmt.Sprintf("%d", r.Size)) },
			Less: func(a, b *row) bool { return a.Size < b.Size },
		},
		{
			Label: "Hits", Width: 70, DefaultDesc: true,
			Cell: func(r *row) { Label(fmt.Sprintf("%d", r.Hits)) },
			Less: func(a, b *row) bool { return a.Hits < b.Hits },
		},
	}
}

func makeRows(n int) []*row {
	words := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel"}
	pkgs := []string{"runtime", "widgets", "shirei", "main"}
	out := make([]*row, 0, n)
	for i := range n {
		out = append(out, &row{
			Name: fmt.Sprintf("%s.%s%02d", pkgs[i%len(pkgs)], words[i%len(words)], i),
			Size: int64(i*i*137 + 42),
			Hits: (i * 31) % 97,
		})
	}
	return out
}

// unitLabel says what the Select currently holds, so the difference between
// "nothing chosen" and a chosen value is visible rather than implied.
func unitLabel(unit string) string {
	if unit == "" {
		return "no unit chosen"
	}
	return "unit: " + unit
}

// treeChildren is a small hierarchy, shaped the way a filesystem would be: the
// children of a key, asked for only where the tree needs them.
func treeChildren(key string) []TreeNode[string] {
	switch key {
	case "":
		return []TreeNode[string]{
			{Key: "src", Label: "src", HasChildren: true, Value: "src"},
			{Key: "docs", Label: "docs", HasChildren: true, Value: "docs"},
			{Key: "go.mod", Label: "go.mod", Value: "go.mod"},
		}
	case "src":
		return []TreeNode[string]{
			{Key: "src/main.go", Label: "main.go", Value: "src/main.go"},
			{Key: "src/ui", Label: "ui", HasChildren: true, Value: "src/ui"},
		}
	case "src/ui":
		return []TreeNode[string]{{Key: "src/ui/view.go", Label: "view.go", Value: "src/ui/view.go"}}
	case "docs":
		return []TreeNode[string]{{Key: "docs/readme.md", Label: "readme.md", Value: "docs/readme.md"}}
	}
	return nil
}

func pickedOr(v string) string {
	if v == "" {
		return "nothing"
	}
	return v
}
