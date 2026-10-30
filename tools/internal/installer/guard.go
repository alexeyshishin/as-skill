package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claude-harness/tools/internal/fsys"
	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func PlatformRoots(project string) []string {
	seen := map[string]bool{}
	add := func(rel string) {
		if rel != "" {
			seen[strings.Split(filepath.ToSlash(rel), "/")[0]] = true
		}
	}
	for _, id := range platform.IDs() {
		p, _ := platform.Get(id)
		add("." + string(id))
		add(p.Agents.Dir)
		add(p.Rules.Dir)
		if p.Hooks.Schema != platform.HooksNone {
			add(p.Hooks.Dir)
		}
	}
	locs, _ := platform.SkillLocations(platform.IDs())
	for l := range locs {
		add(l)
	}
	var out []string
	for r := range seen {
		out = append(out, filepath.Join(project, r))
	}
	sort.Strings(out)
	return out
}

func InsideRoots(projectRoot, p string) error {
	p = filepath.Clean(p)
	matched := ""
	if filepath.IsAbs(p) {
		for _, root := range PlatformRoots(projectRoot) {
			if fsys.StrictlyInside(root, p) {
				matched = root
				break
			}
		}
	}
	if matched == "" {
		return fmt.Errorf("%s is outside the project's platform dirs", p)
	}
	rp, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		return fmt.Errorf("%s resolves outside the project's platform dirs", p)
	}
	base := filepath.Join(rp, filepath.Base(matched))
	if _, err := os.Lstat(matched); err != nil {
		return nil
	}
	if rr, err := filepath.EvalSymlinks(matched); err != nil || rr != base {
		return fmt.Errorf("%s: platform root %s does not resolve to %s", p, matched, base)
	}
	anc := filepath.Dir(p)
	for {
		if _, err := os.Lstat(anc); err == nil {
			break
		}
		next := filepath.Dir(anc)
		if next == anc {
			break
		}
		anc = next
	}
	parent, err := filepath.EvalSymlinks(anc)
	if err == nil {
		if fsys.Within(base, parent) {
			return nil
		}
	}
	return fmt.Errorf("%s resolves outside the project's platform dirs", p)
}

func (in *Installer) preflightUninstall(reg *harness.Registry, domain, skill string) error {
	kinds := map[lockfile.Kind]bool{lockfile.KindRules: true, lockfile.KindSkills: true, lockfile.KindSkill: true, lockfile.KindAgents: true, lockfile.KindHooks: true, lockfile.KindPrompts: true}
	var bad []string
	check := func(p string) {
		if err := InsideRoots(in.ProjectRoot, p); err != nil {
			bad = append(bad, err.Error())
		}
	}
	for _, e := range in.Lock.Entries {
		if e.Domain != domain || !kinds[e.Kind] || e.Mode == lockfile.ModeBlock {
			continue
		}
		if skill != "" && (e.Name != skill || (e.Kind != lockfile.KindSkill && e.Kind != lockfile.KindSkills)) {
			continue
		}
		check(e.Path)
		if e.Name == "" {
			for _, p := range e.CoveredPaths() {
				check(p)
			}
		}
	}
	if in.hasClaude() {
		if m, ok := reg.Domains[domain]; ok {
			kindList := harness.DomainKinds
			if skill != "" {
				kindList = []lockfile.Kind{lockfile.KindSkills}
			}
			for _, kind := range kindList {
				destPath, err := in.resolveTarget(m, kind)
				if err != nil {
					continue
				}
				if skill != "" {
					check(filepath.Join(destPath, skill))
					continue
				}
				entries, err := os.ReadDir(harness.KindDir(in.HarnessRoot, domain, kind))
				if err != nil {
					continue
				}
				for _, e := range entries {
					if !fsys.SkipTop(kind.SkipsReadme(), e.Name()) {
						check(filepath.Join(destPath, e.Name()))
					}
				}
			}
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("uninstall refused, nothing changed: %s", strings.Join(bad, "; "))
	}
	return nil
}
