package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/prompt"
)

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

func fullAnswers(project string) map[string]any {
	return map[string]any{
		"project":   project,
		"domains":   []string{"git", "core"},
		"platforms": []string{"claude"},
		"mode":      "copy",
		"confirm":   true,
	}
}

func TestDialogAddFullOrderAndMultiDomain(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	fp := newFake(t, fullAnswers(p))
	enableDialog(t, fp)
	out, err := run(t, "add", "--harness-root", h)
	if err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("project", "domains", "platforms", "mode", "confirm")
	for _, s := range []string{".claude/skills/git-conventional-commits", ".claude/skills/memory-bank", ".claude/skills-lock.json"} {
		if !exists(filepath.Join(p, s)) {
			t.Errorf("missing %s (out: %s)", s, out)
		}
	}
	for _, want := range []string{"harness:", "project:   " + p, "domains:   git, core", "platforms: .claude (default)", "mode:      copy"} {
		if !strings.Contains(fp.summary, want) {
			t.Errorf("summary lacks %q:\n%s", want, fp.summary)
		}
	}
}

func TestDialogPreAnswersSkipExactlyTheirQuestion(t *testing.T) {
	h := realHarness(t)
	cases := []struct {
		name  string
		extra func(p string) []string
		skip  string
	}{
		{"names", func(p string) []string { return []string{"git"} }, "domains"},
		{"agent", func(p string) []string { return []string{"-a", "claude"} }, "platforms"},
		{"project", func(p string) []string { return []string{"-p", p} }, "project"},
		{"copy", func(p string) []string { return []string{"--copy"} }, "mode"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := t.TempDir()
			ans := fullAnswers(p)
			var want []string
			for _, id := range []string{"project", "domains", "platforms", "mode", "confirm"} {
				if id != c.skip {
					want = append(want, id)
				}
			}
			fp := newFake(t, ans)
			enableDialog(t, fp)
			args := append([]string{"add", "--harness-root", h}, c.extra(p)...)
			if _, err := run(t, args...); err != nil {
				t.Fatal(err)
			}
			fp.wantAsked(want...)
			if !exists(filepath.Join(p, ".claude", "skills-lock.json")) {
				t.Fatal("nothing installed")
			}
		})
	}
}

func TestDialogYesNeverConstructsPrompter(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	oldI, oldN := interactive, newPrompter
	interactive = func() bool { return true }
	newPrompter = func() prompt.Prompter { t.Fatal("prompter constructed under -y"); return nil }
	t.Cleanup(func() { interactive, newPrompter = oldI, oldN })
	if _, err := run(t, "add", "git", "-y", "-p", p, "--copy", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "remove", "git", "-y", "-p", p); err != nil {
		t.Fatal(err)
	}
	// -y without names still hits the gate, no prompter
	if _, err := run(t, "add", "-y", "-p", p, "--harness-root", h); err == nil || !strings.Contains(err.Error(), "needs a terminal") {
		t.Fatalf("want gate error mentioning terminal, got %v", err)
	}
}

func TestDialogDeclineWritesNothing(t *testing.T) {
	h := realHarness(t)
	for name, conf := range map[string]any{"no": false, "ctrl-c": prompt.ErrAborted} {
		t.Run(name, func(t *testing.T) {
			p := t.TempDir()
			ans := fullAnswers(p)
			ans["confirm"] = conf
			fp := newFake(t, ans)
			enableDialog(t, fp)
			_, err := run(t, "add", "--harness-root", h)
			if err == nil || err.Error() != "aborted" {
				t.Fatalf("want aborted, got %v", err)
			}
			ents, _ := os.ReadDir(p)
			if len(ents) != 0 {
				t.Fatalf("project not empty: %v", ents)
			}
		})
	}
}

func TestDialogCtrlCMidwayWritesNothing(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	ans := fullAnswers(p)
	ans["platforms"] = prompt.ErrAborted
	enableDialog(t, newFake(t, ans))
	if _, err := run(t, "add", "--harness-root", h); err == nil || err.Error() != "aborted" {
		t.Fatalf("got %v", err)
	}
	if ents, _ := os.ReadDir(p); len(ents) != 0 {
		t.Fatalf("project not empty: %v", ents)
	}
}

func TestDialogGatedDomainRejectedWhileEnvUnset(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	t.Setenv("OBSIDIAN_VAULT", "")
	ans := fullAnswers(p)
	ans["confirm"] = false
	fp := newFake(t, ans)
	enableDialog(t, fp)
	_, _ = run(t, "add", "--harness-root", h)
	if l := fp.labelOf("domains", "obsidian"); !strings.Contains(l, "needs $OBSIDIAN_VAULT") {
		t.Fatalf("label %q", l)
	}
	v := fp.validate["domains"]
	if v([]string{"git", "obsidian"}) == nil {
		t.Error("gated selection accepted while env unset")
	}
	if v(nil) == nil {
		t.Error("empty selection accepted")
	}
	if err := v([]string{"git"}); err != nil {
		t.Errorf("git rejected: %v", err)
	}
	t.Setenv("OBSIDIAN_VAULT", t.TempDir())
	if err := v([]string{"obsidian"}); err != nil {
		t.Errorf("rejected with env set: %v", err)
	}
}

func TestDialogInstalledLabel(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	if _, err := run(t, "add", "git", "-y", "-p", p, "--copy", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	ans := fullAnswers(p)
	ans["confirm"] = false
	fp := newFake(t, ans)
	enableDialog(t, fp)
	_, _ = run(t, "add", "--harness-root", h)
	if !strings.Contains(fp.labelOf("domains", "git"), "[installed]") {
		t.Errorf("git label %q", fp.labelOf("domains", "git"))
	}
	if strings.Contains(fp.labelOf("domains", "core"), "[installed]") {
		t.Errorf("core label %q", fp.labelOf("domains", "core"))
	}
}

func TestDialogPlatformPreselectionIsDetected(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	if err := os.Mkdir(filepath.Join(p, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	ans := fullAnswers(p)
	ans["confirm"] = false
	fp := newFake(t, ans)
	enableDialog(t, fp)
	_, _ = run(t, "add", "--harness-root", h)
	if got := fp.selected("platforms"); !reflect.DeepEqual(got, []string{"codex"}) {
		t.Fatalf("preselected %v", got)
	}
	if n := len(fp.options["platforms"]); n < 4 {
		t.Fatalf("platform options %d", n)
	}
}

func TestDialogRemoveOptionsFromLockfile(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	if _, err := run(t, "add", "git", "-y", "-p", p, "--copy", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "add", "caveman", "-y", "-p", p, "--copy", "-a", "claude", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp := newFake(t, map[string]any{"remove": []string{"git", "caveman"}, "confirm": true})
	enableDialog(t, fp)
	if _, err := run(t, "remove", "-p", p); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("remove", "confirm")
	var vals []string
	for _, o := range fp.options["remove"] {
		vals = append(vals, o.Value)
	}
	if want := []string{"git", "caveman"}; !reflect.DeepEqual(vals, want) {
		t.Fatalf("options %v want %v", vals, want)
	}
	lock, _ := lockfile.Load(p)
	for _, e := range lock.Entries {
		if e.Domain == "git" || e.Name == "caveman" {
			t.Errorf("row survived: %+v", e)
		}
	}
	if exists(filepath.Join(p, ".claude", "skills", "git-conventional-commits")) {
		t.Error("git skill still on disk")
	}
}

func TestDialogRemovePlatformsAskedOnlyWhenMultiple(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	if _, err := run(t, "add", "git", "-y", "-p", p, "--copy", "-a", "claude,codex", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp := newFake(t, map[string]any{"remove": []string{"git"}, "platforms": []string{"claude", "codex"}, "confirm": false})
	enableDialog(t, fp)
	if _, err := run(t, "remove", "-p", p); err == nil || err.Error() != "aborted" {
		t.Fatalf("got %v", err)
	}
	fp.wantAsked("remove", "platforms", "confirm")
	if got := fp.selected("platforms"); !reflect.DeepEqual(got, []string{"claude", "codex"}) {
		t.Fatalf("preselected %v", got)
	}
	lock, _ := lockfile.Load(p)
	if len(lock.Entries) == 0 {
		t.Fatal("declined remove wiped the lockfile")
	}
}

func TestNonInteractiveGateMentionsTerminal(t *testing.T) {
	_, err := run(t, "add", "git")
	if err == nil || !strings.Contains(err.Error(), "needs a terminal") {
		t.Fatalf("got %v", err)
	}
}

func TestDialogRemoveStandaloneSkillOption(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	if _, err := run(t, "add", "caveman", "-y", "-p", p, "--copy", "-a", "claude", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp := newFake(t, map[string]any{"remove": []string{"caveman"}, "confirm": true})
	enableDialog(t, fp)
	if _, err := run(t, "remove", "-p", p); err != nil {
		t.Fatal(err)
	}
	fp.wantAsked("remove", "confirm")
	if got := fp.options["remove"]; len(got) != 1 || got[0].Value != "caveman" || !strings.Contains(got[0].Label, "skill") {
		t.Fatalf("options %+v", got)
	}
	if exists(filepath.Join(p, ".claude", "skills", "caveman")) {
		t.Error("skill still on disk")
	}
}

func TestDialogPlatformNoDetectedNoteInEmptyProject(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	ans := fullAnswers(p)
	ans["confirm"] = false
	fp := newFake(t, ans)
	enableDialog(t, fp)
	_, _ = run(t, "add", "--harness-root", h)
	for _, o := range fp.options["platforms"] {
		if strings.Contains(o.Label, "detected") {
			t.Errorf("label %q", o.Label)
		}
	}
}
