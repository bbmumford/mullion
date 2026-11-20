package darkmode

import (
	"sync"
	"testing"

	"github.com/bbmumford/mullion/native"
)

func TestOSDarkModeBasic(t *testing.T) {
	// Calling OSDarkMode should not panic and return a boolean.
	v := OSDarkMode()
	t.Logf("OSDarkMode returned: %v", v)

	// Multiple calls must be consistent
	for range 1000 {
		if OSDarkMode() != v {
			t.Fatalf("OSDarkMode returned inconsistent value during rapid sequential calls")
		}
	}
}

func TestOSDarkModeConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 5000 {
				_ = OSDarkMode()
			}
		}()
	}
	wg.Wait()
}

func TestSetDarkModeUpdatesAndRequestsFrame(t *testing.T) {
	initial := isDarkMode.Load()
	defer isDarkMode.Store(initial)

	// Ensure NextFrame flag starts cleared
	native.GetHost().NextFrame.Store(false)

	target := !initial
	setDarkMode(target)

	if isDarkMode.Load() != target {
		t.Fatalf("expected isDarkMode to be %v, got %v", target, isDarkMode.Load())
	}
	if !native.FrameRequested() {
		t.Fatalf("expected native.FrameRequested() to be true after darkmode change")
	}

	// Setting the same value again should not re-trigger frame request if reset
	native.GetHost().NextFrame.Store(false)
	setDarkMode(target)
	if native.FrameRequested() {
		t.Fatalf("expected no new frame request when setting identical darkmode value")
	}
}
