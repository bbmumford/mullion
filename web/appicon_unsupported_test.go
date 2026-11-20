package web

import (
	"errors"
	"testing"

	"github.com/bbmumford/mullion/internal/menubar"
)

// Both sentinels must wrap the standard errors.ErrUnsupported.
//
// The platforms where they are returned are not edge cases: ErrIconUnsupported
// is what Windows and Linux always return, and menubar.ErrUnsupported is what
// Linux always returns. A caller cannot tell those apart from a real failure
// without naming a mullion-specific sentinel, so an application behaving
// correctly on a supported platform reports an error it did not cause.
func TestUnsupportedSentinelsWrapStdlib(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"web.ErrIconUnsupported", ErrIconUnsupported},
		{"menubar.ErrUnsupported", menubar.ErrUnsupported},
	} {
		if !errors.Is(tc.err, errors.ErrUnsupported) {
			t.Errorf("%s does not wrap errors.ErrUnsupported (got %v)", tc.name, tc.err)
		}
	}
}
