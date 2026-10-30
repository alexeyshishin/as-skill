package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
)

const registeredSettings = `{
  "permissions": {"allow": ["Bash(ls)"]},
  "hooks": {
    "SessionStart": [
      {"matcher": "startup|clear", "hooks": [{"type": "command", "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/load.sh"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "/usr/local/bin/foreign.sh"}]},
      {"hooks": [{"type": "command", "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/gate.sh"}]}
    ],
    "PostToolUse": [
      {"matcher": "Edit|Bash", "hooks": [
        {"type": "command", "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/fmt.sh"},
        {"type": "command", "command": "echo mine"}
      ]}
    ]
  },
  "model": "opus"
}`

func hooksHarness(t *testing.T) (*harness.Registry, string) {
	t.Helper()
	harnessRoot := t.TempDir()
	hooksDir := filepath.Join(harnessRoot, "domains", "alpha", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"gate.sh", "load.sh", "fmt.sh"} {
		if err := os.WriteFile(filepath.Join(hooksDir, f), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(harnessRoot, "domains", "alpha", "manifest.yaml"), []byte("name: alpha\ndescription: d\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := harness.LoadRegistry(harnessRoot)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	return reg, harnessRoot
}

func TestUninstallDomainUnregistersItsHooksFromSettings(t *testing.T) {
	for _, mode := range []string{"copy", "link"} {
		t.Run(mode, func(t *testing.T) {
			reg, harnessRoot := hooksHarness(t)
			projectRoot := t.TempDir()
			lock := &lockfile.Lockfile{}
			in := &Installer{HarnessRoot: harnessRoot, ProjectRoot: projectRoot, Mode: lockfile.Mode(mode), Lock: lock}
			if err := in.InstallDomain(reg, "alpha", true); err != nil {
				t.Fatalf("InstallDomain: %v", err)
			}
			settings := filepath.Join(projectRoot, ".claude", "settings.json")
			if err := os.WriteFile(settings, []byte(registeredSettings), 0o640); err != nil {
				t.Fatal(err)
			}

			dry := &Installer{
				HarnessRoot: harnessRoot, ProjectRoot: projectRoot, DryRun: true,
				Lock: &lockfile.Lockfile{Entries: append([]lockfile.Entry(nil), lock.Entries...)},
			}
			if err := dry.UninstallDomain(reg, "alpha"); err != nil {
				t.Fatalf("UninstallDomain dry-run: %v", err)
			}
			if data, _ := os.ReadFile(settings); string(data) != registeredSettings {
				t.Fatalf("--dry-run must not touch settings.json, got:\n%s", data)
			}

			if err := in.UninstallDomain(reg, "alpha"); err != nil {
				t.Fatalf("UninstallDomain: %v", err)
			}
			data, err := os.ReadFile(settings)
			if err != nil {
				t.Fatal(err)
			}
			for _, gone := range []string{"load.sh", "gate.sh", "fmt.sh"} {
				if strings.Contains(string(data), gone) {
					t.Errorf("expected the %s registration to be gone, got:\n%s", gone, data)
				}
			}
			for _, kept := range []string{"foreign.sh", "echo mine", `"model": "opus"`, `"allow"`} {
				if !strings.Contains(string(data), kept) {
					t.Errorf("expected %q to survive, got:\n%s", kept, data)
				}
			}
			if fi, _ := os.Stat(settings); fi.Mode().Perm() != 0o640 {
				t.Errorf("expected settings.json mode 0640 to be preserved, got %v", fi.Mode().Perm())
			}
			if left, _ := filepath.Glob(filepath.Join(projectRoot, ".claude", "settings.json.*")); len(left) != 0 {
				t.Errorf("temp files left behind: %v", left)
			}
		})
	}
}

func TestUninstallDomainKeepsRegistrationOfUntrackedHook(t *testing.T) {
	reg, harnessRoot := hooksHarness(t)
	projectRoot := t.TempDir()
	hooks := filepath.Join(projectRoot, ".claude", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "gate.sh"), []byte("mine\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(projectRoot, ".claude", "settings.json")
	if err := os.WriteFile(settings, []byte(registeredSettings), 0o644); err != nil {
		t.Fatal(err)
	}

	in := &Installer{HarnessRoot: harnessRoot, ProjectRoot: projectRoot, Mode: "copy", Lock: &lockfile.Lockfile{}}
	if err := in.UninstallDomain(reg, "alpha"); err != nil {
		t.Fatalf("UninstallDomain: %v", err)
	}
	if data, _ := os.ReadFile(settings); string(data) != registeredSettings {
		t.Fatalf("hooks not tracked by as-skill must keep their registration, got:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(hooks, "gate.sh")); err != nil {
		t.Fatalf("untracked hook file must survive: %v", err)
	}
}

func TestUninstallDomainWithoutSettingsOrBrokenSettingsStillUninstalls(t *testing.T) {
	for name, content := range map[string]*string{"absent": nil, "broken": ptr("{oops")} {
		t.Run(name, func(t *testing.T) {
			reg, harnessRoot := hooksHarness(t)
			projectRoot := t.TempDir()
			in := &Installer{HarnessRoot: harnessRoot, ProjectRoot: projectRoot, Mode: "copy", Lock: &lockfile.Lockfile{}}
			if err := in.InstallDomain(reg, "alpha", true); err != nil {
				t.Fatal(err)
			}
			settings := filepath.Join(projectRoot, ".claude", "settings.json")
			if content != nil {
				if err := os.WriteFile(settings, []byte(*content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := in.UninstallDomain(reg, "alpha"); err != nil {
				t.Fatalf("UninstallDomain: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(projectRoot, ".claude", "hooks", "gate.sh")); !os.IsNotExist(err) {
				t.Fatalf("hook file should be gone, Lstat err = %v", err)
			}
			data, err := os.ReadFile(settings)
			switch {
			case content == nil && !os.IsNotExist(err):
				t.Fatalf("settings.json must not be created, err=%v", err)
			case content != nil && string(data) != *content:
				t.Fatalf("broken settings.json must be left as is, got %q", data)
			}
		})
	}
}

func ptr(s string) *string { return &s }
