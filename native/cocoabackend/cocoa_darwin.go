//go:build darwin && !ios

// Package cocoabackend is a direct-macOS (AppKit) backend for native. AppKit
// provides the window, run loop, and input. Rasterization writes an IOSurface
// that is set as a CALayer's contents — the window server composites it with no
// per-frame CPU copy. Metal (shirei/gpurender) is the default compositor.
// SHIREI_GPU=0 or a Metal initialization failure selects the software renderer.
package cocoabackend

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/bbmumford/mullion/native"
	"github.com/bbmumford/mullion/native/gpurender"
	"github.com/bbmumford/mullion/native/internal/iconimg"
	"github.com/bbmumford/mullion/native/internal/qwerty"
	"github.com/cli/browser"
	g "go.hasen.dev/generic"
)

// glyphCacheBudget caps total cached glyph-bitmap bytes (enables the shared core
// glyph cache; 0 would disable it). 16 MB holds many thousands of glyph masks.
const glyphCacheBudget = 16 << 20

// Window title and dimensions recorded before Run.
var (
	winTitle       string
	winIconPath    string
	winIconImg     *image.NRGBA
	winW           int
	winH           int
	winQuiet       bool
	winTransparent bool
	winUndecorated bool
	winOverlay     bool
	winKeepAlive   bool
)

// SetupQuiet maps the window without activating the app or making it key.
// Call before Run.
func SetupQuiet() {
	winQuiet = true
}

// SetupTransparent preserves per-pixel alpha through the window compositor.
// Call before Run.
func SetupTransparent() {
	winTransparent = true
}

// SetupUndecorated opens a resizable window without a title bar or buttons.
// Call before Run.
func SetupUndecorated() {
	winUndecorated = true
}

// SetupOverlay opens a transparent borderless window above normal windows,
// visible on every space. Call before Run.
func SetupOverlay() {
	winTransparent = true
	winUndecorated = true
	winOverlay = true
}

// SetupBackground keeps the application running after its last window closes.
func SetupBackground() {
	winKeepAlive = true
}

// SetupWindow records the window parameters. The window is created in Run, on
// the main thread.
func SetupWindow(title string, width int, height int) {
	winTitle = title
	winW = width
	winH = height
}

// SetupIcon records the path of the image (any NSImage-readable format, e.g.
// PNG, including .icns) used as the app's Dock icon — macOS has no title-bar
// icons. Call it before Run; empty means the default icon.
func SetupIcon(imagePath string) {
	winIconPath = imagePath
}

// SetupIconImage is SetupIcon from an in-memory image (e.g. decoded from
// go:embed-ed bytes) instead of a file. It takes precedence over SetupIcon.
func SetupIconImage(img image.Image) {
	winIconImg = iconimg.FromImage(img)
}

// init locks the main goroutine to the main OS thread (thread 0). AppKit requires
// NSApplication/NSWindow to live on thread 0, and init runs there — before main()
// and before any goroutines exist — so main() and therefore Run stay on thread 0
// even after the app spawns background goroutines first. Locking only inside Run
// is too late: by then the scheduler may already have migrated the main goroutine
// off thread 0 (this crashed apps like vbeam/local_gui, which start goroutines
// before Run with "NSWindow should only be instantiated on the main thread").
func init() {
	runtime.LockOSThread()
}

// Run opens the window and runs the AppKit event loop. It must be called
// from the program's main goroutine (AppKit requires the main thread) and
// does not return: quit paths call generic.ExitWithCleanup so AddExitCleanup
// handlers run. AppKit [NSApp terminate:] would otherwise exit without
// returning to Go. System fonts are initialized by shirei on the first
// frame (RunFrameFn), not here.
func Run(fn native.FrameFn) {
	runtime.LockOSThread()

	currentWindow.frameFn = fn
	mainWindow.unsubscribe = native.RegisterWindowUI(mainWindow.ui)

	native.GetHost().GlyphCacheBudgetBytes = glyphCacheBudget
	native.GetHost().TransparentBackground = winTransparent
	native.GetHost().EscapeHatchBackendContext = Context{window: mainWindow}

	leave := enterWindow(mainWindow)
	setupWindow(winTitle, winW, winH)
	if winIconImg != nil {
		b := winIconImg.Bounds()
		setAppIconRGBA(winIconImg.Pix, b.Dx(), b.Dy())
	} else if winIconPath != "" {
		setAppIcon(winIconPath)
	}
	enableZerocopy()
	if g.EnvFalsy("SHIREI_GPU") {
		fmt.Fprintln(os.Stderr, "shirei: software renderer (SHIREI_GPU=0)")
	} else if err := gpurender.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "shirei: software renderer (%v)\n", err)
	} else {
		gpuOK = true
		gpurender.SetCompleteFunc(gpuPresentComplete)
	}
	// Input commands hold the frame lock. Publish the callback only after its
	// native view is initialized, before worker requests can read that view.
	native.SetBackendWake(requestRedraw)
	mainWindow.ready = true
	leave()
	runApp()
	g.ExitWithCleanup(0)
}

func exitWithCleanup() {
	g.ExitWithCleanup(0)
}

// gpuOK is set once at Run if Metal init succeeds and SHIREI_GPU is not 0.
var gpuOK bool

// IOSurface present pool. A few surfaces are rotated so we never render into the
// one the compositor is currently reading (IOSurfaceIsInUse). Static frames present
// nothing — the layer keeps showing the last surface.
type ioSurface struct {
	ref         unsafe.Pointer
	w, h        int
	gpuBusy     bool
	pendingHash uint64
	gpuSeq      uint64
}

func renderAndPresent() {
	scale := native.GetHost().WindowScale
	if scale <= 0 {
		scale = 1
	}
	dw := int(native.GetHost().WindowSize[0]*scale + 0.5)
	dh := int(native.GetHost().WindowSize[1]*scale + 0.5)
	if dw <= 0 || dh <= 0 {
		return
	}

	if currentWindow.havePresented && currentWindow.frameHash == currentWindow.lastPresentedHash && dw == currentWindow.presentW && dh == currentWindow.presentH && !currentWindow.presentDeferred {
		native.EmitFrameMetrics()
		return
	}
	if gpuOK && len(currentWindow.surfacePool) > 0 && currentWindow.surfacePool[0].w == dw && currentWindow.surfacePool[0].h == dh && gpuInFlightHash(currentWindow.frameHash) {
		native.EmitFrameMetrics()
		return
	}

	ensureSurfacePool(dw, dh)
	slot := pickFreeSurface()
	if slot == nil {
		currentWindow.presentDeferred = true
		setWantsFrame(true)
		return
	}
	currentWindow.presentDeferred = false

	if gpuOK {
		err := gpurender.Render(slot.ref, dw, dh, scale, currentWindow.frameSurfaces, currentWindow.frameGlyphRuns, currentWindow.frameGlyphsAdded, currentWindow.frameGlyphsEvicted, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gpurender: %v; software this frame\n", err)
			renderSoftware(slot.ref, dw, dh, scale, currentWindow.frameSurfaces, currentWindow.frameGlyphRuns)
		} else {
			currentWindow.gpuSubmitSeq++
			slot.gpuBusy = true
			slot.pendingHash = currentWindow.frameHash
			slot.gpuSeq = currentWindow.gpuSubmitSeq
			t := &native.ActiveUI().FrameTimings
			t.Painted = true
			t.PaintEnd = time.Now()
			native.EmitFrameMetrics()
			return
		}
	} else {
		renderSoftware(slot.ref, dw, dh, scale, currentWindow.frameSurfaces, currentWindow.frameGlyphRuns)
	}

	setLayerContents(slot.ref)
	currentWindow.lastPresented = slot.ref
	currentWindow.lastPresentedHash = currentWindow.frameHash
	currentWindow.presentW, currentWindow.presentH, currentWindow.havePresented = dw, dh, true
	t := &native.ActiveUI().FrameTimings
	t.Painted = true
	t.PaintEnd = time.Now()
	native.EmitFrameMetrics()
}

func renderSoftware(s unsafe.Pointer, dw, dh int, scale float32, surfaces []native.Surface, runs []native.GlyphRun) {
	base, stride := ioSurfaceLockBuf(s)
	if base == nil {
		return
	}
	buf := unsafe.Slice((*byte)(base), stride*dh)
	currentWindow.softRenderer.RenderInto(buf, stride, dw, dh, scale, surfaces, runs)
	ioSurfaceUnlockBuf(s)
}

func ensureSurfacePool(w, h int) {
	if len(currentWindow.surfacePool) > 0 && currentWindow.surfacePool[0].w == w && currentWindow.surfacePool[0].h == h {
		return
	}
	if gpuOK {
		gpurender.WaitIdle()
	}
	for _, s := range currentWindow.surfacePool {
		if gpuOK {
			gpurender.Forget(s.ref)
		}
		delete(surfaceWindows, s.ref)
		ioSurfaceRelease(s.ref)
	}
	currentWindow.surfacePool = currentWindow.surfacePool[:0]
	currentWindow.lastPresented = nil
	currentWindow.havePresented = false
	currentWindow.gpuPresentedSeq = currentWindow.gpuSubmitSeq
	for i := 0; i < 3; i++ {
		ref := ioSurfaceCreateBuf(w, h)
		if ref == nil {
			continue
		}
		surfaceWindows[ref] = currentWindow
		currentWindow.surfacePool = append(currentWindow.surfacePool, ioSurface{ref: ref, w: w, h: h})
	}
}

func gpuInFlightHash(h uint64) bool {
	for i := range currentWindow.surfacePool {
		s := &currentWindow.surfacePool[i]
		if s.gpuBusy && s.pendingHash == h {
			return true
		}
	}
	return false
}

func gpuPresentComplete(surf unsafe.Pointer) {
	w := surfaceWindows[surf]
	if w == nil || w.closed {
		return
	}
	defer enterWindow(w)()
	var slot *ioSurface
	for i := range currentWindow.surfacePool {
		if currentWindow.surfacePool[i].ref == surf {
			slot = &currentWindow.surfacePool[i]
			break
		}
	}
	if slot == nil || !slot.gpuBusy {
		return
	}
	if slot.gpuSeq < currentWindow.gpuPresentedSeq {
		slot.gpuBusy = false
		return
	}
	setLayerContents(slot.ref)
	currentWindow.lastPresented = slot.ref
	currentWindow.lastPresentedHash = slot.pendingHash
	currentWindow.presentW, currentWindow.presentH, currentWindow.havePresented = slot.w, slot.h, true
	currentWindow.gpuPresentedSeq = slot.gpuSeq
	slot.gpuBusy = false
}

func pickFreeSurface() *ioSurface {
	for i := range currentWindow.surfacePool {
		s := &currentWindow.surfacePool[i]
		if s.ref == currentWindow.lastPresented || s.gpuBusy || ioSurfaceInUseBuf(s.ref) {
			continue
		}
		return s
	}
	return nil
}

func produceFrame(w, h float64) {
	if framesInProgress > 0 || currentWindow.closing || currentWindow.closed {
		currentWindow.gWantsFrame = true
		return
	}
	framesInProgress++
	defer func() { framesInProgress-- }()
	native.GetHost().WindowSize = native.Vec2{float32(w), float32(h)}
	native.GetHost().WindowScale = float32(backingScale())
	currentWindow.lastProducedW, currentWindow.lastProducedH = float32(w), float32(h)

	flushPendingFrameText()
	flushAccessAction()

	out := native.RunFrameFnLocked(currentWindow.frameFn)
	if out.AccessChanged {
		updateAccess(out.Access)
	}

	clear(currentWindow.frameSurfaces) // Release immutable glyph data from the previous snapshot.
	currentWindow.frameSurfaces = append(currentWindow.frameSurfaces[:0], out.Surfaces...)
	currentWindow.frameGlyphRuns = append(currentWindow.frameGlyphRuns[:0], out.GlyphRuns...)
	currentWindow.frameGlyphsAdded = append(currentWindow.frameGlyphsAdded[:0], out.GlyphsAdded...)
	currentWindow.frameGlyphsEvicted = append(currentWindow.frameGlyphsEvicted[:0], out.GlyphsEvicted...)
	currentWindow.haveFrame = true

	if out.Copy != "" {
		setClipboard(out.Copy)
	}
	if out.Paste {
		currentWindow.pendingPaste = getClipboard()
		currentWindow.hasPendingPaste = true
		requestRedraw()
	}
	if out.OpenURL != "" {
		openURL(out.OpenURL)
	}

	setWantsFrame(out.NextFrameRequested || len(currentWindow.accessPending) > 0)
	currentWindow.frameHash = out.SurfacesHash
}

func flushPendingFrameText() {
	if currentWindow.hasPendingPaste {
		currentWindow.pendingText += currentWindow.pendingPaste
		currentWindow.pendingPaste = ""
		currentWindow.hasPendingPaste = false
	}
	if currentWindow.pendingText != "" {
		native.GetFrameInput().Text += currentWindow.pendingText
		currentWindow.pendingText = ""
	}
}

func needsProduce(w, h float64) bool {
	return !currentWindow.haveFrame || float32(w) != currentWindow.lastProducedW || float32(h) != currentWindow.lastProducedH || float32(backingScale()) != native.GetHost().WindowScale
}

func frameRequested() bool { return native.FrameRequested() }

func caretX() float64      { return float64(native.GetHost().CaretPos[0]) }
func caretY() float64      { return float64(native.GetHost().CaretPos[1]) }
func caretHeight() float64 { return float64(native.GetHost().CaretHeight) }

func openURL(url string) {
	if url == "" {
		return
	}
	_ = browser.OpenURL(url)
}

const (
	mouseMove = 0
	mouseDown = 1
	mouseUp   = 2
	mouseDrag = 3
)

func onMouse(x, y float64, action, button int) {
	np := native.Vec2{float32(x), float32(y)}
	prev := native.GetInputState().MousePoint
	native.GetFrameInput().Motion = native.Vec2Add(native.GetFrameInput().Motion, native.Vec2Sub(np, prev))
	native.GetInputState().MousePoint = np
	native.GetInputState().MouseButton = native.MouseButton(button)

	switch action {
	case mouseDown:
		native.GetFrameInput().Mouse = native.MouseClick
	case mouseUp:
		native.GetFrameInput().Mouse = native.MouseRelease
	}
}

func onScroll(dx, dy float64) {
	native.GetFrameInput().Scroll = native.Vec2Add(native.GetFrameInput().Scroll,
		native.Vec2{float32(dx), float32(dy)})
}

func onWindowFocus(focused bool) {
	native.GetHost().WindowFocused = focused
	native.RequestNextFrame()
}

const (
	nsShift   = 1 << 17
	nsControl = 1 << 18
	nsOption  = 1 << 19
	nsCommand = 1 << 20
)

func onModifiers(flags uint) {
	var m native.Modifiers
	if flags&nsShift != 0 {
		m |= native.ModShift
	}
	if flags&nsControl != 0 {
		m |= native.ModCtrl
	}
	if flags&nsOption != 0 {
		m |= native.ModAlt
	}
	if flags&nsCommand != 0 {
		m |= native.ModCmd
	}
	native.GetInputState().Modifiers = m
	syncModKey(m, native.ModShift, native.KeyShift)
	syncModKey(m, native.ModCtrl, native.KeyCtrl)
	syncModKey(m, native.ModAlt, native.KeyAlt)
	syncModKey(m, native.ModCmd, native.KeyCommand)
}

func syncModKey(m, bit native.Modifiers, k native.KeyCode) {
	if m&bit != 0 {
		g.SliceAddUniq(&native.GetInputState().DownKeys, k)
	} else {
		g.SliceRemove(&native.GetInputState().DownKeys, k)
	}
}

func keyDown(vkey int, bare string) { onKeyDown(vkey, bare) }

func setCompositionFromUTF16Offsets(text string, startUTF16, endUTF16 int) {
	onSetComposition(text, startUTF16, endUTF16)
}

func onKeyDown(vkey int, bare string) {
	if code := mapVKey(uint16(vkey), bare); code != native.KeyCodeNone {
		native.GetFrameInput().Key = code
		g.SliceAddUniq(&native.GetInputState().DownKeys, code)
	}
}

func queueCommittedText(text string) {
	if isPrintable(text) {
		currentWindow.pendingText += text
	}
}

func onCommitText(text string) { queueCommittedText(text) }

func utf16OffsetToRuneOffset(s string, units int) int {
	if units <= 0 {
		return 0
	}
	u16 := utf16.Encode([]rune(s))
	if units > len(u16) {
		units = len(u16)
	}
	return len(utf16.Decode(u16[:units]))
}

func onSetComposition(text string, startUTF16, endUTF16 int) {
	start := utf16OffsetToRuneOffset(text, startUTF16)
	end := utf16OffsetToRuneOffset(text, endUTF16)
	if start > end {
		start, end = end, start
	}
	native.GetInputState().Composition = text
	native.GetInputState().CompositionSel = [2]int{start, end}
	native.RequestNextFrame()
}

func onKeyUp(vkey int, bare string) {
	if code := mapVKey(uint16(vkey), bare); code != native.KeyCodeNone {
		g.SliceRemove(&native.GetInputState().DownKeys, code)
	}
}

func isPrintable(s string) bool {
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	if r >= 0xF700 && r <= 0xF8FF {
		return false
	}
	return r >= 0x20 && r != 0x7f
}

const (
	vkReturn        = 0x24
	vkTab           = 0x30
	vkSpace         = 0x31
	vkDelete        = 0x33
	vkEscape        = 0x35
	vkKeypadEnter   = 0x4C
	vkForwardDelete = 0x75
	vkHome          = 0x73
	vkEnd           = 0x77
	vkPageUp        = 0x74
	vkPageDown      = 0x79
	vkLeft          = 0x7B
	vkRight         = 0x7C
	vkDown          = 0x7D
	vkUp            = 0x7E
	vkF1            = 0x7A
	vkF2            = 0x78
	vkF3            = 0x63
	vkF4            = 0x76
	vkF5            = 0x60
	vkF6            = 0x61
	vkF7            = 0x62
	vkF8            = 0x64
	vkF9            = 0x65
	vkF10           = 0x6D
	vkF11           = 0x67
	vkF12           = 0x6F
)

func mapVKey(vk uint16, bare string) native.KeyCode {
	switch vk {
	case vkLeft:
		return native.KeyLeft
	case vkRight:
		return native.KeyRight
	case vkUp:
		return native.KeyUp
	case vkDown:
		return native.KeyDown
	case vkReturn, vkKeypadEnter:
		return native.KeyEnter
	case vkEscape:
		return native.KeyEscape
	case vkDelete:
		return native.KeyDeleteBackward
	case vkForwardDelete:
		return native.KeyDeleteForward
	case vkHome:
		return native.KeyHome
	case vkEnd:
		return native.KeyEnd
	case vkPageUp:
		return native.KeyPageUp
	case vkPageDown:
		return native.KeyPageDown
	case vkTab:
		return native.KeyTab
	case vkSpace:
		return native.KeySpace
	case vkF1:
		return native.KeyF1
	case vkF2:
		return native.KeyF2
	case vkF3:
		return native.KeyF3
	case vkF4:
		return native.KeyF4
	case vkF5:
		return native.KeyF5
	case vkF6:
		return native.KeyF6
	case vkF7:
		return native.KeyF7
	case vkF8:
		return native.KeyF8
	case vkF9:
		return native.KeyF9
	case vkF10:
		return native.KeyF10
	case vkF11:
		return native.KeyF11
	case vkF12:
		return native.KeyF12
	}
	if code := qwerty.FromMacVK(vk); code != native.KeyCodeNone {
		return code
	}
	if bare != "" {
		r := []rune(bare)[0]
		if r >= 'a' && r <= 'z' {
			return native.KeyCode(r - 'a' + 'A')
		}
		if r < 128 {
			return native.KeyCode(r)
		}
	}
	return native.KeyCodeNone
}
