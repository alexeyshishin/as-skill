package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/iostreams"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func runList(args []string, ios *iostreams.IOStreams) error {
	what := "all"
	project, harnessFlag := ".", ""
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-p" || a == "--project":
			i++
			if i >= len(args) {
				return fmt.Errorf("%s needs a value", a)
			}
			project = args[i]
		case a == "--harness-root":
			i++
			if i >= len(args) {
				return fmt.Errorf("%s needs a value", a)
			}
			harnessFlag = args[i]
		case strings.HasPrefix(a, "--project="):
			project = strings.TrimPrefix(a, "--project=")
		case strings.HasPrefix(a, "-"):
			return fmt.Errorf("unknown flag %q", a)
		default:
			pos = append(pos, a)
		}
	}
	if len(pos) > 0 {
		what = pos[0]
	}
	if what == "platforms" {
		printPlatforms(ios.Stdout())
		return nil
	}
	if what != "all" && what != "domains" && what != "skills" {
		return fmt.Errorf("unknown list target %q (want domains|skills|platforms)", what)
	}

	harnessRoot, err := harness.ResolveHarnessRootAuto(harnessFlag)
	if err != nil {
		return err
	}
	reg, err := harness.LoadRegistry(harnessRoot)
	if err != nil {
		return err
	}

	var lock *lockfile.Lockfile
	projectRoot := ""
	if p, perr := resolveProject(project); perr != nil {
		return perr
	} else if lk, lerr := lockfile.Load(p); lerr == nil {
		lock, projectRoot = lk, p
	}
	mark := func(ok bool) string {
		if ok {
			return " installed"
		}
		return ""
	}

	if what == "all" || what == "domains" {
		fmt.Fprintln(ios.Stdout(), "domains:")
		for _, name := range reg.DomainNames() {
			m := reg.Domains[name]
			status := "available"
			if miss, gated := m.Gate(); gated {
				if miss.Kind == harness.MissEnv {
					status = fmt.Sprintf("blocked — needs $%s", miss.Name)
				} else {
					status = fmt.Sprintf("blocked — needs %q binary", miss.Name)
				}
			}
			fmt.Fprintf(ios.Stdout(), "  %-10s %-24s %s%s\n", name, "["+status+"]", m.Description, mark(domainInstalled(lock, harnessRoot, name)))
		}
	}
	if what == "all" || what == "skills" {
		fmt.Fprintln(ios.Stdout(), "skills:")
		for _, name := range reg.DomainNames() {
			skillsDir := harness.KindDir(harnessRoot, name, lockfile.KindSkills)
			entries, err := os.ReadDir(skillsDir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					installed := skillInstalled(lock, projectRoot, e.Name())
					fmt.Fprintf(ios.Stdout(), "  %-30s (%s)%s\n", e.Name(), name, mark(installed))
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
