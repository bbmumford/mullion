package mainthread

import (
	"sync"
	"testing"
)

func TestClaimGivesTheLoopToTheFirstCallerOnly(t *testing.T) {
	t.Cleanup(Release)
	if !Claim("first") {
		t.Fatalf("the loop was already held by %s", Owner())
	}
	if Claim("second") {
		t.Error("a second surface was also given the loop")
	}
	if Owner() != "first" {
		t.Errorf("Owner() = %q, want %q", Owner(), "first")
	}
	if !Claimed() {
		t.Error("Claimed() is false while the loop is held")
	}
}

func TestReleaseLetsTheNextSurfaceTakeTheLoop(t *testing.T) {
	t.Cleanup(Release)
	Claim("first")
	Release()
	if Claimed() {
		t.Fatalf("still held by %q after Release", Owner())
	}
	if !Claim("second") {
		t.Fatal("the loop could not be claimed after Release")
	}
	if Owner() != "second" {
		t.Errorf("Owner() = %q, want %q", Owner(), "second")
	}
}

func TestDrainRunsQueuedWorkInOrder(t *testing.T) {
	t.Cleanup(func() { SetWake(nil); Drain() })
	SetWake(func() {}) // no real loop here, so do not post to the platform

	var order []int
	for i := 1; i <= 3; i++ {
		n := i
		Do(func() { order = append(order, n) })
	}
	Drain()
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("ran %v, want 1 2 3", order)
	}
}

// A handler already running on the main thread may queue more work. That must
// not deadlock on the lock Drain holds, and must not be lost either.
func TestWorkQueuedFromInsideDrainStillRuns(t *testing.T) {
	t.Cleanup(func() { SetWake(nil); Drain() })
	SetWake(func() {})

	var ran []string
	Do(func() {
		ran = append(ran, "outer")
		Do(func() { ran = append(ran, "inner") })
	})
	Drain()
	if len(ran) != 2 || ran[0] != "outer" || ran[1] != "inner" {
		t.Errorf("ran %v, want outer then inner", ran)
	}
}

func TestDoWakesTheLoop(t *testing.T) {
	t.Cleanup(func() { SetWake(nil); Drain() })
	var woke int
	SetWake(func() { woke++ })
	Do(func() {})
	if woke != 1 {
		t.Errorf("the wake ran %d times, want 1", woke)
	}
	Drain()
}

// A wake installed while work is already queued has to fire, or that work waits
// for the next unrelated Do.
func TestSetWakeFiresForWorkAlreadyWaiting(t *testing.T) {
	t.Cleanup(func() { SetWake(nil); Drain() })
	SetWake(func() {})
	Do(func() {})

	var woke int
	SetWake(func() { woke++ })
	if woke != 1 {
		t.Errorf("installing a wake with work queued fired it %d times, want 1", woke)
	}
	Drain()
}

func TestDoIsSafeFromManyGoroutines(t *testing.T) {
	t.Cleanup(func() { SetWake(nil); Drain() })
	SetWake(func() {})

	var mu sync.Mutex
	count := 0
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Do(func() {
				mu.Lock()
				count++
				mu.Unlock()
			})
		}()
	}
	wg.Wait()
	Drain()
	if count != 50 {
		t.Errorf("%d of 50 queued functions ran", count)
	}
}

func TestDoIgnoresNil(t *testing.T) {
	t.Cleanup(func() { SetWake(nil); Drain() })
	SetWake(func() {})
	Do(nil)
	Drain() // would panic if nil had been queued
}
