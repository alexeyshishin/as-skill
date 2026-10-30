package cli

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/installer"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/prompt"
)

func domainReqs(sel selection) []installer.DomainReq {
	reqs := make([]installer.DomainReq, len(sel.Domains))
	for i, d := range sel.Domains {
		reqs[i] = installer.DomainReq{Name: d.Name, Strict: d.Strict}
	}
	return reqs
}

func installAll(in *installer.Installer, reg *harness.Registry, sel selection) error {
	return in.InstallAll(reg, domainReqs(sel), sel.Skills)
}

func collectConflicts(in *installer.Installer, reg *harness.Registry, sel selection) ([]installer.ConflictError, error) {
	plan, err := in.PlanInstall(reg, domainReqs(sel), sel.Skills)
	return plan.Conflicts, err
}

func conflictsError(cs []installer.ConflictError) error {
	var lines []string
	var guard, foreign bool
	for _, c := range cs {
		lines = append(lines, c.Error())
		if c.Reason == installer.ReasonNotIgnored {
			guard = true
		} else {
			foreign = true
		}
	}
	var hints []string
	if guard {
		hints = append(hints, "pass -a claude to install for claude only, --copy to install a snapshot, or git-ignore the roots")
	}
	if foreign {
		hints = append(hints, "remove or move the untracked paths, or run the interactive dialog to overwrite them")
	}
	return fmt.Errorf("%s\nhint: %s (nothing written)", strings.Join(lines, "\n"), strings.Join(hints, "; "))
}

func (pl *addPlan) newInstaller(opts addOpts) *installer.Installer {
	return &installer.Installer{
		HarnessRoot: pl.harness, ProjectRoot: pl.project,
		DryRun: opts.dryRun, Force: opts.force, Mode: pl.mode, Lock: pl.lock, Platforms: pl.plats,
		ExcludeSkills: pl.exclude, Overwrite: pl.overwrite,
	}
}

func platformsTouching(root string, plats []platform.ID) []platform.ID {
	locs, _ := platform.SkillLocations(plats)
	var out []platform.ID
	for _, id := range plats {
		if id == platform.Claude {
			continue
		}
		p, _ := platform.Get(id)
		hit := root == "."+string(id) || root == platform.AgentsRoot ||
			root == platform.RootOf(p.Agents.Dir) || root == platform.RootOf(p.Rules.Dir) ||
			(p.Hooks.Schema != platform.HooksNone && root == platform.RootOf(p.Hooks.Dir))
		for loc, ids := range locs {
			if platform.RootOf(loc) == root && slices.Contains(ids, id) {
				hit = true
			}
		}
		if hit {
			out = append(out, id)
		}
	}
	return out
}

func pathList(project string, paths []string) string {
	lines := make([]string, len(paths))
	for i, p := range paths {
		lines[i] = p
		if rel, err := filepath.Rel(project, p); err == nil && !strings.HasPrefix(rel, "..") {
			lines[i] = rel
		}
	}
	return strings.Join(lines, "\n")
}

func resolveConflicts(p prompt.Prompter, opts addOpts, pl *addPlan, prev map[string]string) (bool, error) {
	asked, overwritten := false, false
	for {
		cs, err := collectConflicts(pl.newInstaller(opts), pl.reg, pl.sel)
		if err != nil {
			return asked, err
		}
		var roots, foreign []string
		for _, c := range cs {
			if c.Reason == installer.ReasonNotIgnored {
				roots = append(roots, c.Path)
			} else {
				foreign = append(foreign, filepath.Clean(c.Path))
			}
		}
		switch {
		case len(roots) > 0:
			asked = true
			if err := resolveGuard(p, pl, roots, prev); err != nil {
				return asked, err
			}
		case len(foreign) > 0:
			if overwritten {
				return asked, fmt.Errorf("still conflicting after overwrite: %s", strings.Join(foreign, ", "))
			}
			asked = true
			ok, err := p.Confirm("overwrite", fmt.Sprintf("Overwrite these %d paths?", len(foreign)), pathList(pl.project, foreign), false)
			if err != nil {
				return asked, abortErr(err)
			}
			if !ok {
				return asked, errAborted
			}
			pl.overwrite = map[string]bool{}
			for _, f := range foreign {
				pl.overwrite[f] = true
			}
			pl.adjust = append(pl.adjust, fmt.Sprintf("overwrite %d untracked path(s): %s", len(foreign), strings.ReplaceAll(pathList(pl.project, foreign), "\n", ", ")))
			overwritten = true
		default:
			return asked, nil
		}
	}
}

func resolveGuard(p prompt.Prompter, pl *addPlan, roots []string, prev map[string]string) error {
	var target platform.ID
	var mine []string
	for _, id := range pl.plats {
		var hit []string
		for _, r := range roots {
			if slices.Contains(platformsTouching(r, pl.plats), id) {
				hit = append(hit, r)
			}
		}
		if len(hit) > 0 {
			target, mine = id, hit
			break
		}
	}
	if target == "" {
		return fmt.Errorf("%s not git-ignored; add to .gitignore or use --copy", strings.Join(roots, ", "))
	}
	def := prev[string(target)]
	if def == "" {
		def = "drop"
		if len(pl.plats) == 1 {
			def = "copy"
		}
	}
	choice, err := p.Select("guard:"+string(target),
		fmt.Sprintf("%s: %s not git-ignored, symlinks there would be committed", target, strings.Join(mine, ", ")),
		[]prompt.Option{
			{Value: "drop", Label: "drop platform " + string(target), Selected: def == "drop"},
			{Value: "copy", Label: "switch ALL platforms to copy (one install mode per run — symlinks for the other platforms become copies too)", Selected: def == "copy"},
			{Value: "abort", Label: "abort", Selected: def == "abort"},
		}, def)
	if err != nil {
		return abortErr(err)
	}
	prev[string(target)] = choice
	switch choice {
	case "copy":
		pl.mode = lockfile.ModeCopy
		pl.adjust = append(pl.adjust, fmt.Sprintf("mode switched to copy for all platforms (%s not git-ignored)", strings.Join(mine, ", ")))
	case "drop":
		var left []platform.ID
		for _, id := range pl.plats {
			if id != target {
				left = append(left, id)
			}
		}
		if len(left) == 0 {
			return errAborted
		}
		pl.plats = left
		pl.adjust = append(pl.adjust, fmt.Sprintf("dropped platform %s (%s not git-ignored)", target, strings.Join(mine, ", ")))
	default:
		return errAborted
	}
	return nil
}
