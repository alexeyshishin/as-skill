package install

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"claude-harness/tools/internal/doctor"
	"claude-harness/tools/internal/fsys"
	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
)

func rerunSetup(t *testing.T, mode string) (*Installer, *harness.Registry, string) {
	t.Helper()
	hRoot, err := harness.ResolveHarnessRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := harness.LoadRegistry(hRoot)
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	return &Installer{HarnessRoot: hRoot, ProjectRoot: project, Mode: lockfile.Mode(mode), Lock: &lockfile.Lockfile{}}, reg, project
}

func TestRerunWithoutForce(t *testing.T) {
	for _, tc := range []struct{ name, first, second string }{
		{"copy-copy", "copy", "copy"},
		{"copy-link", "copy", "link"},
		{"link-copy", "link", "copy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in, reg, project := rerunSetup(t, tc.first)
			if err := in.InstallDomain(reg, "git", true); err != nil {
				t.Fatal(err)
			}
			in.Mode = lockfile.Mode(tc.second)
			if err := in.InstallDomain(reg, "git", true); err != nil {
				t.Fatalf("rerun: %v", err)
			}
			rule := filepath.Join(project, ".claude", "rules", "git-conventions.md")
			fi, err := os.Lstat(rule)
			if err != nil {
				t.Fatal(err)
			}
			if isLink := fi.Mode()&os.ModeSymlink != 0; isLink != (tc.second == "link") {
				t.Fatalf("symlink=%v after %s", isLink, tc.second)
			}
			for _, e := range in.Lock.Entries {
				if string(e.Mode) != tc.second {
					t.Fatalf("stale row %+v", e)
				}
			}
			for _, e := range in.Lock.Entries {
				if tc.second == "copy" {
					if ok, err := doctor.CopyMatches(e); err != nil || !ok {
						t.Fatalf("copy row %s mismatch: %v", e.Path, err)
					}
				}
				if h := doctor.EntryHealth(e); h != doctor.HealthOK {
					t.Fatalf("%s %s", e.Path, h)
				}
			}
		})
	}
}

func TestLinkThenCopyKeepsHarnessSourceIntact(t *testing.T) {
	in, reg, _ := rerunSetup(t, "link")
	if err := in.InstallDomain(reg, "git", true); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(in.HarnessRoot, "domains", "git", "rules", "git-conventions.md")
	before, _ := os.ReadFile(src)
	in.Mode = "copy"
	if err := in.InstallDomain(reg, "git", true); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(src)
	if string(before) != string(after) || len(after) == 0 {
		t.Fatal("harness source modified")
	}
}

func rowsAt(lock *lockfile.Lockfile, p string) int {
	n := 0
	for _, e := range lock.Entries {
		if filepath.Clean(e.Path) == filepath.Clean(p) {
			n++
		}
	}
	return n
}

func TestOrphanRowSkillUnderDomain(t *testing.T) {
	in, reg, project := rerunSetup(t, "link")
	if err := in.InstallDomain(reg, "core", true); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(project, ".claude", "skills", "caveman")
	if err := in.UninstallSkill(reg, "caveman"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("link remains: %v", err)
	}
	if n := rowsAt(in.Lock, dst); n != 0 {
		t.Fatalf("%d orphan rows", n)
	}
}

func TestOrphanRowDomainOverStandaloneSkill(t *testing.T) {
	in, reg, project := rerunSetup(t, "link")
	if err := in.InstallSkill(reg, "caveman"); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(project, ".claude", "skills", "caveman")
	if n := rowsAt(in.Lock, dst); n != 1 {
		t.Fatalf("rows before = %d", n)
	}
	if err := in.UninstallDomain(reg, "core"); err != nil {
		t.Fatal(err)
	}
	if n := rowsAt(in.Lock, dst); n != 0 {
		t.Fatalf("%d orphan rows", n)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("link remains: %v", err)
	}
}

func TestUninstallSkillInsideDomainCopyRow(t *testing.T) {
	in, reg, project := rerunSetup(t, "copy")
	if err := in.InstallDomain(reg, "core", true); err != nil {
		t.Fatal(err)
	}
	err := in.UninstallSkill(reg, "caveman")
	var pe *PartOfDomainError
	if !errors.As(err, &pe) || pe.Domain != "core" || pe.Mode != "copy" {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(project, ".claude", "skills", "caveman")); err != nil {
		t.Fatalf("skill must stay: %v", err)
	}
}

func TestMultiDomainCopyHasNoDrift(t *testing.T) {
	in, reg, project := rerunSetup(t, "copy")
	for _, d := range []string{"code", "git", "core"} {
		if err := in.InstallDomain(reg, d, true); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range in.Lock.Entries {
		if ok, err := doctor.CopyMatches(e); err != nil || !ok {
			t.Fatalf("%s/%s %s drifted: %v", e.Domain, e.Kind, e.Path, err)
		}
	}
	if err := in.Lock.Save(project); err != nil {
		t.Fatal(err)
	}
	untracked, err := doctor.FindUntracked(project, in.Lock)
	if err != nil || len(untracked) != 0 {
		t.Fatalf("untracked %v %v", untracked, err)
	}
}

func TestLegacyWholeDirHashAcceptedSharedDir(t *testing.T) {
	in, reg, _ := rerunSetup(t, "copy")
	domains := []string{"code", "git", "core"}
	legacy := map[string]string{}
	for _, d := range domains {
		if err := in.InstallDomain(reg, d, true); err != nil {
			t.Fatal(err)
		}
		for i, e := range in.Lock.Entries {
			if e.Domain == d && e.Name == "" && e.Mode == "copy" && e.Kind == "skills" {
				h, err := fsys.HashTree(e.Path)
				if err != nil {
					t.Fatal(err)
				}
				in.Lock.Entries[i].SHA256 = h
				legacy[d] = h
			}
		}
	}
	if legacy["code"] == legacy["git"] || legacy["git"] == legacy["core"] {
		t.Fatalf("hashes not distinct: %v", legacy)
	}
	for i := range in.Lock.Entries {
		in.Lock.Entries[i].Items = nil
	}
	for _, e := range in.Lock.Entries {
		if e.Name != "" || e.Mode != "copy" {
			continue
		}
		if ok, err := doctor.CopyMatchesIn(in.Lock, e); err != nil || !ok {
			t.Fatalf("legacy row %s/%s rejected: %v", e.Domain, e.Kind, err)
		}
	}
	for i, e := range in.Lock.Entries {
		if e.Name != "" || e.Mode != "copy" || e.Kind != "skills" {
			continue
		}
		in.Lock.Entries[i].SHA256 = "bogus"
		if ok, _ := doctor.CopyMatchesIn(in.Lock, in.Lock.Entries[i]); ok {
			t.Fatal("bogus hash accepted")
		}
	}
}

func TestItemsRowSelfDescribingAfterSourceDeleted(t *testing.T) {
	in, reg, project := rerunSetup(t, "copy")
	if err := in.InstallDomain(reg, "git", true); err != nil {
		t.Fatal(err)
	}
	var row lockfile.Entry
	for _, e := range in.Lock.Entries {
		if e.Domain == "git" && e.Kind == "skills" && e.Name == "" {
			row = e
		}
	}
	if len(row.Items) == 0 {
		t.Fatalf("no items recorded: %+v", row)
	}
	gone := filepath.Join(t.TempDir(), "deleted-harness")
	for i := range in.Lock.Entries {
		in.Lock.Entries[i].Source = filepath.Join(gone, filepath.Base(in.Lock.Entries[i].Source))
	}
	row.Source = filepath.Join(gone, "x")
	if ok, err := doctor.CopyMatches(row); err != nil || !ok {
		t.Fatalf("items row not verified without source: %v", err)
	}
	for _, n := range row.Items {
		if !in.Lock.Covers(filepath.Join(row.Path, n)) {
			t.Fatalf("%s not covered", n)
		}
	}
	if in.Lock.Covers(filepath.Join(row.Path, "stranger")) {
		t.Fatal("foreign item covered")
	}
	if got := row.CoveredPaths(); len(got) != len(row.Items) {
		t.Fatalf("CoveredPaths %v", got)
	}
	if err := in.Lock.Save(project); err != nil {
		t.Fatal(err)
	}
	untracked, err := doctor.FindUntracked(project, in.Lock)
	if err != nil || len(untracked) != 0 {
		t.Fatalf("untracked %v %v", untracked, err)
	}
}

func TestLegacyRowSourceGoneCannotVerify(t *testing.T) {
	in, reg, _ := rerunSetup(t, "copy")
	if err := in.InstallDomain(reg, "git", true); err != nil {
		t.Fatal(err)
	}
	for _, e := range in.Lock.Entries {
		if e.Name != "" || e.Kind != "skills" {
			continue
		}
		e.Items = nil
		e.SHA256 = "bogus"
		e.Source = filepath.Join(t.TempDir(), "gone")
		if _, err := doctor.CopyMatches(e); !errors.Is(err, doctor.ErrCannotVerify) {
			t.Fatalf("err = %v", err)
		}
	}
}
