//go:build linux

package window

import (
	"github.com/bbmumford/mullion/native/waylandbackend"
	"os"
)

func activatePlatformApp() {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		waylandbackend.Activate()
	}
}
