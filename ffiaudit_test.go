package mullion_test

import (
	"testing"

	"github.com/bbmumford/mullion/internal/ffiaudit"
)

// ffiPointerAllowed lists address-to-uintptr conversions that have been looked
// at and are safe without a runtime.Pinner, keyed by ffiaudit.Finding.Key with
// the reason each one is safe. An entry with no reason is refused: an exception
// nobody explained is one nobody can review.
var ffiPointerAllowed = map[string]string{
	// The accessibility bridge hands COM the address of a Go struct as an
	// interface pointer and converts it back with accessSelf. What keeps the
	// object alive is not the uintptr: accessibility_windows.go holds every
	// provider it has handed out in the package-level accessRetained map, with
	// a refs count alongside, so the object stays reachable and -- the
	// collector not moving heap objects -- stays where it is. These are
	// interior addresses into that object, one hop further than the analysis
	// can see.
	"native/win32backend/accessibility_windows.go:accessPut:stored face":            "the provider is held in the package-level accessRetained map with a refs count, so COM gets an address into an object Go keeps reachable",
	"native/win32backend/accessibility_windows.go:accessGetObject:&p.simple":        "handle into a provider held by accessRetained; accessRelease is a Go function that converts it straight back",
	"native/win32backend/accessibility_windows.go:closeAccess:&p.simple":            "handle into a provider held by accessRetained; accessRelease is a Go function that converts it straight back",
	"native/win32backend/accessibility_windows.go:updateAccess:&p.simple":           "handle into a provider held by accessRetained; accessRelease is a Go function that converts it straight back",
	"native/win32backend/accessibility_windows.go:updateAccess:&c.p.simple":         "handle into a provider held by accessRetained; accessRelease is a Go function that converts it straight back",
	"native/win32backend/accessibility_windows.go:updateAccess:&focusTarget.simple": "handle into a provider held by accessRetained; accessRelease is a Go function that converts it straight back",
	"native/win32backend/accessibility_msaa_windows.go:msaaResult:&p.msaa":          "the MSAA interface address goes into a VARIANT, and the provider behind it is held by accessRetained",
	"native/win32backend/accessibility_msaa_windows.go:msaaAction:&p.msaa":          "handle passed to accessRelease and accessQueue, both Go functions; the provider is held by accessRetained",

	// winmm reads the audio buffers asynchronously for as long as the device is
	// open, far beyond any one call, so a Pinner scoped to waveOpen would be
	// the wrong shape. What holds them is named in the code: the feeder
	// goroutine closes over bufs and hdrs for the life of the device, and
	// waveHdrs keeps the headers in a package variable.
	"native/app/audio_windows.go:waveOpen:stored &bufs[i][0]": "the feeder goroutine closes over bufs and hdrs for the device's lifetime and waveHdrs holds the headers in a package variable, which is what keeps the memory winmm writes into alive",

	// Not an address handed to anything: an identity token. sourceImageOpaque
	// compares it with == and stores it as srcBase to decide whether a cached
	// opacity result still describes the same pixels.
	"native/scaledimage.go:scaledImage:&src.Pix[0]": "used only as a cache identity, compared and stored by sourceImageOpaque, never dereferenced and never passed to native code",

	// The registrations are Go values implementing COM vtables, kept in the
	// environment options object for the life of the process, which is what the
	// #nosec note at the call site records.
	"web/webview2_scheme_windows.go:envOptionsRegistrations:stored r": "the registrations are held by the environment options object for the life of the process, so the array handed to COM cannot outlive them",
}

// ffiPointerInherited lists the conversions this module arrived with. They are
// not approved -- nobody has been through them -- they are recorded so that the
// audit fails on anything new while these are worked through.
//
// All of them are in code carried in from go-shirei and glaze, and all but
// three are on paths this machine cannot build or run: Direct3D 11, the Win32
// accessibility bridge, WebView2 and the Windows audio output. Reviewing one
// means reading the native call's contract to see how long it keeps the
// address, and testing on Windows. Nothing written for this module is in the
// list, which is the part worth keeping true.
//
// Shrinking it is the job. Removing an entry whose code has changed is not
// optional: Partition reports such an entry as stale and the test fails, so the
// list cannot outlive the code it describes.
//
// To rebuild the list from scratch, empty it and run this test: every finding
// is printed with the key to add, so the list is copied from output rather
// than typed.
// ffiPointerInherited lists conversions this module arrived with that nobody
// has read yet. It is empty: all 32 have been gone through -- 21 fixed with
// runtime.Pinner, 11 moved into ffiPointerAllowed above, each with the durable
// Go reference that makes it safe.
//
// Anything appearing here again is new debt, and its entry should say who is
// expected to read it and when rather than sit as a permanent exemption.
// Partition reports an entry whose code has changed as stale and fails the
// test, so the list cannot outlive what it describes.
//
// To rebuild it, empty it and run this test: every finding is printed with the
// key to add, so the list is copied from output rather than typed.
var ffiPointerInherited = []string{}

// TestFFIPointersArePinned audits the whole module for Go memory handed to
// native code as an address the runtime cannot see.
//
// A uintptr holding the address of a Go object is invisible to the collector
// and to the stack copier, so the object can be freed or moved while native
// code still holds the address. Almost nothing in this module gets the
// compiler's help with that: see internal/ffiaudit for which calls do, and why
// purego and objc are not among them.
//
// The audit is syntactic and reads every file whatever its build tags, which is
// the point -- a Windows-only or iOS-only conversion is checked here, where it
// would otherwise not even be compiled.
func TestFFIPointersArePinned(t *testing.T) {
	findings, err := ffiaudit.Scan(".", ffiaudit.Config{Allow: ffiPointerAllowed})
	if err != nil {
		t.Fatal(err)
	}
	fresh, inherited, stale := ffiaudit.Partition(findings, ffiPointerInherited)

	for _, f := range fresh {
		t.Errorf("Go address handed to native code unpinned: %s\n"+
			"\tif it is safe, add %q to ffiPointerAllowed with the reason", f, f.Key)
	}
	for _, key := range stale {
		t.Errorf("ffiPointerInherited lists %q, which matches nothing any more:\n"+
			"\tthe code it described has changed, so re-check it and delete the entry", key)
	}
	if len(inherited) > 0 {
		t.Logf("%d inherited conversions still to review (see ffiPointerInherited)", len(inherited))
	}
}
