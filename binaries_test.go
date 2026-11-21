package mullion_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoCompiledBinaryIsTracked fails if a build artefact has been committed.
//
// Building an example writes its binary into the directory the command runs
// from, which is this one, not next to the source. Three such binaries were
// committed that way and had to be taken out of the history afterwards; a
// fourth was caught only because someone looked at git status at the right
// moment. The ignore rules list them by name, which needs a line per example
// and quietly misses the next one.
//
// This checks what is actually tracked, so it holds whatever the ignore rules
// say and whatever an example is called.
func TestNoCompiledBinaryIsTracked(t *testing.T) {
	out, err := exec.Command("git", "ls-files", "-z").Output()
	if err != nil {
		t.Skipf("git is not available here: %v", err)
	}
	for _, name := range strings.Split(string(out), "\x00") {
		if name == "" {
			continue
		}
		info, err := os.Lstat(name)
		if err != nil || !info.Mode().IsRegular() {
			continue // deleted, or a symlink: not a committed binary
		}
		if kind := executableKind(t, name); kind != "" {
			t.Errorf("%s is a compiled %s binary (%d bytes) and is tracked;\n"+
				"\tbuild artefacts belong outside the repository, and `go build ./examples/<name>/`\n"+
				"\twrites one here unless -o sends it elsewhere",
				name, kind, info.Size())
		}
	}
}

// executableKind reads the magic number and names the format, or "" for
// anything that is not an executable.
func executableKind(t *testing.T, name string) string {
	t.Helper()
	f, err := os.Open(filepath.Clean(name))
	if err != nil {
		return ""
	}
	defer f.Close()

	var magic [4]byte
	if n, _ := f.Read(magic[:]); n < 4 {
		return ""
	}
	switch {
	case magic == [4]byte{0x7f, 'E', 'L', 'F'}:
		return "ELF"
	case magic == [4]byte{0xcf, 0xfa, 0xed, 0xfe}, magic == [4]byte{0xce, 0xfa, 0xed, 0xfe},
		magic == [4]byte{0xfe, 0xed, 0xfa, 0xcf}, magic == [4]byte{0xfe, 0xed, 0xfa, 0xce}:
		return "Mach-O"
	case magic[0] == 0xca && magic[1] == 0xfe && magic[2] == 0xba && magic[3] == 0xbe:
		return "Mach-O universal"
	case magic[0] == 'M' && magic[1] == 'Z':
		return "PE"
	}
	return ""
}
