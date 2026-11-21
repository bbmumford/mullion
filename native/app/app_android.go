//go:build android

package app

import (
	"image"

	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/androidbackend"
	"github.com/bbmumford/mullion/native/internal/iconimg"
)

// SetupWindow records the window's title and preferred size. On Android the
// surface is always full-screen; size is accepted for API parity with desktop.
func SetupWindow(title string, width, height int) {
	native.GetHost().WindowSize = native.Vec2{float32(width), float32(height)}
	androidbackend.SetupWindow(title, width, height)
}

// SetupQuiet is a no-op on Android.
func SetupQuiet() {}

// SetupIcon is a no-op on Android for the spike (the launcher icon comes from
// the APK's resources, assembled by mullion-mobilerun / mullion-bundle).
func SetupIcon(imagePath string) {}

// SetupIconImage is a no-op on Android for the spike.
func SetupIconImage(img image.Image) { _ = img }

// SetupIconBytes is a no-op on Android for the spike.
func SetupIconBytes(data []byte) {
	_ = iconimg.DecodeBytes(data)
}

// Run enters the NativeActivity frame loop and never returns. Activity destroy
// calls generic.ExitWithCleanup. Must be called on the glue's app thread — the
// packaging tool's export file arranges for main() to run there
// (android_main → mullion_android_main → main).
func Run(frameFn native.FrameFn) {
	androidbackend.Run(frameFn)
}
