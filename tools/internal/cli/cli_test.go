package cli

import (
	"os"
	"path/filepath"
	"testing"

	"claude-harness/tools/internal/lockfile"
)

func fakeHarnessWithSkill(t *testing.T) (harnessRoot, skillDir string) {
	t.Helper()
	harnessRoot = t.TempDir()
	skillDir = filepath.Join(harnessRoot, "domains", "testdomain", "skills", "myskill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: myskill\ndescription: test\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harnessRoot, "domains", "testdomain", "manifest.yaml"), []byte("name: testdomain\ndescription: test domain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return harnessRoot, skillDir
}

func TestRunInstallDefaultsToSymlink(t *testing.T) {
	harnessRoot, _ := fakeHarnessWithSkill(t)
	projectRoot := t.TempDir()

	if err := Run([]string{"install", "skill", "myskill", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("Run install: %v", err)
	}

	dst := filepath.Join(projectRoot, ".claude", "skills", "myskill")
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("Lstat(dst): %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected `install` with no flag to symlink, mode=%v", fi.Mode())
	}
}

func TestRunInstallCopyFlagProducesRealFiles(t *testing.T) {
	harnessRoot, _ := fakeHarnessWithSkill(t)
	projectRoot := t.TempDir()

	if err := Run([]string{"install", "skill", "myskill", "--copy", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("Run install --copy: %v", err)
	}

	dst := filepath.Join(projectRoot, ".claude", "skills", "myskill")
	fi, err := os.Lstat(dst)
	if err != nil {
		t.Fatalf("Lstat(dst): %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected `install --copy` to produce a real dir, got a symlink")
	}
	data, err := os.ReadFile(filepath.Join(dst, "SKILL.md"))
	if err != nil {
		t.Fatalf("expected copied SKILL.md to exist: %v", err)
	}
	if string(data) != "---\nname: myskill\ndescription: test\n---\n" {
		t.Fatalf("copied SKILL.md content mismatch: %q", data)
	}
}

func TestRunUninstallCopyModeRemovesRealFiles(t *testing.T) {
	harnessRoot, _ := fakeHarnessWithSkill(t)
	projectRoot := t.TempDir()

	if err := Run([]string{"install", "skill", "myskill", "--copy", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("Run install --copy: %v", err)
	}
	dst := filepath.Join(projectRoot, ".claude", "skills", "myskill")
	if _, err := os.Lstat(dst); err != nil {
		t.Fatalf("precondition: copied dir should exist: %v", err)
	}

	if err := Run([]string{"uninstall", "skill", "myskill", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("Run uninstall: %v", err)
	}

	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("expected copy-mode uninstall to remove the real files, Lstat err = %v", err)
	}
	lock, err := lockfile.Load(projectRoot)
	if err != nil {
		t.Fatalf("lockfile.Load: %v", err)
	}
	if _, ok := lock.Owns(dst); ok {
		t.Fatal("expected lockfile entry to be dropped after uninstall")
	}
}
