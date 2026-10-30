package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claude-harness/tools/internal/lockfile"
)

func rowsFor(t *testing.T, project, domain string) []lockfile.Entry {
	t.Helper()
	lock, err := lockfile.Load(project)
	if err != nil {
		t.Fatal(err)
	}
	var out []lockfile.Entry
	for _, e := range lock.Entries {
		if e.Domain == domain {
			out = append(out, e)
		}
	}
	return out
}

func lexists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func TestRemoveDomainLinkAndCopy(t *testing.T) {
	h := realHarness(t)
	for _, mode := range []string{"link", "copy"} {
		t.Run(mode, func(t *testing.T) {
			p := t.TempDir()
			args := []string{"add", "git", "core", "-y", "-p", p, "--harness-root", h}
			if mode == "copy" {
				args = append(args, "--copy")
			}
			if _, err := run(t, args...); err != nil {
				t.Fatal(err)
			}
			if _, err := run(t, "remove", "git", "-y", "-p", p, "--harness-root", h); err != nil {
				t.Fatal(err)
			}
			if lexists(filepath.Join(p, ".claude", "skills", "git-conventional-commits")) {
				t.Fatal("git skill still on disk")
			}
			if !lexists(filepath.Join(p, ".claude", "skills", "caveman")) {
				t.Fatal("core removed too")
			}
			if rows := rowsFor(t, p, "git"); len(rows) != 0 {
				t.Fatalf("orphan rows: %+v", rows)
			}
		})
	}
}

func TestRemoveSkillUnderLinkedDomainNoOrphan(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if _, err := run(t, "add", "core", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "remove", "caveman", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if lexists(filepath.Join(p, ".claude", "skills", "caveman")) {
		t.Fatal("caveman still on disk")
	}
	for _, e := range rowsFor(t, p, "core") {
		if e.Name == "caveman" {
			t.Fatalf("orphan row %+v", e)
		}
	}
}

func TestRemoveDomainOverStandaloneSkillNoOrphan(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if _, err := run(t, "add", "caveman", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "add", "core", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "remove", "core", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if rows := rowsFor(t, p, "core"); len(rows) != 0 {
		t.Fatalf("orphan rows: %+v", rows)
	}
	if lexists(filepath.Join(p, ".claude", "skills", "caveman")) {
		t.Fatal("caveman still on disk")
	}
}

func TestRemoveSkillInsideCopiedDomainErrors(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if _, err := run(t, "add", "core", "--copy", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	for name, root := range map[string]string{"with harness": h, "without harness": ""} {
		args := []string{"remove", "caveman", "-y", "-p", p}
		d := testDeps()
		if root != "" {
			args = append(args, "--harness-root", root)
		} else {
			d = noHarness(t)
		}
		_, err := runD(t, d, args...)
		if err == nil || !strings.Contains(err.Error(), "installed as part of domain core (copy); remove the domain") {
			t.Fatalf("%s: got %v", name, err)
		}
	}
	if !lexists(filepath.Join(p, ".claude", "skills", "caveman")) {
		t.Fatal("caveman must survive the refused remove")
	}
}

func TestRemoveWorksWithoutHarness(t *testing.T) {
	h := realHarness(t)
	for _, mode := range []string{"link", "copy"} {
		t.Run(mode, func(t *testing.T) {
			p := t.TempDir()
			args := []string{"add", "git", "-y", "-p", p, "--harness-root", h}
			if mode == "copy" {
				args = append(args, "--copy")
			}
			if _, err := run(t, args...); err != nil {
				t.Fatal(err)
			}
			d := noHarness(t)
			if _, err := runD(t, d, "remove", "git", "-y", "-p", p); err != nil {
				t.Fatal(err)
			}
			if lexists(filepath.Join(p, ".claude", "skills", "git-conventional-commits")) {
				t.Fatal("skill still on disk")
			}
			if rows := rowsFor(t, p, "git"); len(rows) != 0 {
				t.Fatalf("orphan rows: %+v", rows)
			}
		})
	}
}

func TestRemoveDefaultsToAllRecordedPlatforms(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if _, err := run(t, "add", "git", "--copy", "-y", "-a", "claude,codex", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "remove", "git", "-y", "-p", p, "--harness-root", h)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "platforms: .claude, .codex") {
		t.Fatalf("platforms line:\n%s", out)
	}
	if lexists(filepath.Join(p, ".agents", "skills", "git-conventional-commits")) || lexists(filepath.Join(p, ".claude", "skills", "git-conventional-commits")) {
		t.Fatal("skills remain")
	}
	if rows := rowsFor(t, p, "git"); len(rows) != 0 {
		t.Fatalf("orphan rows: %+v", rows)
	}
}

func TestRemoveUnknownAndFlags(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if _, err := run(t, "remove", "git", "-y", "-p", p, "--harness-root", h); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("got %v", err)
	}
	if _, err := run(t, "remove", "git", "--copy", "-y", "-p", p); err == nil {
		t.Fatal("--copy must be rejected on remove")
	}
}

func TestRemoveDryRunKeepsEverything(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if _, err := run(t, "add", "git", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	before := normLock(t, p)
	if _, err := run(t, "remove", "git", "--dry-run", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if !lexists(filepath.Join(p, ".claude", "skills", "git-conventional-commits")) || normLock(t, p) != before {
		t.Fatal("dry-run changed state")
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func tempHarness(t *testing.T) string {
	t.Helper()
	real := realHarness(t)
	h := filepath.Join(t.TempDir(), "harness")
	copyFile(t, filepath.Join(real, "go.mod"), filepath.Join(h, "go.mod"))
	copyFile(t, filepath.Join(real, "tools", "main.go"), filepath.Join(h, "tools", "main.go"))
	err := filepath.WalkDir(filepath.Join(real, "domains"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(real, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(h, rel), 0o755)
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		copyFile(t, p, filepath.Join(h, rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func installTemp(t *testing.T, mode string, names []string, extra ...string) (h, p string) {
	t.Helper()
	h, p = tempHarness(t), t.TempDir()
	args := append([]string{"add"}, names...)
	args = append(args, "-y", "-p", p, "--harness-root", h)
	if mode == "copy" {
		args = append(args, "--copy")
	}
	if _, err := run(t, append(args, extra...)...); err != nil {
		t.Fatal(err)
	}
	return h, p
}

func editLock(t *testing.T, p string, fn func(*lockfile.Lockfile)) {
	t.Helper()
	lock, err := lockfile.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	fn(lock)
	if err := lock.Save(p); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveNoHarnessDomainRowsFully(t *testing.T) {
	for _, mode := range []string{"copy", "link"} {
		t.Run(mode, func(t *testing.T) {
			h, p := installTemp(t, mode, []string{"code", "core"})
			foreign := filepath.Join(p, ".claude", "hooks", "README.md")
			if err := os.WriteFile(foreign, []byte("mine"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(h); err != nil {
				t.Fatal(err)
			}
			d := noHarness(t)
			if _, err := runD(t, d, "remove", "code", "core", "-y", "-p", p); err != nil {
				t.Fatal(err)
			}
			if !lexists(foreign) {
				t.Fatal("foreign hooks/README.md was deleted")
			}
			ents, _ := os.ReadDir(filepath.Join(p, ".claude", "skills"))
			if len(ents) != 0 {
				t.Fatalf("skills left: %v", ents)
			}
			ents, _ = os.ReadDir(filepath.Join(p, ".claude", "agents"))
			if len(ents) != 0 {
				t.Fatalf("agents left: %v", ents)
			}
			ents, _ = os.ReadDir(filepath.Join(p, ".claude", "hooks"))
			if len(ents) != 1 {
				t.Fatalf("hooks left besides foreign README: %v", ents)
			}
			lock, _ := lockfile.Load(p)
			if len(lock.Entries) != 0 {
				t.Fatalf("rows left: %+v", lock.Entries)
			}
		})
	}
}

func TestRemoveRefusesRowOutsideProject(t *testing.T) {
	for _, harness := range []string{"with", "without"} {
		t.Run(harness, func(t *testing.T) {
			h, p := installTemp(t, "copy", []string{"git"})
			victim := filepath.Join(t.TempDir(), "victim")
			if err := os.MkdirAll(victim, 0o755); err != nil {
				t.Fatal(err)
			}
			data := filepath.Join(victim, "data")
			if err := os.WriteFile(data, []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			editLock(t, p, func(l *lockfile.Lockfile) {
				l.Entries = append(l.Entries,
					lockfile.Entry{Domain: "git", Kind: "skills", Name: "evil", Mode: "copy", Path: victim},
					lockfile.Entry{Domain: "git", Kind: "rules", Mode: "copy", Path: filepath.Join(p, ".claude", "rules", ".."), Items: []string{"..", "../../" + filepath.Base(victim)}},
					lockfile.Entry{Domain: "git", Kind: "agents", Mode: "copy", Path: filepath.Dir(victim), Items: []string{filepath.Base(victim)}},
				)
			})
			rmArgs := []string{"remove", "git", "-y", "-p", p, "--harness-root", h}
			d := testDeps()
			if harness == "without" {
				if err := os.RemoveAll(h); err != nil {
					t.Fatal(err)
				}
				d = noHarness(t)
				rmArgs = []string{"remove", "git", "-y", "-p", p}
			}
			_, err := runD(t, d, rmArgs...)
			if err == nil || !strings.Contains(err.Error(), "refus") {
				t.Fatalf("want refusal, got %v", err)
			}
			if !lexists(data) {
				t.Fatal("outside path was deleted")
			}
			if !lexists(filepath.Join(p, ".claude")) {
				t.Fatal(".claude was deleted")
			}
		})
	}
}

func TestRemoveLegacyRowWithoutSourceNotExpandable(t *testing.T) {
	h, p := installTemp(t, "copy", []string{"core"})
	editLock(t, p, func(l *lockfile.Lockfile) {
		for i := range l.Entries {
			l.Entries[i].Items = nil
		}
	})
	if err := os.RemoveAll(h); err != nil {
		t.Fatal(err)
	}
	d := noHarness(t)
	_, err := runD(t, d, "remove", "core", "-y", "-p", p)
	if err == nil || !strings.Contains(err.Error(), "reinstall") || !strings.Contains(err.Error(), filepath.Join(p, ".claude", "skills")) {
		t.Fatalf("want non-zero naming the row, got %v", err)
	}
	if !lexists(filepath.Join(p, ".claude", "skills", "caveman")) {
		t.Fatal("legacy row must delete nothing")
	}
	if rows := rowsFor(t, p, "core"); len(rows) == 0 {
		t.Fatal("legacy row must be kept")
	}
}

func TestRemoveSharedLocationKeepsOtherPlatform(t *testing.T) {
	h, p := installTemp(t, "copy", []string{"git"}, "-a", "codex,cursor")
	if err := os.RemoveAll(h); err != nil {
		t.Fatal(err)
	}
	d := noHarness(t)
	skill := filepath.Join(p, ".agents", "skills", "git-conventional-commits")
	if !lexists(skill) {
		t.Fatalf("setup: %s missing", skill)
	}
	if _, err := runD(t, d, "remove", "git", "-a", "codex", "-y", "-p", p); err != nil {
		t.Fatal(err)
	}
	if !lexists(skill) {
		t.Fatal("cursor lost the skill")
	}
	for _, e := range rowsFor(t, p, "git") {
		if e.Path == skill && (len(e.Platforms) != 1 || e.Platforms[0] != "cursor") {
			t.Fatalf("row platforms: %v", e.Platforms)
		}
	}
	if _, err := runD(t, d, "remove", "git", "-a", "cursor", "-y", "-p", p); err != nil {
		t.Fatal(err)
	}
	if lexists(skill) {
		t.Fatal("skill must go once no platform uses it")
	}
}

func TestRemoveSkillCountsClaudeCopyCoveredByDomainRow(t *testing.T) {
	h, p := installTemp(t, "copy", []string{"core"}, "-a", "claude,codex")
	if err := os.RemoveAll(h); err != nil {
		t.Fatal(err)
	}
	d := noHarness(t)
	if _, err := runD(t, d, "add", "caveman", "--copy", "-y", "-a", "codex", "-p", p, "--harness-root", tempHarness(t)); err != nil {
		t.Fatal(err)
	}
	_, err := runD(t, d, "remove", "caveman", "-y", "-p", p)
	if err == nil || !strings.Contains(err.Error(), "installed as part of domain core (copy)") {
		t.Fatalf("got %v", err)
	}
	if !lexists(filepath.Join(p, ".claude", "skills", "caveman")) || !lexists(filepath.Join(p, ".agents", "skills", "caveman")) {
		t.Fatal("nothing may be removed on the refused call")
	}
	if _, err := runD(t, d, "remove", "caveman", "-a", "codex", "-y", "-p", p); err != nil {
		t.Fatal(err)
	}
	if lexists(filepath.Join(p, ".agents", "skills", "caveman")) || !lexists(filepath.Join(p, ".claude", "skills", "caveman")) {
		t.Fatal("-a codex must drop only the codex copy")
	}
}

func TestRemoveSweepsRowsWhoseSourceItemIsGone(t *testing.T) {
	h, p := installTemp(t, "link", []string{"git"})
	gone := filepath.Join(h, "domains", "git", "skills", "git-release-tag")
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	dangling := filepath.Join(p, ".claude", "skills", "git-release-tag")
	if !lexists(dangling) {
		t.Fatal("setup: link missing")
	}
	if _, err := run(t, "remove", "git", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if lexists(dangling) {
		t.Fatal("dangling link survived")
	}
	if rows := rowsFor(t, p, "git"); len(rows) != 0 {
		t.Fatalf("orphan rows: %+v", rows)
	}
}

func TestListDoesNotMarkDomainForSingleSkill(t *testing.T) {
	h, p := installTemp(t, "link", []string{"caveman"})
	out, err := run(t, "list", "domains", "-p", p, "--harness-root", h)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "core ") && strings.HasSuffix(line, " installed") {
			t.Fatalf("core marked installed:\n%s", out)
		}
	}
	out, _ = run(t, "list", "skills", "-p", p, "--harness-root", h)
	if !strings.Contains(out, "caveman") || !strings.Contains(out, "(core) installed") {
		t.Fatalf("caveman must be marked:\n%s", out)
	}
}

func TestDoctorListRejectHomeProject(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip()
	}
	for _, cmd := range [][]string{{"doctor", "-p", home}, {"list", "domains", "-p", home}, {"doctor", "-p", "/definitely/not/here"}} {
		if _, err := run(t, cmd...); err == nil {
			t.Fatalf("%v must fail", cmd)
		}
	}
}

func noHarness(t *testing.T) Deps {
	t.Helper()
	d := testDeps()
	t.Chdir(t.TempDir())
	d.Locator = harness.Locator{Executable: func() (string, error) { return filepath.Join(t.TempDir(), "as-skill"), nil }}
	return d
}

func TestRemoveAddRejectBadExplicitHarnessRoot(t *testing.T) {
	h, p := installTemp(t, "copy", []string{"git"})
	bad := t.TempDir()
	_, err := run(t, "remove", "git", "-y", "-p", p, "--harness-root", bad)
	if err == nil || !strings.Contains(err.Error(), "remove: --harness-root rejected") {
		t.Fatalf("remove: got %v", err)
	}
	if !lexists(filepath.Join(p, ".claude", "skills", "git-conventional-commits")) || len(rowsFor(t, p, "git")) == 0 {
		t.Fatal("remove changed state on rejected harness root")
	}
	if _, err := run(t, "add", "core", "-y", "-p", t.TempDir(), "--harness-root", bad); err == nil {
		t.Fatal("add: want error for bad --harness-root")
	}
	_ = h
}

func TestRemoveRefusesSymlinkedPlatformRoot(t *testing.T) {
	for _, root := range []string{".claude", ".agents"} {
		t.Run(root, func(t *testing.T) {
			h := tempHarness(t)
			p, out := t.TempDir(), t.TempDir()
			if err := os.Symlink(out, filepath.Join(p, root)); err != nil {
				t.Fatal(err)
			}
			victim := filepath.Join(out, "skills", "victim")
			if err := os.MkdirAll(victim, 0o755); err != nil {
				t.Fatal(err)
			}
			editLock(t, p, func(l *lockfile.Lockfile) {
				l.Entries = append(l.Entries, lockfile.Entry{Domain: "git", Kind: "skills", Name: "victim", Mode: "copy", Path: filepath.Join(p, root, "skills", "victim"), Platforms: []string{"claude", "codex"}})
			})
			_, err := run(t, "remove", "git", "-y", "-p", p, "--harness-root", h)
			if err == nil || !strings.Contains(err.Error(), "refus") {
				t.Fatalf("want refusal, got %v", err)
			}
			if !lexists(victim) {
				t.Fatal("victim deleted")
			}
			if len(rowsFor(t, p, "git")) == 0 {
				t.Fatal("rows dropped")
			}
		})
	}
}

func TestRemoveWorksThroughSymlinkedProject(t *testing.T) {
	h := tempHarness(t)
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "proj")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "add", "git", "--copy", "-y", "-p", link, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "remove", "git", "-y", "-p", link, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if lexists(filepath.Join(real, ".claude", "skills", "git-conventional-commits")) {
		t.Fatal("skill still on disk")
	}
}
