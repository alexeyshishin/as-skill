package harness

import (
	"claude-harness/tools/internal/fsys"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claude-harness/tools/internal/frontmatter"
	"claude-harness/tools/internal/lockfile"
)

var DomainKinds = []lockfile.Kind{lockfile.KindRules, lockfile.KindSkills, lockfile.KindAgents, lockfile.KindHooks}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

type Manifest struct {
	Name        string
	Description string
	RequiresEnv []string
	RequiresBin []string
	Targets     map[string]string
}

func ParseManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &Manifest{Targets: map[string]string{}}
	lines := strings.Split(string(data), "\n")

	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "name:"):
			m.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
			i++
		case strings.HasPrefix(trimmed, "description:"):
			m.Description = strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			i++
		case strings.HasPrefix(trimmed, "requires_env:"):
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "requires_env:"))
			i++
			if rest != "" {
				continue
			}
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if !strings.HasPrefix(t, "- ") {
					break
				}
				m.RequiresEnv = append(m.RequiresEnv, strings.TrimSpace(strings.TrimPrefix(t, "- ")))
				i++
			}
		case strings.HasPrefix(trimmed, "requires_bin:"):
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "requires_bin:"))
			i++
			if rest != "" {
				continue
			}
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if !strings.HasPrefix(t, "- ") {
					break
				}
				m.RequiresBin = append(m.RequiresBin, strings.TrimSpace(strings.TrimPrefix(t, "- ")))
				i++
			}
		case strings.HasPrefix(trimmed, "targets:"):
			i++
			for i < len(lines) {
				raw := lines[i]
				t := strings.TrimSpace(raw)
				if t == "" {
					i++
					continue
				}
				if !strings.HasPrefix(raw, " ") && !strings.HasPrefix(raw, "\t") {
					break
				}
				key, val, ok := strings.Cut(t, ":")
				if ok {
					m.Targets[strings.TrimSpace(key)] = strings.TrimSpace(val)
				}
				i++
			}
		default:
			i++
		}
	}
	if m.Name == "" {
		return nil, fmt.Errorf("%s: missing top-level \"name:\"", path)
	}
	return m, nil
}

func ParseSkillFrontmatter(path string) (name string, hasName bool, hasDescription bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, false, err
	}
	doc, err := frontmatter.Parse(data)
	if err != nil {
		return "", false, false, err
	}
	name, _ = doc.String("name")
	name = strings.TrimSpace(name)
	desc, _ := doc.String("description")
	return name, name != "", strings.TrimSpace(desc) != "", nil
}

type Registry struct {
	Domains map[string]*Manifest
	Order   []string
}

func (r *Registry) DomainNames() []string { return r.Order }

func LoadRegistry(harnessRoot string) (*Registry, error) {
	reg := &Registry{Domains: map[string]*Manifest{}}

	domainsDir := DomainsDir(harnessRoot)
	entries, err := os.ReadDir(domainsDir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", domainsDir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(domainsDir, e.Name(), manifestFile)
		if !fileExists(manifestPath) {
			continue
		}
		m, err := ParseManifest(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", manifestPath, err)
		}
		reg.Domains[e.Name()] = m
		reg.Order = append(reg.Order, e.Name())
	}
	sort.Strings(reg.Order)
	if len(reg.Domains) == 0 {
		return nil, fmt.Errorf("no domains with manifest.yaml found under %s", domainsDir)
	}

	return reg, nil
}

func ResolveHarnessRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	dir := abs
	for {
		if dirExists(DomainsDir(dir)) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no claude-harness checkout found at or above %q (looked for domains/) — pass --harness-root", abs)
		}
		dir = parent
	}
}

// Markers that identify a claude-harness checkout.
const (
	harnessModule     = "claude-harness"
	harnessEntrypoint = "tools/main.go"
	manifestFile      = "manifest.yaml"
)

// Locator finds the harness checkout. Executable is os.Executable when nil.
type Locator struct {
	Executable func() (string, error)
}

func (l Locator) executable() (string, error) {
	if l.Executable == nil {
		return os.Executable()
	}
	return l.Executable()
}

type SkillInfo struct {
	Name        string
	Description string
}

func Skills(harnessRoot, domain string) ([]SkillInfo, error) {
	dir := KindDir(harnessRoot, domain, lockfile.KindSkills)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []SkillInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		info := SkillInfo{Name: e.Name()}
		if doc, err := frontmatter.Parse(data); err == nil {
			desc, _ := doc.String("description")
			info.Description = strings.TrimSpace(desc)
		}
		out = append(out, info)
	}
	return out, nil
}

func validHarnessRoot(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	isHarness := false
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[0] == "module" && f[1] == harnessModule {
			isHarness = true
			break
		}
	}
	if !isHarness || !fileExists(filepath.Join(dir, filepath.FromSlash(harnessEntrypoint))) {
		return false
	}
	manifests, _ := filepath.Glob(filepath.Join(DomainDir(dir, "*"), manifestFile))
	return len(manifests) > 0
}

func walkUpHarness(start string) (string, bool) {
	for dir := start; ; {
		if validHarnessRoot(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// Locate resolves the harness root: the flag if set, else cwd walk-up, else the binary's real path.
func (l Locator) Locate(flag string) (string, error) {
	if flag != "" {
		root, err := ResolveHarnessRoot(flag)
		if err != nil {
			return "", err
		}
		if !validHarnessRoot(root) {
			return "", fmt.Errorf("%s is not a claude-harness checkout (need go.mod module claude-harness, tools/main.go, domains/*/manifest.yaml) — pass --harness-root", root)
		}
		return root, nil
	}
	if cwd, err := os.Getwd(); err == nil {
		if root, ok := walkUpHarness(cwd); ok {
			return root, nil
		}
	}
	if exe, err := l.executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			if root, ok := walkUpHarness(filepath.Dir(real)); ok {
				return root, nil
			}
		}
	}
	return "", fmt.Errorf("no claude-harness checkout found from the current directory or the as-skill binary — pass --harness-root")
}

// CwdRoot is the checkout found walking up from the current directory.
func (Locator) CwdRoot() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	return walkUpHarness(cwd)
}

// ExecutableRoot is the checkout found walking up from the real path of the binary.
func (l Locator) ExecutableRoot() (string, bool) {
	exe, err := l.executable()
	if err != nil {
		return "", false
	}
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", false
	}
	return walkUpHarness(filepath.Dir(real))
}

// SkillOwner returns the first domain (in registry order) that ships the skill, or "".
func (r *Registry) SkillOwner(harnessRoot, name string) string {
	for _, d := range r.DomainNames() {
		if fsys.DirExists(SkillDir(harnessRoot, d, name)) {
			return d
		}
	}
	return ""
}

// DomainsDir is <harnessRoot>/domains.
func DomainsDir(harnessRoot string) string { return filepath.Join(harnessRoot, "domains") }

// DomainDir is <harnessRoot>/domains/<domain>.
func DomainDir(harnessRoot, domain string) string {
	return filepath.Join(DomainsDir(harnessRoot), domain)
}

// KindDir is <harnessRoot>/domains/<domain>/<kind>.
func KindDir(harnessRoot, domain string, kind lockfile.Kind) string {
	return filepath.Join(DomainDir(harnessRoot, domain), string(kind))
}

// SkillDir is <harnessRoot>/domains/<domain>/skills/<skill>.
func SkillDir(harnessRoot, domain, skill string) string {
	return filepath.Join(KindDir(harnessRoot, domain, lockfile.KindSkills), skill)
}
