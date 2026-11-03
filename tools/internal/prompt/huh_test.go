package prompt

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func TestDedupe(t *testing.T) {
	got := dedupe([]string{"b", "a", "b", "c", "a"})
	if !reflect.DeepEqual(got, []string{"b", "a", "c"}) {
		t.Fatalf("got %v", got)
	}
}

func TestHuhConfirmHeadless(t *testing.T) {
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\r"), out: &out}
	got, err := p.Confirm("go", "Proceed?", "summary text", true)
	if err != nil {
		t.Fatalf("err %v\n%s", err, out.String())
	}
	if !got {
		t.Fatal("default yes + enter should confirm")
	}
}

func TestHuhMultiSelectHeadless(t *testing.T) {
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\r"), out: &out}
	opts := []Option{{"a", "A", true}, {"b", "B", false}, {"c", "C", true}}
	got, err := p.MultiSelect("d", "Pick", opts, nil)
	if err != nil {
		t.Fatalf("err %v\n%s", err, out.String())
	}
	if !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Fatalf("preselection not honored: %v", got)
	}
}

func TestHuhAbortMapped(t *testing.T) {
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\x03"), out: &out}
	if _, err := p.Confirm("go", "Proceed?", "", true); err != ErrAborted {
		t.Fatalf("got %v want ErrAborted", err)
	}
}

func TestHuhMultiSelectEnterTakesHovered(t *testing.T) {
	nonEmpty := func(v []string) error {
		if len(v) == 0 {
			return errors.New("empty")
		}
		return nil
	}
	opts := []Option{{"a", "A", false}, {"b", "B", false}, {"c", "C", false}}
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"\r", []string{"a"}},
		{"\x1b[B\r", []string{"b"}},
		{" \x1b[B \r", []string{"a", "b"}},
	} {
		var out bytes.Buffer
		p := &huhPrompter{in: strings.NewReader(tc.in), out: &out}
		got, err := p.MultiSelect("d", "Pick", opts, nonEmpty)
		if err != nil {
			t.Fatalf("%q: err %v\n%s", tc.in, err, out.String())
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q: got %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestHuhShiftTabYieldsErrBack(t *testing.T) {
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\x1b[Z"), out: &out}
	if _, err := p.Confirm("go", "Proceed?", "", true); err != ErrBack {
		t.Fatalf("got %v want ErrBack\n%s", err, out.String())
	}
}

func TestHuhMultiSelectPreselectedUntickAllReachesValidator(t *testing.T) {
	var emptyCalls int
	nonEmpty := func(v []string) error {
		if len(v) == 0 {
			emptyCalls++
			return errors.New("select at least one")
		}
		return nil
	}
	opts := []Option{{"a", "A", true}, {"b", "B", true}}
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\x1b[B \x1b[A \r\x1b[B \r"), out: &out}
	got, err := p.MultiSelect("d", "Pick", opts, nonEmpty)
	if err != nil {
		t.Fatalf("err %v\n%s", err, out.String())
	}
	if !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatalf("hovered fallback swallowed the empty set: got %v", got)
	}
	if emptyCalls == 0 {
		t.Fatal("validator never saw the empty set")
	}
}

func TestHuhMultiSelectPreselectedUntickAllReturnsEmpty(t *testing.T) {
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\x1b[B \x1b[A \r"), out: &out}
	got, err := p.MultiSelect("d", "Pick", []Option{{"a", "A", true}, {"b", "B", true}}, nil)
	if err != nil {
		t.Fatalf("err %v\n%s", err, out.String())
	}
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestHuhMultiSelectPreselectedUntickOneKeepsRest(t *testing.T) {
	nonEmpty := func(v []string) error {
		if len(v) == 0 {
			return errors.New("empty")
		}
		return nil
	}
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader(" \r"), out: &out}
	got, err := p.MultiSelect("d", "Pick", []Option{{"a", "A", true}, {"b", "B", true}, {"c", "C", true}}, nonEmpty)
	if err != nil {
		t.Fatalf("err %v\n%s", err, out.String())
	}
	if !reflect.DeepEqual(got, []string{"b", "c"}) {
		t.Fatalf("got %v", got)
	}
}

func TestHuhMultiSelectOptional(t *testing.T) {
	opts := []Option{{"a", "A", false}, {"b", "B", false}}
	for _, tc := range []struct {
		name, in string
		want     []string
		back     bool
	}{
		{"enter with nothing toggled is none", "\r", nil, false},
		{"toggle one", "\x1b[B \r", []string{"b"}, false},
		{"shift+tab is back", "\x1b[Z", nil, true},
	} {
		var out bytes.Buffer
		p := &huhPrompter{in: strings.NewReader(tc.in), out: &out}
		got, err := p.MultiSelectOptional("d", "Pick", opts)
		if tc.back {
			if err != ErrBack {
				t.Fatalf("%s: got %v want ErrBack", tc.name, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: err %v\n%s", tc.name, err, out.String())
		}
		if len(got) != len(tc.want) || (len(got) > 0 && !reflect.DeepEqual(got, tc.want)) {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func newProjectModel(def string) (*formModel, *string) {
	v := def
	in := huh.NewInput().Title("Project directory").Value(&v)
	m := &formModel{form: newForm(in), project: &v, sel: -1}
	m.Init()
	return m, &v
}

func press(m *formModel, msgs ...tea.Msg) {
	for _, msg := range msgs {
		m.Update(msg)
	}
}

func typeRunes(s string) tea.Msg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

var (
	tabKey   = tea.KeyMsg{Type: tea.KeyTab}
	downKey  = tea.KeyMsg{Type: tea.KeyDown}
	upKey    = tea.KeyMsg{Type: tea.KeyUp}
	enterKey = tea.KeyMsg{Type: tea.KeyEnter}
)

func completionTree(t *testing.T, names ...string) string {
	t.Helper()
	tmp := t.TempDir()
	for _, n := range names {
		if err := os.MkdirAll(filepath.Join(tmp, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return tmp
}

func manyDirs(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("d%02d", i)
	}
	return names
}

func TestFormModelNoListBeforeTab(t *testing.T) {
	tmp := completionTree(t, "alpha1", "alpha2")
	m, _ := newProjectModel(tmp + "/al")
	if strings.Contains(m.View(), "  alpha1/") || m.open {
		t.Fatalf("list before tab:\n%s", m.View())
	}
	press(m, typeRunes("p"), typeRunes("h"))
	if strings.Contains(m.View(), "  alpha1/") || m.open {
		t.Fatalf("list while typing:\n%s", m.View())
	}
}

func TestFormModelTabSingleSubstringReplaces(t *testing.T) {
	tmp := completionTree(t, "github", "gitlab", "digit", "docs")
	m, v := newProjectModel(tmp + "/hub")
	press(m, tea.WindowSizeMsg{Width: 400, Height: 50}, tabKey)
	if *v != tmp+"/github/" {
		t.Fatalf("value %q", *v)
	}
	if m.open || strings.Contains(m.View(), "  gitlab/") {
		t.Fatalf("list shown:\n%s", m.View())
	}
	if m.form.State != huh.StateNormal || m.back {
		t.Fatal("tab must not submit or abort")
	}
	if !strings.Contains(m.form.View(), tmp+"/github/") {
		t.Fatalf("textinput not synced:\n%s", m.form.View())
	}
}

func TestFormModelTabCursorAtEndAfterReplacement(t *testing.T) {
	tmp := completionTree(t, "github")
	m, v := newProjectModel(tmp + "/ithu")
	press(m, tabKey, typeRunes("x"))
	if *v != tmp+"/github/x" {
		t.Fatalf("got %q", *v)
	}
}

func TestFormModelTabOpensListUnchanged(t *testing.T) {
	tmp := completionTree(t, "github", "gitlab", "digit", "docs")
	m, v := newProjectModel(tmp + "/git")
	press(m, tabKey)
	if *v != tmp+"/git" {
		t.Fatalf("value changed: %q", *v)
	}
	if !strings.Contains(m.View(), "  github/\n  gitlab/\n  digit/") {
		t.Fatalf("list order:\n%s", m.View())
	}
	if strings.Contains(m.View(), "docs/") || strings.Contains(m.View(), "\n> ") {
		t.Fatalf("unexpected row or marker:\n%s", m.View())
	}
}

func TestFormModelTabCyclesMarker(t *testing.T) {
	tmp := completionTree(t, "Users", "usr", "Volumes", "docs")
	m, v := newProjectModel(tmp + "/u")
	press(m, tabKey)
	if !m.open || m.sel != -1 {
		t.Fatalf("first tab: open=%v sel=%d", m.open, m.sel)
	}
	press(m, tabKey)
	if *v != tmp+"/u" || !m.open || m.sel != 0 {
		t.Fatalf("second tab: value %q open=%v sel=%d", *v, m.open, m.sel)
	}
	if !strings.Contains(m.View(), "> Users/\n  usr/\n  Volumes/") {
		t.Fatalf("marker not on first row:\n%s", m.View())
	}
	press(m, downKey)
	if m.sel != 1 {
		t.Fatalf("down after second tab: sel=%d", m.sel)
	}
	press(m, tabKey)
	if *v != tmp+"/u" || !m.open || m.sel != 2 {
		t.Fatalf("third tab: value %q open=%v sel=%d", *v, m.open, m.sel)
	}
	press(m, tabKey)
	if *v != tmp+"/u" || !m.open || m.sel != 0 {
		t.Fatalf("tab did not wrap: value %q open=%v sel=%d", *v, m.open, m.sel)
	}
	if m.form.State != huh.StateNormal || m.back {
		t.Fatal("tab must not submit")
	}
}

func TestFormModelEnterAcceptsHighlighted(t *testing.T) {
	tmp := completionTree(t, "Users", "usr", "Volumes")
	m, v := newProjectModel(tmp + "/u")
	press(m, tabKey, tabKey, enterKey)
	if *v != tmp+"/Users/" || m.open || m.sel != -1 || m.form.State != huh.StateNormal {
		t.Fatalf("value %q open=%v sel=%d", *v, m.open, m.sel)
	}
	if strings.Contains(m.View(), "usr/") {
		t.Fatalf("list not hidden:\n%s", m.View())
	}
}

func TestFormModelSecondTabOnLongListKeepsFirstRowVisible(t *testing.T) {
	m, _ := newProjectModel(completionTree(t, manyDirs(15)...) + "/d")
	press(m, tea.WindowSizeMsg{Width: 400, Height: 18}, tabKey, tabKey)
	if m.sel != 0 || m.off != 0 || !strings.Contains(m.View(), "> d00/") {
		t.Fatalf("sel=%d off=%d\n%s", m.sel, m.off, m.View())
	}
}

func TestFormModelTabExtendsAllPrefixAndOpens(t *testing.T) {
	tmp := completionTree(t, "alpha1", "alpha2", "beta")
	m, v := newProjectModel(tmp + "/al")
	press(m, tabKey)
	if *v != tmp+"/alpha" || !m.open {
		t.Fatalf("value %q open=%v", *v, m.open)
	}
	press(m, typeRunes("1"))
	if *v != tmp+"/alpha1" || m.open {
		t.Fatalf("value %q open=%v", *v, m.open)
	}
}

func TestFormModelNoMatchAndEmptyTabAreNoOps(t *testing.T) {
	tmp := completionTree(t, "alpha1")
	for _, in := range []string{tmp + "/zzz", ""} {
		m, v := newProjectModel(in)
		press(m, tabKey)
		if *v != in || m.open || m.form.State != huh.StateNormal || m.back {
			t.Errorf("%q: value %q open=%v", in, *v, m.open)
		}
	}
}

func TestFormModelDownDownEnterSubstitutesAndHides(t *testing.T) {
	tmp := completionTree(t, "a1", "a2", "a3")
	m, v := newProjectModel(tmp + "/a")
	press(m, tea.WindowSizeMsg{Width: 400, Height: 50}, tabKey, downKey)
	if !strings.Contains(m.View(), "> a1/\n  a2/\n  a3/") {
		t.Fatalf("marker:\n%s", m.View())
	}
	press(m, downKey, enterKey)
	if *v != tmp+"/a2/" {
		t.Fatalf("value %q", *v)
	}
	if m.open || m.sel != -1 || strings.Contains(m.View(), "  a3/") {
		t.Fatalf("list not hidden:\n%s", m.View())
	}
	if m.form.State != huh.StateNormal || m.back {
		t.Fatal("tab must not submit")
	}
	if !strings.Contains(m.form.View(), tmp+"/a2/") {
		t.Fatalf("textinput not synced:\n%s", m.form.View())
	}
}

func TestFormModelSelectionWraps(t *testing.T) {
	tmp := completionTree(t, "a1", "a2", "a3")
	m, _ := newProjectModel(tmp + "/a")
	press(m, tabKey, upKey)
	if m.sel != 2 || !strings.Contains(m.View(), "> a3/") {
		t.Fatalf("up from none: sel=%d\n%s", m.sel, m.View())
	}
	press(m, downKey)
	if m.sel != 0 {
		t.Fatalf("down wrap: sel=%d", m.sel)
	}
	press(m, upKey)
	if m.sel != 2 {
		t.Fatalf("up wrap: sel=%d", m.sel)
	}
}

func TestFormModelEditHidesAndResets(t *testing.T) {
	tmp := completionTree(t, "a1", "a2", "a3")
	for name, key := range map[string]tea.Msg{
		"type":      typeRunes("x"),
		"backspace": tea.KeyMsg{Type: tea.KeyBackspace},
		"ctrl+u":    tea.KeyMsg{Type: tea.KeyCtrlU},
	} {
		m, _ := newProjectModel(tmp + "/a")
		press(m, tabKey, downKey, key)
		if m.open || m.sel != -1 || m.off != 0 || strings.Contains(m.View(), "  a3/") {
			t.Errorf("%s: list not hidden: open=%v sel=%d", name, m.open, m.sel)
		}
	}
}

func TestFormModelCursorKeysKeepList(t *testing.T) {
	tmp := completionTree(t, "a1", "a2")
	m, _ := newProjectModel(tmp + "/a")
	press(m, tabKey, downKey)
	for _, k := range []tea.KeyType{tea.KeyLeft, tea.KeyRight, tea.KeyHome, tea.KeyEnd} {
		press(m, tea.KeyMsg{Type: k})
		if !m.open || m.sel != 0 {
			t.Fatalf("key %v hid the list", k)
		}
	}
	press(m, tea.WindowSizeMsg{Width: 80, Height: 40})
	if !m.open {
		t.Fatal("window size hid the list")
	}
}

func TestFormModelSecondTabOpensChildList(t *testing.T) {
	tmp := completionTree(t, "docs/one", "docs/two", "docz")
	m, v := newProjectModel(tmp + "/doc")
	press(m, tabKey, downKey, enterKey)
	if *v != tmp+"/docs/" {
		t.Fatalf("got %q", *v)
	}
	press(m, tabKey)
	if !m.open || !strings.Contains(m.View(), "  one/\n  two/") {
		t.Fatalf("child list:\n%s", m.View())
	}
}

func TestFormModelArrowsInertWhenHidden(t *testing.T) {
	tmp := completionTree(t, "a1", "a2")
	m, v := newProjectModel(tmp + "/a")
	press(m, upKey, downKey, upKey)
	if *v != tmp+"/a" || m.open || m.sel != -1 || m.form.State != huh.StateNormal || m.back {
		t.Fatalf("value %q open=%v sel=%d", *v, m.open, m.sel)
	}
	press(m, typeRunes("x"))
	if *v != tmp+"/ax" {
		t.Fatalf("cursor moved: %q", *v)
	}
}

func TestFormModelArrowsNeverEditValueWhenOpen(t *testing.T) {
	tmp := completionTree(t, "a1", "a2")
	m, v := newProjectModel(tmp + "/a")
	press(m, tabKey, downKey, upKey, downKey)
	if *v != tmp+"/a" {
		t.Fatalf("value %q", *v)
	}
	press(m, typeRunes("x"))
	if *v != tmp+"/ax" {
		t.Fatalf("cursor moved: %q", *v)
	}
}

func TestFormModelNonProjectIgnoresCompletion(t *testing.T) {
	tmp := completionTree(t, "alpha1", "alpha2")
	v := tmp + "/al"
	m := &formModel{form: newForm(huh.NewInput().Title("Name").Value(&v)), sel: -1}
	m.Init()
	press(m, tabKey, downKey, upKey)
	if m.open || len(m.list) != 0 || strings.Contains(m.View(), "alpha1/") {
		t.Fatalf("list on non-project field:\n%s", m.View())
	}
	if v != tmp+"/al" {
		t.Fatalf("value changed: %q", v)
	}
}

func TestFormModelListScrollFollowsSelection(t *testing.T) {
	tmp := completionTree(t, manyDirs(15)...)
	m, _ := newProjectModel(tmp + "/d")
	press(m, tabKey)
	if !strings.Contains(m.View(), "  d00/") || !strings.Contains(m.View(), "… +6 more") || strings.Contains(m.View(), "d09/") {
		t.Fatalf("initial window:\n%s", m.View())
	}
	for i := 0; i < 10; i++ {
		press(m, downKey)
	}
	view := m.View()
	if m.sel != 9 || m.off != 1 || !strings.Contains(view, "> d09/") || strings.Contains(view, "  d00/") || !strings.Contains(view, "… +6 more") {
		t.Fatalf("scrolled window sel=%d off=%d:\n%s", m.sel, m.off, view)
	}
	press(m, upKey, upKey)
	if m.sel != 7 || m.off != 1 || !strings.Contains(m.View(), "  d01/") {
		t.Fatalf("sticky bottom row: sel=%d off=%d:\n%s", m.sel, m.off, m.View())
	}
	for m.sel != 0 {
		press(m, upKey)
	}
	press(m, upKey)
	if m.sel != 14 || m.off != 6 || !strings.Contains(m.View(), "> d14/") {
		t.Fatalf("wrap to last: sel=%d off=%d:\n%s", m.sel, m.off, m.View())
	}
	press(m, downKey)
	if m.sel != 0 || m.off != 0 {
		t.Fatalf("wrap to first: sel=%d off=%d", m.sel, m.off)
	}
	view = m.View()
	if n := strings.Count(view[strings.LastIndex(view, "\n\n")+2:], "\n") + 1; n > 10 {
		t.Fatalf("list has %d lines", n)
	}
}

func TestFormModelResizeKeepsSelectionVisibleAndCloses(t *testing.T) {
	tmp := completionTree(t, manyDirs(15)...)
	m, _ := newProjectModel(tmp + "/d")
	press(m, tabKey)
	for i := 0; i < 10; i++ {
		press(m, downKey)
	}
	press(m, tea.WindowSizeMsg{Width: 80, Height: 12})
	if !strings.Contains(m.View(), "> d09/") || !strings.Contains(m.View(), "… +12 more") {
		t.Fatalf("selection lost after shrink:\n%s", m.View())
	}
	press(m, tea.WindowSizeMsg{Width: 80, Height: 8})
	if m.open || strings.Contains(m.View(), "d00/") || strings.Contains(m.View(), "more") {
		t.Fatalf("list not closed:\n%s", m.View())
	}
}

func TestFormModelShortTerminalTabDoesValueWorkOnly(t *testing.T) {
	tmp := completionTree(t, "alpha1", "alpha2", "github")
	m, v := newProjectModel(tmp + "/al")
	press(m, tea.WindowSizeMsg{Width: 80, Height: 8}, tabKey)
	if *v != tmp+"/alpha" || m.open {
		t.Fatalf("value %q open=%v", *v, m.open)
	}
	press(m, downKey, upKey)
	if m.sel != -1 || *v != tmp+"/alpha" {
		t.Fatalf("arrows not inert: sel=%d value %q", m.sel, *v)
	}
	m, v = newProjectModel(tmp + "/hub")
	press(m, tea.WindowSizeMsg{Width: 80, Height: 8}, tabKey)
	if *v != tmp+"/github/" {
		t.Fatalf("single match not filled: %q", *v)
	}
}

func TestFormModelShiftTabAndCtrlC(t *testing.T) {
	m, _ := newProjectModel("/tmp/x")
	press(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if !m.back {
		t.Fatal("shift+tab must set back")
	}
	m, _ = newProjectModel("/tmp/x")
	press(m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.form.State != huh.StateAborted {
		t.Fatalf("state %v", m.form.State)
	}
}

func TestFormModelEnterWithoutHighlightSubmitsTyped(t *testing.T) {
	tmp := completionTree(t, "a1", "a2")
	m, v := newProjectModel(tmp + "/a")
	press(m, tabKey, enterKey)
	if *v != tmp+"/a" {
		t.Fatalf("value %q", *v)
	}
}

func TestHuhProjectInputEnterSubmits(t *testing.T) {
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\r"), out: &out}
	got, err := p.Input("project", "Project directory", "/tmp/x", nil)
	if err != nil || got != "/tmp/x" {
		t.Fatalf("got %q err %v\n%s", got, err, out.String())
	}
}

func TestHuhProjectInputTabDownEnterEnter(t *testing.T) {
	tmp := completionTree(t, "github", "gitlab", "digit")
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\t\x1b[B\r\r"), out: &out}
	got, err := p.Input("project", "Project directory", tmp+"/git", nil)
	if err != nil || got != tmp+"/github/" {
		t.Fatalf("got %q err %v\n%s", got, err, out.String())
	}
}

func TestHuhProjectInputTabTabEnterEnterPicksFirst(t *testing.T) {
	tmp := completionTree(t, "Users", "usr", "Volumes")
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\t\t\r\r"), out: &out}
	got, err := p.Input("project", "Project directory", tmp+"/u", nil)
	if err != nil || got != tmp+"/Users/" {
		t.Fatalf("got %q err %v\n%s", got, err, out.String())
	}
}

func TestHuhProjectInputEnterWithoutHighlightKeepsTyped(t *testing.T) {
	tmp := completionTree(t, "github", "gitlab", "digit")
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\t\r"), out: &out}
	got, err := p.Input("project", "Project directory", tmp+"/git", nil)
	if err != nil || got != tmp+"/git" {
		t.Fatalf("got %q err %v\n%s", got, err, out.String())
	}
}

func TestHuhProjectInputShiftTabYieldsErrBack(t *testing.T) {
	var out bytes.Buffer
	p := &huhPrompter{in: strings.NewReader("\x1b[Z"), out: &out}
	if _, err := p.Input("project", "Project directory", "/tmp/x", nil); err != ErrBack {
		t.Fatalf("got %v want ErrBack\n%s", err, out.String())
	}
}
