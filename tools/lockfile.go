package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type lockEntry struct {
	Domain      string    `json:"domain"`
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	Mode        string    `json:"mode"`
	Path        string    `json:"path"`
	Source      string    `json:"source"`
	SHA256      string    `json:"sha256,omitempty"`
	InstalledAt time.Time `json:"installed_at"`
}

type lockfile struct {
	Entries []lockEntry `json:"entries"`
}

func lockfilePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", "skills-lock.json")
}

func loadLockfile(projectRoot string) (*lockfile, error) {
	path := lockfilePath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &lockfile{}, nil
		}
		return nil, err
	}
	lf := &lockfile{}
	if err := json.Unmarshal(data, lf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return lf, nil
}

func (lf *lockfile) owns(path string) (lockEntry, bool) {
	clean := filepath.Clean(path)
	for _, e := range lf.Entries {
		if filepath.Clean(e.Path) == clean {
			return e, true
		}
	}
	return lockEntry{}, false
}

func (lf *lockfile) find(domain, kind, name string) (lockEntry, bool) {
	for _, e := range lf.Entries {
		if e.Domain == domain && e.Kind == kind && e.Name == name {
			return e, true
		}
	}
	return lockEntry{}, false
}

func (lf *lockfile) record(entry lockEntry) {
	for i, e := range lf.Entries {
		if e.Domain == entry.Domain && e.Kind == entry.Kind && e.Name == entry.Name {
			lf.Entries[i] = entry
			return
		}
	}
	lf.Entries = append(lf.Entries, entry)
}

func (lf *lockfile) remove(domain, kind, name string) {
	out := lf.Entries[:0:0]
	for _, e := range lf.Entries {
		if e.Domain == domain && e.Kind == kind && e.Name == name {
			continue
		}
		out = append(out, e)
	}
	lf.Entries = out
}

func (lf *lockfile) save(projectRoot string) error {
	dir := filepath.Join(projectRoot, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(lockfilePath(projectRoot), data, 0o644)
}

func hashTree(dir string) (string, error) {
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
