package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mkTree(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	for _, d := range []string{"projects", "Pictures", "docs", ".hidden", ".dots"} {
		if err := os.MkdirAll(filepath.Join(tmp, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmp, "pfile"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(tmp, "docs"), filepath.Join(tmp, "plink")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(tmp, "pfile"), filepath.Join(tmp, "pfilelink")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "projects", "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	return tmp
}

func mkDirs(t *testing.T, names ...string) string {
	t.Helper()
	tmp := t.TempDir()
	for _, n := range names {
		if err := os.MkdirAll(filepath.Join(tmp, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return tmp
}

func TestDirMatches(t *testing.T) {
	tmp := mkTree(t)
	for _, tc := range []struct {
		name, in string
		want     []string
	}{
		{"empty", "", nil},
		{"tilde alone", "~", []string{"~/"}},
		{"tilde prefix keeps tilde", "~/pro", []string{"~/projects/"}},
		{"tilde substring", "~/ject", []string{"~/projects/"}},
		{"tilde slash lists children without dots", "~/", []string{"~/Pictures/", "~/docs/", "~/plink/", "~/projects/"}},
		{"case-insensitive", "~/pic", []string{"~/Pictures/"}},
		{"case-insensitive upper", "~/PIC", []string{"~/Pictures/"}},
		{"files excluded, dir symlink included, file symlink excluded", "~/p", []string{"~/Pictures/", "~/plink/", "~/projects/"}},
		{"dot typed shows dot dirs", "~/.", []string{"~/.dots/", "~/.hidden/"}},
		{"dot base substring", "~/.hid", []string{"~/.hidden/"}},
		{"hidden excluded for plain base", "~/hidden", nil},
		{"trailing slash lists children", "~/projects/", []string{"~/projects/alpha/"}},
		{"absolute substring", tmp + "/oc", []string{tmp + "/docs/"}},
		{"nonexistent dir", tmp + "/nope/x", nil},
		{"no match", "~/zzz", nil},
	} {
		if got := dirMatches(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: dirMatches(%q) = %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestDirMatchesSubstringAndOrder(t *testing.T) {
	tmp := mkDirs(t, "github", "gitlab", "digit", "docs", ".github", "git", "legit")
	for _, tc := range []struct {
		name, in string
		want     []string
	}{
		{"substring", tmp + "/hub", []string{tmp + "/github/"}},
		{"upper", tmp + "/HUB", []string{tmp + "/github/"}},
		{"mixed case", tmp + "/Git", []string{tmp + "/git/", tmp + "/github/", tmp + "/gitlab/", tmp + "/digit/", tmp + "/legit/"}},
		{"exact then prefix then substring", tmp + "/git", []string{tmp + "/git/", tmp + "/github/", tmp + "/gitlab/", tmp + "/digit/", tmp + "/legit/"}},
		{"plain base excludes dot dir", tmp + "/hub", []string{tmp + "/github/"}},
		{"dot base finds dot dir", tmp + "/.git", []string{tmp + "/.github/"}},
		{"dirPart verbatim", tmp + "//hub", []string{tmp + "//github/"}},
	} {
		if got := dirMatches(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: dirMatches(%q) = %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestDirMatchesRelative(t *testing.T) {
	tmp := mkTree(t)
	t.Chdir(filepath.Join(tmp, "projects"))
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"./al", []string{"./alpha/"}},
		{"./lph", []string{"./alpha/"}},
		{"../oc", []string{"../docs/"}},
		{"al", []string{"alpha/"}},
		{"lph", []string{"alpha/"}},
		{"./", []string{"./alpha/"}},
	} {
		if got := dirMatches(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("dirMatches(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestDirMatchesReturnsAllSorted(t *testing.T) {
	tmp := t.TempDir()
	for i := 0; i < 60; i++ {
		if err := os.Mkdir(filepath.Join(tmp, fmt.Sprintf("d%03d", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got := dirMatches(tmp + "/d")
	if len(got) != 60 {
		t.Fatalf("len %d want 60", len(got))
	}
	if got[0] != tmp+"/d000/" || got[59] != tmp+"/d059/" {
		t.Fatalf("not sorted: first %q last %q", got[0], got[len(got)-1])
	}
}

func TestSplitPath(t *testing.T) {
	for _, tc := range []struct{ in, dir, base string }{
		{"abc", "", "abc"},
		{"/a/b", "/a/", "b"},
		{"/a/b/", "/a/b/", ""},
		{"~/x", "~/", "x"},
		{"", "", ""},
	} {
		dir, base := splitPath(tc.in)
		if dir != tc.dir || base != tc.base {
			t.Errorf("splitPath(%q) = %q,%q want %q,%q", tc.in, dir, base, tc.dir, tc.base)
		}
	}
}

func TestCommonPrefix(t *testing.T) {
	for _, tc := range []struct {
		name  string
		items []string
		want  string
	}{
		{"empty", nil, ""},
		{"single", []string{"abc/"}, "abc/"},
		{"shared", []string{"abc1/", "abc2/", "abd/"}, "ab"},
		{"multibyte", []string{"проект1/", "проект2/"}, "проект"},
		{"none", []string{"a", "b"}, ""},
	} {
		if got := commonPrefix(tc.items); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestTabResult(t *testing.T) {
	tmp := mkDirs(t, "alpha1", "alpha2", "beta", "github", "gitlab", "digit", "docs/one", "docs/two", "only/child", "проект1", "проект2")
	for _, tc := range []struct {
		name, in, want string
		matches        int
	}{
		{"single prefix", tmp + "/be", tmp + "/beta/", 1},
		{"single substring replaces", tmp + "/hub", tmp + "/github/", 1},
		{"single substring shortens text", tmp + "/ithu", tmp + "/github/", 1},
		{"all prefix extends by lcp", tmp + "/al", tmp + "/alpha", 2},
		{"lcp not longer leaves value", tmp + "/alpha", tmp + "/alpha", 2},
		{"mixed leaves value", tmp + "/git", tmp + "/git", 3},
		{"pure substring never extends", tmp + "/it", tmp + "/it", 3},
		{"multibyte lcp", tmp + "/про", tmp + "/проект", 2},
		{"no match", tmp + "/zzz", tmp + "/zzz", 0},
		{"empty", "", "", 0},
		{"dir slash one child", tmp + "/only/", tmp + "/only/child/", 1},
		{"dir slash two children", tmp + "/docs/", tmp + "/docs/", 2},
		{"idempotent", tmp + "/alpha", tmp + "/alpha", 2},
	} {
		got, matches := tabResult(tc.in)
		if got != tc.want || len(matches) != tc.matches {
			t.Errorf("%s: tabResult(%q) = %q,%d want %q,%d", tc.name, tc.in, got, len(matches), tc.want, tc.matches)
		}
	}
	home := mkTree(t)
	if got, m := tabResult("~"); got != "~/" || len(m) != 1 {
		t.Errorf("tilde: %q %v", got, m)
	}
	if got, _ := tabResult(home + "/doc"); got != home+"/docs/" {
		t.Errorf("abs: %q", got)
	}
}

func TestWrapSel(t *testing.T) {
	for _, tc := range []struct{ sel, delta, total, want int }{
		{-1, 1, 3, 0},
		{0, 1, 3, 1},
		{2, 1, 3, 0},
		{-1, -1, 3, 2},
		{2, -1, 3, 1},
		{0, -1, 3, 2},
		{0, 1, 0, -1},
		{-1, -1, 0, -1},
	} {
		if got := wrapSel(tc.sel, tc.delta, tc.total); got != tc.want {
			t.Errorf("wrapSel(%d,%d,%d) = %d want %d", tc.sel, tc.delta, tc.total, got, tc.want)
		}
	}
}

func TestScrollOff(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		off, sel, total, rows, want int
	}{
		{"fits", 3, 2, 5, 10, 0},
		{"no selection keeps off", 2, -1, 15, 10, 2},
		{"selection in window keeps off", 2, 5, 15, 10, 2},
		{"below window scrolls minimally", 0, 9, 15, 10, 1},
		{"above window scrolls up", 5, 3, 15, 10, 3},
		{"wrap to last", 0, 14, 15, 10, 6},
		{"wrap to first", 6, 0, 15, 10, 0},
		{"clamps off", 99, -1, 15, 10, 6},
		{"rows too small", 4, 2, 15, 1, 0},
	} {
		if got := scrollOff(tc.off, tc.sel, tc.total, tc.rows); got != tc.want {
			t.Errorf("%s: scrollOff(%d,%d,%d,%d) = %d want %d", tc.name, tc.off, tc.sel, tc.total, tc.rows, got, tc.want)
		}
	}
}

func TestRenderMatches(t *testing.T) {
	many := make([]string, 15)
	for i := range many {
		many[i] = fmt.Sprintf("/x/d%02d/", i)
	}
	for _, tc := range []struct {
		name           string
		matches        []string
		sel, off, rows int
		want           string
	}{
		{"nil", nil, -1, 0, 10, ""},
		{"one", []string{"/x/a/"}, -1, 0, 10, ""},
		{"plain rows", []string{"/x/a/", "/x/b/", "/x/c/"}, -1, 0, 10, "  a/\n  b/\n  c/"},
		{"marker", []string{"/x/a/", "/x/b/", "/x/c/"}, 1, 0, 10, "  a/\n> b/\n  c/"},
		{"rows one", []string{"/x/a/", "/x/b/"}, -1, 0, 1, ""},
		{"rows zero", []string{"/x/a/", "/x/b/"}, -1, 0, 0, ""},
		{"relative", []string{"a1/", "a2/"}, -1, 0, 5, "  a1/\n  a2/"},
		{"exactly rows", many[:10], -1, 0, 10, "  d00/\n  d01/\n  d02/\n  d03/\n  d04/\n  d05/\n  d06/\n  d07/\n  d08/\n  d09/"},
		{"overflow top", many, -1, 0, 10, "  d00/\n  d01/\n  d02/\n  d03/\n  d04/\n  d05/\n  d06/\n  d07/\n  d08/\n  … +6 more"},
		{"overflow offset", many, 8, 3, 10, "  d03/\n  d04/\n  d05/\n  d06/\n  d07/\n> d08/\n  d09/\n  d10/\n  d11/\n  … +6 more"},
		{"overflow bottom", many, 14, 6, 10, "  d06/\n  d07/\n  d08/\n  d09/\n  d10/\n  d11/\n  d12/\n  d13/\n> d14/\n  … +6 more"},
		{"off clamped", many, -1, 99, 10, "  d06/\n  d07/\n  d08/\n  d09/\n  d10/\n  d11/\n  d12/\n  d13/\n  d14/\n  … +6 more"},
	} {
		got := renderMatches(tc.matches, tc.sel, tc.off, tc.rows)
		if got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
		if got != "" && strings.Count(got, "\n")+1 > tc.rows {
			t.Errorf("%s: more than %d lines", tc.name, tc.rows)
		}
	}
}
