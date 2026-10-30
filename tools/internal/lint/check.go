package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"claude-harness/tools/internal/frontmatter"
	"claude-harness/tools/internal/fsys"
	"claude-harness/tools/internal/harness"
)

type CheckResult struct {
	Domains  int
	Skills   int
	Findings []Finding
}

func Check(harnessRoot string) (CheckResult, error) {
	var failures []Finding

	domainsDir := harness.DomainsDir(harnessRoot)
	domainEntries, err := os.ReadDir(domainsDir)
	if err != nil {
		return CheckResult{}, fmt.Errorf("reading %s: %w", domainsDir, err)
	}
	var domainNames []string
	for _, de := range domainEntries {
		if !de.IsDir() {
			continue
		}
		domainNames = append(domainNames, de.Name())
		manifestPath := filepath.Join(domainsDir, de.Name(), "manifest.yaml")
		if !fsys.FileExists(manifestPath) {
			failures = append(failures, Finding{manifestPath, "missing manifest.yaml"})
			continue
		}
		m, err := harness.ParseManifest(manifestPath)
		if err != nil {
			failures = append(failures, Finding{Msg: err.Error()})
			continue
		}
		if strings.TrimSpace(m.Description) == "" {
			failures = append(failures, Finding{manifestPath, "missing top-level \"description:\""})
		}
	}
	sort.Strings(domainNames)

	var skillMDs []string
	for _, name := range domainNames {
		matches, _ := filepath.Glob(filepath.Join(domainsDir, name, "skills", "*", "SKILL.md"))
		skillMDs = append(skillMDs, matches...)
	}
	sort.Strings(skillMDs)

	for _, path := range skillMDs {
		for _, msg := range checkSkillSpec(path) {
			failures = append(failures, Finding{path, msg})
		}
	}

	failures = append(failures, Harness(harnessRoot).Findings...)

	return CheckResult{Domains: len(domainNames), Skills: len(skillMDs), Findings: dedupe(failures)}, nil
}

func dedupe(in []Finding) []Finding {
	seen := map[Finding]bool{}
	var out []Finding
	for _, f := range in {
		if seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

var skillNameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var skillAllowedKeys = map[string]bool{
	"name": true, "description": true, "license": true, "compatibility": true,
	"metadata": true, "allowed-tools": true,
	"argument-hint": true,
}

func checkSkillSpec(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{err.Error()}
	}
	doc, err := frontmatter.Parse(data)
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	for _, k := range doc.Keys {
		if !skillAllowedKeys[k] {
			out = append(out, fmt.Sprintf("line %d: frontmatter key %q is not in the Agent Skills spec (allowed: name, description, license, compatibility, metadata, allowed-tools, argument-hint)", doc.Fields[k].Line, k))
		}
	}
	dirName := filepath.Base(filepath.Dir(path))
	name, hasName := doc.String("name")
	name = strings.TrimSpace(name)
	switch {
	case !hasName || name == "":
		out = append(out, "frontmatter missing \"name:\"")
	case name != dirName:
		out = append(out, fmt.Sprintf("frontmatter name %q does not match directory name %q", name, dirName))
	case !skillNameRe.MatchString(name):
		out = append(out, fmt.Sprintf("name %q must match [a-z0-9-] without leading, trailing or double hyphens", name))
	case len(name) > 64:
		out = append(out, fmt.Sprintf("name is %d characters, max 64", len(name)))
	}
	desc, hasDesc := doc.String("description")
	desc = strings.TrimSpace(desc)
	if !hasDesc || desc == "" {
		out = append(out, "frontmatter missing \"description:\"")
	} else if n := len([]rune(desc)); n > 1024 {
		out = append(out, fmt.Sprintf("description is %d characters, max 1024", n))
	}
	if v, ok := doc.Fields["allowed-tools"]; ok && v.Kind != frontmatter.Scalar {
		out = append(out, fmt.Sprintf("line %d: allowed-tools must be a string (comma-separated), not a list or map", v.Line))
	}
	if v, ok := doc.Fields["metadata"]; ok && v.Kind != frontmatter.Map {
		out = append(out, fmt.Sprintf("line %d: metadata must be a map of string values", v.Line))
	}
	return out
}
