package cli

import (
	"claude-harness/tools/internal/fsys"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/doctor"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func runDoctor(args []string, d Deps) error {
	ios := d.IO
	opts, err := parseStatusArgs(args)
	if err != nil {
		return err
	}
	harnessRoot, herr := d.resolveHarness(opts.harnessRoot)
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
	checked, failed := 0, 0
	for _, e := range entries {
		checked++
		h := doctor.EntryHealth(e)
		entryFailed := h == doctor.HealthMissing || h == doctor.HealthBroken
		if entryFailed {
			fmt.Fprintf(ios.Stdout(), "as-skill: doctor: %-11s %s/%s/%s — %s\n", h, e.Domain, e.Kind, e.Name, e.Path)
		}

		switch e.Mode {
		case lockfile.ModeLink:
			if herr == nil && !fsys.PathExists(e.Source) {
				fmt.Fprintf(ios.Stdout(), "as-skill: doctor: %-11s %s/%s/%s — source %s no longer exists under harness root %s\n",
					doctor.HealthSourceGone, e.Domain, e.Kind, e.Name, e.Source, harnessRoot)
				entryFailed = true
			}
		case lockfile.ModeRender:
			if h == doctor.HealthMissing {
				break
			}
			stale, drifted := doctor.RenderState(e)
			if stale {
				fmt.Fprintf(ios.Stdout(), "as-skill: doctor: %-11s %s/%s/%s — source %s changed since render; reinstall to refresh %s\n",
					doctor.HealthStale, e.Domain, e.Kind, e.Name, e.Source, e.Path)
				entryFailed = true
			}
			if drifted {
				fmt.Fprintf(ios.Stdout(), "as-skill: doctor: %-11s %s/%s/%s — %s was edited after render (generated file, not meant for hand edits)\n",
					doctor.HealthDrifted, e.Domain, e.Kind, e.Name, e.Path)
				entryFailed = true
			}
		case lockfile.ModeCopy:
			if h == doctor.HealthMissing {
				break
			}
			if e.SHA256 == "" {
				break
			}
			match, merr := doctor.CopyMatchesIn(lock, e)
			if errors.Is(merr, doctor.ErrCannotVerify) {
				fmt.Fprintf(ios.Stdout(), "as-skill: doctor: warning: %-13s %s/%s/%s — no recorded item list and the source checkout is gone; reinstall once with the current binary to fix\n",
					"CANNOT-VERIFY", e.Domain, e.Kind, e.Name)
				break
			}
			if merr != nil {
				fmt.Fprintf(ios.Stdout(), "as-skill: doctor: warning: %s/%s/%s — could not hash %s: %v\n", e.Domain, e.Kind, e.Name, e.Path, merr)
				break
			}
			if !match {
				fmt.Fprintf(ios.Stdout(), "as-skill: doctor: %-11s %s/%s/%s — recomputed hash differs from the recorded SHA256 (ambiguous: the target may have been hand-edited, or the source in domains/ may have moved on since install — not disambiguated)\n",
					doctor.HealthDrifted, e.Domain, e.Kind, e.Name)
				entryFailed = true
			}
		}
		if entryFailed {
			failed++
		}
	}

	untracked, err := doctor.FindUntracked(projectRoot, lock)
	if err != nil {
		return err
	}
	for _, p := range untracked {
		fmt.Fprintf(ios.Stdout(), "as-skill: doctor: %-11s %s — present on disk, no lockfile entry\n", doctor.HealthUntracked, p)
	}
	checked += len(untracked)
	failed += len(untracked)

	for _, d := range doctor.Duplicates(projectRoot, lock) {
		fmt.Fprintf(ios.Stdout(), "as-skill: doctor: warning: %-15s %s\n", doctor.DuplicateSkill, d)
	}
	for _, u := range doctor.UnverifiedCaps(lock) {
		fmt.Fprintf(ios.Stdout(), "as-skill: doctor: warning: %-15s %s\n", doctor.Unverified, u)
	}

	if codexHooksNeedTrust(projectRoot, lock) {
		fmt.Fprintf(ios.Stdout(), "as-skill: doctor: warning: %-15s codex hooks need trust: review in /hooks\n", "HOOKS-TRUST")
	}

	if herr == nil {
		lint := lintHarness(harnessRoot)
		for _, f := range lint.findings {
			fmt.Fprintf(ios.Stdout(), "as-skill: doctor: %-11s %s\n", "lint", f)
		}
		checked += lint.checked
		failed += lint.failed
	}

	fmt.Fprintf(ios.Stdout(), "doctor: %d checks, %d/%d ok, %d failed\n", checked, checked-failed, checked, failed)
	if failed > 0 {
		return fmt.Errorf("doctor: %d failed", failed)
	}
	return nil
}

func codexHooksNeedTrust(projectRoot string, lock *lockfile.Lockfile) bool {
	has := false
	for _, p := range doctor.LockPlatforms(lock) {
		if p == platform.Codex {
			has = true
		}
	}
	if !has {
		return false
	}
	spec, _ := platform.Get(platform.Codex)
	data, err := os.ReadFile(filepath.Join(projectRoot, spec.Hooks.ConfigFile))
	return err == nil && strings.Contains(string(data), "./"+spec.Hooks.Dir+"/")
}
