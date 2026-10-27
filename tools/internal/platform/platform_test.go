package platform

import (
	"reflect"
	"strings"
	"testing"
)

func TestTableInvariants(t *testing.T) {
	if len(table) != 4 {
		t.Fatalf("want 4 platforms, got %d", len(table))
	}
	for _, id := range order {
		p, ok := Get(id)
		if !ok || p.ID != id {
			t.Fatalf("%s missing", id)
		}
		if len(p.SkillDirsRead) == 0 || p.Agents.Dir == "" || p.Agents.Ext == "" || p.Agents.Format == "" {
			t.Errorf("%s: incomplete spec", id)
		}
		if p.Rules.Mode == "" || p.Rules.Dir == "" || p.InstructionFile == "" || p.Hooks.Schema == "" {
			t.Errorf("%s: incomplete spec", id)
		}
		if p.Hooks.Schema != HooksNone && (p.Hooks.Dir == "" || p.Hooks.ConfigFile == "") {
			t.Errorf("%s: hooks incomplete", id)
		}
		for _, k := range []string{"skills", "agents", "rules", "hooks"} {
			if _, ok := p.Verified[k]; !ok {
				t.Errorf("%s: Verified[%s] absent", id, k)
			}
		}
	}
	if _, ok := Get("copilot"); ok {
		t.Error("copilot must not exist")
	}
	cu := table[Cursor].Hooks.EventMap
	if !reflect.DeepEqual(cu["PostToolUse"], []string{"afterFileEdit", "afterShellExecution"}) || cu["Stop"][0] != "stop" {
		t.Errorf("cursor eventmap: %v", cu)
	}
	if table[Codex].Hooks.EventMap["Stop"][0] != "Stop" {
		t.Error("codex eventmap")
	}
	if !reflect.DeepEqual(table[Codex].ReservedAgentNames, []string{"default", "worker", "explorer"}) {
		t.Error("codex reserved")
	}
	if !reflect.DeepEqual(table[OpenCode].ReservedAgentNames, []string{"build", "plan", "general", "explore"}) {
		t.Error("opencode reserved")
	}
	if table[OpenCode].Hooks.Schema != HooksNone {
		t.Error("opencode hooks")
	}
}

func TestParsePlatforms(t *testing.T) {
	cases := []struct {
		in   []string
		want []ID
		err  string
	}{
		{nil, []ID{Claude}, ""},
		{[]string{""}, []ID{Claude}, ""},
		{[]string{"codex"}, []ID{Codex}, ""},
		{[]string{"opencode,claude"}, []ID{Claude, OpenCode}, ""},
		{[]string{"cursor", "codex", "cursor"}, []ID{Codex, Cursor}, ""},
		{[]string{"all"}, []ID{Claude, Codex, Cursor, OpenCode}, ""},
		{[]string{" Claude , all"}, []ID{Claude, Codex, Cursor, OpenCode}, ""},
		{[]string{"claude,copilot"}, nil, "claude, codex, cursor, opencode"},
	}
	for _, c := range cases {
		got, err := ParsePlatforms(c.in)
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%v: err %v", c.in, err)
			}
			continue
		}
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%v: got %v %v want %v", c.in, got, err, c.want)
		}
	}
}

func TestSkillLocations(t *testing.T) {
	cases := []struct {
		name string
		ids  []ID
		want map[string][]ID
		dup  []ID
	}{
		{"claude", []ID{Claude}, map[string][]ID{".claude/skills": {Claude}}, nil},
		{"claude+cursor", []ID{Claude, Cursor}, map[string][]ID{".claude/skills": {Claude, Cursor}}, nil},
		{"all-but-codex", []ID{Claude, Cursor, OpenCode}, map[string][]ID{".claude/skills": {Claude, Cursor, OpenCode}}, nil},
		{"codex+cursor+opencode", []ID{Codex, Cursor, OpenCode}, map[string][]ID{".agents/skills": {Codex, Cursor, OpenCode}}, nil},
		{"cursor", []ID{Cursor}, map[string][]ID{".agents/skills": {Cursor}}, nil},
		{"claude+codex", []ID{Claude, Codex}, map[string][]ID{".claude/skills": {Claude}, ".agents/skills": {Codex}}, nil},
		{"all", []ID{Claude, Codex, Cursor, OpenCode}, map[string][]ID{".claude/skills": {Claude, Cursor, OpenCode}, ".agents/skills": {Codex}}, []ID{Cursor, OpenCode}},
		{"claude+codex+opencode", []ID{Claude, Codex, OpenCode}, map[string][]ID{".claude/skills": {Claude, OpenCode}, ".agents/skills": {Codex}}, []ID{OpenCode}},
	}
	for _, c := range cases {
		got, dup := SkillLocations(c.ids)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: locs %v want %v", c.name, got, c.want)
		}
		if !reflect.DeepEqual(dup, c.dup) {
			t.Errorf("%s: dup %v want %v", c.name, dup, c.dup)
		}
	}
}
