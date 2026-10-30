package cli

import (
	"os"
	"path/filepath"
	"strings"
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
	markHarness(t, harnessRoot)
	return harnessRoot, skillDir
}

func markHarness(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module claude-harness\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tools", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunInstallDefaultsToSymlink(t *testing.T) {
	harnessRoot, _ := fakeHarnessWithSkill(t)
	projectRoot := t.TempDir()

	if err := runCLI([]string{"install", "skill", "myskill", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
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

	if err := runCLI([]string{"install", "skill", "myskill", "--copy", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
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

	if err := runCLI([]string{"install", "skill", "myskill", "--copy", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
		t.Fatalf("Run install --copy: %v", err)
	}
	dst := filepath.Join(projectRoot, ".claude", "skills", "myskill")
	if _, err := os.Lstat(dst); err != nil {
		t.Fatalf("precondition: copied dir should exist: %v", err)
	}

	if err := runCLI([]string{"uninstall", "skill", "myskill", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
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

	seed := &lockfile.Lockfile{}
	seed.Record(lockfile.Entry{
		Domain: "core", Kind: "skill", Name: "caveman",
		Mode: "link", Path: dst, Source: skillDir,
		InstalledAt: time.Now(),
	})
	if err := seed.Save(projectRoot); err != nil {
		t.Fatal(err)
	}

	if err := runCLI([]string{"install", "domain", "core", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
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

	if err := runCLI([]string{"uninstall", "domain", "core", "--harness-root", harnessRoot, "--project", projectRoot}); err != nil {
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

func TestCheckSkillSpec(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "SKILL.md")
	cases := map[string]struct {
		src  string
		want string
	}{
		"ok":        {"---\nname: my-skill\ndescription: d\nargument-hint: \"[x]\"\nallowed-tools: Read, Write\nmetadata:\n  k: v\n---\n", ""},
		"extra key": {"---\nname: my-skill\ndescription: d\nmodel: x\n---\n", "line 4"},
		"list tool": {"---\nname: my-skill\ndescription: d\nallowed-tools:\n  - Read\n---\n", "allowed-tools must be a string"},
		"bad name":  {"---\nname: My_Skill\ndescription: d\n---\n", "does not match directory"},
		"long desc": {"---\nname: my-skill\ndescription: " + strings.Repeat("a", 1025) + "\n---\n", "max 1024"},
	}
	for name, c := range cases {
		if err := os.WriteFile(p, []byte(c.src), 0o644); err != nil {
			t.Fatal(err)
		}
		got := strings.Join(checkSkillSpec(p), "\n")
		if c.want == "" && got != "" || c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want %q", name, got, c.want)
		}
	}
}
