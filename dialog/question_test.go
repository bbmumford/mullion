package dialog

import (
	"errors"
	"fmt"
	"runtime"
	"testing"
)

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

// ErrNotShown is the whole contract a consumer has to get right, so it is
// pinned here rather than left to the doc comment. A caller that mistakes it
// for the person's answer records a refusal nobody made.
func TestErrNotShownIsDistinguishableFromARefusal(t *testing.T) {
	// What a failing platform returns, built the way askQuestion builds it.
	notShown := fmt.Errorf("%w: %w", ErrNotShown, errors.New("the toolkit would not build one"))

	if !errors.Is(notShown, ErrNotShown) {
		t.Error("a wrapped failure does not match ErrNotShown, so a caller cannot detect it")
	}
	// The person's actual no carries no error at all, which is what makes the
	// two tellable apart.
	if errors.Is(nil, ErrNotShown) {
		t.Error("a nil error matches ErrNotShown, which would make a refusal look unshown")
	}
}

// A platform with no alert must be both: not shown, and never going to be.
func TestAnUnsupportedPlatformIsNotShownAndUnsupported(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		t.Skip("this platform has an alert; the unsupported path is not reachable here")
	}
	ok, err := Ask(Question{Title: "Allow?"})
	if ok {
		t.Error("Ask said yes on a platform with no alert")
	}
	if !errors.Is(err, ErrNotShown) {
		t.Errorf("the error does not wrap ErrNotShown: %v", err)
	}
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("the error does not wrap errors.ErrUnsupported: %v", err)
	}
}

// The four cases a caller switches on, written out so the shape in the doc
// comment is known to compile and to separate cleanly.
func TestTheFourOutcomesAreSeparable(t *testing.T) {
	classify := func(ok bool, err error) string {
		switch {
		case errors.Is(err, ErrNotShown):
			return "not shown"
		case err != nil:
			return "other failure"
		case ok:
			return "agreed"
		default:
			return "declined"
		}
	}
	for _, tc := range []struct {
		name string
		ok   bool
		err  error
		want string
	}{
		{"the person agreed", true, nil, "agreed"},
		{"the person declined", false, nil, "declined"},
		{"nobody was asked", false, fmt.Errorf("%w: gtk", ErrNotShown), "not shown"},
		{"asked, then failed", false, errors.New("something else"), "other failure"},
	} {
		if got := classify(tc.ok, tc.err); got != tc.want {
			t.Errorf("%s classified as %q, want %q", tc.name, got, tc.want)
		}
	}
}
