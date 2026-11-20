package mullion

import (
	"errors"
	"strings"
	"testing"

	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/shell"
)

// Which surface drives the loop is the decision this package exists to make,
// and getting it wrong shows up as a process that hangs or crashes inside
// AppKit rather than as a Go error. These tests pin it, and pin the refusals
// that keep an application from asking for the arrangement that cannot work.

func TestLoopOwnerFollowsWhatEachBackendCanDo(t *testing.T) {
	draw := native.FrameFn(func() {})
	for _, tc := range []struct {
		name string
		app  App
		want surface
	}{
		{
			name: "a Go-drawn window takes the loop",
			app:  App{Window: &Window{Draw: draw}},
			want: surfaceNative,
		},
		{
			name: "a Go-drawn window takes it even with a tray present",
			app:  App{Window: &Window{Draw: draw}, Tray: &shell.Tray{}},
			want: surfaceNative,
		},
		{
			name: "a WebView window takes it when there is no Go-drawn one",
			app:  App{Window: &Window{Page: &Page{HTML: "<p>hi"}}},
			want: surfaceWeb,
		},
		{
			name: "a WebView window takes it over a tray",
			app:  App{Window: &Window{Page: &Page{HTML: "<p>hi"}}, Tray: &shell.Tray{}},
			want: surfaceWeb,
		},
		{
			name: "the tray takes it when there is no window",
			app:  App{Tray: &shell.Tray{}},
			want: surfaceTray,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.app.loopOwner(); got != tc.want {
				t.Errorf("loopOwner() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateRefusesAnAppThatCannotRun(t *testing.T) {
	draw := native.FrameFn(func() {})
	for _, tc := range []struct {
		name string
		app  App
		says string
	}{
		{
			name: "no surface at all",
			app:  App{Name: "Empty"},
			says: "no user interface",
		},
		{
			name: "a window with nothing in it",
			app:  App{Window: &Window{Title: "Blank"}},
			says: "nothing to show",
		},
		{
			name: "a window drawn two ways at once",
			app:  App{Window: &Window{Draw: draw, Page: &Page{HTML: "<p>hi"}}},
			says: "set one of them",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.app.validate()
			if err == nil {
				t.Fatal("validate accepted an App that cannot run")
			}
			if !strings.Contains(err.Error(), tc.says) {
				t.Errorf("the error does not say what is wrong: %v", err)
			}
		})
	}
}

func TestValidateAcceptsTheWorkingArrangements(t *testing.T) {
	draw := native.FrameFn(func() {})
	for _, tc := range []struct {
		name string
		app  App
	}{
		{"a Go-drawn window", App{Window: &Window{Draw: draw}}},
		{"a WebView window", App{Window: &Window{Page: &Page{HTML: "<p>hi"}}}},
		{"a tray on its own", App{Tray: &shell.Tray{}}},
		{"a window and a tray", App{Window: &Window{Draw: draw}, Tray: &shell.Tray{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.app.validate(); err != nil {
				t.Errorf("validate rejected a working App: %v", err)
			}
		})
	}
}

// Asking for a second Go-drawn window is the one thing this arrangement cannot
// do. It has to be refused in words that say what to do instead, because the
// alternative is a second entry into the platform's loop.
func TestOpenWindowRefusesASecondGoDrawnWindow(t *testing.T) {
	s := &Session{owner: surfaceNative}
	_, err := s.OpenWindow(Window{Draw: func() {}})
	if err == nil {
		t.Fatal("OpenWindow accepted a Go-drawn window")
	}
	if !strings.Contains(err.Error(), "App.Window") {
		t.Errorf("the error does not say where a Go-drawn window has to go: %v", err)
	}
}

func TestOpenWindowNeedsSomethingToShow(t *testing.T) {
	s := &Session{owner: surfaceNative}
	if _, err := s.OpenWindow(Window{Title: "Blank"}); err == nil {
		t.Fatal("OpenWindow accepted a Window with no Page")
	}
}

func TestDefaultWindowSize(t *testing.T) {
	w := Window{}
	if w.width() != 800 || w.height() != 600 {
		t.Errorf("default size = %dx%d, want 800x600", w.width(), w.height())
	}
	sized := Window{Width: 1024, Height: 768}
	if sized.width() != 1024 || sized.height() != 768 {
		t.Errorf("size = %dx%d, want 1024x768", sized.width(), sized.height())
	}
}

// Run is called from a goroutine here, never the pinned main one, so it must
// refuse rather than reach a window API from the wrong thread.
func TestRunRefusesOffTheMainThread(t *testing.T) {
	err := Run(App{Tray: &shell.Tray{}})
	if err == nil {
		t.Fatal("Run accepted a call from a test goroutine")
	}
	if !strings.Contains(err.Error(), "main") {
		t.Errorf("the error does not name the thread it needs: %v", err)
	}
}

// A tray with no icon of its own should show the application's, so an App
// states its artwork once.
func TestNotifyWithoutATrayIsReportedNotIgnored(t *testing.T) {
	s := &Session{app: App{Name: "Demo"}}
	err := s.Notify("Title", "Body")
	if err == nil {
		t.Fatal("Notify without a tray reported success")
	}
	if errors.Is(err, errors.ErrUnsupported) {
		t.Error("a missing tray was reported as an unsupported platform, which it is not")
	}
}
