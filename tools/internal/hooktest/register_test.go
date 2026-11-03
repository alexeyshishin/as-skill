package hooktest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegisterHooksSkipsHooksWithoutClaude(t *testing.T) {
	root := project(t, ".claude")
	bare := filepath.Join(root, ".claude", "hooks", "bare.sh")
	if err := os.WriteFile(bare, []byte("#!/usr/bin/env bash\n# register: SessionStart\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	script, err := filepath.Abs("../../../domains/code/skills/code-init/scripts/register-hooks.sh")
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("bash", script, "--project", root).Output()
	if err != nil {
		t.Fatalf("register-hooks: %v", err)
	}
	for _, want := range []string{"added\ttest-gate.sh\tStop", "added\tprettier-format.sh\tPostToolUse", "added\tbare.sh\tSessionStart"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "test-gate-record.sh") {
		t.Errorf("codex/cursor-only hook offered for claude:\n%s", out)
	}
	settings, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(settings), "test-gate-record.sh") {
		t.Errorf("codex/cursor-only hook written to settings.json:\n%s", settings)
	}
}
