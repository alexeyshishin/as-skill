package install

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/fsys"
	"claude-harness/tools/internal/lockfile"
)

const (
	ReasonForeign    = "foreign"
	ReasonNotIgnored = "not-ignored"
)

type ConflictError struct {
	Path   string
	Reason string
}

func (e ConflictError) Error() string {
	if e.Reason == ReasonNotIgnored {
		return fmt.Sprintf("refusing: %s is not git-ignored; add %s to .gitignore or use --copy (or --force)", e.Path, e.Path)
	}
	return (&fsys.ForeignError{Path: e.Path}).Error()
}

func (in *Installer) overwrites(dst string) bool {
	return in.Overwrite[filepath.Clean(dst)]
}

func (in *Installer) tracked(p string) (string, bool) {
	e, ok := in.Lock.Covering(p)
	return string(e.Mode), ok
}

func (in *Installer) linkOne(src, dst string) error {
	prior, err := fsys.LinkOne(src, dst, in.DryRun, in.Force || in.overwrites(dst), in.tracked)
	if prior.Tracked && lockfile.Mode(prior.Mode) != lockfile.ModeLink {
		fmt.Fprintf(in.Stderr(), "as-skill: warning: %s was installed via %q, replacing with a link — mixed channels may cause confusion\n", dst, prior.Mode)
	}
	var fe *fsys.ForeignError
	if errors.As(err, &fe) {
		return &ConflictError{Path: fe.Path, Reason: ReasonForeign}
	}
	return err
}

func (in *Installer) excluded(domain string, kind lockfile.Kind, name string) bool {
	return kind == lockfile.KindSkills && in.ExcludeSkills[domain+"/"+name]
}

func (in *Installer) hasExclusions(domain string) bool {
	prefix := domain + "/"
	for k, v := range in.ExcludeSkills {
		if v && strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}
