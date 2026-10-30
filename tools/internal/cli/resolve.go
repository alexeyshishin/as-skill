package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/fsutil"
	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/transfer"
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
	for _, e := range m.RequiresEnv {
		if os.Getenv(e) == "" {
			return "$" + e + " is not set"
		}
	}
	for _, b := range m.RequiresBin {
		if !fsutil.HasBin(b) {
			return fmt.Sprintf("the %q binary is not on $PATH", b)
		}
	}
	return ""
}

func skillOwner(harnessRoot string, reg *harness.Registry, name string) string {
	for _, d := range reg.DomainNames() {
		if fsutil.DirExists(filepath.Join(harnessRoot, "domains", d, "skills", name)) {
			return d
		}
	}
	return ""
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
		owner := skillOwner(harnessRoot, reg, n)
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
		} else if !fsutil.DirExists(filepath.Join(project, dir)) {
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
		if e.Domain != "as-skill" && e.Name == name && (e.Kind == "skill" || e.Kind == "skills") {
			return true
		}
	}
	for _, p := range skillLocPaths(projectRoot, name) {
		if lock.Covers(p) {
			return true
		}
	}
	return false
}

// domainInstalled marks a domain only when it has a domain-wide (kind-level)
// row, or every top-level item of its kinds is covered by some row.
func domainInstalled(lock *lockfile.Lockfile, harnessRoot, domain string) bool {
	if lock == nil {
		return false
	}
	var rows []lockfile.Entry
	for _, e := range lock.Entries {
		if e.Domain != domain {
			continue
		}
		if e.Name == "" {
			return true
		}
		rows = append(rows, e)
	}
	if len(rows) == 0 {
		return false
	}
	total := 0
	for _, kind := range harness.DomainKinds {
		names, err := transfer.TopNames(filepath.Join(harnessRoot, "domains", domain, kind), kind)
		if err != nil {
			continue
		}
		for _, n := range names {
			total++
			covered := false
			for _, e := range rows {
				k := e.Kind == kind || (kind == "skills" && e.Kind == "skill")
				if k && (e.Name == n || e.Name == strings.TrimSuffix(n, filepath.Ext(n))) {
					covered = true
					break
				}
			}
			if !covered {
				return false
			}
		}
	}
	return total > 0
}
