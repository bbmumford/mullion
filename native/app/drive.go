package app

import (
	"os"
	"strconv"

	"github.com/bbmumford/mullion/native"
	g "go.hasen.dev/generic"
)

// SetupDrive listens for mullion/drive when MULLION_DRIVE_PORT is set.
// Call after SetupWindow, before Run. No port (or a non-positive value)
// is a no-op. SetupQuiet unless MULLION_DRIVE_FOREGROUND is on
// (1 / true / yes / on), so go test does not steal focus.
func SetupDrive() {
	p, err := strconv.Atoi(os.Getenv("MULLION_DRIVE_PORT"))
	if err != nil || p <= 0 {
		return
	}
	native.AcceptInputCommands(p)
	if !g.EnvTruthy("MULLION_DRIVE_FOREGROUND") {
		SetupQuiet()
	}
}
