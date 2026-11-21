// Package widgets is the control set for the Go-drawn surface: buttons,
// fields, lists, tables, menus and the compositions over them.
//
// It is go-mullion's widget package, merged into mullion, with a handful of
// additions marked as such in their own files. See ../ALTERED.md.
//
// # How a widget works here
//
// A widget is a function called during a frame, not an object that is created
// and kept. It draws what your state says and writes your state back through a
// pointer you give it:
//
//	widgets.CheckBox(&settings.Enabled, "Enabled")
//	widgets.Slider(&zoom, widgets.SliderAttrs{Min: 0.5, Max: 4})
//	widgets.TextInput(&query)
//
// There is nothing to construct, nothing to attach a handler to and nothing to
// free. A widget that is not called this frame is not on screen, so showing
// and hiding is an if statement.
//
// Widgets that need more state than a single value -- which rows of a tree are
// open, what is half-typed into a date field -- take a state struct that you
// own, for the same reason: so it survives a rebuild, and so you can save it
// with the rest of the window.
//
// # Reading a result
//
// A widget that can be activated returns whether it was, on the frame it was:
//
//	if widgets.Button(widgets.NoIcon, "Save") {
//	    save()
//	}
//
// There is no callback to register, so there is no question of which goroutine
// it runs on or whether it outlives the thing it refers to.
//
// # Colour
//
// Every widget reads CurrentColorScheme, which mullion keeps in step with the
// desktop's light and dark setting. A container's own background does not
// follow it automatically, so take the colour from the scheme rather than
// writing a literal:
//
//	native.Container(native.Attrs(
//	    native.BackgroundVec(widgets.CurrentColorScheme.Surfaces.Panel.Background),
//	), body)
//
// An application with its own palette sets it once through mullion.App's
// LightScheme and DarkScheme.
//
// # Colours are HSLA
//
// Every colour here is a Vec4 of hue in 0 to 360, saturation and lightness in
// 0 to 100, and alpha in 0 to 1 -- not red, green, blue and alpha. Vec4{120,
// 60, 45, 1} is a green; the same four numbers read as RGBA would be a
// saturated green too, which is why the mistake survives a quick look. What
// gives it away is Vec4{1, 0, 0, 1}: meant as red, it is a hue of one degree
// at one per cent saturation and one per cent lightness, which draws as
// near-black. A colour that comes out grey has almost always been written as
// RGBA.
package widgets
