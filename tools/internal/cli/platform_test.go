package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/lint"
)

func capture(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		done <- b.String()
	}()
	runErr := fn()
	w.Close()
	os.Stdout = old
	return <-done, runErr
}

func agentHarness(t *testing.T) (harness, agentSrc string) {
	t.Helper()
	harness, _ = fakeHarnessWithSkill(t)
	agentSrc = filepath.Join(harness, "domains", "testdomain", "agents", "helper.md")
	if err := os.MkdirAll(filepath.Dir(agentSrc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentSrc, []byte("---\nname: helper\ndescription: d\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return harness, agentSrc
}

func TestListPlatformsMatrix(t *testing.T) {
	out, err := capture(t, func() error { return runCLI([]string{"list", "platforms"}) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"PLATFORM", "claude", "codex", "cursor", "opencode", ".agents/skills", ".cursor/rules", "render", "block", "unsupported", "verified 2026-10-27", "unverified"} {
		if !strings.Contains(out, want) {
			t.Errorf("list platforms missing %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "\n") != 17 {
		t.Errorf("want header + 16 rows:\n%s", out)
	}
}

func TestUnknownPlatformFailsListingValid(t *testing.T) {
	err := runCLI([]string{"install", "all", "--platform", "vim", "--project", t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "claude, codex, cursor, opencode") {
		t.Fatalf("got %v", err)
	}
}

func TestHooksVerbDryRunAndOpenCode(t *testing.T) {
	project := t.TempDir()
	hooks := filepath.Join(project, ".codex", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "gate.sh"), []byte("#!/bin/sh\n# register: Stop\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := capture(t, func() error {
		return runCLI([]string{"hooks", "enable", "gate", "--platform", "codex", "--project", project, "--dry-run"})
	})
	if err != nil || !strings.Contains(out, "would register") {
		t.Fatalf("dry-run: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(project, ".codex", "hooks.json")); err == nil {
		t.Fatal("dry-run wrote file")
	}
	out, err = capture(t, func() error {
		return runCLI([]string{"hooks", "enable", "all", "--platform", "codex", "--project", project})
	})
	if err != nil || !strings.Contains(out, "UNVERIFIED") {
		t.Fatalf("enable: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(project, ".codex", "hooks.json")); err != nil {
		t.Fatal("codex enable did not write hooks.json")
	}
	out, err = capture(t, func() error {
		return runCLI([]string{"hooks", "enable", "all", "--platform", "opencode", "--project", project})
	})
	if err != nil || !strings.Contains(out, "unsupported") {
		t.Fatalf("opencode: %v\n%s", err, out)
	}
}

func installPlatforms(t *testing.T, harness, project string, extra ...string) {
	t.Helper()
	args := append([]string{"install", "domain", "testdomain", "--harness-root", harness, "--project", project}, extra...)
	if _, err := capture(t, func() error { return runCLI(args) }); err != nil {
		t.Fatal(err)
	}
}

func doctorCmd(t *testing.T, harness, project string, extra ...string) (string, error) {
	args := append([]string{"doctor", "--harness-root", harness, "--project", project}, extra...)
	return capture(t, func() error { return runCLI(args) })
}

func TestDoctorStaleAndDrifted(t *testing.T) {
	harness, agentSrc := agentHarness(t)
	project := t.TempDir()
	installPlatforms(t, harness, project, "--platform", "codex", "--copy")
	out, err := doctorCmd(t, harness, project)
	if err != nil {
		t.Fatalf("clean doctor failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(agentSrc, []byte("---\nname: helper\ndescription: changed\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = doctorCmd(t, harness, project)
	if err == nil || !strings.Contains(out, "STALE") {
		t.Fatalf("want STALE: %v\n%s", err, out)
	}
	toml := filepath.Join(project, ".codex", "agents", "helper.toml")
	data, _ := os.ReadFile(toml)
	if err := os.WriteFile(toml, append(data, []byte("# hand edit\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = doctorCmd(t, harness, project)
	if err == nil || !strings.Contains(out, "DRIFTED") {
		t.Fatalf("want DRIFTED: %v\n%s", err, out)
	}
}

func TestDoctorDuplicateSkillIsWarningOnly(t *testing.T) {
	harness, _ := agentHarness(t)
	project := t.TempDir()
	installPlatforms(t, harness, project, "--platform", "claude,codex,cursor", "--copy")
	out, err := doctorCmd(t, harness, project)
	if err != nil {
		t.Fatalf("warnings must not fail doctor: %v\n%s", err, out)
	}
	if !strings.Contains(out, "DUPLICATE-SKILL") || !strings.Contains(out, `"myskill"`) || !strings.Contains(out, "cursor sees it twice") {
		t.Fatalf("missing duplicate warning:\n%s", out)
	}
}

func TestStatusPlatformFilterAndColumn(t *testing.T) {
	harness, _ := agentHarness(t)
	project := t.TempDir()
	installPlatforms(t, harness, project, "--platform", "codex,cursor", "--copy")
	out, err := capture(t, func() error { return runCLI([]string{"status", "--project", project}) })
	if err != nil || !strings.Contains(out, "PLATFORMS") || !strings.Contains(out, "codex,cursor") {
		t.Fatalf("status: %v\n%s", err, out)
	}
	out, err = capture(t, func() error { return runCLI([]string{"status", "--project", project, "--platform", "cursor"}) })
	if err != nil || strings.Contains(out, ".codex/agents") || !strings.Contains(out, ".cursor/agents") {
		t.Fatalf("filtered status: %v\n%s", err, out)
	}
}

func TestUninstallPlatformFlag(t *testing.T) {
	harness, _ := agentHarness(t)
	project := t.TempDir()
	installPlatforms(t, harness, project, "--platform", "codex,cursor", "--copy")
	if _, err := capture(t, func() error {
		return runCLI([]string{"uninstall", "domain", "testdomain", "--harness-root", harness, "--project", project, "--platform", "cursor"})
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, ".agents", "skills", "myskill")); err != nil {
		t.Fatal("shared skill removed")
	}
	if _, err := os.Stat(filepath.Join(project, ".cursor", "agents", "helper.md")); err == nil {
		t.Fatal("cursor agent kept")
	}
}

func TestLintAgentNames(t *testing.T) {
	harness, _ := agentHarness(t)
	other := filepath.Join(harness, "domains", "testdomain", "agents", "dup.md")
	if err := os.WriteFile(other, []byte("---\nname: helper\ndescription: d\n---\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := lint.Harness(harness)
	if !strings.Contains(findingsText(res.Findings), `agent name "helper" duplicates`) {
		t.Fatalf("no duplicate finding: %v", res.Findings)
	}
	if err := os.WriteFile(other, []byte("---\nname: explorer\ndescription: d\n---\nx\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(findingsText(lint.Harness(harness).Findings), `reserved on codex`) {
		t.Fatal("no reserved finding")
	}
}

func TestDoctorUnverifiedFiresForNonClaude(t *testing.T) {
	harness, _ := agentHarness(t)
	project := t.TempDir()
	installPlatforms(t, harness, project, "--platform", "codex", "--copy")
	out, err := doctorCmd(t, harness, project)
	if err != nil {
		t.Fatalf("warnings must not fail doctor: %v\n%s", err, out)
	}
	if !strings.Contains(out, "UNVERIFIED") || !strings.Contains(out, "codex agents capability is unverified") {
		t.Fatalf("no UNVERIFIED warning:\n%s", out)
	}
	claude := t.TempDir()
	installPlatforms(t, harness, claude, "--copy")
	out, _ = doctorCmd(t, harness, claude)
	if strings.Contains(out, "UNVERIFIED") {
		t.Fatalf("claude must be verified:\n%s", out)
	}
}

func TestDoctorCodexHooksTrustWarning(t *testing.T) {
	harness, _ := agentHarness(t)
	project := t.TempDir()
	installPlatforms(t, harness, project, "--platform", "codex", "--copy")
	out, _ := doctorCmd(t, harness, project)
	if strings.Contains(out, "need trust") {
		t.Fatalf("unexpected trust warning:\n%s", out)
	}
	cfg := filepath.Join(project, ".codex", "hooks.json")
	if err := os.WriteFile(cfg, []byte(`{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"./.codex/hooks/gate.sh"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := doctorCmd(t, harness, project)
	if err != nil {
		t.Fatalf("warning must not fail doctor: %v\n%s", err, out)
	}
	if !strings.Contains(out, "codex hooks need trust: review in /hooks") {
		t.Fatalf("no trust warning:\n%s", out)
	}
}

func findingsText(fs []lint.Finding) string {
	var out []string
	for _, f := range fs {
		out = append(out, f.String())
	}
	return strings.Join(out, "\n")
}
