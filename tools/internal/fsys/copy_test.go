package fsys

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func write(t *testing.T, p, c string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCopyTreeDereferencesSymlinksInsideRoot(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "references", "rules", "a.md"), "rule")
	write(t, filepath.Join(root, "references", "one.md"), "one")
	write(t, filepath.Join(root, "domains", "d", "skills", "s", "SKILL.md"), "skill")
	skill := filepath.Join(root, "domains", "d", "skills", "s")
	if err := os.Symlink("../../../../references/rules", filepath.Join(skill, "rules")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../../references/one.md", filepath.Join(skill, "one.md")); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "out")
	n, err := CopyTreeRoot(false, skill, dst, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("copied %d files, want 3", n)
	}
	for p, want := range map[string]string{"rules/a.md": "rule", "one.md": "one", "SKILL.md": "skill"} {
		got, err := os.ReadFile(filepath.Join(dst, p))
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q, %v", p, got, err)
		}
		fi, _ := os.Lstat(filepath.Join(dst, p))
		if fi.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("%s is still a symlink", p)
		}
	}
}

func TestCopyTreeRejectsSymlinkEscapingRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	write(t, filepath.Join(outside, "secret"), "x")
	skill := filepath.Join(root, "domains", "d", "skills", "s")
	write(t, filepath.Join(skill, "SKILL.md"), "skill")
	if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(skill, "leak")); err != nil {
		t.Fatal(err)
	}
	_, err := CopyTreeRoot(false, skill, filepath.Join(t.TempDir(), "out"), root, false)
	if err == nil || !strings.Contains(err.Error(), "outside the harness root") {
		t.Fatalf("err = %v, want escape error", err)
	}
}

func TestCopyTreeCycleGuard(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "domains", "d", "skills", "s")
	write(t, filepath.Join(skill, "SKILL.md"), "skill")
	if err := os.Symlink(".", filepath.Join(skill, "loop")); err != nil {
		t.Fatal(err)
	}
	_, err := CopyTreeRoot(false, skill, filepath.Join(t.TempDir(), "out"), root, false)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("err = %v, want cycle error", err)
	}
}

func TestCopyTreeBrokenSymlinkErrors(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "domains", "d", "skills", "s")
	write(t, filepath.Join(skill, "SKILL.md"), "skill")
	if err := os.Symlink("missing", filepath.Join(skill, "bad")); err != nil {
		t.Fatal(err)
	}
	if _, err := CopyTreeRoot(false, skill, filepath.Join(t.TempDir(), "out"), root, false); err == nil {
		t.Fatal("want error for broken symlink")
	}
}

func TestNonExecutableScripts(t *testing.T) {
	dir := t.TempDir()
	files := map[string]os.FileMode{
		"b.sh":    0o644,
		"a.sh":    0o644,
		"exec.sh": 0o755,
		"note.md": 0o644,
		"c.txt":   0o600,
	}
	for n, m := range files {
		if err := os.WriteFile(filepath.Join(dir, n), nil, m); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(dir, n), m); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "d.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(dir, "a.sh"), filepath.Join(dir, "b.sh")}
	for i := 0; i < 3; i++ {
		got := NonExecutableScripts(dir)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if got := NonExecutableScripts(filepath.Join(dir, "missing")); got != nil {
		t.Fatalf("missing dir: %v", got)
	}
}
