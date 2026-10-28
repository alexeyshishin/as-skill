package cli

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"claude-harness/tools/internal/health"
	"claude-harness/tools/internal/installer"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/prompt"
	"claude-harness/tools/internal/registry"
)

var interactive = prompt.IsInteractive

var newPrompter = func() prompt.Prompter { return prompt.NewHuh() }

var errAborted = errors.New("aborted")

func abortErr(err error) error {
	if errors.Is(err, prompt.ErrAborted) {
		return errAborted
	}
	return err
}

// wantDialog: a terminal, no -y, and not a fully specified dry-run.
func wantDialog(o addOpts) bool {
	if o.yes || (o.dryRun && len(o.names) > 0) {
		return false
	}
	return interactive()
}

func askProject(p prompt.Prompter, o addOpts) (string, error) {
	if o.projectSet {
		return resolveProject(o.project)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	v, err := p.Input("project", "Project directory", cwd, func(s string) error {
		_, err := resolveProject(s)
		return err
	})
	if err != nil {
		return "", abortErr(err)
	}
	if strings.TrimSpace(v) == "" {
		v = cwd
	}
	return resolveProject(v)
}

func gatedLabel(m *registry.Manifest) string {
	for _, e := range m.RequiresEnv {
		if os.Getenv(e) == "" {
			return "needs $" + e
		}
	}
	for _, b := range m.RequiresBin {
		if gatedReason(&registry.Manifest{RequiresBin: []string{b}}) != "" {
			return "needs " + b + " on $PATH"
		}
	}
	return ""
}

func askDomains(p prompt.Prompter, harnessRoot string, reg *registry.Registry, lock *lockfile.Lockfile) ([]string, error) {
	var opts []prompt.Option
	for _, n := range reg.DomainNames() {
		m := reg.Domains[n]
		label := n
		if m.Description != "" {
			d := []rune(m.Description)
			if len(d) > 50 {
				d = append(d[:49], '…')
			}
			label += " — " + string(d)
		}
		if g := gatedLabel(m); g != "" {
			label += " (" + g + ")"
		}
		if domainInstalled(lock, harnessRoot, n) {
			label += " [installed]"
		}
		opts = append(opts, prompt.Option{Value: n, Label: label})
	}
	validate := func(sel []string) error {
		if len(sel) == 0 {
			return errors.New("select at least one domain")
		}
		for _, n := range sel {
			if m, ok := reg.Domains[n]; ok {
				if g := gatedLabel(m); g != "" {
					return fmt.Errorf("domain %q is gated (%s)", n, g)
				}
			}
		}
		return nil
	}
	got, err := p.MultiSelect("domains", "Domains to add", opts, validate)
	if err != nil {
		return nil, abortErr(err)
	}
	if err := validate(got); err != nil {
		return nil, err
	}
	return got, nil
}

func allPlatformOptions(selected map[platform.ID]bool, note string) []prompt.Option {
	var opts []prompt.Option
	for _, id := range platform.IDs() {
		l := string(id)
		if selected[id] && note != "" {
			l += " (" + note + ")"
		}
		opts = append(opts, prompt.Option{Value: string(id), Label: l, Selected: selected[id]})
	}
	return opts
}

func askPlatforms(p prompt.Prompter, detected []platform.ID, real bool) ([]platform.ID, error) {
	sel := map[platform.ID]bool{}
	for _, id := range detected {
		sel[id] = true
	}
	note := ""
	if real {
		note = "detected"
	}
	got, err := p.MultiSelect("platforms", "Platforms", allPlatformOptions(sel, note), func(s []string) error {
		if len(s) == 0 {
			return errors.New("select at least one platform")
		}
		return nil
	})
	if err != nil {
		return nil, abortErr(err)
	}
	if len(got) == 0 {
		return nil, errors.New("no platform selected")
	}
	return platform.ParsePlatforms(got)
}

func askMode(p prompt.Prompter) (string, error) {
	v, err := p.Select("mode", "Install mode", []prompt.Option{
		{Value: "link", Label: "symlink — edits in the checkout show up immediately", Selected: true},
		{Value: "copy", Label: "copy — static snapshot, independent of the checkout"},
	}, "link")
	if err != nil {
		return "", abortErr(err)
	}
	return v, nil
}

func confirm(p prompt.Prompter, title, summary string) error {
	ok, err := p.Confirm("confirm", title, summary, true)
	if err != nil {
		return abortErr(err)
	}
	if !ok {
		return errAborted
	}
	return nil
}

type addPlan struct {
	project, harness string
	reg              *registry.Registry
	sel              selection
	lock             *lockfile.Lockfile
	plats            []platform.ID
	detected         bool
	dialog           bool
	mode             string
}

func addDialog(opts addOpts) (*addPlan, error) {
	p := newPrompter()
	pl := &addPlan{dialog: true}
	var err error
	if pl.project, err = askProject(p, opts); err != nil {
		return nil, err
	}
	if pl.harness, err = registry.ResolveHarnessRootAuto(opts.harnessRoot); err != nil {
		return nil, err
	}
	if pl.reg, err = registry.LoadRegistry(pl.harness); err != nil {
		return nil, err
	}
	if pl.lock, err = lockfile.Load(pl.project); err != nil {
		return nil, err
	}
	names := opts.names
	if len(names) == 0 {
		if names, err = askDomains(p, pl.harness, pl.reg, pl.lock); err != nil {
			return nil, err
		}
	}
	if pl.sel, err = resolveNames(pl.harness, pl.reg, names); err != nil {
		return nil, err
	}
	if len(pl.sel.Domains) == 0 && len(pl.sel.Skills) == 0 {
		return nil, errors.New("nothing to add")
	}
	det, src := detectPlatforms(pl.project, pl.lock)
	platSrc := "default"
	if len(src) > 0 {
		platSrc = "detected from " + strings.Join(uniqStrings(src), ", ")
	}
	if len(opts.agents) > 0 {
		platSrc = "-a"
		if pl.plats, err = platform.ParsePlatforms(opts.agents); err != nil {
			return nil, err
		}
	} else if pl.plats, err = askPlatforms(p, det, len(src) > 0); err != nil {
		return nil, err
	}
	pl.mode = "link"
	if opts.copy {
		pl.mode = "copy"
	} else if pl.mode, err = askMode(p); err != nil {
		return nil, err
	}
	if opts.dryRun {
		return pl, nil
	}
	if err := confirm(p, "Install?", addSummary(opts, pl, platSrc)); err != nil {
		return nil, err
	}
	return pl, nil
}

func addSummary(opts addOpts, pl *addPlan, platSrc string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "harness:   %s\n", pl.harness)
	if opts.harnessRoot == "" {
		c, cok := registry.CwdHarnessRoot()
		e, eok := registry.ExecutableHarnessRoot()
		if cok && eok && !sameDir(c, e) {
			fmt.Fprintf(&b, "warning:   cwd checkout %s differs from binary checkout %s; using %s\n", c, e, pl.harness)
		}
	}
	fmt.Fprintf(&b, "project:   %s\n", pl.project)
	if d := pl.sel.domainNames(); len(d) > 0 {
		fmt.Fprintf(&b, "domains:   %s\n", strings.Join(d, ", "))
	}
	if len(pl.sel.Skills) > 0 {
		fmt.Fprintf(&b, "skills:    %s\n", strings.Join(pl.sel.Skills, ", "))
	}
	fmt.Fprintf(&b, "platforms: %s (%s)\n", strings.Join(platformDirNames(pl.plats), ", "), platSrc)
	for _, id := range pl.plats {
		if g := installer.CapGap(id); g != "" {
			fmt.Fprintf(&b, "warning:   %s: %s\n", id, g)
		}
	}
	for _, d := range health.Duplicates(pl.project, pl.lock) {
		fmt.Fprintf(&b, "warning:   %s\n", d)
	}
	mode := "symlink"
	if pl.mode == "copy" {
		mode = "copy"
	}
	fmt.Fprintf(&b, "mode:      %s", mode)
	return b.String()
}

// removeChoices lists installed domains and standalone skills from the lockfile.
func removeChoices(lock *lockfile.Lockfile) []prompt.Option {
	domains := map[string]bool{}
	skills := map[string]bool{}
	// A domain is "whole" when any of its rows is not a named skill row; a
	// domain with only named skill rows (install skill, or a claude per-item
	// link of skills alone; the lockfile loader folds kind "skill" into
	// "skills") is offered as standalone skills.
	whole := map[string]bool{}
	for _, e := range lock.Entries {
		if e.Domain == "as-skill" || e.Kind == "instructions" {
			continue
		}
		if !((e.Kind == "skill" || e.Kind == "skills") && e.Name != "") {
			whole[e.Domain] = true
		}
	}
	for _, e := range lock.Entries {
		if e.Domain == "as-skill" || e.Kind == "instructions" {
			continue
		}
		if whole[e.Domain] {
			domains[e.Domain] = true
		} else if e.Name != "" {
			skills[e.Name] = true
		}
	}
	var opts []prompt.Option
	for _, d := range sortedKeys(domains) {
		opts = append(opts, prompt.Option{Value: d, Label: d + " (domain)"})
	}
	for _, s := range sortedKeys(skills) {
		if domains[s] {
			continue
		}
		opts = append(opts, prompt.Option{Value: s, Label: s + " (skill)"})
	}
	return opts
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func askRemoveNames(p prompt.Prompter, lock *lockfile.Lockfile, project string) ([]string, error) {
	opts := removeChoices(lock)
	if len(opts) == 0 {
		return nil, fmt.Errorf("nothing installed in %s (see `as-skill doctor`)", project)
	}
	got, err := p.MultiSelect("remove", "Remove", opts, func(s []string) error {
		if len(s) == 0 {
			return errors.New("select at least one item")
		}
		return nil
	})
	if err != nil {
		return nil, abortErr(err)
	}
	if len(got) == 0 {
		return nil, errors.New("nothing selected")
	}
	return got, nil
}

func askRemovePlatforms(p prompt.Prompter, recorded []platform.ID) ([]platform.ID, error) {
	sel := map[platform.ID]bool{}
	for _, id := range recorded {
		sel[id] = true
	}
	var opts []prompt.Option
	for _, id := range recorded {
		opts = append(opts, prompt.Option{Value: string(id), Label: string(id), Selected: true})
	}
	got, err := p.MultiSelect("platforms", "Platforms to remove from", opts, func(s []string) error {
		if len(s) == 0 {
			return errors.New("select at least one platform")
		}
		return nil
	})
	if err != nil {
		return nil, abortErr(err)
	}
	if len(got) == 0 {
		return nil, errors.New("no platform selected")
	}
	return platform.ParsePlatforms(got)
}

func removeSummary(project string, names []string, plats []platform.ID) string {
	return fmt.Sprintf("project:   %s\nremove:    %s\nplatforms: %s", project,
		strings.Join(names, ", "), strings.Join(platformDirNames(plats), ", "))
}
