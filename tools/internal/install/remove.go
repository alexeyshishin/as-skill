package install

import (
	"errors"
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

type RemoveTarget struct {
	Name   string
	Domain bool
	Rows   []lockfile.Entry
	Cover  []lockfile.Entry
}

type PartOfDomainError struct {
	Skill  string
	Domain string
	Mode   lockfile.Mode
}

func (e *PartOfDomainError) Error() string {
	return fmt.Sprintf("skill %q installed as part of domain %s (%s); remove the domain", e.Skill, e.Domain, e.Mode)
}

type removeUnit struct {
	target  RemoveTarget
	harness bool
}

type removeState struct {
	in      *Installer
	reg     *harness.Registry
	want    map[string]bool
	cli     bool
	refused []string
	legacy  []string
}

func SkillLocPaths(projectRoot, name string) []string {
	locs, _ := platform.SkillLocations(platform.IDs())
	var out []string
	for l := range locs {
		out = append(out, filepath.Join(projectRoot, l, name))
	}
	sort.Strings(out)
	return out
}

func coverRows(lock *lockfile.Lockfile, projectRoot, name string) []lockfile.Entry {
	var out []lockfile.Entry
	for _, p := range SkillLocPaths(projectRoot, name) {
		if e, ok := lock.Covering(p); ok && e.Name == "" {
			out = append(out, e)
		}
	}
	return out
}

func ResolveRemoveTargets(lock *lockfile.Lockfile, projectRoot string, names []string) ([]RemoveTarget, error) {
	var out []RemoveTarget
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			continue
		}
		seen[n] = true
		var dRows, sRows []lockfile.Entry
		for _, e := range lock.Entries {
			if e.Domain == n && e.Domain != "as-skill" {
				dRows = append(dRows, e)
			}
			if e.Name == n && e.Domain != "as-skill" && (e.Kind == lockfile.KindSkill || e.Kind == lockfile.KindSkills) {
				sRows = append(sRows, e)
			}
		}
		switch {
		case len(dRows) > 0:
			out = append(out, RemoveTarget{n, true, dRows, nil})
		case len(sRows) > 0:
			out = append(out, RemoveTarget{n, false, sRows, coverRows(lock, projectRoot, n)})
		default:
			if cov := coverRows(lock, projectRoot, n); len(cov) > 0 {
				return nil, &PartOfDomainError{Skill: n, Domain: cov[0].Domain, Mode: cov[0].Mode}
			}
			return nil, fmt.Errorf("%q is not installed in this project (see `as-skill doctor`)", n)
		}
	}
	return out, nil
}

func checkPartOfDomain(t RemoveTarget, plats []platform.ID) error {
	if t.Domain {
		return nil
	}
	for _, c := range t.Cover {
		hit := false
		for _, p := range c.Platforms {
			for _, id := range plats {
				if p == string(id) {
					hit = true
				}
			}
		}
		if !hit {
			continue
		}
		own := false
		for _, r := range t.Rows {
			if filepath.Clean(r.Path) == filepath.Join(filepath.Clean(c.Path), t.Name) {
				own = true
			}
		}
		if !own {
			return &PartOfDomainError{Skill: t.Name, Domain: c.Domain, Mode: c.Mode}
		}
	}
	return nil
}

func (in *Installer) newRemovePlan(reg *harness.Registry, cli bool) *Plan {
	st := &removeState{in: in, reg: reg, want: map[string]bool{}, cli: cli}
	for _, p := range in.plats() {
		st.want[string(p)] = true
	}
	return &Plan{rm: st}
}

func (in *Installer) RemovePlan(reg *harness.Registry, targets []RemoveTarget) (*Plan, error) {
	for _, t := range targets {
		if err := checkPartOfDomain(t, in.plats()); err != nil {
			return nil, err
		}
	}
	p := in.newRemovePlan(reg, true)
	for _, t := range targets {
		u := &removeUnit{target: t}
		if reg != nil {
			if t.Domain {
				_, u.harness = reg.Domains[t.Name]
			} else {
				u.harness = reg.SkillOwner(in.HarnessRoot, t.Name) != ""
			}
		}
		p.Ops = append(p.Ops, Op{Type: OpRemove, Name: t.Name, rm: u})
	}
	return p, nil
}

func (in *Installer) UninstallDomain(reg *harness.Registry, name string) error {
	if _, ok := reg.Domains[name]; !ok {
		return fmt.Errorf("unknown domain %q (available: %s)", name, strings.Join(reg.DomainNames(), ", "))
	}
	p := in.newRemovePlan(reg, false)
	p.Ops = []Op{{Type: OpRemove, Name: name, rm: &removeUnit{target: RemoveTarget{Name: name, Domain: true}, harness: true}}}
	return in.Apply(p)
}

func (in *Installer) UninstallSkill(reg *harness.Registry, name string) error {
	if reg.SkillOwner(in.HarnessRoot, name) == "" {
		return fmt.Errorf("unknown skill %q (see `as-skill list skills`)", name)
	}
	p := in.newRemovePlan(reg, false)
	p.Ops = []Op{{Type: OpRemove, Name: name, rm: &removeUnit{target: RemoveTarget{Name: name}, harness: true}}}
	return in.Apply(p)
}

func (p *Plan) Validate() error {
	if p.rm == nil {
		return nil
	}
	return p.rm.validate(p)
}

func (p *Plan) Problems() error {
	if p.rm == nil {
		return nil
	}
	return p.rm.problems()
}

func (st *removeState) validate(p *Plan) error {
	in := st.in
	if st.cli {
		for _, op := range p.Ops {
			if err := st.preflight(op.rm.target.Rows); err != nil {
				return err
			}
		}
	}
	for _, op := range p.Ops {
		if !op.rm.harness {
			continue
		}
		t := op.rm.target
		domain, skill := t.Name, ""
		if !t.Domain {
			domain, skill = st.reg.SkillOwner(in.HarnessRoot, t.Name), t.Name
		}
		if err := in.preflightUninstall(st.reg, domain, skill); err != nil {
			return err
		}
	}
	return nil
}

func (st *removeState) rowHit(e lockfile.Entry) bool {
	if len(e.Platforms) == 0 {
		return true
	}
	for _, p := range e.Platforms {
		if st.want[p] {
			return true
		}
	}
	return false
}

func (st *removeState) preflight(rows []lockfile.Entry) error {
	in := st.in
	var bad []string
	for _, e := range rows {
		if !st.rowHit(e) || e.Mode == lockfile.ModeBlock {
			continue
		}
		paths, _ := in.rowPaths(e)
		for _, p := range paths {
			if InsideRoots(in.ProjectRoot, p) != nil {
				bad = append(bad, p)
			}
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("remove refused, nothing changed: lockfile rows point outside the project's platform dirs: %s", strings.Join(bad, ", "))
	}
	return nil
}

func (st *removeState) problems() error {
	if len(st.refused) == 0 && len(st.legacy) == 0 {
		return nil
	}
	var msg []string
	if len(st.refused) > 0 {
		msg = append(msg, fmt.Sprintf("refused %d path(s) outside the project's platform dirs (rows kept): %s", len(st.refused), strings.Join(st.refused, ", ")))
	}
	if len(st.legacy) > 0 {
		msg = append(msg, fmt.Sprintf("%d legacy copy row(s) have no recorded item list and their source checkout is gone, so nothing was removed for them (rows kept): %s; reinstall once with the current binary (as-skill add ... --copy) and remove again, or delete the files manually",
			len(st.legacy), strings.Join(st.legacy, ", ")))
	}
	return fmt.Errorf("remove incomplete: %s", strings.Join(msg, "; "))
}

func (in *Installer) applyRemove(p *Plan) error {
	st := p.rm
	for _, op := range p.Ops {
		u := op.rm
		if u.harness {
			var err error
			if u.target.Domain {
				err = in.uninstallDomain(st.reg, u.target.Name)
			} else {
				err = in.uninstallSkill(st.reg, u.target.Name)
				var pe *PartOfDomainError
				if st.cli && errors.As(err, &pe) {
					err = fmt.Errorf("%s", pe.Error())
				}
			}
			if err != nil {
				return err
			}
			if st.cli {
				in.sweep(st, u.target.Rows, true)
			}
			continue
		}
		if st.cli {
			in.sweep(st, u.target.Rows, false)
		}
	}
	return nil
}

func (in *Installer) uninstallDomain(reg *harness.Registry, name string) error {
	if in.hasClaude() {
		if err := in.uninstallClaudeDomain(reg.Domains[name], name); err != nil {
			return err
		}
	}
	in.uninstallNonClaude(name, "")
	return nil
}

func (in *Installer) uninstallClaudeDomain(m *harness.Manifest, name string) error {
	for _, kind := range harness.DomainKinds {
		srcDir := harness.KindDir(in.HarnessRoot, name, kind)
		if !fsys.DirExists(srcDir) {
			continue
		}
		destPath, err := in.resolveTarget(m, kind)
		if err != nil {
			return fmt.Errorf("domain %q: %w", name, err)
		}
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			return fmt.Errorf("domain %q: %w", name, err)
		}
		var goneHooks []string
		for _, e := range entries {
			itemName := e.Name()
			if fsys.SkipTop(kind.SkipsReadme(), itemName) {
				continue
			}
			dst := filepath.Join(destPath, itemName)
			if gone := in.uninstallOne(name, kind, itemName, dst); gone && kind == lockfile.KindHooks {
				goneHooks = append(goneHooks, itemName)
			}
		}
		if kind == lockfile.KindHooks {
			in.unregisterHooks(goneHooks)
		}

		if parent, tracked := in.Lock.Find(name, kind, ""); tracked && len(minusPlats(parent.Platforms, in.plats())) == 0 {
			in.Lock.Remove(name, kind, "")
		}
	}
	return nil
}

func (in *Installer) uninstallSkill(reg *harness.Registry, name string) error {
	owner := reg.SkillOwner(in.HarnessRoot, name)
	m := reg.Domains[owner]
	if in.hasClaude() {
		destBase, err := in.resolveTarget(m, lockfile.KindSkills)
		if err != nil {
			return fmt.Errorf("skill %q: %w", name, err)
		}
		dst := filepath.Join(destBase, name)
		if _, owned := in.Lock.Owns(dst); !owned {
			if parent, ok := in.Lock.Covering(dst); ok && parent.Name == "" {
				if _, err := os.Lstat(dst); err == nil {
					return &PartOfDomainError{Skill: name, Domain: parent.Domain, Mode: parent.Mode}
				}
			}
		}
		in.uninstallOne(owner, lockfile.KindSkill, name, dst)
	}
	in.uninstallNonClaude(owner, name)
	return nil
}

func (in *Installer) uninstallOne(domain string, kind lockfile.Kind, name, dst string) bool {
	entry, tracked := in.Lock.Covering(dst)
	if !tracked {
		if parent, ok := in.Lock.Find(domain, kind, ""); ok {
			entry, tracked = parent, true
		}
	}
	if !tracked {
		if _, err := os.Lstat(dst); os.IsNotExist(err) {
			return false
		}
		fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — not tracked by as-skill\n", dst)
		return false
	}
	fi, err := os.Lstat(dst)
	if err != nil {
		fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — tracked but nothing on disk\n", dst)
		return os.IsNotExist(err)
	}
	if remaining := minusPlats(entry.Platforms, in.plats()); len(entry.Platforms) > 0 && len(remaining) == len(entry.Platforms) {
		return false
	} else if len(remaining) > 0 {
		if in.DryRun {
			fmt.Fprintf(in.Stdout(), "would keep     %s — still used by %s\n", dst, strings.Join(remaining, ","))
			return false
		}
		in.setEntryPlatforms(entry, remaining)
		fmt.Fprintf(in.Stdout(), "kept           %s — still used by %s\n", dst, strings.Join(remaining, ","))
		return false
	}
	isSymlink := fi.Mode()&os.ModeSymlink != 0
	switch entry.Mode {
	case lockfile.ModeLink:
		if !isSymlink {
			fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — tracked as a link but not a symlink on disk\n", dst)
			return false
		}
	case lockfile.ModeCopy:
		if isSymlink {
			fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — tracked as a copy but is a symlink on disk\n", dst)
			return false
		}
	default:
		fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — unrecognized tracked mode %q\n", dst, entry.Mode)
		return false
	}
	if in.DryRun {
		fmt.Fprintf(in.Stdout(), "would remove   %s\n", dst)
		return true
	}
	var removeErr error
	if entry.Mode == lockfile.ModeLink {
		removeErr = os.Remove(dst)
	} else {
		removeErr = os.RemoveAll(dst)
	}
	if removeErr != nil {
		fmt.Fprintf(in.Stderr(), "as-skill: warning: failed removing %s: %v\n", dst, removeErr)
		return false
	}
	in.Lock.RemoveAllAt(dst)
	fmt.Fprintf(in.Stdout(), "uninstalled    %s\n", dst)
	return true
}

func (in *Installer) uninstallNonClaude(domain, skill string) {
	kinds := map[lockfile.Kind]bool{lockfile.KindRules: true, lockfile.KindSkills: true, lockfile.KindSkill: true, lockfile.KindAgents: true, lockfile.KindHooks: true, lockfile.KindPrompts: true}
	plats := in.plats()
	gone := map[platform.ID][]string{}
	snapshot := append([]lockfile.Entry(nil), in.Lock.Entries...)
	for _, e := range snapshot {
		if e.Domain != domain || !kinds[e.Kind] {
			continue
		}
		if skill != "" && (e.Name != skill || (e.Kind != lockfile.KindSkill && e.Kind != lockfile.KindSkills)) {
			continue
		}
		if in.hasClaude() && fsys.Within(filepath.Join(in.ProjectRoot, platform.ClaudeHome), e.Path) {
			continue
		}
		remaining := minusPlats(e.Platforms, plats)
		if len(remaining) == len(e.Platforms) {
			continue
		}
		if len(remaining) > 0 {
			if in.DryRun {
				fmt.Fprintf(in.Stdout(), "would keep     %s — still used by %s\n", e.Path, strings.Join(remaining, ","))
				continue
			}
			in.setEntryPlatforms(e, remaining)
			fmt.Fprintf(in.Stdout(), "kept           %s — still used by %s\n", e.Path, strings.Join(remaining, ","))
			continue
		}
		if e.Name != "" && in.removeEntry(e) && e.Kind == lockfile.KindHooks {
			for _, id := range plats {
				gone[id] = append(gone[id], e.Name)
			}
		} else if e.Name == "" && !in.DryRun {
			in.Lock.RemoveAt(e.Domain, e.Kind, e.Name, e.Path)
		}
	}
	for id, names := range gone {
		if id != platform.Claude {
			in.unregisterHooksFor(id, names)
		}
	}
	if err := in.syncAgentsMD(); err != nil {
		fmt.Fprintf(in.Stderr(), "as-skill: warning: updating %s: %v\n", agentsMD, err)
	}
}

func (in *Installer) removeEntry(e lockfile.Entry) bool {
	fi, err := os.Lstat(e.Path)
	if err != nil {
		fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — tracked but nothing on disk\n", e.Path)
		return false
	}
	isLink := fi.Mode()&os.ModeSymlink != 0
	switch e.Mode {
	case lockfile.ModeLink:
		if !isLink {
			fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — tracked as a link but not a symlink on disk\n", e.Path)
			return false
		}
	case lockfile.ModeCopy, lockfile.ModeRender:
		if isLink {
			fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — tracked as %s but is a symlink on disk\n", e.Path, e.Mode)
			return false
		}
	default:
		fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — unrecognized tracked mode %q\n", e.Path, e.Mode)
		return false
	}
	if err := InsideRoots(in.ProjectRoot, e.Path); err != nil {
		fmt.Fprintf(in.Stderr(), "as-skill: error: refusing %v\n", err)
		return false
	}
	if in.DryRun {
		fmt.Fprintf(in.Stdout(), "would remove   %s\n", e.Path)
		return true
	}
	var rerr error
	if e.Mode == lockfile.ModeCopy {
		rerr = os.RemoveAll(e.Path)
	} else {
		rerr = os.Remove(e.Path)
	}
	if rerr != nil {
		fmt.Fprintf(in.Stderr(), "as-skill: warning: failed removing %s: %v\n", e.Path, rerr)
		return false
	}
	in.Lock.RemoveAt(e.Domain, e.Kind, e.Name, e.Path)
	fmt.Fprintf(in.Stdout(), "uninstalled    %s\n", e.Path)
	return true
}

func (in *Installer) sweep(st *removeState, rows []lockfile.Entry, afterHarness bool) {
	for _, row := range rows {
		e, ok := in.Lock.FindAt(row.Domain, row.Kind, row.Name, row.Path)
		if !ok {
			continue
		}
		if afterHarness && in.DryRun && (e.Source == "" || e.Mode == lockfile.ModeCopy || fsys.PathExists(e.Source)) {
			continue
		}
		in.removeRow(st, e)
	}
}

func (in *Installer) removeRow(st *removeState, e lockfile.Entry) {
	if !st.rowHit(e) {
		return
	}
	if e.Mode == lockfile.ModeBlock {
		fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — managed block needs the harness checkout\n", e.Path)
		return
	}
	var remaining []string
	for _, p := range e.Platforms {
		if !st.want[p] {
			remaining = append(remaining, p)
		}
	}
	if len(remaining) > 0 {
		if in.DryRun {
			fmt.Fprintf(in.Stdout(), "would keep     %s — still used by %s\n", e.Path, strings.Join(remaining, ","))
			return
		}
		e.Platforms = remaining
		in.Lock.Record(e)
		fmt.Fprintf(in.Stdout(), "kept           %s — still used by %s\n", e.Path, strings.Join(remaining, ","))
		return
	}
	paths, ok := in.rowPaths(e)
	if !ok {
		fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — legacy copy row has no item list and its source is gone\n", e.Path)
		st.legacy = append(st.legacy, e.Path)
		return
	}
	clean := true
	var goneHooks []string
	for _, p := range paths {
		if ierr := InsideRoots(in.ProjectRoot, p); ierr != nil {
			fmt.Fprintf(in.Stderr(), "as-skill: error: refusing %v\n", ierr)
			st.refused = append(st.refused, p)
			clean = false
			continue
		}
		fi, err := os.Lstat(p)
		if err != nil {
			continue
		}
		isLink := fi.Mode()&os.ModeSymlink != 0
		if e.Mode == lockfile.ModeLink && !isLink {
			fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — tracked as a link but not a symlink on disk\n", p)
			clean = false
			continue
		}
		if (e.Mode == lockfile.ModeCopy || e.Mode == lockfile.ModeRender) && isLink {
			fmt.Fprintf(in.Stderr(), "as-skill: warning: skipping %s — tracked as %s but is a symlink on disk\n", p, e.Mode)
			clean = false
			continue
		}
		if in.DryRun {
			fmt.Fprintf(in.Stdout(), "would remove   %s\n", p)
			if e.Kind == lockfile.KindHooks {
				goneHooks = append(goneHooks, filepath.Base(p))
			}
			continue
		}
		var rerr error
		if isLink {
			rerr = os.Remove(p)
		} else {
			rerr = os.RemoveAll(p)
		}
		if rerr != nil {
			fmt.Fprintf(in.Stderr(), "as-skill: warning: failed removing %s: %v\n", p, rerr)
			clean = false
			continue
		}
		fmt.Fprintf(in.Stdout(), "uninstalled    %s\n", p)
		if e.Kind == lockfile.KindHooks {
			goneHooks = append(goneHooks, filepath.Base(p))
		}
	}
	if len(goneHooks) > 0 {
		for _, id := range in.rowPlats(st, e) {
			in.unregisterHooksFor(id, goneHooks)
		}
	}
	if clean && !in.DryRun {
		if e.Name == "" {
			in.Lock.RemoveAt(e.Domain, e.Kind, "", e.Path)
		} else {
			in.Lock.RemoveAllAt(e.Path)
		}
	}
}

func (in *Installer) rowPlats(st *removeState, e lockfile.Entry) []platform.ID {
	if len(e.Platforms) == 0 {
		return in.plats()
	}
	var out []platform.ID
	for _, id := range platform.IDs() {
		for _, p := range e.Platforms {
			if p == string(id) && st.want[p] {
				out = append(out, id)
			}
		}
	}
	return out
}

func (in *Installer) rowPaths(e lockfile.Entry) (paths []string, ok bool) {
	switch {
	case e.Name != "":
		return []string{e.Path}, true
	case e.Mode == lockfile.ModeCopy && len(e.Items) > 0:
		return e.CoveredPaths(), true
	case e.Mode == lockfile.ModeCopy:
		names, err := fsys.TopNames(e.Source, e.Kind.SkipsReadme())
		if err != nil {
			return nil, false
		}
		for _, n := range names {
			if p := filepath.Join(e.Path, n); in.Lock.Covers(p) {
				paths = append(paths, p)
			}
		}
		return paths, true
	}
	return []string{e.Path}, true
}
