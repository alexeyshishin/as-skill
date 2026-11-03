package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectPromptTitles(t *testing.T) {
	fp := newFake(t, map[string]any{
		"domains": seq{[]string{"git"}, []string{"git"}},
		"mode":    "copy",
		"hooks":   []string{},
		"confirm": true,
		"project": "x",
	})
	proj := ""
	w := withProject(fp, &proj)

	if _, err := w.MultiSelect("domains", "Domains", nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := fp.titles["domains"]; got != "Domains" {
		t.Errorf("empty project: title %q", got)
	}

	proj = "/work/app"
	if _, err := w.MultiSelect("domains", "Domains", nil, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := fp.titles["domains"], "Domains · "+shortPath("/work/app"); got != want {
		t.Errorf("multiselect title %q want %q", got, want)
	}
	if _, err := w.Select("mode", "Mode", nil, ""); err != nil {
		t.Fatal(err)
	}
	if got, want := fp.titles["mode"], "Mode · "+shortPath("/work/app"); got != want {
		t.Errorf("select title %q want %q", got, want)
	}
	if _, err := w.MultiSelectOptional("hooks", "Hooks", nil); err != nil {
		t.Fatal(err)
	}
	if got, want := fp.titles["hooks"], "Hooks · "+shortPath("/work/app"); got != want {
		t.Errorf("optional title %q want %q", got, want)
	}
	if _, err := w.Confirm("confirm", "Proceed?", "summary", true); err != nil {
		t.Fatal(err)
	}
	if got := fp.titles["confirm"]; got != "Proceed?" {
		t.Errorf("confirm title %q", got)
	}
	if _, err := w.Input("project", "Project directory", "", nil); err != nil {
		t.Fatal(err)
	}
	if got := fp.titles["project"]; got != "Project directory" {
		t.Errorf("project title %q", got)
	}
}

func TestShortPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cases := []struct{ in, want string }{
		{home, "~"},
		{filepath.Join(home, "work", "app"), filepath.Join("~", "work", "app")},
		{home + "x", home + "x"},
		{"/opt/app", "/opt/app"},
	}
	for _, c := range cases {
		if got := shortPath(c.in); got != c.want {
			t.Errorf("shortPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDialogAddShowsProjectInTitles(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	fp := newFake(t, fullAnswers(p))
	if _, err := runD(t, enableDialog(t, fp), "add", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	suffix := " · " + shortPath(p)
	for _, id := range []string{"domains", "customize", "platforms", "mode", "hooks"} {
		if !strings.HasSuffix(fp.titles[id], suffix) {
			t.Errorf("%s title %q lacks suffix %q", id, fp.titles[id], suffix)
		}
	}
	for _, id := range []string{"project", "confirm"} {
		if strings.HasSuffix(fp.titles[id], suffix) {
			t.Errorf("%s title %q must not carry the project", id, fp.titles[id])
		}
	}
}

func TestDialogRemoveShowsProjectInTitles(t *testing.T) {
	h, p := realHarness(t), t.TempDir()
	if _, err := run(t, "add", "git", "-y", "-p", p, "--copy", "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	fp := newFake(t, map[string]any{"remove": []string{"git"}, "confirm": true})
	if _, err := runD(t, enableDialog(t, fp), "remove", "-p", p); err != nil {
		t.Fatal(err)
	}
	if got, want := fp.titles["remove"], "Remove · "+shortPath(p); got != want {
		t.Errorf("remove title %q want %q", got, want)
	}
}
