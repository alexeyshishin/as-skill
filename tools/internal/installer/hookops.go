package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/hookcfg"
	"claude-harness/tools/internal/platform"
)

func (in *Installer) unregisterHooks(hookNames []string) {
	in.unregisterHooksFor(platform.Claude, hookNames)
}

func (in *Installer) unregisterHooksFor(p platform.ID, hookNames []string) {
	spec, ok := platform.Get(p)
	if !ok || len(hookNames) == 0 || spec.Hooks.ConfigFile == "" {
		return
	}
	path := filepath.Join(in.ProjectRoot, spec.Hooks.ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — %v\n", path, err)
		}
		return
	}
	names := make(map[string]bool, len(hookNames))
	for _, n := range hookNames {
		names[n] = true
	}
	out, removed, err := hookcfg.RemoveRegistrations(data, in.ProjectRoot, names, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: skipping %s — %v, hook entries left in place\n", path, err)
		return
	}
	if len(removed) == 0 {
		return
	}
	verb := "unregistered  "
	if in.DryRun {
		verb = "would unregister"
	}
	for _, r := range removed {
		fmt.Printf("%s %s (%s) in %s\n", verb, r.Hook, r.Event, path)
	}
	if in.DryRun {
		return
	}
	if err := hookcfg.WriteFileAtomic(path, out); err != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: failed updating %s: %v\n", path, err)
	}
}

func (in *Installer) EnableHooks(p platform.ID, names []string) error {
	spec, ok := platform.Get(p)
	if !ok {
		return fmt.Errorf("unknown platform %q", p)
	}
	switch spec.Hooks.Schema {
	case platform.HooksNone:
		fmt.Printf("hooks unsupported on %s\n", p)
		return nil
	}
	if err := in.enableHooks(p, names); err != nil {
		return err
	}
	switch p {
	case platform.Codex:
		fmt.Println("UNVERIFIED: adapters follow vendor docs (2026-10-27), not run on a real CLI. Codex requires trusting project hooks: review in /hooks (or --dangerously-bypass-hook-trust for one-off).")
	case platform.Cursor:
		fmt.Println("UNVERIFIED: adapters follow vendor docs (2026-10-27), not run on a real CLI. Cursor stop cannot block; it submits a followup_message (cap 2).")
	}
	return nil
}

func (in *Installer) enableHooks(p platform.ID, names []string) error {
	spec, _ := platform.Get(p)
	if err := in.refuseGlobalSettings(p); err != nil {
		return err
	}
	regs, err := hookcfg.HookFiles(filepath.Join(in.ProjectRoot, spec.Hooks.Dir), names, p)
	if err != nil {
		return err
	}
	return in.applyRegs(p, regs)
}

func (in *Installer) refuseGlobalSettings(p platform.ID) error {
	if p == platform.Claude {
		if home, err := os.UserHomeDir(); err == nil && filepath.Clean(home) == filepath.Clean(in.ProjectRoot) {
			return fmt.Errorf("refusing: %s/.claude/settings.json is the global settings file", in.ProjectRoot)
		}
	}
	return nil
}

// applyRegs merges regs into the platform's hook config file (read-only in DryRun).
func (in *Installer) applyRegs(p platform.ID, regs []hookcfg.HookReg) error {
	spec, _ := platform.Get(p)
	path := filepath.Join(in.ProjectRoot, spec.Hooks.ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var also []byte
	if p == platform.Claude {
		also, _ = os.ReadFile(filepath.Join(in.ProjectRoot, platform.ClaudeHome, "settings.local.json"))
	}
	out, results, err := hookcfg.AddRegistrations(data, p, regs, also)
	if err != nil {
		fmt.Fprintf(os.Stderr, "as-skill: warning: %s: %v — left untouched\n", path, err)
		return fmt.Errorf("%s: %w", path, err)
	}
	any := false
	for _, r := range results {
		verb := "present"
		if r.Added {
			any = true
			verb = "registered"
			if in.DryRun {
				verb = "would register"
			}
		}
		fmt.Printf("%-15s %s (%s) in %s\n", verb, r.Name, r.Event, path)
	}
	if !any || in.DryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, serr := os.Stat(path); serr != nil {
		return os.WriteFile(path, out, 0o644)
	}
	return hookcfg.WriteFileAtomic(path, out)
}

func (in *Installer) DisableHooks(p platform.ID, names []string) error {
	spec, ok := platform.Get(p)
	if !ok {
		return fmt.Errorf("unknown platform %q", p)
	}
	if spec.Hooks.Schema == platform.HooksNone {
		fmt.Printf("hooks unsupported on %s\n", p)
		return nil
	}
	var list []string
	for _, n := range names {
		if n == "all" {
			list = nil
			entries, err := os.ReadDir(filepath.Join(in.ProjectRoot, spec.Hooks.Dir))
			if err != nil {
				return fmt.Errorf("reading hooks dir: %w", err)
			}
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".sh") {
					list = append(list, e.Name())
				}
			}
			break
		}
		if !strings.HasSuffix(n, ".sh") {
			n += ".sh"
		}
		list = append(list, n)
	}
	in.unregisterHooksFor(p, list)
	return nil
}
