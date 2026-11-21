package mullion

import (
	"errors"
	"strings"
	"testing"
)

// The window controls act on the Go-drawn window. Asked of a WebView or tray
// application they have nothing to act on, and the thing that must not happen
// is silence: a caller who moves a window and sees nothing move should be told
// why at the call rather than left to guess.
func TestWindowControlsRefuseWhenAnotherSurfaceDrives(t *testing.T) {
	for _, owner := range []surface{surfaceWeb, surfaceTray} {
		s := &Session{owner: owner}
		for _, tc := range []struct {
			name string
			call func() error
		}{
			{"Resize", func() error { return s.Resize(100, 100) }},
			{"SetMinSize", func() error { return s.SetMinSize(10, 10) }},
			{"Center", s.Center},
			{"MoveTo", func() error { return s.MoveTo(0, 0) }},
			{"Activate", s.Activate},
		} {
			err := tc.call()
			if err == nil {
				t.Errorf("%s under %s returned no error", tc.name, owner)
				continue
			}
			if !errors.Is(err, errors.ErrUnsupported) {
				t.Errorf("%s under %s: error does not wrap ErrUnsupported: %v", tc.name, owner, err)
			}
			// The message has to name the call and what is driving, or it
			// sends the reader to the wrong place.
			if !strings.Contains(err.Error(), tc.name) || !strings.Contains(err.Error(), string(owner)) {
				t.Errorf("%s under %s: message names neither the call nor the owner: %v", tc.name, owner, err)
			}
		}
	}
}

// The two readers answer rather than error, because a caller asking where a
// window is wants a position or an honest "not known", not an error to handle.
func TestWindowReadersAnswerFalseWhenAnotherSurfaceDrives(t *testing.T) {
	s := &Session{owner: surfaceTray}
	if _, _, ok := s.Position(); ok {
		t.Error("Position claimed to know where a window is with no Go-drawn window")
	}
	if s.PointerInside() {
		t.Error("PointerInside said yes with no Go-drawn window")
	}
}

func TestOpenPopupRefusesWithoutTheGoDrawnSurface(t *testing.T) {
	s := &Session{owner: surfaceWeb}
	_, err := s.OpenPopup(PopupOptions{}, func() {})
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("error does not wrap ErrUnsupported: %v", err)
	}
}

// A popup with nothing to draw is a mistake worth naming, not an empty window.
func TestOpenPopupRefusesAMissingDrawFunction(t *testing.T) {
	s := &Session{owner: surfaceNative}
	if _, err := s.OpenPopup(PopupOptions{}, nil); err == nil {
		t.Error("OpenPopup accepted a nil draw function")
	}
}

func TestStartAudioRefusesAMissingFill(t *testing.T) {
	s := &Session{owner: surfaceNative}
	if err := s.StartAudio(48000, nil); err == nil {
		t.Error("StartAudio accepted a nil fill function")
	}
}
