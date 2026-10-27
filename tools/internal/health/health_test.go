package health

import (
	"os"
	"path/filepath"
	"testing"

	"claude-harness/tools/internal/lockfile"
)

func TestFindUntrackedCopyParentDoesNotHideForeign(t *testing.T) {
	project := t.TempDir()
	src := filepath.Join(t.TempDir(), "skills")
	dest := filepath.Join(project, ".claude", "skills")
	for _, p := range []string{filepath.Join(src, "mine"), filepath.Join(dest, "mine"), filepath.Join(dest, "foreign")} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	lock := &lockfile.Lockfile{Entries: []lockfile.Entry{{Domain: "d", Kind: "skills", Mode: "copy", Path: dest, Source: src}}}
	got, err := FindUntracked(project, lock)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dest, "foreign")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("want [%s], got %v", want, got)
	}
}
