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
	"claude-harness/tools/internal/platform"
)

var (
	scriptRefRe   = regexp.MustCompile(`scripts/[A-Za-z0-9_.-]+\.(?:sh|py)`)
	agentRefRe    = regexp.MustCompile(`\.claude/agents/([A-Za-z0-9_.-]+)\.md`)
	registerHdrRe = regexp.MustCompile(`(?m)^#\s*register:\s*\S+`)
	claudePathRe  = regexp.MustCompile(`subagent_type|AskUserQuestion|~/\.claude/rules|\.claude/rules|\.claude/agents`)
)

type Finding struct {
	Path string
	Msg  string
}

func (f Finding) String() string {
	if f.Path == "" {
		return f.Msg
	}
	return f.Path + ": " + f.Msg
}

const claudePathOK = "<!-- claude-path-ok -->"

func claudePathIssues(path, body string) []Finding {
	var out []Finding
	for i, line := range strings.Split(body, "\n") {
		if strings.HasSuffix(strings.TrimRight(line, " \t\r"), claudePathOK) {
			continue
		}
		if m := claudePathRe.FindString(line); m != "" {
			out = append(out, Finding{path, fmt.Sprintf("line %d: Claude-only reference %q without %s", i+1, m, claudePathOK)})
		}
	}
	return out
}

var commitTypes = map[string]bool{"feat": true, "fix": true, "refactor": true, "perf": true, "docs": true, "test": true, "build": true, "ci": true, "chore": true, "revert": true}

func bareAgentRefIssues(path, body string, stems map[string]bool) []Finding {
	var out []Finding
	seen := map[string]bool{}
	for stem := range stems {
		_, suffix, ok := strings.Cut(stem, "-")
		if !ok || suffix == "" || stems[suffix] || commitTypes[suffix] || seen[suffix] {
			continue
		}
		seen[suffix] = true
		if strings.Contains(body, "`"+suffix+"`") {
			out = append(out, Finding{path, fmt.Sprintf("bare agent name `%s`; use `%s`", suffix, stem)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

func referenceLinkIssues(domainsDir string) []Finding {
	var out []Finding
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
				out = append(out, Finding{p, "dangling symlink"})
				return nil
			}
			if st, err := os.Stat(real); err != nil || !st.Mode().IsRegular() {
				out = append(out, Finding{p, "symlink does not resolve to a regular file"})
				return nil
			}
			ok := false
			for _, a := range allowed {
				ok = ok || fsys.Within(a, real)
			}
			if !ok {
				out = append(out, Finding{p, fmt.Sprintf("symlink escapes its domain (resolves to %s)", real)})
			}
			return nil
		})
	}
	return out
}

func plainScalarIssues(path, body string) []Finding {
	if !strings.HasPrefix(body, "---\n") {
		return []Finding{{path, "missing frontmatter (file does not start with \"---\")"}}
	}
	var out []Finding
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
			out = append(out, Finding{path, fmt.Sprintf("frontmatter %q is an unquoted value containing \": \" — invalid YAML; use a \">\" block or quotes", key)})
		}
	}
	return out
}

type Result struct {
	Checked  int
	Failed   int
	Findings []Finding
}

func Harness(harnessRoot string) Result {
	domainsDir := harness.DomainsDir(harnessRoot)
	var findings []Finding
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
			findings = append(findings, Finding{p, err.Error()})
			continue
		}
		findings = append(findings, plainScalarIssues(p, string(data))...)
		findings = append(findings, claudePathIssues(p, string(data))...)
		findings = append(findings, bareAgentRefIssues(p, string(data), agents)...)
		stem := strings.TrimSuffix(filepath.Base(p), ".md")
		name := stem
		if doc, err := frontmatter.Parse(data); err == nil {
			if v, ok := doc.String("name"); ok {
				if v != stem {
					findings = append(findings, Finding{p, fmt.Sprintf("agent name %q does not match filename %q", v, stem)})
				}
				if v != "" {
					name = v
				}
			}
		}
		if prev, dup := agentNames[name]; dup {
			findings = append(findings, Finding{p, fmt.Sprintf("agent name %q duplicates %s", name, prev)})
		} else {
			agentNames[name] = p
		}
		for _, id := range platform.IDs() {
			pl, _ := platform.Get(id)
			for _, r := range pl.ReservedAgentNames {
				if r == name {
					findings = append(findings, Finding{p, fmt.Sprintf("agent name %q is reserved on %s", name, id)})
				}
			}
		}
	}

	skillMDs, _ := filepath.Glob(filepath.Join(domainsDir, "*", "skills", "*", "SKILL.md"))
	sort.Strings(skillMDs)
	checked += len(skillMDs)
	skillDirs := map[string]string{}
	domainSet := map[string]bool{}
	if des, err := os.ReadDir(domainsDir); err == nil {
		for _, de := range des {
			if de.IsDir() {
				domainSet[de.Name()] = true
			}
		}
	}
	for _, path := range skillMDs {
		skillDir := filepath.Dir(path)
		name, hasName, hasDescription, err := harness.ParseSkillFrontmatter(path)
		if err != nil {
			raw, _ := os.ReadFile(path)
			if issues := plainScalarIssues(path, string(raw)); len(issues) > 0 {
				findings = append(findings, issues...)
			} else {
				findings = append(findings, Finding{path, err.Error()})
			}
			continue
		}
		if !hasName {
			findings = append(findings, Finding{path, "frontmatter missing \"name:\""})
		} else if name != filepath.Base(skillDir) {
			findings = append(findings, Finding{path, fmt.Sprintf("frontmatter name %q does not match directory name %q", name, filepath.Base(skillDir))})
		}
		if !hasDescription {
			findings = append(findings, Finding{path, "frontmatter missing \"description:\""})
		}
		dirName := filepath.Base(skillDir)
		domainName := filepath.Base(filepath.Dir(filepath.Dir(skillDir)))
		if prev, dup := skillDirs[dirName]; dup {
			findings = append(findings, Finding{path, fmt.Sprintf("skill dir name %q duplicates %s", dirName, prev)})
		} else {
			skillDirs[dirName] = path
		}
		if dirName == "all" {
			findings = append(findings, Finding{path, fmt.Sprintf("skill dir name %q is reserved (selects every domain)", dirName)})
		}
		if _, isDomain := domainSet[dirName]; isDomain {
			findings = append(findings, Finding{path, fmt.Sprintf("skill dir name %q collides with a domain name (domain %s)", dirName, domainName)})
		}

		data, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, Finding{path, err.Error()})
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
				findings = append(findings, Finding{path, fmt.Sprintf("references %s, not found under %s", ref, skillDir)})
			}
		}
		for _, m := range agentRefRe.FindAllStringSubmatch(body, -1) {
			if seen[m[0]] {
				continue
			}
			seen[m[0]] = true
			if !agents[m[1]] {
				findings = append(findings, Finding{path, fmt.Sprintf("references %s, no domains/*/agents/%s.md", m[0], m[1])})
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
			findings = append(findings, Finding{p, err.Error()})
			continue
		}
		if !registerHdrRe.Match(data) {
			findings = append(findings, Finding{p, "missing \"# register: <Event>\" header"})
		}
	}

	sort.Slice(findings, func(i, j int) bool { return findings[i].String() < findings[j].String() })
	failedFiles := map[string]bool{}
	for _, f := range findings {
		path, _, _ := strings.Cut(f.String(), ": ")
		failedFiles[path] = true
	}
	return Result{Checked: checked, Failed: len(failedFiles), Findings: findings}
}
