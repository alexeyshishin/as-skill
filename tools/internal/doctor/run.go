package doctor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/fsys"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

const (
	CannotVerify = "CANNOT-VERIFY"
	HooksTrust   = "HOOKS-TRUST"
)

type Finding struct {
	Severity Severity
	Code     string
	Entry    string
	Msg      string
}

type Input struct {
	ProjectRoot string
	HarnessRoot string
	HarnessOK   bool
	Lock        *lockfile.Lockfile
	Entries     []lockfile.Entry
}

type Result struct {
	Findings []Finding
	Checked  int
	Failed   int
}

func entryLabel(e lockfile.Entry) string {
	return fmt.Sprintf("%s/%s/%s", e.Domain, e.Kind, e.Name)
}

func Run(in Input) (Result, error) {
	var res Result
	add := func(sev Severity, code, entry, msg string) {
		res.Findings = append(res.Findings, Finding{Severity: sev, Code: code, Entry: entry, Msg: msg})
	}
	for _, e := range in.Entries {
		res.Checked++
		label := entryLabel(e)
		h := EntryHealth(e)
		failed := h == HealthMissing || h == HealthBroken
		if failed {
			add(SeverityError, h, label, e.Path)
		}
		switch e.Mode {
		case lockfile.ModeLink:
			if in.HarnessOK && !fsys.PathExists(e.Source) {
				add(SeverityError, HealthSourceGone, label, fmt.Sprintf("source %s no longer exists under harness root %s", e.Source, in.HarnessRoot))
				failed = true
			}
		case lockfile.ModeRender:
			if h == HealthMissing {
				break
			}
			stale, drifted := RenderState(e)
			if stale {
				add(SeverityError, HealthStale, label, fmt.Sprintf("source %s changed since render; reinstall to refresh %s", e.Source, e.Path))
				failed = true
			}
			if drifted {
				add(SeverityError, HealthDrifted, label, fmt.Sprintf("%s was edited after render (generated file, not meant for hand edits)", e.Path))
				failed = true
			}
		case lockfile.ModeCopy:
			if h == HealthMissing || e.SHA256 == "" {
				break
			}
			match, merr := CopyMatchesIn(in.Lock, e)
			if errors.Is(merr, ErrCannotVerify) {
				add(SeverityWarning, CannotVerify, label, "no recorded item list and the source checkout is gone; reinstall once with the current binary to fix")
				break
			}
			if merr != nil {
				add(SeverityWarning, "", label, fmt.Sprintf("could not hash %s: %v", e.Path, merr))
				break
			}
			if !match {
				add(SeverityError, HealthDrifted, label, "recomputed hash differs from the recorded SHA256 (ambiguous: the target may have been hand-edited, or the source in domains/ may have moved on since install — not disambiguated)")
				failed = true
			}
		}
		if failed {
			res.Failed++
		}
	}

	untracked, err := FindUntracked(in.ProjectRoot, in.Lock)
	if err != nil {
		return res, err
	}
	for _, p := range untracked {
		add(SeverityError, HealthUntracked, p, "present on disk, no lockfile entry")
	}
	res.Checked += len(untracked)
	res.Failed += len(untracked)

	for _, d := range Duplicates(in.ProjectRoot, in.Lock) {
		add(SeverityWarning, DuplicateSkill, "", d)
	}
	for _, u := range UnverifiedCaps(in.Lock) {
		add(SeverityWarning, Unverified, "", u)
	}
	if CodexHooksNeedTrust(in.ProjectRoot, in.Lock) {
		add(SeverityWarning, HooksTrust, "", "codex hooks need trust: review in /hooks")
	}
	return res, nil
}

func CodexHooksNeedTrust(projectRoot string, lock *lockfile.Lockfile) bool {
	has := false
	for _, p := range LockPlatforms(lock) {
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
