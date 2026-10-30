package cli

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/prompt"
)

var (
	allGit  = []string{"git-conventional-commits", "git-pr-description", "git-release-tag"}
	allCore = []string{"advanced-search", "caveman", "memory-bank", "memory-bank-defrag"}
)

func without(l []string, drop ...string) []string {
	var out []string
	for _, s := range l {
		if !contains(drop, s) {
			out = append(out, s)
		}
	}
	return out
}

func isLink(p string) bool {
	fi, err := os.Lstat(p)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	w.Close()
	os.Stderr = old
	return <-done
}

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitProject(t *testing.T, unignored ...string) string {
	t.Helper()
	p := t.TempDir()
	gitInitDir(t, p)
	for _, d := range unignored {
		writeFile(t, filepath.Join(p, d, "keep"), "x")
	}
	return p
}

func assertNoRows(t *testing.T, p, skill string) {
	t.Helper()
	lock, err := lockfile.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range lock.Entries {
		if e.Name == skill || filepath.Base(e.Path) == skill || contains(e.Items, skill) {
			t.Errorf("lockfile row for excluded skill %s: %+v", skill, e)
		}
	}
}

func customAnswers(p, mode string, git, core []string) map[string]any {
	ans := fullAnswers(p)
	ans["mode"] = mode
	ans["customize"] = true
	ans["skills:git"] = git
	ans["skills:core"] = core
	return ans
}

func TestDialogCustomizeAskedAfterDomainsDefaultNo(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	ans := fullAnswers(p)
	ans["customize"] = false
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "platforms", "mode", "hooks", "confirm")
	if fp.confDef["customize"] {
		t.Error("customize default must be No")
	}
	if strings.Contains(fp.summary, "deselected") {
		t.Errorf("summary warns without exclusions:\n%s", fp.summary)
	}
	for _, s := range append(allGit, allCore...) {
		if !exists(filepath.Join(p, ".claude", "skills", s)) {
			t.Errorf("missing %s", s)
		}
	}
}

func TestDialogCustomizeIDsOrderAndPreselectionAllOn(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	fp := newFake(t, customAnswers(p, "copy", allGit, allCore))
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "skills:git", "skills:core", "platforms", "mode", "hooks", "confirm")
	if got := fp.selected("skills:git"); !reflect.DeepEqual(got, allGit) {
		t.Errorf("git preselected %v", got)
	}
	if got := fp.selected("skills:core"); !reflect.DeepEqual(got, allCore) {
		t.Errorf("core preselected %v", got)
	}
	if strings.Contains(fp.summary, "deselected") {
		t.Errorf("nothing deselected, summary:\n%s", fp.summary)
	}
}

func TestDialogCustomizeBindsPerDomainAndReachesInstaller(t *testing.T) {
	h := realHarness(t)
	cases := []struct {
		name, dom, skill string
		git, core        []string
	}{
		{"second domain only", "core", "caveman", allGit, without(allCore, "caveman")},
		{"first domain only", "git", "git-release-tag", without(allGit, "git-release-tag"), allCore},
	}
	for _, mode := range []string{"link", "copy"} {
		for _, c := range cases {
			t.Run(mode+"/"+c.name, func(t *testing.T) {
				p := t.TempDir()
				fp := newFake(t, customAnswers(p, mode, c.git, c.core))
				enableDialog(t, fp)
				if _, err := run(t, "add", "--harness-root", h); err != nil {
					t.Fatal(err)
				}
				for _, s := range append(append([]string(nil), allGit...), allCore...) {
					got := lexists(filepath.Join(p, ".claude", "skills", s))
					if want := s != c.skill; got != want {
						t.Errorf("%s present=%v want %v", s, got, want)
					}
				}
				assertNoRows(t, p, c.skill)
				for _, f := range []string{".claude/rules/git-conventions.md", ".claude/agents/git-flow.md", ".claude/hooks/memory-bank-load.sh"} {
					if !lexists(filepath.Join(p, f)) {
						t.Errorf("domain part %s missing", f)
					}
				}
				if mode == "link" && !isLink(filepath.Join(p, ".claude", "skills", "memory-bank")) {
					t.Error("link mode: kept skill not a symlink")
				}
				for _, w := range []string{"deselect:  " + c.dom + "/" + c.skill, "warning:   agents may reference deselected skills"} {
					if !strings.Contains(fp.summary, w) {
						t.Errorf("summary lacks %q:\n%s", w, fp.summary)
					}
				}
				if strings.Contains(fp.summary, "already installed, deselected, kept") {
					t.Errorf("nothing was installed before:\n%s", fp.summary)
				}
			})
		}
	}
}

func TestDialogCustomizeNamesPreAnsweredStillAsks(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	ans := fullAnswers(p)
	ans["customize"] = true
	ans["skills:git"] = without(allGit, "git-pr-description")
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "git", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("customize", "skills:git", "platforms", "mode", "confirm")
	if exists(filepath.Join(p, ".claude", "skills", "git-pr-description")) || !exists(filepath.Join(p, ".claude", "skills", "git-release-tag")) {
		t.Error("exclusion not applied")
	}
}

func TestDialogCustomizeNotAskedForStandaloneSkills(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	ans := fullAnswers(p)
	ans["customize"] = true
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "caveman", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("platforms", "mode", "confirm")
}

func TestDialogYesExcludesNothing(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	oldI, oldN := interactive, newPrompter
	interactive = func() bool { return true }
	newPrompter = func() prompt.Prompter { t.Fatal("prompter constructed under -y"); return nil }
	t.Cleanup(func() { interactive, newPrompter = oldI, oldN })
	if _, err := run(t, "add", "core", "-y", "--copy", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	for _, s := range allCore {
		if !exists(filepath.Join(p, ".claude", "skills", s)) {
			t.Errorf("missing %s", s)
		}
	}
}

func TestDialogBackWithinSkillSteps(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	ans := customAnswers(p, "copy", nil, nil)
	noRelease := without(allGit, "git-release-tag")
	ans["skills:git"] = seq{noRelease, noRelease}
	ans["skills:core"] = seq{prompt.ErrBack, without(allCore, "caveman")}
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "skills:git", "skills:core", "skills:git", "skills:core", "platforms", "mode", "hooks", "confirm")
	if got := fp.history["skills:git"]; !reflect.DeepEqual(got, [][]string{allGit, noRelease}) {
		t.Fatalf("git preselection %v", got)
	}
	if exists(filepath.Join(p, ".claude", "skills", "git-release-tag")) || exists(filepath.Join(p, ".claude", "skills", "caveman")) {
		t.Error("exclusions lost")
	}
}

func TestDialogBackFromFirstSkillStepReasksCustomize(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	ans := customAnswers(p, "copy", nil, nil)
	ans["customize"] = seq{true, false}
	ans["skills:git"] = seq{prompt.ErrBack}
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "skills:git", "customize", "platforms", "mode", "hooks", "confirm")
	if !fp.confDef["customize"] {
		t.Error("re-asked customize must default to the previous answer (Yes)")
	}
}

func TestDialogBackFromCustomizeGoesToDomains(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	ans := fullAnswers(p)
	ans["customize"] = seq{prompt.ErrBack, false}
	ans["domains"] = seq{[]string{"git"}, []string{"git"}}
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h, "--dry-run"); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "domains", "customize", "platforms", "mode")
	if got := fp.history["domains"]; !reflect.DeepEqual(got, [][]string{nil, {"git"}}) {
		t.Fatalf("domains preselection %v", got)
	}
}

func TestDialogBackFromPlatformsGoesToCustomizeKeepingAnswers(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	noRelease := without(allGit, "git-release-tag")
	ans := customAnswers(p, "copy", nil, allCore)
	ans["customize"] = seq{true, true}
	ans["skills:git"] = seq{noRelease, noRelease}
	ans["skills:core"] = seq{allCore, allCore}
	ans["platforms"] = seq{prompt.ErrBack, []string{"claude"}}
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "skills:git", "skills:core", "platforms",
		"customize", "skills:git", "skills:core", "platforms", "mode", "hooks", "confirm")
	if got := fp.history["skills:git"]; !reflect.DeepEqual(got, [][]string{allGit, noRelease}) {
		t.Fatalf("git preselection after back %v", got)
	}
	if !fp.confDef["customize"] {
		t.Error("customize default after back must be Yes")
	}
}

func TestDialogPlatformListAnnotatesUnignoredRoots(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t)
	ans := fullAnswers(p)
	ans["confirm"] = false
	fp := newFake(t, ans)
	enableDialog(t, fp)
	_, _ = run(t, "add", "--harness-root", h)
	if l := fp.labelOf("platforms", "cursor"); !strings.Contains(l, ".cursor") || !strings.Contains(l, "not git-ignored") {
		t.Errorf("cursor label %q", l)
	}
	if l := fp.labelOf("platforms", "claude"); strings.Contains(l, "git-ignored") {
		t.Errorf("claude label %q", l)
	}
	writeFile(t, filepath.Join(p, ".gitignore"), ".cursor/\n.agents/\n.codex/\n.opencode/\n")
	fp = newFake(t, fullAnswers(p))
	fp.answers["confirm"] = false
	enableDialog(t, fp)
	_, _ = run(t, "add", "--harness-root", h)
	for _, id := range []string{"claude", "codex", "cursor", "opencode"} {
		if l := fp.labelOf("platforms", id); strings.Contains(l, "git-ignored") {
			t.Errorf("%s label %q after gitignore", id, l)
		}
	}
}

func guardAnswers(p string, plats []string) map[string]any {
	ans := fullAnswers(p)
	ans["domains"] = []string{"git"}
	ans["platforms"] = plats
	ans["mode"] = "link"
	return ans
}

func TestDialogGuardSwitchToCopy(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := guardAnswers(p, []string{"claude", "cursor"})
	ans["guard:cursor"] = "copy"
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "platforms", "mode", "guard:cursor", "confirm")
	var vals []string
	for _, o := range fp.options["guard:cursor"] {
		vals = append(vals, o.Value)
	}
	if !reflect.DeepEqual(vals, []string{"drop", "copy", "abort"}) || fp.selDef["guard:cursor"] != "drop" {
		t.Fatalf("options %v default %q", vals, fp.selDef["guard:cursor"])
	}
	for _, w := range []string{"mode:      copy", "adjusted:  mode switched to copy for all platforms", "platforms: .claude, .cursor"} {
		if !strings.Contains(fp.summary, w) {
			t.Errorf("summary lacks %q:\n%s", w, fp.summary)
		}
	}
	if isLink(filepath.Join(p, ".claude", "skills", "git-conventional-commits")) || !exists(filepath.Join(p, ".claude", "skills", "git-conventional-commits")) {
		t.Error("expected a real copy on claude")
	}
	if !exists(filepath.Join(p, ".cursor", "agents", "git-flow.md")) {
		t.Error("cursor agent not installed")
	}
}

func guardValues(fp *fakePrompt, id string) []string {
	var vals []string
	for _, o := range fp.options[id] {
		vals = append(vals, o.Value)
	}
	return vals
}

func TestDialogGuardDefaultsToDropWithSeveralPlatforms(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := guardAnswers(p, []string{"claude", "cursor"})
	ans["guard:cursor"] = "drop"
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if fp.selDef["guard:cursor"] != "drop" {
		t.Errorf("default %q, want drop", fp.selDef["guard:cursor"])
	}
	if got := fp.history["guard:cursor"]; len(got) != 1 || !reflect.DeepEqual(got[0], []string{"drop"}) {
		t.Errorf("preselected %v, want [[drop]]", got)
	}
}

func TestDialogGuardDefaultsToCopyForOnlyPlatform(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := guardAnswers(p, []string{"cursor"})
	ans["guard:cursor"] = "copy"
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if fp.selDef["guard:cursor"] != "copy" {
		t.Errorf("default %q, want copy", fp.selDef["guard:cursor"])
	}
	if got := fp.history["guard:cursor"]; len(got) != 1 || !reflect.DeepEqual(got[0], []string{"copy"}) {
		t.Errorf("preselected %v, want [[copy]]", got)
	}
	if vals := guardValues(fp, "guard:cursor"); !reflect.DeepEqual(vals, []string{"drop", "copy", "abort"}) {
		t.Errorf("options %v", vals)
	}
}

func TestDialogGuardCopyLabelNamesAllPlatforms(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := guardAnswers(p, []string{"claude", "cursor"})
	ans["guard:cursor"] = "drop"
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	l := fp.labelOf("guard:cursor", "copy")
	for _, w := range []string{"ALL platforms", "one install mode per run", "other platforms become copies too"} {
		if !strings.Contains(l, w) {
			t.Errorf("copy label %q lacks %q", l, w)
		}
	}
}

func TestDialogGuardPreviousChoiceBeatsDropDefaultAfterBack(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := guardAnswers(p, []string{"claude", "cursor"})
	ans["guard:cursor"] = seq{"copy", "copy"}
	ans["confirm"] = seq{prompt.ErrBack, true}
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if got := fp.history["guard:cursor"]; len(got) != 2 || !reflect.DeepEqual(got[0], []string{"drop"}) || !reflect.DeepEqual(got[1], []string{"copy"}) {
		t.Fatalf("preselection per ask %v, want [[drop] [copy]]", got)
	}
	if fp.selDef["guard:cursor"] != "copy" {
		t.Errorf("re-asked default %q, want previous choice copy", fp.selDef["guard:cursor"])
	}
}

func TestDialogGuardDropPlatform(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := guardAnswers(p, []string{"claude", "cursor"})
	ans["guard:cursor"] = "drop"
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "platforms", "mode", "guard:cursor", "confirm")
	for _, w := range []string{"platforms: .claude (", "adjusted:  dropped platform cursor"} {
		if !strings.Contains(fp.summary, w) {
			t.Errorf("summary lacks %q:\n%s", w, fp.summary)
		}
	}
	if !isLink(filepath.Join(p, ".claude", "skills", "git-conventional-commits")) {
		t.Error("link mode must stay for the remaining platform")
	}
	if exists(filepath.Join(p, ".cursor", "agents")) || exists(filepath.Join(p, ".agents")) {
		t.Error("dropped platform was installed")
	}
}

func TestDialogGuardAbortWritesNothing(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := guardAnswers(p, []string{"claude", "cursor"})
	ans["guard:cursor"] = "abort"
	enableDialog(t, newFake(t, ans))
	if _, err := run(t, "add", "--harness-root", h); err == nil || err.Error() != "aborted" {
		t.Fatalf("got %v", err)
	}
	for _, d := range []string{".claude", ".agents", ".cursor/agents", ".cursor/rules"} {
		if exists(filepath.Join(p, d)) {
			t.Errorf("%s written", d)
		}
	}
}

func TestDialogGuardDropLastPlatformAborts(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := guardAnswers(p, []string{"cursor"})
	ans["guard:cursor"] = "drop"
	enableDialog(t, newFake(t, ans))
	if _, err := run(t, "add", "--harness-root", h); err == nil || err.Error() != "aborted" {
		t.Fatalf("got %v", err)
	}
	if exists(filepath.Join(p, ".agents")) || exists(filepath.Join(p, ".claude")) {
		t.Error("written")
	}
}

func TestDialogGuardRecollectsUntilClean(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor", ".codex")
	ans := guardAnswers(p, []string{"codex", "cursor"})
	ans["guard:codex"] = "drop"
	ans["guard:cursor"] = "copy"
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "platforms", "mode", "guard:codex", "guard:cursor", "confirm")
	if exists(filepath.Join(p, ".codex", "agents")) {
		t.Error("dropped codex installed")
	}
	if !exists(filepath.Join(p, ".agents", "skills", "git-conventional-commits")) || isLink(filepath.Join(p, ".agents", "skills", "git-conventional-commits")) {
		t.Error("cursor skills not copied to .agents/skills")
	}
}

func TestDialogBackFromGuardGoesToMode(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := guardAnswers(p, []string{"claude", "cursor"})
	ans["mode"] = seq{"link", "link"}
	ans["guard:cursor"] = seq{prompt.ErrBack, "drop"}
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "platforms", "mode", "guard:cursor", "mode", "guard:cursor", "confirm")
	if got := fp.history["mode"]; len(got) != 2 || !reflect.DeepEqual(got[1], []string{"link"}) {
		t.Fatalf("mode preselection %v", got)
	}
}

func TestDialogBackFromConfirmReasksGuardKeepingChoice(t *testing.T) {
	h := realHarness(t)
	p := gitProject(t, ".cursor")
	ans := guardAnswers(p, []string{"claude", "cursor"})
	ans["guard:cursor"] = seq{"copy", "drop"}
	ans["confirm"] = seq{prompt.ErrBack, true}
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "platforms", "mode", "guard:cursor", "confirm", "guard:cursor", "confirm")
	if fp.selDef["guard:cursor"] != "copy" {
		t.Errorf("re-asked default %q, want previous choice copy", fp.selDef["guard:cursor"])
	}
	if !isLink(filepath.Join(p, ".claude", "skills", "git-conventional-commits")) || exists(filepath.Join(p, ".cursor", "agents")) {
		t.Error("final state must be link mode, cursor dropped")
	}
}

func foreignCore(t *testing.T, p string) {
	t.Helper()
	for _, s := range []string{"caveman", "memory-bank"} {
		writeFile(t, filepath.Join(p, ".claude", "skills", s, "mine.txt"), s)
	}
	writeFile(t, filepath.Join(p, ".claude", "rules", "mine.md"), "rule")
}

func coreLinkAnswers(p string) map[string]any {
	ans := fullAnswers(p)
	ans["domains"] = []string{"core"}
	ans["mode"] = "link"
	ans["customize"] = false
	return ans
}

func TestDialogOverwriteYesWritesOnlyListedPaths(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	foreignCore(t, p)
	writeFile(t, filepath.Join(p, ".claude", "skills", "advanced-search", "mine.txt"), "excluded")
	ans := coreLinkAnswers(p)
	ans["customize"] = true
	ans["skills:core"] = without(allCore, "advanced-search")
	ans["overwrite"] = true
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "skills:core", "platforms", "mode", "overwrite", "hooks", "confirm")
	if want := ".claude/skills/caveman\n.claude/skills/memory-bank"; fp.descs["overwrite"] != want {
		t.Fatalf("listed paths %q, want %q", fp.descs["overwrite"], want)
	}
	if fp.confDef["overwrite"] {
		t.Error("overwrite default must be No")
	}
	if !strings.Contains(fp.summary, "adjusted:  overwrite 2 untracked path(s)") {
		t.Errorf("summary:\n%s", fp.summary)
	}
	for _, s := range []string{"caveman", "memory-bank", "memory-bank-defrag"} {
		if !isLink(filepath.Join(p, ".claude", "skills", s)) {
			t.Errorf("%s not linked", s)
		}
	}
	for _, f := range []string{".claude/skills/advanced-search/mine.txt", ".claude/rules/mine.md"} {
		if !exists(filepath.Join(p, f)) {
			t.Errorf("unlisted %s was touched", f)
		}
	}
	if isLink(filepath.Join(p, ".claude", "skills", "advanced-search")) {
		t.Error("excluded foreign dir replaced")
	}
}

func TestDialogOverwriteNoAbortsWritingNothing(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	foreignCore(t, p)
	ans := coreLinkAnswers(p)
	ans["overwrite"] = false
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err == nil || err.Error() != "aborted" {
		t.Fatalf("got %v", err)
	}
	fp.wantAsked("project", "domains", "customize", "platforms", "mode", "overwrite")
	if exists(filepath.Join(p, ".claude", "skills-lock.json")) {
		t.Error("lockfile written")
	}
	ents, _ := os.ReadDir(filepath.Join(p, ".claude", "skills"))
	if len(ents) != 2 {
		t.Errorf("skills dir changed: %v", ents)
	}
	for _, s := range []string{"caveman", "memory-bank"} {
		if isLink(filepath.Join(p, ".claude", "skills", s)) || !exists(filepath.Join(p, ".claude", "skills", s, "mine.txt")) {
			t.Errorf("foreign %s modified", s)
		}
	}
	if exists(filepath.Join(p, ".claude", "hooks")) {
		t.Error("hooks written")
	}
}

func TestDialogBackFromOverwriteGoesToMode(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	foreignCore(t, p)
	ans := coreLinkAnswers(p)
	ans["mode"] = seq{"link", "copy"}
	ans["overwrite"] = seq{prompt.ErrBack}
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "customize", "platforms", "mode", "overwrite", "mode", "hooks", "confirm")
	if isLink(filepath.Join(p, ".claude", "skills", "advanced-search")) {
		t.Error("copy mode expected after going back")
	}
}

func TestDialogReservedAgentNameIsHardError(t *testing.T) {
	h, src := agentHarness(t)
	if err := os.WriteFile(src, []byte("---\nname: default\ndescription: d\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := t.TempDir()
	fp := newFake(t, map[string]any{"customize": false, "confirm": true})
	enableDialog(t, fp)
	_, err := run(t, "add", "testdomain", "-a", "codex", "--copy", "-p", p, "--harness-root", h)
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("got %v", err)
	}
	fp.wantAsked("customize")
	if ents, _ := os.ReadDir(p); len(ents) != 0 {
		t.Errorf("project not empty: %v", ents)
	}
}

func TestYesGuardConflictFailsBeforeWrite(t *testing.T) {
	h := realHarness(t)
	for _, agents := range []string{"cursor", "claude,cursor"} {
		t.Run(agents, func(t *testing.T) {
			p := gitProject(t, ".cursor")
			_, err := run(t, "add", "git", "-y", "-a", agents, "-p", p, "--harness-root", h)
			if err == nil {
				t.Fatal("want conflict error")
			}
			for _, w := range []string{".cursor is not git-ignored", "-a claude", "--copy"} {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error lacks %q: %v", w, err)
				}
			}
			for _, d := range []string{".claude", ".agents", ".cursor/agents"} {
				if exists(filepath.Join(p, d)) {
					t.Errorf("%s written", d)
				}
			}
		})
	}
}

func TestYesForeignConflictFailsBeforeWrite(t *testing.T) {
	h := realHarness(t)
	for name, args := range map[string][]string{
		"domain": {"core"},
		"skills": {"advanced-search", "caveman", "memory-bank"},
	} {
		t.Run(name, func(t *testing.T) {
			p := t.TempDir()
			foreignCore(t, p)
			_, err := run(t, append(append([]string{"add"}, args...), "-y", "-a", "claude", "-p", p, "--harness-root", h)...)
			if err == nil {
				t.Fatal("want conflict error")
			}
			for _, w := range []string{filepath.Join(p, ".claude", "skills", "caveman"), "not tracked by as-skill", "hint:", "nothing written"} {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error lacks %q: %v", w, err)
				}
			}
			if exists(filepath.Join(p, ".claude", "skills", "advanced-search")) || exists(filepath.Join(p, ".claude", "skills-lock.json")) || exists(filepath.Join(p, ".claude", "hooks")) {
				t.Error("partial write before the conflict")
			}
		})
	}
}

func TestDialogSummaryListsDeselectedAlreadyInstalled(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	if _, err := run(t, "add", "caveman", "-y", "--copy", "-a", "claude", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	ans := fullAnswers(p)
	ans["domains"] = []string{"core"}
	ans["customize"] = true
	ans["skills:core"] = without(allCore, "caveman", "advanced-search")
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{
		"warning:   agents may reference deselected skills",
		"warning:   core/caveman: already installed, deselected, kept — use remove",
		"deselect:  core/advanced-search, core/caveman",
	} {
		if !strings.Contains(fp.summary, w) {
			t.Errorf("summary lacks %q:\n%s", w, fp.summary)
		}
	}
	if strings.Contains(fp.summary, "core/advanced-search: already installed") {
		t.Errorf("absent skill reported as installed:\n%s", fp.summary)
	}
	if !exists(filepath.Join(p, ".claude", "skills", "caveman")) {
		t.Error("deselected-but-installed skill was pruned")
	}
	if exists(filepath.Join(p, ".claude", "skills", "advanced-search")) {
		t.Error("deselected skill installed")
	}
}

func TestRemoveDomainKeepsForeignExcludedSkill(t *testing.T) {
	h := realHarness(t)
	for _, mode := range []string{"link", "copy"} {
		t.Run(mode, func(t *testing.T) {
			p := t.TempDir()
			writeFile(t, filepath.Join(p, ".claude", "skills", "caveman", "mine.txt"), "foreign")
			ans := fullAnswers(p)
			ans["domains"] = []string{"core"}
			ans["mode"] = mode
			ans["customize"] = true
			ans["skills:core"] = without(allCore, "caveman", "advanced-search")
			enableDialog(t, newFake(t, ans))
			if _, err := run(t, "add", "--harness-root", h); err != nil {
				t.Fatal(err)
			}
			oldI := interactive
			interactive = func() bool { return false }
			defer func() { interactive = oldI }()
			var rmErr error
			stderr := captureStderr(t, func() {
				_, rmErr = run(t, "remove", "core", "-y", "-p", p, "--harness-root", h)
			})
			if rmErr != nil {
				t.Fatal(rmErr)
			}
			b, err := os.ReadFile(filepath.Join(p, ".claude", "skills", "caveman", "mine.txt"))
			if err != nil || string(b) != "foreign" {
				t.Fatalf("foreign excluded skill touched: %v %q", err, b)
			}
			if strings.Contains(stderr, "advanced-search") {
				t.Errorf("warning for excluded absent skill:\n%s", stderr)
			}
			for _, s := range []string{"memory-bank", "memory-bank-defrag"} {
				if lexists(filepath.Join(p, ".claude", "skills", s)) {
					t.Errorf("%s survived remove", s)
				}
			}
			lock, _ := lockfile.Load(p)
			for _, e := range lock.Entries {
				if e.Domain == "core" {
					t.Errorf("row survived: %+v", e)
				}
			}
		})
	}
}

func TestDialogSummaryNoKeptHintForForeignUntracked(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	writeFile(t, filepath.Join(p, ".claude", "skills", "caveman", "mine.txt"), "foreign")
	ans := fullAnswers(p)
	ans["domains"] = []string{"core"}
	ans["customize"] = true
	ans["skills:core"] = without(allCore, "caveman")
	fp := newFake(t, ans)
	enableDialog(t, fp)
	if _, err := run(t, "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fp.summary, "deselect:  core/caveman") {
		t.Errorf("summary lacks deselect line:\n%s", fp.summary)
	}
	if strings.Contains(fp.summary, "already installed, deselected, kept") {
		t.Errorf("untracked foreign dir reported as installed:\n%s", fp.summary)
	}
}

func TestUnignoredRootsSkippedForCopyAndForce(t *testing.T) {
	p := gitProject(t, ".cursor", ".codex")
	if got := unignoredRoots(p, false, false); len(got) == 0 {
		t.Fatal("baseline: link mode must report unignored roots")
	}
	if got := unignoredRoots(p, true, false); len(got) != 0 {
		t.Errorf("--copy: %v", got)
	}
	if got := unignoredRoots(p, false, true); len(got) != 0 {
		t.Errorf("--force: %v", got)
	}
}

func TestDialogPlatformListNoUnignoredNoteUnderCopyAndForce(t *testing.T) {
	h := realHarness(t)
	for _, flag := range []string{"--copy", "--force"} {
		t.Run(flag, func(t *testing.T) {
			p := gitProject(t)
			ans := fullAnswers(p)
			ans["confirm"] = false
			fp := newFake(t, ans)
			enableDialog(t, fp)
			_, _ = run(t, "add", "--harness-root", h, flag)
			for _, o := range fp.options["platforms"] {
				if strings.Contains(o.Label, "not git-ignored") {
					t.Errorf("label %q under %s", o.Label, flag)
				}
			}
		})
	}
}

func TestDialogCustomizeRejectsEmptySkillSelection(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	ans := customAnswers(p, "copy", allGit, allCore)
	ans["confirm"] = false
	fp := newFake(t, ans)
	enableDialog(t, fp)
	_, _ = run(t, "add", "--harness-root", h)
	v := fp.validate["skills:git"]
	if v == nil {
		t.Fatal("skills:git has no validator")
	}
	err := v(nil)
	if err == nil || !strings.Contains(err.Error(), "select at least one skill") || !strings.Contains(err.Error(), "don't choose the domain") {
		t.Errorf("empty accepted or unclear message: %v", err)
	}
	if err := v([]string{"git-release-tag"}); err != nil {
		t.Errorf("non-empty rejected: %v", err)
	}
}

func TestDialogCustomizeEmptyAnswerWritesNothing(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	enableDialog(t, newFake(t, customAnswers(p, "copy", []string{}, allCore)))
	_, err := run(t, "add", "--harness-root", h)
	if err == nil || !strings.Contains(err.Error(), "select at least one skill") {
		t.Fatalf("got %v", err)
	}
	if ents, _ := os.ReadDir(p); len(ents) != 0 {
		t.Fatalf("project not empty: %v", ents)
	}
}
