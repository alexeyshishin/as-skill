package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

// planner tests write only source fixtures; planning itself must not touch the project.
func srcTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func newPlanInstaller(mode lockfile.Mode, ids ...platform.ID) (*Installer, string) {
	proj := filepath.Join("/nonexistent", "proj")
	return &Installer{HarnessRoot: "/nonexistent/h", ProjectRoot: proj, Mode: mode, Lock: &lockfile.Lockfile{}, Platforms: ids}, proj
}

func TestPlanKindAgentsCodexCopy(t *testing.T) {
	src := srcTree(t, map[string]string{"a.md": "---\nname: alpha\n---\nbody"})
	in, proj := newPlanInstaller(lockfile.ModeCopy, platform.Codex)
	p, err := in.planKind(&harness.Manifest{}, "d", lockfile.KindAgents, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 1 {
		t.Fatalf("ops = %+v", p.Ops)
	}
	op := p.Ops[0]
	if op.Type != OpRender || op.Kind != lockfile.KindAgents || op.Name != "a.md" ||
		!strings.HasPrefix(op.Dst, filepath.Join(proj, ".codex", "agents")) || !strings.Contains(op.Dst, "alpha") {
		t.Fatalf("op = %+v", op)
	}
	if len(p.Warnings) != 1 || p.Warnings[0].Stream != Stdout || p.Warnings[0].Before != 1 ||
		!strings.Contains(p.Warnings[0].Text, "(1 file)") {
		t.Fatalf("warnings = %+v", p.Warnings)
	}
}

func TestPlanKindAgentsLinkAddsPromptOp(t *testing.T) {
	src := srcTree(t, map[string]string{"a.md": "body"})
	in, proj := newPlanInstaller(lockfile.ModeLink, platform.OpenCode)
	p, err := in.planKind(&harness.Manifest{}, "d", lockfile.KindAgents, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 || p.Ops[0].Type != OpLink || p.Ops[0].Kind != lockfile.KindPrompts ||
		p.Ops[0].Dst != filepath.Join(proj, platform.PromptsDir, "a.md") || p.Ops[1].Type != OpRender {
		t.Fatalf("ops = %+v", p.Ops)
	}
}

func TestPlanKindRulesCursorMDCAndBlock(t *testing.T) {
	src := srcTree(t, map[string]string{"r.md": "rule"})
	in, proj := newPlanInstaller(lockfile.ModeLink, platform.Cursor)
	p, err := in.planKind(&harness.Manifest{}, "d", lockfile.KindRules, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 || p.Ops[0].Type != OpLink || p.Ops[0].Dst != filepath.Join(proj, platform.AgentsRulesDir, "r.md") ||
		p.Ops[1].Type != OpRender || filepath.Ext(p.Ops[1].Dst) != ".mdc" {
		t.Fatalf("cursor ops = %+v", p.Ops)
	}

	in, proj = newPlanInstaller(lockfile.ModeCopy, platform.Codex)
	p, err = in.planKind(&harness.Manifest{}, "d", lockfile.KindRules, src)
	if err != nil {
		t.Fatal(err)
	}
	last := p.Ops[len(p.Ops)-1]
	if last.Type != OpBlock || last.Dst != filepath.Join(proj, agentsMD) {
		t.Fatalf("codex ops = %+v", p.Ops)
	}
	if p.Ops[0].Type != OpCopy || !p.Ops[0].Item {
		t.Fatalf("codex first op = %+v", p.Ops[0])
	}
}

func TestPlanKindHooks(t *testing.T) {
	src := srcTree(t, map[string]string{"h.sh": "#!/bin/sh\n"})
	in, _ := newPlanInstaller(lockfile.ModeCopy, platform.OpenCode)
	p, err := in.planKind(&harness.Manifest{}, "d", lockfile.KindHooks, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 0 || len(p.Warnings) != 1 || !strings.Contains(p.Warnings[0].Text, "hooks unsupported on opencode") {
		t.Fatalf("opencode plan = %+v", p)
	}

	in, _ = newPlanInstaller(lockfile.ModeCopy, platform.Codex)
	p, err = in.planKind(&harness.Manifest{}, "d", lockfile.KindHooks, src)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(p.Ops); n != 2 || p.Ops[n-1].Type != OpHook {
		t.Fatalf("codex ops = %+v", p.Ops)
	}
	last := p.Warnings[len(p.Warnings)-1]
	if last.Before != 2 || !strings.Contains(last.Text, "hooks enable") {
		t.Fatalf("warnings = %+v", p.Warnings)
	}
}

func TestPlanKindSkillsMixedClaudeCodexDupWarning(t *testing.T) {
	src := srcTree(t, map[string]string{"s/SKILL.md": "x"})
	in, proj := newPlanInstaller(lockfile.ModeCopy, platform.Claude, platform.Codex, platform.OpenCode)
	p, err := in.planKind(&harness.Manifest{}, "d", lockfile.KindSkills, src)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Ops) != 2 || p.Ops[0].Dst != filepath.Join(proj, ".agents", "skills", "s") || !p.Ops[0].Item ||
		p.Ops[1].Dst != filepath.Join(proj, ".claude", "skills") || p.Ops[1].Item {
		t.Fatalf("ops = %+v", p.Ops)
	}
	if len(p.Warnings) == 0 || p.Warnings[0].Before != 0 || p.Warnings[0].Stream != Stderr ||
		!strings.Contains(p.Warnings[0].Text, "DUPLICATE-SKILL") {
		t.Fatalf("warnings = %+v", p.Warnings)
	}
	p, _ = in.planKind(&harness.Manifest{}, "d", lockfile.KindSkills, src)
	for _, w := range p.Warnings {
		if strings.Contains(w.Text, "DUPLICATE-SKILL") {
			t.Fatalf("dup warned twice: %+v", p.Warnings)
		}
	}
}

func TestPlanKindReservedAgentIsPartial(t *testing.T) {
	src := srcTree(t, map[string]string{"a.md": "body"})
	in, _ := newPlanInstaller(lockfile.ModeCopy, platform.Codex)
	pf, _ := platform.Get(platform.Codex)
	if len(pf.ReservedAgentNames) == 0 {
		t.Skip("no reserved names")
	}
	src = srcTree(t, map[string]string{"a.md": "body", pf.ReservedAgentNames[0] + ".md": "x"})
	p, err := in.planKind(&harness.Manifest{}, "d", lockfile.KindAgents, src)
	if err == nil || !strings.Contains(err.Error(), "reserved") || len(p.Ops) != 1 {
		t.Fatalf("err=%v ops=%+v", err, p.Ops)
	}
}

func TestPlanGapsOnce(t *testing.T) {
	in, _ := newPlanInstaller(lockfile.ModeCopy, platform.Codex, platform.Cursor, platform.OpenCode)
	first := in.planGaps()
	for _, w := range first.Warnings {
		if w.Stream != Stderr || !strings.HasPrefix(w.Text, "as-skill: note: ") {
			t.Fatalf("warning = %+v", w)
		}
	}
	if len(in.planGaps().Warnings) != 0 {
		t.Fatal("gaps planned twice")
	}
}
