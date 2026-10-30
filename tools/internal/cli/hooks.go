package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/installer"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func runHooks(args []string, d Deps) error {
	if len(args) == 0 {
		return errors.New("hooks needs a verb: enable|disable")
	}
	verb := args[0]
	if verb != "enable" && verb != "disable" {
		return fmt.Errorf("unknown hooks verb %q (want enable|disable)", verb)
	}
	positionals, opts, err := parseInstallArgs(args[1:])
	if err != nil {
		return err
	}
	if len(positionals) == 0 && len(opts.domains) == 0 {
		return fmt.Errorf("hooks %s needs a hook name or \"all\"", verb)
	}
	plats, err := platform.ParsePlatforms(opts.platforms)
	if err != nil {
		return err
	}
	projectRoot, err := filepath.Abs(opts.project)
	if err != nil {
		return err
	}
	in := &installer.Installer{ProjectRoot: projectRoot, DryRun: opts.dryRun, Lock: &lockfile.Lockfile{}}
	if len(opts.domains) > 0 {
		return runHooksDomain(in, verb, positionals, opts, plats, d)
	}
	for _, p := range plats {
		if verb == "enable" {
			err = in.EnableHooks(p, positionals)
		} else {
			err = in.DisableHooks(p, positionals)
		}
		if err != nil {
			if opts.dryRun && verb == "enable" && errors.Is(err, os.ErrNotExist) {
				err = fmt.Errorf("%w (pass --domain <name> to preview from the harness source)", err)
			}
			return err
		}
	}
	return nil
}

func runHooksDomain(in *installer.Installer, verb string, names []string, opts installOpts, plats []platform.ID, d Deps) error {
	ios := d.IO
	root := ""
	if opts.harnessSet {
		root = opts.harnessRoot
	}
	hRoot, err := d.Locator.Locate(root)
	if err != nil {
		return err
	}
	reg, err := harness.LoadRegistry(hRoot)
	if err != nil {
		return err
	}
	for _, d := range opts.domains {
		if _, ok := reg.Domains[d]; !ok {
			return fmt.Errorf("unknown domain %q (domains: %s)", d, strings.Join(reg.DomainNames(), ", "))
		}
	}
	in.HarnessRoot = hRoot
	all := len(names) == 0
	for _, n := range names {
		if n == "all" {
			all = true
		}
	}
	for _, p := range plats {
		if spec, _ := platform.Get(p); spec.Hooks.Schema == platform.HooksNone {
			fmt.Fprintf(ios.Stdout(), "hooks unsupported on %s\n", p)
			continue
		}
		avail, err := in.DomainHookNames(reg, opts.domains, p)
		if err != nil {
			return err
		}
		sel := avail
		if !all {
			have := map[string]bool{}
			for _, a := range avail {
				have[a] = true
			}
			sel = nil
			for _, n := range names {
				if !strings.HasSuffix(n, ".sh") {
					n += ".sh"
				}
				if !have[n] {
					return fmt.Errorf("hook %q is not registerable on %s from domain(s) %s", n, p, strings.Join(opts.domains, ", "))
				}
				sel = append(sel, n)
			}
		}
		if len(sel) == 0 {
			fmt.Fprintf(ios.Stdout(), "no registerable hooks in %s for %s\n", strings.Join(opts.domains, ", "), p)
			continue
		}
		if verb == "enable" {
			err = in.RegisterHooks(reg, opts.domains, p, sel)
		} else {
			err = in.DisableHooks(p, sel)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
