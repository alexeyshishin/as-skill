package health

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/registry"
	"claude-harness/tools/internal/transfer"
)

const (
	HealthOK         = "OK"
	HealthMissing    = "MISSING"
	HealthBroken     = "BROKEN"
	HealthSourceGone = "SOURCE-GONE"
	HealthDrifted    = "DRIFTED"
	HealthUntracked  = "UNTRACKED"
	HealthStale      = "STALE"
	DuplicateSkill   = "DUPLICATE-SKILL"
	Unverified       = "UNVERIFIED"
)

func EntryHealth(e lockfile.Entry) string {
	fi, err := os.Lstat(e.Path)
	if err != nil {
		return HealthMissing
	}
	if e.Mode == "link" {
		if fi.Mode()&os.ModeSymlink == 0 {
			return HealthBroken
		}
		target, err := os.Readlink(e.Path)
		if err != nil || target != e.Source {
			return HealthBroken
		}
	}
	return HealthOK
}

func SortedEntries(lock *lockfile.Lockfile) []lockfile.Entry {
	entries := append([]lockfile.Entry(nil), lock.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Domain != entries[j].Domain {
			return entries[i].Domain < entries[j].Domain
		}
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Path < entries[j].Path
	})
	return entries
}

func FilterEntries(entries []lockfile.Entry, plats []string) []lockfile.Entry {
	if len(plats) == 0 {
		return entries
	}
	var out []lockfile.Entry
	for _, e := range entries {
		for _, p := range e.Platforms {
			if slices.Contains(plats, p) {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

func PrintEntryTable(entries []lockfile.Entry, healthOf func(lockfile.Entry) string) {
	if len(entries) == 0 {
		fmt.Println("no entries tracked in skills-lock.json")
		return
	}
	fmt.Printf("%-10s %-8s %-24s %-6s %-11s %-22s %s\n", "DOMAIN", "KIND", "NAME", "MODE", "HEALTH", "PLATFORMS", "PATH")
	for _, e := range entries {
		fmt.Printf("%-10s %-8s %-24s %-6s %-11s %-22s %s\n", e.Domain, e.Kind, e.Name, e.Mode, healthOf(e), strings.Join(e.Platforms, ","), e.Path)
	}
}

func LockPlatforms(lock *lockfile.Lockfile) []platform.ID {
	seen := map[string]bool{string(platform.Claude): true}
	for _, e := range lock.Entries {
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
	return out
}

func RenderState(e lockfile.Entry) (stale, drifted bool) {
	if e.Mode != lockfile.ModeRender {
		return false, false
	}
	if data, err := os.ReadFile(e.Source); err == nil && e.SrcSHA256 != "" && transfer.HashBytes(data) != e.SrcSHA256 {
		stale = true
	}
	if out, err := os.ReadFile(e.Path); err == nil && e.SHA256 != "" && transfer.HashBytes(out) != e.SHA256 {
		drifted = true
	}
	return stale, drifted
}

func platformDirs(plats []platform.ID) []string {
	seen := map[string]bool{}
	var dirs []string
	add := func(d string) {
		if d != "" && !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	for _, kind := range registry.DomainKinds {
		add(".claude/" + kind)
	}
	for _, id := range plats {
		p, ok := platform.Get(id)
		if !ok || id == platform.Claude {
			continue
		}
		locs, _ := platform.SkillLocations([]platform.ID{id})
		for loc := range locs {
			add(loc)
		}
		add(p.Agents.Dir)
		add(p.Rules.Dir)
		add(p.Hooks.Dir)
		add(".agents/prompts")
	}
	return dirs
}

func Duplicates(projectRoot string, lock *lockfile.Lockfile) []string {
	var out []string
	reported := map[string]bool{}
	for _, id := range LockPlatforms(lock) {
		p, _ := platform.Get(id)
		where := map[string][]string{}
		for _, d := range p.SkillDirsRead {
			entries, err := os.ReadDir(filepath.Join(projectRoot, d))
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
					where[e.Name()] = append(where[e.Name()], d)
				}
			}
		}
		var names []string
		for n, ds := range where {
			if len(ds) > 1 {
				names = append(names, n)
			}
		}
		sort.Strings(names)
		for _, n := range names {
			msg := fmt.Sprintf("skill %q is in %s — %s sees it twice", n, strings.Join(where[n], " and "), id)
			if !reported[msg] {
				reported[msg] = true
				out = append(out, msg)
			}
		}
	}
	return out
}

func UnverifiedCaps(lock *lockfile.Lockfile) []string {
	var out []string
	for _, id := range LockPlatforms(lock) {
		p, _ := platform.Get(id)
		for _, cap := range []string{"skills", "agents", "rules", "hooks"} {
			if p.Verified[cap] == "" {
				out = append(out, fmt.Sprintf("%s %s capability is unverified", id, cap))
			}
		}
	}
	return out
}

func FindUntracked(projectRoot string, lock *lockfile.Lockfile) ([]string, error) {
	opaque := map[string]bool{}
	for _, e := range lock.Entries {
		if e.Name != "" || e.Source == "" && len(e.Items) == 0 {
			opaque[filepath.Clean(e.Path)] = true
		}
	}
	var untracked []string
	for _, rel := range platformDirs(LockPlatforms(lock)) {
		dir := filepath.Join(projectRoot, rel)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if lock.Covers(p) || opaque[filepath.Clean(dir)] && !hasSourcedRow(lock, dir) {
				continue
			}
			untracked = append(untracked, p)
		}
	}
	sort.Strings(untracked)
	return untracked, nil
}

var ErrCannotVerify = errors.New("cannot verify: row has no items and its source is gone")

// CopyMatches verifies one row without lock context (no cumulative variant).
func CopyMatches(e lockfile.Entry) (bool, error) {
	return CopyMatchesIn(nil, e)
}

// CopyMatchesIn verifies a copy row. Rows with Items hash over Items. Legacy
// kind-level rows try TopNames(Source), the cumulative set (earlier rows at the
// same Path), then the whole-dir hash. Returns ErrCannotVerify when nothing
// matched and the names are unknowable (Source gone, no Items).
func CopyMatchesIn(lock *lockfile.Lockfile, e lockfile.Entry) (bool, error) {
	if e.Name != "" || e.Kind == "" && e.Source == "" {
		hash, err := transfer.HashPath(e.Path)
		if err != nil {
			return false, err
		}
		return hash == e.SHA256, nil
	}
	if len(e.Items) > 0 {
		hash, err := transfer.HashNames(e.Path, e.Items)
		if err != nil {
			return false, err
		}
		return hash == e.SHA256, nil
	}
	try := func(names []string) (bool, error) {
		hash, err := transfer.HashNames(e.Path, names)
		if err != nil {
			return false, err
		}
		return hash == e.SHA256, nil
	}
	srcKnown := false
	var own []string
	if e.Source != "" {
		if names, err := transfer.TopNames(e.Source, e.Kind); err == nil {
			srcKnown = true
			own = names
			if ok, err := try(own); err != nil || ok {
				return ok, err
			}
		}
	}
	if lock != nil {
		set := map[string]bool{}
		for _, n := range own {
			set[n] = true
		}
		complete := srcKnown
		for _, o := range lock.Entries {
			if o.Name != "" || filepath.Clean(o.Path) != filepath.Clean(e.Path) || !o.InstalledAt.Before(e.InstalledAt) {
				continue
			}
			names := o.Items
			if len(names) == 0 && o.Source != "" {
				names, _ = transfer.TopNames(o.Source, o.Kind)
			}
			if len(names) == 0 {
				complete = false
			}
			for _, n := range names {
				set[n] = true
			}
		}
		if len(set) > 0 && complete {
			all := make([]string, 0, len(set))
			for n := range set {
				all = append(all, n)
			}
			sort.Strings(all)
			if ok, err := try(all); err != nil || ok {
				return ok, err
			}
		}
	}
	hash, err := transfer.HashPath(e.Path)
	if err != nil {
		return false, err
	}
	if hash == e.SHA256 {
		return true, nil
	}
	if !srcKnown {
		return false, ErrCannotVerify
	}
	return false, nil
}

func hasSourcedRow(lock *lockfile.Lockfile, dir string) bool {
	clean := filepath.Clean(dir)
	for _, e := range lock.Entries {
		if e.Name == "" && (e.Source != "" || len(e.Items) > 0) && filepath.Clean(e.Path) == clean {
			return true
		}
	}
	return false
}
