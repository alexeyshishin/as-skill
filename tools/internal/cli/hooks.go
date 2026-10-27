package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"claude-harness/tools/internal/installer"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func runHooks(args []string) error {
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
	if len(positionals) == 0 {
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
	for _, p := range plats {
		if verb == "enable" {
			err = in.EnableHooks(p, positionals)
		} else {
			err = in.DisableHooks(p, positionals)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
