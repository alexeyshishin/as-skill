package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/installer"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

type installOpts struct {
	project     string
	harnessRoot string
	dryRun      bool
	force       bool
	copy        bool
	platforms   []string
	domains     []string
	harnessSet  bool
}

func parseInstallArgs(args []string) ([]string, installOpts, error) {
	opts := installOpts{project: ".", harnessRoot: "."}
	var positionals []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasVal := a, "", false
		if strings.HasPrefix(a, "--") {
			if before, after, found := strings.Cut(a, "="); found {
				key, val, hasVal = before, after, true
			}
		}
		takeValue := func() (string, error) {
			if hasVal {
				return val, nil
			}
			i++
			if i >= len(args) {
				return "", fmt.Errorf("%s needs a value", key)
			}
			return args[i], nil
		}
		switch key {
		case "--project", "-p":
			v, err := takeValue()
			if err != nil {
				return nil, opts, err
			}
			opts.project = v
		case "--harness-root":
			v, err := takeValue()
			if err != nil {
				return nil, opts, err
			}
			opts.harnessRoot, opts.harnessSet = v, true
		case "--platform", "--agent", "-a":
			v, err := takeValue()
			if err != nil {
				return nil, opts, err
			}
			opts.platforms = append(opts.platforms, v)
		case "--domain", "-d":
			v, err := takeValue()
			if err != nil {
				return nil, opts, err
			}
			for _, d := range strings.Split(v, ",") {
				if d = strings.TrimSpace(d); d != "" {
					opts.domains = append(opts.domains, d)
				}
			}
		case "--dry-run":
			opts.dryRun = true
		case "--force":
			opts.force = true
		case "--copy":
			opts.copy = true
		default:
			if strings.HasPrefix(a, "-") {
				return nil, opts, fmt.Errorf("unknown flag %q", a)
			}
			positionals = append(positionals, a)
		}
	}
	return positionals, opts, nil
}

func runInstall(args []string) error {
	if len(args) == 0 {
		return errors.New("install needs a verb: domain|domains|all|skill")
	}
	verb := args[0]
	positionals, opts, err := parseInstallArgs(args[1:])
	if err != nil {
		return err
	}
	plats, err := platform.ParsePlatforms(opts.platforms)
	if err != nil {
		return err
	}
	mode := "link"
	if opts.copy {
		mode = "copy"
	}

	harnessRoot, err := harness.ResolveHarnessRoot(opts.harnessRoot)
	if err != nil {
		return err
	}
	projectRoot, err := filepath.Abs(opts.project)
	if err != nil {
		return err
	}
	reg, err := harness.LoadRegistry(harnessRoot)
	if err != nil {
		return err
	}
	lock, err := lockfile.Load(projectRoot)
	if err != nil {
		return err
	}
	in := &installer.Installer{
		HarnessRoot: harnessRoot, ProjectRoot: projectRoot,
		DryRun: opts.dryRun, Force: opts.force, Mode: mode, Lock: lock, Platforms: plats,
	}

	var runErr error
	switch verb {
	case "domain":
		if len(positionals) != 1 {
			return errors.New("install domain needs exactly one domain name")
		}
		if err := in.Preflight(reg, positionals); err != nil {
			return err
		}
		runErr = installDomains(in, reg, positionals, true)
	case "domains":
		if len(positionals) == 0 {
			return errors.New("install domains needs at least one domain name")
		}
		if err := in.Preflight(reg, positionals); err != nil {
			return err
		}
		runErr = installDomains(in, reg, positionals, true)
	case "all":
		if len(positionals) != 0 {
			return fmt.Errorf("install all takes no domain names (got %s)", strings.Join(positionals, ", "))
		}
		if err := in.Preflight(reg, reg.DomainNames()); err != nil {
			return err
		}
		runErr = installDomains(in, reg, reg.DomainNames(), false)
	case "skill":
		if len(positionals) != 1 {
			return errors.New("install skill needs exactly one skill name")
		}
		runErr = in.InstallSkill(reg, positionals[0])
	default:
		return fmt.Errorf("unknown install verb %q (want domain|domains|all|skill)", verb)
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
	fmt.Printf("done -> %s\n", filepath.Join(projectRoot, ".claude"))
	return nil
}

func installDomains(in *installer.Installer, reg *harness.Registry, names []string, strict bool) error {
	for _, name := range names {
		if err := in.InstallDomain(reg, name, strict); err != nil {
			return err
		}
	}
	return nil
}
