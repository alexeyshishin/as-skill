package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "as-skill: error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return errors.New("no command given")
	}
	switch args[0] {
	case "install":
		return runInstall(args[1:])
	case "list":
		return runList(args[1:])
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `as-skill — install claude-harness domains and skills into a project's .claude/

Usage:
  as-skill install domain  <name>              install one domain
  as-skill install domains <name> [name...]    install several domains
  as-skill install all                         install every domain + core skills
  as-skill install skill   <name>              install one skill (domain-owned or core)
  as-skill list [domains|skills]                show what's installable

Flags (install only):
  --project PATH        target project root, gets a .claude/ (default ".")
  --harness-root PATH   this repo's checkout (default: auto-detected upward from ".")
  --with-core            also install core/skills/* (domain/domains/skill modes;
                          "all" always includes them)
  --dry-run              print what would be copied, write nothing

Examples:
  as-skill install domain git --project ~/code/my-app
  as-skill install domains git content --project .
  as-skill install all --project ~/code/my-app
  as-skill install skill obsidian-ingest --project .
`)
}

type manifest struct {
	Name        string
	Description string
	RequiresEnv []string
	Targets     map[string]string // "rules" | "skills" | "agents" -> raw path, may hold ${VAR}
}

func parseManifest(path string) (*manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &manifest{Targets: map[string]string{}}
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
				continue // inline value (e.g. "[]") — nothing more to consume
			}
			for i < len(lines) {
				t := strings.TrimSpace(lines[i])
				if !strings.HasPrefix(t, "- ") {
					break
				}
				m.RequiresEnv = append(m.RequiresEnv, strings.TrimSpace(strings.TrimPrefix(t, "- ")))
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
					break // dedented — targets: block ended
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

type registry struct {
	domains    map[string]*manifest
	order      []string // domain names, sorted
	coreSkills []string // core/skills/* names, sorted
}

func (r *registry) domainNames() []string { return r.order }

func loadRegistry(harnessRoot string) (*registry, error) {
	reg := &registry{domains: map[string]*manifest{}}

	domainsDir := filepath.Join(harnessRoot, "domains")
	entries, err := os.ReadDir(domainsDir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", domainsDir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(domainsDir, e.Name(), "manifest.yaml")
		if !fileExists(manifestPath) {
			continue
		}
		m, err := parseManifest(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", manifestPath, err)
		}
		reg.domains[e.Name()] = m
		reg.order = append(reg.order, e.Name())
	}
	sort.Strings(reg.order)
	if len(reg.domains) == 0 {
		return nil, fmt.Errorf("no domains with manifest.yaml found under %s", domainsDir)
	}

	coreDir := filepath.Join(harnessRoot, "core", "skills")
	if entries, err := os.ReadDir(coreDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && fileExists(filepath.Join(coreDir, e.Name(), "SKILL.md")) {
				reg.coreSkills = append(reg.coreSkills, e.Name())
			}
		}
		sort.Strings(reg.coreSkills)
	}

	return reg, nil
}

func resolveHarnessRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	dir := abs
	for {
		if dirExists(filepath.Join(dir, "domains")) && dirExists(filepath.Join(dir, "core", "skills")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no claude-harness checkout found at or above %q (looked for domains/ + core/skills/) — pass --harness-root", abs)
		}
		dir = parent
	}
}

const claudeHomeVar = "CLAUDE_HOME"

var domainKinds = []string{"rules", "skills", "agents", "hooks"} // hooks has no manifest target; defaults below

type installer struct {
	harnessRoot string
	projectRoot string
	dryRun      bool
}

func (in *installer) installDomain(reg *registry, name string, strict bool) error {
	m, ok := reg.domains[name]
	if !ok {
		return fmt.Errorf("unknown domain %q (available: %s)", name, strings.Join(reg.domainNames(), ", "))
	}
	for _, envVar := range m.RequiresEnv {
		if os.Getenv(envVar) != "" {
			continue
		}
		msg := fmt.Sprintf("domain %q requires $%s, which is not set", name, envVar)
		if strict {
			return errors.New(msg)
		}
		fmt.Fprintf(os.Stderr, "as-skill: skipping — %s\n", msg)
		return nil
	}
	for _, kind := range domainKinds {
		srcDir := filepath.Join(in.harnessRoot, "domains", name, kind)
		if !dirExists(srcDir) {
			continue
		}
		destPath, err := in.resolveTarget(m, kind)
		if err != nil {
			return fmt.Errorf("domain %q: %w", name, err)
		}
		n, err := copyTree(srcDir, destPath, in.dryRun)
		if err != nil {
			return fmt.Errorf("domain %q: copying %s: %w", name, kind, err)
		}
		in.report(name, kind, destPath, n)
		if kind == "hooks" && !in.dryRun {
			if err := makeShellScriptsExecutable(destPath); err != nil {
				return fmt.Errorf("domain %q: chmod hooks: %w", name, err)
			}
		}
	}
	return nil
}

func (in *installer) installSkill(reg *registry, name string) error {
	for _, cs := range reg.coreSkills {
		if cs != name {
			continue
		}
		src := filepath.Join(in.harnessRoot, "core", "skills", name)
		dest := filepath.Join(in.projectRoot, ".claude", "skills", name)
		n, err := copyTree(src, dest, in.dryRun)
		if err != nil {
			return fmt.Errorf("core skill %q: %w", name, err)
		}
		in.report("core", "skill", dest, n)
		return nil
	}

	owner := ""
	for _, dname := range reg.domainNames() {
		if dirExists(filepath.Join(in.harnessRoot, "domains", dname, "skills", name)) {
			owner = dname
			break
		}
	}
	if owner == "" {
		return fmt.Errorf("unknown skill %q (see `as-skill list skills`)", name)
	}
	m := reg.domains[owner]
	for _, envVar := range m.RequiresEnv {
		if os.Getenv(envVar) == "" {
			return fmt.Errorf("skill %q belongs to domain %q, which requires $%s (not set)", name, owner, envVar)
		}
	}
	destBase, err := in.resolveTarget(m, "skills")
	if err != nil {
		return fmt.Errorf("skill %q: %w", name, err)
	}
	dest := filepath.Join(destBase, name)
	src := filepath.Join(in.harnessRoot, "domains", owner, "skills", name)
	n, err := copyTree(src, dest, in.dryRun)
	if err != nil {
		return fmt.Errorf("skill %q: %w", name, err)
	}
	in.report(owner, "skill", dest, n)
	return nil
}

func (in *installer) installCoreSkills(reg *registry) error {
	for _, name := range reg.coreSkills {
		src := filepath.Join(in.harnessRoot, "core", "skills", name)
		dest := filepath.Join(in.projectRoot, ".claude", "skills", name)
		n, err := copyTree(src, dest, in.dryRun)
		if err != nil {
			return fmt.Errorf("core skill %q: %w", name, err)
		}
		in.report("core", name, dest, n)
	}
	return nil
}

func (in *installer) resolveTarget(m *manifest, kind string) (string, error) {
	if raw, ok := m.Targets[kind]; ok {
		return expandTarget(raw, in.projectRoot)
	}
	return filepath.Join(in.projectRoot, ".claude", kind), nil
}

func (in *installer) report(scope, kind, dest string, n int) {
	verb := "installed"
	if in.dryRun {
		verb = "would install"
	}
	plural := "s"
	if n == 1 {
		plural = ""
	}
	fmt.Printf("%-13s %-8s %-7s -> %s (%d file%s)\n", verb, scope, kind, dest, n, plural)
}

var varPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func expandTarget(raw, projectRoot string) (string, error) {
	var missing string
	out := varPattern.ReplaceAllStringFunc(raw, func(tok string) string {
		name := varPattern.FindStringSubmatch(tok)[1]
		if name == claudeHomeVar {
			return filepath.Join(projectRoot, ".claude")
		}
		v, ok := os.LookupEnv(name)
		if !ok || v == "" {
			missing = name
			return ""
		}
		return v
	})
	if missing != "" {
		return "", fmt.Errorf("target %q references $%s, which is not set", raw, missing)
	}
	return filepath.Clean(out), nil
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func copyTree(src, dst string, dryRun bool) (int, error) {
	info, err := os.Stat(src)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("%s is not a directory", src)
	}
	count := 0
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			if dryRun {
				return nil
			}
			return os.MkdirAll(target, 0o755)
		}
		if d.Type()&fs.ModeSymlink != 0 {
			fmt.Fprintf(os.Stderr, "as-skill: warning: skipping symlink %s\n", path)
			return nil
		}
		count++
		if dryRun {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
	return count, err
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(info.Mode().Perm())
}

func makeShellScriptsExecutable(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sh") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		if err := os.Chmod(p, info.Mode().Perm()|0o111); err != nil {
			return err
		}
	}
	return nil
}

type installOpts struct {
	project     string
	harnessRoot string
	dryRun      bool
	withCore    bool
}

func parseInstallArgs(args []string) ([]string, installOpts, error) {
	opts := installOpts{project: ".", harnessRoot: "."}
	var positionals []string

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
				return nil, opts, err
			}
			opts.project = v
		case "--harness-root":
			v, err := takeValue()
			if err != nil {
				return nil, opts, err
			}
			opts.harnessRoot = v
		case "--dry-run":
			opts.dryRun = true
		case "--with-core":
			opts.withCore = true
		default:
			if strings.HasPrefix(a, "-") {
				return nil, opts, fmt.Errorf("unknown flag %q", a)
			}
			positionals = append(positionals, a)
		}
	}
	return positionals, opts, nil
}

func runInstall(args []string) error {
	if len(args) == 0 {
		return errors.New("install needs a mode: domain|domains|all|skill")
	}
	mode := args[0]
	positionals, opts, err := parseInstallArgs(args[1:])
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
	reg, err := loadRegistry(harnessRoot)
	if err != nil {
		return err
	}
	in := &installer{harnessRoot: harnessRoot, projectRoot: projectRoot, dryRun: opts.dryRun}

	switch mode {
	case "domain":
		if len(positionals) != 1 {
			return errors.New("install domain needs exactly one domain name")
		}
		if err := in.installDomain(reg, positionals[0], true); err != nil {
			return err
		}
		if opts.withCore {
			if err := in.installCoreSkills(reg); err != nil {
				return err
			}
		}
	case "domains":
		if len(positionals) == 0 {
			return errors.New("install domains needs at least one domain name")
		}
		for _, name := range positionals {
			if err := in.installDomain(reg, name, true); err != nil {
				return err
			}
		}
		if opts.withCore {
			if err := in.installCoreSkills(reg); err != nil {
				return err
			}
		}
	case "all":
		if len(positionals) != 0 {
			return fmt.Errorf("install all takes no domain names (got %s)", strings.Join(positionals, ", "))
		}
		for _, name := range reg.domainNames() {
			if err := in.installDomain(reg, name, false); err != nil {
				return err
			}
		}
		if err := in.installCoreSkills(reg); err != nil {
			return err
		}
	case "skill":
		if len(positionals) != 1 {
			return errors.New("install skill needs exactly one skill name")
		}
		if err := in.installSkill(reg, positionals[0]); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown install mode %q (want domain|domains|all|skill)", mode)
	}

	fmt.Printf("done -> %s\n", filepath.Join(projectRoot, ".claude"))
	return nil
}

func runList(args []string) error {
	what := "all"
	if len(args) > 0 {
		what = args[0]
	}
	if what != "all" && what != "domains" && what != "skills" {
		return fmt.Errorf("unknown list target %q (want domains|skills)", what)
	}

	harnessRoot, err := resolveHarnessRoot(".")
	if err != nil {
		return err
	}
	reg, err := loadRegistry(harnessRoot)
	if err != nil {
		return err
	}

	if what == "all" || what == "domains" {
		fmt.Println("domains:")
		for _, name := range reg.domainNames() {
			m := reg.domains[name]
			status := "available"
			for _, envVar := range m.RequiresEnv {
				if os.Getenv(envVar) == "" {
					status = fmt.Sprintf("blocked — needs $%s", envVar)
					break
				}
			}
			fmt.Printf("  %-10s %-24s %s\n", name, "["+status+"]", m.Description)
		}
	}
	if what == "all" || what == "skills" {
		fmt.Println("skills:")
		for _, name := range reg.domainNames() {
			skillsDir := filepath.Join(harnessRoot, "domains", name, "skills")
			entries, err := os.ReadDir(skillsDir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					fmt.Printf("  %-30s (%s)\n", e.Name(), name)
				}
			}
		}
		for _, name := range reg.coreSkills {
			fmt.Printf("  %-30s (core)\n", name)
		}
	}
	return nil
}
