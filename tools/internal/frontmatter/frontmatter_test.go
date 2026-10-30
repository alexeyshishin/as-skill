package frontmatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseShapes(t *testing.T) {
	src := "---\nname: x\ndescription: >\n  folded line one\n  line two\n\n  para\nliteral: |\n  a\n  b\nstrip: >-\n  s\nq: \"a \\\"b\\\" c\"\nsq: 'it''s'\nlist:\n  - a\n  - \"b c\"\nflat:\n- x\n- y\nflow: [a, 'b']\nmeta:\n  k1: v1\n  k2: \"v 2\"\nempty:\nplain: hello # c\n---\nbody\n"
	d, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string]string{
		"name":        "x",
		"description": "folded line one line two\npara\n",
		"literal":     "a\nb\n",
		"strip":       "s",
		"q":           `a "b" c`,
		"sq":          "it's",
		"empty":       "",
		"plain":       "hello",
	}
	for k, want := range checks {
		if got, _ := d.String(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if l := d.Fields["list"].List; len(l) != 2 || l[1] != "b c" {
		t.Errorf("list = %v", l)
	}
	if l := d.Fields["flat"].List; len(l) != 2 || l[1] != "y" {
		t.Errorf("flat = %v", l)
	}
	if l := d.Fields["flow"].List; len(l) != 2 || l[1] != "b" {
		t.Errorf("flow = %v", l)
	}
	if m := d.Fields["meta"].Map; m["k1"] != "v1" || m["k2"] != "v 2" {
		t.Errorf("meta = %v", m)
	}
	if d.Body != "body\n" {
		t.Errorf("body = %q", d.Body)
	}
	if d.Fields["q"].Line != 13 {
		t.Errorf("q line = %d", d.Fields["q"].Line)
	}
}

func TestParseMalformed(t *testing.T) {
	cases := map[string]struct{ src, want string }{
		"no open":      {"name: x\n", "line 1"},
		"no close":     {"---\nname: x\n", "not terminated"},
		"dup":          {"---\nname: a\nname: b\n---\n", "line 3: duplicate"},
		"unterminated": {"---\nname: x\nd: \"abc\n---\n", "line 3: unterminated"},
		"trailing":     {"---\nd: \"a\" b\n---\n", "line 2: unexpected text"},
		"colon":        {"---\nd: a: b\n---\n", "line 2: plain scalar"},
		"indent":       {"---\nname: x\n  stray: 1\n---\n", "line 3"},
		"tab":          {"---\nl:\n\t- a\n---\n", "line 3"},
		"badkey":       {"---\njust text\n---\n", "line 2"},
		"deep":         {"---\nm:\n  a:\n    b: 1\n---\n", "line 3"},
		"flowmap":      {"---\nm: {a: 1}\n---\n", "line 2: flow maps"},
		"emptyitem":    {"---\nl:\n  -\n---\n", "line 3: empty list item"},
		"cont":         {"---\nd: abc\n  more\n---\n", "line 3"},
		"badescape":    {"---\nd: \"\\q\"\n---\n", "line 2: unsupported escape"},
	}
	for name, c := range cases {
		_, err := Parse([]byte(c.src))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want containing %q", name, err, c.want)
		}
	}
}

func TestParseRealFiles(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, pat := range []string{"domains/*/skills/*/SKILL.md", "domains/*/agents/*.md"} {
		m, _ := filepath.Glob(filepath.Join(root, pat))
		files = append(files, m...)
	}
	if len(files) < 20 {
		t.Fatalf("found only %d files", len(files))
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		d, err := Parse(data)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		if s, _ := d.String("name"); s == "" {
			t.Errorf("%s: no name", f)
		}
		if s, _ := d.String("description"); strings.TrimSpace(s) == "" {
			t.Errorf("%s: no description", f)
		}
	}
}
