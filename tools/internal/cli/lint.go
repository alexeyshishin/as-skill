package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"claude-harness/tools/internal/registry"
)

var (
	scriptRefRe   = regexp.MustCompile(`scripts/[A-Za-z0-9_.-]+\.(?:sh|py)`)
	agentRefRe    = regexp.MustCompile(`\.claude/agents/([A-Za-z0-9_.-]+)\.md`)
	registerHdrRe = regexp.MustCompile(`(?m)^#\s*register:\s*\S+`)
)

func plainScalarIssues(path, body string) []string {
	if !strings.HasPrefix(body, "---\n") {
		return []string{fmt.Sprintf("%s: missing frontmatter (file does not start with \"---\")", path)}
	}
	var out []string
	for _, line := range strings.Split(body[4:], "\n") {
		if strings.TrimSpace(line) == "---" {
			break
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok || key == "" || strings.ContainsAny(key[:1], " \t-#") {
			continue
		}
		val = strings.TrimSpace(val)
		if val == "" || strings.ContainsAny(val[:1], ">|\"'[{&*!%@`") {
			continue
		}
		if strings.Contains(val, ": ") || strings.HasSuffix(val, ":") {
			out = append(out, fmt.Sprintf("%s: frontmatter %q is an unquoted value containing \": \" — invalid YAML; use a \">\" block or quotes", path, key))
		}
	}
	return out
}

type lintResult struct {
	checked  int
	failed   int
	findings []string
}

func lintHarness(harnessRoot string) lintResult {
	domainsDir := filepath.Join(harnessRoot, "domains")
	var findings []string
	checked := 0

	agents := map[string]bool{}
	agentFiles, _ := filepath.Glob(filepath.Join(domainsDir, "*", "agents", "*.md"))
	for _, p := range agentFiles {
		agents[strings.TrimSuffix(filepath.Base(p), ".md")] = true
	}

	sort.Strings(agentFiles)
	checked += len(agentFiles)
	for _, p := range agentFiles {
		data, err := os.ReadFile(p)
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s: %v", p, err))
			continue
		}
		findings = append(findings, plainScalarIssues(p, string(data))...)
	}

	skillMDs, _ := filepath.Glob(filepath.Join(domainsDir, "*", "skills", "*", "SKILL.md"))
	sort.Strings(skillMDs)
	checked += len(skillMDs)
	for _, path := range skillMDs {
		skillDir := filepath.Dir(path)
		name, hasName, hasDescription, err := registry.ParseSkillFrontmatter(path)
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if !hasName {
			findings = append(findings, fmt.Sprintf("%s: frontmatter missing \"name:\"", path))
		} else if name != filepath.Base(skillDir) {
			findings = append(findings, fmt.Sprintf("%s: frontmatter name %q does not match directory name %q", path, name, filepath.Base(skillDir)))
		}
		if !hasDescription {
			findings = append(findings, fmt.Sprintf("%s: frontmatter missing \"description:\"", path))
		}

		data, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		body := string(data)
		findings = append(findings, plainScalarIssues(path, body)...)
		seen := map[string]bool{}
		for _, ref := range scriptRefRe.FindAllString(body, -1) {
			if seen[ref] {
				continue
			}
			seen[ref] = true
			if _, err := os.Stat(filepath.Join(skillDir, ref)); err != nil {
				findings = append(findings, fmt.Sprintf("%s: references %s, not found under %s", path, ref, skillDir))
			}
		}
		for _, m := range agentRefRe.FindAllStringSubmatch(body, -1) {
			if seen[m[0]] {
				continue
			}
			seen[m[0]] = true
			if !agents[m[1]] {
				findings = append(findings, fmt.Sprintf("%s: references %s, no domains/*/agents/%s.md", path, m[0], m[1]))
			}
		}
	}

	hooks, _ := filepath.Glob(filepath.Join(domainsDir, "*", "hooks", "*.sh"))
	sort.Strings(hooks)
	for _, p := range hooks {
		if strings.HasSuffix(p, "-lib.sh") {
			continue
		}
		checked++
		data, err := os.ReadFile(p)
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s: %v", p, err))
			continue
		}
		if !registerHdrRe.Match(data) {
			findings = append(findings, fmt.Sprintf("%s: missing \"# register: <Event>\" header", p))
		}
	}

	sort.Strings(findings)
	failedFiles := map[string]bool{}
	for _, f := range findings {
		path, _, _ := strings.Cut(f, ": ")
		failedFiles[path] = true
	}
	return lintResult{checked: checked, failed: len(failedFiles), findings: findings}
}
