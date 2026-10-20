package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	healthOK         = "OK"
	healthMissing    = "MISSING"
	healthBroken     = "BROKEN"
	healthSourceGone = "SOURCE-GONE"
	healthDrifted    = "DRIFTED"
	healthUntracked  = "UNTRACKED"
)

type statusOpts struct {
	project     string
	harnessRoot string
}

func parseStatusArgs(args []string) (statusOpts, error) {
	opts := statusOpts{project: ".", harnessRoot: "."}
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasVal := a, "", false
		if strings.HasPrefix(a, "--") {
			if before, after, found := strings.Cut(a, "="); found {
				key, val, hasVal = before, after, true
			}
		}
		takeValue := func() (string, error) {
			if hasVal {
				return val, nil
			}
			i++
			if i >= len(args) {
				return "", fmt.Errorf("%s needs a value", key)
			}
			return args[i], nil
		}
		switch key {
		case "--project", "-p":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.project = v
		case "--harness-root":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.harnessRoot = v
		default:
			return opts, fmt.Errorf("unknown flag %q", a)
		}
	}
	return opts, nil
}

func entryHealth(e lockEntry) string {
	fi, err := os.Lstat(e.Path)
	if err != nil {
		return healthMissing
	}
	if e.Mode == "link" {
		if fi.Mode()&os.ModeSymlink == 0 {
			return healthBroken
		}
		target, err := os.Readlink(e.Path)
		if err != nil || target != e.Source {
			return healthBroken
		}
	}
	return healthOK
}

func sortedEntries(lock *lockfile) []lockEntry {
	entries := append([]lockEntry(nil), lock.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Domain != entries[j].Domain {
			return entries[i].Domain < entries[j].Domain
		}
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Name < entries[j].Name
	})
	return entries
}

func printEntryTable(entries []lockEntry, healthOf func(lockEntry) string) {
	if len(entries) == 0 {
		fmt.Println("no entries tracked in skills-lock.json")
		return
	}
	fmt.Printf("%-10s %-8s %-24s %-6s %-11s %s\n", "DOMAIN", "KIND", "NAME", "MODE", "HEALTH", "PATH")
	for _, e := range entries {
		fmt.Printf("%-10s %-8s %-24s %-6s %-11s %s\n", e.Domain, e.Kind, e.Name, e.Mode, healthOf(e), e.Path)
	}
}

func runStatus(args []string) error {
	opts, err := parseStatusArgs(args)
	if err != nil {
		return err
	}
	projectRoot, err := filepath.Abs(opts.project)
	if err != nil {
		return err
	}
	lock, err := loadLockfile(projectRoot)
	if err != nil {
		return err
	}
	printEntryTable(sortedEntries(lock), entryHealth)
	return nil
}

func runDoctor(args []string) error {
	opts, err := parseStatusArgs(args)
	if err != nil {
		return err
	}
	harnessRoot, err := resolveHarnessRoot(opts.harnessRoot)
	if err != nil {
		return err
	}
	projectRoot, err := filepath.Abs(opts.project)
	if err != nil {
		return err
	}
	lock, err := loadLockfile(projectRoot)
	if err != nil {
		return err
	}

	entries := sortedEntries(lock)
	problems := 0
	if len(entries) == 0 {
		fmt.Println("no entries tracked in skills-lock.json")
	} else {
		fmt.Printf("%-10s %-8s %-24s %-6s %-11s %s\n", "DOMAIN", "KIND", "NAME", "MODE", "HEALTH", "PATH")
	}
	for _, e := range entries {
		health := entryHealth(e)
		fmt.Printf("%-10s %-8s %-24s %-6s %-11s %s\n", e.Domain, e.Kind, e.Name, e.Mode, health, e.Path)
		if health == healthMissing || health == healthBroken {
			problems++
		}

		switch e.Mode {
		case "link":
			if !pathExists(e.Source) {
				fmt.Printf("as-skill: doctor: %-11s %s/%s/%s — source %s no longer exists under harness root %s\n",
					healthSourceGone, e.Domain, e.Kind, e.Name, e.Source, harnessRoot)
				problems++
			}
		case "copy":
			if health == healthMissing {
				continue
			}
			hash, herr := hashTree(e.Path)
			if herr != nil {
				fmt.Printf("as-skill: doctor: warning: %s/%s/%s — could not hash %s: %v\n", e.Domain, e.Kind, e.Name, e.Path, herr)
				continue
			}
			if e.SHA256 != "" && hash != e.SHA256 {
				fmt.Printf("as-skill: doctor: %-11s %s/%s/%s — recomputed hash differs from the recorded SHA256 (ambiguous: the target may have been hand-edited, or the source in domains/ may have moved on since install — not disambiguated)\n",
					healthDrifted, e.Domain, e.Kind, e.Name)
				problems++
			}
		}
	}

	untracked, err := findUntracked(projectRoot, lock)
	if err != nil {
		return err
	}
	for _, p := range untracked {
		fmt.Printf("as-skill: doctor: %-11s %s — present on disk under .claude/, no lockfile entry\n", healthUntracked, p)
		problems++
	}

	if problems > 0 {
		return fmt.Errorf("doctor: %d finding(s)", problems)
	}
	return nil
}

func findUntracked(projectRoot string, lock *lockfile) ([]string, error) {
	tracked := map[string]bool{}
	for _, e := range lock.Entries {
		tracked[filepath.Clean(e.Path)] = true
	}
	var untracked []string
	for _, kind := range domainKinds {
		dir := filepath.Join(projectRoot, ".claude", kind)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if !tracked[filepath.Clean(p)] {
				untracked = append(untracked, p)
			}
		}
	}
	sort.Strings(untracked)
	return untracked, nil
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func runCheck(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("check takes no arguments (got %s)", strings.Join(args, " "))
	}
	harnessRoot, err := resolveHarnessRoot(".")
	if err != nil {
		return err
	}

	var failures []string

	domainsDir := filepath.Join(harnessRoot, "domains")
	domainEntries, err := os.ReadDir(domainsDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", domainsDir, err)
	}
	var domainNames []string
	for _, de := range domainEntries {
		if !de.IsDir() {
			continue
		}
		domainNames = append(domainNames, de.Name())
		manifestPath := filepath.Join(domainsDir, de.Name(), "manifest.yaml")
		if !fileExists(manifestPath) {
			failures = append(failures, fmt.Sprintf("%s: missing manifest.yaml", manifestPath))
			continue
		}
		m, err := parseManifest(manifestPath)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		if strings.TrimSpace(m.Description) == "" {
			failures = append(failures, fmt.Sprintf("%s: missing top-level \"description:\"", manifestPath))
		}
	}
	sort.Strings(domainNames)

	var skillMDs []string
	for _, name := range domainNames {
		matches, _ := filepath.Glob(filepath.Join(domainsDir, name, "skills", "*", "SKILL.md"))
		skillMDs = append(skillMDs, matches...)
	}
	coreMatches, _ := filepath.Glob(filepath.Join(harnessRoot, "core", "skills", "*", "SKILL.md"))
	skillMDs = append(skillMDs, coreMatches...)
	sort.Strings(skillMDs)

	for _, path := range skillMDs {
		dirName := filepath.Base(filepath.Dir(path))
		name, hasName, hasDescription, err := parseSkillFrontmatter(path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if !hasName {
			failures = append(failures, fmt.Sprintf("%s: frontmatter missing \"name:\"", path))
		} else if name != dirName {
			failures = append(failures, fmt.Sprintf("%s: frontmatter name %q does not match directory name %q", path, name, dirName))
		}
		if !hasDescription {
			failures = append(failures, fmt.Sprintf("%s: frontmatter missing \"description:\"", path))
		}
	}

	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintln(os.Stderr, "as-skill: check:", f)
		}
		return fmt.Errorf("check: %d failure(s)", len(failures))
	}
	fmt.Printf("check: OK — %d manifest(s), %d SKILL.md(s)\n", len(domainNames), len(skillMDs))
	return nil
}

func parseSkillFrontmatter(path string) (name string, hasName bool, hasDescription bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, false, err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", false, false, fmt.Errorf("missing frontmatter (file does not start with \"---\")")
	}
	i := 1
	for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
		trimmed := strings.TrimSpace(lines[i])
		switch {
		case strings.HasPrefix(trimmed, "name:"):
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
			hasName = name != ""
		case strings.HasPrefix(trimmed, "description:"):
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			hasDescription = rest != ""
		}
		i++
	}
	if i >= len(lines) {
		return name, hasName, hasDescription, fmt.Errorf("frontmatter not terminated with a closing \"---\"")
	}
	return name, hasName, hasDescription, nil
}
