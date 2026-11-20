package ffiaudit

import (
	"strings"
	"testing"
)

// scan runs the analysis over one source fixture and returns the expressions it
// reported, so a case reads as "this shape is caught" or "this shape is not".
func scan(t *testing.T, src string, cfg Config) []string {
	t.Helper()
	findings, err := ScanSource("p.go", "package p\n\n"+src, cfg)
	if err != nil {
		t.Fatalf("parsing the fixture: %v", err)
	}
	var out []string
	for _, f := range findings {
		out = append(out, f.Expr)
	}
	return out
}

func wants(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("reported %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("reported[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

const procPreamble = `
import (
	"syscall"
	"unsafe"
	"runtime"
)

var dll = syscall.NewLazyDLL("x.dll")
var proc = dll.NewProc("f")
`

// The whole point of the audit: the two calls look identical at the call site,
// and only one of them keeps the referent alive.
func TestDirectProcCallIsSafeAndEverythingElseIsNot(t *testing.T) {
	got := scan(t, procPreamble+`
type caller interface{ Call(a ...uintptr) (uintptr, uintptr, error) }

func direct(x int)            { proc.Call(uintptr(unsafe.Pointer(&x))) }
func viaSyscall(x int)        { syscall.SyscallN(0, uintptr(unsafe.Pointer(&x))) }
func viaInterface(c caller, x int) { c.Call(uintptr(unsafe.Pointer(&x))) }
func send(a uintptr)          { _ = a }
func viaWrapper(x int)        { send(uintptr(unsafe.Pointer(&x))) }
`, Config{})
	// direct and viaSyscall are covered by //go:uintptrescapes; the interface
	// and the wrapper have lost it.
	wants(t, got, "&x", "&x")
}

// purego.SyscallN carries //go:uintptrescapes, so a conversion written in its
// own argument list is safe -- easy to assume otherwise, and asserting the
// wrong way round here would have the audit demand pins that buy nothing.
//
// objc is the opposite: Send reaches objc_msgSend through a func value built by
// RegisterFunc, which has no directive on it, and that is the mechanism most of
// this module's macOS code goes through.
func TestPuregoSyscallNIsSafeAndObjcIsNot(t *testing.T) {
	got := scan(t, `
import (
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

func viaPurego(x int)  { purego.SyscallN(0, uintptr(unsafe.Pointer(&x))) }
func viaObjc(id objc.ID, sel objc.SEL, x int) { id.Send(sel, uintptr(unsafe.Pointer(&x))) }
`, Config{})
	wants(t, got, "&x")
}

// A wrapper around an annotated call loses the directive, which is the whole
// reason a COM or msgSend helper cannot be trusted with an address.
func TestAWrapperAroundSyscallNLosesTheDirective(t *testing.T) {
	got := scan(t, `
import (
	"unsafe"

	"github.com/ebitengine/purego"
)

func call(fn uintptr, args ...uintptr) uintptr {
	r, _, _ := purego.SyscallN(fn, args...)
	return r
}

func viaWrapper(x int) { call(0, uintptr(unsafe.Pointer(&x))) }
`, Config{})
	wants(t, got, "&x")
}

// FFI bound inside a package-level literal is where a library resolves its
// symbols, and a scan of top-level declarations alone never enters it.
func TestConversionsInsideFunctionLiteralsAreFound(t *testing.T) {
	got := scan(t, `
import (
	"sync"
	"unsafe"
)

func send(a uintptr) { _ = a }

var load = sync.OnceValue(func() int {
	var x int
	send(uintptr(unsafe.Pointer(&x)))
	return x
})
`, Config{})
	wants(t, got, "&x")
}

// The name has to lead back to the code, so a literal is reported under the
// variable that holds it.
func TestAFindingInsideALiteralIsNamedAfterItsVariable(t *testing.T) {
	findings, err := ScanSource("p.go", `package p

import (
	"sync"
	"unsafe"
)

func send(a uintptr) { _ = a }

var load = sync.OnceValue(func() int {
	var x int
	send(uintptr(unsafe.Pointer(&x)))
	return x
})
`, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("reported %d findings, want 1", len(findings))
	}
	if got := findings[0].Func; got != "load.func" {
		t.Errorf("Func = %q, want %q so the name leads back to the code", got, "load.func")
	}
}

// A pin in the enclosing function is still in effect inside a literal that runs
// before its Unpin.
func TestAPinOutsideALiteralCoversInsideIt(t *testing.T) {
	got := scan(t, `
import (
	"runtime"
	"unsafe"
)

func send(a uintptr) { _ = a }

func outer(x int) {
	var p runtime.Pinner
	p.Pin(&x)
	defer p.Unpin()
	func() { send(uintptr(unsafe.Pointer(&x))) }()
}
`, Config{})
	wants(t, got)
}

func TestPinnedBeforeTheCallIsAccepted(t *testing.T) {
	got := scan(t, procPreamble+`
func send(a uintptr) { _ = a }

func pinnedFirst(x int) {
	var p runtime.Pinner
	p.Pin(&x)
	defer p.Unpin()
	send(uintptr(unsafe.Pointer(&x)))
}
`, Config{})
	wants(t, got)
}

// A pin written after the conversion protects nothing. Matching a pin anywhere
// in the function would call this correct, which is why position is compared.
func TestPinnedAfterTheCallIsStillReported(t *testing.T) {
	got := scan(t, procPreamble+`
func send(a uintptr) { _ = a }

func pinnedTooLate(x int) {
	send(uintptr(unsafe.Pointer(&x)))
	var p runtime.Pinner
	p.Pin(&x)
	defer p.Unpin()
}
`, Config{})
	wants(t, got, "&x")
}

// Pinning an object pins all of it, so an address into a field of something
// already pinned is not a finding. Without this, correct code is reported and
// the allowlist fills up with entries that mean nothing.
//
// An index is the exception, and &v.g[0] below is reported even though pinning
// v does cover an embedded array: with a slice field the same syntax would not
// be covered at all, and the difference is not visible without type
// information. See TestPinningASliceDoesNotCoverItsElements.
func TestPinningTheRootCoversAddressesInsideIt(t *testing.T) {
	got := scan(t, procPreamble+`
type s struct{ f int; g [4]byte }

func send(a uintptr) { _ = a }

func pinnedRoot(v s) {
	var p runtime.Pinner
	p.Pin(&v)
	defer p.Unpin()
	send(uintptr(unsafe.Pointer(&v.f)))
	send(uintptr(unsafe.Pointer(&v.g[0])))
}

func pinnedElement(b []byte) {
	var p runtime.Pinner
	p.Pin(&b[0])
	defer p.Unpin()
	send(uintptr(unsafe.Pointer(&b[0])))
}
`, Config{})
	// &v.f is covered by the pin of v; &v.g[0] is an index, so it is asked for
	// by name even though pinning v happens to cover it.
	wants(t, got, "&v.g[0]")
}

// Pinning a slice pins its header, which is a different object from the
// backing array &b[0] points into: accepting it would wave through a fix that
// protects the wrong memory. Telling a slice from an array needs type
// information this analysis does not have, so it refuses both and asks for the
// element to be pinned as written.
func TestPinningASliceDoesNotCoverItsElements(t *testing.T) {
	got := scan(t, procPreamble+`
func send(a uintptr) { _ = a }

func pinnedHeaderOnly(b []byte) {
	var p runtime.Pinner
	p.Pin(&b)
	defer p.Unpin()
	send(uintptr(unsafe.Pointer(&b[0])))
}
`, Config{})
	wants(t, got, "&b[0]")
}

// An address of a package-level variable is safe: its storage is static or kept
// reachable by the variable, so it is neither collected nor moved.
func TestAddressOfAPackageVariableIsAccepted(t *testing.T) {
	got := scan(t, procPreamble+`
var table [4]uintptr
var buf []byte

func send(a uintptr) { _ = a }

func global()  { send(uintptr(unsafe.Pointer(&table[0]))) }
func slice()   { send(uintptr(unsafe.Pointer(&buf[0]))) }
`, Config{})
	wants(t, got)
}

// A local of the same name is not that package variable.
func TestALocalShadowingAPackageVariableIsReported(t *testing.T) {
	got := scan(t, procPreamble+`
var table [4]uintptr

func send(a uintptr) { _ = a }

func shadow() {
	var table [4]uintptr
	send(uintptr(unsafe.Pointer(&table[0])))
}
`, Config{})
	wants(t, got, "&table[0]")
}

// Storing an address is the hazard no call can cover: native code reads the
// place later, outside any argument list.
func TestStoresBeyondTheCallAreReported(t *testing.T) {
	got := scan(t, procPreamble+`
type item struct{ data uintptr }

var slot [2]uintptr
var global uintptr

func fieldLiteral(s *uint16) item { return item{data: uintptr(unsafe.Pointer(s))} }

func fieldAssign(s *uint16) {
	var it item
	it.data = uintptr(unsafe.Pointer(s))
}

func elementAssign(s *uint16) { slot[0] = uintptr(unsafe.Pointer(s)) }

func packageVarAssign(s *uint16) { global = uintptr(unsafe.Pointer(s)) }
`, Config{})
	wants(t, got, "s", "s", "s", "s")
}

// A stored address is fine once the referent is pinned for as long as native
// code holds it.
func TestAPinnedStoreIsAccepted(t *testing.T) {
	got := scan(t, procPreamble+`
type item struct{ data uintptr }

func fieldPinned(s *uint16) {
	var p runtime.Pinner
	p.Pin(s)
	defer p.Unpin()
	var it item
	it.data = uintptr(unsafe.Pointer(s))
}
`, Config{})
	wants(t, got)
}

// A plain pointer-to-uintptr conversion in an argument is not reported: where
// it points is not visible syntactically, and reporting every one would bury
// the findings that are plainly an address of Go memory. The store case above
// is what catches the dangerous half of that.
func TestAPointerVariableInAnArgumentIsNotReported(t *testing.T) {
	got := scan(t, procPreamble+`
func send(a uintptr) { _ = a }

func viaPointerVar(s *uint16) { send(uintptr(unsafe.Pointer(s))) }
`, Config{})
	wants(t, got)
}

func TestAllowSilencesByKeyAndNothingElse(t *testing.T) {
	src := procPreamble + `
func send(a uintptr) { _ = a }

func one(x int) { send(uintptr(unsafe.Pointer(&x))) }
func two(y int) { send(uintptr(unsafe.Pointer(&y))) }
`
	got := scan(t, src, Config{Allow: map[string]string{
		"p.go:one:&x": "the fixture's reason",
	}})
	wants(t, got, "&y")
}

// An exception nobody explained is one nobody can review, so a reasonless entry
// is an error rather than a silent pass.
func TestAnAllowEntryWithoutAReasonIsRefused(t *testing.T) {
	_, err := Scan(t.TempDir(), Config{Allow: map[string]string{"k": "  "}})
	if err == nil {
		t.Fatal("Scan accepted an allow entry with no reason")
	}
	if !strings.Contains(err.Error(), "no reason") {
		t.Errorf("the error does not say what is wrong: %v", err)
	}
}

func TestFindingCarriesWhereAndWhy(t *testing.T) {
	findings, err := ScanSource("pkg/thing.go", "package p\n"+procPreamble+`
func send(a uintptr) { _ = a }

func F(x int) { send(uintptr(unsafe.Pointer(&x))) }
`, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("reported %d findings, want 1", len(findings))
	}
	f := findings[0]
	if f.File != "pkg/thing.go" || f.Func != "F" || f.Expr != "&x" {
		t.Errorf("finding does not locate the code: %+v", f)
	}
	if f.Kind != KindArgument {
		t.Errorf("Kind = %q, want %q", f.Kind, KindArgument)
	}
	if f.Key != "pkg/thing.go:F:&x" {
		t.Errorf("Key = %q, which is what an allowlist entry has to match", f.Key)
	}
	if !strings.Contains(f.Why, "runtime.Pinner") {
		t.Errorf("Why does not name the fix: %q", f.Why)
	}
	if f.Line == 0 {
		t.Error("Line is zero")
	}
}

// Nested modules and testdata are other people's code with their own audits.
func TestScanSkipsNestedModulesAndTestdata(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		t.Helper()
		if err := mkdirWrite(root, rel, src); err != nil {
			t.Fatal(err)
		}
	}
	bad := "package p\nimport \"unsafe\"\nfunc send(a uintptr) { _ = a }\nfunc F(x int) { send(uintptr(unsafe.Pointer(&x))) }\n"
	write("ok/a.go", bad)
	write("nested/go.mod", "module nested\n\ngo 1.27.1\n")
	write("nested/b.go", bad)
	write("testdata/c.go", bad)
	write(".hidden/d.go", bad)

	findings, err := Scan(root, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("reported %d findings, want only the one in ok/: %v", len(findings), findings)
	}
	if findings[0].File != "ok/a.go" {
		t.Errorf("reported %q, want ok/a.go", findings[0].File)
	}
}

func TestPartitionSeparatesNewDebtFromInherited(t *testing.T) {
	findings := []Finding{
		{Key: "a.go:F:&x", Expr: "&x"},
		{Key: "b.go:G:&y", Expr: "&y"},
	}
	fresh, inherited, stale := Partition(findings, []string{"a.go:F:&x", "gone.go:H:&z"})

	if len(fresh) != 1 || fresh[0].Key != "b.go:G:&y" {
		t.Errorf("fresh = %v, want only b.go:G:&y", fresh)
	}
	if len(inherited) != 1 || inherited[0].Key != "a.go:F:&x" {
		t.Errorf("inherited = %v, want only a.go:F:&x", inherited)
	}
	// A baseline entry describing code that has changed is a claim about
	// something that no longer exists, and has to be re-checked rather than
	// left to sit there.
	if len(stale) != 1 || stale[0] != "gone.go:H:&z" {
		t.Errorf("stale = %v, want only gone.go:H:&z", stale)
	}
}

func TestPartitionWithNoBaselineMakesEverythingFresh(t *testing.T) {
	findings := []Finding{{Key: "a.go:F:&x"}, {Key: "b.go:G:&y"}}
	fresh, inherited, stale := Partition(findings, nil)
	if len(fresh) != 2 || len(inherited) != 0 || len(stale) != 0 {
		t.Errorf("fresh=%d inherited=%d stale=%d, want 2/0/0", len(fresh), len(inherited), len(stale))
	}
}
