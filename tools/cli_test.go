package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLinkOneCreatesAbsoluteTargetSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst")
	lock := &lockfile{}

	if err := linkOne(src, dst, false, false, lock); err != nil {
		t.Fatalf("linkOne: %v", err)
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

			lock := &lockfile{}

			if err := linkOne(src, dst, false, true, lock); err != nil {
				t.Fatalf("linkOne with force: %v", err)
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
	lock := &lockfile{}

	if err := linkOne(src, dst, true, false, lock); err != nil {
		t.Fatalf("linkOne dry-run: %v", err)
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
	lock := &lockfile{}

	err := linkOne(src, dst, false, false, lock)
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

func TestLockfileRoundTrip(t *testing.T) {
	projectRoot := t.TempDir()

	lock := &lockfile{}
	entry := lockEntry{
		Domain:      "git",
		Kind:        "skill",
		Name:        "git-conventional-commits",
		Mode:        "link",
		Path:        filepath.Join(projectRoot, ".claude", "skills", "git-conventional-commits"),
		Source:      "/harness/domains/git/skills/git-conventional-commits",
		InstalledAt: time.Now(),
	}
	lock.record(entry)

	if err := lock.save(projectRoot); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := loadLockfile(projectRoot)
	if err != nil {
		t.Fatalf("loadLockfile: %v", err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(loaded.Entries))
	}

	got, ok := loaded.owns(entry.Path)
	if !ok {
		t.Fatalf("owns(%q) = false, want true", entry.Path)
	}
	if got.Domain != entry.Domain || got.Kind != entry.Kind || got.Name != entry.Name || got.Source != entry.Source {
		t.Fatalf("round-tripped entry = %+v, want %+v", got, entry)
	}

	if _, ok := loaded.owns(filepath.Join(projectRoot, "nope")); ok {
		t.Fatal("owns() found an entry for a path that was never recorded")
	}
}

func TestUninstallOneRemovesOnlyTrackedEntriesMatchingRecordedMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	linked := filepath.Join(dir, "linked")
	if err := os.Symlink(src, linked); err != nil {
		t.Fatal(err)
	}

	copied := filepath.Join(dir, "copied")
	if err := os.WriteFile(copied, []byte("copied content"), 0o644); err != nil {
		t.Fatal(err)
	}

	foreign := filepath.Join(dir, "foreign")
	if err := os.WriteFile(foreign, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	lock := &lockfile{}
	lock.record(lockEntry{Domain: "d", Kind: "skill", Name: "linked", Mode: "link", Path: linked, Source: src})
	lock.record(lockEntry{Domain: "d", Kind: "skill", Name: "copied", Mode: "copy", Path: copied, Source: src})

	in := &installer{projectRoot: dir, lock: lock}
	in.uninstallOne("d", "skill", "linked", linked)
	in.uninstallOne("d", "skill", "copied", copied)
	in.uninstallOne("d", "skill", "foreign", foreign)

	if _, err := os.Lstat(linked); !os.IsNotExist(err) {
		t.Fatalf("expected tracked symlink to be removed, Lstat err = %v", err)
	}
	if _, ok := lock.owns(linked); ok {
		t.Fatal("expected lockfile entry for the removed symlink to be dropped")
	}

	if _, err := os.Lstat(copied); !os.IsNotExist(err) {
		t.Fatalf("expected tracked copy-mode file to be removed, Lstat err = %v", err)
	}
	if _, ok := lock.owns(copied); ok {
		t.Fatal("expected lockfile entry for the removed copy-mode file to be dropped")
	}

	if data, err := os.ReadFile(foreign); err != nil || string(data) != "keep me" {
		t.Fatalf("foreign file should survive uninstall untouched, err=%v data=%q", err, data)
	}
}

func TestUninstallOneCopyModeRemovesRealTree(t *testing.T) {
	dir := t.TempDir()
	copiedDir := filepath.Join(dir, "copied-skill")
	if err := os.MkdirAll(filepath.Join(copiedDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(copiedDir, "nested", "SKILL.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	lock := &lockfile{}
	lock.record(lockEntry{Domain: "d", Kind: "skill", Name: "copied-skill", Mode: "copy", Path: copiedDir, Source: "/wherever"})

	in := &installer{projectRoot: dir, lock: lock}
	in.uninstallOne("d", "skill", "copied-skill", copiedDir)

	if _, err := os.Lstat(copiedDir); !os.IsNotExist(err) {
		t.Fatalf("expected copied tree to be fully removed, Lstat err = %v", err)
	}
	if _, ok := lock.owns(copiedDir); ok {
		t.Fatal("expected lockfile entry for the removed copy-mode tree to be dropped")
	}
}

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
	if err := os.MkdirAll(filepath.Join(harnessRoot, "core", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	return harnessRoot, skillDir
}

func TestRunInstallDefaultsToSymlink(t *testing.T) {
	harnessRoot, _ := fakeHarnessWithSkill(t)
	projectRoot := t.TempDir()

	if err := run([]string{"install", "skill", "myskill", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("run install: %v", err)
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

	if err := run([]string{"install", "skill", "myskill", "--copy", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("run install --copy: %v", err)
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

	if err := run([]string{"install", "skill", "myskill", "--copy", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("run install --copy: %v", err)
	}
	dst := filepath.Join(projectRoot, ".claude", "skills", "myskill")
	if _, err := os.Lstat(dst); err != nil {
		t.Fatalf("precondition: copied dir should exist: %v", err)
	}

	if err := run([]string{"uninstall", "skill", "myskill", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("run uninstall: %v", err)
	}

	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("expected copy-mode uninstall to remove the real files, Lstat err = %v", err)
	}
	lock, err := loadLockfile(projectRoot)
	if err != nil {
		t.Fatalf("loadLockfile: %v", err)
	}
	if _, ok := lock.owns(dst); ok {
		t.Fatal("expected lockfile entry to be dropped after uninstall")
	}
}

func TestUninstallDomainCopyModeRemovesOnlyThisDomainsItems(t *testing.T) {
	harnessRoot := t.TempDir()
	for _, dom := range []string{"alpha", "beta"} {
		skillDir := filepath.Join(harnessRoot, "domains", dom, "skills", dom+"-skill")
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(harnessRoot, "domains", dom, "manifest.yaml"), []byte("name: "+dom+"\ndescription: d\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(harnessRoot, "core", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}

	reg, err := loadRegistry(harnessRoot)
	if err != nil {
		t.Fatalf("loadRegistry: %v", err)
	}

	projectRoot := t.TempDir()
	lock := &lockfile{}
	in := &installer{harnessRoot: harnessRoot, projectRoot: projectRoot, mode: "copy", lock: lock}

	if err := in.installDomain(reg, "alpha", true); err != nil {
		t.Fatalf("installDomain alpha: %v", err)
	}
	if err := in.installDomain(reg, "beta", true); err != nil {
		t.Fatalf("installDomain beta: %v", err)
	}

	sharedDir := filepath.Join(projectRoot, ".claude", "skills")
	alphaItem := filepath.Join(sharedDir, "alpha-skill")
	betaItem := filepath.Join(sharedDir, "beta-skill")
	for _, p := range []string{alphaItem, betaItem} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("precondition: %s should have been copy-installed: %v", p, err)
		}
	}

	if err := in.uninstallDomain(reg, "alpha"); err != nil {
		t.Fatalf("uninstallDomain alpha: %v", err)
	}

	if _, err := os.Lstat(alphaItem); !os.IsNotExist(err) {
		t.Fatalf("expected alpha's copy-mode item to be removed, Lstat err = %v", err)
	}
	if _, err := os.Stat(betaItem); err != nil {
		t.Fatalf("beta's item in the shared directory must survive alpha's uninstall: %v", err)
	}
	if _, ok := lock.find("alpha", "skills", ""); ok {
		t.Fatal("expected alpha's whole-directory lockfile entry to be dropped")
	}
	if _, ok := lock.find("beta", "skills", ""); !ok {
		t.Fatal("expected beta's whole-directory lockfile entry to remain")
	}
}

func TestRequiresBinGating(t *testing.T) {
	const missingBin = "as-skill-test-definitely-nonexistent-binary-xyz"
	if _, err := exec.LookPath(missingBin); err == nil {
		t.Skipf("test binary %q unexpectedly found on $PATH", missingBin)
	}

	m := &manifest{Name: "testdomain", RequiresBin: []string{missingBin}, Targets: map[string]string{}}
	reg := &registry{domains: map[string]*manifest{"testdomain": m}, order: []string{"testdomain"}}

	dir := t.TempDir()

	t.Run("strict explicit domain hard-errors", func(t *testing.T) {
		in := &installer{harnessRoot: dir, projectRoot: dir, mode: "copy", lock: &lockfile{}}
		if err := in.installDomain(reg, "testdomain", true); err == nil {
			t.Fatal("expected strict install to fail when the required binary is missing")
		}
	})

	t.Run("warn-and-skip in all mode returns nil", func(t *testing.T) {
		in := &installer{harnessRoot: dir, projectRoot: dir, mode: "copy", lock: &lockfile{}}
		if err := in.installDomain(reg, "testdomain", false); err != nil {
			t.Fatalf("expected warn-and-skip install to return nil, got %v", err)
		}
		if len(in.lock.Entries) != 0 {
			t.Fatalf("expected nothing to be installed/recorded, got %d lockfile entries", len(in.lock.Entries))
		}
	})
}

func TestRequiresBinGatingSkillOwnerDomainAlwaysStrict(t *testing.T) {
	const missingBin = "as-skill-test-definitely-nonexistent-binary-xyz"
	if _, err := exec.LookPath(missingBin); err == nil {
		t.Skipf("test binary %q unexpectedly found on $PATH", missingBin)
	}

	harnessRoot := t.TempDir()
	skillDir := filepath.Join(harnessRoot, "domains", "testdomain", "skills", "myskill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: myskill\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &manifest{Name: "testdomain", RequiresBin: []string{missingBin}, Targets: map[string]string{}}
	reg := &registry{domains: map[string]*manifest{"testdomain": m}, order: []string{"testdomain"}}

	projectRoot := t.TempDir()
	in := &installer{harnessRoot: harnessRoot, projectRoot: projectRoot, mode: "copy", lock: &lockfile{}}

	if err := in.installSkill(reg, "myskill"); err == nil {
		t.Fatal("expected installSkill to hard-error when the owner domain's required binary is missing")
	}
}
