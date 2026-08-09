package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/installer"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/registry"
)

func runUninstall(args []string) error {
	if len(args) == 0 {
		return errors.New("uninstall needs a verb: domain|domains|all|skill")
	}
	verb := args[0]
	positionals, opts, err := parseInstallArgs(args[1:])
	if err != nil {
		return err
	}

	harnessRoot, err := registry.ResolveHarnessRoot(opts.harnessRoot)
	if err != nil {
		return err
	}
	projectRoot, err := filepath.Abs(opts.project)
	if err != nil {
		return err
	}
	reg, err := registry.LoadRegistry(harnessRoot)
	if err != nil {
		return err
	}
	lock, err := lockfile.Load(projectRoot)
	if err != nil {
		return err
	}
	in := &installer.Installer{
		HarnessRoot: harnessRoot, ProjectRoot: projectRoot,
		DryRun: opts.dryRun, Lock: lock,
	}

	switch verb {
	case "domain":
		if len(positionals) != 1 {
			return errors.New("uninstall domain needs exactly one domain name")
		}
		if err := in.UninstallDomain(reg, positionals[0]); err != nil {
			return err
		}
		if opts.withCore {
			in.UninstallCoreSkills(reg)
		}
	case "domains":
		if len(positionals) == 0 {
			return errors.New("uninstall domains needs at least one domain name")
		}
		for _, name := range positionals {
			if err := in.UninstallDomain(reg, name); err != nil {
				return err
			}
		}
		if opts.withCore {
			in.UninstallCoreSkills(reg)
		}
	case "all":
		if len(positionals) != 0 {
			return fmt.Errorf("uninstall all takes no domain names (got %s)", strings.Join(positionals, ", "))
		}
		for _, name := range reg.DomainNames() {
			if err := in.UninstallDomain(reg, name); err != nil {
				return err
			}
		}
		in.UninstallCoreSkills(reg)
	case "skill":
		if len(positionals) != 1 {
			return errors.New("uninstall skill needs exactly one skill name")
		}
		if err := in.UninstallSkill(reg, positionals[0]); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown uninstall verb %q (want domain|domains|all|skill)", verb)
	}

	fmt.Printf("done -> %s\n", filepath.Join(projectRoot, ".claude"))
	if !opts.dryRun {
		if err := lock.Save(projectRoot); err != nil {
			return fmt.Errorf("saving lockfile: %w", err)
		}
	}
	return nil
}
