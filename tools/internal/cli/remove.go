package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claude-harness/tools/internal/fsutil"
	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/installer"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/prompt"
	"claude-harness/tools/internal/transfer"
)

type removeTarget struct {
	name   string
	domain bool
	rows   []lockfile.Entry
	cover  []lockfile.Entry
}

func skillLocPaths(projectRoot, name string) []string {
	locs, _ := platform.SkillLocations(platform.IDs())
	var out []string
	for l := range locs {
		out = append(out, filepath.Join(projectRoot, l, name))
	}
	sort.Strings(out)
	return out
}

func coverRows(lock *lockfile.Lockfile, projectRoot, name string) []lockfile.Entry {
	var out []lockfile.Entry
	for _, p := range skillLocPaths(projectRoot, name) {
		if e, ok := lock.Covering(p); ok && e.Name == "" {
			out = append(out, e)
		}
	}
	return out
}

func removeTargets(lock *lockfile.Lockfile, projectRoot string, names []string) ([]removeTarget, error) {
	var out []removeTarget
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			continue
		}
		seen[n] = true
		var dRows, sRows []lockfile.Entry
		for _, e := range lock.Entries {
			if e.Domain == n && e.Domain != "as-skill" {
				dRows = append(dRows, e)
			}
			if e.Name == n && e.Domain != "as-skill" && (e.Kind == "skill" || e.Kind == "skills") {
				sRows = append(sRows, e)
			}
		}
		switch {
		case len(dRows) > 0:
			out = append(out, removeTarget{n, true, dRows, nil})
		case len(sRows) > 0:
			out = append(out, removeTarget{n, false, sRows, coverRows(lock, projectRoot, n)})
		default:
			if cov := coverRows(lock, projectRoot, n); len(cov) > 0 {
				return nil, &installer.PartOfDomainError{Skill: n, Domain: cov[0].Domain, Mode: cov[0].Mode}
			}
			return nil, fmt.Errorf("%q is not installed in this project (see `as-skill doctor`)", n)
		}
	}
	return out, nil
}

func recordedPlatforms(rows []lockfile.Entry) []platform.ID {
	seen := map[string]bool{}
	for _, e := range rows {
		for _, p := range e.Platforms {
			seen[p] = true
		}
	}
	var out []platform.ID
	for _, id := range platform.IDs() {
		if seen[string(id)] {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return []platform.ID{platform.Claude}
	}
	return out
}

func runRemove(args []string) error {
	opts, err := parseAddArgs("remove", args)
	if err != nil {
		return err
	}
	if opts.help {
		printUsage(os.Stdout)
		return nil
	}
	dlg := wantDialog(opts)
	if !dlg && (len(opts.names) == 0 || !(opts.yes || opts.dryRun)) {
		return gateError(opts)
	}
	var pr prompt.Prompter
	if dlg {
		pr = newPrompter()
	}
	var (
		projectRoot, harnessRoot string
		prevProj                 string
		prevNames                []string
		prevPlats                []platform.ID
		lock                     *lockfile.Lockfile
		targets                  []removeTarget
		reg                      *harness.Registry
		plats                    []platform.ID
	)
	nameArgs := len(opts.names) > 0
	names := opts.names
	err = runSteps([]func() (bool, error){
		func() (bool, error) {
			var err error
			if !dlg {
				projectRoot, err = resolveProject(opts.project)
				return false, err
			}
			projectRoot, err = askProject(pr, opts, prevProj)
			if err == nil {
				prevProj = projectRoot
			}
			return !opts.projectSet, err
		},
		func() (bool, error) {
			var err error
			lock, err = lockfile.Load(projectRoot)
			return false, err
		},
		func() (bool, error) {
			if !dlg || nameArgs {
				return false, nil
			}
			var err error
			if names, err = askRemoveNames(pr, lock, projectRoot, prevNames); err != nil {
				return true, err
			}
			prevNames = names
			return true, nil
		},
		func() (bool, error) {
			var err error
			targets, err = removeTargets(lock, projectRoot, names)
			if err != nil {
				return false, err
			}
			var herr error
			harnessRoot, herr = resolveHarness(opts.harnessRoot)
			reg = nil
			if herr == nil {
				r, rerr := harness.LoadRegistry(harnessRoot)
				if rerr == nil {
					reg = r
				} else if opts.harnessRoot != "" {
					return false, fmt.Errorf("remove: --harness-root rejected: %w", rerr)
				}
			} else if opts.harnessRoot != "" {
				return false, fmt.Errorf("remove: --harness-root rejected: %w", herr)
			}
			return false, nil
		},
		func() (bool, error) {
			var allRows []lockfile.Entry
			for _, t := range targets {
				allRows = append(allRows, t.rows...)
				allRows = append(allRows, t.cover...)
			}
			if len(opts.agents) > 0 {
				var err error
				plats, err = platform.ParsePlatforms(opts.agents)
				return false, err
			}
			plats = recordedPlatforms(allRows)
			if dlg && len(plats) > 1 {
				var err error
				if plats, err = askRemovePlatforms(pr, plats, prevPlats); err != nil {
					return true, err
				}
				prevPlats = plats
				return true, nil
			}
			return false, nil
		},
		func() (bool, error) {
			if dlg && !opts.dryRun {
				return true, confirm(pr, "Remove?", removeSummary(projectRoot, names, plats))
			}
			return false, nil
		},
	})
	if err != nil {
		return err
	}
	opts.names = names
	if reg == nil {
		fmt.Fprintln(os.Stderr, "as-skill: note: harness checkout not found; removing files and lockfile rows only (hook registrations and AGENTS.md blocks are left in place)")
	}
	fmt.Printf("platforms: %s\n", strings.Join(platformDirNames(plats), ", "))

	for _, t := range targets {
		if err := checkPartOfDomain(t, plats); err != nil {
			return err
		}
	}

	in := &installer.Installer{
		HarnessRoot: harnessRoot, ProjectRoot: projectRoot,
		DryRun: opts.dryRun, Lock: lock, Platforms: plats,
	}
	rm := &remover{want: map[string]bool{}, lock: lock, project: projectRoot, dry: opts.dryRun}
	for _, p := range plats {
		rm.want[string(p)] = true
	}
	for _, t := range targets {
		if err := rm.preflight(t.rows); err != nil {
			return err
		}
	}
	var runErr error
	for _, t := range targets {
		runErr = removeOne(in, reg, rm, t)
		if runErr != nil {
			break
		}
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
	if err := rm.problems(); err != nil {
		return err
	}
	for _, id := range plats {
		fmt.Printf("done -> %s\n", filepath.Join(projectRoot, "."+string(id)))
	}
	return nil
}

func checkPartOfDomain(t removeTarget, plats []platform.ID) error {
	if t.domain {
		return nil
	}
	for _, c := range t.cover {
		hit := false
		for _, p := range c.Platforms {
			for _, id := range plats {
				if p == string(id) {
					hit = true
				}
			}
		}
		if !hit {
			continue
		}
		own := false
		for _, r := range t.rows {
			if filepath.Clean(r.Path) == filepath.Join(filepath.Clean(c.Path), t.name) {
				own = true
			}
		}
		if !own {
			return &installer.PartOfDomainError{Skill: t.name, Domain: c.Domain, Mode: c.Mode}
		}
	}
	return nil
}

func removeOne(in *installer.Installer, reg *harness.Registry, rm *remover, t removeTarget) error {
	if reg != nil {
		var err error
		handled := false
		if t.domain {
			if _, ok := reg.Domains[t.name]; ok {
				handled, err = true, in.UninstallDomain(reg, t.name)
			}
		} else if skillOwner(in.HarnessRoot, reg, t.name) != "" {
			handled = true
			err = in.UninstallSkill(reg, t.name)
			var pe *installer.PartOfDomainError
			if errors.As(err, &pe) {
				err = fmt.Errorf("%s", pe.Error())
			}
		}
		if handled {
			if err != nil {
				return err
			}
			rm.sweep(t.rows, true)
			return nil
		}
	}
	rm.sweep(t.rows, false)
	return nil
}

type remover struct {
	lock    *lockfile.Lockfile
	project string
	dry     bool
	want    map[string]bool
	refused []string
	legacy  []string
}

func (r *remover) problems() error {
	if len(r.refused) == 0 && len(r.legacy) == 0 {
		return nil
	}
	var msg []string
	if len(r.refused) > 0 {
		msg = append(msg, fmt.Sprintf("refused %d path(s) outside the project's platform dirs (rows kept): %s", len(r.refused), strings.Join(r.refused, ", ")))
	}
	if len(r.legacy) > 0 {
		msg = append(msg, fmt.Sprintf("%d legacy copy row(s) have no recorded item list and their source checkout is gone, so nothing was removed for them (rows kept): %s; reinstall once with the current binary (as-skill add ... --copy) and remove again, or delete the files manually",
			len(r.legacy), strings.Join(r.legacy, ", ")))
	}
	return fmt.Errorf("remove incomplete: %s", strings.Join(msg, "; "))
}

// sweep removes the given rows (re-read from the lock; rows already gone are
// skipped) purely from lockfile data. afterHarness limits dry-run output to
// rows whose source item vanished so the harness walk is not previewed twice.
func (r *remover) sweep(rows []lockfile.Entry, afterHarness bool) {
	for _, row := range rows {
		e, ok := r.lock.FindAt(row.Domain, row.Kind, row.Name, row.Path)
		if !ok {
			continue
		}
		if afterHarness && r.dry && (e.Source == "" || e.Mode == "copy" || fsutil.PathExists(e.Source)) {
			continue
		}
		r.removeRow(e)
	}
}

func (r *remover) removeRow(e lockfile.Entry) {
	hit := len(e.Platforms) == 0
	for _, p := range e.Platforms {
		if r.want[p] {
			hit = true
		}
	}
	if !hit {
		return
	}
	if e.Mode == "block" {
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — managed block needs the harness checkout\n", e.Path)
		return
	}
	var remaining []string
	for _, p := range e.Platforms {
		if !r.want[p] {
			remaining = append(remaining, p)
		}
	}
	if len(remaining) > 0 {
		if r.dry {
			fmt.Printf("would keep     %s — still used by %s\n", e.Path, strings.Join(remaining, ","))
			return
		}
		e.Platforms = remaining
		r.lock.Record(e)
		fmt.Printf("kept           %s — still used by %s\n", e.Path, strings.Join(remaining, ","))
		return
	}
	paths, ok := r.rowPaths(e)
	if !ok {
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — legacy copy row has no item list and its source is gone\n", e.Path)
		r.legacy = append(r.legacy, e.Path)
		return
	}
	clean := true
	for _, p := range paths {
		if ierr := installer.InsideRoots(r.project, p); ierr != nil {
			fmt.Fprintf(os.Stderr, "as-skill: error: refusing %v\n", ierr)
			r.refused = append(r.refused, p)
			clean = false
			continue
		}
		fi, err := os.Lstat(p)
		if err != nil {
			continue
		}
		isLink := fi.Mode()&os.ModeSymlink != 0
		if e.Mode == "link" && !isLink {
			fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — tracked as a link but not a symlink on disk\n", p)
			clean = false
			continue
		}
		if r.dry {
			fmt.Printf("would remove   %s\n", p)
			continue
		}
		var rerr error
		if isLink {
			rerr = os.Remove(p)
		} else {
			rerr = os.RemoveAll(p)
		}
		if rerr != nil {
			fmt.Fprintf(os.Stderr, "as-skill: warning: failed removing %s: %v\n", p, rerr)
			clean = false
			continue
		}
		fmt.Printf("uninstalled    %s\n", p)
	}
	if clean && !r.dry {
		if e.Name == "" {
			r.lock.RemoveAt(e.Domain, e.Kind, "", e.Path)
		} else {
			r.lock.RemoveAllAt(e.Path)
		}
	}
}

// rowPaths lists the concrete paths a row owns. ok=false: a legacy copy row
// that cannot be expanded because its source is gone.
func (r *remover) rowPaths(e lockfile.Entry) (paths []string, ok bool) {
	switch {
	case e.Name != "":
		return []string{e.Path}, true
	case e.Mode == "copy" && len(e.Items) > 0:
		return e.CoveredPaths(), true
	case e.Mode == "copy":
		names, err := transfer.TopNames(e.Source, e.Kind)
		if err != nil {
			return nil, false
		}
		for _, n := range names {
			if p := filepath.Join(e.Path, n); r.lock.Covers(p) {
				paths = append(paths, p)
			}
		}
		return paths, true
	}
	return []string{e.Path}, true
}

// preflight refuses, before the installer touches anything, rows owning paths
// outside the project's platform dirs (the installer does not check this).
func (r *remover) preflight(rows []lockfile.Entry) error {
	var bad []string
	for _, e := range rows {
		hit := false
		for _, p := range e.Platforms {
			if r.want[p] {
				hit = true
			}
		}
		if !hit || e.Mode == "block" {
			continue
		}
		paths, _ := r.rowPaths(e)
		for _, p := range paths {
			if installer.InsideRoots(r.project, p) != nil {
				bad = append(bad, p)
			}
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("remove refused, nothing changed: lockfile rows point outside the project's platform dirs: %s", strings.Join(bad, ", "))
	}
	return nil
}
