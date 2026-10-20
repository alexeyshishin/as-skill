package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

// TestCoreLockfileMigrationInstallThenUninstall regression-tests the
// lockfile migration for anyone who installed core skills the old way
// (before core got its own domains/core/manifest.yaml). The pre-migration
// lockfile records core skills as {domain:"core", kind:"skill"} (singular);
// post-migration the generic domain-install path records {kind:"skills"}
// (plural). Without the migration, reinstalling would append a duplicate
// row and uninstalling would leave an orphaned one behind.
func TestCoreLockfileMigrationInstallThenUninstall(t *testing.T) {
	harnessRoot := t.TempDir()
	skillDir := filepath.Join(harnessRoot, "domains", "core", "skills", "caveman")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: caveman\ndescription: test\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harnessRoot, "domains", "core", "manifest.yaml"), []byte("name: core\ndescription: core skills\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	projectRoot := t.TempDir()
	dst := filepath.Join(projectRoot, ".claude", "skills", "caveman")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(skillDir, dst); err != nil {
		t.Fatal(err)
	}

	// Seed a pre-migration lockfile row, as the old InstallCoreSkills
	// link-mode path would have left it.
	seed := &lockfile.Lockfile{}
	seed.Record(lockfile.Entry{
		Domain: "core", Kind: "skill", Name: "caveman",
		Mode: "link", Path: dst, Source: skillDir,
		InstalledAt: time.Now(),
	})
	if err := seed.Save(projectRoot); err != nil {
		t.Fatal(err)
	}

	if err := Run([]string{"install", "domain", "core", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("Run install domain core: %v", err)
	}

	afterInstall, err := lockfile.Load(projectRoot)
	if err != nil {
		t.Fatalf("lockfile.Load after install: %v", err)
	}
	var caveman []lockfile.Entry
	for _, e := range afterInstall.Entries {
		if e.Domain == "core" && e.Name == "caveman" {
			caveman = append(caveman, e)
		}
	}
	if len(caveman) != 1 {
		t.Fatalf("expected exactly 1 lockfile row for core/caveman after reinstall (no duplicate), got %d: %+v", len(caveman), caveman)
	}
	if caveman[0].Kind != "skills" {
		t.Fatalf("expected migrated row to carry kind %q, got %q", "skills", caveman[0].Kind)
	}

	if err := Run([]string{"uninstall", "domain", "core", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("Run uninstall domain core: %v", err)
	}

	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("expected caveman to be removed from disk, Lstat err = %v", err)
	}
	final, err := lockfile.Load(projectRoot)
	if err != nil {
		t.Fatalf("lockfile.Load after uninstall: %v", err)
	}
	for _, e := range final.Entries {
		if e.Domain == "core" {
			t.Fatalf("expected no orphaned core lockfile rows after uninstall, found %+v", e)
		}
	}
}
