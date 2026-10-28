package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/installer"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/registry"
)

func runAdd(args []string) error {
	opts, err := parseAddArgs("add", args)
	if err != nil {
		return err
	}
	if opts.help {
		printUsage(os.Stdout)
		return nil
	}
	dlg := wantDialog(opts)
	if !dlg && (len(opts.names) == 0 || !(opts.yes || opts.dryRun)) {
		return gateError(opts)
	}
	var pl *addPlan
	if dlg {
		pl, err = addDialog(opts)
	} else {
		pl, err = planNonInteractive(opts)
	}
	if err != nil {
		return err
	}
	projectRoot, harnessRoot, reg, sel, lock, plats, mode := pl.project, pl.harness, pl.reg, pl.sel, pl.lock, pl.plats, pl.mode
	detected := pl.detected || pl.dialog
	in := &installer.Installer{
		HarnessRoot: harnessRoot, ProjectRoot: projectRoot,
		DryRun: opts.dryRun, Force: opts.force, Mode: mode, Lock: lock, Platforms: plats,
	}

	if detected && mode == "link" {
		kinds := []string{"skills", "agents", "rules", "hooks"}
		if len(sel.Domains) == 0 {
			kinds = []string{"skills"}
		}
		if gerr := in.GuardRoots(kinds...); gerr != nil {
			return fmt.Errorf("%w\n(platforms: %s) pass -a claude to install for claude only, or --copy to install a snapshot",
				gerr, strings.Join(platformDirNames(plats), ", "))
		}
	}

	if len(sel.Domains) > 0 {
		if err := in.Preflight(reg, sel.domainNames()); err != nil {
			return err
		}
	}
	var runErr error
	for _, d := range sel.Domains {
		if runErr = in.InstallDomain(reg, d.Name, d.Strict); runErr != nil {
			break
		}
	}
	if runErr == nil {
		for _, s := range sel.Skills {
			if runErr = in.InstallSkill(reg, s); runErr != nil {
				break
			}
		}
	}
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
	for _, id := range plats {
		fmt.Printf("done -> %s\n", filepath.Join(projectRoot, "."+string(id)))
	}
	return nil
}

func planNonInteractive(opts addOpts) (*addPlan, error) {
	pl := &addPlan{mode: "link"}
	var err error
	if pl.project, err = resolveProject(opts.project); err != nil {
		return nil, err
	}
	if pl.harness, err = registry.ResolveHarnessRootAuto(opts.harnessRoot); err != nil {
		return nil, err
	}
	if pl.reg, err = registry.LoadRegistry(pl.harness); err != nil {
		return nil, err
	}
	if pl.sel, err = resolveNames(pl.harness, pl.reg, opts.names); err != nil {
		return nil, err
	}
	if len(pl.sel.Domains) == 0 && len(pl.sel.Skills) == 0 {
		return nil, errors.New("nothing to add")
	}
	if pl.lock, err = lockfile.Load(pl.project); err != nil {
		return nil, err
	}
	if len(opts.agents) > 0 {
		if pl.plats, err = platform.ParsePlatforms(opts.agents); err != nil {
			return nil, err
		}
	} else {
		var src []string
		pl.plats, src = detectPlatforms(pl.project, pl.lock)
		pl.detected = len(src) > 0
		if pl.detected {
			fmt.Printf("platforms: detected from %s\n", strings.Join(uniqStrings(src), ", "))
		} else {
			fmt.Println("platforms: default (claude)")
		}
	}
	if opts.copy {
		pl.mode = "copy"
	}
	return pl, nil
}
