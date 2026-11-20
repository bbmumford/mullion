package mullion_test

import (
	"testing"

	"github.com/bbmumford/mullion/internal/ffiaudit"
)

// ffiPointerAllowed lists address-to-uintptr conversions that have been looked
// at and are safe without a runtime.Pinner, keyed by ffiaudit.Finding.Key with
// the reason each one is safe. An entry with no reason is refused: an exception
// nobody explained is one nobody can review.
var ffiPointerAllowed = map[string]string{}

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
var ffiPointerInherited = []string{
	"native/app/audio_windows.go:waveOpen:stored &bufs[i][0]",
	"native/gpurender/d3d11_windows.go:createPipeline:&bd",
	"native/gpurender/d3d11_windows.go:createPipeline:&cbd",
	"native/gpurender/d3d11_windows.go:createPipeline:&cbuf",
	"native/gpurender/d3d11_windows.go:createPipeline:&elems[0]",
	"native/gpurender/d3d11_windows.go:createPipeline:&factor[0]",
	"native/gpurender/d3d11_windows.go:createPipeline:&rd",
	"native/gpurender/d3d11_windows.go:createPipeline:&samp",
	"native/gpurender/d3d11_windows.go:createPipeline:&sd",
	"native/gpurender/d3d11_windows.go:createTex:stored &pix[0]",
	"native/gpurender/d3d11_windows.go:gpuSubmit:&color[0]",
	"native/gpurender/d3d11_windows.go:gpuSubmit:&inst",
	"native/gpurender/d3d11_windows.go:gpuSubmit:&off",
	"native/gpurender/d3d11_windows.go:gpuSubmit:&rtv",
	"native/gpurender/d3d11_windows.go:gpuSubmit:&sc",
	"native/gpurender/d3d11_windows.go:gpuSubmit:&srv",
	"native/gpurender/d3d11_windows.go:gpuSubmit:&stride",
	"native/gpurender/d3d11_windows.go:gpuSubmit:&uni",
	"native/gpurender/d3d11_windows.go:gpuSubmit:&vp",
	"native/gpurender/gles.go:glesCompile:&csrc[0]",
	"native/gpurender/metal_macos.go:metalEncode:&u.pix[0]",
	"native/scaledimage.go:scaledImage:&src.Pix[0]",
	"native/win32backend/accessibility_msaa_windows.go:msaaAction:&p.msaa",
	"native/win32backend/accessibility_msaa_windows.go:msaaResult:&p.msaa",
	"native/win32backend/accessibility_windows.go:accessGetObject:&p.simple",
	"native/win32backend/accessibility_windows.go:accessPut:stored face",
	"native/win32backend/accessibility_windows.go:closeAccess:&p.simple",
	"native/win32backend/accessibility_windows.go:updateAccess:&c.p.simple",
	"native/win32backend/accessibility_windows.go:updateAccess:&focusTarget.simple",
	"native/win32backend/accessibility_windows.go:updateAccess:&p.simple",
	"web/webview2_scheme_windows.go:envOptionsRegistrations:stored r",
}

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
