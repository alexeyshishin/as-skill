package cli

import (
	"fmt"
	"path/filepath"

	"claude-harness/tools/internal/doctor"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

type statusOpts struct {
	project     string
	harnessRoot string
	platforms   []string
}

func parseStatusArgs(args []string) (statusOpts, error) {
	opts := statusOpts{project: "."}
	fs := newFlagSet()
	fs.StringVarP(&opts.project, "project", "p", ".", "")
	fs.StringVar(&opts.harnessRoot, "harness-root", "", "")
	fs.VarP(listValue{&opts.platforms}, "agent", "a", "")
	hiddenAlias(fs, "platform", &opts.platforms)
	positionals, err := parseFlags(fs, args)
	if err != nil {
		return opts, err
	}
	if len(positionals) > 0 {
		return opts, fmt.Errorf("unknown flag %q", positionals[0])
	}
	return opts, nil
}

func runStatus(args []string, d Deps) error {
	ios := d.IO
	opts, err := parseStatusArgs(args)
	if err != nil {
		return err
	}
	projectRoot, err := filepath.Abs(opts.project)
	if err != nil {
		return err
	}
	lock, err := lockfile.Load(projectRoot)
	if err != nil {
		return err
	}
	filter, err := platformFilter(opts.platforms)
	if err != nil {
		return err
	}
	doctor.PrintEntryTable(ios.Stdout(), doctor.FilterEntries(doctor.SortedEntries(lock), filter), doctor.EntryHealth)
	return nil
}

func platformFilter(vals []string) ([]string, error) {
	if len(vals) == 0 {
		return nil, nil
	}
	ids, err := platform.ParsePlatforms(vals)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out, nil
}
