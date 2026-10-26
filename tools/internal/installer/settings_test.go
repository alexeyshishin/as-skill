package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/registry"
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

func keysOf(t *testing.T, raw []byte) []string {
	t.Helper()
	obj, err := parseObject(raw)
	if err != nil {
		t.Fatalf("parseObject: %v", err)
	}
	keys := make([]string, len(obj))
	for i, m := range obj {
		keys[i] = m.key
	}
	return keys
}

func TestRemoveHookRegistrationsDropsOnlyOwnedHandlers(t *testing.T) {
	names := map[string]bool{"load.sh": true, "gate.sh": true, "fmt.sh": true}
	out, removed, err := removeHookRegistrations([]byte(registeredSettings), "/proj", names)
	if err != nil {
		t.Fatalf("removeHookRegistrations: %v", err)
	}

	wantRemoved := map[string]bool{"SessionStart/load.sh": true, "Stop/gate.sh": true, "PostToolUse/fmt.sh": true}
	if len(removed) != len(wantRemoved) {
		t.Fatalf("expected %d removals, got %+v", len(wantRemoved), removed)
	}
	for _, r := range removed {
		if !wantRemoved[r.Event+"/"+r.Hook] {
			t.Fatalf("unexpected removal %+v", r)
		}
	}

	if got, want := keysOf(t, out), []string{"permissions", "hooks", "model"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("top-level key order changed: got %v want %v", got, want)
	}
	var doc struct {
		Hooks map[string][]struct {
			Matcher string
			Hooks   []struct{ Command string }
		}
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if _, ok := doc.Hooks["SessionStart"]; ok {
		t.Fatal("SessionStart had only our hook, expected the event key to be dropped")
	}
	stop := doc.Hooks["Stop"]
	if len(stop) != 1 || stop[0].Hooks[0].Command != "/usr/local/bin/foreign.sh" {
		t.Fatalf("expected only the foreign Stop entry to survive, got %+v", stop)
	}
	post := doc.Hooks["PostToolUse"]
	if len(post) != 1 || post[0].Matcher != "Edit|Bash" || len(post[0].Hooks) != 1 || post[0].Hooks[0].Command != "echo mine" {
		t.Fatalf("expected the mixed entry to keep its matcher and foreign handler only, got %+v", post)
	}
	if !strings.Contains(string(out), `"allow": [`) || !strings.HasSuffix(string(out), "}\n") {
		t.Fatalf("expected re-indented output with trailing newline, got:\n%s", out)
	}
}

func TestRemoveHookRegistrationsDropsEmptyHooksKey(t *testing.T) {
	in := `{"model":"opus","hooks":{"Stop":[{"hooks":[{"type":"command","command":"\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/gate.sh"}]}]}}`
	out, removed, err := removeHookRegistrations([]byte(in), "/proj", map[string]bool{"gate.sh": true})
	if err != nil || len(removed) != 1 {
		t.Fatalf("err=%v removed=%+v", err, removed)
	}
	if got, want := keysOf(t, out), []string{"model"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected the emptied hooks key to be dropped, got keys %v", got)
	}
}

func TestRemoveHookRegistrationsCommandForms(t *testing.T) {
	names := map[string]bool{"x.sh": true}
	cases := []struct {
		cmd  string
		owns bool
	}{
		{`\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/x.sh`, true},
		{`$CLAUDE_PROJECT_DIR/.claude/hooks/x.sh`, true},
		{`\"${CLAUDE_PROJECT_DIR}/.claude/hooks/x.sh\"`, true},
		{`/proj/.claude/hooks/x.sh`, true},
		{`.claude/hooks/x.sh`, true},
		{`./.claude/hooks/x.sh`, true},
		{`/other/.claude/hooks/x.sh`, false},
		{`$CLAUDE_PROJECT_DIR/.claude/hooks/x.sh.bak`, false},
		{`$CLAUDE_PROJECT_DIR/.claude/hooks/sub/x.sh`, false},
		{`$CLAUDE_PROJECT_DIR/.claude/hooks/y.sh`, false},
		{`$CLAUDE_PROJECT_DIR/.claude/hooks/x.sh --flag`, false},
	}
	for _, c := range cases {
		in := `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"` + c.cmd + `"}]}]}}`
		_, removed, err := removeHookRegistrations([]byte(in), "/proj", names)
		if err != nil {
			t.Fatalf("%s: %v", c.cmd, err)
		}
		if (len(removed) == 1) != c.owns {
			t.Errorf("command %s: owned=%v, want %v", c.cmd, len(removed) == 1, c.owns)
		}
	}
}

func TestRemoveHookRegistrationsLeavesUnexpectedShapesAlone(t *testing.T) {
	names := map[string]bool{"x.sh": true}
	for _, in := range []string{
		`{"model":"opus"}`,
		`{"hooks":"nope"}`,
		`{"hooks":[1,2]}`,
		`{"hooks":{"Stop":"nope"}}`,
		`{"hooks":{"Stop":[1,"a",{"hooks":"nope"},{"matcher":"x"}]}}`,
		`{"hooks":{"Stop":null}}`,
	} {
		out, removed, err := removeHookRegistrations([]byte(in), "/proj", names)
		if err != nil || len(removed) != 0 || string(out) != in {
			t.Errorf("%s: expected untouched input, got out=%q removed=%+v err=%v", in, out, removed, err)
		}
	}
	for _, in := range []string{`{oops`, `[]`, `"str"`, ``} {
		if _, _, err := removeHookRegistrations([]byte(in), "/proj", names); err == nil {
			t.Errorf("%q: expected an error", in)
		}
	}
}

func TestRemoveHookRegistrationsKeepsKeyBytes(t *testing.T) {
	in := `{"env":{"A&B":"<x>"},"hooks":{"Stop":[{"hooks":[{"type":"command","command":".claude/hooks/x.sh"}]}]},"n":1.50}`
	out, _, err := removeHookRegistrations([]byte(in), "/proj", map[string]bool{"x.sh": true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"A&B": "<x>"`, `"n": 1.50`} {
		if !strings.Contains(string(out), want) {
			t.Errorf("expected %q to survive verbatim, got:\n%s", want, out)
		}
	}
}

func hooksHarness(t *testing.T) (*registry.Registry, string) {
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
	reg, err := registry.LoadRegistry(harnessRoot)
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
			in := &Installer{HarnessRoot: harnessRoot, ProjectRoot: projectRoot, Mode: mode, Lock: lock}
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
