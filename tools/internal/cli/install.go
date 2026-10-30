package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/install"
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
	var domains []string
	fs := newFlagSet()
	fs.StringVarP(&opts.project, "project", "p", ".", "")
	fs.StringVar(&opts.harnessRoot, "harness-root", ".", "")
	fs.VarP(listValue{&opts.platforms}, "agent", "a", "")
	hiddenAlias(fs, "platform", &opts.platforms)
	fs.VarP(listValue{&domains}, "domain", "d", "")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "")
	fs.BoolVar(&opts.force, "force", false, "")
	fs.BoolVar(&opts.copy, "copy", false, "")
	positionals, err := parseFlags(fs, args)
	if err != nil {
		return nil, opts, err
	}
	opts.harnessSet = fs.Changed("harness-root")
	for _, v := range domains {
		for _, d := range strings.Split(v, ",") {
			if d = strings.TrimSpace(d); d != "" {
				opts.domains = append(opts.domains, d)
			}
		}
	}
	return positionals, opts, nil
}

func runInstall(args []string, d Deps) error {
	ios := d.IO
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
	mode := lockfile.ModeLink
	if opts.copy {
		mode = lockfile.ModeCopy
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
	in := &install.Installer{
		HarnessRoot: harnessRoot, ProjectRoot: projectRoot,
		DryRun: opts.dryRun, Force: opts.force, Mode: mode, Lock: lock, Platforms: plats,
	}

	var runErr error
	switch verb {
	case "domain":
		if len(positionals) != 1 {
			return errors.New("install domain needs exactly one domain name")
		}
		runErr = installDomains(in, reg, positionals, true)
	case "domains":
		if len(positionals) == 0 {
			return errors.New("install domains needs at least one domain name")
		}
		runErr = installDomains(in, reg, positionals, true)
	case "all":
		if len(positionals) != 0 {
			return fmt.Errorf("install all takes no domain names (got %s)", strings.Join(positionals, ", "))
		}
		runErr = installDomains(in, reg, reg.DomainNames(), false)
	case "skill":
		if len(positionals) != 1 {
			return errors.New("install skill needs exactly one skill name")
		}
		runErr = in.InstallAll(reg, nil, positionals[:1])
	default:
		return fmt.Errorf("unknown install verb %q (want domain|domains|all|skill)", verb)
	}

	var pre *install.PreflightError
	if errors.As(runErr, &pre) {
		return runErr
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
	fmt.Fprintf(ios.Stdout(), "done -> %s\n", filepath.Join(projectRoot, platform.ClaudeHome))
	return nil
}

func installDomains(in *install.Installer, reg *harness.Registry, names []string, strict bool) error {
	reqs := make([]install.DomainReq, len(names))
	for k, name := range names {
		reqs[k] = install.DomainReq{Name: name, Strict: strict}
	}
	return in.InstallAll(reg, reqs, nil)
}
