package web

import "unsafe"

// The file dialogs reach every surface in this module, not only a WebView.
//
// The platform code for them lives in this package because WKWebView's own
// <input type=file> handler shares the panel configuration, and splitting the
// two would mean two copies of the AppKit, COM and GTK plumbing. These
// window-parameterised entry points are what the dialog package calls, so an
// application built on the Go-drawn surface opens the same native dialogs by
// handing over its window rather than by creating a WebView it does not want.
//
// window is the parent the dialog belongs to. Pass nil for none: a dialog with
// no parent still opens, it is simply not positioned over a window or kept in
// front of one. macOS ignores the parent entirely -- its panels are
// application-modal.

type fileDialogKind int

const (
	dialogOpenFile fileDialogKind = iota
	dialogOpenFiles
	dialogOpenDirectory
	dialogSaveFile
)

// OpenFileFrom shows a modal open-file dialog parented to window and returns
// the chosen path, empty when the user cancelled.
func OpenFileFrom(window unsafe.Pointer, opts FileDialogOptions) (string, error) {
	paths, err := fileDialogFrom(window, dialogOpenFile, opts)
	return firstPathOrEmpty(paths), err
}

// OpenFilesFrom shows a modal open-file dialog that allows more than one
// selection and returns the chosen paths, nil when the user cancelled.
func OpenFilesFrom(window unsafe.Pointer, opts FileDialogOptions) ([]string, error) {
	return fileDialogFrom(window, dialogOpenFiles, opts)
}

// OpenDirectoryFrom shows a modal directory chooser parented to window and
// returns the chosen path, empty when the user cancelled.
func OpenDirectoryFrom(window unsafe.Pointer, opts FileDialogOptions) (string, error) {
	paths, err := fileDialogFrom(window, dialogOpenDirectory, opts)
	return firstPathOrEmpty(paths), err
}

// SaveFileFrom shows a modal save-file dialog parented to window and returns
// the chosen path, empty when the user cancelled.
func SaveFileFrom(window unsafe.Pointer, opts FileDialogOptions) (string, error) {
	paths, err := fileDialogFrom(window, dialogSaveFile, opts)
	return firstPathOrEmpty(paths), err
}

func firstPathOrEmpty(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}
