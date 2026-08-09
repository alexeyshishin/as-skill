package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
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
	case "uninstall":
		return runUninstall(args[1:])
	case "list":
		return runList(args[1:])
	case "status":
		return runStatus(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "check":
		return runCheck(args[1:])
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
  as-skill install domain  <name>              symlink one domain (default mode)
  as-skill install domains <name> [name...]    symlink several domains
  as-skill install all                         symlink every domain + core skills
  as-skill install skill   <name>              symlink one skill (domain-owned or core)
  as-skill install ... --copy                  copy a static snapshot instead of
                                                symlinking (same shapes as above; for
                                                sharing, or projects that shouldn't
                                                depend on this checkout's lifetime)
  as-skill uninstall domain|domains|all|skill ... remove whatever `+"`install`"+` placed
                                                (symlinks for link-mode entries, the
                                                real copied files/dirs for copy-mode
                                                entries; never touches foreign paths)
  as-skill list [domains|skills]                show what's installable
  as-skill status [--project PATH]              list lockfile entries: mode,
                                                path, health (OK/MISSING/BROKEN)
  as-skill doctor [--project PATH]
                  [--harness-root PATH]         status, plus source-gone/hash-
                                                drift/untracked checks; exits
                                                non-zero on any finding (CI gate)
  as-skill check                                validate domains/*/manifest.yaml
                                                and skills/*/SKILL.md in this
                                                harness checkout; no --project

Flags (install/uninstall):
  --project PATH        target project root, gets a .claude/ (default ".")
  --harness-root PATH   this repo's checkout (default: auto-detected upward from ".")
  --with-core            also install/uninstall core/skills/* (domain/domains/skill
                          modes; "all" always includes them)
  --copy                 install: copy a static snapshot instead of symlinking
                          (default: symlink)
  --dry-run              print what would happen, write nothing
  --force                install: overwrite existing content at the destination
                          even if as-skill isn't tracking it (default: refuse)

Examples:
  as-skill install domain git --project ~/code/my-app
  as-skill install domains git content --project .
  as-skill install all --project ~/code/my-app --copy
  as-skill install skill obsidian-ingest --project . --copy
  as-skill uninstall domain git --project .
`)
}

type manifest struct {
	Name        string
	Description string
	RequiresEnv []string
	RequiresBin []string
	Targets     map[string]string
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

type registry struct {
	domains    map[string]*manifest
	order      []string
	coreSkills []string
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

var domainKinds = []string{"rules", "skills", "agents", "hooks"}

type installer struct {
	harnessRoot string
	projectRoot string
	dryRun      bool
	force       bool
	mode        string
	lock        *lockfile
}

func (in *installer) place(domain, kind, name, src, dst string) (int, error) {
	if in.mode == "link" {
		if kind == "skill" {
			if !in.dryRun {
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					return 0, err
				}
			}
			if err := linkOne(src, dst, in.dryRun, in.force, in.lock); err != nil {
				return 0, err
			}
			if !in.dryRun {
				in.lock.record(lockEntry{
					Domain: domain, Kind: kind, Name: name,
					Mode: "link", Path: dst, Source: src,
					InstalledAt: time.Now(),
				})
			}
			return 1, nil
		}
		return linkTree(domain, kind, src, dst, in.dryRun, in.force, in.lock)
	}

	if entry, ok := in.lock.owns(dst); ok && entry.Mode != "copy" {
		fmt.Fprintf(os.Stderr, "as-skill: warning: %s was installed via %q, replacing with a copy — mixed channels may cause confusion\n", dst, entry.Mode)
	}
	n, err := copyTree(src, dst, in.dryRun)
	if err != nil {
		return n, err
	}
	if !in.dryRun {
		hash, err := hashTree(dst)
		if err != nil {
			return n, fmt.Errorf("hashing %s: %w", dst, err)
		}
		in.lock.record(lockEntry{
			Domain: domain, Kind: kind, Name: name,
			Mode: "copy", Path: dst, Source: src, SHA256: hash,
			InstalledAt: time.Now(),
		})
	}
	return n, nil
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
	for _, bin := range m.RequiresBin {
		if hasBin(bin) {
			continue
		}
		msg := fmt.Sprintf("domain %q requires the %q binary, which is not on $PATH", name, bin)
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
		n, err := in.place(name, kind, "", srcDir, destPath)
		if err != nil {
			return fmt.Errorf("domain %q: copying %s: %w", name, kind, err)
		}
		in.report(name, kind, destPath, n)
		if kind == "hooks" && !in.dryRun {
			if in.mode == "copy" {
				if err := makeShellScriptsExecutable(destPath); err != nil {
					return fmt.Errorf("domain %q: chmod hooks: %w", name, err)
				}
			} else {
				warnHooksNotExecutable(srcDir)
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
		n, err := in.place("core", "skill", name, src, dest)
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
	for _, bin := range m.RequiresBin {
		if !hasBin(bin) {
			return fmt.Errorf("skill %q belongs to domain %q, which requires the %q binary (not found on $PATH)", name, owner, bin)
		}
	}
	destBase, err := in.resolveTarget(m, "skills")
	if err != nil {
		return fmt.Errorf("skill %q: %w", name, err)
	}
	dest := filepath.Join(destBase, name)
	src := filepath.Join(in.harnessRoot, "domains", owner, "skills", name)
	n, err := in.place(owner, "skill", name, src, dest)
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
		n, err := in.place("core", "skill", name, src, dest)
		if err != nil {
			return fmt.Errorf("core skill %q: %w", name, err)
		}
		in.report("core", name, dest, n)
	}
	return nil
}

func (in *installer) uninstallDomain(reg *registry, name string) error {
	m, ok := reg.domains[name]
	if !ok {
		return fmt.Errorf("unknown domain %q (available: %s)", name, strings.Join(reg.domainNames(), ", "))
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
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			return fmt.Errorf("domain %q: %w", name, err)
		}
		for _, e := range entries {
			itemName := e.Name()
			dst := filepath.Join(destPath, itemName)
			in.uninstallOne(name, kind, itemName, dst)
		}

		if _, tracked := in.lock.find(name, kind, ""); tracked {
			in.lock.remove(name, kind, "")
		}
	}
	return nil
}

func (in *installer) uninstallSkill(reg *registry, name string) error {
	for _, cs := range reg.coreSkills {
		if cs != name {
			continue
		}
		dest := filepath.Join(in.projectRoot, ".claude", "skills", name)
		in.uninstallOne("core", "skill", name, dest)
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
	destBase, err := in.resolveTarget(m, "skills")
	if err != nil {
		return fmt.Errorf("skill %q: %w", name, err)
	}
	dest := filepath.Join(destBase, name)
	in.uninstallOne(owner, "skill", name, dest)
	return nil
}

func (in *installer) uninstallCoreSkills(reg *registry) {
	for _, name := range reg.coreSkills {
		dest := filepath.Join(in.projectRoot, ".claude", "skills", name)
		in.uninstallOne("core", "skill", name, dest)
	}
}

func (in *installer) uninstallOne(domain, kind, name, dst string) {
	entry, tracked := in.lock.owns(dst)
	if !tracked {
		if parent, ok := in.lock.find(domain, kind, ""); ok {
			entry, tracked = parent, true
		}
	}
	if !tracked {
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — not tracked by as-skill\n", dst)
		return
	}
	fi, err := os.Lstat(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — tracked but nothing on disk\n", dst)
		return
	}
	isSymlink := fi.Mode()&os.ModeSymlink != 0
	switch entry.Mode {
	case "link":
		if !isSymlink {
			fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — tracked as a link but not a symlink on disk\n", dst)
			return
		}
	case "copy":
		if isSymlink {
			fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — tracked as a copy but is a symlink on disk\n", dst)
			return
		}
	default:
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — unrecognized tracked mode %q\n", dst, entry.Mode)
		return
	}
	if in.dryRun {
		fmt.Printf("would uninstall %s\n", dst)
		return
	}
	var removeErr error
	if entry.Mode == "link" {
		removeErr = os.Remove(dst)
	} else {
		removeErr = os.RemoveAll(dst)
	}
	if removeErr != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: failed removing %s: %v\n", dst, removeErr)
		return
	}
	in.lock.remove(domain, kind, name)
	fmt.Printf("uninstalled    %s\n", dst)
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

func hasBin(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
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

func warnHooksNotExecutable(srcDir string) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sh") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Mode().Perm()&0o111 != 0 {
			continue
		}
		fmt.Fprintf(os.Stderr, "as-skill: warning: %s is not executable; the symlinked hook won't be either (fix it in domains/)\n", filepath.Join(srcDir, e.Name()))
	}
}

type installOpts struct {
	project     string
	harnessRoot string
	dryRun      bool
	withCore    bool
	force       bool
	copy        bool
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
		case "--force":
			opts.force = true
		case "--copy":
			opts.copy = true
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
		return errors.New("install needs a verb: domain|domains|all|skill")
	}
	verb := args[0]
	positionals, opts, err := parseInstallArgs(args[1:])
	if err != nil {
		return err
	}
	mode := "link"
	if opts.copy {
		mode = "copy"
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
	lock, err := loadLockfile(projectRoot)
	if err != nil {
		return err
	}
	in := &installer{
		harnessRoot: harnessRoot, projectRoot: projectRoot,
		dryRun: opts.dryRun, force: opts.force, mode: mode, lock: lock,
	}

	switch verb {
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
		return fmt.Errorf("unknown install verb %q (want domain|domains|all|skill)", verb)
	}

	fmt.Printf("done -> %s\n", filepath.Join(projectRoot, ".claude"))
	if !opts.dryRun {
		if err := lock.save(projectRoot); err != nil {
			return fmt.Errorf("saving lockfile: %w", err)
		}
	}
	return nil
}

func runUninstall(args []string) error {
	if len(args) == 0 {
		return errors.New("uninstall needs a verb: domain|domains|all|skill")
	}
	verb := args[0]
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
	lock, err := loadLockfile(projectRoot)
	if err != nil {
		return err
	}
	in := &installer{
		harnessRoot: harnessRoot, projectRoot: projectRoot,
		dryRun: opts.dryRun, lock: lock,
	}

	switch verb {
	case "domain":
		if len(positionals) != 1 {
			return errors.New("uninstall domain needs exactly one domain name")
		}
		if err := in.uninstallDomain(reg, positionals[0]); err != nil {
			return err
		}
		if opts.withCore {
			in.uninstallCoreSkills(reg)
		}
	case "domains":
		if len(positionals) == 0 {
			return errors.New("uninstall domains needs at least one domain name")
		}
		for _, name := range positionals {
			if err := in.uninstallDomain(reg, name); err != nil {
				return err
			}
		}
		if opts.withCore {
			in.uninstallCoreSkills(reg)
		}
	case "all":
		if len(positionals) != 0 {
			return fmt.Errorf("uninstall all takes no domain names (got %s)", strings.Join(positionals, ", "))
		}
		for _, name := range reg.domainNames() {
			if err := in.uninstallDomain(reg, name); err != nil {
				return err
			}
		}
		in.uninstallCoreSkills(reg)
	case "skill":
		if len(positionals) != 1 {
			return errors.New("uninstall skill needs exactly one skill name")
		}
		if err := in.uninstallSkill(reg, positionals[0]); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown uninstall verb %q (want domain|domains|all|skill)", verb)
	}

	fmt.Printf("done -> %s\n", filepath.Join(projectRoot, ".claude"))
	if !opts.dryRun {
		if err := lock.save(projectRoot); err != nil {
			return fmt.Errorf("saving lockfile: %w", err)
		}
	}
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
			if status == "available" {
				for _, bin := range m.RequiresBin {
					if !hasBin(bin) {
						status = fmt.Sprintf("blocked — needs %q binary", bin)
						break
					}
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
