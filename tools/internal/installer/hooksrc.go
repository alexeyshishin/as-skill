package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/registry"
)

// HookInfo describes one registerable hook of a domain, read from source.
type HookInfo struct {
	Name      string
	Event     string
	Platforms []platform.ID // hook-capable platforms its `# platforms:` header allows
}

type srcHook struct {
	HookInfo
	reg hookReg
}

// sourceHooks lists hooks with a `# register:` header in the domains' source
// hooks dirs (the destination is never read), deduped by name, in domain order.
func (in *Installer) sourceHooks(reg *registry.Registry, domains []string) ([]srcHook, error) {
	var out []srcHook
	seen := map[string]bool{}
	for _, d := range domains {
		if _, ok := reg.Domains[d]; !ok {
			return nil, fmt.Errorf("unknown domain %q", d)
		}
		dir := filepath.Join(in.HarnessRoot, "domains", d, "hooks")
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
			ev, m, ok := readRegisterHeader(path)
			if !ok || seen[n] {
				continue
			}
			seen[n] = true
			h := srcHook{reg: hookReg{Name: n, Event: ev, Matcher: m}}
			h.Name, h.Event = n, ev
			for _, id := range platform.IDs() {
				if spec, _ := platform.Get(id); spec.Hooks.Schema != platform.HooksNone && hookTargets(path, id) {
					h.Platforms = append(h.Platforms, id)
				}
			}
			out = append(out, h)
		}
	}
	return out, nil
}

// DomainHooks lists the registerable hooks of the domains from source.
func (in *Installer) DomainHooks(reg *registry.Registry, domains []string) ([]HookInfo, error) {
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

// DomainHookNames returns the hooks of the domains registerable on p: they
// carry a `# register:` header and `# platforms:` allows p. Nil for a
// platform without hook support.
func (in *Installer) DomainHookNames(reg *registry.Registry, domains []string, p platform.ID) ([]string, error) {
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

// RegisterHooks registers the named hooks of the domains on p. A real run
// reads the placed hook files like EnableHooks; a dry run builds the entries
// from the domains' source, so it needs no prior placement.
func (in *Installer) RegisterHooks(reg *registry.Registry, domains []string, p platform.ID, names []string) error {
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
	var regs []hookReg
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
