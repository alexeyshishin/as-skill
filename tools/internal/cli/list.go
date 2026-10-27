package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claude-harness/tools/internal/fsutil"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/registry"
)

func runList(args []string) error {
	what := "all"
	if len(args) > 0 {
		what = args[0]
	}
	if what == "platforms" {
		printPlatforms(os.Stdout)
		return nil
	}
	if what != "all" && what != "domains" && what != "skills" {
		return fmt.Errorf("unknown list target %q (want domains|skills|platforms)", what)
	}

	harnessRoot, err := registry.ResolveHarnessRoot(".")
	if err != nil {
		return err
	}
	reg, err := registry.LoadRegistry(harnessRoot)
	if err != nil {
		return err
	}

	if what == "all" || what == "domains" {
		fmt.Println("domains:")
		for _, name := range reg.DomainNames() {
			m := reg.Domains[name]
			status := "available"
			for _, envVar := range m.RequiresEnv {
				if os.Getenv(envVar) == "" {
					status = fmt.Sprintf("blocked — needs $%s", envVar)
					break
				}
			}
			if status == "available" {
				for _, bin := range m.RequiresBin {
					if !fsutil.HasBin(bin) {
						status = fmt.Sprintf("blocked — needs %q binary", bin)
						break
					}
				}
			}
			fmt.Printf("  %-10s %-24s %s\n", name, "["+status+"]", m.Description)
		}
	}
	if what == "all" || what == "skills" {
		fmt.Println("skills:")
		for _, name := range reg.DomainNames() {
			skillsDir := filepath.Join(harnessRoot, "domains", name, "skills")
			entries, err := os.ReadDir(skillsDir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					fmt.Printf("  %-30s (%s)\n", e.Name(), name)
				}
			}
		}
	}
	return nil
}

func platformCaps(p platform.Platform) [][3]string {
	locs, _ := platform.SkillLocations([]platform.ID{p.ID})
	var skillLocs []string
	for l := range locs {
		skillLocs = append(skillLocs, l)
	}
	sort.Strings(skillLocs)

	agentMode := "render"
	if p.Agents.Format == platform.AgentClaudeLink {
		agentMode = "link"
	}
	rulesLoc, rulesMode := p.Rules.Dir, "link"
	switch p.Rules.Mode {
	case platform.RulesDirMDC:
		rulesMode = "render"
	case platform.RulesAgentsMDBlock:
		rulesLoc, rulesMode = p.Rules.Dir+" + "+p.InstructionFile, "block"
	}
	hooksLoc, hooksMode := p.Hooks.Dir+" -> "+p.Hooks.ConfigFile, "link"
	if p.Hooks.Schema == platform.HooksNone {
		hooksLoc, hooksMode = "-", "unsupported"
	}
	return [][3]string{
		{"skills", strings.Join(skillLocs, ","), "link"},
		{"agents", p.Agents.Dir, agentMode},
		{"rules", rulesLoc, rulesMode},
		{"hooks", hooksLoc, hooksMode},
	}
}

func printPlatforms(w io.Writer) {
	fmt.Fprintf(w, "%-9s %-7s %-34s %-12s %s\n", "PLATFORM", "KIND", "LOCATION", "MODE", "VERIFIED")
	for _, id := range platform.IDs() {
		p, _ := platform.Get(id)
		for _, c := range platformCaps(p) {
			ver := "unverified"
			if v := p.Verified[c[0]]; v != "" {
				ver = "verified " + v
			}
			if c[2] == "unsupported" {
				ver = "-"
				if v := p.Verified[c[0]]; v != "" {
					ver = "verified " + v
				}
			}
			fmt.Fprintf(w, "%-9s %-7s %-34s %-12s %s\n", id, c[0], c[1], c[2], ver)
		}
	}
}
