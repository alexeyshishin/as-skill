package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hooksRun(t *testing.T, h string, args ...string) (string, error) {
	return run(t, append([]string{"hooks"}, append(args, "--harness-root", h)...)...)
}

func TestHooksDomainDryRunFromSource(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	out, err := hooksRun(t, h, "enable", "--domain", "code,core", "-a", "claude", "--dry-run", "-p", p)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"test-gate.sh", "prettier-format.sh", "memory-bank-load.sh", "memory-bank-logger.sh"} {
		if !strings.Contains(out, "would register") || !strings.Contains(out, n) {
			t.Errorf("missing %s in:\n%s", n, out)
		}
	}
	if strings.Contains(out, "test-gate-record.sh") {
		t.Errorf("test-gate-record must not register on claude:\n%s", out)
	}
	if es, _ := os.ReadDir(p); len(es) != 0 {
		t.Errorf("dry run wrote %v", es)
	}
}

func TestHooksDomainCursorAndOpenCode(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	out, err := hooksRun(t, h, "enable", "--domain", "code,core", "-a", "cursor", "--dry-run", "-p", p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "test-gate-record.sh") || strings.Contains(out, "memory-bank-logger.sh") {
		t.Errorf("cursor set wrong:\n%s", out)
	}
	out, err = hooksRun(t, h, "enable", "--domain", "code,core", "-a", "opencode", "--dry-run", "-p", p)
	if err != nil || !strings.Contains(out, "hooks unsupported on opencode") {
		t.Errorf("opencode: %v\n%s", err, out)
	}
}

func TestHooksDomainExplicitName(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	if _, err := hooksRun(t, h, "enable", "memory-bank-logger", "--domain", "core", "-a", "cursor", "--dry-run", "-p", p); err == nil || !strings.Contains(err.Error(), "memory-bank-logger.sh") {
		t.Errorf("want claude-only error, got %v", err)
	}
	if _, err := hooksRun(t, h, "enable", "memory-bank-logger", "--domain", "core", "-a", "claude", "--dry-run", "-p", p); err != nil {
		t.Errorf("claude: %v", err)
	}
	if _, err := hooksRun(t, h, "enable", "test-gate", "--domain", "core", "-a", "claude", "--dry-run", "-p", p); err == nil {
		t.Error("hook from another domain must fail")
	}
}

func TestHooksDomainUnknown(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	_, err := hooksRun(t, h, "enable", "--domain", "nope", "-a", "claude", "--dry-run", "-p", p)
	if err == nil || !strings.Contains(err.Error(), "domains:") || !strings.Contains(err.Error(), "core") {
		t.Errorf("got %v", err)
	}
}

func TestHooksDomainRealEnableDisable(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	if _, err := run(t, "add", "code", "core", "-a", "claude", "--copy", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(p, ".claude", "settings.json")
	if _, err := hooksRun(t, h, "enable", "--domain", "core", "-a", "claude", "-p", p); err != nil {
		t.Fatal(err)
	}
	s := readOr(sp)
	if !strings.Contains(s, "memory-bank-load.sh") || !strings.Contains(s, "memory-bank-logger.sh") {
		t.Errorf("core hooks not registered:\n%s", s)
	}
	if strings.Contains(s, "test-gate.sh") || strings.Contains(s, "prettier-format.sh") {
		t.Errorf("code hooks leaked:\n%s", s)
	}
	if _, err := hooksRun(t, h, "disable", "--domain", "core", "-a", "claude", "-p", p); err != nil {
		t.Fatal(err)
	}
	if s := readOr(sp); strings.Contains(s, "memory-bank") {
		t.Errorf("not removed:\n%s", s)
	}
}

func TestHooksNoDomainDryRunHint(t *testing.T) {
	p := t.TempDir()
	_, err := run(t, "hooks", "enable", "test-gate", "--dry-run", "-a", "claude", "-p", p)
	if err == nil || !strings.Contains(err.Error(), "reading ") || !strings.Contains(err.Error(), "--domain") {
		t.Errorf("got %v", err)
	}
	if _, err := run(t, "hooks", "enable", "-a", "claude", "-p", p); err == nil || !strings.Contains(err.Error(), "needs a hook name") {
		t.Errorf("got %v", err)
	}
}

func TestLegacyParseUnaffectedByDomainFlag(t *testing.T) {
	pos, o, err := parseInstallArgs([]string{"domains", "code", "--project", "x", "-a", "claude"})
	if err != nil || len(pos) != 2 || o.project != "x" || len(o.domains) != 0 || o.harnessSet {
		t.Errorf("%v %v %+v", pos, err, o)
	}
}
