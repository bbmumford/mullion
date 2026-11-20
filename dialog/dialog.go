// Package dialog opens the platform's own file dialogs, for either surface.
//
// The dialogs are a system facility rather than a property of a window, so an
// application asks for one here and names the window it should belong to. A
// Go-drawn window and a WebView window both give that handle, which is what
// makes this the one spelling for both; the platform code behind it lives
// beside the WebView, because WKWebView's file-input handler shares it.
//
// Every call is modal: it blocks the calling goroutine until the user answers,
// and is safe to call from a frame function or a menu handler on the main
// thread as well, where it runs the dialog in place rather than waiting on a
// thread it would be blocking.
//
// Platform support is macOS, Windows and Linux. iOS, Android and a wasm build
// have no modal panel that returns filesystem paths, and report an error
// wrapping errors.ErrUnsupported.
package dialog

import (
	"unsafe"

	"github.com/bbmumford/mullion/web"
)

// A Filter is one entry in the dialog's file-type list.
type Filter struct {
	// Name is what the user sees, such as "Images".
	Name string

	// Extensions are the extensions it matches, without a dot: {"png", "jpg"}.
	Extensions []string
}

// Options configures a dialog. The zero Options is valid: the platform's
// default dialog, starting wherever it last was.
type Options struct {
	// Title replaces the dialog's default title.
	Title string

	// Directory is the folder to open in. Empty leaves the platform's choice,
	// which is usually where the user last was.
	Directory string

	// Filename is the name offered in a save dialog, and ignored by the others.
	Filename string

	// Filters restricts what the user can pick. Empty shows all files.
	Filters []Filter

	// Parent is the window the dialog belongs to -- a Go-drawn window's native
	// handle, or a WebView's Window(). Nil opens a dialog with no parent, which
	// works but is not positioned over a window or kept in front of one. macOS
	// ignores it, since its panels are application-modal.
	Parent unsafe.Pointer
}

// OpenFile asks for one existing file and returns its path. An empty path with
// a nil error is a cancelled dialog, which is a normal answer and not a
// failure.
func OpenFile(opts Options) (string, error) {
	return web.OpenFileFrom(opts.Parent, opts.webOptions())
}

// OpenFiles asks for any number of existing files. A nil slice with a nil error
// is a cancelled dialog.
func OpenFiles(opts Options) ([]string, error) {
	return web.OpenFilesFrom(opts.Parent, opts.webOptions())
}

// OpenDirectory asks for one folder. Filters are ignored: a folder chooser has
// no file types to filter.
func OpenDirectory(opts Options) (string, error) {
	return web.OpenDirectoryFrom(opts.Parent, opts.webOptions())
}

// SaveFile asks where to write a file and returns the path, which may not exist
// yet. The platform asks about overwriting an existing file; a path coming back
// means the user agreed.
func SaveFile(opts Options) (string, error) {
	return web.SaveFileFrom(opts.Parent, opts.webOptions())
}

func (o Options) webOptions() web.FileDialogOptions {
	filters := make([]web.FileFilter, 0, len(o.Filters))
	for _, f := range o.Filters {
		filters = append(filters, web.FileFilter{Name: f.Name, Extensions: f.Extensions})
	}
	return web.FileDialogOptions{
		Title:     o.Title,
		Directory: o.Directory,
		Filename:  o.Filename,
		Filters:   filters,
	}
}
