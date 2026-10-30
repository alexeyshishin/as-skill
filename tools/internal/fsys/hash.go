package fsys

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claude-harness/tools/internal/lockfile"
)

func TopNames(src string, kind lockfile.Kind) ([]string, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !SkipTop(kind, e.Name()) {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func HashTree(dir string) (string, error) {
	return hashTree(dir, nil)
}

func HashNames(dir string, names []string) (string, error) {
	keep := make(map[string]bool, len(names))
	for _, n := range names {
		keep[n] = true
	}
	return hashTree(dir, keep)
}

func hashTree(dir string, keep map[string]bool) (string, error) {
	var rels []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if keep != nil {
			top, _, _ := strings.Cut(rel, string(filepath.Separator))
			if !keep[top] {
				return nil
			}
		}
		rels = append(rels, rel)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(rels)

	h := sha256.New()
	for _, rel := range rels {
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return "", err
		}
		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write(data)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func HashPath(p string) (string, error) {
	fi, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return HashTree(p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return HashBytes(data), nil
}

func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
