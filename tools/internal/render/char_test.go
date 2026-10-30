package render

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var updateChar = flag.Bool("update-char", false, "rewrite characterization golden")

var sourceSynthetic = map[string]string{
	"empty":           "",
	"no-fm":           "just body\n",
	"unterminated":    "---\nname: a\nbody\n",
	"empty-fm":        "---\n---\nbody\n",
	"crlf":            "---\r\nname: a\r\ndescription: d\r\nmodel: m\r\neffort: high\r\n---\r\nbody\r\n",
	"quoted":          "---\nname: \"a b\"\ndescription: 'it''s'\nmodel: \"x\\ty\"\n---\nb\n",
	"inline-list":     "---\nname: a\ntools: [x, y]\ndescription: d\n---\nb\n",
	"folded":          "---\nname: a\ndescription: >\n  one\n  two\n---\nb\n",
	"literal":         "---\nname: a\ndescription: |\n  one\n  two\n---\nb\n",
	"colon-desc":      "---\nname: a\ndescription: x: y\n---\nb\n",
	"trailing-cmt":    "---\nname: a # c\ndescription: d\n---\nb\n",
	"dup-key":         "---\nname: a\nname: b\n---\nb\n",
	"closing-eof":     "---\nname: a\n---",
	"closing-nonl":    "---\nname: a\n---\nbody-no-nl",
	"body-hr":         "---\nname: a\n---\nx\n---\ny\n",
	"indented":        "---\n  name: a\ndescription: d\n---\nb\n",
	"no-colon":        "---\nname: a\njunk\ndescription: d\n---\nb\n",
	"comment":         "---\n# c\nname: a\n---\nb\n",
	"empty-value":     "---\nname:\ndescription: d\n---\nb\n",
	"list-block":      "---\nname: a\ntools:\n  - x\n  - y\ndescription: d\n---\nb\n",
	"trailing-ws-fm":  "---  \nname: a\n---  \nb\n",
	"bad-escape":      "---\nname: \"a\\qb\"\n---\nb\n",
	"unterm-quote":    "---\nname: \"a\n---\nb\n",
	"key-space":       "---\nname : a\n---\nb\n",
	"value-colon-url": "---\nname: a\ndescription: see http://x.y\n---\nb\n",
}

func TestParseSourceRealFiles(t *testing.T) {
	var files []string
	for _, pat := range []string{"../../../domains/*/skills/*/SKILL.md", "../../../domains/*/agents/*.md"} {
		m, _ := filepath.Glob(pat)
		files = append(files, m...)
	}
	if len(files) == 0 {
		t.Fatal("no real sources found")
	}
	for _, p := range files {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		s, err := parseSource(data)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		stem := strings.TrimSuffix(filepath.Base(p), ".md")
		if filepath.Base(p) == "SKILL.md" {
			stem = filepath.Base(filepath.Dir(p))
		}
		if s.name != stem {
			t.Errorf("%s: name %q != %q", p, s.name, stem)
		}
		if strings.TrimSpace(s.description) == "" {
			t.Errorf("%s: empty description", p)
		}
		if strings.TrimSpace(s.body) == "" {
			t.Errorf("%s: empty body", p)
		}
	}
}

func TestCharacterizeParseSource(t *testing.T) {
	got := map[string]string{}
	res := func(data []byte) string {
		s, err := parseSource(data)
		if err != nil {
			return "ERR " + err.Error()
		}
		b, _ := json.Marshal(map[string]string{"name": s.name, "description": s.description, "model": s.model, "effort": s.effort, "body": s.body})
		return string(b)
	}
	for k, v := range sourceSynthetic {
		got["syn:"+k] = res([]byte(v))
	}
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k + "\t" + got[k] + "\n")
	}
	golden := filepath.Join("testdata", "source_char.golden")
	if *updateChar {
		if err := os.WriteFile(golden, []byte(sb.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != sb.String() {
		wl, gl := strings.Split(string(want), "\n"), strings.Split(sb.String(), "\n")
		for i := 0; i < len(wl) || i < len(gl); i++ {
			var w, g string
			if i < len(wl) {
				w = wl[i]
			}
			if i < len(gl) {
				g = gl[i]
			}
			if w != g {
				t.Errorf("delta:\n old: %s\n new: %s", w, g)
			}
		}
	}
}
