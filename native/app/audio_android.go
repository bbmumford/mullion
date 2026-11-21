//go:build android

package app

/*
#cgo LDFLAGS: -ldl -llog

int mullion_aaudio_start(int sampleRate);
int mullion_aaudio_restart(void);
*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"
)

//export mullionAudioFill
func mullionAudioFill(buf *C.float, frames C.int) {
	audioNoteFill()
	out := unsafe.Slice((*float32)(unsafe.Pointer(buf)), int(frames))
	if fill := audioFill; fill != nil {
		fill(out)
	} else {
		clear(out)
	}
}

// audioStart opens the default output via an AAudio callback stream (mono
// float, low-latency mode, ~2 bursts ≈ 8ms). libaaudio.so is dlopen'd at
// runtime: AAudio needs API 26+, and on older devices (Galaxy J2 era) the
// library is absent — StartAudio reports unavailable and the app runs
// silent. Disconnects (route change, headphones) stop the callbacks; the
// shared watchdog notices the stall and rebuilds the stream.
func audioStart(sampleRate int) error {
	if st := int(C.mullion_aaudio_start(C.int(sampleRate))); st != 0 {
		if st == -100 {
			return fmt.Errorf("AAudio unavailable (needs Android 8.0 / API 26)")
		}
		return fmt.Errorf("AAudio start error %d", st)
	}
	audioLastFill.Store(time.Now().UnixNano())
	go audioWatchdog(func() error {
		if st := int(C.mullion_aaudio_restart()); st != 0 {
			return fmt.Errorf("AAudio restart error %d", st)
		}
		return nil
	})
	return nil
}
