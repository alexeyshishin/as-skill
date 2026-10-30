package prompt

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
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
	// untick b then a (cursor ends on a), Enter on the empty list (blocked), tick b, Enter
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
