package ffiaudit

import (
	"os"
	"path/filepath"
)

// mkdirWrite writes rel under root, creating the directories it needs.
func mkdirWrite(root, rel, src string) error {
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(src), 0o600)
}
