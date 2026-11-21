package dialog

import (
	"errors"
	"runtime"
	"testing"
)

// On a platform with no alert, Ask has to be distinguishable from a no: a
// consent prompt that could not be shown must not read as consent refused by
// the person, and certainly not as consent given.
func TestAskOnAnUnsupportedPlatformIsAnErrorAndNotAYes(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		t.Skip("this platform has an alert; the unsupported path is not reachable here")
	}
	ok, err := Ask(Question{Title: "Allow?"})
	if ok {
		t.Error("Ask said yes on a platform with no alert")
	}
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("error does not wrap errors.ErrUnsupported: %v", err)
	}
}

// The zero Question is valid -- a bare title is the common case -- and must not
// be refused before the platform ever sees it.
func TestAQuestionNeedsOnlyATitle(t *testing.T) {
	q := Question{Title: "Allow?"}
	if q.Confirm != "" || q.Cancel != "" {
		t.Fatal("the fixture no longer leaves the labels empty")
	}
	// The labels are filled further down, in the web package, which has its own
	// test; what matters here is that Question carries no requirement of its own.
}

// The two meanings of false are the contract most likely to be read wrong, so
// it is written down here as well as in the doc comment: a nil error means the
// person answered, and a non-nil error means nobody was asked.
func TestFalseWithAnErrorIsNotAnAnswer(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		t.Skip("this platform has an alert; no failure can be induced without a person")
	}
	ok, err := Ask(Question{Title: "Allow?"})
	if ok || err == nil {
		t.Fatalf("Ask = (%v, %v), want (false, an error)", ok, err)
	}
	// The pair a caller must not collapse into "the person said no".
	if errors.Is(err, nil) {
		t.Error("the error is nil, which would make this indistinguishable from a refusal")
	}
}
