package mullion

import (
	"fmt"

	"github.com/cli/browser"
)

// OpenURL hands url to the system browser, or to whatever handles its scheme.
//
// It works from anywhere: a tray menu handler, a goroutine, a frame function,
// or a process with no window at all. The Go-drawn surface has an OpenURL of
// its own in native/widgets, but that one asks the backend to open the URL
// after the current frame, so it reaches nothing outside a frame -- which is
// where a tray-only application does all of its work.
//
// Windows is the reason this is a package rather than an exec: the handler is
// invoked through ShellExecute, not "cmd /c start", which would split a URL at
// its first ampersand and drop the query string.
func OpenURL(url string) error {
	if url == "" {
		return fmt.Errorf("mullion: no URL to open")
	}
	if err := browser.OpenURL(url); err != nil {
		return fmt.Errorf("mullion: opening %s: %w", url, err)
	}
	return nil
}
