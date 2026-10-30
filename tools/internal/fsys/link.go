package fsys

import (
	"fmt"
	"os"
	"path/filepath"
)

type ForeignError struct {
	Path string
}

func (e *ForeignError) Error() string {
	return fmt.Sprintf("%s exists, not tracked by as-skill, use --force", e.Path)
}

type Prior struct {
	Tracked bool
	Mode    string
}

type Linked struct {
	Name string
	Src  string
	Dst  string
}

func LinkOne(src, dst string, dryRun, force bool, tracked func(dst string) (mode string, ok bool)) (Prior, error) {
	var prior Prior
	if _, err := os.Lstat(dst); err == nil {
		mode, ok := tracked(dst)
		prior = Prior{Tracked: ok, Mode: mode}
		if !ok && !force {
			return prior, &ForeignError{Path: dst}
		}
		if !dryRun {
			if err := os.RemoveAll(dst); err != nil {
				return prior, fmt.Errorf("removing %s: %w", dst, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return prior, err
	}

	if dryRun {
		return prior, nil
	}
	if err := os.Symlink(src, dst); err != nil {
		return prior, fmt.Errorf("linking %s -> %s: %w", src, dst, err)
	}
	return prior, nil
}

func LinkTreeWith(skipReadme bool, src, dstDir string, dryRun bool, skip func(name string) bool, link func(src, dst string) error) ([]Linked, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return nil, err
	}
	if !dryRun {
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return nil, err
		}
	}
	var done []Linked
	for _, e := range entries {
		name := e.Name()
		if SkipTop(skipReadme, name) || (skip != nil && skip(name)) {
			continue
		}
		srcPath := filepath.Join(src, name)
		dstPath := filepath.Join(dstDir, name)
		if err := link(srcPath, dstPath); err != nil {
			return done, err
		}
		done = append(done, Linked{Name: name, Src: srcPath, Dst: dstPath})
	}
	return done, nil
}
