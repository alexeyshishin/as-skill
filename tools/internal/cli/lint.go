package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"claude-harness/tools/internal/frontmatter"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/registry"
)

var (
	scriptRefRe   = regexp.MustCompile(`scripts/[A-Za-z0-9_.-]+\.(?:sh|py)`)
	agentRefRe    = regexp.MustCompile(`\.claude/agents/([A-Za-z0-9_.-]+)\.md`)
	registerHdrRe = regexp.MustCompile(`(?m)^#\s*register:\s*\S+`)
	claudePathRe  = regexp.MustCompile(`subagent_type|AskUserQuestion|~/\.claude/rules|\.claude/rules|\.claude/agents`)
)

const claudePathOK = "<!-- claude-path-ok -->"

func claudePathIssues(path, body string) []string {
	var out []string
	for i, line := range strings.Split(body, "\n") {
		if strings.HasSuffix(strings.TrimRight(line, " \t\r"), claudePathOK) {
			continue
		}
		if m := claudePathRe.FindString(line); m != "" {
			out = append(out, fmt.Sprintf("%s: line %d: Claude-only reference %q without %s", path, i+1, m, claudePathOK))
		}
	}
	return out
}

var commitTypes = map[string]bool{"feat": true, "fix": true, "refactor": true, "perf": true, "docs": true, "test": true, "build": true, "ci": true, "chore": true, "revert": true}

func bareAgentRefIssues(path, body string, stems map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	for stem := range stems {
		_, suffix, ok := strings.Cut(stem, "-")
		if !ok || suffix == "" || stems[suffix] || commitTypes[suffix] || seen[suffix] {
			continue
		}
		seen[suffix] = true
		if strings.Contains(body, "`"+suffix+"`") {
			out = append(out, fmt.Sprintf("%s: bare agent name `%s`; use `%s`", path, suffix, stem))
		}
	}
	sort.Strings(out)
	return out
}

func within(root, p string) bool {
	r, err := filepath.Rel(root, p)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

func referenceLinkIssues(domainsDir string) []string {
	var out []string
	links, _ := filepath.Glob(filepath.Join(domainsDir, "*", "skills", "*", "references"))
	for _, refDir := range links {
		domainDir := filepath.Dir(filepath.Dir(filepath.Dir(refDir)))
		allowed := []string{domainDir}
		for i, a := range allowed {
			if r, err := filepath.EvalSymlinks(a); err == nil {
				allowed[i] = r
			}
		}
		_ = filepath.WalkDir(refDir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.Type()&os.ModeSymlink == 0 {
				return nil
			}
			real, err := filepath.EvalSymlinks(p)
			if err != nil {
				out = append(out, fmt.Sprintf("%s: dangling symlink", p))
				return nil
			}
			if st, err := os.Stat(real); err != nil || !st.Mode().IsRegular() {
				out = append(out, fmt.Sprintf("%s: symlink does not resolve to a regular file", p))
				return nil
			}
			ok := false
			for _, a := range allowed {
				ok = ok || within(a, real)
			}
			if !ok {
				out = append(out, fmt.Sprintf("%s: symlink escapes its domain (resolves to %s)", p, real))
			}
			return nil
		})
	}
	return out
}

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
	agentNames := map[string]string{}
	checked += len(agentFiles)
	for _, p := range agentFiles {
		data, err := os.ReadFile(p)
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s: %v", p, err))
			continue
		}
		findings = append(findings, plainScalarIssues(p, string(data))...)
		findings = append(findings, claudePathIssues(p, string(data))...)
		findings = append(findings, bareAgentRefIssues(p, string(data), agents)...)
		if doc, err := frontmatter.Parse(data); err == nil {
			if v, ok := doc.String("name"); ok && v != strings.TrimSuffix(filepath.Base(p), ".md") {
				findings = append(findings, fmt.Sprintf("%s: agent name %q does not match filename %q", p, v, strings.TrimSuffix(filepath.Base(p), ".md")))
			}
		}
		name := strings.TrimSuffix(filepath.Base(p), ".md")
		if doc, err := frontmatter.Parse(data); err == nil {
			if v, ok := doc.String("name"); ok && v != "" {
				name = v
			}
		}
		if prev, dup := agentNames[name]; dup {
			findings = append(findings, fmt.Sprintf("%s: agent name %q duplicates %s", p, name, prev))
		} else {
			agentNames[name] = p
		}
		for _, id := range platform.IDs() {
			pl, _ := platform.Get(id)
			for _, r := range pl.ReservedAgentNames {
				if r == name {
					findings = append(findings, fmt.Sprintf("%s: agent name %q is reserved on %s", p, name, id))
				}
			}
		}
	}

	skillMDs, _ := filepath.Glob(filepath.Join(domainsDir, "*", "skills", "*", "SKILL.md"))
	sort.Strings(skillMDs)
	checked += len(skillMDs)
	for _, path := range skillMDs {
		skillDir := filepath.Dir(path)
		name, hasName, hasDescription, err := registry.ParseSkillFrontmatter(path)
		if err != nil {
			raw, _ := os.ReadFile(path)
			if issues := plainScalarIssues(path, string(raw)); len(issues) > 0 {
				findings = append(findings, issues...)
			} else {
				findings = append(findings, fmt.Sprintf("%s: %v", path, err))
			}
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
		findings = append(findings, claudePathIssues(path, body)...)
		findings = append(findings, bareAgentRefIssues(path, body, agents)...)
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

	findings = append(findings, referenceLinkIssues(domainsDir)...)

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
