package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// MainPkg is a candidate package directory found by scanning for func main
// in .go files. The list is a typing aid, not a precise package graph: false
// positives are fine; the menu is filterable.
type MainPkg struct {
	ImportPath string // unused for display when Rel is set; kept for callers
	Dir        string // absolute
	Rel        string // path relative to scan root, for display / config
}

// MullionPkg is an alias kept for call-site compatibility.
type MullionPkg = MainPkg

type pkgCache struct {
	mu       sync.Mutex
	root     string
	pkgs     []MainPkg
	ready    bool
	scanning bool
	err      error
}

var pkgsCache pkgCache

func ensureMullionPackages(root string) (pkgs []MainPkg, ready bool, err error) {
	pkgsCache.mu.Lock()
	defer pkgsCache.mu.Unlock()
	if pkgsCache.root != root {
		pkgsCache.root = root
		pkgsCache.pkgs = nil
		pkgsCache.ready = false
		pkgsCache.scanning = true
		pkgsCache.err = nil
		go scanMainPackages(root)
	}
	return append([]MainPkg(nil), pkgsCache.pkgs...), pkgsCache.ready, pkgsCache.err
}

func invalidatePackages() {
	pkgsCache.mu.Lock()
	pkgsCache.root = ""
	pkgsCache.pkgs = nil
	pkgsCache.ready = false
	pkgsCache.scanning = false
	pkgsCache.err = nil
	pkgsCache.mu.Unlock()
}

func finishPkgScan(root string, pkgs []MainPkg, err error) {
	pkgsCache.mu.Lock()
	if pkgsCache.root == root {
		pkgsCache.pkgs = pkgs
		pkgsCache.err = err
		pkgsCache.ready = true
		pkgsCache.scanning = false
	}
	pkgsCache.mu.Unlock()
	wakeUI()
}

func scanMainPackages(root string) {
	if root == "" {
		finishPkgScan(root, nil, fmt.Errorf("no scan root"))
		return
	}
	pkgs, err := listMainPackages(root)
	finishPkgScan(root, pkgs, err)
}

// listMainPackages finds directories that contain a .go file matching "func main"
// via textsearch (same walk/match path as haystack; not go list).
// listMainPackages finds the main packages under root.
//
// Upstream text-searched the tree for "func main", which also matches the
// phrase in a comment or a string literal, and misses one behind a build tag.
// A bundler that picks the wrong directory produces a broken bundle quietly,
// so this asks the toolchain instead: go list knows what a main package is,
// which files the build constraints select, and where the module ends.
//
// It also drops a dependency, which is the other half of aligning with this
// module: the search package was the only thing pulling go.hasen.dev/textsearch
// in.
func listMainPackages(root string) ([]MainPkg, error) {
	cmd := exec.Command("go", "list", "-f", "{{.Name}}\t{{.Dir}}", "./...")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("go list in %s: %s", root, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("go list in %s: %w", root, err)
	}

	var pkgs []MainPkg
	for _, line := range strings.Split(string(out), "\n") {
		name, dir, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok || name != "main" || dir == "" {
			continue
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			rel = dir
		}
		pkgs = append(pkgs, MainPkg{Dir: dir, Rel: filepath.ToSlash(rel)})
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Rel < pkgs[j].Rel })
	return pkgs, nil
}
