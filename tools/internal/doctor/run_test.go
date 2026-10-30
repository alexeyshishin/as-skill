package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"claude-harness/tools/internal/lockfile"
)

func find(fs []Finding, code string) *Finding {
	for i := range fs {
		if fs[i].Code == code {
			return &fs[i]
		}
	}
	return nil
}

func TestRunSourceGoneAndMissing(t *testing.T) {
	project := t.TempDir()
	missing := filepath.Join(project, ".claude", "skills", "gone")
	linked := filepath.Join(project, ".claude", "skills", "linked")
	src := filepath.Join(t.TempDir(), "nope")
	if err := os.MkdirAll(filepath.Dir(linked), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(src, linked); err != nil {
		t.Fatal(err)
	}
	entries := []lockfile.Entry{
		{Domain: "d", Kind: "skills", Name: "gone", Mode: lockfile.ModeCopy, Path: missing},
		{Domain: "d", Kind: "skills", Name: "linked", Mode: lockfile.ModeLink, Path: linked, Source: src},
	}
	lock := &lockfile.Lockfile{Entries: entries}
	res, err := Run(Input{ProjectRoot: project, HarnessRoot: "/h", HarnessOK: true, Lock: lock, Entries: entries})
	if err != nil {
		t.Fatal(err)
	}
	if f := find(res.Findings, HealthMissing); f == nil || f.Severity != SeverityError || f.Entry != "d/skills/gone" || f.Msg != missing {
		t.Fatalf("missing finding: %+v", res.Findings)
	}
	if f := find(res.Findings, HealthSourceGone); f == nil || f.Entry != "d/skills/linked" {
		t.Fatalf("source-gone finding: %+v", res.Findings)
	}
	if res.Checked != 2 || res.Failed != 2 {
		t.Fatalf("checked=%d failed=%d", res.Checked, res.Failed)
	}

	res, err = Run(Input{ProjectRoot: project, HarnessOK: false, Lock: lock, Entries: entries[1:]})
	if err != nil {
		t.Fatal(err)
	}
	if find(res.Findings, HealthSourceGone) != nil {
		t.Fatalf("source-gone must be skipped without harness: %+v", res.Findings)
	}
}

func TestRunUntrackedCountsAsFailure(t *testing.T) {
	project := t.TempDir()
	foreign := filepath.Join(project, ".claude", "skills", "foreign")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Run(Input{ProjectRoot: project, Lock: &lockfile.Lockfile{}})
	if err != nil {
		t.Fatal(err)
	}
	f := find(res.Findings, HealthUntracked)
	if f == nil || f.Entry != foreign || res.Checked != 1 || res.Failed != 1 {
		t.Fatalf("res=%+v", res)
	}
}

func TestRunWarningsDoNotFail(t *testing.T) {
	lock := &lockfile.Lockfile{Entries: []lockfile.Entry{{Domain: "d", Mode: lockfile.ModeCopy, Platforms: []string{"codex"}}}}
	res, err := Run(Input{ProjectRoot: t.TempDir(), Lock: lock})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 0 {
		t.Fatalf("failed=%d", res.Failed)
	}
	if f := find(res.Findings, Unverified); f == nil || f.Severity != SeverityWarning {
		t.Fatalf("want UNVERIFIED warning: %+v", res.Findings)
	}
}
