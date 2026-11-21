package bind

import (
	"sync"
	"testing"
)

// A test runs off the main thread, so a Set here is a pending one: it lands at
// the next Sync. That is the model, and these tests exercise the path a
// goroutine actually takes.
func TestGetAndSetRoundTrip(t *testing.T) {
	v := New("start")
	if got := v.Get(); got != "start" {
		t.Errorf("Get = %q, want start", got)
	}
	v.Set("changed")
	Sync()
	if got := v.Get(); got != "changed" {
		t.Errorf("after Set and Sync, Get = %q, want changed", got)
	}
}

// A watcher is the point of the package: it must fire for a Set.
func TestWatchFiresOnSet(t *testing.T) {
	v := New(0)
	var got []int
	v.Watch(func(n int) { got = append(got, n) })

	v.Set(1)
	Sync()
	v.Set(2)
	Sync()
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("watcher saw %v, want [1 2]", got)
	}
}

// Setting the same value again is not a change, and a watcher that fires for
// it would make "save on change" write the file on every frame.
func TestSettingTheSameValueNotifiesNobody(t *testing.T) {
	v := New("same")
	calls := 0
	v.Watch(func(string) { calls++ })

	v.Set("same")
	Sync()
	v.Set("same")
	Sync()
	if calls != 0 {
		t.Errorf("watcher fired %d times for an unchanged value", calls)
	}
	v.Set("different")
	Sync()
	if calls != 1 {
		t.Errorf("watcher fired %d times for one real change", calls)
	}
}

// This is the half that lets binding stack on widgets that take a plain
// pointer: the widget writes through Ptr, and Sync notices at the end of the
// frame.
func TestSyncNoticesAnEditMadeThroughThePointer(t *testing.T) {
	v := New(float32(1))
	var seen []float32
	v.Watch(func(f float32) { seen = append(seen, f) })

	// What a slider does.
	*v.Ptr() = 2.5
	if len(seen) != 0 {
		t.Fatal("the watcher fired during the frame, before Sync")
	}
	Sync()
	if len(seen) != 1 || seen[0] != 2.5 {
		t.Errorf("after Sync the watcher saw %v, want [2.5]", seen)
	}
	// A second Sync with no further edit must not fire again.
	Sync()
	if len(seen) != 1 {
		t.Errorf("a second Sync fired again: %v", seen)
	}
}

// A widget that writes the same value back -- which a text input does on every
// frame it is focused -- must not look like a change.
func TestSyncIgnoresAWriteThatChangesNothing(t *testing.T) {
	v := New("text")
	calls := 0
	v.Watch(func(string) { calls++ })

	*v.Ptr() = "text"
	Sync()
	if calls != 0 {
		t.Errorf("watcher fired %d times for a write of the same value", calls)
	}
}

func TestWatchIgnoresNil(t *testing.T) {
	v := New(1)
	v.Watch(nil)
	v.Set(2)
	Sync() // would panic if nil had been stored
}

// Several watchers on one value all hear about it, in the order they were
// added.
func TestEveryWatcherHears(t *testing.T) {
	v := New(0)
	var order []string
	v.Watch(func(int) { order = append(order, "first") })
	v.Watch(func(int) { order = append(order, "second") })
	v.Set(1)
	Sync()
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Errorf("watchers ran %v, want [first second]", order)
	}
}

// Set is documented as safe from any goroutine. Off the main thread it
// marshals, so with no loop running the work queues rather than racing; what
// this checks is that concurrent callers do not corrupt the value or the
// registry.
func TestSetIsSafeFromManyGoroutines(t *testing.T) {
	v := New(0)
	var wg sync.WaitGroup
	for i := 1; i <= 50; i++ {
		wg.Add(1)
		go func(n int) { defer wg.Done(); v.Set(n) }(i)
	}
	wg.Wait()
	Sync()
	// The value is whichever arrived last; what matters is that reading it
	// does not race and the registry is intact.
	_ = v.Get()
	if Count() < 1 {
		t.Error("the registry lost its values")
	}
}
