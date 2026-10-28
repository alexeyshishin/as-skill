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
	if len(opts.names) == 0 || !(opts.yes || opts.dryRun) {
		return gateError(opts)
	}
	projectRoot, err := resolveProject(opts.project)
	if err != nil {
		return err
	}
	harnessRoot, err := registry.ResolveHarnessRootAuto(opts.harnessRoot)
	if err != nil {
		return err
	}
	reg, err := registry.LoadRegistry(harnessRoot)
	if err != nil {
		return err
	}
	sel, err := resolveNames(harnessRoot, reg, opts.names)
	if err != nil {
		return err
	}
	if len(sel.Domains) == 0 && len(sel.Skills) == 0 {
		return errors.New("nothing to add")
	}
	lock, err := lockfile.Load(projectRoot)
	if err != nil {
		return err
	}

	var plats []platform.ID
	detected := false
	if len(opts.agents) > 0 {
		plats, err = platform.ParsePlatforms(opts.agents)
		if err != nil {
			return err
		}
	} else {
		var src []string
		plats, src = detectPlatforms(projectRoot, lock)
		detected = len(src) > 0
		if detected {
			fmt.Printf("platforms: detected from %s\n", strings.Join(uniqStrings(src), ", "))
		} else {
			fmt.Println("platforms: default (claude)")
		}
	}
	mode := "link"
	if opts.copy {
		mode = "copy"
	}
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
			return fmt.Errorf("%w\n(platforms detected: %s) pass -a claude to install for claude only, or --copy to install a snapshot",
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
