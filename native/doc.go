// Package native is the Go-drawn user interface surface: windows whose every
// pixel is drawn by this process rather than by the platform's widgets or by a
// browser engine.
//
// It is go-shirei, merged into mullion. The import path and the package name
// changed; the design did not. See ALTERED.md beside this file.
//
// # The frame model
//
// There is no widget tree and there are no widget objects. A frame function
// runs once per frame and describes what the window contains by calling into
// this package; the next frame runs it again. State lives in your variables,
// and what is on screen is whatever those variables held when the frame ran.
//
//	func draw() {
//	    native.Container(native.Attrs(native.Pad(16), native.Gap(8)), func() {
//	        native.Label("Hello")
//	    })
//	}
//
// Nothing is created and nothing is destroyed, so nothing can fall out of step
// with your state and there is no second copy to keep in sync. The cost is
// that the function runs again from the top each frame, which is why it should
// read state and describe layout rather than do work.
//
// # Asking for a frame
//
// Frames are drawn on demand, not on a clock: input causes one, and so does
// RequestNextFrame. A change made from a goroutine draws nothing until
// something asks, which is the usual reason an interface built this way
// appears to have stopped. Either call RequestNextFrame after the change, or
// hold the value in a mullion/bind.Value, which asks for you.
//
// # Threads
//
// The frame function runs on the thread the platform gave the process, and so
// does everything this package calls. Touch it from one goroutine only, and
// reach it from others through mullion's Session.Do or bind.Value.Set.
//
// # What is where
//
// This package has the layout primitives, the attributes that configure them,
// text, images, input and the host interface the backends implement. The
// widgets subpackage has the controls built on top: buttons, fields, lists,
// tables and the rest. A program normally dot-imports both.
package native
