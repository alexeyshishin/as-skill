package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintHarness(t *testing.T) {
	good := "---\nname: s1\ndescription: d\n---\n"
	agd := "---\nname: ag\ndescription: d\n---\n"
	tests := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{"clean", map[string]string{
			"d/skills/s1/SKILL.md":      good + "run scripts/go.sh and .claude/agents/ag.md <!-- claude-path-ok -->",
			"d/skills/s1/scripts/go.sh": "",
			"d/agents/ag.md":            agd,
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
			"d/skills/s1/SKILL.md": good + ".claude/agents/ghost.md <!-- claude-path-ok -->",
		}, []string{"no domains/*/agents/ghost.md"}},
		{"agent in other domain ok", map[string]string{
			"d/skills/s1/SKILL.md": good + ".claude/agents/ag.md <!-- claude-path-ok -->",
			"e/agents/ag.md":       agd,
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
		{"agent name differs from filename", map[string]string{
			"d/agents/ag.md": "---\nname: other\ndescription: d\n---\n",
		}, []string{`does not match filename "ag"`}},
		{"claude-only path in skill", map[string]string{
			"d/skills/s1/SKILL.md": good + "use subagent_type here\n",
		}, []string{`Claude-only reference "subagent_type"`}},
		{"claude-only path in agent", map[string]string{
			"d/agents/ag.md": agd + "read ~/.claude/rules/x.md\n",
		}, []string{"Claude-only reference"}},
		{"claude-path marker ok", map[string]string{
			"d/skills/s1/SKILL.md":  good + "dirs .claude/agents <!-- claude-path-ok -->\nsee .claude/settings.json and .claude/skills-lock.json\n",
			"d/skills/s1/README.md": ".claude/rules\n",
			"d/agents/ag.md":        agd,
		}, nil},
		{"bare agent suffix in agent", map[string]string{
			"d/agents/d-ag.md": "---\nname: d-ag\ndescription: d\n---\nuse `ag`\n",
		}, []string{"bare agent name `ag`"}},
		{"bare suffix equal to other stem ok", map[string]string{
			"d/agents/d-ag.md": "---\nname: d-ag\ndescription: d\n---\nuse `ag`\n",
			"d/agents/ag.md":   agd,
		}, nil},
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
			got := lintHarness(root).findings
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

func TestLintReferenceSymlinks(t *testing.T) {
	good := "---\nname: s1\ndescription: d\n---\n"
	root := t.TempDir()
	dom := filepath.Join(root, "domains", "d")
	if err := os.MkdirAll(filepath.Join(dom, "skills", "s1", "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(dom, "shared"), 0o755)
	os.WriteFile(filepath.Join(dom, "skills", "s1", "SKILL.md"), []byte(good), 0o644)
	os.WriteFile(filepath.Join(dom, "shared", "ok.md"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(root, "outside.md"), []byte("x"), 0o644)
	refs := filepath.Join(dom, "skills", "s1", "references")
	os.Symlink("../../../shared/ok.md", filepath.Join(refs, "ok.md"))
	os.Symlink("../../../shared/nope.md", filepath.Join(refs, "dangling.md"))
	os.Symlink("../../../../../outside.md", filepath.Join(refs, "escape.md"))
	got := lintHarness(root).findings
	if len(got) != 2 {
		t.Fatalf("got %q", got)
	}
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "dangling.md: dangling") || !strings.Contains(joined, "escape.md: symlink escapes") {
		t.Fatalf("got %q", got)
	}
}

func TestLintSkillNameRules(t *testing.T) {
	root := t.TempDir()
	skill := func(domain, name string) {
		d := filepath.Join(root, "domains", domain, "skills", name)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: d\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	skill("a", "same")
	skill("b", "same")
	skill("a", "b")
	skill("a", "all")
	skill("a", "fine")
	joined := strings.Join(lintHarness(root).findings, "\n")
	for _, want := range []string{`skill dir name "same" duplicates`, `collides with a domain name`, `"all" is reserved`} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing finding %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, `"fine"`) {
		t.Errorf("fine flagged:\n%s", joined)
	}
}
