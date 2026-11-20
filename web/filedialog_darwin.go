//go:build darwin

package web

import "unsafe"

// fileDialogFrom runs an NSOpenPanel or NSSavePanel and returns the paths
// chosen.
//
// window is unused: these panels are application-modal, which is what the
// upstream WebView methods present too, so a panel is not attached to a window
// and there is no parent to pass. Keeping the parameter lets the dialog package
// have one signature across platforms where Windows and Linux do need it.
func fileDialogFrom(window unsafe.Pointer, kind fileDialogKind, opts FileDialogOptions) ([]string, error) {
	if kind == dialogSaveFile {
		if path := presentSavePanelFor(opts); path != "" {
			return []string{path}, nil
		}
		return nil, nil
	}
	canFiles := kind != dialogOpenDirectory
	canDirs := kind == dialogOpenDirectory
	multiple := kind == dialogOpenFiles
	return presentOpenPanelFor(opts, canFiles, canDirs, multiple), nil
}
