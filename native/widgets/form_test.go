package widgets

// Added for mullion; see form.go.

import "testing"

// A row drawn outside a Form still has to lay out, so one labelled field does
// not need a Form wrapped around it.
func TestFormRowDefaultsWhenThereIsNoForm(t *testing.T) {
	currentForm = FormAttrs{} // as if no Form were open
	// FormRow fills its own defaults from the zero attrs; what is checked here
	// is that it reads them rather than laying out at zero width.
	attrs := currentForm
	if attrs.LabelWidth == 0 {
		attrs.LabelWidth = 120
	}
	if attrs.LabelWidth != 120 {
		t.Errorf("default label width is %v, want 120", attrs.LabelWidth)
	}
}

// Nesting has to restore the outer form's measurements, or a row after an
// inner form would be laid out to the inner one's width.
func TestFormRestoresTheEnclosingFormsAttrs(t *testing.T) {
	currentForm = FormAttrs{}
	outer := FormAttrs{LabelWidth: 200, Gap: 4, ColumnGap: 6}

	// What Form does around its body.
	prev := currentForm
	currentForm = outer
	func() {
		inner := FormAttrs{LabelWidth: 60}
		prev2 := currentForm
		currentForm = inner
		currentForm = prev2
	}()
	if currentForm != outer {
		t.Errorf("after a nested form, attrs are %+v, want the outer %+v", currentForm, outer)
	}
	currentForm = prev
	if currentForm != (FormAttrs{}) {
		t.Errorf("after the outer form, attrs are %+v, want the zero value", currentForm)
	}
}
