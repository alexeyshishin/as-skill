package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/prompt"
)

func hookAnswers(p string, doms, plats, hooks []string) map[string]any {
	ans := fullAnswers(p)
	ans["domains"] = doms
	ans["platforms"] = plats
	ans["hooks"] = hooks
	return ans
}

func readOr(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}

func TestDialogHooksRegisterOnlySelectedPerPlatform(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	all := []string{"test-gate.sh", "prettier-format.sh", "test-gate-record.sh", "memory-bank-load.sh", "memory-bank-logger.sh"}
	fp := newFake(t, hookAnswers(p, []string{"code", "core"}, []string{"claude", "cursor"}, all))
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	claude := readOr(filepath.Join(p, ".claude", "settings.json"))
	for _, n := range []string{"test-gate.sh", "prettier-format.sh", "memory-bank-load.sh", "memory-bank-logger.sh"} {
		if !strings.Contains(claude, n) {
			t.Errorf("claude settings lacks %s:\n%s", n, claude)
		}
	}
	cursor := readOr(filepath.Join(p, ".cursor", "hooks.json"))
	if cursor == "" || strings.Contains(cursor, "memory-bank-logger") {
		t.Errorf("cursor must register hooks but never memory-bank-logger:\n%s", cursor)
	}
	if !strings.Contains(cursor, "test-gate-record.sh") {
		t.Errorf("cursor lacks test-gate-record:\n%s", cursor)
	}
	if !strings.Contains(fp.summary, "warning:   memory-bank-logger.sh sends") {
		t.Errorf("summary lacks logger warning:\n%s", fp.summary)
	}
}

func TestDialogHooksSubsetOnly(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	enableDialog(t, newFake(t, hookAnswers(p, []string{"code", "core"}, []string{"claude"}, []string{"memory-bank-load.sh"})))
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	s := readOr(filepath.Join(p, ".claude", "settings.json"))
	if !strings.Contains(s, "memory-bank-load.sh") || strings.Contains(s, "test-gate") || strings.Contains(s, "memory-bank-logger") {
		t.Errorf("only memory-bank-load expected:\n%s", s)
	}
}

func TestDialogHooksEmptyRegistersNothing(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	fp := newFake(t, hookAnswers(p, []string{"code", "core"}, []string{"claude"}, []string{}))
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(p, ".claude", "settings.json")) {
		t.Error("settings.json created with no hooks selected")
	}
	if !strings.Contains(fp.summary, "hooks:     none") {
		t.Errorf("summary:\n%s", fp.summary)
	}
}

func TestDialogHooksNotAskedWithoutApplicableHooks(t *testing.T) {
	h := realHarness(t)
	for _, tc := range []struct {
		name  string
		doms  []string
		plats []string
	}{
		{"opencode only", []string{"code", "core"}, []string{"opencode"}},
		{"domain without hooks", []string{"git"}, []string{"claude"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := t.TempDir()
			ans := hookAnswers(p, tc.doms, tc.plats, []string{})
			ans["mode"] = "copy"
			fp := newFake(t, ans)
			enableDialog(t, fp)
			if _, err := run(t, "add", "--harness-root", h); err != nil {
				t.Fatal(err)
			}
			for _, id := range fp.asked {
				if id == "hooks" {
					t.Fatalf("hooks asked: %v", fp.asked)
				}
			}
			for _, f := range []string{".claude/settings.json", ".cursor/hooks.json", ".codex/hooks.json"} {
				if exists(filepath.Join(p, f)) {
					t.Errorf("%s created", f)
				}
			}
		})
	}
}

func TestAddYesNeverRegistersHooks(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	if _, err := run(t, "add", "code", "core", "--harness-root", h, "-p", p, "--copy", "-y"); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(p, ".claude", "settings.json")) || exists(filepath.Join(p, ".cursor", "hooks.json")) {
		t.Error("-y registered hooks")
	}
}

func TestDialogHooksDryRunEmptyProjectPreviews(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	enableDialog(t, newFake(t, hookAnswers(p, []string{"code", "core"}, []string{"claude"}, []string{"test-gate.sh", "memory-bank-load.sh"})))
	out, err := run(t, "add", "--harness-root", h, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"would register", "test-gate.sh (Stop)", "memory-bank-load.sh (SessionStart)"} {
		if !strings.Contains(out, w) {
			t.Errorf("stdout lacks %q:\n%s", w, out)
		}
	}
	if ents, _ := os.ReadDir(p); len(ents) != 0 {
		t.Errorf("dry-run wrote files: %v", ents)
	}
}

func TestDialogHooksBackReturnsToPreviousStepKeepingSelection(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	ans := hookAnswers(p, []string{"code", "core"}, []string{"claude"}, nil)
	ans["hooks"] = seq{[]string{"test-gate.sh"}, []string{}}
	ans["confirm"] = seq{prompt.ErrBack, true}
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "platforms", "mode", "hooks", "confirm", "hooks", "confirm")
	if got := fp.history["hooks"]; len(got) != 2 || len(got[0]) != 0 || len(got[1]) != 1 || got[1][0] != "test-gate.sh" {
		t.Errorf("preselection after back: %v", got)
	}
	if exists(filepath.Join(p, ".claude", "settings.json")) {
		t.Error("empty second selection must register nothing")
	}
}

func TestDialogBackFromHooksGoesToConflictsStep(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := hookAnswers(p, []string{"core"}, []string{"claude", "cursor"}, nil)
	ans["mode"] = "link"
	ans["guard:cursor"] = "drop"
	ans["hooks"] = seq{prompt.ErrBack, []string{}}
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "platforms", "mode", "guard:cursor", "hooks", "guard:cursor", "hooks", "confirm")
}

func TestDialogHooksOnlyOnPlatformsKeptByConflictStep(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := hookAnswers(p, []string{"core"}, []string{"claude", "cursor"}, []string{"memory-bank-load.sh"})
	ans["mode"] = "link"
	ans["guard:cursor"] = "drop"
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if lbl := fp.labelOf("hooks", "memory-bank-load.sh"); strings.Contains(lbl, "cursor") {
		t.Errorf("label offers dropped platform: %q", lbl)
	}
	if !strings.Contains(readOr(filepath.Join(p, ".claude", "settings.json")), "memory-bank-load.sh") {
		t.Error("claude not registered")
	}
	if exists(filepath.Join(p, ".cursor", "hooks.json")) {
		t.Error("dropped platform got hooks.json")
	}
}

func TestDialogHooksLabelsAndOrder(t *testing.T) {
	h := realHarness(t)
	var orders [][]string
	for i := 0; i < 2; i++ {
		fp := newFake(t, hookAnswers(t.TempDir(), []string{"code", "core"}, []string{"claude", "cursor"}, []string{}))
		enableDialog(t, fp)
		if _, err := run(t, "add", "--harness-root", h, "--dry-run"); err != nil {
			t.Fatal(err)
		}
		var vals []string
		for _, o := range fp.options["hooks"] {
			vals = append(vals, o.Value)
			if o.Selected {
				t.Errorf("%s preselected", o.Value)
			}
		}
		orders = append(orders, vals)
		if l := fp.labelOf("hooks", "memory-bank-logger.sh"); !strings.Contains(l, "SECURITY") || !strings.Contains(l, "transcript") || !strings.Contains(l, "(claude)") {
			t.Errorf("logger label: %q", l)
		}
		if l := fp.labelOf("hooks", "test-gate.sh"); l != "test-gate.sh — Stop (claude, cursor)" {
			t.Errorf("label: %q", l)
		}
	}
	if strings.Join(orders[0], ",") != strings.Join(orders[1], ",") || len(orders[0]) != 5 {
		t.Errorf("order not deterministic: %v", orders)
	}
}
