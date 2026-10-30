package installer

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"claude-harness/tools/internal/frontmatter"
	"claude-harness/tools/internal/fsutil"
	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/render"
	"claude-harness/tools/internal/transfer"
)

const (
	claudeSkillsLoc = ".claude/skills"
	promptsDir      = ".agents/prompts"
	agentsRulesDir  = ".agents/rules"
	agentsMD        = "AGENTS.md"
	ModeBlock       = "block"
)

func (in *Installer) plats() []platform.ID {
	if len(in.Platforms) == 0 {
		return []platform.ID{platform.Claude}
	}
	return in.Platforms
}

func (in *Installer) hasClaude() bool {
	for _, p := range in.plats() {
		if p == platform.Claude {
			return true
		}
	}
	return false
}

func strs(ids []platform.ID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

func orderPlats(vals []string) []string {
	seen := map[string]bool{}
	for _, v := range vals {
		seen[v] = true
	}
	var out []string
	for _, id := range platform.IDs() {
		if seen[string(id)] {
			out = append(out, string(id))
			delete(seen, string(id))
		}
	}
	var rest []string
	for v := range seen {
		rest = append(rest, v)
	}
	sort.Strings(rest)
	return append(out, rest...)
}

func unionPlats(a, b []string) []string {
	return orderPlats(append(append([]string(nil), a...), b...))
}

func minusPlats(a []string, ids []platform.ID) []string {
	drop := map[string]bool{}
	for _, id := range ids {
		drop[string(id)] = true
	}
	var out []string
	for _, v := range a {
		if !drop[v] {
			out = append(out, v)
		}
	}
	return out
}

func (in *Installer) dupPlatforms(locs map[string][]platform.ID) []platform.ID {
	occupied := map[string]bool{}
	for loc := range locs {
		occupied[loc] = true
	}
	known := map[string]bool{}
	for _, e := range in.Lock.Entries {
		for _, pl := range e.Platforms {
			known[pl] = true
		}
		if e.Kind != "skills" && e.Kind != "skill" {
			continue
		}
		for _, loc := range []string{".claude/skills", ".agents/skills", ".cursor/skills", ".opencode/skills", ".codex/skills"} {
			if under(filepath.Join(in.ProjectRoot, loc), e.Path) {
				occupied[loc] = true
			}
		}
	}
	for _, id := range in.plats() {
		known[string(id)] = true
	}
	var dup []platform.ID
	for _, id := range []platform.ID{platform.Cursor, platform.OpenCode} {
		if !known[string(id)] {
			continue
		}
		p, _ := platform.Get(id)
		n := 0
		for _, d := range p.SkillDirsRead {
			if occupied[d] {
				n++
			}
		}
		if n >= 2 {
			dup = append(dup, id)
		}
	}
	return dup
}

func (in *Installer) warnDup(locs map[string][]platform.ID) {
	dup := in.dupPlatforms(locs)
	if len(dup) == 0 || in.dupWarned {
		return
	}
	in.dupWarned = true
	fmt.Fprintf(os.Stderr, "as-skill: warning: skills land in both .claude/skills and .agents/skills; %s will see each skill twice (doctor reports DUPLICATE-SKILL)\n", strings.Join(strs(dup), ", "))
}

func under(root, p string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(p))
	return err == nil && !strings.HasPrefix(rel, "..")
}

func (in *Installer) snapshotPlatforms(domain, kind, dest string) map[string][]string {
	out := map[string][]string{}
	var all []string
	for _, e := range in.Lock.Entries {
		if e.Domain == domain && e.Kind == kind && under(dest, e.Path) {
			out[filepath.Clean(e.Path)] = e.Platforms
			all = unionPlats(all, e.Platforms)
		}
	}
	out[filepath.Clean(dest)] = all
	return out
}

func (in *Installer) applyPlatforms(domain, kind, dest string, plats []platform.ID, prior map[string][]string) {
	if in.DryRun {
		return
	}
	for i, e := range in.Lock.Entries {
		if e.Domain == domain && e.Kind == kind && under(dest, e.Path) {
			p, ok := prior[filepath.Clean(e.Path)]
			if !ok {
				p = prior[filepath.Clean(dest)]
			}
			in.Lock.Entries[i].Platforms = unionPlats(p, strs(plats))
		}
	}
}

func (in *Installer) setEntryPlatforms(e lockfile.Entry, plats []string) {
	for i, x := range in.Lock.Entries {
		if x.Domain == e.Domain && x.Kind == e.Kind && x.Name == e.Name && filepath.Clean(x.Path) == filepath.Clean(e.Path) {
			in.Lock.Entries[i].Platforms = plats
		}
	}
}

var nonClaudeRoots = map[string]bool{".agents": true, ".codex": true, ".cursor": true, ".opencode": true}

func (in *Installer) guard(dst string) error {
	if in.Mode != "link" || in.Force {
		return nil
	}
	rel, err := filepath.Rel(in.ProjectRoot, dst)
	if err != nil {
		return nil
	}
	return in.guardRoot(strings.Split(filepath.ToSlash(rel), "/")[0])
}

func (in *Installer) guardRoot(root string) error {
	if in.Mode != "link" || in.Force || !nonClaudeRoots[root] {
		return nil
	}
	if ok, done := in.gitChecked[root]; done {
		if ok {
			return nil
		}
		return in.conflict(root, ReasonNotIgnored)
	}
	if in.gitChecked == nil {
		in.gitChecked = map[string]bool{}
	}
	ok := gitIgnored(in.ProjectRoot, root)
	in.gitChecked[root] = ok
	if ok {
		return nil
	}
	return in.conflict(root, ReasonNotIgnored)
}

func gitIgnored(project, root string) bool {
	if _, err := exec.LookPath("git"); err != nil {
		return true
	}
	if err := exec.Command("git", "-C", project, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		return true
	}
	err := exec.Command("git", "-C", project, "check-ignore", "-q", root+"/x").Run()
	if err == nil {
		return true
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
		return false
	}
	return true
}

var capGaps = map[platform.ID]string{
	platform.Codex:    "agents: frontmatter edits not live (doctor reports STALE); hooks: needs trust via /hooks",
	platform.Cursor:   "agents: frontmatter edits not live (doctor reports STALE); hooks: stop cannot block",
	platform.OpenCode: "agents: frontmatter edits not live (doctor reports STALE); hooks: unsupported",
}

// CapGap returns the known capability gap note for a platform, or "".
func CapGap(id platform.ID) string { return capGaps[id] }

func (in *Installer) noteGaps() {
	if in.gapsDone {
		return
	}
	in.gapsDone = true
	for _, id := range in.plats() {
		if g, ok := capGaps[id]; ok {
			fmt.Fprintf(os.Stderr, "as-skill: note: %s: %s\n", id, g)
		}
	}
}

func rootOf(rel string) string {
	return strings.Split(filepath.ToSlash(rel), "/")[0]
}

func (in *Installer) touchedRoots(kinds map[string]bool) []string {
	roots := map[string]bool{}
	if kinds["skills"] {
		locs, _ := platform.SkillLocations(in.plats())
		for loc := range locs {
			roots[rootOf(loc)] = true
		}
	}
	for _, id := range in.plats() {
		if id == platform.Claude {
			continue
		}
		p, _ := platform.Get(id)
		if kinds["agents"] {
			roots[rootOf(p.Agents.Dir)] = true
			if in.Mode == "link" {
				roots[rootOf(promptsDir)] = true
			}
		}
		if kinds["rules"] {
			roots[rootOf(p.Rules.Dir)] = true
		}
		if kinds["hooks"] && p.Hooks.Schema != platform.HooksNone {
			roots[rootOf(p.Hooks.Dir)] = true
		}
	}
	var out []string
	for r := range roots {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

func (in *Installer) preflightRoots(kinds map[string]bool) error {
	in.noteGaps()
	for _, r := range in.touchedRoots(kinds) {
		if err := in.guardRoot(r); err != nil {
			return err
		}
	}
	return nil
}

func (in *Installer) GuardRoots(kinds ...string) error {
	set := map[string]bool{}
	for _, k := range kinds {
		set[k] = true
	}
	for _, r := range in.touchedRoots(set) {
		if err := in.guardRoot(r); err != nil {
			return err
		}
	}
	return nil
}

func (in *Installer) runnable(m *harness.Manifest) bool {
	for _, e := range m.RequiresEnv {
		if os.Getenv(e) == "" {
			return false
		}
	}
	for _, b := range m.RequiresBin {
		if !fsutil.HasBin(b) {
			return false
		}
	}
	return true
}

func (in *Installer) Preflight(reg *harness.Registry, domains []string) error {
	for _, name := range domains {
		m, ok := reg.Domains[name]
		if !ok || !in.runnable(m) {
			continue
		}
		kinds := map[string]bool{}
		for _, kind := range harness.DomainKinds {
			if fsutil.DirExists(filepath.Join(in.HarnessRoot, "domains", name, kind)) {
				kinds[kind] = true
			}
		}
		if kinds["agents"] {
			if err := in.checkReserved(filepath.Join(in.HarnessRoot, "domains", name, "agents")); err != nil {
				return fmt.Errorf("domain %q: %w", name, err)
			}
		}
		if err := in.preflightRoots(kinds); err != nil {
			return err
		}
		if err := in.preflightForeign(name, kinds); err != nil {
			return err
		}
	}
	return nil
}

func (in *Installer) preflightForeign(name string, kinds map[string]bool) error {
	var paths []string
	itemPaths := func(kind, srcDir, dstDir string, plats []platform.ID) {
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if transfer.SkipTop(kind, e.Name()) || in.excluded(name, kind, e.Name()) || !hookWanted(kind, filepath.Join(srcDir, e.Name()), plats) {
				continue
			}
			paths = append(paths, filepath.Join(dstDir, e.Name()))
		}
	}
	for kind := range kinds {
		srcDir := filepath.Join(in.HarnessRoot, "domains", name, kind)
		if kind == "skills" {
			locs, _ := platform.SkillLocations(in.plats())
			for loc, plats := range locs {
				if (loc != claudeSkillsLoc || in.hasExclusions(name)) && in.Mode != "link" {
					itemPaths(kind, srcDir, filepath.Join(in.ProjectRoot, loc), plats)
				}
			}
			continue
		}
		for _, id := range in.plats() {
			if id == platform.Claude {
				continue
			}
			p, _ := platform.Get(id)
			switch kind {
			case "hooks":
				if p.Hooks.Schema != platform.HooksNone && in.Mode != "link" {
					itemPaths(kind, srcDir, filepath.Join(in.ProjectRoot, p.Hooks.Dir), []platform.ID{id})
				}
			case "agents", "rules":
				entries, err := in.mdItems(srcDir)
				if err != nil {
					return err
				}
				for _, e := range entries {
					stem := strings.TrimSuffix(e.Name(), ".md")
					switch {
					case kind == "agents":
						data, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
						if err != nil {
							return err
						}
						paths = append(paths, filepath.Join(in.ProjectRoot, p.Agents.Dir, agentNameOf(data, stem)+p.Agents.Ext))
					case p.Rules.Mode == platform.RulesDirMDC:
						paths = append(paths, filepath.Join(in.ProjectRoot, p.Rules.Dir, stem+".mdc"))
					case in.Mode != "link":
						paths = append(paths, filepath.Join(in.ProjectRoot, p.Rules.Dir, e.Name()))
					}
				}
			}
		}
	}
	for _, dst := range paths {
		if err := in.refuseForeign(dst); err != nil {
			return fmt.Errorf("domain %q: %w", name, err)
		}
	}
	return nil
}

func hookWanted(kind, path string, plats []platform.ID) bool {
	if kind != "hooks" {
		return true
	}
	for _, id := range plats {
		if hookTargets(path, id) {
			return true
		}
	}
	return false
}

func (in *Installer) checkReserved(srcDir string) error {
	entries, err := in.mdItems(srcDir)
	if err != nil {
		return err
	}
	for _, id := range in.plats() {
		p, _ := platform.Get(id)
		if id == platform.Claude || len(p.ReservedAgentNames) == 0 {
			continue
		}
		for _, e := range entries {
			data, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
			if err != nil {
				return err
			}
			if err := reservedErr(p, agentNameOf(data, strings.TrimSuffix(e.Name(), ".md")), e.Name()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (in *Installer) installKind(m *harness.Manifest, name, kind, srcDir string) error {
	switch kind {
	case "skills":
		locs, _ := platform.SkillLocations(in.plats())
		in.warnDup(locs)
		for _, loc := range slices.Sorted(maps.Keys(locs)) {
			if loc == claudeSkillsLoc {
				if err := in.claudeKind(m, name, kind, srcDir, locs[loc]); err != nil {
					return err
				}
				continue
			}
			if err := in.placeItems(name, kind, srcDir, filepath.Join(in.ProjectRoot, loc), locs[loc]); err != nil {
				return fmt.Errorf("domain %q: %s: %w", name, kind, err)
			}
		}
		return nil
	}
	if in.hasClaude() {
		if err := in.claudeKind(m, name, kind, srcDir, []platform.ID{platform.Claude}); err != nil {
			return err
		}
	}
	for _, id := range in.plats() {
		if id == platform.Claude {
			continue
		}
		p, _ := platform.Get(id)
		var err error
		switch kind {
		case "agents":
			err = in.placeAgents(name, srcDir, p)
		case "rules":
			err = in.placeRules(name, srcDir, p)
		case "hooks":
			err = in.placeHooks(name, srcDir, p)
		}
		if err != nil {
			return fmt.Errorf("domain %q: %s on %s: %w", name, kind, id, err)
		}
	}
	return nil
}

func (in *Installer) placeItems(domain, kind, srcDir, dstDir string, plats []platform.ID) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	n := 0
	for _, e := range entries {
		if transfer.SkipTop(kind, e.Name()) || in.excluded(domain, kind, e.Name()) || !hookWanted(kind, filepath.Join(srcDir, e.Name()), plats) {
			continue
		}
		if err := in.placeOne(domain, kind, e.Name(), filepath.Join(srcDir, e.Name()), filepath.Join(dstDir, e.Name()), plats); err != nil {
			return err
		}
		n++
	}
	in.report(domain, kind, dstDir, n)
	return nil
}

func (in *Installer) placeOne(domain, kind, name, src, dst string, plats []platform.ID) error {
	if err := in.guard(dst); err != nil {
		return err
	}
	pl := strs(plats)
	if prior, ok := in.Lock.FindAt(domain, kind, name, dst); ok {
		pl = unionPlats(prior.Platforms, pl)
	}
	if !in.DryRun {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
	}
	if in.Mode == "link" {
		if err := in.linkOne(src, dst); err != nil {
			return err
		}
		if !in.DryRun {
			in.Lock.Record(lockfile.Entry{
				Domain: domain, Kind: kind, Name: name, Mode: lockfile.ModeLink,
				Path: dst, Source: src, Platforms: pl, InstalledAt: time.Now(),
			})
		}
		return nil
	}
	if err := in.refuseForeign(dst); err != nil {
		return err
	}
	if !in.DryRun {
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
	}
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		if _, err := transfer.CopyTreeRoot(kind, src, dst, in.HarnessRoot, in.DryRun); err != nil {
			return err
		}
	} else if !in.DryRun {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, fi.Mode().Perm()); err != nil {
			return err
		}
	}
	if in.DryRun {
		return nil
	}
	hash, err := transfer.HashPath(dst)
	if err != nil {
		return fmt.Errorf("hashing %s: %w", dst, err)
	}
	in.Lock.Record(lockfile.Entry{
		Domain: domain, Kind: kind, Name: name, Mode: lockfile.ModeCopy,
		Path: dst, Source: src, SHA256: hash, Platforms: pl, InstalledAt: time.Now(),
	})
	return nil
}

func (in *Installer) refuseForeign(dst string) error {
	if _, err := os.Lstat(dst); err != nil {
		return nil
	}
	if _, ok := in.Lock.Covering(dst); ok || in.Force || in.overwrites(dst) {
		return nil
	}
	return in.conflict(dst, ReasonForeign)
}

func (in *Installer) writeRendered(domain, kind, name, src, dst string, srcData, out []byte, p platform.ID) error {
	if err := in.guard(dst); err != nil {
		return err
	}
	if err := in.refuseForeign(dst); err != nil {
		return err
	}
	pl := []string{string(p)}
	if prior, ok := in.Lock.FindAt(domain, kind, name, dst); ok {
		pl = unionPlats(prior.Platforms, pl)
	}
	if in.DryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if fi, err := os.Lstat(dst); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if err := os.Remove(dst); err != nil {
			return err
		}
	}
	if err := os.WriteFile(dst, out, 0o644); err != nil {
		return err
	}
	in.Lock.Record(lockfile.Entry{
		Domain: domain, Kind: kind, Name: name, Mode: lockfile.ModeRender,
		Path: dst, Source: src, SHA256: transfer.HashBytes(out), SrcSHA256: transfer.HashBytes(srcData),
		Platforms: pl, InstalledAt: time.Now(),
	})
	return nil
}

func (in *Installer) mdItems(srcDir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, err
	}
	var out []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, e)
		}
	}
	return out, nil
}

func (in *Installer) placeAgents(domain, srcDir string, p platform.Platform) error {
	entries, err := in.mdItems(srcDir)
	if err != nil {
		return err
	}
	mode := render.Copy
	if in.Mode == "link" {
		mode = render.Link
	}
	n := 0
	for _, e := range entries {
		stem := strings.TrimSuffix(e.Name(), ".md")
		src := filepath.Join(srcDir, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		agentName := agentNameOf(data, stem)
		if err := reservedErr(p, agentName, e.Name()); err != nil {
			return err
		}
		promptRel := ""
		if mode == render.Link {
			promptRel = promptsDir + "/" + e.Name()
			if err := in.placeOne(domain, "prompts", e.Name(), src, filepath.Join(in.ProjectRoot, promptRel), []platform.ID{p.ID}); err != nil {
				return err
			}
		}
		out, err := render.RenderAgent(data, filepath.ToSlash(relToHarness(in.HarnessRoot, src)), stem, render.Format(p.Agents.Format), mode, promptRel)
		if err != nil {
			return err
		}
		dst := filepath.Join(in.ProjectRoot, p.Agents.Dir, agentName+p.Agents.Ext)
		if err := in.writeRendered(domain, "agents", e.Name(), src, dst, data, out, p.ID); err != nil {
			return err
		}
		n++
	}
	in.report(domain, "agents", filepath.Join(in.ProjectRoot, p.Agents.Dir), n)
	return nil
}

func agentNameOf(data []byte, stem string) string {
	if doc, err := frontmatter.Parse(data); err == nil {
		if v, ok := doc.String("name"); ok && v != "" {
			return v
		}
	}
	return stem
}

func reservedErr(p platform.Platform, agentName, file string) error {
	for _, r := range p.ReservedAgentNames {
		if agentName == r {
			return fmt.Errorf("agent name %q (%s) is reserved on %s", agentName, file, p.ID)
		}
	}
	return nil
}

func relToHarness(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil || strings.HasPrefix(r, "..") {
		return p
	}
	return r
}

func (in *Installer) placeRules(domain, srcDir string, p platform.Platform) error {
	entries, err := in.mdItems(srcDir)
	if err != nil {
		return err
	}
	n := 0
	for _, e := range entries {
		stem := strings.TrimSuffix(e.Name(), ".md")
		src := filepath.Join(srcDir, e.Name())
		switch p.Rules.Mode {
		case platform.RulesDirMDC:
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			rel := agentsRulesDir + "/" + e.Name()
			body := string(data)
			if in.Mode == "link" {
				if err := in.placeOne(domain, "rules", e.Name(), src, filepath.Join(in.ProjectRoot, rel), []platform.ID{p.ID}); err != nil {
					return err
				}
				body = "Follow the rule in `" + rel + "`.\n"
			} else if !strings.HasSuffix(body, "\n") {
				body += "\n"
			}
			out := []byte("---\ndescription: " + stem + " rule\nalwaysApply: true\n---\n" + body)
			dst := filepath.Join(in.ProjectRoot, p.Rules.Dir, stem+".mdc")
			if err := in.writeRendered(domain, "rules", e.Name(), src, dst, data, out, p.ID); err != nil {
				return err
			}
		default:
			if err := in.placeOne(domain, "rules", e.Name(), src, filepath.Join(in.ProjectRoot, p.Rules.Dir, e.Name()), []platform.ID{p.ID}); err != nil {
				return err
			}
		}
		n++
	}
	in.report(domain, "rules", filepath.Join(in.ProjectRoot, p.Rules.Dir), n)
	if p.Rules.Mode == platform.RulesAgentsMDBlock {
		return in.syncAgentsMD()
	}
	return nil
}

func (in *Installer) placeHooks(domain, srcDir string, p platform.Platform) error {
	if p.Hooks.Schema == platform.HooksNone {
		fmt.Fprintf(os.Stderr, "as-skill: warning: hooks unsupported on %s\n", p.ID)
		return nil
	}
	dst := filepath.Join(in.ProjectRoot, p.Hooks.Dir)
	if err := in.placeItems(domain, "hooks", srcDir, dst, []platform.ID{p.ID}); err != nil {
		return err
	}
	if !in.DryRun {
		if in.Mode == "copy" {
			if err := transfer.MakeShellScriptsExecutable(dst); err != nil {
				return err
			}
		} else {
			transfer.WarnHooksNotExecutable(srcDir)
		}
	}
	fmt.Fprintf(os.Stderr, "as-skill: warning: hooks on %s: files placed; run `as-skill hooks enable` to register (UNVERIFIED adapters)\n", p.ID)
	return nil
}

func (in *Installer) syncAgentsMD() error {
	path := filepath.Join(in.ProjectRoot, agentsMD)
	rulesRoot := filepath.Join(in.ProjectRoot, agentsRulesDir)
	var files []string
	var plats []string
	for _, e := range in.Lock.Entries {
		if e.Kind != "rules" || (e.Mode != lockfile.ModeLink && e.Mode != lockfile.ModeCopy) || !under(rulesRoot, e.Path) {
			continue
		}
		var hit []string
		for _, pl := range e.Platforms {
			if pl == string(platform.Codex) || pl == string(platform.OpenCode) {
				hit = append(hit, pl)
			}
		}
		if len(hit) == 0 {
			continue
		}
		files = append(files, agentsRulesDir+"/"+filepath.Base(e.Path))
		plats = unionPlats(plats, hit)
	}
	sort.Strings(files)
	existing, rerr := os.ReadFile(path)
	exists := rerr == nil
	prior, tracked := in.Lock.FindAt("as-skill", "instructions", agentsMD, path)
	if len(files) == 0 {
		if !tracked {
			return nil
		}
		if in.DryRun {
			fmt.Printf("would update   %s\n", path)
			return nil
		}
		if exists {
			stripped, _ := render.RemoveBlock(existing)
			if prior.Source == "created" && len(bytes.TrimSpace(stripped)) == 0 {
				if err := os.Remove(path); err != nil {
					return err
				}
			} else if !bytes.Equal(stripped, existing) {
				if err := os.WriteFile(path, stripped, 0o644); err != nil {
					return err
				}
			}
		}
		in.Lock.RemoveAt("as-skill", "instructions", agentsMD, path)
		return nil
	}
	var b strings.Builder
	b.WriteString("Always follow these rules:\n")
	for _, f := range files {
		b.WriteString("- `" + f + "`\n")
	}
	if in.DryRun {
		fmt.Printf("would update   %s\n", path)
		return nil
	}
	out := render.UpsertBlock(existing, b.String())
	if !exists || !bytes.Equal(out, existing) {
		if err := os.WriteFile(path, out, 0o644); err != nil {
			return err
		}
	}
	src := "created"
	if tracked {
		src = prior.Source
	} else if exists {
		src = "existing"
	}
	in.Lock.Record(lockfile.Entry{
		Domain: "as-skill", Kind: "instructions", Name: agentsMD, Mode: ModeBlock,
		Path: path, Source: src, Platforms: plats, InstalledAt: time.Now(),
	})
	return nil
}

func (in *Installer) uninstallNonClaude(domain, skill string) {
	kinds := map[string]bool{"rules": true, "skills": true, "skill": true, "agents": true, "hooks": true, "prompts": true}
	plats := in.plats()
	gone := map[platform.ID][]string{}
	snapshot := append([]lockfile.Entry(nil), in.Lock.Entries...)
	for _, e := range snapshot {
		if e.Domain != domain || !kinds[e.Kind] {
			continue
		}
		if skill != "" && (e.Name != skill || (e.Kind != "skill" && e.Kind != "skills")) {
			continue
		}
		if in.hasClaude() && under(filepath.Join(in.ProjectRoot, ".claude"), e.Path) {
			continue
		}
		remaining := minusPlats(e.Platforms, plats)
		if len(remaining) == len(e.Platforms) {
			continue
		}
		if len(remaining) > 0 {
			if in.DryRun {
				fmt.Printf("would keep     %s — still used by %s\n", e.Path, strings.Join(remaining, ","))
				continue
			}
			in.setEntryPlatforms(e, remaining)
			fmt.Printf("kept           %s — still used by %s\n", e.Path, strings.Join(remaining, ","))
			continue
		}
		if e.Name != "" && in.removeEntry(e) && e.Kind == "hooks" {
			for _, id := range plats {
				gone[id] = append(gone[id], e.Name)
			}
		} else if e.Name == "" && !in.DryRun {
			in.Lock.RemoveAt(e.Domain, e.Kind, e.Name, e.Path)
		}
	}
	for id, names := range gone {
		if id != platform.Claude {
			in.unregisterHooksFor(id, names)
		}
	}
	if err := in.syncAgentsMD(); err != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: updating %s: %v\n", agentsMD, err)
	}
}

func (in *Installer) removeEntry(e lockfile.Entry) bool {
	fi, err := os.Lstat(e.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — tracked but nothing on disk\n", e.Path)
		return false
	}
	isLink := fi.Mode()&os.ModeSymlink != 0
	switch e.Mode {
	case lockfile.ModeLink:
		if !isLink {
			fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — tracked as a link but not a symlink on disk\n", e.Path)
			return false
		}
	case lockfile.ModeCopy, lockfile.ModeRender:
		if isLink {
			fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — tracked as %s but is a symlink on disk\n", e.Path, e.Mode)
			return false
		}
	default:
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — unrecognized tracked mode %q\n", e.Path, e.Mode)
		return false
	}
	if err := InsideRoots(in.ProjectRoot, e.Path); err != nil {
		fmt.Fprintf(os.Stderr, "as-skill: error: refusing %v\n", err)
		return false
	}
	if in.DryRun {
		fmt.Printf("would remove   %s\n", e.Path)
		return true
	}
	var rerr error
	if e.Mode == lockfile.ModeCopy {
		rerr = os.RemoveAll(e.Path)
	} else {
		rerr = os.Remove(e.Path)
	}
	if rerr != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: failed removing %s: %v\n", e.Path, rerr)
		return false
	}
	in.Lock.RemoveAt(e.Domain, e.Kind, e.Name, e.Path)
	fmt.Printf("uninstalled    %s\n", e.Path)
	return true
}
