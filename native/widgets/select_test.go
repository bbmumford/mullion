package widgets

// Added for mullion; see select.go.

import "testing"

// Select's job is the bookkeeping, and the part worth pinning is what it shows
// before anything is chosen: an empty value is "no choice yet", not a chosen
// empty string, and the two must not look the same.
func TestSelectShowsThePlaceholderUntilSomethingIsChosen(t *testing.T) {
	var chosen string
	if label := selectLabel(&chosen, "Choose a unit", func(s string) string { return s }); label != "Choose a unit" {
		t.Errorf("empty value showed %q, want the placeholder", label)
	}
	chosen = "lines"
	if label := selectLabel(&chosen, "Choose a unit", func(s string) string { return s }); label != "lines" {
		t.Errorf("chosen value showed %q, want lines", label)
	}
}

// A value that is not its own label is the reason SelectExt exists, so the
// label function has to be what decides the text.
func TestSelectExtShowsWhatTheLabelFunctionSays(t *testing.T) {
	type unit int
	const bytesUnit unit = 2
	chosen := bytesUnit
	label := selectLabel(&chosen, "none", func(u unit) string {
		if u == bytesUnit {
			return "bytes"
		}
		return "other"
	})
	if label != "bytes" {
		t.Errorf("showed %q, want bytes", label)
	}
}

func TestSelectExtIgnoresNilArguments(t *testing.T) {
	// Neither should panic; both report no change.
	if SelectExt[string](nil, "x", []string{"a"}, func(s string) string { return s }) {
		t.Error("a nil value reported a change")
	}
	var v string
	if SelectExt(&v, "x", []string{"a"}, nil) {
		t.Error("a nil label function reported a change")
	}
}
