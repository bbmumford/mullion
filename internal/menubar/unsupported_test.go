package menubar_test

import (
	"errors"
	"testing"

	"github.com/bbmumford/mullion/internal/menubar"
)

// The sentinel must wrap the standard errors.ErrUnsupported, so a caller can
// tell "no menu backend on this platform" from a real failure with the one
// check the standard library defines for it, without naming this package.
func TestErrUnsupportedWrapsStdlib(t *testing.T) {
	if !errors.Is(menubar.ErrUnsupported, errors.ErrUnsupported) {
		t.Errorf("menubar.ErrUnsupported does not wrap errors.ErrUnsupported (got %v)", menubar.ErrUnsupported)
	}
}
