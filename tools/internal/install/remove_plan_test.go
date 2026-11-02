package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/iostreams"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func rmProject(t *testing.T) string {
	t.Helper()
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(proj, ".claude", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	return proj
}

func rmWrite(t *testing.T, p, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func rmExists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func rmInstaller(proj string, rows []lockfile.Entry) *Installer {
	return &Installer{
		ProjectRoot: proj,
		Lock:        &lockfile.Lockfile{Entries: append([]lockfile.Entry(nil), rows...)},
		Platforms:   []platform.ID{platform.Claude},
		Reporter:    Reporter{IO: iostreams.Std()},
	}
}

func rmApply(t *testing.T, in *Installer, targets ...RemoveTarget) (*Plan, error) {
	t.Helper()
	p, err := in.RemovePlan(nil, targets)
	if err != nil {
		t.Fatal(err)
	}
	return p, in.Apply(p)
}

func TestRemovePlanBuildsOneOpPerTarget(t *testing.T) {
	proj := rmProject(t)
	skill := filepath.Join(proj, ".claude", "skills", "s")
	rmWrite(t, filepath.Join(skill, "SKILL.md"), "x")
	rows := []lockfile.Entry{{Domain: "d", Kind: lockfile.KindSkills, Name: "s", Mode: lockfile.ModeCopy, Path: skill, Platforms: []string{"claude"}}}
	in := rmInstaller(proj, rows)
	p, err := in.RemovePlan(nil, []RemoveTarget{{Name: "d", Domain: true, Rows: rows}, {Name: "s", Rows: rows}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 {
		t.Fatalf("ops = %d, want 2", len(p.Ops))
	}
	for i, name := range []string{"d", "s"} {
		if p.Ops[i].Type != OpRemove || p.Ops[i].Name != name || p.Ops[i].rm.harness {
			t.Fatalf("op %d = %+v", i, p.Ops[i])
		}
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("inside-root plan rejected: %v", err)
	}
	if err := in.Apply(p); err != nil {
		t.Fatal(err)
	}
	if rmExists(skill) || len(in.Lock.Entries) != 0 {
		t.Fatalf("skill or row survived: %v", in.Lock.Entries)
	}
}

func TestRemovePlanLegacyRowWithoutPlatformsOutsideRefused(t *testing.T) {
	proj := rmProject(t)
	victim := filepath.Join(t.TempDir(), "victim")
	rmWrite(t, victim, "x")
	rows := []lockfile.Entry{{Domain: "d", Kind: lockfile.KindSkills, Name: "evil", Mode: lockfile.ModeCopy, Path: victim}}
	p, err := rmInstaller(proj, rows).RemovePlan(nil, []RemoveTarget{{Name: "evil", Rows: rows}})
	if err != nil {
		t.Fatal(err)
	}
	if verr := p.Validate(); verr == nil || !strings.Contains(verr.Error(), victim) {
		t.Fatalf("Validate = %v", verr)
	}
}

func TestRemoveKeepsForeignFileInSharedDir(t *testing.T) {
	proj := rmProject(t)
	dir := filepath.Join(proj, ".claude", "skills")
	rmWrite(t, filepath.Join(dir, "mine", "SKILL.md"), "x")
	rmWrite(t, filepath.Join(dir, "foreign", "SKILL.md"), "keep")
	rows := []lockfile.Entry{{Domain: "d", Kind: lockfile.KindSkills, Mode: lockfile.ModeCopy, Path: dir, Items: []string{"mine"}, Platforms: []string{"claude"}}}
	in := rmInstaller(proj, rows)
	if _, err := rmApply(t, in, RemoveTarget{Name: "d", Domain: true, Rows: rows}); err != nil {
		t.Fatal(err)
	}
	if rmExists(filepath.Join(dir, "mine")) || !rmExists(filepath.Join(dir, "foreign", "SKILL.md")) {
		t.Fatal("tracked item must go, foreign must stay")
	}
	if len(in.Lock.Entries) != 0 {
		t.Fatalf("rows = %v", in.Lock.Entries)
	}
}

func TestRemoveLinkRowOverRealDirSkipped(t *testing.T) {
	proj := rmProject(t)
	skill := filepath.Join(proj, ".claude", "skills", "s")
	rmWrite(t, filepath.Join(skill, "SKILL.md"), "user data")
	rows := []lockfile.Entry{{Domain: "d", Kind: lockfile.KindSkills, Name: "s", Mode: lockfile.ModeLink, Path: skill, Platforms: []string{"claude"}}}
	in := rmInstaller(proj, rows)
	if _, err := rmApply(t, in, RemoveTarget{Name: "s", Rows: rows}); err != nil {
		t.Fatal(err)
	}
	if !rmExists(filepath.Join(skill, "SKILL.md")) {
		t.Fatal("real dir under a link row was removed")
	}
	if len(in.Lock.Entries) != 1 {
		t.Fatalf("row must be kept, got %v", in.Lock.Entries)
	}
}

func TestRemoveCopyRowOverSymlinkSkipped(t *testing.T) {
	proj := rmProject(t)
	target := t.TempDir()
	rmWrite(t, filepath.Join(target, "SKILL.md"), "user data")
	skill := filepath.Join(proj, ".claude", "skills", "s")
	if err := os.Symlink(target, skill); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []lockfile.Mode{lockfile.ModeCopy, lockfile.ModeRender} {
		rows := []lockfile.Entry{{Domain: "d", Kind: lockfile.KindSkills, Name: "s", Mode: mode, Path: skill, Platforms: []string{"claude"}}}
		in := rmInstaller(proj, rows)
		if _, err := rmApply(t, in, RemoveTarget{Name: "s", Rows: rows}); err != nil {
			t.Fatal(err)
		}
		if !rmExists(skill) || !rmExists(filepath.Join(target, "SKILL.md")) {
			t.Fatalf("%s: symlink or its target removed", mode)
		}
		if len(in.Lock.Entries) != 1 {
			t.Fatalf("%s: row must be kept, got %v", mode, in.Lock.Entries)
		}
	}
}

func TestRemoveRefusesEscapingRows(t *testing.T) {
	proj := rmProject(t)
	out := t.TempDir()
	rmWrite(t, filepath.Join(out, "x", "keep"), "x")
	if err := os.Symlink(out, filepath.Join(proj, ".claude", "skills", "esc")); err != nil {
		t.Fatal(err)
	}
	for name, p := range map[string]string{
		"symlink":   filepath.Join(proj, ".claude", "skills", "esc", "x"),
		"traversal": filepath.Join(proj, ".claude", "skills", "..", "..", filepath.Base(out)),
	} {
		rows := []lockfile.Entry{{Domain: "d", Kind: lockfile.KindSkills, Name: "x", Mode: lockfile.ModeCopy, Path: p, Platforms: []string{"claude"}}}
		in := rmInstaller(proj, rows)
		_, err := rmApply(t, in, RemoveTarget{Name: "x", Rows: rows})
		if err == nil || !strings.Contains(err.Error(), "nothing changed") {
			t.Fatalf("%s: err = %v", name, err)
		}
		if !rmExists(filepath.Join(out, "x", "keep")) || len(in.Lock.Entries) != 1 {
			t.Fatalf("%s: victim or row touched", name)
		}
	}
}

func TestRemoveValidateRejectsSecondTargetBeforeFirstRemoved(t *testing.T) {
	proj := rmProject(t)
	first := filepath.Join(proj, ".claude", "skills", "a")
	rmWrite(t, filepath.Join(first, "SKILL.md"), "x")
	victim := filepath.Join(t.TempDir(), "victim")
	rmWrite(t, victim, "x")
	r1 := lockfile.Entry{Domain: "d", Kind: lockfile.KindSkills, Name: "a", Mode: lockfile.ModeCopy, Path: first, Platforms: []string{"claude"}}
	r2 := lockfile.Entry{Domain: "d", Kind: lockfile.KindSkills, Name: "b", Mode: lockfile.ModeCopy, Path: victim, Platforms: []string{"claude"}}
	in := rmInstaller(proj, []lockfile.Entry{r1, r2})
	_, err := rmApply(t, in, RemoveTarget{Name: "a", Rows: []lockfile.Entry{r1}}, RemoveTarget{Name: "b", Rows: []lockfile.Entry{r2}})
	if err == nil || !strings.Contains(err.Error(), victim) {
		t.Fatalf("err = %v", err)
	}
	if !rmExists(first) || !rmExists(victim) || len(in.Lock.Entries) != 2 {
		t.Fatal("op #1 ran before validation failed")
	}
}

func TestRemoveUnregistersOnlyRemovedHooksAndSparesLocalSettings(t *testing.T) {
	proj := rmProject(t)
	hooks := filepath.Join(proj, ".claude", "hooks")
	rmWrite(t, filepath.Join(hooks, "gate.sh"), "#!/bin/sh\n")
	rmWrite(t, filepath.Join(hooks, "fmt.sh"), "#!/bin/sh\n")
	real := filepath.Join(t.TempDir(), "load.sh")
	rmWrite(t, real, "#!/bin/sh\n")
	if err := os.Symlink(real, filepath.Join(hooks, "load.sh")); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(proj, ".claude", "settings.json")
	local := filepath.Join(proj, ".claude", "settings.local.json")
	rmWrite(t, settings, registeredSettings)
	rmWrite(t, local, registeredSettings)
	rows := []lockfile.Entry{{Domain: "alpha", Kind: lockfile.KindHooks, Mode: lockfile.ModeCopy, Path: hooks, Items: []string{"gate.sh", "fmt.sh", "load.sh"}, Platforms: []string{"claude"}}}
	in := rmInstaller(proj, rows)
	if _, err := rmApply(t, in, RemoveTarget{Name: "alpha", Domain: true, Rows: rows}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(settings)
	if strings.Contains(string(data), "gate.sh") || strings.Contains(string(data), "fmt.sh") {
		t.Fatalf("removed hooks still registered:\n%s", data)
	}
	if !strings.Contains(string(data), "load.sh") || !strings.Contains(string(data), "foreign.sh") {
		t.Fatalf("kept hook or foreign entry unregistered:\n%s", data)
	}
	if l, _ := os.ReadFile(local); string(l) != registeredSettings {
		t.Fatal("settings.local.json modified")
	}
	if rmExists(filepath.Join(hooks, "gate.sh")) || !rmExists(filepath.Join(hooks, "load.sh")) {
		t.Fatal("wrong hook files removed")
	}
	if len(in.Lock.Entries) != 1 {
		t.Fatalf("row with skipped item must be kept, got %v", in.Lock.Entries)
	}
}

func TestRemovePartialFailureKeepsOnlyUnremovedRows(t *testing.T) {
	proj := rmProject(t)
	dir := filepath.Join(proj, ".claude", "skills")
	rmWrite(t, filepath.Join(dir, "gone", "SKILL.md"), "x")
	rmWrite(t, filepath.Join(dir, "stuck", "SKILL.md"), "user")
	r1 := lockfile.Entry{Domain: "d", Kind: lockfile.KindSkills, Name: "gone", Mode: lockfile.ModeCopy, Path: filepath.Join(dir, "gone"), Platforms: []string{"claude"}}
	r2 := lockfile.Entry{Domain: "d", Kind: lockfile.KindSkills, Name: "stuck", Mode: lockfile.ModeLink, Path: filepath.Join(dir, "stuck"), Platforms: []string{"claude"}}
	in := rmInstaller(proj, []lockfile.Entry{r1, r2})
	if _, err := rmApply(t, in, RemoveTarget{Name: "d", Domain: true, Rows: []lockfile.Entry{r1, r2}}); err != nil {
		t.Fatal(err)
	}
	if len(in.Lock.Entries) != 1 || in.Lock.Entries[0].Name != "stuck" {
		t.Fatalf("rows = %v", in.Lock.Entries)
	}
}

func TestRemovePlanPartOfDomainRejected(t *testing.T) {
	proj := filepath.Join("/nonexistent", "proj")
	cover := lockfile.Entry{Domain: "d", Kind: lockfile.KindSkills, Mode: lockfile.ModeCopy, Path: filepath.Join(proj, ".claude", "skills"), Platforms: []string{"claude"}}
	in := &Installer{ProjectRoot: proj, Lock: &lockfile.Lockfile{}, Platforms: []platform.ID{platform.Claude}}
	_, err := in.RemovePlan(nil, []RemoveTarget{{Name: "s", Cover: []lockfile.Entry{cover}}})
	if _, ok := err.(*PartOfDomainError); !ok {
		t.Fatalf("err = %v, want PartOfDomainError", err)
	}
}

func TestRemovePlanValidateRejectsOutsideRootBeforeAnyOp(t *testing.T) {
	proj := filepath.Join("/nonexistent", "proj")
	outside := filepath.Join("/nonexistent", "victim")
	rows := []lockfile.Entry{{Domain: "d", Kind: lockfile.KindSkills, Name: "evil", Mode: lockfile.ModeCopy, Path: outside, Platforms: []string{"claude"}}}
	in := &Installer{ProjectRoot: proj, Lock: &lockfile.Lockfile{Entries: rows}, Platforms: []platform.ID{platform.Claude}}
	p, err := in.RemovePlan(nil, []RemoveTarget{{Name: "evil", Rows: rows}})
	if err != nil {
		t.Fatal(err)
	}
	verr := p.Validate()
	if verr == nil || !strings.Contains(verr.Error(), "remove refused, nothing changed") || !strings.Contains(verr.Error(), outside) {
		t.Fatalf("Validate = %v", verr)
	}
	if aerr := in.Apply(p); aerr == nil || aerr.Error() != verr.Error() {
		t.Fatalf("Apply = %v, want %v", aerr, verr)
	}
}

func TestUninstallOneKeepsUntrackedFileUnderItemizedParent(t *testing.T) {
	proj := rmProject(t)
	dir := filepath.Join(proj, ".claude", "skills")
	rmWrite(t, filepath.Join(dir, "tracked", "SKILL.md"), "x")
	rmWrite(t, filepath.Join(dir, "mine", "SKILL.md"), "user")
	row := lockfile.Entry{Domain: "d", Kind: lockfile.KindSkills, Mode: lockfile.ModeCopy, Path: dir, Items: []string{"tracked"}, Platforms: []string{"claude"}}
	in := rmInstaller(proj, []lockfile.Entry{row})
	ios, _, errBuf := iostreams.Test()
	in.Reporter = Reporter{IO: ios}
	dst := filepath.Join(dir, "mine")
	if in.uninstallOne("d", lockfile.KindSkills, "mine", dst) {
		t.Fatal("uninstallOne reported clean for untracked file")
	}
	if !rmExists(filepath.Join(dst, "SKILL.md")) {
		t.Fatal("untracked user file was removed")
	}
	if want := "skipping " + dst + " — not tracked by as-skill"; !strings.Contains(errBuf.String(), want) {
		t.Fatalf("missing warning %q, got %q", want, errBuf.String())
	}
	if len(in.Lock.Entries) != 1 || len(in.Lock.Entries[0].Items) != 1 || in.Lock.Entries[0].Items[0] != "tracked" {
		t.Fatalf("row changed: %+v", in.Lock.Entries)
	}
}

func TestUninstallOneLegacyWholeTreeRowStillRemoves(t *testing.T) {
	proj := rmProject(t)
	dir := filepath.Join(proj, ".claude", "skills")
	rmWrite(t, filepath.Join(dir, "beta-one", "SKILL.md"), "x")
	row := lockfile.Entry{Domain: "d", Kind: lockfile.KindSkills, Mode: lockfile.ModeCopy, Path: dir, Source: filepath.Join(proj, "gone"), Platforms: []string{"claude"}}
	in := rmInstaller(proj, []lockfile.Entry{row})
	ios, _, _ := iostreams.Test()
	in.Reporter = Reporter{IO: ios}
	dst := filepath.Join(dir, "beta-one")
	if !in.uninstallOne("d", lockfile.KindSkills, "beta-one", dst) {
		t.Fatal("legacy row did not report removal")
	}
	if rmExists(dst) {
		t.Fatal("legacy row file kept")
	}
}
