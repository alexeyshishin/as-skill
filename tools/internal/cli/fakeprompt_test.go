package cli

import (
	"os"
	"reflect"
	"testing"

	"claude-harness/tools/internal/prompt"
)

func TestMain(m *testing.M) {
	interactive = func() bool { return false }
	os.Exit(m.Run())
}

// fakePrompt answers by question id: string (Input/Select), []string
// (MultiSelect), bool (Confirm) or an error (returned as-is).
type fakePrompt struct {
	t        *testing.T
	answers  map[string]any
	asked    []string
	options  map[string][]prompt.Option
	validate map[string]func([]string) error
	summary  string
	history  map[string][][]string // per id: selected values at each ask
}

func newFake(t *testing.T, answers map[string]any) *fakePrompt {
	return &fakePrompt{t: t, answers: answers, options: map[string][]prompt.Option{}, validate: map[string]func([]string) error{}}
}

// enableDialog opts a test in to the interactive path with fp as the prompter.
func enableDialog(t *testing.T, fp *fakePrompt) {
	oldI, oldN := interactive, newPrompter
	interactive = func() bool { return true }
	newPrompter = func() prompt.Prompter { return fp }
	t.Cleanup(func() { interactive, newPrompter = oldI, oldN })
}

// seq scripts successive answers to one question id (e.g. prompt.ErrBack, then a value).
type seq []any

func (f *fakePrompt) answer(id string) any {
	f.t.Helper()
	f.asked = append(f.asked, id)
	a, ok := f.answers[id]
	if !ok {
		f.t.Fatalf("unexpected prompt id %q (asked so far: %v)", id, f.asked)
	}
	if q, ok := a.(seq); ok {
		if len(q) == 0 {
			f.t.Fatalf("scripted answers for %q exhausted", id)
		}
		a = q[0]
		f.answers[id] = q[1:]
	}
	return a
}

func (f *fakePrompt) Input(id, title, def string, validate func(string) error) (string, error) {
	a := f.answer(id)
	if err, ok := a.(error); ok {
		return "", err
	}
	return a.(string), nil
}

func (f *fakePrompt) MultiSelect(id, title string, opts []prompt.Option, validate func([]string) error) ([]string, error) {
	a := f.answer(id)
	f.options[id] = opts
	f.validate[id] = validate
	f.record(id, opts)
	if err, ok := a.(error); ok {
		return nil, err
	}
	return a.([]string), nil
}

func (f *fakePrompt) Select(id, title string, opts []prompt.Option, def string) (string, error) {
	a := f.answer(id)
	f.options[id] = opts
	f.record(id, opts)
	if err, ok := a.(error); ok {
		return "", err
	}
	return a.(string), nil
}

func (f *fakePrompt) Confirm(id, title, desc string, def bool) (bool, error) {
	a := f.answer(id)
	f.summary = desc
	if err, ok := a.(error); ok {
		return false, err
	}
	return a.(bool), nil
}

func (f *fakePrompt) wantAsked(want ...string) {
	f.t.Helper()
	if !reflect.DeepEqual(f.asked, want) {
		f.t.Fatalf("asked %v, want %v", f.asked, want)
	}
}

func (f *fakePrompt) labelOf(id, value string) string {
	for _, o := range f.options[id] {
		if o.Value == value {
			return o.Label
		}
	}
	return ""
}

func (f *fakePrompt) selected(id string) []string {
	var out []string
	for _, o := range f.options[id] {
		if o.Selected {
			out = append(out, o.Value)
		}
	}
	return out
}

func (f *fakePrompt) record(id string, opts []prompt.Option) {
	if f.history == nil {
		f.history = map[string][][]string{}
	}
	var sel []string
	for _, o := range opts {
		if o.Selected {
			sel = append(sel, o.Value)
		}
	}
	f.history[id] = append(f.history[id], sel)
}
