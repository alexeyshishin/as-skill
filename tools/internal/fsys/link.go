package fsys

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"claude-harness/tools/internal/lockfile"
)

type ForeignError struct {
	Path string
}

func (e *ForeignError) Error() string {
	return fmt.Sprintf("%s exists, not tracked by as-skill, use --force", e.Path)
}

func LinkOne(src, dst string, dryRun, force bool, lock *lockfile.Lockfile) error {
	if _, err := os.Lstat(dst); err == nil {
		entry, tracked := lock.Covering(dst)
		if !tracked && !force {
			return &ForeignError{Path: dst}
		}
		if tracked && entry.Mode != lockfile.ModeLink {
			fmt.Fprintf(os.Stderr, "as-skill: warning: %s was installed via %q, replacing with a link — mixed channels may cause confusion\n", dst, entry.Mode)
		}
		if !dryRun {
			if err := os.RemoveAll(dst); err != nil {
				return fmt.Errorf("removing %s: %w", dst, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if dryRun {
		return nil
	}
	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("linking %s -> %s: %w", src, dst, err)
	}
	return nil
}

func LinkTreeWith(domain string, kind lockfile.Kind, src, dstDir string, dryRun bool, lock *lockfile.Lockfile, skip func(name string) bool, link func(src, dst string) error) (int, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return 0, err
	}
	if !dryRun {
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return 0, err
		}
	}
	count := 0
	for _, e := range entries {
		name := e.Name()
		if SkipTop(kind, name) || (skip != nil && skip(name)) {
			continue
		}
		srcPath := filepath.Join(src, name)
		dstPath := filepath.Join(dstDir, name)
		if err := link(srcPath, dstPath); err != nil {
			return count, err
		}
		count++
		if !dryRun {
			lock.Record(lockfile.Entry{
				Domain:      domain,
				Kind:        kind,
				Name:        name,
				Mode:        lockfile.ModeLink,
				Path:        dstPath,
				Source:      srcPath,
				InstalledAt: time.Now(),
			})
		}
	}
	return count, nil
}
