package platform

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

type ID string

const (
	Claude   ID = "claude"
	Codex    ID = "codex"
	Cursor   ID = "cursor"
	OpenCode ID = "opencode"
)

type AgentFormat string

const (
	AgentClaudeLink AgentFormat = "claude-link"
	AgentCodexTOML  AgentFormat = "codex-toml"
	AgentCursorMD   AgentFormat = "cursor-md"
	AgentOpenCodeMD AgentFormat = "opencode-md"
)

type RulesMode string

const (
	RulesDirMD         RulesMode = "dir-md"
	RulesDirMDC        RulesMode = "dir-mdc"
	RulesAgentsMDBlock RulesMode = "agents-md-block"
)

type HookSchema string

const (
	HooksClaudeSettings HookSchema = "claude-settings"
	HooksCodex          HookSchema = "codex-hooks"
	HooksCursor         HookSchema = "cursor-hooks"
	HooksNone           HookSchema = "none"
)

type AgentSpec struct {
	Dir    string
	Ext    string
	Format AgentFormat
}

type RulesSpec struct {
	Mode RulesMode
	Dir  string
}

type HookSpec struct {
	Dir        string
	ConfigFile string
	Schema     HookSchema
	EventMap   map[string][]string
}

type Platform struct {
	ID                 ID
	SkillDirsRead      []string
	Agents             AgentSpec
	Rules              RulesSpec
	Hooks              HookSpec
	InstructionFile    string
	Verified           map[string]string
	ReservedAgentNames []string
}

const verifiedDate = "2026-10-27"

var order = []ID{Claude, Codex, Cursor, OpenCode}

var table = map[ID]Platform{
	Claude: {
		ID:            Claude,
		SkillDirsRead: []string{ClaudeSkillsLoc},
		Agents:        AgentSpec{Dir: ".claude/agents", Ext: ".md", Format: AgentClaudeLink},
		Rules:         RulesSpec{Mode: RulesDirMD, Dir: ".claude/rules"},
		Hooks: HookSpec{
			Dir:        ".claude/hooks",
			ConfigFile: ".claude/settings.json",
			Schema:     HooksClaudeSettings,
			EventMap:   map[string][]string{},
		},
		InstructionFile: "CLAUDE.md",
		Verified: map[string]string{
			"skills": verifiedDate, "agents": verifiedDate, "rules": verifiedDate, "hooks": verifiedDate,
		},
	},
	Codex: {
		ID:            Codex,
		SkillDirsRead: []string{".agents/skills"},
		Agents:        AgentSpec{Dir: ".codex/agents", Ext: ".toml", Format: AgentCodexTOML},
		Rules:         RulesSpec{Mode: RulesAgentsMDBlock, Dir: ".agents/rules"},
		Hooks: HookSpec{
			Dir:        ".codex/hooks",
			ConfigFile: ".codex/hooks.json",
			Schema:     HooksCodex,
			EventMap:   map[string][]string{},
		},
		InstructionFile:    "AGENTS.md",
		Verified:           map[string]string{"skills": "", "agents": "", "rules": "", "hooks": ""},
		ReservedAgentNames: []string{"default", "worker", "explorer"},
	},
	Cursor: {
		ID:            Cursor,
		SkillDirsRead: []string{".agents/skills", ".cursor/skills", ClaudeSkillsLoc, ".codex/skills"},
		Agents:        AgentSpec{Dir: ".cursor/agents", Ext: ".md", Format: AgentCursorMD},
		Rules:         RulesSpec{Mode: RulesDirMDC, Dir: ".cursor/rules"},
		Hooks: HookSpec{
			Dir:        ".cursor/hooks",
			ConfigFile: ".cursor/hooks.json",
			Schema:     HooksCursor,
			EventMap: map[string][]string{
				"Stop":         {"stop"},
				"SessionStart": {"sessionStart"},
				"SessionEnd":   {"sessionEnd"},
				"PostToolUse":  {"afterFileEdit", "afterShellExecution"},
			},
		},
		InstructionFile: "AGENTS.md",
		Verified:        map[string]string{"skills": "", "agents": "", "rules": "", "hooks": ""},
	},
	OpenCode: {
		ID:            OpenCode,
		SkillDirsRead: []string{".opencode/skills", ClaudeSkillsLoc, ".agents/skills"},
		Agents:        AgentSpec{Dir: ".opencode/agents", Ext: ".md", Format: AgentOpenCodeMD},
		Rules:         RulesSpec{Mode: RulesAgentsMDBlock, Dir: ".agents/rules"},
		Hooks: HookSpec{
			Schema:   HooksNone,
			EventMap: map[string][]string{},
		},
		InstructionFile:    "AGENTS.md",
		Verified:           map[string]string{"skills": "", "agents": "", "rules": "", "hooks": ""},
		ReservedAgentNames: []string{"build", "plan", "general", "explore"},
	},
}

func init() {
	c := table[Codex]
	for _, e := range []string{"Stop", "SessionStart", "SessionEnd", "PostToolUse"} {
		c.Hooks.EventMap[e] = []string{e}
	}
	table[Codex] = c
}

func IDs() []ID {
	return append([]ID(nil), order...)
}

func Get(id ID) (Platform, bool) {
	p, ok := table[id]
	return p, ok
}

func ValidList() string {
	s := make([]string, len(order))
	for i, id := range order {
		s[i] = string(id)
	}
	return strings.Join(s, ", ")
}

func ParsePlatforms(vals []string) ([]ID, error) {
	seen := map[ID]bool{}
	for _, v := range vals {
		for _, tok := range strings.Split(v, ",") {
			tok = strings.ToLower(strings.TrimSpace(tok))
			if tok == "" {
				continue
			}
			if tok == "all" {
				for _, id := range order {
					seen[id] = true
				}
				continue
			}
			id := ID(tok)
			if _, ok := table[id]; !ok {
				return nil, fmt.Errorf("unknown platform %q (valid: %s, all)", tok, ValidList())
			}
			seen[id] = true
		}
	}
	if len(seen) == 0 {
		return []ID{Claude}, nil
	}
	var out []ID
	for _, id := range order {
		if seen[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

func SkillLocations(ids []ID) (map[string][]ID, []ID) {
	locs := map[string][]ID{}
	if len(ids) == 0 {
		return locs, nil
	}
	hasClaude := slices.Contains(ids, Claude)
	for _, id := range order {
		if !slices.Contains(ids, id) {
			continue
		}
		switch id {
		case Claude:
			locs[ClaudeSkillsLoc] = append(locs[ClaudeSkillsLoc], id)
		case Codex:
			locs[".agents/skills"] = append(locs[".agents/skills"], id)
		default:
			if hasClaude {
				locs[ClaudeSkillsLoc] = append(locs[ClaudeSkillsLoc], id)
			} else {
				locs[".agents/skills"] = append(locs[".agents/skills"], id)
			}
		}
	}
	var dup []ID
	for _, id := range []ID{Cursor, OpenCode} {
		if !slices.Contains(ids, id) {
			continue
		}
		n := 0
		for loc := range locs {
			for _, d := range table[id].SkillDirsRead {
				if d == loc {
					n++
					break
				}
			}
		}
		if n >= 2 {
			dup = append(dup, id)
		}
	}
	sort.Slice(dup, func(i, j int) bool { return dup[i] < dup[j] })
	return locs, dup
}
