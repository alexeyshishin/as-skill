package transfer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"claude-harness/tools/internal/lockfile"
)

func TestLinkOneCreatesAbsoluteTargetSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst")
	lock := &lockfile.Lockfile{}

	if err := LinkOne(src, dst, false, false, lock); err != nil {
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

			lock := &lockfile.Lockfile{}

			if err := LinkOne(src, dst, false, true, lock); err != nil {
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
	lock := &lockfile.Lockfile{}

	if err := LinkOne(src, dst, true, false, lock); err != nil {
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
	lock := &lockfile.Lockfile{}

	err := LinkOne(src, dst, false, false, lock)
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

func linkOneWith(lock *lockfile.Lockfile) func(src, dst string) error {
	return func(s, d string) error { return LinkOne(s, d, false, false, lock) }
}

func TestLinkTreeWithSkipLeavesNamesOutAndUnrecorded(t *testing.T) {
	src := linkTreeSrc(t)
	dst := filepath.Join(t.TempDir(), "out")
	lock := &lockfile.Lockfile{}

	n, err := LinkTreeWith("d", "skills", src, dst, false, lock, func(name string) bool { return name == "b" }, linkOneWith(lock))
	if err != nil || n != 2 {
		t.Fatalf("LinkTreeWith = %d, %v; want 2, nil", n, err)
	}
	for name, want := range map[string]bool{"a": true, "b": false, "c": true} {
		_, err := os.Lstat(filepath.Join(dst, name))
		if (err == nil) != want {
			t.Fatalf("%s present = %v, want %v", name, err == nil, want)
		}
		_, owned := lock.Owns(filepath.Join(dst, name))
		if owned != want {
			t.Fatalf("%s row = %v, want %v", name, owned, want)
		}
	}
}

func TestLinkTreeWithNilSkipLinksEverything(t *testing.T) {
	src := linkTreeSrc(t)
	dst := filepath.Join(t.TempDir(), "out")
	lock := &lockfile.Lockfile{}
	n, err := LinkTreeWith("d", "skills", src, dst, false, lock, nil, linkOneWith(lock))
	if err != nil || n != 3 {
		t.Fatalf("LinkTreeWith = %d, %v; want 3, nil", n, err)
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
	err := LinkOne(src, dst, false, false, &lockfile.Lockfile{})
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
	n, err := LinkTreeWith("d", "skills", src, dst, true, &lockfile.Lockfile{}, nil, func(s, d string) error {
		seen = append(seen, filepath.Base(d))
		return nil
	})
	if err != nil || n != 3 || len(seen) != 3 {
		t.Fatalf("LinkTreeWith = %d, %v, seen %v", n, err, seen)
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
	lock := &lockfile.Lockfile{}
	var swallowed []string
	n, err := LinkTreeWith("d", "skills", src, dst, false, lock, nil, func(s, d string) error {
		err := LinkOne(s, d, false, false, lock)
		var fe *ForeignError
		if errors.As(err, &fe) {
			swallowed = append(swallowed, fe.Path)
			return nil
		}
		return err
	})
	if err != nil || n != 3 {
		t.Fatalf("LinkTreeWith = %d, %v; want 3, nil", n, err)
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
	lock := &lockfile.Lockfile{}
	n, err := LinkTreeWith("d", "skills", src, dst, false, lock, nil, linkOneWith(lock))
	var fe *ForeignError
	if !errors.As(err, &fe) || fe.Path != foreign {
		t.Fatalf("want *ForeignError for %s, got %v", foreign, err)
	}
	if n != 1 {
		t.Fatalf("count = %d, want 1 (only a linked before the refusal)", n)
	}
	if _, owned := lock.Owns(foreign); owned {
		t.Fatal("refused path recorded in lock")
	}
	if _, owned := lock.Owns(filepath.Join(dst, "c")); owned {
		t.Fatal("tree continued past the refusal")
	}
}
