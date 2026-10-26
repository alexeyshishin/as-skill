package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintHarness(t *testing.T) {
	good := "---\nname: s1\ndescription: d\n---\n"
	tests := []struct {
		name  string
		files map[string]string
		want  []string // substrings, one per expected finding
	}{
		{"clean", map[string]string{
			"d/skills/s1/SKILL.md":      good + "run scripts/go.sh and .claude/agents/ag.md",
			"d/skills/s1/scripts/go.sh": "",
			"d/agents/ag.md":            good,
			"d/hooks/h.sh":              "#!/bin/sh\n# register: Stop\n",
			"d/hooks/test-gate-lib.sh":  "",
		}, nil},
		{"name mismatch and empty description", map[string]string{
			"d/skills/s1/SKILL.md": "---\nname: other\ndescription:\n---\n",
		}, []string{`does not match directory name "s1"`, `missing "description:"`}},
		{"missing name", map[string]string{
			"d/skills/s1/SKILL.md": "---\ndescription: d\n---\n",
		}, []string{`missing "name:"`}},
		{"missing script", map[string]string{
			"d/skills/s1/SKILL.md": good + "scripts/nope.py scripts/nope.py",
		}, []string{"references scripts/nope.py"}},
		{"missing agent", map[string]string{
			"d/skills/s1/SKILL.md": good + ".claude/agents/ghost.md",
		}, []string{"no domains/*/agents/ghost.md"}},
		{"agent in other domain ok", map[string]string{
			"d/skills/s1/SKILL.md": good + ".claude/agents/ag.md",
			"e/agents/ag.md":       good,
		}, nil},
		{"agent description with colon", map[string]string{
			"d/agents/ag.md": "---\nname: ag\ndescription: Scope: a/**\n---\n",
		}, []string{`"description" is an unquoted value containing ": "`}},
		{"skill description with colon", map[string]string{
			"d/skills/s1/SKILL.md": "---\nname: s1\ndescription: Use when: x\n---\n",
		}, []string{`"description" is an unquoted value containing ": "`}},
		{"folded and quoted values ok", map[string]string{
			"d/agents/ag.md": "---\nname: ag\ndescription: >\n  Scope: a/**\ntools: \"x: y\"\n---\n",
		}, nil},
		{"agent without frontmatter", map[string]string{
			"d/agents/ag.md": "# no fm\n",
		}, []string{"missing frontmatter"}},
		{"hook without register", map[string]string{
			"d/hooks/h.sh": "#!/bin/sh\n",
		}, []string{`missing "# register: <Event>"`}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range tc.files {
				p := filepath.Join(root, "domains", rel)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := lintHarness(root)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d findings %q, want %d", len(got), got, len(tc.want))
			}
			for _, w := range tc.want {
				found := false
				for _, g := range got {
					found = found || strings.Contains(g, w)
				}
				if !found {
					t.Errorf("no finding containing %q in %q", w, got)
				}
			}
		})
	}
}
