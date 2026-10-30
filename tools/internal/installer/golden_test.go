package installer

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
)

func snapshotInstall(t *testing.T, mode string) string {
	t.Helper()
	hRoot, err := harness.ResolveHarnessRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := harness.LoadRegistry(hRoot)
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	in := &Installer{HarnessRoot: hRoot, ProjectRoot: project, Mode: lockfile.Mode(mode), Lock: lock}
	for _, d := range []string{"code", "git", "core"} {
		if err := in.InstallDomain(reg, d, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := lock.Save(project); err != nil {
		t.Fatal(err)
	}

	rel := func(p, base string) string {
		r, err := filepath.Rel(base, p)
		if err != nil || strings.HasPrefix(r, "..") {
			return p
		}
		return r
	}
	var lines []string
	err = filepath.WalkDir(project, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		r := rel(p, project)
		if r == "." || r == filepath.Join(".claude", "skills-lock.json") {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			lines = append(lines, "L "+r+" -> "+rel(target, hRoot))
			return nil
		}
		if d.IsDir() {
			lines = append(lines, "D "+r)
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		lines = append(lines, "F "+r+" "+info.Mode().Perm().String())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)

	raw, err := os.ReadFile(filepath.Join(project, ".claude", "skills-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	lf := &lockfile.Lockfile{}
	if err := json.Unmarshal(raw, lf); err != nil {
		t.Fatal(err)
	}
	type normEntry struct {
		Domain, Kind, Name, Mode, Path, Source string
		HasSHA                                 bool
		Platforms                              []string
		SrcSHA256                              string
	}
	var entries []normEntry
	for _, e := range lf.Entries {
		entries = append(entries, normEntry{e.Domain, string(e.Kind), e.Name, string(e.Mode), rel(e.Path, project), rel(e.Source, hRoot), e.SHA256 != "", e.Platforms, e.SrcSHA256})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	lockJSON, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(lines, "\n") + "\n--- lockfile ---\n" + string(lockJSON) + "\n"
}

func TestDefaultInstallGolden(t *testing.T) {
	for _, mode := range []string{"link", "copy"} {
		t.Run(mode, func(t *testing.T) {
			got := snapshotInstall(t, mode)
			path := filepath.Join("testdata", "golden", mode+".txt")
			if os.Getenv("UPDATE_GOLDEN") != "" {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("install snapshot differs from %s (UPDATE_GOLDEN=1 to regenerate)\n--- got ---\n%s", path, got)
			}
		})
	}
}
