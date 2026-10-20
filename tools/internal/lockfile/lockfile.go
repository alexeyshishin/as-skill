package lockfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Entry struct {
	Domain      string    `json:"domain"`
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	Mode        string    `json:"mode"`
	Path        string    `json:"path"`
	Source      string    `json:"source"`
	SHA256      string    `json:"sha256,omitempty"`
	InstalledAt time.Time `json:"installed_at"`
}

type Lockfile struct {
	Entries []Entry `json:"entries"`
}

func path(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", "skills-lock.json")
}

func Load(projectRoot string) (*Lockfile, error) {
	p := path(projectRoot)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Lockfile{}, nil
		}
		return nil, err
	}
	lf := &Lockfile{}
	if err := json.Unmarshal(data, lf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", p, err)
	}
	migrateCoreSkillKind(lf)
	return lf, nil
}

func migrateCoreSkillKind(lf *Lockfile) {
	for i, e := range lf.Entries {
		if e.Domain == "core" && e.Kind == "skill" {
			lf.Entries[i].Kind = "skills"
		}
	}
}

func (lf *Lockfile) Owns(target string) (Entry, bool) {
	clean := filepath.Clean(target)
	for _, e := range lf.Entries {
		if filepath.Clean(e.Path) == clean {
			return e, true
		}
	}
	return Entry{}, false
}

func (lf *Lockfile) Find(domain, kind, name string) (Entry, bool) {
	for _, e := range lf.Entries {
		if e.Domain == domain && e.Kind == kind && e.Name == name {
			return e, true
		}
	}
	return Entry{}, false
}

func (lf *Lockfile) Record(entry Entry) {
	for i, e := range lf.Entries {
		if e.Domain == entry.Domain && e.Kind == entry.Kind && e.Name == entry.Name {
			lf.Entries[i] = entry
			return
		}
	}
	lf.Entries = append(lf.Entries, entry)
}

func (lf *Lockfile) Remove(domain, kind, name string) {
	out := lf.Entries[:0:0]
	for _, e := range lf.Entries {
		if e.Domain == domain && e.Kind == kind && e.Name == name {
			continue
		}
		out = append(out, e)
	}
	lf.Entries = out
}

func (lf *Lockfile) Save(projectRoot string) error {
	dir := filepath.Join(projectRoot, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path(projectRoot), data, 0o644)
}
