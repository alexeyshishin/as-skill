package installer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/health"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func assertLockClean(t *testing.T, lock *lockfile.Lockfile) {
	t.Helper()
	covered := map[string]string{}
	for _, e := range lock.Entries {
		if h := health.EntryHealth(e); h != health.HealthOK {
			t.Fatalf("%s: %s", e.Path, h)
		}
		if e.Mode == lockfile.ModeCopy {
			ok, err := health.CopyMatchesIn(lock, e)
			if err != nil || !ok {
				t.Fatalf("%s %s/%s/%s: DRIFTED (match=%v err=%v)", e.Path, e.Domain, e.Kind, e.Name, ok, err)
			}
		}
		for _, p := range e.CoveredPaths() {
			p = filepath.Clean(p)
			if prev, dup := covered[p]; dup {
				t.Fatalf("%s covered twice: %s and %s/%s/%s", p, prev, e.Domain, e.Kind, e.Name)
			}
			covered[p] = e.Domain + "/" + string(e.Kind) + "/" + e.Name
		}
	}
}

func installAlpha(t *testing.T, root, project, mode string, lock *lockfile.Lockfile, reg *harness.Registry, exclude map[string]bool) {
	t.Helper()
	in := newInst(root, project, mode, lock, platform.Claude)
	in.ExcludeSkills = exclude
	var err error
	captureStderr(t, func() {
		_, err = captureStdout(t, func() error { return in.InstallDomain(reg, "alpha", true) })
	})
	if err != nil {
		t.Fatalf("install alpha (%s, exclude=%v): %v", mode, exclude, err)
	}
}

func changeSource(t *testing.T, root string) {
	t.Helper()
	write(t, filepath.Join(root, "domains", "alpha", "skills", "sk", "SKILL.md"), "---\nname: sk\ndescription: d\n---\nchanged body\n", 0o644)
}

func skillRows(lock *lockfile.Lockfile) []lockfile.Entry {
	var out []lockfile.Entry
	for _, e := range lock.Entries {
		if e.Domain == "alpha" && e.Kind == "skills" {
			out = append(out, e)
		}
	}
	return out
}

func TestRecopyFullThenExclusionsReconcilesKindRow(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	installAlpha(t, root, project, "copy", lock, reg, nil)
	assertLockClean(t, lock)
	changeSource(t, root)
	installAlpha(t, root, project, "copy", lock, reg, map[string]bool{"alpha/sk2": true})

	assertLockClean(t, lock)
	parent, ok := lock.Find("alpha", "skills", "")
	if !ok || strings.Join(parent.Items, ",") != "sk2" {
		t.Fatalf("kind-level row not narrowed to the excluded skill: %+v ok=%v", parent, ok)
	}
	if _, ok := lock.FindAt("alpha", "skills", "sk", filepath.Join(project, ".claude", "skills", "sk")); !ok {
		t.Fatal("no per-item row for sk")
	}
	if !strings.Contains(readStr(t, filepath.Join(project, ".claude", "skills", "sk", "SKILL.md")), "changed body") {
		t.Fatal("sk not refreshed")
	}
	if !exists(filepath.Join(project, ".claude", "skills", "sk2")) {
		t.Fatal("previously copied sk2 removed")
	}
}

func TestRecopyFullThenExclusionsDropsKindRowWhenNoExcludedLeft(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	installAlpha(t, root, project, "copy", lock, reg, nil)
	if err := os.RemoveAll(filepath.Join(project, ".claude", "skills", "sk2")); err != nil {
		t.Fatal(err)
	}
	changeSource(t, root)
	installAlpha(t, root, project, "copy", lock, reg, map[string]bool{"alpha/sk2": true})

	assertLockClean(t, lock)
	if _, ok := lock.Find("alpha", "skills", ""); ok {
		t.Fatal("kind-level skills row kept although no excluded skill is on disk")
	}
}

func TestRecopyExclusionsThenFullSupersedesPerItemRows(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	installAlpha(t, root, project, "copy", lock, reg, map[string]bool{"alpha/sk2": true})
	assertLockClean(t, lock)
	changeSource(t, root)
	installAlpha(t, root, project, "copy", lock, reg, nil)

	assertLockClean(t, lock)
	for _, e := range skillRows(lock) {
		if e.Name != "" {
			t.Fatalf("per-item row survived the full copy: %+v", e)
		}
	}
	parent, ok := lock.Find("alpha", "skills", "")
	if !ok || strings.Join(parent.Items, ",") != "sk,sk2" {
		t.Fatalf("kind-level row = %+v ok=%v", parent, ok)
	}
	if !strings.Contains(readStr(t, filepath.Join(project, ".claude", "skills", "sk", "SKILL.md")), "changed body") {
		t.Fatal("sk not refreshed")
	}
}

func TestRecopyFullThenFullStaysClean(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	installAlpha(t, root, project, "copy", lock, reg, nil)
	changeSource(t, root)
	installAlpha(t, root, project, "copy", lock, reg, nil)
	assertLockClean(t, lock)
}

func TestRecopyExclusionsRoundTripBothDirections(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	ex := map[string]bool{"alpha/sk2": true}
	for i, step := range []map[string]bool{nil, ex, nil, ex, ex, nil} {
		write(t, filepath.Join(root, "domains", "alpha", "skills", "sk", "SKILL.md"), "---\nname: sk\ndescription: d\n---\nbody "+string(rune('a'+i))+"\n", 0o644)
		installAlpha(t, root, project, "copy", lock, reg, step)
		assertLockClean(t, lock)
	}
}

func TestLinkThenCopyWithExclusionsStaysClean(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	installAlpha(t, root, project, "link", lock, reg, nil)
	changeSource(t, root)
	installAlpha(t, root, project, "copy", lock, reg, map[string]bool{"alpha/sk2": true})
	assertLockClean(t, lock)
	if fi, err := os.Lstat(filepath.Join(project, ".claude", "skills", "sk")); err != nil || fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("sk should be a real copy, err=%v", err)
	}
	if fi, err := os.Lstat(filepath.Join(project, ".claude", "skills", "sk2")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("excluded sk2 should stay a link, err=%v", err)
	}
}

func TestCopyWithExclusionsThenLinkStaysClean(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	installAlpha(t, root, project, "copy", lock, reg, map[string]bool{"alpha/sk2": true})
	changeSource(t, root)
	installAlpha(t, root, project, "link", lock, reg, nil)
	assertLockClean(t, lock)
	for _, n := range []string{"sk", "sk2"} {
		if fi, err := os.Lstat(filepath.Join(project, ".claude", "skills", n)); err != nil || fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s should be a link, err=%v", n, err)
		}
	}
	for _, e := range lock.Entries {
		if e.Mode != lockfile.ModeLink && e.Kind == "skills" {
			t.Fatalf("stale copy row: %+v", e)
		}
	}
}

func TestRecopyDryRunLeavesLockUntouched(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	installAlpha(t, root, project, "copy", lock, reg, nil)
	before := append([]lockfile.Entry(nil), lock.Entries...)
	changeSource(t, root)
	in := newInst(root, project, "copy", lock, platform.Claude)
	in.ExcludeSkills = map[string]bool{"alpha/sk2": true}
	in.DryRun = true
	captureStderr(t, func() {
		captureStdout(t, func() error { return in.InstallDomain(reg, "alpha", true) })
	})
	if len(lock.Entries) != len(before) {
		t.Fatalf("dry run changed rows: %d -> %d", len(before), len(lock.Entries))
	}
	for i, e := range lock.Entries {
		if e.SHA256 != before[i].SHA256 || strings.Join(e.Items, ",") != strings.Join(before[i].Items, ",") {
			t.Fatalf("dry run changed row %s", e.Path)
		}
	}
}

func TestRecopyRealCoreOverlayWithChangedSource(t *testing.T) {
	realRoot, err := harness.ResolveHarnessRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.CopyFS(filepath.Join(root, "domains", "core"), os.DirFS(filepath.Join(realRoot, "domains", "core"))); err != nil {
		t.Fatal(err)
	}
	reg, err := harness.LoadRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	skills, err := os.ReadDir(filepath.Join(root, "domains", "core", "skills"))
	if err != nil || len(skills) < 2 {
		t.Fatalf("core skills: %v %v", skills, err)
	}
	var names []string
	for _, s := range skills {
		names = append(names, s.Name())
	}
	sort.Strings(names)
	excluded, changed := names[0], names[1]
	touch := func(body string) {
		write(t, filepath.Join(root, "domains", "core", "skills", changed, "zz-recopy.md"), body, 0o644)
	}

	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	install := func(ex map[string]bool) {
		in := newInst(root, project, "copy", lock, platform.Claude)
		in.ExcludeSkills = ex
		var ierr error
		captureStderr(t, func() {
			_, ierr = captureStdout(t, func() error { return in.InstallDomain(reg, "core", true) })
		})
		if ierr != nil {
			t.Fatal(ierr)
		}
	}
	ex := map[string]bool{"core/" + excluded: true}

	touch("v1")
	install(nil)
	assertLockClean(t, lock)
	touch("v2")
	install(ex)
	assertLockClean(t, lock)
	touch("v3")
	install(nil)
	assertLockClean(t, lock)
	if readStr(t, filepath.Join(project, ".claude", "skills", changed, "zz-recopy.md")) != "v3" {
		t.Fatal("changed skill not refreshed")
	}
}

func installAlphaPlats(t *testing.T, root, project string, lock *lockfile.Lockfile, reg *harness.Registry, exclude map[string]bool, plats ...platform.ID) {
	t.Helper()
	in := newInst(root, project, "copy", lock, plats...)
	in.ExcludeSkills = exclude
	var err error
	captureStderr(t, func() {
		_, err = captureStdout(t, func() error { return in.InstallDomain(reg, "alpha", true) })
	})
	if err != nil {
		t.Fatalf("install alpha (exclude=%v, plats=%v): %v", exclude, plats, err)
	}
}

func TestRecopyExclusionsThenFullKeepsPlatformUnion(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	installAlphaPlats(t, root, project, lock, reg, map[string]bool{"alpha/sk2": true}, platform.Claude, platform.Cursor)
	installAlphaPlats(t, root, project, lock, reg, nil, platform.Claude)

	assertLockClean(t, lock)
	parent, ok := lock.Find("alpha", "skills", "")
	if !ok || strings.Join(parent.Platforms, ",") != "claude,cursor" {
		t.Fatalf("kind-level platforms = %+v ok=%v", parent.Platforms, ok)
	}
}

func TestRecopyFullThenExclusionsKeepsPlatformUnion(t *testing.T) {
	reg, root := exHarness(t)
	project := t.TempDir()
	lock := &lockfile.Lockfile{}
	installAlphaPlats(t, root, project, lock, reg, nil, platform.Claude, platform.Cursor)
	installAlphaPlats(t, root, project, lock, reg, map[string]bool{"alpha/sk2": true}, platform.Claude)

	assertLockClean(t, lock)
	n := 0
	for _, e := range skillRows(lock) {
		if strings.Join(e.Platforms, ",") != "claude,cursor" {
			t.Fatalf("row %q/%s platforms = %v", e.Name, e.Path, e.Platforms)
		}
		n++
	}
	if n == 0 {
		t.Fatal("no skill rows")
	}
}
