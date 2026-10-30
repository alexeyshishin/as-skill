package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/hookcfg"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
)

type HookInfo struct {
	Name      string
	Event     string
	Platforms []platform.ID
}

type srcHook struct {
	HookInfo
	reg hookcfg.HookReg
}

func (in *Installer) sourceHooks(reg *harness.Registry, domains []string) ([]srcHook, error) {
	var out []srcHook
	seen := map[string]bool{}
	for _, d := range domains {
		if _, ok := reg.Domains[d]; !ok {
			return nil, fmt.Errorf("unknown domain %q", d)
		}
		dir := harness.KindDir(in.HarnessRoot, d, lockfile.KindHooks)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".sh") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, n := range names {
			path := filepath.Join(dir, n)
			ev, m, ok := hookcfg.ReadRegisterHeader(path)
			if !ok || seen[n] {
				continue
			}
			seen[n] = true
			h := srcHook{reg: hookcfg.HookReg{Name: n, Event: ev, Matcher: m}}
			h.Name, h.Event = n, ev
			for _, id := range platform.IDs() {
				if spec, _ := platform.Get(id); spec.Hooks.Schema != platform.HooksNone && hookcfg.HookTargets(path, id) {
					h.Platforms = append(h.Platforms, id)
				}
			}
			out = append(out, h)
		}
	}
	return out, nil
}

func (in *Installer) DomainHooks(reg *harness.Registry, domains []string) ([]HookInfo, error) {
	hs, err := in.sourceHooks(reg, domains)
	if err != nil {
		return nil, err
	}
	out := make([]HookInfo, len(hs))
	for i, h := range hs {
		out[i] = h.HookInfo
	}
	return out, nil
}

func (in *Installer) DomainHookNames(reg *harness.Registry, domains []string, p platform.ID) ([]string, error) {
	hs, err := in.DomainHooks(reg, domains)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, h := range hs {
		for _, id := range h.Platforms {
			if id == p {
				out = append(out, h.Name)
			}
		}
	}
	return out, nil
}

func (in *Installer) RegisterHooks(reg *harness.Registry, domains []string, p platform.ID, names []string) error {
	if !in.DryRun {
		return in.EnableHooks(p, names)
	}
	spec, ok := platform.Get(p)
	if !ok {
		return fmt.Errorf("unknown platform %q", p)
	}
	if spec.Hooks.Schema == platform.HooksNone {
		fmt.Printf("hooks unsupported on %s\n", p)
		return nil
	}
	if err := in.refuseGlobalSettings(p); err != nil {
		return err
	}
	hs, err := in.sourceHooks(reg, domains)
	if err != nil {
		return err
	}
	var regs []hookcfg.HookReg
	for _, h := range hs {
		for _, n := range names {
			if n != h.Name {
				continue
			}
			for _, id := range h.Platforms {
				if id == p {
					regs = append(regs, h.reg)
				}
			}
		}
	}
	return in.applyRegs(p, regs)
}
