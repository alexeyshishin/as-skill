package install

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func TestPlanClaudeKindWholeTreeCopy(t *testing.T) {
	proj := filepath.Join("/nonexistent", "proj")
	in := &Installer{HarnessRoot: "/nonexistent/h", ProjectRoot: proj, Mode: lockfile.ModeCopy, Lock: &lockfile.Lockfile{}}
	m := &harness.Manifest{}
	src := "/nonexistent/h/domains/d/skills"
	p, err := in.planClaudeKind(m, "d", lockfile.KindSkills, src, []platform.ID{platform.Claude})
	if err != nil {
		t.Fatal(err)
	}
	want := []Op{{
		Type: OpCopy, Src: src, Dst: filepath.Join(proj, ".claude", "skills"),
		Mode: lockfile.ModeCopy, Domain: "d", Kind: lockfile.KindSkills,
		Platforms: []platform.ID{platform.Claude},
	}}
	if !reflect.DeepEqual(p.Ops, want) {
		t.Fatalf("ops = %+v\nwant %+v", p.Ops, want)
	}
	if len(p.Warnings) != 0 {
		t.Fatalf("unexpected warnings %+v", p.Warnings)
	}
}

func TestPlanClaudeKindExclusionsAndMixedChannel(t *testing.T) {
	proj := "/nonexistent/proj"
	dst := filepath.Join(proj, ".claude", "skills")
	lock := &lockfile.Lockfile{}
	lock.Record(lockfile.Entry{Domain: "d", Kind: lockfile.KindSkills, Mode: lockfile.ModeLink, Path: dst})
	in := &Installer{
		ProjectRoot: proj, Mode: lockfile.ModeLink, Lock: lock,
		ExcludeSkills: map[string]bool{"d/b": true, "d/a": true, "d/off": false, "other/x": true},
	}
	p, err := in.planClaudeKind(&harness.Manifest{}, "d", lockfile.KindSkills, "/nonexistent/src", []platform.ID{platform.Claude})
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Ops[0]; got.Type != OpLink || !reflect.DeepEqual(got.Exclude, []string{"a", "b"}) {
		t.Fatalf("op = %+v", got)
	}

	in.Mode = lockfile.ModeCopy
	in.ExcludeSkills = nil
	p, err = in.planClaudeKind(&harness.Manifest{}, "d", lockfile.KindSkills, "/nonexistent/src", []platform.ID{platform.Claude})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Warnings) != 1 || p.Warnings[0].Before != 0 || p.Warnings[0].Stream != Stderr ||
		!strings.Contains(p.Warnings[0].Text, "replacing with a copy") {
		t.Fatalf("warnings = %+v", p.Warnings)
	}
}

func TestPlanClaudeSkill(t *testing.T) {
	proj := "/nonexistent/proj"
	in := &Installer{ProjectRoot: proj, Mode: lockfile.ModeLink, Lock: &lockfile.Lockfile{}}
	p, err := in.planClaudeSkill(&harness.Manifest{}, "d", "s", "/nonexistent/src/s", []platform.ID{platform.Claude})
	if err != nil {
		t.Fatal(err)
	}
	op := p.Ops[0]
	if op.Type != OpLink || op.Kind != lockfile.KindSkill || op.Name != "s" || op.Dst != filepath.Join(proj, ".claude", "skills", "s") {
		t.Fatalf("op = %+v", op)
	}
}
