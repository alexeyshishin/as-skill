package platform

import (
	"path/filepath"
	"sort"
	"strings"
)

const (
	ClaudeHome      = ".claude"
	ClaudeSkillsLoc = ClaudeHome + "/skills"
	AgentsRoot      = ".agents"
	PromptsDir      = AgentsRoot + "/prompts"
	AgentsRulesDir  = AgentsRoot + "/rules"
)

var capGaps = map[ID]string{
	Codex:    "agents: frontmatter edits not live (doctor reports STALE); hooks: needs trust via /hooks",
	Cursor:   "agents: frontmatter edits not live (doctor reports STALE); hooks: stop cannot block",
	OpenCode: "agents: frontmatter edits not live (doctor reports STALE); hooks: unsupported",
}

// CapGap returns the known capability gap note for a platform, or "".
func CapGap(id ID) string { return capGaps[id] }

// AllSkillDirs returns the sorted union of every platform's SkillDirsRead.
func AllSkillDirs() []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range order {
		for _, d := range table[id].SkillDirsRead {
			if !seen[d] {
				seen[d] = true
				out = append(out, d)
			}
		}
	}
	sort.Strings(out)
	return out
}

// LinkRoots returns the sorted non-claude platform roots: AgentsRoot plus ".<id>" per non-claude platform.
func LinkRoots() []string {
	out := []string{AgentsRoot}
	for _, id := range order {
		if id != Claude {
			out = append(out, "."+string(id))
		}
	}
	sort.Strings(out)
	return out
}

func IsLinkRoot(root string) bool {
	for _, r := range LinkRoots() {
		if r == root {
			return true
		}
	}
	return false
}

// RootOf returns the first path element of rel.
func RootOf(rel string) string {
	return strings.Split(filepath.ToSlash(rel), "/")[0]
}
