package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyWriteVerbsRefuseHome(t *testing.T) {
	h := realHarness(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := filepath.Join(home, "proj")
	if err := os.Mkdir(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	_, addErr := run(t, "add", "git", "-y", "-p", home, "--harness-root", h)
	if addErr == nil || !strings.Contains(addErr.Error(), "is $HOME") {
		t.Fatalf("add baseline: %v", addErr)
	}
	cases := map[string][]string{
		"install":       {"install", "domain", "git", "--harness-root", h},
		"uninstall":     {"uninstall", "domain", "git", "--harness-root", h},
		"hooks enable":  {"hooks", "enable", "all"},
		"hooks disable": {"hooks", "disable", "all"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := run(t, append(append([]string{}, args...), "-p", home)...)
			if err == nil || err.Error() != addErr.Error() {
				t.Fatalf("want %q, got %v", addErr, err)
			}
			if _, err := os.Lstat(filepath.Join(home, ".claude")); err == nil {
				t.Fatal("wrote into HOME")
			}
		})
	}
	t.Run("non-home works", func(t *testing.T) {
		if _, err := run(t, "install", "domain", "git", "-p", proj, "--harness-root", h); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(filepath.Join(proj, ".claude", "skills", "git-conventional-commits")); err != nil {
			t.Fatal(err)
		}
		if _, err := run(t, "uninstall", "domain", "git", "-p", proj, "--harness-root", h); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("status unrestricted", func(t *testing.T) {
		if _, err := run(t, "status", "-p", home); err != nil {
			t.Fatal(err)
		}
	})
}
