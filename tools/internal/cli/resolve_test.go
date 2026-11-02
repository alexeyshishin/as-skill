package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func realRegistry(t *testing.T) (string, *harness.Registry) {
	t.Helper()
	h := realHarness(t)
	reg, err := harness.LoadRegistry(h)
	if err != nil {
		t.Fatal(err)
	}
	return h, reg
}

func TestResolveNamesMatrix(t *testing.T) {
	h, reg := realRegistry(t)
	t.Setenv("OBSIDIAN_VAULT", "")
	sel, err := resolveNames(h, reg, []string{"code", "caveman", "git-conventional-commits"})
	if err != nil {
		t.Fatal(err)
	}
	if got := sel.domainNames(); !reflect.DeepEqual(got, []string{"code"}) || !sel.Domains[0].Strict {
		t.Fatalf("domains %+v", sel.Domains)
	}
	if !reflect.DeepEqual(sel.Skills, []string{"caveman", "git-conventional-commits"}) {
		t.Fatalf("skills %v", sel.Skills)
	}

	sel, err = resolveNames(h, reg, []string{"core", "caveman", "caveman"})
	if err != nil || len(sel.Skills) != 0 || len(sel.Domains) != 1 {
		t.Fatalf("collision not dropped: %+v %v", sel, err)
	}

	sel, err = resolveNames(h, reg, []string{"all"})
	if err != nil || len(sel.Domains) != len(reg.DomainNames()) {
		t.Fatalf("all: %+v %v", sel, err)
	}
	for _, d := range sel.Domains {
		if d.Strict {
			t.Fatalf("all must be non-strict: %+v", d)
		}
	}
	sel, _ = resolveNames(h, reg, []string{"all", "git"})
	for _, d := range sel.Domains {
		if d.Strict != (d.Name == "git") {
			t.Fatalf("explicit strict only: %+v", d)
		}
	}

	for _, n := range []string{"obsidian", "obsidian-ingest"} {
		if _, err := resolveNames(h, reg, []string{n}); err == nil || !strings.Contains(err.Error(), "OBSIDIAN_VAULT") {
			t.Fatalf("%s gated: %v", n, err)
		}
	}
	_, err = resolveNames(h, reg, []string{"zzz"})
	if err == nil || !strings.Contains(err.Error(), "as-skill list skills") || !strings.Contains(err.Error(), "devops") {
		t.Fatalf("unknown: %v", err)
	}
	t.Setenv("OBSIDIAN_VAULT", t.TempDir())
	if _, err := resolveNames(h, reg, []string{"obsidian"}); err != nil {
		t.Fatalf("ungated: %v", err)
	}
}

func TestDetectPlatforms(t *testing.T) {
	mk := func(dirs ...string) string {
		p := t.TempDir()
		for _, d := range dirs {
			if err := os.MkdirAll(filepath.Join(p, d), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		return p
	}
	ids := func(in ...platform.ID) []platform.ID { return in }

	got, src := detectPlatforms(mk(), &lockfile.Lockfile{})
	if !reflect.DeepEqual(got, ids(platform.Claude)) || len(src) != 0 {
		t.Fatalf("empty: %v %v", got, src)
	}
	got, src = detectPlatforms(mk(".cursor"), &lockfile.Lockfile{})
	if !reflect.DeepEqual(got, ids(platform.Cursor)) || !reflect.DeepEqual(src, []string{".cursor"}) {
		t.Fatalf("cursor: %v %v", got, src)
	}
	p := mk(".claude")
	if err := os.WriteFile(filepath.Join(p, ".claude", "skills-lock.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, src = detectPlatforms(p, &lockfile.Lockfile{})
	if !reflect.DeepEqual(got, ids(platform.Claude)) || len(src) != 0 {
		t.Fatalf("lock-only .claude must not count: %v %v", got, src)
	}
	if err := os.MkdirAll(filepath.Join(p, ".claude", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, src = detectPlatforms(p, &lockfile.Lockfile{})
	if !reflect.DeepEqual(src, []string{".claude"}) {
		t.Fatalf("real .claude: %v %v", got, src)
	}
	lock := &lockfile.Lockfile{Entries: []lockfile.Entry{{Domain: "git", Kind: "skills", Name: "x", Platforms: []string{"codex"}}}}
	got, src = detectPlatforms(mk(), lock)
	if !reflect.DeepEqual(got, ids(platform.Codex)) || len(src) != 1 {
		t.Fatalf("lockfile codex only (no claude seeding): %v %v", got, src)
	}
	got, _ = detectPlatforms(mk(".opencode", ".codex"), lock)
	if !reflect.DeepEqual(got, ids(platform.Codex, platform.OpenCode)) {
		t.Fatalf("union/order: %v", got)
	}
}

func TestParseAddArgs(t *testing.T) {
	o, err := parseAddArgs("add", []string{"git", "-a", "claude,codex", "--agent=cursor", "--platform", "opencode", "-p", "x", "--copy", "-y", "--dry-run", "--force", "--harness-root", "h", "core"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(o.names, []string{"git", "core"}) || len(o.agents) != 3 || !o.projectSet || !o.copy || !o.yes || !o.dryRun || !o.force || o.harnessRoot != "h" {
		t.Fatalf("%+v", o)
	}
	if _, err := parseAddArgs("add", []string{"--bogus"}); err == nil {
		t.Fatal("unknown flag accepted")
	}
	if _, err := parseAddArgs("add", []string{"-p"}); err == nil {
		t.Fatal("missing value accepted")
	}
}
