package lockfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModeKindConstValues(t *testing.T) {
	for got, want := range map[string]string{
		string(ModeLink): "link", string(ModeCopy): "copy", string(ModeRender): "render", string(ModeBlock): "block",
		string(KindRules): "rules", string(KindSkills): "skills", string(KindSkill): "skill", string(KindAgents): "agents",
		string(KindHooks): "hooks", string(KindPrompts): "prompts", string(KindInstructions): "instructions",
	} {
		if got != want {
			t.Errorf("const %q != %q", got, want)
		}
	}
}

func TestUnknownModeKindRoundTrip(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{"entries":[{"domain":"x","kind":"strange","name":"n","mode":"weird","path":"p","source":"s","platforms":["claude"],"installed_at":"2024-01-01T00:00:00Z"}]}`
	if err := os.WriteFile(filepath.Join(dir, "skills-lock.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	lf, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(lf.Entries) != 1 || lf.Entries[0].Mode != "weird" || lf.Entries[0].Kind != "strange" {
		t.Fatalf("unexpected load: %+v", lf.Entries)
	}
	if err := lf.Save(root); err != nil {
		t.Fatal(err)
	}
	lf2, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if lf2.Entries[0].Mode != "weird" || lf2.Entries[0].Kind != "strange" {
		t.Fatalf("round-trip changed: %+v", lf2.Entries)
	}
}
