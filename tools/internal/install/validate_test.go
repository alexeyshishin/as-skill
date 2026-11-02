package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func validateInstaller(t *testing.T, proj string) *Installer {
	t.Helper()
	return &Installer{HarnessRoot: t.TempDir(), ProjectRoot: proj, Mode: lockfile.ModeCopy, Lock: &lockfile.Lockfile{}}
}

func TestApplyAcceptsPlanInsideRoots(t *testing.T) {
	proj := t.TempDir()
	src := filepath.Join(t.TempDir(), "r.md")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := validateInstaller(t, proj)
	dst := filepath.Join(proj, ".claude", "rules", "r.md")
	p := &Plan{Ops: []Op{{Type: OpCopy, Item: true, Src: src, Dst: dst, Mode: lockfile.ModeCopy, Domain: "d", Kind: lockfile.KindRules, Name: "r.md", Platforms: []platform.ID{platform.Claude}}}}
	if err := in.ValidateInstall(p); err != nil {
		t.Fatal(err)
	}
	if err := in.Apply(p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatal(err)
	}
}

func TestApplyRejectsOutsideDstBeforeAnyWrite(t *testing.T) {
	proj := t.TempDir()
	out := t.TempDir()
	src := filepath.Join(t.TempDir(), "r.md")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := validateInstaller(t, proj)
	inside := filepath.Join(proj, ".claude", "rules", "ok.md")
	bad := filepath.Join(out, "evil.md")
	mk := func(dst string) Op {
		return Op{Type: OpCopy, Item: true, Src: src, Dst: dst, Mode: lockfile.ModeCopy, Domain: "d", Kind: lockfile.KindRules, Name: filepath.Base(dst), Platforms: []platform.ID{platform.Claude}}
	}
	p := &Plan{Ops: []Op{mk(inside), mk(bad)}}
	err := in.Apply(p)
	if err == nil || !strings.Contains(err.Error(), "outside the project's platform dirs") {
		t.Fatalf("err = %v", err)
	}
	if _, serr := os.Lstat(inside); serr == nil {
		t.Fatal("op #1 written before validation")
	}
	if _, serr := os.Lstat(bad); serr == nil {
		t.Fatal("outside dst written")
	}
}

func TestApplyRejectsSymlinkedRootBeforeAnyWrite(t *testing.T) {
	proj := t.TempDir()
	out := t.TempDir()
	if err := os.Symlink(out, filepath.Join(proj, ".claude")); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "r.md")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := validateInstaller(t, proj)
	dst := filepath.Join(proj, ".claude", "rules", "r.md")
	p := &Plan{Ops: []Op{{Type: OpCopy, Item: true, Src: src, Dst: dst, Mode: lockfile.ModeCopy, Domain: "d", Kind: lockfile.KindRules, Name: "r.md", Platforms: []platform.ID{platform.Claude}}}}
	err := in.Apply(p)
	if err == nil || !strings.Contains(err.Error(), "does not resolve") {
		t.Fatalf("err = %v", err)
	}
	if _, serr := os.Lstat(filepath.Join(out, "rules")); serr == nil {
		t.Fatal("wrote through symlinked root")
	}
}
