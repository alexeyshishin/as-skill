package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func TestInsideRoots(t *testing.T) {
	proj := t.TempDir()
	out := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(out, filepath.Join(proj, ".agents", "skills")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for name, p := range map[string]string{
		"outside":   out,
		"root":      filepath.Join(proj, ".claude"),
		"traversal": filepath.Join(proj, ".claude", "..", "..", "etc"),
		"relative":  ".claude/skills/x",
		"symlink":   filepath.Join(proj, ".agents", "skills", "x"),
	} {
		if InsideRoots(proj, p) == nil {
			t.Errorf("%s: want refusal for %s", name, p)
		}
	}
	for _, p := range []string{filepath.Join(proj, ".claude", "skills", "x"), filepath.Join(proj, ".agents", "rules", "r.md")} {
		if err := InsideRoots(proj, p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

func TestUninstallRefusesOutsideRows(t *testing.T) {
	for _, plat := range []platform.ID{platform.Claude, platform.Codex} {
		for _, kind := range []struct{ kind, mode string }{{"skill", "copy"}, {"skill", "link"}, {"rules", "copy"}, {"agents", "render"}, {"hooks", "copy"}} {
			for _, variant := range []string{"absolute", "traversal", "symlink"} {
				for _, viaSkill := range []bool{false, true} {
					t.Run(string(plat)+"/"+kind.kind+"-"+kind.mode+"/"+variant, func(t *testing.T) {
						in, reg, project := rerunSetup(t, "copy")
						in.Platforms = []platform.ID{plat}
						victim := filepath.Join(t.TempDir(), "victim")
						if err := os.MkdirAll(victim, 0o755); err != nil {
							t.Fatal(err)
						}
						keep := filepath.Join(victim, "keep.txt")
						if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
							t.Fatal(err)
						}
						var path string
						switch variant {
						case "absolute":
							path = victim
						case "traversal":
							rel, _ := filepath.Rel(filepath.Join(project, ".claude", "skills"), victim)
							path = filepath.Join(project, ".claude", "skills", rel)
						case "symlink":
							if err := os.MkdirAll(filepath.Join(project, ".agents"), 0o755); err != nil {
								t.Fatal(err)
							}
							if err := os.Symlink(filepath.Dir(victim), filepath.Join(project, ".agents", "skills")); err != nil {
								t.Fatal(err)
							}
							path = filepath.Join(project, ".agents", "skills", "victim")
						}
						name := "git-conventional-commits"
						if kind.kind != "skill" {
							name = "git-conventions"
						}
						in.Lock.Record(lockfile.Entry{Domain: "git", Kind: lockfile.Kind(kind.kind), Name: name, Mode: lockfile.Mode(kind.mode), Path: path, Platforms: []string{string(plat)}})
						var err error
						if viaSkill && kind.kind == "skill" {
							err = in.UninstallSkill(reg, name)
						} else {
							err = in.UninstallDomain(reg, "git")
						}
						if err == nil || !strings.Contains(err.Error(), "refused") {
							t.Fatalf("want refusal, got %v", err)
						}
						if _, serr := os.Stat(keep); serr != nil {
							t.Fatalf("victim damaged: %v", serr)
						}
						if _, ok := in.Lock.FindAt("git", lockfile.Kind(kind.kind), name, path); !ok {
							t.Fatal("row must be kept")
						}
					})
				}
			}
		}
	}
}

func TestUninstallRefusesOutsideItems(t *testing.T) {
	in, reg, project := rerunSetup(t, "copy")
	in.Platforms = []platform.ID{platform.Codex}
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.MkdirAll(victim, 0o755); err != nil {
		t.Fatal(err)
	}
	in.Lock.Record(lockfile.Entry{Domain: "git", Kind: "skills", Mode: "copy", Path: filepath.Join(project, ".agents", "skills"), Items: []string{"../../../../.." + victim}, Platforms: []string{"codex"}})
	if err := in.UninstallDomain(reg, "git"); err == nil {
		t.Fatal("want refusal")
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatal(err)
	}
}

func TestUninstallRefusesSymlinkedPlatformRoot(t *testing.T) {
	for _, root := range []string{".claude", ".agents"} {
		for _, viaSkill := range []bool{false, true} {
			t.Run(root, func(t *testing.T) {
				in, reg, project := rerunSetup(t, "copy")
				out := t.TempDir()
				if err := os.Symlink(out, filepath.Join(project, root)); err != nil {
					t.Fatal(err)
				}
				keep := filepath.Join(out, "skills", "victim", "keep.txt")
				if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(project, root, "skills", "victim")
				name := "git-conventional-commits"
				in.Lock.Record(lockfile.Entry{Domain: "git", Kind: "skill", Name: name, Mode: "copy", Path: path, Platforms: []string{"claude"}})
				var err error
				if viaSkill {
					err = in.UninstallSkill(reg, name)
				} else {
					err = in.UninstallDomain(reg, "git")
				}
				if err == nil || !strings.Contains(err.Error(), "refused") {
					t.Fatalf("want refusal, got %v", err)
				}
				if _, serr := os.Stat(keep); serr != nil {
					t.Fatalf("victim damaged: %v", serr)
				}
				if _, ok := in.Lock.FindAt("git", "skill", name, path); !ok {
					t.Fatal("row must be kept")
				}
			})
		}
	}
}

func TestInsideRootsThroughSymlinkedProject(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "proj")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(real, ".claude", "skills", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, proj := range []string{link, real} {
		if err := InsideRoots(proj, filepath.Join(proj, ".claude", "skills", "x")); err != nil {
			t.Errorf("%s: %v", proj, err)
		}
		if err := InsideRoots(proj, filepath.Join(proj, ".claude", "skills", "new")); err != nil {
			t.Errorf("%s nonexistent: %v", proj, err)
		}
	}
}
