package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claude-harness/tools/internal/transfer"
)

const (
	ReasonForeign    = "foreign"
	ReasonNotIgnored = "not-ignored"
)

// ConflictError is a recoverable install refusal. For ReasonForeign Path is the
// untracked destination (usable as an Overwrite key); for ReasonNotIgnored it is
// the platform root (".cursor", ".agents", ...) that is not git-ignored.
type ConflictError struct {
	Path   string
	Reason string
}

func (e ConflictError) Error() string {
	if e.Reason == ReasonNotIgnored {
		return fmt.Sprintf("refusing: %s is not git-ignored; add %s to .gitignore or use --copy (or --force)", e.Path, e.Path)
	}
	return (&transfer.ForeignError{Path: e.Path}).Error()
}

// Collect runs run (the caller's real Preflight/InstallDomain/InstallSkill
// sequence) as a quiet dry pass: nothing is written to disk or the lockfile,
// stdout/stderr are silenced, and recoverable refusals are gathered instead of
// aborting. Reserved agent names and other failures still come back as err.
func (in *Installer) Collect(run func() error) ([]ConflictError, error) {
	savedDry, savedDup, savedGaps := in.DryRun, in.dupWarned, in.gapsDone
	in.DryRun, in.collecting, in.conflicts = true, true, nil
	restoreIO := silenceIO()
	err := run()
	restoreIO()
	found := in.conflicts
	in.DryRun, in.collecting, in.conflicts = savedDry, false, nil
	in.dupWarned, in.gapsDone = savedDup, savedGaps
	sort.Slice(found, func(i, j int) bool {
		if found[i].Reason != found[j].Reason {
			return found[i].Reason < found[j].Reason
		}
		return found[i].Path < found[j].Path
	})
	return found, err
}

func silenceIO() func() {
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return func() {}
	}
	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = null, null
	return func() {
		os.Stdout, os.Stderr = stdout, stderr
		null.Close()
	}
}

func (in *Installer) conflict(path, reason string) error {
	c := ConflictError{Path: path, Reason: reason}
	if !in.collecting {
		return &c
	}
	for _, x := range in.conflicts {
		if x == c {
			return nil
		}
	}
	in.conflicts = append(in.conflicts, c)
	return nil
}

func (in *Installer) overwrites(dst string) bool {
	return in.Overwrite[filepath.Clean(dst)]
}

func (in *Installer) linkOne(src, dst string) error {
	err := transfer.LinkOne(src, dst, in.DryRun, in.Force || in.overwrites(dst), in.Lock)
	var fe *transfer.ForeignError
	if errors.As(err, &fe) {
		return in.conflict(fe.Path, ReasonForeign)
	}
	return err
}

func (in *Installer) excluded(domain, kind, name string) bool {
	return kind == "skills" && in.ExcludeSkills[domain+"/"+name]
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
