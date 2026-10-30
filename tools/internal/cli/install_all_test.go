package cli

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/harness"
)

func realHarness(t *testing.T) string {
	t.Helper()
	h, err := harness.ResolveHarnessRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func sameContent(t *testing.T, got, want string) {
	t.Helper()
	g, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	w, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(g) != string(w) {
		t.Fatalf("%s differs from %s", got, want)
	}
}

func TestInstallAllCopyAllPlatforms(t *testing.T) {
	harness := realHarness(t)
	t.Setenv("OBSIDIAN_VAULT", t.TempDir())
	project := t.TempDir()
	args := []string{"install", "all", "--copy", "--platform", "claude,codex,cursor,opencode", "--harness-root", harness, "--project", project}
	if _, err := capture(t, func() error { return runCLI(args) }); err != nil {
		t.Fatal(err)
	}
	err := filepath.WalkDir(project, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			t.Errorf("symlink remains: %s", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	skills := filepath.Join(project, ".claude", "skills")
	for _, c := range []struct{ skill, domain, rule, agent string }{
		{"git-conventional-commits", "git", "git-conventions", ""},
		{"code-plan", "code", "", "code-planner"},
	} {
		ref := filepath.Join(skills, c.skill, "references")
		src := filepath.Join(harness, "domains", c.domain)
		if c.rule != "" {
			f := filepath.Join(ref, "rules", c.rule+".md")
			if st, err := os.Lstat(f); err != nil || !st.Mode().IsRegular() {
				t.Fatalf("%s not a regular file: %v", f, err)
			}
			sameContent(t, f, filepath.Join(src, "rules", c.rule+".md"))
		}
		if c.agent != "" {
			f := filepath.Join(ref, "agents", c.agent+".md")
			if st, err := os.Lstat(f); err != nil || !st.Mode().IsRegular() {
				t.Fatalf("%s not a regular file: %v", f, err)
			}
			sameContent(t, f, filepath.Join(src, "agents", c.agent+".md"))
		}
	}
	out, _ := doctor(t, harness, project)
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "as-skill: doctor:") || strings.Contains(line, "warning:") {
			continue
		}
		if !strings.Contains(line, "DRIFTED") || !strings.Contains(line, "/ — recomputed hash") {
			t.Errorf("unexpected doctor failure: %s", line)
		}
	}
}

func TestInstallLinkReferencesResolve(t *testing.T) {
	harness := realHarness(t)
	project := t.TempDir()
	gitInitDir(t, project)
	if err := os.WriteFile(filepath.Join(project, ".gitignore"), []byte(".agents\n.codex\n.cursor\n.opencode\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{"install", "domains", "git", "code", "--platform", "claude,codex", "--harness-root", harness, "--project", project}
	if _, err := capture(t, func() error { return runCLI(args) }); err != nil {
		t.Fatal(err)
	}
	skill := filepath.Join(project, ".claude", "skills", "git-conventional-commits")
	if st, err := os.Lstat(skill); err != nil || st.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("skill not a symlink: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(skill, "references", "rules", "git-conventions.md"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := os.ReadFile(filepath.Join(harness, "domains", "git", "rules", "git-conventions.md"))
	if string(got) != string(want) {
		t.Fatal("rule content mismatch through symlinked skill")
	}
	if _, err := os.ReadFile(filepath.Join(project, ".claude", "skills", "code-plan", "references", "agents", "code-planner.md")); err != nil {
		t.Fatal(err)
	}
}

func gitInitDir(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
}
