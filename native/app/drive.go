package app

import (
	"os"
	"strconv"

	g "go.hasen.dev/generic"
	"github.com/bbmumford/mullion/native"
)

// SetupDrive listens for shirei/drive when SHIREI_DRIVE_PORT is set.
// Call after SetupWindow, before Run. No port (or a non-positive value)
// is a no-op. SetupQuiet unless SHIREI_DRIVE_FOREGROUND is on
// (1 / true / yes / on), so go test does not steal focus.
func SetupDrive() {
	p, err := strconv.Atoi(os.Getenv("SHIREI_DRIVE_PORT"))
	if err != nil || p <= 0 {
		return
	}
	native.AcceptInputCommands(p)
	if !g.EnvTruthy("SHIREI_DRIVE_FOREGROUND") {
		SetupQuiet()
	}
}
