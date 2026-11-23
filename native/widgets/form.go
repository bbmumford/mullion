package widgets

// Added for mullion; not part of the merged go-shirei snapshot. Kept in a file
// of its own so a later re-sync with upstream can see at a glance what is
// local.

import (
	. "github.com/bbmumford/mullion/native"
)

// FormAttrs tunes a form. The zero value is the ordinary one.
type FormAttrs struct {
	// LabelWidth is the fixed width of the label column, in points. Zero takes
	// 120.
	//
	// It is fixed rather than measured because a column that resizes as its
	// longest label changes makes every field shift sideways while the person
	// is reading. Pick a width that fits the labels and leave it.
	LabelWidth float32

	// Gap separates the rows. Zero takes 8.
	Gap float32

	// ColumnGap separates a label from its field. Zero takes 10.
	ColumnGap float32
}

// Form lays out labelled rows with the labels in one column and the fields in
// another, which is what a settings panel or a properties pane wants and what
// a stack of containers has to be talked into doing.
//
// The rows come from FormRow calls inside fn.
//
//	widgets.Form(widgets.FormAttrs{}, func() {
//	    widgets.FormRow("Name", func() { widgets.TextInput(&name) })
//	    widgets.FormRow("Enabled", func() { widgets.CheckBox(&on, "") })
//	})
func Form(attrs FormAttrs, fn func()) {
	if attrs.LabelWidth == 0 {
		attrs.LabelWidth = 120
	}
	if attrs.Gap == 0 {
		attrs.Gap = 8
	}
	if attrs.ColumnGap == 0 {
		attrs.ColumnGap = 10
	}

	prev := currentForm
	currentForm = attrs
	defer func() { currentForm = prev }()

	Container(Attrs(Expand, Gap(comfort(attrs.Gap))), fn)
}

// FormRow is one labelled row of a Form: the label in the label column, and
// whatever field draws beside it.
//
// Called outside a Form it still draws, using the default measurements, so a
// single row does not need a Form around it.
func FormRow(label string, field func()) {
	attrs := currentForm
	if attrs.LabelWidth == 0 {
		attrs.LabelWidth = 120
	}
	if attrs.ColumnGap == 0 {
		attrs.ColumnGap = 10
	}

	Container(Attrs(Row, Expand, CrossMid, Gap(comfort(attrs.ColumnGap))), func() {
		// CrossMid on the row, not the label: the label sits against the
		// middle of whatever height the field turns out to be, so a one-line
		// label beside a text area lines up with its first line rather than
		// floating in the middle of it.
		Container(Attrs(FixWidth(comfort(attrs.LabelWidth))), func() {
			if label != "" {
				Label(label, FontSize(comfort(12)))
			}
		})
		Container(Attrs(Grow(1)), func() {
			if field != nil {
				field()
			}
		})
	})
}

// currentForm is the attrs of the Form being built, so a row does not have to
// be told what its parent chose. Frames run on one thread, so a plain variable
// is enough and a stack is not needed beyond saving and restoring around the
// nesting.
var currentForm FormAttrs
