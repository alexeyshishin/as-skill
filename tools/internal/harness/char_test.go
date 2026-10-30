package harness

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var updateChar = flag.Bool("update-char", false, "rewrite characterization golden")

var manifestSynthetic = map[string]string{
	"empty":           "",
	"name-only":       "name: x\n",
	"missing-name":    "description: d\n",
	"crlf":            "name: x\r\ndescription: d\r\nrequires_env:\r\n  - A\r\n  - B\r\ntargets:\r\n  rules: ${CLAUDE_HOME}/rules\r\n",
	"inline-list":     "name: x\nrequires_env: [A, B]\nrequires_bin: []\n",
	"block-list":      "name: x\nrequires_env:\n  - A\n  - B\nrequires_bin:\n  - git\n",
	"quoted":          "name: \"x\"\ndescription: 'it''s'\ntargets:\n  rules: \"${CLAUDE_HOME}/rules\"\n",
	"trailing-cmt":    "name: x # c\ndescription: d # c\n",
	"colon-desc":      "name: x\ndescription: a: b\n",
	"dup-key":         "name: x\nname: y\n",
	"tab-targets":     "name: x\ntargets:\n\trules: r\n",
	"targets-blank":   "name: x\ntargets:\n  rules: r\n\n  skills: s\n",
	"targets-nocolon": "name: x\ntargets:\n  junk\n  rules: r\n",
	"unknown-key":     "name: x\nfoo: bar\nbaz:\n  - 1\n",
	"block-scalar":    "name: x\ndescription: >\n  a\n  b\n",
	"indented-top":    "  name: x\n",
	"list-no-dash":    "name: x\nrequires_env:\n  A\n",
	"empty-name":      "name:\n",
	"comments-only":   "# c\n# d\n",
	"bad-line":        "name: x\n!!!\n",
	"nested-target":   "name: x\ntargets:\n  rules:\n    deep: v\n",
}

func charManifestResult(data []byte, path string) string {
	f := filepath.Join(os.TempDir(), fmt.Sprintf("char-%d-manifest.yaml", os.Getpid()))
	defer os.Remove(f)
	if err := os.WriteFile(f, data, 0o644); err != nil {
		return "WRITE ERR " + err.Error()
	}
	m, err := ParseManifest(f)
	if err != nil {
		return "ERR " + strings.ReplaceAll(err.Error(), f, "<path>")
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func TestCharacterizeParseManifest(t *testing.T) {
	got := map[string]string{}
	real, _ := filepath.Glob("../../../domains/*/manifest.yaml")
	if len(real) == 0 {
		t.Fatal("no real manifests found")
	}
	for _, p := range real {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		got["real:"+filepath.ToSlash(strings.TrimPrefix(p, "../../../"))] = charManifestResult(data, p)
	}
	for k, v := range manifestSynthetic {
		got["syn:"+k] = charManifestResult([]byte(v), k)
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
	golden := filepath.Join("testdata", "manifest_char.golden")
	if *updateChar {
		os.MkdirAll("testdata", 0o755)
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
