package render

import (
	"bytes"
	"strings"
	"testing"
)

func TestBlockRoundTrip(t *testing.T) {
	cases := map[string]string{
		"absent":  "",
		"user":    "# My rules\n\nkeep this\n",
		"crlf":    "a\r\nb\r\n",
		"no-nl-x": "",
	}
	for name, orig := range cases {
		once := UpsertBlock([]byte(orig), "rule one\nrule two")
		if !bytes.HasPrefix(once, []byte(orig)) {
			t.Errorf("%s: prefix altered", name)
		}
		if !strings.Contains(string(once), BlockBegin+"\nrule one\nrule two\n"+BlockEnd) {
			t.Errorf("%s: block missing: %q", name, once)
		}
		twice := UpsertBlock(once, "rule one\nrule two")
		if !bytes.Equal(once, twice) {
			t.Errorf("%s: not idempotent", name)
		}
		back, ok := RemoveBlock(once)
		if !ok || string(back) != orig {
			t.Errorf("%s: remove got %q ok=%v want %q", name, back, ok, orig)
		}
	}
}

func TestBlockReplaceKeepsOutside(t *testing.T) {
	orig := "top\n\n" + BlockBegin + "\nold\n" + BlockEnd + "\n\nbottom\n"
	got := UpsertBlock([]byte(orig), "new")
	want := "top\n\n" + BlockBegin + "\nnew\n" + BlockEnd + "\n\nbottom\n"
	if string(got) != want {
		t.Errorf("got %q", got)
	}
	rm, ok := RemoveBlock(got)
	if !ok || string(rm) != "top\n\n\n\nbottom\n" {
		t.Errorf("remove %q", rm)
	}
}

func TestRemoveAbsent(t *testing.T) {
	in := []byte("nothing here\n")
	out, ok := RemoveBlock(in)
	if ok || !bytes.Equal(out, in) {
		t.Error("absent")
	}
	in = []byte(BlockBegin + "\nunterminated\n")
	if out, ok := RemoveBlock(in); ok || !bytes.Equal(out, in) {
		t.Error("unterminated")
	}
}
