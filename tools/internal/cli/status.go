package cli

import (
	"fmt"
	"path/filepath"
	"strings"

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
				return opts, err
			}
			opts.project = v
		case "--harness-root":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.harnessRoot = v
		case "--platform", "--agent", "-a":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.platforms = append(opts.platforms, v)
		default:
			return opts, fmt.Errorf("unknown flag %q", a)
		}
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
