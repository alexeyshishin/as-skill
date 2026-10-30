package lockfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLockfileRoundTrip(t *testing.T) {
	projectRoot := t.TempDir()

	lock := &Lockfile{}
	entry := Entry{
		Domain:      "git",
		Kind:        "skill",
		Name:        "git-conventional-commits",
		Mode:        "link",
		Path:        filepath.Join(projectRoot, ".claude", "skills", "git-conventional-commits"),
		Source:      "/harness/domains/git/skills/git-conventional-commits",
		InstalledAt: time.Now(),
	}
	lock.Record(entry)

	if err := lock.Save(projectRoot); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(loaded.Entries))
	}

	got, ok := loaded.Owns(entry.Path)
	if !ok {
		t.Fatalf("Owns(%q) = false, want true", entry.Path)
	}
	if got.Domain != entry.Domain || got.Kind != entry.Kind || got.Name != entry.Name || got.Source != entry.Source {
		t.Fatalf("round-tripped entry = %+v, want %+v", got, entry)
	}

	if _, ok := loaded.Owns(filepath.Join(projectRoot, "nope")); ok {
		t.Fatal("Owns() found an entry for a path that was never recorded")
	}
}

func TestRealLockfileRoundTripsByteIdentical(t *testing.T) {
	orig, err := os.ReadFile(filepath.Join("testdata", ".claude", "skills-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "skills-lock.json"), orig, 0o644); err != nil {
		t.Fatal(err)
	}
	lf, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range lf.Entries {
		if len(e.Platforms) != 1 || e.Platforms[0] != "claude" {
			t.Fatalf("entry %s loaded with platforms %v, want [claude]", e.Name, e.Platforms)
		}
	}
	if err := lf.Save(root); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".claude", "skills-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(orig) {
		t.Fatal("re-saved lockfile differs from original bytes")
	}
}

func TestSameNameDifferentPathsDoNotCollide(t *testing.T) {
	lf := &Lockfile{}
	a := Entry{Domain: "d", Kind: "skills", Name: "s", Mode: "link", Path: "/p/.claude/skills/s"}
	b := Entry{Domain: "d", Kind: "skills", Name: "s", Mode: "render", Path: "/p/.agents/skills/s", Platforms: []string{"codex"}, SrcSHA256: "abc"}
	lf.Record(a)
	lf.Record(b)
	if len(lf.Entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(lf.Entries))
	}
	lf.Record(a)
	if len(lf.Entries) != 2 {
		t.Fatalf("re-record duplicated: %d entries", len(lf.Entries))
	}
	if e, ok := lf.FindAt("d", "skills", "s", b.Path); !ok || e.Mode != "render" || e.SrcSHA256 != "abc" {
		t.Fatalf("FindAt = %+v, %v", e, ok)
	}
	if _, ok := lf.Find("d", "skills", "s"); !ok {
		t.Fatal("legacy Find lost")
	}
	lf.RemoveAt("d", "skills", "s", a.Path)
	if len(lf.Entries) != 1 || lf.Entries[0].Path != b.Path {
		t.Fatalf("RemoveAt wrong: %+v", lf.Entries)
	}
	root := t.TempDir()
	if err := lf.Save(root); err != nil {
		t.Fatal(err)
	}
	back, err := Load(root)
	if err != nil || len(back.Entries[0].Platforms) != 1 || back.Entries[0].Platforms[0] != "codex" {
		t.Fatalf("platforms not preserved: %+v %v", back, err)
	}
}

func TestCovers(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(src, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	skills := filepath.Join(dir, ".claude", "skills")
	lf := &Lockfile{}
	lf.Record(Entry{Domain: "d", Kind: "skills", Name: "", Mode: "copy", Path: skills, Source: src})
	lf.Record(Entry{Domain: "d", Kind: "skill", Name: "x", Mode: "link", Path: filepath.Join(skills, "x"), Source: "/s"})
	for p, want := range map[string]bool{
		filepath.Join(skills, "a"):         true,
		filepath.Join(skills, "x"):         true,
		filepath.Join(skills, "b"):         false,
		filepath.Join(dir, "other", "a"):   false,
		filepath.Join(skills, "a", "deep"): false,
	} {
		if got := lf.Covers(p); got != want {
			t.Errorf("Covers(%s) = %v, want %v", p, got, want)
		}
	}
}

func TestRemoveAllAtIgnoresKind(t *testing.T) {
	lf := &Lockfile{}
	lf.Record(Entry{Domain: "d", Kind: "skill", Name: "x", Path: "/p/x"})
	lf.Record(Entry{Domain: "d", Kind: "skills", Name: "x", Path: "/p/x"})
	lf.Record(Entry{Domain: "d", Kind: "skills", Name: "y", Path: "/p/y"})
	lf.RemoveAllAt("/p/x")
	if len(lf.Entries) != 1 || lf.Entries[0].Name != "y" {
		t.Fatalf("%+v", lf.Entries)
	}
}
