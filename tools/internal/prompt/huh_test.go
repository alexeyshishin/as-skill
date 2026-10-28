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
