package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/fsys"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/render"
)

const (
	stagePlace = "place"
	stageChmod = "chmod"
)

// OpError carries the failing op and stage so callers keep their error wording.
type OpError struct {
	Op    Op
	Stage string
	Err   error
}

func (e *OpError) Error() string { return e.Err.Error() }
func (e *OpError) Unwrap() error { return e.Err }

// Apply runs the plan in order. With in.DryRun it prints the same rows and
// writes nothing.
func (in *Installer) Apply(p *Plan) error {
	if p.git == nil {
		p.git = gitCache{}
	}
	for i, op := range p.Ops {
		in.emitWarnings(p, i)
		if err := in.applyOp(p.git, op); err != nil {
			return err
		}
	}
	in.emitWarnings(p, len(p.Ops))
	return nil
}

func (in *Installer) emitWarnings(p *Plan, before int) {
	for _, w := range p.Warnings {
		if w.Before != before || (w.WriteOnly && in.DryRun) {
			continue
		}
		if w.Stream == Stdout {
			fmt.Println(w.Text)
		} else {
			fmt.Fprintln(os.Stderr, w.Text)
		}
	}
}

func (in *Installer) applyOp(c gitCache, op Op) error {
	switch {
	case op.Item:
		if err := in.placeOne(c, op.Domain, op.Kind, op.Name, op.Src, op.Dst, op.Platforms); err != nil {
			return &OpError{op, stagePlace, err}
		}
		return nil
	case op.Type == OpRender:
		if err := in.applyRender(c, op); err != nil {
			return &OpError{op, stagePlace, err}
		}
		return nil
	case op.Type == OpBlock:
		if err := in.syncAgentsMD(); err != nil {
			return &OpError{op, stagePlace, err}
		}
		return nil
	case op.Type == OpHook:
		if !in.DryRun && op.Mode == lockfile.ModeCopy {
			if err := fsys.MakeShellScriptsExecutable(op.Dst); err != nil {
				return &OpError{op, stageChmod, err}
			}
		}
		return nil
	}
	prior := in.snapshotPlatforms(op.Domain, op.Kind, op.Dst)
	if op.Kind != lockfile.KindSkill {
		defer in.applyPlatforms(op.Domain, op.Kind, op.Dst, op.Platforms, prior)
	}
	if len(op.Exclude) > 0 && op.Type == OpCopy {
		if err := in.placeItems(c, op.Domain, op.Kind, op.Src, op.Dst, op.Platforms); err != nil {
			return &OpError{op, stagePlace, err}
		}
		if !in.DryRun {
			in.keepExcluded(op.Domain, op.Kind, op.Dst, func(item string) bool { return in.excluded(op.Domain, op.Kind, item) })
		}
		return nil
	}
	n, err := in.place(op.Domain, op.Kind, op.Name, op.Src, op.Dst)
	if err != nil {
		return &OpError{op, stagePlace, err}
	}
	if op.Kind == lockfile.KindSkill {
		in.applyPlatforms(op.Domain, op.Kind, op.Dst, op.Platforms, prior)
	}
	in.report(op.Domain, string(op.Kind), op.Dst, n)
	if op.Kind == lockfile.KindHooks && op.Type == OpCopy && !in.DryRun {
		if err := fsys.MakeShellScriptsExecutable(op.Dst); err != nil {
			return &OpError{op, stageChmod, err}
		}
	}
	return nil
}

// renderOp reads op.Src and renders it; writes nothing.
func (in *Installer) renderOp(op Op) (data, out []byte, pf platform.Platform, err error) {
	data, err = os.ReadFile(op.Src)
	if err != nil {
		return nil, nil, pf, err
	}
	pf, _ = platform.Get(op.Platforms[0])
	stem := strings.TrimSuffix(op.Name, ".md")
	if op.Kind == lockfile.KindAgents {
		promptRel := ""
		if in.Mode == lockfile.ModeLink {
			promptRel = platform.PromptsDir + "/" + op.Name
		}
		out, err = render.RenderAgent(data, filepath.ToSlash(relToHarness(in.HarnessRoot, op.Src)), stem, pf.Agents.Format, in.Mode, promptRel)
		if err != nil {
			return nil, nil, pf, err
		}
	} else {
		out = render.MDCStub(stem, platform.AgentsRulesDir+"/"+op.Name, string(data), in.Mode)
	}
	return data, out, pf, nil
}

func (in *Installer) applyRender(c gitCache, op Op) error {
	data, out, pf, err := in.renderOp(op)
	if err != nil {
		return err
	}
	return in.writeRendered(c, op.Domain, op.Kind, op.Name, op.Src, op.Dst, data, out, pf.ID)
}
