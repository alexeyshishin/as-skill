package platform

import (
	"reflect"
	"testing"
)

func TestPathConsts(t *testing.T) {
	for got, want := range map[string]string{
		ClaudeHome: ".claude", ClaudeSkillsLoc: ".claude/skills", PromptsDir: ".agents/prompts",
		AgentsRulesDir: ".agents/rules", AgentsRoot: ".agents",
	} {
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	}
}

func TestAllSkillDirs(t *testing.T) {
	want := []string{".agents/skills", ".claude/skills", ".codex/skills", ".cursor/skills", ".opencode/skills"}
	if got := AllSkillDirs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestLinkRoots(t *testing.T) {
	want := []string{".agents", ".codex", ".cursor", ".opencode"}
	if got := LinkRoots(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for _, r := range want {
		if !IsLinkRoot(r) {
			t.Errorf("%s not link root", r)
		}
	}
	for _, r := range []string{".claude", "", "x"} {
		if IsLinkRoot(r) {
			t.Errorf("%q link root", r)
		}
	}
}

func TestCapGap(t *testing.T) {
	const st = "agents: frontmatter edits not live (doctor reports STALE); hooks: "
	for id, want := range map[ID]string{
		Claude: "", Codex: st + "needs trust via /hooks", Cursor: st + "stop cannot block", OpenCode: st + "unsupported",
	} {
		if got := CapGap(id); got != want {
			t.Errorf("%s: %q", id, got)
		}
	}
}

func TestRootOf(t *testing.T) {
	for in, want := range map[string]string{"a/b": "a", "a": "a", ".agents/skills": ".agents", "": ""} {
		if got := RootOf(in); got != want {
			t.Errorf("%q: %q want %q", in, got, want)
		}
	}
}
