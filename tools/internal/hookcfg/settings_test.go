package hookcfg

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const registeredSettings = `{
  "permissions": {"allow": ["Bash(ls)"]},
  "hooks": {
    "SessionStart": [
      {"matcher": "startup|clear", "hooks": [{"type": "command", "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/load.sh"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "/usr/local/bin/foreign.sh"}]},
      {"hooks": [{"type": "command", "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/gate.sh"}]}
    ],
    "PostToolUse": [
      {"matcher": "Edit|Bash", "hooks": [
        {"type": "command", "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/fmt.sh"},
        {"type": "command", "command": "echo mine"}
      ]}
    ]
  },
  "model": "opus"
}`

func keysOf(t *testing.T, raw []byte) []string {
	t.Helper()
	obj, err := parseObject(raw)
	if err != nil {
		t.Fatalf("parseObject: %v", err)
	}
	keys := make([]string, len(obj))
	for i, m := range obj {
		keys[i] = m.key
	}
	return keys
}

func TestRemoveHookRegistrationsDropsOnlyOwnedHandlers(t *testing.T) {
	names := map[string]bool{"load.sh": true, "gate.sh": true, "fmt.sh": true}
	out, removed, err := removeHookRegistrations([]byte(registeredSettings), "/proj", names)
	if err != nil {
		t.Fatalf("removeHookRegistrations: %v", err)
	}

	wantRemoved := map[string]bool{"SessionStart/load.sh": true, "Stop/gate.sh": true, "PostToolUse/fmt.sh": true}
	if len(removed) != len(wantRemoved) {
		t.Fatalf("expected %d removals, got %+v", len(wantRemoved), removed)
	}
	for _, r := range removed {
		if !wantRemoved[r.Event+"/"+r.Hook] {
			t.Fatalf("unexpected removal %+v", r)
		}
	}

	if got, want := keysOf(t, out), []string{"permissions", "hooks", "model"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("top-level key order changed: got %v want %v", got, want)
	}
	var doc struct {
		Hooks map[string][]struct {
			Matcher string
			Hooks   []struct{ Command string }
		}
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if _, ok := doc.Hooks["SessionStart"]; ok {
		t.Fatal("SessionStart had only our hook, expected the event key to be dropped")
	}
	stop := doc.Hooks["Stop"]
	if len(stop) != 1 || stop[0].Hooks[0].Command != "/usr/local/bin/foreign.sh" {
		t.Fatalf("expected only the foreign Stop entry to survive, got %+v", stop)
	}
	post := doc.Hooks["PostToolUse"]
	if len(post) != 1 || post[0].Matcher != "Edit|Bash" || len(post[0].Hooks) != 1 || post[0].Hooks[0].Command != "echo mine" {
		t.Fatalf("expected the mixed entry to keep its matcher and foreign handler only, got %+v", post)
	}
	if !strings.Contains(string(out), `"allow": [`) || !strings.HasSuffix(string(out), "}\n") {
		t.Fatalf("expected re-indented output with trailing newline, got:\n%s", out)
	}
}

func TestRemoveHookRegistrationsDropsEmptyHooksKey(t *testing.T) {
	in := `{"model":"opus","hooks":{"Stop":[{"hooks":[{"type":"command","command":"\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/gate.sh"}]}]}}`
	out, removed, err := removeHookRegistrations([]byte(in), "/proj", map[string]bool{"gate.sh": true})
	if err != nil || len(removed) != 1 {
		t.Fatalf("err=%v removed=%+v", err, removed)
	}
	if got, want := keysOf(t, out), []string{"model"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected the emptied hooks key to be dropped, got keys %v", got)
	}
}

func TestRemoveHookRegistrationsCommandForms(t *testing.T) {
	names := map[string]bool{"x.sh": true}
	cases := []struct {
		cmd  string
		owns bool
	}{
		{`\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/x.sh`, true},
		{`$CLAUDE_PROJECT_DIR/.claude/hooks/x.sh`, true},
		{`\"${CLAUDE_PROJECT_DIR}/.claude/hooks/x.sh\"`, true},
		{`/proj/.claude/hooks/x.sh`, true},
		{`.claude/hooks/x.sh`, true},
		{`./.claude/hooks/x.sh`, true},
		{`/other/.claude/hooks/x.sh`, false},
		{`$CLAUDE_PROJECT_DIR/.claude/hooks/x.sh.bak`, false},
		{`$CLAUDE_PROJECT_DIR/.claude/hooks/sub/x.sh`, false},
		{`$CLAUDE_PROJECT_DIR/.claude/hooks/y.sh`, false},
		{`$CLAUDE_PROJECT_DIR/.claude/hooks/x.sh --flag`, false},
	}
	for _, c := range cases {
		in := `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"` + c.cmd + `"}]}]}}`
		_, removed, err := removeHookRegistrations([]byte(in), "/proj", names)
		if err != nil {
			t.Fatalf("%s: %v", c.cmd, err)
		}
		if (len(removed) == 1) != c.owns {
			t.Errorf("command %s: owned=%v, want %v", c.cmd, len(removed) == 1, c.owns)
		}
	}
}

func TestRemoveHookRegistrationsLeavesUnexpectedShapesAlone(t *testing.T) {
	names := map[string]bool{"x.sh": true}
	for _, in := range []string{
		`{"model":"opus"}`,
		`{"hooks":"nope"}`,
		`{"hooks":[1,2]}`,
		`{"hooks":{"Stop":"nope"}}`,
		`{"hooks":{"Stop":[1,"a",{"hooks":"nope"},{"matcher":"x"}]}}`,
		`{"hooks":{"Stop":null}}`,
	} {
		out, removed, err := removeHookRegistrations([]byte(in), "/proj", names)
		if err != nil || len(removed) != 0 || string(out) != in {
			t.Errorf("%s: expected untouched input, got out=%q removed=%+v err=%v", in, out, removed, err)
		}
	}
	for _, in := range []string{`{oops`, `[]`, `"str"`, ``} {
		if _, _, err := removeHookRegistrations([]byte(in), "/proj", names); err == nil {
			t.Errorf("%q: expected an error", in)
		}
	}
}

func TestRemoveHookRegistrationsKeepsKeyBytes(t *testing.T) {
	in := `{"env":{"A&B":"<x>"},"hooks":{"Stop":[{"hooks":[{"type":"command","command":".claude/hooks/x.sh"}]}]},"n":1.50}`
	out, _, err := removeHookRegistrations([]byte(in), "/proj", map[string]bool{"x.sh": true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"A&B": "<x>"`, `"n": 1.50`} {
		if !strings.Contains(string(out), want) {
			t.Errorf("expected %q to survive verbatim, got:\n%s", want, out)
		}
	}
}
