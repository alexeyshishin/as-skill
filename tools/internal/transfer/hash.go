package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

func HashTree(dir string) (string, error) {
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
