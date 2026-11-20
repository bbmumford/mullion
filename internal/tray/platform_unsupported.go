//go:build !darwin && !linux && !windows

package tray

import (
	"errors"
	"fmt"
)

// There is no notification area on iOS, on Android or in a wasm build. A
// platform tray that says so lets the module compile for the targets the
// Go-drawn surface reaches, so an application keeps one import list and finds
// out at runtime, from an error it can test with errors.Is, that it has no tray
// to put an icon in.

var errNoTray = fmt.Errorf("tray: no notification area on this platform: %w", errors.ErrUnsupported)

// NewPlatformTray returns a tray that reports every operation as unsupported.
func NewPlatformTray(callbacks *Callbacks) PlatformTray { return unsupportedTray{} }

type unsupportedTray struct{}

func (unsupportedTray) Create() error                      { return errNoTray }
func (unsupportedTray) SetIcon(png []byte) error           { return errNoTray }
func (unsupportedTray) SetTooltip(text string) error       { return errNoTray }
func (unsupportedTray) SetMenu(menu *Menu) error           { return errNoTray }
func (unsupportedTray) ShowNotification(t, m string) error { return errNoTray }
func (unsupportedTray) Show() error                        { return errNoTray }
func (unsupportedTray) Hide() error                        { return errNoTray }
func (unsupportedTray) Bounds() (int, int, int, int)       { return 0, 0, 0, 0 }
func (unsupportedTray) Run() error                         { return errNoTray }
func (unsupportedTray) Destroy()                           {}
