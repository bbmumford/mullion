// Package ffiaudit finds Go memory handed to native code as an address the
// runtime cannot see.
//
// A uintptr holding the address of a Go object is invisible to the garbage
// collector and to the stack copier: nothing keeps the object alive and nothing
// stops it being moved, so the address can be stale by the time native code
// reads it. The result is corruption that depends on when a collection happens,
// which is the hardest kind of bug to find by running the program.
//
// Exactly one thing makes such a conversion safe without further work: writing
// it directly in the argument list of a call the compiler knows about --
// syscall.Syscall*, syscall.SyscallN, or Call on a *syscall.Proc,
// *syscall.LazyProc, *windows.Proc or *windows.LazyProc. Those carry
// //go:uintptrescapes, which keeps the referent alive and unmoved for the
// duration of the call (the fourth unsafe.Pointer rule). The directive is lost
// the moment the same call is reached through an interface or a func value.
//
// Nothing else in this module has it. purego's SyscallN is an ordinary Go
// function, objc's Send reaches objc_msgSend through reflection, and the
// Wayland marshal helpers and COM vtable wrappers are plain Go. Every address
// crossing those has to be pinned with runtime.Pinner, point into a
// package-level variable, or be allowed by name with a reason. That is why this
// audit matters more here than in any of the packages this module was built
// from: they each used one FFI mechanism, and this one uses several.
//
// The analysis is syntactic, so it is cheap and needs no build for a platform
// it cannot build. It therefore reads every file whatever its build tags, which
// is the point -- a Windows-only or iOS-only conversion is checked on a Mac.
//
// The idea and much of the matching comes from the pointer audit in
// gogpu/systray, widened here from one package to a whole module and from one
// FFI library to any.
package ffiaudit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Kind says which shape of hazard a finding is.
type Kind string

const (
	// KindArgument is an address converted to uintptr and passed to a call that
	// does not keep the referent alive.
	KindArgument Kind = "argument"

	// KindStored is an address converted to uintptr and written into memory
	// that outlives the statement -- a struct field, a slice or array element,
	// or a package-level variable. Native code reads such a place outside any
	// argument list, so no call can keep the referent alive for it.
	KindStored Kind = "stored"
)

// A Finding is one address the runtime cannot see.
type Finding struct {
	// File is the path relative to the scanned root, with forward slashes.
	File string

	// Line is the line the conversion is written on.
	Line int

	// Func is the function containing it.
	Func string

	// Expr is the address expression, as written.
	Expr string

	// Kind is the shape of the hazard.
	Kind Kind

	// Why says what is wrong, in a sentence that names the fix.
	Why string

	// Key identifies this finding for Config.Allow. It is "file:func:expr" for
	// an argument and "file:func:stored expr" for a store, so it survives the
	// code moving down the file.
	Key string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: %s: %s (%s)", f.File, f.Line, f.Func, f.Expr, f.Why)
}

// Config settles what a scan reads and what it forgives.
type Config struct {
	// Allow maps a Finding.Key to the reason it is safe. An entry with an empty
	// reason is refused by Scan: an exception nobody explained is one nobody
	// can review.
	Allow map[string]string

	// IncludeTests reads _test.go files too. Off by default: test code that
	// hands an address to native code is doing it under a test's own lifetime,
	// and the carried-in packages have a lot of it.
	IncludeTests bool

	// SkipDir is consulted for every directory below the root, with a path
	// relative to it. Dot-directories, testdata and nested modules are always
	// skipped.
	SkipDir func(rel string) bool
}

// Scan walks every package directory under root and returns what it finds,
// ordered by file and line.
func Scan(root string, cfg Config) ([]Finding, error) {
	for key, reason := range cfg.Allow {
		if strings.TrimSpace(reason) == "" {
			return nil, fmt.Errorf("ffiaudit: allow entry %q has no reason", key)
		}
	}
	var findings []Finding
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel != "." {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "testdata" || name == "tmp" {
				return filepath.SkipDir
			}
			// A nested module is its own code with its own audit.
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			if cfg.SkipDir != nil && cfg.SkipDir(rel) {
				return filepath.SkipDir
			}
		}
		found, err := scanDir(root, path, cfg)
		if err != nil {
			return err
		}
		findings = append(findings, found...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		return findings[i].Line < findings[j].Line
	})
	return findings, nil
}

// ScanSource analyses one file's source, for testing the analysis itself.
func ScanSource(name, src string, cfg Config) ([]Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil, err
	}
	files, paths := []*ast.File{file}, []string{name}
	return analyse(fset, files, paths, cfg), nil
}

func scanDir(root, dir string, cfg Config) ([]Finding, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	var files []*ast.File
	var paths []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		if !cfg.IncludeTests && strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
		paths = append(paths, filepath.ToSlash(rel))
	}
	if len(files) == 0 {
		return nil, nil
	}
	return analyse(fset, files, paths, cfg), nil
}

func analyse(fset *token.FileSet, files []*ast.File, paths []string, cfg Config) []Finding {
	procs := escapingProcs(fset, files)
	globals := packageVars(files)
	var findings []Finding
	for i, f := range files {
		findings = append(findings, fileFindings(fset, f, paths[i], procs, globals, cfg)...)
	}
	return findings
}

// fileFindings walks one file, including the bodies of function literals.
//
// Visiting only top-level function declarations would miss a whole shape of
// code: FFI bound inside a package-level literal, such as
// `var load = sync.OnceValue(func() ...)`, which is how a library resolves its
// symbols once. Those bodies hold some of the most pointer-heavy code there is,
// and they are not reachable from a scan of FuncDecls.
func fileFindings(fset *token.FileSet, file *ast.File, rel string, procs map[string]bool, globals map[string]*ast.ValueSpec, cfg Config) []Finding {
	var findings []Finding
	var (
		stack []ast.Node
		names []string // enclosing function names, outermost first
		pins  []pinSet // one per enclosing function body
	)
	// A pin in an enclosing function is still in effect inside a literal that
	// runs before its Unpin, so a conversion is covered by any of them.
	covers := func(expr string, operand ast.Expr, offset int) bool {
		for _, p := range pins {
			if p.covers(expr, operand, offset) {
				return true
			}
		}
		return false
	}
	funcName := func() string {
		if len(names) == 0 {
			return "<file>"
		}
		return strings.Join(names, ".")
	}
	// A package-level literal is named after the variable holding it, so a
	// finding inside one can be found again.
	varName := func(n ast.Node) string {
		if vs, ok := n.(*ast.ValueSpec); ok && len(vs.Names) > 0 {
			return vs.Names[0].Name
		}
		return ""
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			switch last.(type) {
			case *ast.FuncDecl, *ast.FuncLit:
				names = names[:len(names)-1]
				pins = pins[:len(pins)-1]
			case *ast.ValueSpec:
				if varName(last) != "" {
					names = names[:len(names)-1]
				}
			}
			return true
		}
		stack = append(stack, n)

		switch v := n.(type) {
		case *ast.FuncDecl:
			names = append(names, v.Name.Name)
			pins = append(pins, pinsInBody(fset, v.Body))
			return true
		case *ast.FuncLit:
			names = append(names, "func")
			pins = append(pins, pinsInBody(fset, v.Body))
			return true
		case *ast.ValueSpec:
			if name := varName(v); name != "" {
				names = append(names, name)
			}
			return true
		}
		if len(pins) == 0 {
			// Not inside any function body: a type or a constant.
			return true
		}
		{

			// A conversion written into somewhere that outlives the statement.
			// Checked before the argument case because the store is the hazard
			// whatever call it sits in.
			if operand, ok := pointerToUintptr(n); ok {
				if where, stored := storedBeyondTheCall(stack, globals); stored {
					expr := exprString(fset, operand)
					at := fset.Position(n.Pos())
					f := Finding{
						File: rel, Line: at.Line, Func: funcName(),
						Expr: expr, Kind: KindStored,
						Why: "the address is written into " + where +
							", which native code reads outside any argument list, so no call keeps the referent alive: pin it for as long as native code holds it",
						Key: rel + ":" + funcName() + ":stored " + expr,
					}
					if _, allowed := cfg.Allow[f.Key]; !allowed && !covers(expr, operand, at.Offset) {
						findings = append(findings, f)
					}
					return true
				}
			}

			addr, ok := addressToUintptr(n)
			if !ok {
				return true
			}
			if call, ok := stack[len(stack)-2].(*ast.CallExpr); ok && keepsReferentAlive(call, procs) {
				return true
			}
			if addressOfPackageVar(addr, globals) {
				return true
			}
			expr := exprString(fset, addr)
			at := fset.Position(n.Pos())
			f := Finding{
				File: rel, Line: at.Line, Func: funcName(),
				Expr: expr, Kind: KindArgument,
				Why: "this call does not carry //go:uintptrescapes, so the referent is neither kept alive nor held still: pin it with runtime.Pinner before the call",
				Key: rel + ":" + funcName() + ":" + expr,
			}
			if _, allowed := cfg.Allow[f.Key]; !allowed && !covers(expr, addr, at.Offset) {
				findings = append(findings, f)
			}
		}
		return true
	})
	return findings
}

// pinSet records what runtime.Pinner was asked to pin, and where.
//
// Both halves matter. A pin written after the conversion protects nothing, so
// position is kept and compared. And pinning an object pins all of it, so a
// pin of the root covers an address into a field or an element of it -- without
// that, correct code such as Pin(&s) followed by &s.field would be reported.
type pinSet struct {
	exact map[string]int // expression as written -> offset of the earliest pin
	roots map[string]int // root identifier -> offset of the earliest pin
}

func (p pinSet) covers(expr string, operand ast.Expr, offset int) bool {
	if at, ok := p.exact[expr]; ok && at < offset {
		return true
	}
	if root := rootIdent(operand); root != "" {
		if at, ok := p.roots[root]; ok && at < offset {
			return true
		}
	}
	return false
}

// pinsInBody collects the argument of every X.Pin(arg) call, by position.
//
// A nil body is a declaration whose implementation is elsewhere -- assembly, or
// a linkname -- and has nothing to collect. It still gets an empty set, because
// the caller keeps one per enclosing function and the stack has to balance.
func pinsInBody(fset *token.FileSet, body *ast.BlockStmt) pinSet {
	pins := pinSet{exact: map[string]int{}, roots: map[string]int{}}
	if body == nil {
		return pins
	}
	note := func(m map[string]int, key string, at int) {
		if key == "" {
			return
		}
		if prev, ok := m[key]; !ok || at < prev {
			m[key] = at
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Pin" {
			return true
		}
		at := fset.Position(call.Pos()).Offset
		arg := call.Args[0]
		note(pins.exact, exprString(fset, arg), at)
		note(pins.roots, rootIdent(arg), at)
		return true
	})
	return pins
}

// rootIdent walks &x.f, x.f, &x and so on down to the identifier the memory
// belongs to, and returns "" if it cannot get there.
//
// It stops at an index. &s[0] on a slice points into a backing array that is a
// different object from the slice header s, so pinning s would not pin it;
// pinning an array a does cover &a[0], but telling a slice from an array needs
// type information this analysis does not have. Refusing to walk through an
// index keeps it from accepting a pin that protects the wrong object, at the
// cost of asking for &a[0] to be pinned as written.
func rootIdent(e ast.Expr) string {
	for {
		switch v := e.(type) {
		case *ast.UnaryExpr:
			if v.Op != token.AND {
				return ""
			}
			e = v.X
		case *ast.IndexExpr:
			return ""
		case *ast.SelectorExpr:
			e = v.X
		case *ast.ParenExpr:
			e = v.X
		case *ast.StarExpr:
			e = v.X
		case *ast.Ident:
			return v.Name
		default:
			return ""
		}
	}
}

// storedBeyondTheCall reports whether the innermost node of stack is written
// somewhere that outlives the statement, and names the place.
func storedBeyondTheCall(stack []ast.Node, globals map[string]*ast.ValueSpec) (string, bool) {
	if len(stack) < 3 {
		return "", false
	}
	n := stack[len(stack)-1]
	switch parent := stack[len(stack)-2].(type) {
	case *ast.KeyValueExpr:
		if _, inLiteral := stack[len(stack)-3].(*ast.CompositeLit); inLiteral && parent.Value == n {
			return "a field of a composite literal", true
		}
	case *ast.CompositeLit:
		for _, elt := range parent.Elts {
			if elt == n {
				return "an element of a composite literal", true
			}
		}
	case *ast.AssignStmt:
		for i, rhs := range parent.Rhs {
			if rhs != n || i >= len(parent.Lhs) {
				continue
			}
			switch lhs := parent.Lhs[i].(type) {
			case *ast.SelectorExpr:
				return "a struct field", true
			case *ast.IndexExpr:
				return "a slice, array or map element", true
			case *ast.StarExpr:
				return "memory through a pointer", true
			case *ast.Ident:
				// A package-level variable outlives every call made with it.
				if globals[lhs.Name] != nil && (lhs.Obj == nil || globals[lhs.Name] == declOf(lhs)) {
					return "a package-level variable", true
				}
			}
		}
	}
	return "", false
}

func declOf(id *ast.Ident) *ast.ValueSpec {
	if id.Obj == nil {
		return nil
	}
	spec, _ := id.Obj.Decl.(*ast.ValueSpec)
	return spec
}

// pointerToUintptr matches uintptr(unsafe.Pointer(x)) and returns x.
func pointerToUintptr(n ast.Node) (ast.Expr, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil, false
	}
	if id, ok := call.Fun.(*ast.Ident); !ok || id.Name != "uintptr" {
		return nil, false
	}
	inner, ok := call.Args[0].(*ast.CallExpr)
	if !ok || len(inner.Args) != 1 {
		return nil, false
	}
	sel, ok := inner.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Pointer" {
		return nil, false
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "unsafe" {
		return nil, false
	}
	return inner.Args[0], true
}

// addressToUintptr matches uintptr(unsafe.Pointer(&x...)) and returns &x....
//
// A pointer variable converted to uintptr is not matched: where it points is
// not visible here, and reporting every such conversion would bury the ones
// that are plainly an address of Go memory. The store case above catches the
// dangerous half of what this misses.
func addressToUintptr(n ast.Node) (ast.Expr, bool) {
	operand, ok := pointerToUintptr(n)
	if !ok {
		return nil, false
	}
	addr, ok := operand.(*ast.UnaryExpr)
	if !ok || addr.Op != token.AND {
		return nil, false
	}
	return addr, true
}

// keepsReferentAlive reports whether call is one of the few whose uintptr
// arguments the compiler keeps alive and unmoved.
//
// The set is short, and what is absent from it is the point. objc.Send and
// objc.ID.Send reach objc_msgSend through a func value built by
// purego.RegisterFunc, every func value produced by purego.RegisterLibFunc is
// an ordinary Go call, and goffi's CallFunction is Go code: none of them carries
// //go:uintptrescapes.
//
// What the set does include is any function annotated with that directive and
// called directly, which is syscall.Syscall* and -- this is easy to get wrong
// -- purego.SyscallN, which carries it too (purego/syscall.go). The directive
// only reaches a conversion written in the argument list of the annotated call
// itself, so a wrapper of any of these loses it; that is why the check looks at
// the immediate parent call and no further.
func keepsReferentAlive(call *ast.CallExpr, procs map[string]bool) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	// syscall.Syscall, Syscall6, Syscall9, SyscallN, and purego.SyscallN.
	if pkg, ok := sel.X.(*ast.Ident); ok {
		if pkg.Name == "syscall" && strings.HasPrefix(sel.Sel.Name, "Syscall") {
			return true
		}
		if pkg.Name == "purego" && sel.Sel.Name == "SyscallN" {
			return true
		}
	}
	if sel.Sel.Name != "Call" {
		return false
	}
	// Only a direct call on the concrete proc type keeps the directive; reached
	// through an interface or a func value it is gone, so the receiver has to
	// be a name this package declared as a proc.
	switch recv := sel.X.(type) {
	case *ast.Ident:
		return procs[recv.Name]
	case *ast.SelectorExpr:
		return procs[recv.Sel.Name]
	}
	return false
}

// escapingProcs returns the names holding a *Proc or *LazyProc of the syscall
// package or of golang.org/x/sys/windows, whose Call methods carry
// //go:uintptrescapes.
func escapingProcs(fset *token.FileSet, files []*ast.File) map[string]bool {
	isType := func(e ast.Expr, names ...string) bool {
		s := exprString(fset, e)
		for _, n := range names {
			if s == n {
				return true
			}
		}
		return false
	}
	dlls := map[string]bool{}
	procs := map[string]bool{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			var names []*ast.Ident
			var typ ast.Expr
			switch v := n.(type) {
			case *ast.ValueSpec:
				names, typ = v.Names, v.Type
			case *ast.Field:
				names, typ = v.Names, v.Type
			default:
				return true
			}
			if typ == nil {
				return true
			}
			for _, id := range names {
				switch {
				case isType(typ, "*syscall.Proc", "*syscall.LazyProc", "*windows.Proc", "*windows.LazyProc"):
					procs[id.Name] = true
				case isType(typ, "*syscall.DLL", "*syscall.LazyDLL", "*windows.DLL", "*windows.LazyDLL"):
					dlls[id.Name] = true
				}
			}
			return true
		})
	}
	record := func(name string, value ast.Expr) {
		call, ok := value.(*ast.CallExpr)
		if !ok {
			return
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok {
			return
		}
		loader := sel.Sel.Name == "NewLazyDLL" || sel.Sel.Name == "NewLazySystemDLL" ||
			sel.Sel.Name == "LoadDLL" || sel.Sel.Name == "MustLoadDLL"
		switch {
		case (x.Name == "syscall" || x.Name == "windows") && loader:
			dlls[name] = true
		case dlls[x.Name] && (sel.Sel.Name == "NewProc" || sel.Sel.Name == "FindProc" || sel.Sel.Name == "MustFindProc"):
			procs[name] = true
		}
	}
	// Twice: a proc may be declared above the DLL it comes from.
	for range 2 {
		for _, f := range files {
			ast.Inspect(f, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.ValueSpec:
					for i := range v.Values {
						if i < len(v.Names) {
							record(v.Names[i].Name, v.Values[i])
						}
					}
				case *ast.AssignStmt:
					if len(v.Rhs) == 1 && len(v.Lhs) >= 1 {
						if id, ok := v.Lhs[0].(*ast.Ident); ok {
							record(id.Name, v.Rhs[0])
						}
					}
				}
				return true
			})
		}
	}
	return procs
}

// packageVars returns the package-level variable declarations, by name. The
// parser resolves an identifier only within one file, so a package-level
// variable declared in another file of the same package arrives unresolved.
func packageVars(files []*ast.File) map[string]*ast.ValueSpec {
	vars := map[string]*ast.ValueSpec{}
	for _, f := range files {
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, id := range vs.Names {
					vars[id.Name] = vs
				}
			}
		}
	}
	return vars
}

// addressOfPackageVar reports whether addr points into a package-level
// variable. Such storage is static, or heap memory the variable keeps
// reachable: the stack copier never moves it and the collector keeps it, so the
// address stays good for as long as native code holds it.
func addressOfPackageVar(addr ast.Expr, globals map[string]*ast.ValueSpec) bool {
	unary, ok := addr.(*ast.UnaryExpr)
	if !ok {
		return false
	}
	e := unary.X
	for {
		switch v := e.(type) {
		case *ast.IndexExpr:
			e = v.X
		case *ast.SelectorExpr:
			e = v.X
		case *ast.ParenExpr:
			e = v.X
		case *ast.Ident:
			if v.Obj != nil {
				// Resolved here: a package-level variable only if this is the
				// top-level declaration and not a local shadowing its name.
				spec, ok := v.Obj.Decl.(*ast.ValueSpec)
				return ok && globals[v.Name] == spec
			}
			return globals[v.Name] != nil
		default:
			return false
		}
	}
}

func exprString(fset *token.FileSet, e ast.Expr) string {
	var b strings.Builder
	_ = printer.Fprint(&b, fset, e)
	return b.String()
}

// Partition splits findings against a baseline of keys.
//
// A module that carries code in from elsewhere starts with findings it did not
// write, and on a tree this size there are too many to review at once. Failing
// on all of them makes the audit red from the first run, and a test that is
// always red gets deleted rather than read. Listing them instead keeps the
// audit sharp for new code while the inherited ones are worked through.
//
// fresh is what no baseline entry covers: new debt, which should fail. stale
// is baseline entries that match nothing any more -- the code they described
// has changed, so the entry is a claim about code that no longer exists and
// should be removed, which is what stops the list outliving its subject.
func Partition(findings []Finding, baseline []string) (fresh, inherited []Finding, stale []string) {
	known := make(map[string]bool, len(baseline))
	seen := make(map[string]bool, len(baseline))
	for _, k := range baseline {
		known[k] = true
	}
	for _, f := range findings {
		if known[f.Key] {
			seen[f.Key] = true
			inherited = append(inherited, f)
			continue
		}
		fresh = append(fresh, f)
	}
	for _, k := range baseline {
		if !seen[k] {
			stale = append(stale, k)
		}
	}
	sort.Strings(stale)
	return fresh, inherited, stale
}
