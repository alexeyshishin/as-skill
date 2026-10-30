package installer

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

// merge appends sub after p, shifting its warning positions.
func (p *Plan) merge(sub *Plan) {
	if sub == nil {
		return
	}
	off := len(p.Ops)
	p.Ops = append(p.Ops, sub.Ops...)
	p.Conflicts = append(p.Conflicts, sub.Conflicts...)
	for _, w := range sub.Warnings {
		w.Before += off
		p.Warnings = append(p.Warnings, w)
	}
}

// row queues a stdout line printed after the ops planned so far.
func (p *Plan) row(text string) {
	p.Warnings = append(p.Warnings, Warning{Text: text, Stream: Stdout, Before: len(p.Ops)})
}

func (in *Installer) itemOp(domain string, kind lockfile.Kind, name, src, dst string, plats []platform.ID) Op {
	t := OpCopy
	if in.Mode == lockfile.ModeLink {
		t = OpLink
	}
	return Op{Type: t, Src: src, Dst: dst, Mode: in.Mode, Domain: domain, Kind: kind, Name: name, Platforms: plats, Item: true}
}

// planDup queues the duplicate-skills warning, once per Plan.
func (pl *Planner) planDup(p *Plan, locs map[string][]platform.ID) {
	dup := pl.in.dupPlatforms(locs)
	if len(dup) == 0 || pl.dupNoted {
		return
	}
	pl.dupNoted = true
	p.Warnings = append(p.Warnings, Warning{
		Text:   fmt.Sprintf("as-skill: warning: skills land in both .claude/skills and .agents/skills; %s will see each skill twice (doctor reports DUPLICATE-SKILL)", strings.Join(strs(dup), ", ")),
		Stream: Stderr, Before: len(p.Ops),
	})
}

// planGaps queues the capability-gap notes, once per Plan.
func (pl *Planner) planGaps() *Plan {
	p := &Plan{}
	if pl.gapsNoted {
		return p
	}
	pl.gapsNoted = true
	for _, id := range pl.in.plats() {
		if g := platform.CapGap(id); g != "" {
			p.Warnings = append(p.Warnings, Warning{Text: fmt.Sprintf("as-skill: note: %s: %s", id, g), Stream: Stderr})
		}
	}
	return p
}

// planItems plans each wanted child of srcDir at dstDir plus the summary row.
func (in *Installer) planItems(domain string, kind lockfile.Kind, srcDir, dstDir string, plats []platform.ID) (*Plan, error) {
	p := &Plan{}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return p, err
	}
	n := 0
	for _, e := range entries {
		if skipItem(in, domain, kind, srcDir, e.Name(), plats) {
			continue
		}
		p.Ops = append(p.Ops, in.itemOp(domain, kind, e.Name(), filepath.Join(srcDir, e.Name()), filepath.Join(dstDir, e.Name()), plats))
		n++
	}
	p.row(in.reportText(domain, string(kind), dstDir, n))
	return p, nil
}

func (in *Installer) planAgents(domain, srcDir string, pf platform.Platform) (*Plan, error) {
	p := &Plan{}
	entries, err := in.mdItems(srcDir)
	if err != nil {
		return p, err
	}
	plats := []platform.ID{pf.ID}
	n := 0
	for _, e := range entries {
		src := filepath.Join(srcDir, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			return p, err
		}
		agentName := agentNameOf(data, strings.TrimSuffix(e.Name(), ".md"))
		if err := reservedErr(pf, agentName, e.Name()); err != nil {
			return p, err
		}
		if in.Mode == lockfile.ModeLink {
			p.Ops = append(p.Ops, in.itemOp(domain, lockfile.KindPrompts, e.Name(), src, filepath.Join(in.ProjectRoot, platform.PromptsDir, e.Name()), plats))
		}
		p.Ops = append(p.Ops, Op{
			Type: OpRender, Src: src, Dst: filepath.Join(in.ProjectRoot, pf.Agents.Dir, agentName+pf.Agents.Ext),
			Mode: lockfile.ModeRender, Domain: domain, Kind: lockfile.KindAgents, Name: e.Name(), Platforms: plats,
		})
		n++
	}
	p.row(in.reportText(domain, string(lockfile.KindAgents), filepath.Join(in.ProjectRoot, pf.Agents.Dir), n))
	return p, nil
}

func (in *Installer) planRules(domain, srcDir string, pf platform.Platform) (*Plan, error) {
	p := &Plan{}
	entries, err := in.mdItems(srcDir)
	if err != nil {
		return p, err
	}
	plats := []platform.ID{pf.ID}
	n := 0
	for _, e := range entries {
		stem := strings.TrimSuffix(e.Name(), ".md")
		src := filepath.Join(srcDir, e.Name())
		switch pf.Rules.Mode {
		case platform.RulesDirMDC:
			if _, err := os.ReadFile(src); err != nil {
				return p, err
			}
			if in.Mode == lockfile.ModeLink {
				p.Ops = append(p.Ops, in.itemOp(domain, lockfile.KindRules, e.Name(), src, filepath.Join(in.ProjectRoot, platform.AgentsRulesDir, e.Name()), plats))
			}
			p.Ops = append(p.Ops, Op{
				Type: OpRender, Src: src, Dst: filepath.Join(in.ProjectRoot, pf.Rules.Dir, stem+".mdc"),
				Mode: lockfile.ModeRender, Domain: domain, Kind: lockfile.KindRules, Name: e.Name(), Platforms: plats,
			})
		default:
			p.Ops = append(p.Ops, in.itemOp(domain, lockfile.KindRules, e.Name(), src, filepath.Join(in.ProjectRoot, pf.Rules.Dir, e.Name()), plats))
		}
		n++
	}
	p.row(in.reportText(domain, string(lockfile.KindRules), filepath.Join(in.ProjectRoot, pf.Rules.Dir), n))
	if pf.Rules.Mode == platform.RulesAgentsMDBlock {
		p.Ops = append(p.Ops, Op{
			Type: OpBlock, Dst: filepath.Join(in.ProjectRoot, agentsMD), Mode: lockfile.ModeBlock,
			Domain: "as-skill", Kind: lockfile.KindInstructions, Name: agentsMD, Platforms: plats,
		})
	}
	return p, nil
}

func (in *Installer) planHooks(domain, srcDir string, pf platform.Platform) (*Plan, error) {
	p := &Plan{}
	if pf.Hooks.Schema == platform.HooksNone {
		p.Warnings = append(p.Warnings, Warning{Text: fmt.Sprintf("as-skill: warning: hooks unsupported on %s", pf.ID), Stream: Stderr})
		return p, nil
	}
	dst := filepath.Join(in.ProjectRoot, pf.Hooks.Dir)
	p, err := in.planItems(domain, lockfile.KindHooks, srcDir, dst, []platform.ID{pf.ID})
	if err != nil {
		return p, err
	}
	p.Ops = append(p.Ops, Op{
		Type: OpHook, Src: srcDir, Dst: dst, Mode: in.Mode,
		Domain: domain, Kind: lockfile.KindHooks, Platforms: []platform.ID{pf.ID},
	})
	if in.Mode != lockfile.ModeCopy {
		for _, t := range hookExecWarnings(srcDir) {
			p.Warnings = append(p.Warnings, Warning{Text: t, Stream: Stderr, Before: len(p.Ops), WriteOnly: true})
		}
	}
	p.Warnings = append(p.Warnings, Warning{
		Text:   fmt.Sprintf("as-skill: warning: hooks on %s: files placed; run `as-skill hooks enable` to register (UNVERIFIED adapters)", pf.ID),
		Stream: Stderr, Before: len(p.Ops),
	})
	return p, nil
}

// planKind plans one domain kind across all selected platforms. On error the
// returned Plan holds everything planned before the failure; the caller applies
// it first.
func (pl *Planner) planKind(m *harness.Manifest, name string, kind lockfile.Kind, srcDir string) (*Plan, error) {
	in := pl.in
	p := &Plan{}
	if kind == lockfile.KindSkills {
		locs, _ := platform.SkillLocations(in.plats())
		pl.planDup(p, locs)
		for _, loc := range slices.Sorted(maps.Keys(locs)) {
			if loc == platform.ClaudeSkillsLoc {
				sub, err := in.planClaudeKind(m, name, kind, srcDir, locs[loc])
				if err != nil {
					return p, err
				}
				p.merge(sub)
				continue
			}
			sub, err := in.planItems(name, kind, srcDir, filepath.Join(in.ProjectRoot, loc), locs[loc])
			p.merge(sub)
			if err != nil {
				return p, fmt.Errorf("domain %q: %s: %w", name, kind, err)
			}
		}
		return p, nil
	}
	if in.hasClaude() {
		sub, err := in.planClaudeKind(m, name, kind, srcDir, []platform.ID{platform.Claude})
		if err != nil {
			return p, err
		}
		p.merge(sub)
	}
	for _, id := range in.plats() {
		if id == platform.Claude {
			continue
		}
		pf, _ := platform.Get(id)
		var sub *Plan
		var err error
		switch kind {
		case lockfile.KindAgents:
			sub, err = in.planAgents(name, srcDir, pf)
		case lockfile.KindRules:
			sub, err = in.planRules(name, srcDir, pf)
		case lockfile.KindHooks:
			sub, err = in.planHooks(name, srcDir, pf)
		}
		p.merge(sub)
		if err != nil {
			return p, fmt.Errorf("domain %q: %s on %s: %w", name, kind, id, err)
		}
	}
	return p, nil
}

// planSkill plans one skill across all selected skill locations.
func (pl *Planner) planSkill(m *harness.Manifest, owner, name, src string) (*Plan, error) {
	in := pl.in
	p := &Plan{}
	locs, _ := platform.SkillLocations(in.plats())
	pl.planDup(p, locs)
	for _, loc := range slices.Sorted(maps.Keys(locs)) {
		if loc == platform.ClaudeSkillsLoc {
			sub, err := in.planClaudeSkill(m, owner, name, src, locs[loc])
			if err != nil {
				return p, err
			}
			p.merge(sub)
			continue
		}
		dest := filepath.Join(in.ProjectRoot, loc, name)
		p.Ops = append(p.Ops, in.itemOp(owner, lockfile.KindSkill, name, src, dest, locs[loc]))
		p.row(in.reportText(owner, string(lockfile.KindSkill), dest, 1))
	}
	return p, nil
}
