package fsys

import (
	"path/filepath"
	"strings"
)

// Within reports whether p is root itself or lies under root. Both paths are
// cleaned first; the check is lexical (no symlink resolution). A relative/absolute
// mismatch (filepath.Rel error) yields false. A child whose name merely starts
// with ".." (e.g. "..foo") is inside.
func Within(root, p string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(p))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// StrictlyInside is Within but false when p equals root.
func StrictlyInside(root, p string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(p))
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
