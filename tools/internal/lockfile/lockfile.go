package lockfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	ModeLink   = "link"
	ModeCopy   = "copy"
	ModeRender = "render"

	DefaultPlatform = "claude"
)

type Entry struct {
	Domain      string    `json:"domain"`
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	Mode        string    `json:"mode"`
	Path        string    `json:"path"`
	Source      string    `json:"source"`
	SHA256      string    `json:"sha256,omitempty"`
	Platforms   []string  `json:"platforms,omitempty"`
	SrcSHA256   string    `json:"src_sha256,omitempty"`
	Items       []string  `json:"items,omitempty"`
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
	migrateDefaultPlatform(lf)
	return lf, nil
}

func migrateDefaultPlatform(lf *Lockfile) {
	for i, e := range lf.Entries {
		if len(e.Platforms) == 0 {
			lf.Entries[i].Platforms = []string{DefaultPlatform}
		}
	}
}

func claudeOnly(p []string) bool {
	return len(p) == 1 && p[0] == DefaultPlatform
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

func (lf *Lockfile) Covering(dst string) (Entry, bool) {
	if e, ok := lf.Owns(dst); ok {
		return e, true
	}
	clean := filepath.Clean(dst)
	dir := filepath.Dir(clean)
	base := filepath.Base(clean)
	for _, e := range lf.Entries {
		if e.Name != "" || filepath.Clean(e.Path) != dir {
			continue
		}
		if len(e.Items) > 0 {
			for _, n := range e.Items {
				if n == base {
					return e, true
				}
			}
			continue
		}
		if e.Source == "" {
			continue
		}
		if _, err := os.Lstat(filepath.Join(e.Source, base)); err == nil {
			return e, true
		}
	}
	return Entry{}, false
}

// CoveredPaths lists the concrete paths a kind-level row placed. Rows with
// Items need no Source; legacy rows without Items fall back to listing Source
// (every top-level entry, SkipTop not applied). Named rows return their own Path.
func (e Entry) CoveredPaths() []string {
	if e.Name != "" {
		return []string{e.Path}
	}
	if len(e.Items) > 0 {
		out := make([]string, 0, len(e.Items))
		for _, n := range e.Items {
			out = append(out, filepath.Join(e.Path, n))
		}
		return out
	}
	if e.Source == "" {
		return nil
	}
	ents, err := os.ReadDir(e.Source)
	if err != nil {
		return nil
	}
	var out []string
	for _, d := range ents {
		out = append(out, filepath.Join(e.Path, d.Name()))
	}
	return out
}

func (lf *Lockfile) Covers(dst string) bool {
	_, ok := lf.Covering(dst)
	return ok
}

func (lf *Lockfile) RemoveAllAt(path string) {
	clean := filepath.Clean(path)
	out := lf.Entries[:0:0]
	for _, e := range lf.Entries {
		if filepath.Clean(e.Path) == clean {
			continue
		}
		out = append(out, e)
	}
	lf.Entries = out
}

func (lf *Lockfile) Find(domain, kind, name string) (Entry, bool) {
	for _, e := range lf.Entries {
		if e.Domain == domain && e.Kind == kind && e.Name == name {
			return e, true
		}
	}
	return Entry{}, false
}

func (lf *Lockfile) FindAt(domain, kind, name, path string) (Entry, bool) {
	clean := filepath.Clean(path)
	for _, e := range lf.Entries {
		if e.Domain == domain && e.Kind == kind && e.Name == name && filepath.Clean(e.Path) == clean {
			return e, true
		}
	}
	return Entry{}, false
}

func (lf *Lockfile) Record(entry Entry) {
	if len(entry.Platforms) == 0 {
		entry.Platforms = []string{DefaultPlatform}
	}
	clean := filepath.Clean(entry.Path)
	for i, e := range lf.Entries {
		if e.Domain == entry.Domain && e.Kind == entry.Kind && e.Name == entry.Name && filepath.Clean(e.Path) == clean {
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

func (lf *Lockfile) RemoveAt(domain, kind, name, path string) {
	clean := filepath.Clean(path)
	out := lf.Entries[:0:0]
	for _, e := range lf.Entries {
		if e.Domain == domain && e.Kind == kind && e.Name == name && filepath.Clean(e.Path) == clean {
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
	out := &Lockfile{Entries: make([]Entry, len(lf.Entries))}
	copy(out.Entries, lf.Entries)
	for i, e := range out.Entries {
		if claudeOnly(e.Platforms) {
			out.Entries[i].Platforms = nil
		}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path(projectRoot), data, 0o644)
}
