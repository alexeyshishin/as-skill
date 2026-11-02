package cli

import (
	"fmt"
	"strings"

	"claude-harness/tools/internal/doctor"
	"claude-harness/tools/internal/lint"
	"claude-harness/tools/internal/lockfile"
)

func runDoctor(args []string, d Deps) error {
	ios := d.IO
	opts, err := parseStatusArgs(args)
	if err != nil {
		return err
	}
	harnessRoot, herr := d.Locator.Locate(opts.harnessRoot)
	if herr != nil && opts.harnessRoot != "" {
		return fmt.Errorf("doctor: --harness-root rejected: %w", herr)
	}
	projectRoot, err := resolveProject(opts.project)
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
	entries := doctor.FilterEntries(doctor.SortedEntries(lock), filter)
	doctor.PrintEntryTable(ios.Stdout(), entries, doctor.EntryHealth)
	if herr != nil {
		if opts.harnessRoot != "" {
			fmt.Fprintf(ios.Stdout(), "as-skill: doctor: note: harness checkout not found (--harness-root rejected: %v); skipping SOURCE-GONE and lint checks\n", herr)
		} else {
			fmt.Fprintln(ios.Stdout(), "as-skill: doctor: note: harness checkout not found; skipping SOURCE-GONE and lint checks")
		}
	}
	res, rerr := doctor.Run(doctor.Input{
		ProjectRoot: projectRoot,
		HarnessRoot: harnessRoot,
		HarnessOK:   herr == nil,
		Lock:        lock,
		Entries:     entries,
	})
	for _, f := range res.Findings {
		fmt.Fprintln(ios.Stdout(), renderFinding(f))
	}
	if rerr != nil {
		return rerr
	}
	checked, failed := res.Checked, res.Failed

	if herr == nil {
		res := lint.Harness(harnessRoot)
		for _, f := range res.Findings {
			fmt.Fprintf(ios.Stdout(), "as-skill: doctor: %-11s %s\n", "lint", f)
		}
		checked += res.Checked
		failed += res.Failed
	}

	fmt.Fprintf(ios.Stdout(), "doctor: %d checks, %d/%d ok, %d failed\n", checked, checked-failed, checked, failed)
	if failed > 0 {
		return fmt.Errorf("doctor: %d failed", failed)
	}
	return nil
}

func renderFinding(f doctor.Finding) string {
	width := 11
	prefix := "as-skill: doctor: "
	if f.Severity == doctor.SeverityWarning {
		prefix += "warning: "
		width = 15
		if f.Code == doctor.CannotVerify {
			width = 13
		}
	}
	var b strings.Builder
	b.WriteString(prefix)
	if f.Code != "" {
		fmt.Fprintf(&b, "%-*s ", width, f.Code)
	}
	if f.Entry == "" {
		b.WriteString(f.Msg)
		return b.String()
	}
	b.WriteString(f.Entry)
	b.WriteString(" — ")
	b.WriteString(f.Msg)
	return b.String()
}
