package render

import (
	"testing"

	"claude-harness/tools/internal/lockfile"
)

func TestMDCStub(t *testing.T) {
	hdr := "---\ndescription: r1 rule\nalwaysApply: true\n---\n"
	if got := string(MDCStub("r1", ".agents/rules/r1.md", "body", lockfile.ModeLink)); got != hdr+"Follow the rule in `.agents/rules/r1.md`.\n" {
		t.Errorf("link: %q", got)
	}
	if got := string(MDCStub("r1", "x", "body", lockfile.ModeCopy)); got != hdr+"body\n" {
		t.Errorf("copy: %q", got)
	}
	if got := string(MDCStub("r1", "x", "body\n", lockfile.ModeCopy)); got != hdr+"body\n" {
		t.Errorf("copy nl: %q", got)
	}
}

func TestAgentsBlockBody(t *testing.T) {
	got := AgentsBlockBody([]string{".agents/rules/a.md", ".agents/rules/b.md"})
	want := "Always follow these rules:\n- `.agents/rules/a.md`\n- `.agents/rules/b.md`\n"
	if got != want {
		t.Errorf("%q", got)
	}
}
