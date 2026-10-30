package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"claude-harness/tools/internal/fsutil"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/registry"
	"claude-harness/tools/internal/transfer"
)

type Installer struct {
	HarnessRoot string
	ProjectRoot string
	DryRun      bool
	Force       bool
	Mode        string
	Lock        *lockfile.Lockfile
	Platforms   []platform.ID

	ExcludeSkills map[string]bool
	Overwrite     map[string]bool

	gitChecked map[string]bool
	dupWarned  bool
	gapsDone   bool
	collecting bool
	conflicts  []ConflictError
}

func (in *Installer) place(domain, kind, name, src, dst string) (int, error) {
	if in.Mode == "link" {
		if kind == "skill" {
			if !in.DryRun {
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					return 0, err
				}
			}
			if err := in.linkOne(src, dst); err != nil {
				return 0, err
			}
			if !in.DryRun {
				in.Lock.Record(lockfile.Entry{
					Domain: domain, Kind: kind, Name: name,
					Mode: "link", Path: dst, Source: src,
					InstalledAt: time.Now(),
				})
			}
			return 1, nil
		}
		var skip func(string) bool
		if kind == "skills" && in.hasExclusions(domain) {
			skip = func(item string) bool { return in.excluded(domain, kind, item) }
		}
		n, err := transfer.LinkTreeWith(domain, kind, src, dst, in.DryRun, in.Lock, skip, in.linkOne)
		if err == nil && !in.DryRun && name == "" {
			if skip == nil {
				in.Lock.RemoveAt(domain, kind, "", dst)
			} else {
				in.keepExcluded(domain, kind, dst, skip)
			}
		}
		return n, err
	}

	if entry, ok := in.Lock.Owns(dst); ok && entry.Mode != "copy" {
		fmt.Fprintf(os.Stderr, "as-skill: warning: %s was installed via %q, replacing with a copy — mixed channels may cause confusion\n", dst, entry.Mode)
	}
	if !in.DryRun && name == "" {
		in.dropLinksUnder(kind, src, dst)
	}
	n, err := transfer.CopyTree(kind, src, dst, in.DryRun)
	if err != nil {
		return n, err
	}
	if !in.DryRun {
		var hash string
		var items []string
		if name == "" {
			names, nerr := transfer.TopNames(src, kind)
			items = append(items, names...)
			sort.Strings(items)
			if nerr != nil {
				return n, nerr
			}
			hash, err = transfer.HashNames(dst, names)
		} else {
			hash, err = transfer.HashTree(dst)
		}
		if err != nil {
			return n, fmt.Errorf("hashing %s: %w", dst, err)
		}
		for _, it := range items {
			in.Lock.RemoveAllAt(filepath.Join(dst, it))
		}
		in.Lock.Record(lockfile.Entry{
			Domain: domain, Kind: kind, Name: name,
			Mode: "copy", Path: dst, Source: src, SHA256: hash, Items: items,
			InstalledAt: time.Now(),
		})
	}
	return n, nil
}

func (in *Installer) keepExcluded(domain, kind, dst string, skip func(string) bool) {
	parent, ok := in.Lock.FindAt(domain, kind, "", dst)
	if !ok {
		return
	}
	if parent.Mode != lockfile.ModeCopy {
		in.Lock.RemoveAt(domain, kind, "", dst)
		return
	}
	var keep []string
	for _, p := range parent.CoveredPaths() {
		item := filepath.Base(p)
		if fi, err := os.Lstat(p); skip(item) && err == nil && fi.Mode()&os.ModeSymlink == 0 {
			keep = append(keep, item)
		}
	}
	sort.Strings(keep)
	hash, err := transfer.HashNames(dst, keep)
	if len(keep) == 0 || err != nil {
		in.Lock.RemoveAt(domain, kind, "", dst)
		return
	}
	parent.Items, parent.SHA256 = keep, hash
	in.Lock.Record(parent)
}

func (in *Installer) dropLinksUnder(kind, src, dst string) {
	names, err := transfer.TopNames(src, kind)
	if err != nil {
		return
	}
	for _, n := range names {
		p := filepath.Join(dst, n)
		fi, err := os.Lstat(p)
		if err != nil || fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		if e, ok := in.Lock.Owns(p); ok && e.Mode == "link" {
			if os.Remove(p) == nil {
				in.Lock.RemoveAllAt(p)
			}
		}
	}
}

func (in *Installer) InstallDomain(reg *registry.Registry, name string, strict bool) error {
	m, ok := reg.Domains[name]
	if !ok {
		return fmt.Errorf("unknown domain %q (available: %s)", name, strings.Join(reg.DomainNames(), ", "))
	}
	for _, envVar := range m.RequiresEnv {
		if os.Getenv(envVar) != "" {
			continue
		}
		msg := fmt.Sprintf("domain %q requires $%s, which is not set", name, envVar)
		if strict {
			return errors.New(msg)
		}
		fmt.Fprintf(os.Stderr, "as-skill: skipping — %s\n", msg)
		return nil
	}
	for _, bin := range m.RequiresBin {
		if fsutil.HasBin(bin) {
			continue
		}
		msg := fmt.Sprintf("domain %q requires the %q binary, which is not on $PATH", name, bin)
		if strict {
			return errors.New(msg)
		}
		fmt.Fprintf(os.Stderr, "as-skill: skipping — %s\n", msg)
		return nil
	}
	if err := in.Preflight(reg, []string{name}); err != nil {
		return err
	}
	for _, kind := range registry.DomainKinds {
		srcDir := filepath.Join(in.HarnessRoot, "domains", name, kind)
		if !fsutil.DirExists(srcDir) {
			continue
		}
		if err := in.installKind(m, name, kind, srcDir); err != nil {
			return err
		}
	}
	return nil
}

func (in *Installer) claudeKind(m *registry.Manifest, name, kind, srcDir string, plats []platform.ID) error {
	destPath, err := in.resolveTarget(m, kind)
	if err != nil {
		return fmt.Errorf("domain %q: %w", name, err)
	}
	prior := in.snapshotPlatforms(name, kind, destPath)
	defer in.applyPlatforms(name, kind, destPath, plats, prior)
	if kind == "skills" && in.Mode != "link" && in.hasExclusions(name) {
		if err := in.placeItems(name, kind, srcDir, destPath, plats); err != nil {
			return fmt.Errorf("domain %q: copying %s: %w", name, kind, err)
		}
		if !in.DryRun {
			in.keepExcluded(name, kind, destPath, func(item string) bool { return in.excluded(name, kind, item) })
		}
		return nil
	}
	n, err := in.place(name, kind, "", srcDir, destPath)
	if err != nil {
		return fmt.Errorf("domain %q: copying %s: %w", name, kind, err)
	}
	in.report(name, kind, destPath, n)
	if kind == "hooks" && !in.DryRun {
		if in.Mode == "copy" {
			if err := transfer.MakeShellScriptsExecutable(destPath); err != nil {
				return fmt.Errorf("domain %q: chmod hooks: %w", name, err)
			}
		} else {
			transfer.WarnHooksNotExecutable(srcDir)
		}
	}
	return nil
}

func (in *Installer) InstallSkill(reg *registry.Registry, name string) error {
	owner := ""
	for _, dname := range reg.DomainNames() {
		if fsutil.DirExists(filepath.Join(in.HarnessRoot, "domains", dname, "skills", name)) {
			owner = dname
			break
		}
	}
	if owner == "" {
		return fmt.Errorf("unknown skill %q (see `as-skill list skills`)", name)
	}
	m := reg.Domains[owner]
	for _, envVar := range m.RequiresEnv {
		if os.Getenv(envVar) == "" {
			return fmt.Errorf("skill %q belongs to domain %q, which requires $%s (not set)", name, owner, envVar)
		}
	}
	for _, bin := range m.RequiresBin {
		if !fsutil.HasBin(bin) {
			return fmt.Errorf("skill %q belongs to domain %q, which requires the %q binary (not found on $PATH)", name, owner, bin)
		}
	}
	src := filepath.Join(in.HarnessRoot, "domains", owner, "skills", name)
	if err := in.preflightRoots(map[string]bool{"skills": true}); err != nil {
		return err
	}
	locs, _ := platform.SkillLocations(in.plats())
	in.warnDup(locs)
	for _, loc := range sortedLocs(locs) {
		var dest string
		if loc == claudeSkillsLoc {
			destBase, err := in.resolveTarget(m, "skills")
			if err != nil {
				return fmt.Errorf("skill %q: %w", name, err)
			}
			dest = filepath.Join(destBase, name)
			prior := in.snapshotPlatforms(owner, "skill", dest)
			n, err := in.place(owner, "skill", name, src, dest)
			if err != nil {
				return fmt.Errorf("skill %q: %w", name, err)
			}
			in.applyPlatforms(owner, "skill", dest, locs[loc], prior)
			in.report(owner, "skill", dest, n)
			continue
		}
		dest = filepath.Join(in.ProjectRoot, loc, name)
		if err := in.placeOne(owner, "skill", name, src, dest, locs[loc]); err != nil {
			return fmt.Errorf("skill %q: %w", name, err)
		}
		in.report(owner, "skill", dest, 1)
	}
	return nil
}

func (in *Installer) UninstallDomain(reg *registry.Registry, name string) error {
	m, ok := reg.Domains[name]
	if !ok {
		return fmt.Errorf("unknown domain %q (available: %s)", name, strings.Join(reg.DomainNames(), ", "))
	}
	if err := in.preflightUninstall(reg, name, ""); err != nil {
		return err
	}
	if in.hasClaude() {
		if err := in.uninstallClaudeDomain(m, name); err != nil {
			return err
		}
	}
	in.uninstallNonClaude(name, "")
	return nil
}

func (in *Installer) uninstallClaudeDomain(m *registry.Manifest, name string) error {
	for _, kind := range registry.DomainKinds {
		srcDir := filepath.Join(in.HarnessRoot, "domains", name, kind)
		if !fsutil.DirExists(srcDir) {
			continue
		}
		destPath, err := in.resolveTarget(m, kind)
		if err != nil {
			return fmt.Errorf("domain %q: %w", name, err)
		}
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			return fmt.Errorf("domain %q: %w", name, err)
		}
		var goneHooks []string
		for _, e := range entries {
			itemName := e.Name()
			if transfer.SkipTop(kind, itemName) {
				continue
			}
			dst := filepath.Join(destPath, itemName)
			if gone := in.uninstallOne(name, kind, itemName, dst); gone && kind == "hooks" {
				goneHooks = append(goneHooks, itemName)
			}
		}
		if kind == "hooks" {
			in.unregisterHooks(goneHooks)
		}

		if parent, tracked := in.Lock.Find(name, kind, ""); tracked && len(minusPlats(parent.Platforms, in.plats())) == 0 {
			in.Lock.Remove(name, kind, "")
		}
	}
	return nil
}

type PartOfDomainError struct {
	Skill  string
	Domain string
	Mode   string
}

func (e *PartOfDomainError) Error() string {
	return fmt.Sprintf("skill %q installed as part of domain %s (%s); remove the domain", e.Skill, e.Domain, e.Mode)
}

func (in *Installer) UninstallSkill(reg *registry.Registry, name string) error {
	owner := ""
	for _, dname := range reg.DomainNames() {
		if fsutil.DirExists(filepath.Join(in.HarnessRoot, "domains", dname, "skills", name)) {
			owner = dname
			break
		}
	}
	if owner == "" {
		return fmt.Errorf("unknown skill %q (see `as-skill list skills`)", name)
	}
	m := reg.Domains[owner]
	if err := in.preflightUninstall(reg, owner, name); err != nil {
		return err
	}
	if in.hasClaude() {
		destBase, err := in.resolveTarget(m, "skills")
		if err != nil {
			return fmt.Errorf("skill %q: %w", name, err)
		}
		dst := filepath.Join(destBase, name)
		if _, owned := in.Lock.Owns(dst); !owned {
			if parent, ok := in.Lock.Covering(dst); ok && parent.Name == "" {
				if _, err := os.Lstat(dst); err == nil {
					return &PartOfDomainError{Skill: name, Domain: parent.Domain, Mode: parent.Mode}
				}
			}
		}
		in.uninstallOne(owner, "skill", name, dst)
	}
	in.uninstallNonClaude(owner, name)
	return nil
}

func (in *Installer) uninstallOne(domain, kind, name, dst string) bool {
	entry, tracked := in.Lock.Covering(dst)
	if !tracked {
		if parent, ok := in.Lock.Find(domain, kind, ""); ok {
			entry, tracked = parent, true
		}
	}
	if !tracked {
		if _, err := os.Lstat(dst); os.IsNotExist(err) {
			return false
		}
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — not tracked by as-skill\n", dst)
		return false
	}
	fi, err := os.Lstat(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — tracked but nothing on disk\n", dst)
		return os.IsNotExist(err)
	}
	if remaining := minusPlats(entry.Platforms, in.plats()); len(entry.Platforms) > 0 && len(remaining) == len(entry.Platforms) {
		return false
	} else if len(remaining) > 0 {
		if in.DryRun {
			fmt.Printf("would keep     %s — still used by %s\n", dst, strings.Join(remaining, ","))
			return false
		}
		in.setEntryPlatforms(entry, remaining)
		fmt.Printf("kept           %s — still used by %s\n", dst, strings.Join(remaining, ","))
		return false
	}
	isSymlink := fi.Mode()&os.ModeSymlink != 0
	switch entry.Mode {
	case "link":
		if !isSymlink {
			fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — tracked as a link but not a symlink on disk\n", dst)
			return false
		}
	case "copy":
		if isSymlink {
			fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — tracked as a copy but is a symlink on disk\n", dst)
			return false
		}
	default:
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — unrecognized tracked mode %q\n", dst, entry.Mode)
		return false
	}
	if in.DryRun {
		fmt.Printf("would remove   %s\n", dst)
		return true
	}
	var removeErr error
	if entry.Mode == "link" {
		removeErr = os.Remove(dst)
	} else {
		removeErr = os.RemoveAll(dst)
	}
	if removeErr != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: failed removing %s: %v\n", dst, removeErr)
		return false
	}
	in.Lock.RemoveAllAt(dst)
	fmt.Printf("uninstalled    %s\n", dst)
	return true
}

func (in *Installer) report(scope, kind, dest string, n int) {
	verb := "installed"
	if in.DryRun {
		verb = "would install"
	}
	plural := "s"
	if n == 1 {
		plural = ""
	}
	fmt.Printf("%-13s %-8s %-7s -> %s (%d file%s)\n", verb, scope, kind, dest, n, plural)
}
