package installer

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"claude-harness/tools/internal/fsys"
	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

func (in *Installer) claudeOp(domain string, kind lockfile.Kind, name, src, dst string, plats []platform.ID) Op {
	t := OpCopy
	if in.Mode == lockfile.ModeLink {
		t = OpLink
	}
	op := Op{Type: t, Src: src, Dst: dst, Mode: in.Mode, Domain: domain, Kind: kind, Name: name, Platforms: plats}
	if kind == lockfile.KindSkills && in.hasExclusions(domain) {
		prefix := domain + "/"
		for k, v := range in.ExcludeSkills {
			if v && strings.HasPrefix(k, prefix) {
				op.Exclude = append(op.Exclude, strings.TrimPrefix(k, prefix))
			}
		}
		slices.Sort(op.Exclude)
	}
	return op
}

func (in *Installer) copyWarnings(p *Plan, op Op) {
	if op.Type != OpCopy || len(op.Exclude) > 0 {
		return
	}
	if entry, ok := in.Lock.Owns(op.Dst); ok && entry.Mode != lockfile.ModeCopy {
		p.Warnings = append(p.Warnings, Warning{
			Text:   fmt.Sprintf("as-skill: warning: %s was installed via %q, replacing with a copy — mixed channels may cause confusion", op.Dst, entry.Mode),
			Stream: Stderr, Before: len(p.Ops) - 1,
		})
	}
}

func (in *Installer) planClaudeKind(m *harness.Manifest, name string, kind lockfile.Kind, srcDir string, plats []platform.ID) (*Plan, error) {
	destPath, err := in.resolveTarget(m, kind)
	if err != nil {
		return nil, fmt.Errorf("domain %q: %w", name, err)
	}
	p := &Plan{}
	op := in.claudeOp(name, kind, "", srcDir, destPath, plats)
	p.Ops = append(p.Ops, op)
	in.copyWarnings(p, op)
	if kind == lockfile.KindHooks && op.Type == OpLink {
		for _, t := range hookExecWarnings(srcDir) {
			p.Warnings = append(p.Warnings, Warning{Text: t, Stream: Stderr, Before: len(p.Ops), WriteOnly: true})
		}
	}
	return p, nil
}

func (in *Installer) planClaudeSkill(m *harness.Manifest, owner, name, src string, plats []platform.ID) (*Plan, error) {
	destBase, err := in.resolveTarget(m, lockfile.KindSkills)
	if err != nil {
		return nil, fmt.Errorf("skill %q: %w", name, err)
	}
	p := &Plan{}
	op := in.claudeOp(owner, lockfile.KindSkill, name, src, filepath.Join(destBase, name), plats)
	p.Ops = append(p.Ops, op)
	in.copyWarnings(p, op)
	return p, nil
}

func hookExecWarnings(srcDir string) []string {
	var out []string
	for _, p := range fsys.NonExecutableScripts(srcDir) {
		out = append(out, fmt.Sprintf("as-skill: warning: %s is not executable; the symlinked hook won't be either (fix it in domains/)", p))
	}
	return out
}
