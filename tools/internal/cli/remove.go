package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/install"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/prompt"
)

func recordedPlatforms(rows []lockfile.Entry) []platform.ID {
	seen := map[string]bool{}
	for _, e := range rows {
		for _, p := range e.Platforms {
			seen[p] = true
		}
	}
	var out []platform.ID
	for _, id := range platform.IDs() {
		if seen[string(id)] {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return []platform.ID{platform.Claude}
	}
	return out
}

func runRemove(args []string, d Deps) error {
	ios := d.IO
	opts, err := parseAddArgs("remove", args)
	if err != nil {
		return err
	}
	if opts.help {
		printUsage(ios.Stdout())
		return nil
	}
	dlg := wantDialog(d, opts)
	if !dlg && (len(opts.names) == 0 || !(opts.yes || opts.dryRun)) {
		return gateError(opts)
	}
	var pr prompt.Prompter
	if dlg {
		pr = d.newPrompter()
	}
	var (
		projectRoot, harnessRoot string
		prevProj                 string
		prevNames                []string
		prevPlats                []platform.ID
		lock                     *lockfile.Lockfile
		targets                  []install.RemoveTarget
		reg                      *harness.Registry
		plats                    []platform.ID
	)
	nameArgs := len(opts.names) > 0
	names := opts.names
	err = runSteps([]func() (bool, error){
		func() (bool, error) {
			var err error
			if !dlg {
				projectRoot, err = resolveProject(opts.project)
				return false, err
			}
			projectRoot, err = askProject(pr, opts, prevProj)
			if err == nil {
				prevProj = projectRoot
			}
			return !opts.projectSet, err
		},
		func() (bool, error) {
			var err error
			lock, err = lockfile.Load(projectRoot)
			return false, err
		},
		func() (bool, error) {
			if !dlg || nameArgs {
				return false, nil
			}
			var err error
			if names, err = askRemoveNames(pr, lock, projectRoot, prevNames); err != nil {
				return true, err
			}
			prevNames = names
			return true, nil
		},
		func() (bool, error) {
			var err error
			targets, err = install.ResolveRemoveTargets(lock, projectRoot, names)
			if err != nil {
				return false, err
			}
			var herr error
			harnessRoot, herr = d.resolveHarness(opts.harnessRoot)
			reg = nil
			if herr == nil {
				r, rerr := harness.LoadRegistry(harnessRoot)
				if rerr == nil {
					reg = r
				} else if opts.harnessRoot != "" {
					return false, fmt.Errorf("remove: --harness-root rejected: %w", rerr)
				}
			} else if opts.harnessRoot != "" {
				return false, fmt.Errorf("remove: --harness-root rejected: %w", herr)
			}
			return false, nil
		},
		func() (bool, error) {
			var allRows []lockfile.Entry
			for _, t := range targets {
				allRows = append(allRows, t.Rows...)
				allRows = append(allRows, t.Cover...)
			}
			if len(opts.agents) > 0 {
				var err error
				plats, err = platform.ParsePlatforms(opts.agents)
				return false, err
			}
			plats = recordedPlatforms(allRows)
			if dlg && len(plats) > 1 {
				var err error
				if plats, err = askRemovePlatforms(pr, plats, prevPlats); err != nil {
					return true, err
				}
				prevPlats = plats
				return true, nil
			}
			return false, nil
		},
		func() (bool, error) {
			if dlg && !opts.dryRun {
				return true, confirm(pr, "Remove?", removeSummary(projectRoot, names, plats))
			}
			return false, nil
		},
	})
	if err != nil {
		return err
	}
	opts.names = names
	if reg == nil {
		fmt.Fprintln(ios.Stderr(), "as-skill: note: harness checkout not found; removing files and lockfile rows only (hook registrations and AGENTS.md blocks are left in place)")
	}
	fmt.Fprintf(ios.Stdout(), "platforms: %s\n", strings.Join(platformDirNames(plats), ", "))

	in := &install.Installer{
		HarnessRoot: harnessRoot, ProjectRoot: projectRoot,
		DryRun: opts.dryRun, Lock: lock, Platforms: plats,
		Reporter: install.Reporter{IO: ios},
	}
	plan, err := in.RemovePlan(reg, targets)
	if err != nil {
		return err
	}
	runErr := in.Apply(plan)
	if !opts.dryRun {
		if err := lock.Save(projectRoot); err != nil {
			if runErr != nil {
				return runErr
			}
			return fmt.Errorf("saving lockfile: %w", err)
		}
	}
	if runErr != nil {
		return runErr
	}
	if err := plan.Problems(); err != nil {
		return err
	}
	for _, id := range plats {
		fmt.Fprintf(ios.Stdout(), "done -> %s\n", filepath.Join(projectRoot, "."+string(id)))
	}
	return nil
}
