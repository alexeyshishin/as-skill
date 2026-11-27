package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"claude-harness/tools/internal/fsys"
	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/install"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

type domainSel struct {
	Name   string
	Strict bool
}

type selection struct {
	Domains []domainSel
	Skills  []string
}

func gatedReason(m *harness.Manifest) string {
	miss, gated := m.Gate()
	if !gated {
		return ""
	}
	if miss.Kind == harness.MissEnv {
		return "$" + miss.Name + " is not set"
	}
	return fmt.Sprintf("the %q binary is not on $PATH", miss.Name)
}

func resolveNames(harnessRoot string, reg *harness.Registry, names []string) (selection, error) {
	var sel selection
	explicit := map[string]bool{}
	all := false
	skillOwners := map[string]string{}
	var skillOrder []string
	for _, n := range names {
		if n == "all" {
			all = true
			continue
		}
		if m, ok := reg.Domains[n]; ok {
			if r := gatedReason(m); r != "" {
				return sel, fmt.Errorf("domain %q is gated: %s", n, r)
			}
			explicit[n] = true
			continue
		}
		owner := reg.SkillOwner(harnessRoot, n)
		if owner == "" {
			return sel, fmt.Errorf("unknown name %q (domains: %s; see `as-skill list skills`)", n, strings.Join(reg.DomainNames(), ", "))
		}
		if r := gatedReason(reg.Domains[owner]); r != "" {
			return sel, fmt.Errorf("skill %q belongs to domain %q, which is gated: %s", n, owner, r)
		}
		if _, dup := skillOwners[n]; !dup {
			skillOwners[n] = owner
			skillOrder = append(skillOrder, n)
		}
	}
	seen := map[string]bool{}
	if all {
		for _, d := range reg.DomainNames() {
			sel.Domains = append(sel.Domains, domainSel{d, explicit[d]})
			seen[d] = true
		}
	}
	for _, n := range names {
		if explicit[n] && !seen[n] {
			sel.Domains = append(sel.Domains, domainSel{n, true})
			seen[n] = true
		}
	}
	for _, s := range skillOrder {
		if !seen[skillOwners[s]] {
			sel.Skills = append(sel.Skills, s)
		}
	}
	return sel, nil
}

func (s selection) domainNames() []string {
	out := make([]string, len(s.Domains))
	for i, d := range s.Domains {
		out[i] = d.Name
	}
	return out
}

func detectPlatforms(project string, lock *lockfile.Lockfile) ([]platform.ID, []string) {
	found := map[platform.ID]string{}
	for _, id := range platform.IDs() {
		dir := "." + string(id)
		if id == platform.Claude {
			if !claudeDirMeaningful(filepath.Join(project, dir)) {
				continue
			}
		} else if !fsys.DirExists(filepath.Join(project, dir)) {
			continue
		}
		found[id] = dir
	}
	if lock != nil {
		for _, e := range lock.Entries {
			for _, p := range e.Platforms {
				id := platform.ID(p)
				if _, ok := platform.Get(id); ok {
					if _, have := found[id]; !have {
						found[id] = "lockfile"
					}
				}
			}
		}
	}
	var ids []platform.ID
	var src []string
	for _, id := range platform.IDs() {
		if s, ok := found[id]; ok {
			ids = append(ids, id)
			src = append(src, s)
		}
	}
	if len(ids) == 0 {
		return []platform.ID{platform.Claude}, nil
	}
	return ids, src
}

func claudeDirMeaningful(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() != "skills-lock.json" {
			return true
		}
	}
	return false
}

func uniqStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func platformDirNames(ids []platform.ID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = "." + string(id)
	}
	return out
}

func skillInstalled(lock *lockfile.Lockfile, projectRoot, name string) bool {
	if lock == nil {
		return false
	}
	for _, e := range lock.Entries {
		if e.Domain != "as-skill" && e.Name == name && (e.Kind == lockfile.KindSkill || e.Kind == lockfile.KindSkills) {
			return true
		}
	}
	for _, p := range install.SkillLocPaths(projectRoot, name) {
		if lock.Covers(p) {
			return true
		}
	}
	return false
}

type installState int

const (
	installNone installState = iota
	installPartial
	installFull
)

type domainStatus struct {
	State       installState
	Have, Total int
}

func rowsCover(rows []lockfile.Entry, kind lockfile.Kind, name string) bool {
	for _, e := range rows {
		if !(e.Kind == kind || (kind == lockfile.KindSkills && e.Kind == lockfile.KindSkill)) {
			continue
		}
		if e.Name != "" {
			if e.Name == name || e.Name == strings.TrimSuffix(name, filepath.Ext(name)) {
				return true
			}
			continue
		}
		if len(e.Items) > 0 {
			if slices.Contains(e.Items, name) {
				return true
			}
			continue
		}
		if e.Source == "" {
			return true
		}
		if _, err := os.Lstat(filepath.Join(e.Source, name)); err == nil {
			return true
		}
	}
	return false
}

func domainState(lock *lockfile.Lockfile, harnessRoot, domain string) domainStatus {
	if lock == nil {
		return domainStatus{}
	}
	var rows []lockfile.Entry
	for _, e := range lock.Entries {
		if e.Domain == domain {
			rows = append(rows, e)
		}
	}
	if len(rows) == 0 {
		return domainStatus{}
	}
	var st domainStatus
	for _, kind := range harness.DomainKinds {
		names, err := fsys.TopNames(harness.KindDir(harnessRoot, domain, kind), kind.SkipsReadme())
		if err != nil {
			continue
		}
		for _, n := range names {
			st.Total++
			if rowsCover(rows, kind, n) {
				st.Have++
			}
		}
	}
	switch {
	case st.Total == 0 || st.Have == 0:
		st.State = installNone
	case st.Have == st.Total:
		st.State = installFull
	default:
		st.State = installPartial
	}
	return st
}

func domainInstalled(lock *lockfile.Lockfile, harnessRoot, domain string) bool {
	return domainState(lock, harnessRoot, domain).State == installFull
}
