package fsys

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLinkOneCreatesAbsoluteTargetSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst")
	if _, err := LinkOne(src, dst, false, false, untracked); err != nil {
		t.Fatalf("LinkOne: %v", err)
	}

	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("Lstat(dst): %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("dst is not a symlink: mode=%v", fi.Mode())
	}
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if target != src {
		t.Fatalf("symlink target = %q, want %q", target, src)
	}
	if !filepath.IsAbs(target) {
		t.Fatalf("symlink target %q is not absolute", target)
	}
}

func TestLinkOneReplacesPreExistingContent(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, dst string)
	}{
		{
			name: "file",
			setup: func(t *testing.T, dst string) {
				if err := os.WriteFile(dst, []byte("old file"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "dir",
			setup: func(t *testing.T, dst string) {
				if err := os.MkdirAll(filepath.Join(dst, "nested"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "symlink",
			setup: func(t *testing.T, dst string) {
				elsewhere := filepath.Join(filepath.Dir(dst), "elsewhere")
				if err := os.WriteFile(elsewhere, []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(elsewhere, dst); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "src")
			if err := os.WriteFile(src, []byte("new content"), 0o644); err != nil {
				t.Fatal(err)
			}
			dst := filepath.Join(dir, "dst")
			tc.setup(t, dst)

			if _, err := LinkOne(src, dst, false, true, untracked); err != nil {
				t.Fatalf("LinkOne with force: %v", err)
			}

			fi, err := os.Lstat(dst)
			if err != nil {
				t.Fatalf("Lstat(dst): %v", err)
			}
			if fi.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("dst was not replaced with a symlink: mode=%v", fi.Mode())
			}
			target, err := os.Readlink(dst)
			if err != nil {
				t.Fatal(err)
			}
			if target != src {
				t.Fatalf("target = %q, want %q", target, src)
			}
		})
	}
}

func TestLinkOneDryRunLeavesNothing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst")
	if _, err := LinkOne(src, dst, true, false, untracked); err != nil {
		t.Fatalf("LinkOne dry-run: %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("expected dst to not exist after dry-run, Lstat err = %v", err)
	}
}

func TestLinkOneRefusesUntrackedExistingContentWithoutForce(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst")
	if err := os.WriteFile(dst, []byte("pre-existing, not tracked"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LinkOne(src, dst, false, false, untracked)
	if err == nil {
		t.Fatal("expected error refusing untracked existing content without --force")
	}

	got, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatalf("dst should still exist: %v", readErr)
	}
	if string(got) != "pre-existing, not tracked" {
		t.Fatalf("dst content changed despite refusal: %q", got)
	}
}

func linkTreeSrc(t *testing.T) string {
	t.Helper()
	src := filepath.Join(t.TempDir(), "skills")
	for _, n := range []string{"a", "b", "c"} {
		if err := os.MkdirAll(filepath.Join(src, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return src
}

func untracked(string) (string, bool) { return "", false }

func linkOneWith() func(src, dst string) error {
	return func(s, d string) error {
		_, err := LinkOne(s, d, false, false, untracked)
		return err
	}
}

func TestLinkTreeWithSkipLeavesNamesOutAndUnrecorded(t *testing.T) {
	src := linkTreeSrc(t)
	dst := filepath.Join(t.TempDir(), "out")
	done, err := LinkTreeWith(false, src, dst, false, func(name string) bool { return name == "b" }, linkOneWith())
	if err != nil || len(done) != 2 {
		t.Fatalf("LinkTreeWith = %d, %v; want 2, nil", len(done), err)
	}
	for name, want := range map[string]bool{"a": true, "b": false, "c": true} {
		_, err := os.Lstat(filepath.Join(dst, name))
		if (err == nil) != want {
			t.Fatalf("%s present = %v, want %v", name, err == nil, want)
		}
		listed := false
		for _, d := range done {
			listed = listed || d.Name == name
		}
		if listed != want {
			t.Fatalf("%s returned = %v, want %v", name, listed, want)
		}
	}
}

func TestLinkTreeWithNilSkipLinksEverything(t *testing.T) {
	src := linkTreeSrc(t)
	dst := filepath.Join(t.TempDir(), "out")
	done, err := LinkTreeWith(false, src, dst, false, nil, linkOneWith())
	if err != nil || len(done) != 3 {
		t.Fatalf("LinkTreeWith = %d, %v; want 3, nil", len(done), err)
	}
	for _, name := range []string{"a", "b", "c"} {
		target, err := os.Readlink(filepath.Join(dst, name))
		if err != nil || target != filepath.Join(src, name) {
			t.Fatalf("%s -> %q, %v", name, target, err)
		}
	}
}

func TestLinkOneForeignErrorTypedWithUnchangedText(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	for _, p := range []string{src, dst} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := LinkOne(src, dst, false, false, untracked)
	var fe *ForeignError
	if !errors.As(err, &fe) || fe.Path != dst {
		t.Fatalf("want *ForeignError for %s, got %v", dst, err)
	}
	if want := dst + " exists, not tracked by as-skill, use --force"; err.Error() != want {
		t.Fatalf("text = %q, want %q", err.Error(), want)
	}
}

func TestLinkTreeWithCustomLinkerCanSwallowRefusal(t *testing.T) {
	src := linkTreeSrc(t)
	dst := filepath.Join(t.TempDir(), "out")
	var seen []string
	done, err := LinkTreeWith(false, src, dst, true, nil, func(s, d string) error {
		seen = append(seen, filepath.Base(d))
		return nil
	})
	if err != nil || len(done) != 3 || len(seen) != 3 {
		t.Fatalf("LinkTreeWith = %d, %v, seen %v", len(done), err, seen)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("dry run created %s: %v", dst, err)
	}
}

func TestLinkTreeWithCustomLinkerSwallowsTypedForeignError(t *testing.T) {
	src := linkTreeSrc(t)
	dst := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(dst, "b")
	if err := os.WriteFile(foreign, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	var swallowed []string
	done, err := LinkTreeWith(false, src, dst, false, nil, func(s, d string) error {
		_, err := LinkOne(s, d, false, false, untracked)
		var fe *ForeignError
		if errors.As(err, &fe) {
			swallowed = append(swallowed, fe.Path)
			return nil
		}
		return err
	})
	if err != nil || len(done) != 3 {
		t.Fatalf("LinkTreeWith = %d, %v; want 3, nil", len(done), err)
	}
	if len(swallowed) != 1 || swallowed[0] != foreign {
		t.Fatalf("swallowed = %v, want [%s]", swallowed, foreign)
	}
	if got, err := os.ReadFile(foreign); err != nil || string(got) != "mine" {
		t.Fatalf("foreign file touched: %q, %v", got, err)
	}
}

func TestLinkTreeWithCustomLinkerPropagatesTypedForeignError(t *testing.T) {
	src := linkTreeSrc(t)
	dst := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(dst, "b")
	if err := os.WriteFile(foreign, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	done, err := LinkTreeWith(false, src, dst, false, nil, linkOneWith())
	var fe *ForeignError
	if !errors.As(err, &fe) || fe.Path != foreign {
		t.Fatalf("want *ForeignError for %s, got %v", foreign, err)
	}
	if len(done) != 1 {
		t.Fatalf("count = %d, want 1 (only a linked before the refusal)", len(done))
	}
	if len(done) != 1 || done[0].Name != "a" {
		t.Fatalf("done = %v, want only a", done)
	}
}
