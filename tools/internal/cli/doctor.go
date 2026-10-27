package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/fsutil"
	"claude-harness/tools/internal/health"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/registry"
	"claude-harness/tools/internal/transfer"
)

func runDoctor(args []string) error {
	opts, err := parseStatusArgs(args)
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
	lock, err := lockfile.Load(projectRoot)
	if err != nil {
		return err
	}

	filter, err := platformFilter(opts.platforms)
	if err != nil {
		return err
	}
	entries := health.FilterEntries(health.SortedEntries(lock), filter)
	checked, failed := 0, 0
	headerShown := false
	for _, e := range entries {
		checked++
		h := health.EntryHealth(e)
		entryFailed := h == health.HealthMissing || h == health.HealthBroken
		if entryFailed {
			if !headerShown {
				fmt.Printf("%-10s %-8s %-24s %-6s %-11s %s\n", "DOMAIN", "KIND", "NAME", "MODE", "HEALTH", "PATH")
				headerShown = true
			}
			fmt.Printf("%-10s %-8s %-24s %-6s %-11s %s\n", e.Domain, e.Kind, e.Name, e.Mode, h, e.Path)
		}

		switch e.Mode {
		case "link":
			if !fsutil.PathExists(e.Source) {
				fmt.Printf("as-skill: doctor: %-11s %s/%s/%s — source %s no longer exists under harness root %s\n",
					health.HealthSourceGone, e.Domain, e.Kind, e.Name, e.Source, harnessRoot)
				entryFailed = true
			}
		case "render":
			if h == health.HealthMissing {
				break
			}
			stale, drifted := health.RenderState(e)
			if stale {
				fmt.Printf("as-skill: doctor: %-11s %s/%s/%s — source %s changed since render; reinstall to refresh %s\n",
					health.HealthStale, e.Domain, e.Kind, e.Name, e.Source, e.Path)
				entryFailed = true
			}
			if drifted {
				fmt.Printf("as-skill: doctor: %-11s %s/%s/%s — %s was edited after render (generated file, not meant for hand edits)\n",
					health.HealthDrifted, e.Domain, e.Kind, e.Name, e.Path)
				entryFailed = true
			}
		case "copy":
			if h == health.HealthMissing {
				break
			}
			hash, herr := transfer.HashPath(e.Path)
			if herr != nil {
				fmt.Printf("as-skill: doctor: warning: %s/%s/%s — could not hash %s: %v\n", e.Domain, e.Kind, e.Name, e.Path, herr)
				break
			}
			if e.SHA256 != "" && hash != e.SHA256 {
				fmt.Printf("as-skill: doctor: %-11s %s/%s/%s — recomputed hash differs from the recorded SHA256 (ambiguous: the target may have been hand-edited, or the source in domains/ may have moved on since install — not disambiguated)\n",
					health.HealthDrifted, e.Domain, e.Kind, e.Name)
				entryFailed = true
			}
		}
		if entryFailed {
			failed++
		}
	}

	untracked, err := health.FindUntracked(projectRoot, lock)
	if err != nil {
		return err
	}
	for _, p := range untracked {
		fmt.Printf("as-skill: doctor: %-11s %s — present on disk, no lockfile entry\n", health.HealthUntracked, p)
	}
	checked += len(untracked)
	failed += len(untracked)

	for _, d := range health.Duplicates(projectRoot, lock) {
		fmt.Printf("as-skill: doctor: warning: %-15s %s\n", health.DuplicateSkill, d)
	}
	for _, u := range health.UnverifiedCaps(lock) {
		fmt.Printf("as-skill: doctor: warning: %-15s %s\n", health.Unverified, u)
	}

	if codexHooksNeedTrust(projectRoot, lock) {
		fmt.Printf("as-skill: doctor: warning: %-15s codex hooks need trust: review in /hooks\n", "HOOKS-TRUST")
	}

	lint := lintHarness(harnessRoot)
	for _, f := range lint.findings {
		fmt.Printf("as-skill: doctor: %-11s %s\n", "lint", f)
	}
	checked += lint.checked
	failed += lint.failed

	fmt.Printf("doctor: %d checks, %d/%d ok, %d failed\n", checked, checked-failed, checked, failed)
	if failed > 0 {
		return fmt.Errorf("doctor: %d failed", failed)
	}
	return nil
}

func codexHooksNeedTrust(projectRoot string, lock *lockfile.Lockfile) bool {
	has := false
	for _, p := range health.LockPlatforms(lock) {
		if p == platform.Codex {
			has = true
		}
	}
	if !has {
		return false
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, ".codex", "hooks.json"))
	return err == nil && strings.Contains(string(data), "./.codex/hooks/")
}
