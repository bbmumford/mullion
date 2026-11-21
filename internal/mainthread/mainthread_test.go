package mainthread

import (
	"sort"
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

// Post and Do reach the same thread by different routes. The part of Post that
// can lose work is its registry: one C callback serves every posted call, so the
// closure cannot travel with it and an id is looked up instead.
//
// These tests drive that registry directly rather than waiting for a platform
// queue. A test binary runs no event loop, so on a platform where libdispatch
// or GLib resolves, a posted call is handed over and simply waits there -- which
// would make a test that drains and asserts look like a loss when it is only an
// absence of anything to service it.

// pendingPosts returns the ids waiting in the registry, oldest first.
func pendingPosts(t *testing.T) []uintptr {
	t.Helper()
	postMu.Lock()
	defer postMu.Unlock()
	ids := make([]uintptr, 0, len(postFns))
	for id := range postFns {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func TestPostRegistersWorkAndRunsItExactlyOnce(t *testing.T) {
	t.Cleanup(func() { SetWake(nil); Drain() })

	var ran int
	Post(func() { ran++ })

	ids := pendingPosts(t)
	if len(ids) != 1 {
		t.Fatalf("%d posted calls waiting, want 1", len(ids))
	}
	runPosted(ids[0])
	if ran != 1 {
		t.Errorf("the posted function ran %d times, want 1", ran)
	}
	// The id is consumed: the platform callback cannot fire twice for one call,
	// but if it did, the work must not run again.
	runPosted(ids[0])
	if ran != 1 {
		t.Errorf("a repeated callback ran it again (%d times)", ran)
	}
}

func TestPostKeepsEachCallSeparate(t *testing.T) {
	t.Cleanup(func() { SetWake(nil); Drain() })

	var got []int
	for i := 1; i <= 5; i++ {
		n := i
		Post(func() { got = append(got, n) })
	}
	ids := pendingPosts(t)
	if len(ids) != 5 {
		t.Fatalf("%d posted calls waiting, want 5", len(ids))
	}
	for _, id := range ids {
		runPosted(id)
	}
	if len(got) != 5 {
		t.Fatalf("ran %d of 5: %v", len(got), got)
	}
	for i, n := range got {
		if n != i+1 {
			t.Errorf("ids did not map back to their own closures: %v", got)
			break
		}
	}
}

func TestPostIgnoresNil(t *testing.T) {
	t.Cleanup(func() { SetWake(nil); Drain() })
	before := len(pendingPosts(t))
	Post(nil)
	if after := len(pendingPosts(t)); after != before {
		t.Errorf("Post(nil) registered something: %d entries, was %d", after, before)
	}
}

// The registry must not grow: a call that has run is gone from it, or a
// long-lived process leaks one entry per dispatch.
func TestPostLeavesNothingBehind(t *testing.T) {
	t.Cleanup(func() { SetWake(nil); Drain() })

	for range 20 {
		Post(func() {})
	}
	for _, id := range pendingPosts(t) {
		runPosted(id)
	}
	if left := len(pendingPosts(t)); left != 0 {
		t.Errorf("%d entries left in the registry after every callback fired", left)
	}
}

// An id the platform never gave out must do nothing rather than panic: a stray
// or late callback is the shape of bug this registry exists to absorb.
func TestRunPostedWithAnUnknownIdDoesNothing(t *testing.T) {
	runPosted(0)
	runPosted(^uintptr(0))
}
