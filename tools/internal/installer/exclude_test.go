package installer

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/health"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func exHarness(t *testing.T) (*harness.Registry, string) {
	t.Helper()
	_, root := platHarness(t)
	write(t, filepath.Join(root, "domains", "alpha", "skills", "sk2", "SKILL.md"), "---\nname: sk2\ndescription: d\n---\nbody\n", 0o644)
	b := filepath.Join(root, "domains", "beta")
	write(t, filepath.Join(b, "manifest.yaml"), "name: beta\ndescription: d\n", 0o644)
	write(t, filepath.Join(b, "skills", "sk3", "SKILL.md"), "---\nname: sk3\ndescription: d\n---\nbody\n", 0o644)
	write(t, filepath.Join(b, "skills", "sk4", "SKILL.md"), "---\nname: sk4\ndescription: d\n---\nbody\n", 0o644)
	write(t, filepath.Join(b, "rules", "r2.md"), "# rule two\n", 0o644)
	reg, err := harness.LoadRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	return reg, root
}

func exclusions() map[string]bool {
	return map[string]bool{"alpha/sk2": true, "beta/sk3": true}
}

func hasRow(lock *lockfile.Lockfile, domain, kind string) bool {
	for _, e := range lock.Entries {
		if e.Domain == domain && e.Kind == kind {
			return true
		}
	}
	return false
}

func allPaths(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		out = append(out, p)
		return nil
	})
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var exPlatforms = []struct {
	name  string
	plats []platform.ID
	loc   string
}{
	{"claude", []platform.ID{platform.Claude}, ".claude/skills"},
	{"codex", []platform.ID{platform.Codex}, ".agents/skills"},
	{"cursor", []platform.ID{platform.Cursor}, ".agents/skills"},
	{"opencode", []platform.ID{platform.OpenCode}, ".agents/skills"},
	{"claude+cursor", []platform.ID{platform.Claude, platform.Cursor}, ".claude/skills"},
}

func TestExcludeSkillsAbsentEverywhereOthersPlaced(t *testing.T) {
	for _, mode := range []string{"link", "copy"} {
		for _, pc := range exPlatforms {
			t.Run(mode+"/"+pc.name, func(t *testing.T) {
				reg, root := exHarness(t)
				project := t.TempDir()
				lock := &lockfile.Lockfile{}
				in := newInst(root, project, mode, lock, pc.plats...)
				in.ExcludeSkills = exclusions()
				for _, d := range []string{"alpha", "beta"} {
					if err := in.InstallDomain(reg, d, true); err != nil {
						t.Fatalf("%s: %v", d, err)
					}
				}
				for _, loc := range []string{".claude/skills", ".agents/skills"} {
					for _, ex := range []string{"sk2", "sk3"} {
						if exists(filepath.Join(project, loc, ex)) {
							t.Fatalf("excluded %s placed in %s", ex, loc)
						}
					}
				}
				for _, ok := range []string{"sk", "sk4"} {
					if !exists(filepath.Join(project, pc.loc, ok)) {
						t.Fatalf("%s missing from %s", ok, pc.loc)
					}
				}
				for _, e := range lock.Entries {
					if e.Kind != "skills" {
						continue
					}
					base := filepath.Base(e.Path)
					if e.Name == "sk2" || e.Name == "sk3" || base == "sk2" || base == "sk3" {
						t.Fatalf("row for excluded skill: %+v", e)
					}
					for _, it := range e.Items {
						if it == "sk2" || it == "sk3" {
							t.Fatalf("excluded skill in row items: %+v", e)
						}
					}
					if e.Name == "" {
						t.Fatalf("kind-level skills row despite exclusions: %+v", e)
					}
				}
				if _, ok := lock.FindAt("alpha", "skills", "sk", filepath.Join(project, pc.loc, "sk")); !ok {
					t.Fatal("no per-item row for sk")
				}
				if _, ok := lock.FindAt("beta", "skills", "sk4", filepath.Join(project, pc.loc, "sk4")); !ok {
					t.Fatal("no per-item row for sk4")
				}
				kinds := []string{"rules", "agents", "hooks"}
				for _, k := range kinds {
					if k == "hooks" && len(pc.plats) == 1 && pc.plats[0] == platform.OpenCode {
						continue
					}
					if !hasRow(lock, "alpha", k) {
						t.Fatalf("alpha %s not placed", k)
					}
				}
				if !hasRow(lock, "beta", "rules") {
					t.Fatal("beta rules not placed")
				}
				if pc.plats[0] == platform.Claude {
					for _, p := range []string{"rules/r1.md", "agents/worker-bee.md", "hooks/gate.sh", "hooks/fmt.sh", "rules/r2.md"} {
						if !exists(filepath.Join(project, ".claude", p)) {
							t.Fatalf(".claude/%s missing", p)
						}
					}
				}
				for _, e := range lock.Entries {
					if h := health.EntryHealth(e); h != health.HealthOK {
						t.Fatalf("%s: %s", e.Path, h)
					}
				}
			})
		}
	}
}

func TestExcludeSkillsScopedToTheirDomain(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	in := newInst(root, project, "copy", lock, platform.Claude)
	in.ExcludeSkills = map[string]bool{"alpha/sk2": true}
	for _, d := range []string{"alpha", "beta"} {
		if err := in.InstallDomain(reg, d, true); err != nil {
			t.Fatal(err)
		}
	}
	if exists(filepath.Join(project, ".claude", "skills", "sk2")) {
		t.Fatal("alpha/sk2 placed")
	}
	for _, n := range []string{"sk", "sk3", "sk4"} {
		if !exists(filepath.Join(project, ".claude", "skills", n)) {
			t.Fatalf("%s missing", n)
		}
	}
	if _, ok := lock.Find("alpha", "skills", ""); ok {
		t.Fatal("alpha got a kind-level row despite exclusions")
	}
	if _, ok := lock.Find("beta", "skills", ""); !ok {
		t.Fatal("beta lost its kind-level row")
	}
}

func TestExcludeSkillsPartialOverFullCopyKeepsExcludedTracked(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	if err := newInst(root, project, "copy", lock, platform.Claude).InstallDomain(reg, "alpha", true); err != nil {
		t.Fatal(err)
	}
	in := newInst(root, project, "link", lock, platform.Claude)
	in.ExcludeSkills = map[string]bool{"alpha/sk2": true}
	var err error
	captureStderr(t, func() { err = in.InstallDomain(reg, "alpha", true) })
	if err != nil {
		t.Fatal(err)
	}
	sk2 := filepath.Join(project, ".claude", "skills", "sk2")
	if fi, err := os.Lstat(sk2); err != nil || fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("previously copied sk2 must stay a real dir, err=%v", err)
	}
	if fi, err := os.Lstat(filepath.Join(project, ".claude", "skills", "sk")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("sk should now be a link, err=%v", err)
	}
	parent, ok := lock.Find("alpha", "skills", "")
	if !ok || len(parent.Items) != 1 || parent.Items[0] != "sk2" {
		t.Fatalf("kind-level row not narrowed to the kept skill: %+v ok=%v", parent, ok)
	}
	for _, e := range lock.Entries {
		if h := health.EntryHealth(e); h != health.HealthOK {
			t.Fatalf("%s: %s", e.Path, h)
		}
	}
	captureStdout(t, func() error {
		return newInst(root, project, "link", lock, platform.Claude).UninstallDomain(reg, "alpha")
	})
	if exists(sk2) || exists(filepath.Join(project, ".claude", "skills", "sk")) {
		t.Fatal("remove left skills behind")
	}
}

func TestUninstallAfterPartialInstallLeavesForeignExcludedSkill(t *testing.T) {
	for _, mode := range []string{"link", "copy"} {
		for _, pc := range exPlatforms {
			t.Run(mode+"/"+pc.name, func(t *testing.T) {
				reg, root := exHarness(t)
				project := t.TempDir()
				lock := &lockfile.Lockfile{}
				foreignDir := filepath.Join(project, pc.loc, "sk2")
				write(t, filepath.Join(foreignDir, "mine.md"), "mine", 0o644)
				outside := t.TempDir()
				write(t, filepath.Join(outside, "f"), "target", 0o644)
				foreignLink := filepath.Join(project, pc.loc, "sk3")
				if err := os.Symlink(outside, foreignLink); err != nil {
					t.Fatal(err)
				}
				in := newInst(root, project, mode, lock, pc.plats...)
				in.ExcludeSkills = exclusions()
				for _, d := range []string{"alpha", "beta"} {
					if err := in.InstallDomain(reg, d, true); err != nil {
						t.Fatalf("%s: %v", d, err)
					}
				}
				un := newInst(root, project, mode, lock, pc.plats...)
				captureStdout(t, func() error {
					for _, d := range []string{"alpha", "beta"} {
						if err := un.UninstallDomain(reg, d); err != nil {
							t.Fatalf("uninstall %s: %v", d, err)
						}
					}
					return nil
				})
				if readStr(t, filepath.Join(foreignDir, "mine.md")) != "mine" {
					t.Fatal("foreign excluded dir damaged")
				}
				if tg, err := os.Readlink(foreignLink); err != nil || tg != outside {
					t.Fatalf("foreign excluded symlink damaged: %q %v", tg, err)
				}
				for _, n := range []string{"sk", "sk4"} {
					if exists(filepath.Join(project, pc.loc, n)) {
						t.Fatalf("%s survived uninstall", n)
					}
				}
				for _, e := range lock.Entries {
					if e.Domain == "alpha" || e.Domain == "beta" {
						t.Fatalf("row survived uninstall: %+v", e)
					}
				}
			})
		}
	}
}

func TestUninstallAfterPartialInstallNoNotTrackedNoiseForAbsentExcluded(t *testing.T) {
	for _, mode := range []string{"link", "copy"} {
		t.Run(mode, func(t *testing.T) {
			reg, root := exHarness(t)
			project := t.TempDir()
			lock := &lockfile.Lockfile{}
			in := newInst(root, project, mode, lock, platform.Claude)
			in.ExcludeSkills = exclusions()
			for _, d := range []string{"alpha", "beta"} {
				if err := in.InstallDomain(reg, d, true); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			out := captureStderr(t, func() {
				captureStdout(t, func() error {
					for _, d := range []string{"alpha", "beta"} {
						if e := newInst(root, project, mode, lock, platform.Claude).UninstallDomain(reg, d); e != nil {
							err = e
						}
					}
					return nil
				})
			})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out, "not tracked") {
				t.Fatalf("noise for absent excluded skills:\n%s", out)
			}
		})
	}
}

func collectAlpha(in *Installer, reg *harness.Registry) ([]ConflictError, error) {
	return in.Collect(func() error {
		if err := in.Preflight(reg, []string{"alpha"}); err != nil {
			return err
		}
		return in.InstallDomain(reg, "alpha", true)
	})
}

func TestCollectGathersConflictsWithoutWritingOrPrinting(t *testing.T) {
	cx, _ := platform.Get(platform.Codex)
	cases := []struct {
		name    string
		mode    string
		plats   []platform.ID
		foreign []string
	}{
		{"link-claude", "link", []platform.ID{platform.Claude}, []string{".claude/rules/r1.md", ".claude/hooks/gate.sh", ".claude/skills/sk"}},
		{"copy-codex", "copy", []platform.ID{platform.Claude, platform.Codex}, []string{cx.Agents.Dir + "/bee" + cx.Agents.Ext, ".agents/skills/sk"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg, root := platHarness(t)
			project := t.TempDir()
			for _, f := range tc.foreign {
				write(t, filepath.Join(project, f), "mine", 0o644)
			}
			lock := &lockfile.Lockfile{}
			in := newInst(root, project, tc.mode, lock, tc.plats...)
			before := allPaths(t, project)
			var got []ConflictError
			var err error
			var stdout string
			stderr := captureStderr(t, func() {
				stdout, _ = captureStdout(t, func() error {
					got, err = collectAlpha(in, reg)
					return nil
				})
			})
			if err != nil {
				t.Fatalf("collect: %v", err)
			}
			if stderr != "" || stdout != "" {
				t.Fatalf("collect printed:\nstderr=%q\nstdout=%q", stderr, stdout)
			}
			if len(got) < 2 {
				t.Fatalf("want >=2 conflicts, got %v", got)
			}
			paths := map[string]bool{}
			for _, c := range got {
				if c.Reason != ReasonForeign {
					t.Fatalf("unexpected reason: %+v", c)
				}
				if paths[c.Path] {
					t.Fatalf("duplicate conflict %s", c.Path)
				}
				paths[c.Path] = true
				if want := c.Path + " exists, not tracked by as-skill, use --force"; c.Error() != want {
					t.Fatalf("text = %q, want %q", c.Error(), want)
				}
			}
			for _, f := range tc.foreign {
				if !paths[filepath.Join(project, f)] {
					t.Fatalf("%s not collected; got %v", f, got)
				}
			}
			if !sameStrings(before, allPaths(t, project)) {
				t.Fatalf("collect wrote files:\nbefore=%v\nafter=%v", before, allPaths(t, project))
			}
			for _, f := range tc.foreign {
				if readStr(t, filepath.Join(project, f)) != "mine" {
					t.Fatalf("%s modified", f)
				}
			}
			if len(lock.Entries) != 0 {
				t.Fatalf("collect recorded rows: %+v", lock.Entries)
			}
			if in.DryRun || in.collecting {
				t.Fatal("collect left installer in dry/collecting state")
			}
		})
	}
}

func TestCollectRestoresOneShotNotes(t *testing.T) {
	reg, root := platHarness(t)
	project := t.TempDir()
	in := newInst(root, project, "copy", &lockfile.Lockfile{}, platform.Claude, platform.Codex)
	if got, err := collectAlpha(in, reg); err != nil || len(got) != 0 {
		t.Fatalf("collect = %v, %v", got, err)
	}
	var err error
	out := captureStderr(t, func() {
		captureStdout(t, func() error { err = in.InstallDomain(reg, "alpha", true); return nil })
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "note: codex:") {
		t.Fatalf("capability note swallowed by collect:\n%s", out)
	}
}

func TestCollectClaudeCopyWithExclusionsReportsForeignItem(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	write(t, filepath.Join(project, ".claude", "skills", "sk", "mine.md"), "mine", 0o644)
	write(t, filepath.Join(project, ".claude", "skills", "sk2", "mine.md"), "mine", 0o644)
	in := newInst(root, project, "copy", &lockfile.Lockfile{}, platform.Claude)
	in.ExcludeSkills = map[string]bool{"alpha/sk2": true}
	err := in.Preflight(reg, []string{"alpha"})
	var ce *ConflictError
	if !errors.As(err, &ce) || ce.Path != filepath.Join(project, ".claude", "skills", "sk") {
		t.Fatalf("want foreign refusal for sk only, got %v", err)
	}
	if exists(filepath.Join(project, ".claude", "rules")) {
		t.Fatal("preflight wrote")
	}
}

func TestOverwriteReplacesOnlyListedPaths(t *testing.T) {
	t.Run("link-claude", func(t *testing.T) {
		reg, root := platHarness(t)
		project := t.TempDir()
		rule := filepath.Join(project, ".claude", "rules", "r1.md")
		hook := filepath.Join(project, ".claude", "hooks", "gate.sh")
		write(t, rule, "mine-rule", 0o644)
		write(t, hook, "mine-hook", 0o644)
		lock := &lockfile.Lockfile{}
		in := newInst(root, project, "link", lock, platform.Claude)
		got, err := collectAlpha(in, reg)
		if err != nil || len(got) != 2 {
			t.Fatalf("collect = %v, %v", got, err)
		}
		in.Overwrite = map[string]bool{rule: true}
		err = in.InstallDomain(reg, "alpha", true)
		var ce *ConflictError
		if !errors.As(err, &ce) || ce.Path != hook || ce.Reason != ReasonForeign {
			t.Fatalf("want refusal for %s only, got %v", hook, err)
		}
		if in.Force {
			t.Fatal("overwrite must not set Force")
		}
		if fi, err := os.Lstat(rule); err != nil || fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("listed path not replaced: %v", err)
		}
		if readStr(t, hook) != "mine-hook" {
			t.Fatal("unlisted foreign file replaced")
		}
		in.Overwrite[hook] = true
		if err := in.InstallDomain(reg, "alpha", true); err != nil {
			t.Fatalf("after listing both: %v", err)
		}
		if fi, err := os.Lstat(hook); err != nil || fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("hook not replaced: %v", err)
		}
	})
	t.Run("copy-codex", func(t *testing.T) {
		reg, root := platHarness(t)
		project := t.TempDir()
		cx, _ := platform.Get(platform.Codex)
		agent := filepath.Join(project, cx.Agents.Dir, "bee"+cx.Agents.Ext)
		skill := filepath.Join(project, ".agents", "skills", "sk")
		write(t, agent, "mine-agent", 0o644)
		write(t, filepath.Join(skill, "mine.md"), "mine-skill", 0o644)
		in := newInst(root, project, "copy", &lockfile.Lockfile{}, platform.Codex)
		in.Overwrite = map[string]bool{agent: true}
		err := in.InstallDomain(reg, "alpha", true)
		var ce *ConflictError
		if !errors.As(err, &ce) || ce.Path != skill {
			t.Fatalf("want refusal for %s only, got %v", skill, err)
		}
		if readStr(t, agent) != "mine-agent" || readStr(t, filepath.Join(skill, "mine.md")) != "mine-skill" {
			t.Fatal("refused install touched foreign files")
		}
		in.Overwrite[skill] = true
		if err := in.InstallDomain(reg, "alpha", true); err != nil {
			t.Fatalf("after listing both: %v", err)
		}
		if readStr(t, agent) == "mine-agent" || exists(filepath.Join(skill, "mine.md")) {
			t.Fatal("listed paths not replaced")
		}
	})
}

func TestCollectGuardFailureIsNotIgnoredConflict(t *testing.T) {
	reg, root := platHarness(t)
	project := t.TempDir()
	gitInit(t, project)
	in := newInst(root, project, "link", &lockfile.Lockfile{}, platform.Claude, platform.Codex)
	before := allPaths(t, project)
	got, err := collectAlpha(in, reg)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(got) != 2 || got[0].Path != ".agents" || got[1].Path != ".codex" {
		t.Fatalf("want .agents and .codex, got %v", got)
	}
	for _, c := range got {
		if c.Reason != ReasonNotIgnored {
			t.Fatalf("reason = %q", c.Reason)
		}
		want := "refusing: " + c.Path + " is not git-ignored; add " + c.Path + " to .gitignore or use --copy (or --force)"
		if c.Error() != want {
			t.Fatalf("text = %q, want %q", c.Error(), want)
		}
	}
	if !sameStrings(before, allPaths(t, project)) {
		t.Fatal("collect wrote files")
	}

	err = in.Preflight(reg, []string{"alpha"})
	var ce *ConflictError
	if !errors.As(err, &ce) || ce.Reason != ReasonNotIgnored || !strings.Contains(err.Error(), "is not git-ignored") {
		t.Fatalf("non-collecting Preflight must still refuse, got %v", err)
	}

	in.Mode = "copy"
	got, err = collectAlpha(in, reg)
	if err != nil || len(got) != 0 {
		t.Fatalf("copy mode: %v, %v", got, err)
	}
}

func TestCollectReservedAgentNameStaysHardError(t *testing.T) {
	reg, root := platHarness(t)
	write(t, filepath.Join(root, "domains", "alpha", "agents", "explorer.md"), "---\nname: explorer\ndescription: d\n---\nx\n", 0o644)
	project := t.TempDir()
	in := newInst(root, project, "copy", &lockfile.Lockfile{}, platform.Codex)
	got, err := collectAlpha(in, reg)
	var ce *ConflictError
	if err == nil || !strings.Contains(err.Error(), "reserved") || errors.As(err, &ce) {
		t.Fatalf("want plain reserved error, got %v (conflicts %v)", err, got)
	}
	if in.DryRun {
		t.Fatal("collect left DryRun set after error")
	}
}
