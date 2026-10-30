package harness

import (
	"os"
	"path/filepath"
	"testing"
)

func writeHarness(t *testing.T, dir string, module string) {
	t.Helper()
	for p, c := range map[string]string{
		"go.mod":                       "module " + module + "\n\ngo 1.22\n",
		"tools/main.go":                "package main\n",
		"domains/x/manifest.yaml":      "name: x\n",
		"domains/x/skills/s1/SKILL.md": "---\nname: s1\ndescription: does s1\n---\nbody\n",
		"domains/x/skills/nodoc/f":     "",
	} {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveHarnessRootAuto(t *testing.T) {
	real := func(p string) string {
		r, err := filepath.EvalSymlinks(p)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	good := real(t.TempDir())
	writeHarness(t, good, "claude-harness")
	bad := real(t.TempDir())
	writeHarness(t, bad, "other")
	unrelated := real(t.TempDir())

	if err := os.WriteFile(filepath.Join(good, "as-skill"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unrelated, "as-skill"), nil, 0o755); err != nil {
		t.Fatal(err)
	}

	old := executable
	defer func() { executable = old }()
	executable = func() (string, error) { return filepath.Join(good, "as-skill"), nil }

	chdir(t, filepath.Join(good, "domains", "x"))
	if got, err := ResolveHarnessRootAuto(""); err != nil || got != good {
		t.Fatalf("cwd walk-up: %q %v", got, err)
	}

	chdir(t, unrelated)
	if got, err := ResolveHarnessRootAuto(""); err != nil || got != good {
		t.Fatalf("exe fallback: %q %v", got, err)
	}

	chdir(t, bad)
	if got, err := ResolveHarnessRootAuto(""); err != nil || got != good {
		t.Fatalf("marker rejection should fall to exe: %q %v", got, err)
	}

	link := filepath.Join(unrelated, "bin")
	if err := os.Symlink(filepath.Join(good, "as-skill"), link); err != nil {
		t.Fatal(err)
	}
	executable = func() (string, error) { return link, nil }
	if got, err := ResolveHarnessRootAuto(""); err != nil || got != good {
		t.Fatalf("symlinked exe: %q %v", got, err)
	}

	executable = func() (string, error) { return filepath.Join(unrelated, "as-skill"), nil }
	if _, err := ResolveHarnessRootAuto(""); err == nil {
		t.Fatal("expected error with no valid harness")
	}

	if _, err := ResolveHarnessRootAuto(bad); err == nil {
		t.Fatal("flag pointing at a non-harness dir must be rejected")
	}
	if got, err := ResolveHarnessRootAuto(good); err != nil || got != good {
		t.Fatalf("valid flag: %q %v", got, err)
	}
	if got, err := ResolveHarnessRoot(bad); err != nil || got != bad {
		t.Fatalf("legacy resolver unchanged: %q %v", got, err)
	}
}

func TestSkills(t *testing.T) {
	dir := t.TempDir()
	writeHarness(t, dir, "claude-harness")
	got, err := Skills(dir, "x")
	if err != nil || len(got) != 1 || got[0].Name != "s1" || got[0].Description != "does s1" {
		t.Fatalf("%+v %v", got, err)
	}
	if got, err := Skills(dir, "nope"); err != nil || got != nil {
		t.Fatalf("%+v %v", got, err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}
