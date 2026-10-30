package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"claude-harness/tools/internal/fsys"
	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

type Installer struct {
	HarnessRoot string
	ProjectRoot string
	DryRun      bool
	Force       bool
	Mode        lockfile.Mode
	Lock        *lockfile.Lockfile
	Platforms   []platform.ID
	Reporter

	ExcludeSkills map[string]bool
	Overwrite     map[string]bool
}

func (in *Installer) place(domain string, kind lockfile.Kind, name, src, dst string) (int, error) {
	if in.Mode == lockfile.ModeLink {
		if kind == lockfile.KindSkill {
			if err := in.mkdirParent(dst); err != nil {
				return 0, err
			}
			if err := in.linkOne(src, dst); err != nil {
				return 0, err
			}
			in.record(lockfile.Entry{
				Domain: domain, Kind: kind, Name: name,
				Mode: lockfile.ModeLink, Path: dst, Source: src,
				InstalledAt: time.Now(),
			})
			return 1, nil
		}
		var skip func(string) bool
		if kind == lockfile.KindSkills && in.hasExclusions(domain) {
			skip = func(item string) bool { return in.excluded(domain, kind, item) }
		}
		linked, err := fsys.LinkTreeWith(kind.SkipsReadme(), src, dst, in.DryRun, skip, in.linkOne)
		n := len(linked)
		for _, l := range linked {
			in.record(lockfile.Entry{
				Domain:      domain,
				Kind:        kind,
				Name:        l.Name,
				Mode:        lockfile.ModeLink,
				Path:        l.Dst,
				Source:      l.Src,
				InstalledAt: time.Now(),
			})
		}
		if err == nil && name == "" {
			in.settleParent(domain, kind, dst, skip)
		}
		return n, err
	}

	if name == "" {
		in.dropLinksUnder(kind, src, dst)
	}
	n, err := fsys.CopyTreeRoot(kind.SkipsReadme(), src, dst, in.HarnessRoot, in.DryRun)
	if err != nil {
		return n, err
	}
	return n, in.recordCopy(domain, kind, name, src, dst)
}

func (in *Installer) recordCopy(domain string, kind lockfile.Kind, name, src, dst string) error {
	if in.DryRun {
		return nil
	}
	var hash string
	var items []string
	var err error
	if name == "" {
		names, nerr := fsys.TopNames(src, kind.SkipsReadme())
		items = append(items, names...)
		sort.Strings(items)
		if nerr != nil {
			return nerr
		}
		hash, err = fsys.HashNames(dst, names)
	} else {
		hash, err = fsys.HashTree(dst)
	}
	if err != nil {
		return fmt.Errorf("hashing %s: %w", dst, err)
	}
	for _, it := range items {
		in.Lock.RemoveAllAt(filepath.Join(dst, it))
	}
	in.Lock.Record(lockfile.Entry{
		Domain: domain, Kind: kind, Name: name,
		Mode: lockfile.ModeCopy, Path: dst, Source: src, SHA256: hash, Items: items,
		InstalledAt: time.Now(),
	})
	return nil
}

func (in *Installer) record(e lockfile.Entry) {
	if !in.DryRun {
		in.Lock.Record(e)
	}
}

func (in *Installer) mkdirParent(dst string) error {
	if in.DryRun {
		return nil
	}
	return os.MkdirAll(filepath.Dir(dst), 0o755)
}

func (in *Installer) settleParent(domain string, kind lockfile.Kind, dst string, skip func(string) bool) {
	if in.DryRun {
		return
	}
	if skip == nil {
		in.Lock.RemoveAt(domain, kind, "", dst)
	} else {
		in.keepExcluded(domain, kind, dst, skip)
	}
}

func (in *Installer) keepExcluded(domain string, kind lockfile.Kind, dst string, skip func(string) bool) {
	if in.DryRun {
		return
	}
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
	hash, err := fsys.HashNames(dst, keep)
	if len(keep) == 0 || err != nil {
		in.Lock.RemoveAt(domain, kind, "", dst)
		return
	}
	parent.Items, parent.SHA256 = keep, hash
	in.Lock.Record(parent)
}

func (in *Installer) dropLinksUnder(kind lockfile.Kind, src, dst string) {
	if in.DryRun {
		return
	}
	names, err := fsys.TopNames(src, kind.SkipsReadme())
	if err != nil {
		return
	}
	for _, n := range names {
		p := filepath.Join(dst, n)
		fi, err := os.Lstat(p)
		if err != nil || fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		if e, ok := in.Lock.Owns(p); ok && e.Mode == lockfile.ModeLink {
			if os.Remove(p) == nil {
				in.Lock.RemoveAllAt(p)
			}
		}
	}
}

func (in *Installer) InstallDomain(reg *harness.Registry, name string, strict bool) error {
	return in.InstallAll(reg, []DomainReq{{Name: name, Strict: strict}}, nil)
}

func (in *Installer) InstallSkill(reg *harness.Registry, name string) error {
	return in.InstallAll(reg, nil, []string{name})
}

func (in *Installer) report(scope, kind, dest string, n int) {
	fmt.Fprintln(in.Stdout(), in.reportText(scope, kind, dest, n))
}

func (in *Installer) reportText(scope, kind, dest string, n int) string {
	verb := "installed"
	if in.DryRun {
		verb = "would install"
	}
	plural := "s"
	if n == 1 {
		plural = ""
	}
	return fmt.Sprintf("%-13s %-8s %-7s -> %s (%d file%s)", verb, scope, kind, dest, n, plural)
}
