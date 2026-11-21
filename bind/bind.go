// Package bind adds change notification to the Go-drawn surface without
// replacing how it reads state.
//
// # What binding is for
//
// A retained toolkit keeps a tree of widget objects, each holding its own copy
// of what it displays. Change your variable and the widget does not know, so
// such toolkits offer bindings: a value object widgets subscribe to, which
// tells them to update when it is set. Binding exists because the widget's
// copy can fall out of step with yours.
//
// The Go-drawn surface here has no copy to fall out of step. The frame
// function runs again and reads your variables, so what is on screen is what
// they hold. Most of what binding is for therefore does not arise, which is
// why this package is small and why nothing requires it.
//
// # What is left over
//
// Three things the frame model does not give you on its own:
//
// A change made from a goroutine does not repaint. Nothing asked for a frame,
// so the new value sits there until something else causes one. This is the
// most common way an immediate-mode interface appears to hang.
//
// Nothing tells you a value changed. To save a setting when it is edited, or
// validate it, you would keep a previous copy and compare it yourself, every
// frame, for every field.
//
// A value computed from others is recomputed every frame whether its inputs
// moved or not, which matters when the computation is not cheap.
//
// # How this stacks rather than replaces
//
// A Value hands out an ordinary pointer. Every widget that already takes a
// *bool, *string or *float32 takes one from here unchanged, so binding is
// added per value rather than adopted per application, and a program can mix
// bound and plain variables in the same frame with nothing to reconcile.
//
//	zoom := bind.New(float32(1))
//	zoom.Watch(func(v float32) { saveSetting("zoom", v) })
//
//	// in the frame
//	widgets.Slider(zoom.Ptr(), widgets.SliderAttrs{Min: 0.5, Max: 4})
//
//	// from anywhere
//	go func() { zoom.Set(2) }()
//
// Both paths notify: the slider's edit is noticed at the end of the frame, and
// the goroutine's Set is applied on the main thread and asks for a repaint.
package bind

import (
	"sync"

	"github.com/bbmumford/mullion/internal/mainthread"
	"github.com/bbmumford/mullion/native"
)

// A Value holds state the interface reads and something else may change.
//
// The zero Value is not usable; make one with New.
//
// # Which thread owns what
//
// The value itself belongs to the main thread, because that is where the frame
// reads it and where a widget writes it through Ptr. Set may be called from
// anywhere: off the main thread it leaves the new value in a pending slot that
// the next Sync picks up, rather than writing the same word a widget may be
// writing. Nothing is shared without the mutex, and nothing waits on a loop
// that may not be running.
type Value[T comparable] struct {
	// v is the main thread's copy. Ptr hands out its address, so only the main
	// thread may touch it.
	v T

	mu       sync.Mutex
	snapshot T  // the last synced value, for a reader off the main thread
	pending  *T // a Set from another goroutine, waiting for Sync
	seen     T  // what the last Sync saw, for noticing a widget's own edit
	watchers []func(T)
}

// New makes a Value holding v and registers it, so that edits made through its
// pointer are noticed. It is safe to call before the loop starts.
func New[T comparable](v T) *Value[T] {
	b := &Value[T]{v: v, snapshot: v, seen: v}
	register(b)
	return b
}

// Get reads the value.
//
// On the main thread it reads the current one. Elsewhere it reads the value as
// of the last Sync, which is the most recent one that can be read without
// racing a widget mid-frame. For a goroutine that needs to act on every change
// rather than sample the state, Watch is the right call.
func (b *Value[T]) Get() T {
	if mainthread.OnMainThread() {
		return b.v
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.snapshot
}

// Ptr is the address a widget writes through.
//
// It is the plain pointer the widgets already take, which is what lets this
// package add to them rather than replace them. A widget's edit is noticed at
// the next Sync rather than at the moment it happens, because the widget writes
// during the frame and there is nothing to compare against until the frame has
// finished.
//
// Read and write it from the main thread only. Set is the way to change a value
// from anywhere else.
func (b *Value[T]) Ptr() *T { return &b.v }

// Set changes the value, tells the watchers, and asks for a repaint.
//
// It may be called from any goroutine. On the main thread it takes effect at
// once. Elsewhere it is held until the next Sync, so it never writes a word a
// widget may be writing in the same instant; the repaint is requested
// immediately either way, so the Sync that applies it is the next frame's.
func (b *Value[T]) Set(v T) {
	if mainthread.OnMainThread() {
		b.applyLocked(v)
		return
	}
	b.mu.Lock()
	b.pending = &v
	b.mu.Unlock()
	native.RequestNextFrame()
}

// Watch calls fn whenever the value changes, however it changed -- from a
// widget, from Set, from anywhere.
//
// fn runs on the thread that noticed the change, which for a widget's edit and
// for a pending Set is the main thread, so it may touch the interface. It
// should not block, because the frame is waiting for it.
func (b *Value[T]) Watch(fn func(T)) {
	if fn == nil {
		return
	}
	b.mu.Lock()
	b.watchers = append(b.watchers, fn)
	b.mu.Unlock()
}

// applyLocked writes the value from the main thread and notifies if it moved.
func (b *Value[T]) applyLocked(v T) {
	if b.v == v {
		return
	}
	b.v = v
	b.notify(v)
	native.RequestNextFrame()
}

// notify records the new value and calls the watchers.
func (b *Value[T]) notify(v T) {
	b.mu.Lock()
	b.snapshot, b.seen = v, v
	watchers := make([]func(T), len(b.watchers))
	copy(watchers, b.watchers)
	b.mu.Unlock()

	for _, fn := range watchers {
		fn(v)
	}
}

// sync applies a pending Set, notices an edit made through Ptr, and reports
// whether the value moved. The main thread calls it, once a frame.
func (b *Value[T]) sync() bool {
	b.mu.Lock()
	pending := b.pending
	b.pending = nil
	seen := b.seen
	b.mu.Unlock()

	if pending != nil {
		b.v = *pending
	}
	if b.v == seen {
		return false
	}
	b.notify(b.v)
	return true
}

// syncer is what Sync walks. The interface exists so one registry can hold
// Values of every type.
type syncer interface{ sync() bool }

var (
	registryMu sync.Mutex
	registry   []syncer
)

func register(s syncer) {
	registryMu.Lock()
	registry = append(registry, s)
	registryMu.Unlock()
}

// Sync applies the Sets made from other goroutines and notifies the watchers of
// every value a widget changed, returning how many values moved.
//
// The runtime calls it each frame, so an application using mullion.Run never
// does. One driving the Go-drawn surface directly should call it once a frame.
func Sync() int {
	registryMu.Lock()
	all := make([]syncer, len(registry))
	copy(all, registry)
	registryMu.Unlock()

	moved := 0
	for _, s := range all {
		if s.sync() {
			moved++
		}
	}
	return moved
}

// Count reports how many values are registered, for a test that wants to know
// the registry is not growing without bound.
func Count() int {
	registryMu.Lock()
	defer registryMu.Unlock()
	return len(registry)
}
