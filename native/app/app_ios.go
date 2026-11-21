//go:build ios

package app

import (
	"image"

	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/internal/iconimg"
	"github.com/bbmumford/mullion/native/iosbackend"
)

// SetupWindow records the window's title and preferred size. On iOS the view is
// full-screen (safe-area); size is accepted for API parity with desktop.
func SetupWindow(title string, width, height int) {
	native.GetHost().WindowSize = native.Vec2{float32(width), float32(height)}
	iosbackend.SetupWindow(title, width, height)
}

// SetupQuiet is a no-op on iOS.
func SetupQuiet() {}

// SetupIcon is a no-op on iOS for the spike (bundle icon comes from the host
// app's Assets/Info.plist).
func SetupIcon(imagePath string) {}

// SetupIconImage is a no-op on iOS for the spike.
func SetupIconImage(img image.Image) { _ = img }

// SetupIconBytes is a no-op on iOS for the spike.
func SetupIconBytes(data []byte) {
	_ = iconimg.DecodeBytes(data)
}

// Run attaches the UIKit content view and returns; CADisplayLink drives frames
// afterward. UIApplicationWillTerminate calls generic.ExitWithCleanup.
// Must be called on the main goroutine after UIApplicationMain has started
// (mullion-mobilerun / ioshost arrange this via mullion_ios_run).
func Run(frameFn native.FrameFn) {
	iosbackend.Run(frameFn)
}
