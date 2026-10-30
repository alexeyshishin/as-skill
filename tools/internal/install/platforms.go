package install

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"claude-harness/tools/internal/frontmatter"
	"claude-harness/tools/internal/fsys"
	"claude-harness/tools/internal/hookcfg"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/render"
)

const agentsMD = "AGENTS.md"

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
		if e.Kind != lockfile.KindSkills && e.Kind != lockfile.KindSkill {
			continue
		}
		for _, loc := range platform.AllSkillDirs() {
			if fsys.Within(filepath.Join(in.ProjectRoot, loc), e.Path) {
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

func (in *Installer) snapshotPlatforms(domain string, kind lockfile.Kind, dest string) map[string][]string {
	out := map[string][]string{}
	var all []string
	for _, e := range in.Lock.Entries {
		if e.Domain == domain && e.Kind == kind && fsys.Within(dest, e.Path) {
			out[filepath.Clean(e.Path)] = e.Platforms
			all = unionPlats(all, e.Platforms)
		}
	}
	out[filepath.Clean(dest)] = all
	return out
}

func (in *Installer) applyPlatforms(domain string, kind lockfile.Kind, dest string, plats []platform.ID, prior map[string][]string) {
	if in.DryRun {
		return
	}
	for i, e := range in.Lock.Entries {
		if e.Domain == domain && e.Kind == kind && fsys.Within(dest, e.Path) {
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

type gitCache map[string]bool

func (in *Installer) unignored(c gitCache, root string) bool {
	if in.Mode != lockfile.ModeLink || in.Force || !platform.IsLinkRoot(root) {
		return false
	}
	ok, done := c[root]
	if !done {
		ok = gitIgnored(in.ProjectRoot, root)
		c[root] = ok
	}
	return !ok
}

func (in *Installer) guardRootErr(c gitCache, root string) error {
	if in.unignored(c, root) {
		return &ConflictError{Path: root, Reason: ReasonNotIgnored}
	}
	return nil
}

func (in *Installer) dstRoot(dst string) (string, bool) {
	rel, err := filepath.Rel(in.ProjectRoot, dst)
	if err != nil {
		return "", false
	}
	return strings.Split(filepath.ToSlash(rel), "/")[0], true
}

func (in *Installer) guard(c gitCache, dst string) error {
	if in.Mode != lockfile.ModeLink || in.Force {
		return nil
	}
	root, ok := in.dstRoot(dst)
	if !ok {
		return nil
	}
	return in.guardRootErr(c, root)
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

func (in *Installer) touchedRoots(kinds map[lockfile.Kind]bool) []string {
	roots := map[string]bool{}
	if kinds[lockfile.KindSkills] {
		locs, _ := platform.SkillLocations(in.plats())
		for loc := range locs {
			roots[platform.RootOf(loc)] = true
		}
	}
	for _, id := range in.plats() {
		if id == platform.Claude {
			continue
		}
		p, _ := platform.Get(id)
		if kinds[lockfile.KindAgents] {
			roots[platform.RootOf(p.Agents.Dir)] = true
			if in.Mode == lockfile.ModeLink {
				roots[platform.RootOf(platform.PromptsDir)] = true
			}
		}
		if kinds[lockfile.KindRules] {
			roots[platform.RootOf(p.Rules.Dir)] = true
		}
		if kinds[lockfile.KindHooks] && p.Hooks.Schema != platform.HooksNone {
			roots[platform.RootOf(p.Hooks.Dir)] = true
		}
	}
	var out []string
	for r := range roots {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

func hookWanted(kind lockfile.Kind, path string, plats []platform.ID) bool {
	if kind != lockfile.KindHooks {
		return true
	}
	for _, id := range plats {
		if hookcfg.HookTargets(path, id) {
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

func kindApplyErr(name string, kind lockfile.Kind, err error) error {
	var oe *OpError
	if !errors.As(err, &oe) {
		return err
	}
	op := oe.Op
	claude := !op.Item && (op.Type == OpLink || op.Type == OpCopy)
	switch {
	case claude && oe.Stage == stageChmod:
		return fmt.Errorf("domain %q: chmod hooks: %w", name, oe.Err)
	case claude:
		return fmt.Errorf("domain %q: copying %s: %w", name, kind, oe.Err)
	case kind == lockfile.KindSkills:
		return fmt.Errorf("domain %q: %s: %w", name, kind, oe.Err)
	}
	return fmt.Errorf("domain %q: %s on %s: %w", name, kind, op.Platforms[0], oe.Err)
}

func skipItem(in *Installer, domain string, kind lockfile.Kind, srcDir, name string, plats []platform.ID) bool {
	return fsys.SkipTop(kind.SkipsReadme(), name) || in.excluded(domain, kind, name) || !hookWanted(kind, filepath.Join(srcDir, name), plats)
}

func (in *Installer) placeItems(c gitCache, domain string, kind lockfile.Kind, srcDir, dstDir string, plats []platform.ID) error {
	plan, err := in.planItems(domain, kind, srcDir, dstDir, plats)
	if err != nil {
		return err
	}
	plan.git = c
	if err := in.Apply(plan); err != nil {
		var oe *OpError
		if errors.As(err, &oe) {
			return oe.Err
		}
		return err
	}
	return nil
}

func (in *Installer) placeOne(c gitCache, domain string, kind lockfile.Kind, name, src, dst string, plats []platform.ID) error {
	if err := in.guard(c, dst); err != nil {
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
	if in.Mode == lockfile.ModeLink {
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
		if _, err := fsys.CopyTreeRoot(kind.SkipsReadme(), src, dst, in.HarnessRoot, in.DryRun); err != nil {
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
	hash, err := fsys.HashPath(dst)
	if err != nil {
		return fmt.Errorf("hashing %s: %w", dst, err)
	}
	in.Lock.Record(lockfile.Entry{
		Domain: domain, Kind: kind, Name: name, Mode: lockfile.ModeCopy,
		Path: dst, Source: src, SHA256: hash, Platforms: pl, InstalledAt: time.Now(),
	})
	return nil
}

func (in *Installer) isForeign(dst string) bool {
	if _, err := os.Lstat(dst); err != nil {
		return false
	}
	if _, ok := in.Lock.Covering(dst); ok || in.Force || in.overwrites(dst) {
		return false
	}
	return true
}

func (in *Installer) refuseForeign(dst string) error {
	if in.isForeign(dst) {
		return &ConflictError{Path: dst, Reason: ReasonForeign}
	}
	return nil
}

func (in *Installer) writeRendered(c gitCache, domain string, kind lockfile.Kind, name, src, dst string, srcData, out []byte, p platform.ID) error {
	if err := in.guard(c, dst); err != nil {
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
		Path: dst, Source: src, SHA256: fsys.HashBytes(out), SrcSHA256: fsys.HashBytes(srcData),
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

func (in *Installer) syncAgentsMD() error {
	path := filepath.Join(in.ProjectRoot, agentsMD)
	rulesRoot := filepath.Join(in.ProjectRoot, platform.AgentsRulesDir)
	var files []string
	var plats []string
	for _, e := range in.Lock.Entries {
		if e.Kind != lockfile.KindRules || (e.Mode != lockfile.ModeLink && e.Mode != lockfile.ModeCopy) || !fsys.Within(rulesRoot, e.Path) {
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
		files = append(files, platform.AgentsRulesDir+"/"+filepath.Base(e.Path))
		plats = unionPlats(plats, hit)
	}
	sort.Strings(files)
	existing, rerr := os.ReadFile(path)
	exists := rerr == nil
	prior, tracked := in.Lock.FindAt("as-skill", lockfile.KindInstructions, agentsMD, path)
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
		in.Lock.RemoveAt("as-skill", lockfile.KindInstructions, agentsMD, path)
		return nil
	}
	if in.DryRun {
		fmt.Printf("would update   %s\n", path)
		return nil
	}
	out := render.UpsertBlock(existing, render.AgentsBlockBody(files))
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
		Domain: "as-skill", Kind: lockfile.KindInstructions, Name: agentsMD, Mode: lockfile.ModeBlock,
		Path: path, Source: src, Platforms: plats, InstalledAt: time.Now(),
	})
	return nil
}
