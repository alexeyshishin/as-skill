package installer

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

// DomainReq names a domain to install; Strict makes a gated domain an error instead of a skip.
type DomainReq struct {
	Name   string
	Strict bool
}

// Planner builds one Plan. It owns the per-Plan state: the git-ignore cache and
// the one-shot notes. collect gathers conflicts into Plan.Conflicts (and checks
// every op read-only); otherwise the first conflict is returned as an error.
type Planner struct {
	in        *Installer
	plan      *Plan
	collect   bool
	git       gitCache
	dupNoted  bool
	gapsNoted bool
}

func (in *Installer) newPlanner(collect bool) *Planner {
	g := gitCache{}
	return &Planner{in: in, plan: &Plan{git: g}, collect: collect, git: g}
}

// add records a conflict (deduped) or, when not collecting, returns it as the error.
func (pl *Planner) add(c Conflict) error {
	if !pl.collect {
		return &c
	}
	for _, x := range pl.plan.Conflicts {
		if x == c {
			return nil
		}
	}
	pl.plan.Conflicts = append(pl.plan.Conflicts, c)
	return nil
}

func (pl *Planner) guardRoot(root string) error {
	if pl.in.unignored(pl.git, root) {
		return pl.add(Conflict{Path: root, Reason: ReasonNotIgnored})
	}
	return nil
}

func (pl *Planner) guard(dst string) error {
	in := pl.in
	if in.Mode != lockfile.ModeLink || in.Force {
		return nil
	}
	root, ok := in.dstRoot(dst)
	if !ok {
		return nil
	}
	return pl.guardRoot(root)
}

func (pl *Planner) foreign(dst string) error {
	if pl.in.isForeign(dst) {
		return pl.add(Conflict{Path: dst, Reason: ReasonForeign})
	}
	return nil
}

func (pl *Planner) noteGaps() {
	pl.plan.merge(pl.planGaps())
}

func (pl *Planner) preflightRoots(kinds map[lockfile.Kind]bool) error {
	pl.noteGaps()
	for _, r := range pl.in.touchedRoots(kinds) {
		if err := pl.guardRoot(r); err != nil {
			return err
		}
	}
	return nil
}

func (in *Installer) runnable(m *harness.Manifest) bool {
	_, gated := m.Gate()
	return !gated
}

func (pl *Planner) preflight(reg *harness.Registry, domains []string) error {
	in := pl.in
	for _, name := range domains {
		m, ok := reg.Domains[name]
		if !ok || !in.runnable(m) {
			continue
		}
		kinds := map[lockfile.Kind]bool{}
		for _, kind := range harness.DomainKinds {
			if fsys.DirExists(harness.KindDir(in.HarnessRoot, name, kind)) {
				kinds[kind] = true
			}
		}
		if kinds[lockfile.KindAgents] {
			if err := in.checkReserved(harness.KindDir(in.HarnessRoot, name, lockfile.KindAgents)); err != nil {
				return fmt.Errorf("domain %q: %w", name, err)
			}
		}
		if err := pl.preflightRoots(kinds); err != nil {
			return err
		}
		if err := pl.preflightForeign(name, kinds); err != nil {
			return err
		}
	}
	return nil
}

func (pl *Planner) preflightForeign(name string, kinds map[lockfile.Kind]bool) error {
	in := pl.in
	var paths []string
	itemPaths := func(kind lockfile.Kind, srcDir, dstDir string, plats []platform.ID) {
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if fsys.SkipTop(kind.SkipsReadme(), e.Name()) || in.excluded(name, kind, e.Name()) || !hookWanted(kind, filepath.Join(srcDir, e.Name()), plats) {
				continue
			}
			paths = append(paths, filepath.Join(dstDir, e.Name()))
		}
	}
	for kind := range kinds {
		srcDir := harness.KindDir(in.HarnessRoot, name, kind)
		if kind == lockfile.KindSkills {
			locs, _ := platform.SkillLocations(in.plats())
			for loc, plats := range locs {
				if (loc != platform.ClaudeSkillsLoc || in.hasExclusions(name)) && in.Mode != lockfile.ModeLink {
					itemPaths(kind, srcDir, filepath.Join(in.ProjectRoot, loc), plats)
				}
			}
			continue
		}
		for _, id := range in.plats() {
			if id == platform.Claude {
				continue
			}
			p, _ := platform.Get(id)
			switch kind {
			case lockfile.KindHooks:
				if p.Hooks.Schema != platform.HooksNone && in.Mode != lockfile.ModeLink {
					itemPaths(kind, srcDir, filepath.Join(in.ProjectRoot, p.Hooks.Dir), []platform.ID{id})
				}
			case lockfile.KindAgents, lockfile.KindRules:
				entries, err := in.mdItems(srcDir)
				if err != nil {
					return err
				}
				for _, e := range entries {
					stem := strings.TrimSuffix(e.Name(), ".md")
					switch {
					case kind == lockfile.KindAgents:
						data, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
						if err != nil {
							return err
						}
						paths = append(paths, filepath.Join(in.ProjectRoot, p.Agents.Dir, agentNameOf(data, stem)+p.Agents.Ext))
					case p.Rules.Mode == platform.RulesDirMDC:
						paths = append(paths, filepath.Join(in.ProjectRoot, p.Rules.Dir, stem+".mdc"))
					case in.Mode != lockfile.ModeLink:
						paths = append(paths, filepath.Join(in.ProjectRoot, p.Rules.Dir, e.Name()))
					}
				}
			}
		}
	}
	for _, dst := range paths {
		if err := pl.foreign(dst); err != nil {
			return fmt.Errorf("domain %q: %w", name, err)
		}
	}
	return nil
}

// PreflightError marks a refusal before anything was planned or placed.
type PreflightError struct{ Err error }

func (e *PreflightError) Error() string { return e.Err.Error() }
func (e *PreflightError) Unwrap() error { return e.Err }

// build plans domains then skills into pl.plan. On error pl.plan holds
// everything planned before the failure; the caller applies it first.
func (pl *Planner) build(reg *harness.Registry, domains []DomainReq, skills []string) error {
	if len(domains) > 0 {
		names := make([]string, len(domains))
		for i, d := range domains {
			names[i] = d.Name
		}
		if err := pl.preflight(reg, names); err != nil {
			return &PreflightError{Err: err}
		}
	}
	for _, d := range domains {
		if err := pl.buildDomain(reg, d); err != nil {
			return err
		}
	}
	for _, s := range skills {
		if err := pl.buildSkill(reg, s); err != nil {
			return err
		}
	}
	return nil
}

func (pl *Planner) buildDomain(reg *harness.Registry, d DomainReq) error {
	in := pl.in
	name := d.Name
	m, ok := reg.Domains[name]
	if !ok {
		return fmt.Errorf("unknown domain %q (available: %s)", name, strings.Join(reg.DomainNames(), ", "))
	}
	if miss, gated := m.Gate(); gated {
		var msg string
		if miss.Kind == harness.MissEnv {
			msg = fmt.Sprintf("domain %q requires $%s, which is not set", name, miss.Name)
		} else {
			msg = fmt.Sprintf("domain %q requires the %q binary, which is not on $PATH", name, miss.Name)
		}
		if d.Strict {
			return errors.New(msg)
		}
		pl.plan.Warnings = append(pl.plan.Warnings, Warning{
			Text: "as-skill: skipping — " + msg, Stream: Stderr, Before: len(pl.plan.Ops),
		})
		return nil
	}
	for _, kind := range harness.DomainKinds {
		srcDir := harness.KindDir(in.HarnessRoot, name, kind)
		if !fsys.DirExists(srcDir) {
			continue
		}
		sub, perr := pl.planKind(m, name, kind, srcDir)
		if sub == nil {
			sub = &Plan{}
		}
		for i := range sub.Ops {
			sub.Ops[i].unit, sub.Ops[i].unitKind = name, kind
		}
		if err := pl.addChecked(sub); err != nil {
			return err
		}
		if perr != nil {
			return perr
		}
	}
	return nil
}

func (pl *Planner) buildSkill(reg *harness.Registry, name string) error {
	in := pl.in
	owner := reg.SkillOwner(in.HarnessRoot, name)
	if owner == "" {
		return fmt.Errorf("unknown skill %q (see `as-skill list skills`)", name)
	}
	m := reg.Domains[owner]
	if miss, gated := m.Gate(); gated {
		if miss.Kind == harness.MissEnv {
			return fmt.Errorf("skill %q belongs to domain %q, which requires $%s (not set)", name, owner, miss.Name)
		}
		return fmt.Errorf("skill %q belongs to domain %q, which requires the %q binary (not found on $PATH)", name, owner, miss.Name)
	}
	src := harness.SkillDir(in.HarnessRoot, owner, name)
	if err := pl.preflightRoots(map[lockfile.Kind]bool{lockfile.KindSkills: true}); err != nil {
		return err
	}
	sub, perr := pl.planSkill(m, owner, name, src)
	if sub == nil {
		sub = &Plan{}
	}
	for i := range sub.Ops {
		sub.Ops[i].unit, sub.Ops[i].unitKind = name, lockfile.KindSkill
	}
	if err := pl.addChecked(sub); err != nil {
		return err
	}
	return perr
}

// addChecked merges sub into the plan; when collecting it also runs the
// read-only dry check of sub's ops (what a dry Apply would hit), whose hard
// errors come back wrapped like Apply's.
func (pl *Planner) addChecked(sub *Plan) error {
	pl.plan.merge(sub)
	if !pl.collect {
		return nil
	}
	for _, op := range sub.Ops {
		if err := pl.checkOp(op); err != nil {
			return wrapApplyErr(&OpError{op, stagePlace, err})
		}
	}
	return nil
}

func (pl *Planner) checkOp(op Op) error {
	in := pl.in
	switch {
	case op.Item:
		return pl.checkItem(op)
	case op.Type == OpRender:
		if _, _, _, err := in.renderOp(op); err != nil {
			return err
		}
		if err := pl.guard(op.Dst); err != nil {
			return err
		}
		return pl.foreign(op.Dst)
	case op.Type == OpBlock || op.Type == OpHook:
		return nil
	}
	if len(op.Exclude) > 0 && op.Type == OpCopy {
		sub, err := in.planItems(op.Domain, op.Kind, op.Src, op.Dst, op.Platforms)
		if err != nil {
			return err
		}
		for _, s := range sub.Ops {
			if err := pl.checkItem(s); err != nil {
				return err
			}
		}
		return nil
	}
	if in.Mode == lockfile.ModeLink {
		if op.Kind == lockfile.KindSkill {
			return pl.checkLink(op.Src, op.Dst)
		}
		var skip func(string) bool
		if op.Kind == lockfile.KindSkills && in.hasExclusions(op.Domain) {
			skip = func(item string) bool { return in.excluded(op.Domain, op.Kind, item) }
		}
		_, err := fsys.LinkTreeWith(op.Kind.SkipsReadme(), op.Src, op.Dst, true, skip, pl.checkLink)
		return err
	}
	_, err := fsys.CopyTreeRoot(op.Kind.SkipsReadme(), op.Src, op.Dst, in.HarnessRoot, true)
	return err
}

func (pl *Planner) checkLink(src, dst string) error {
	in := pl.in
	_, err := fsys.LinkOne(src, dst, true, in.Force || in.overwrites(dst), in.tracked)
	var fe *fsys.ForeignError
	if errors.As(err, &fe) {
		return pl.add(Conflict{Path: fe.Path, Reason: ReasonForeign})
	}
	return err
}

func (pl *Planner) checkItem(op Op) error {
	in := pl.in
	if err := pl.guard(op.Dst); err != nil {
		return err
	}
	if in.Mode == lockfile.ModeLink {
		return pl.checkLink(op.Src, op.Dst)
	}
	if err := pl.foreign(op.Dst); err != nil {
		return err
	}
	fi, err := os.Stat(op.Src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		if _, err := fsys.CopyTreeRoot(op.Kind.SkipsReadme(), op.Src, op.Dst, in.HarnessRoot, true); err != nil {
			return err
		}
	}
	return nil
}

func sortConflicts(cs []Conflict) {
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].Reason != cs[j].Reason {
			return cs[i].Reason < cs[j].Reason
		}
		return cs[i].Path < cs[j].Path
	})
}

// wrapApplyErr keeps the error wording of the pre-plan installer.
func wrapApplyErr(err error) error {
	var oe *OpError
	if !errors.As(err, &oe) {
		return err
	}
	if oe.Op.unitKind == lockfile.KindSkill {
		return fmt.Errorf("skill %q: %w", oe.Op.unit, oe.Err)
	}
	return kindApplyErr(oe.Op.unit, oe.Op.unitKind, err)
}

// PlanInstall plans the selection without touching disk or printing. Recoverable
// refusals land in Plan.Conflicts (sorted, deduped); other failures come back as
// err. The Plan is never nil.
func (in *Installer) PlanInstall(reg *harness.Registry, domains []DomainReq, skills []string) (*Plan, error) {
	pl := in.newPlanner(true)
	err := pl.build(reg, domains, skills)
	sortConflicts(pl.plan.Conflicts)
	return pl.plan, err
}

// InstallAll plans the selection, refusing on the first conflict, and applies it.
func (in *Installer) InstallAll(reg *harness.Registry, domains []DomainReq, skills []string) error {
	pl := in.newPlanner(false)
	perr := pl.build(reg, domains, skills)
	if err := in.Apply(pl.plan); err != nil {
		return wrapApplyErr(err)
	}
	return perr
}

// PlanRoots plans only the git-ignore guard of the roots the kinds would touch.
func (in *Installer) PlanRoots(kinds ...lockfile.Kind) *Plan {
	pl := in.newPlanner(true)
	_ = pl.guardRoots(kinds)
	sortConflicts(pl.plan.Conflicts)
	return pl.plan
}

func (pl *Planner) guardRoots(kinds []lockfile.Kind) error {
	set := map[lockfile.Kind]bool{}
	for _, k := range kinds {
		set[k] = true
	}
	for _, r := range pl.in.touchedRoots(set) {
		if err := pl.guardRoot(r); err != nil {
			return err
		}
	}
	return nil
}

// GuardRoots returns the first not-ignored root the kinds would touch.
func (in *Installer) GuardRoots(kinds ...lockfile.Kind) error {
	return in.newPlanner(false).guardRoots(kinds)
}

// Preflight checks reserved names, git-ignore roots and untracked destinations
// for the domains, refusing on the first problem.
func (in *Installer) Preflight(reg *harness.Registry, domains []string) error {
	pl := in.newPlanner(false)
	err := pl.preflight(reg, domains)
	_ = in.Apply(pl.plan)
	return err
}
