package install

import (
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func TestRemovePlanBuildsOneOpPerTarget(t *testing.T) {
	proj := filepath.Join("/nonexistent", "proj")
	rows := []lockfile.Entry{{Domain: "d", Kind: lockfile.KindSkills, Name: "s", Mode: lockfile.ModeLink, Path: filepath.Join(proj, ".claude", "skills", "s")}}
	in := &Installer{ProjectRoot: proj, Lock: &lockfile.Lockfile{Entries: rows}, Platforms: []platform.ID{platform.Claude}}
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
