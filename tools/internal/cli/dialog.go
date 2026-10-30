package cli

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/health"
	"claude-harness/tools/internal/installer"
	"claude-harness/tools/internal/lockfile"
	"claude-harness/tools/internal/platform"
	"claude-harness/tools/internal/prompt"
)

var errAborted = errors.New("aborted")

func abortErr(err error) error {
	if errors.Is(err, prompt.ErrAborted) {
		return errAborted
	}
	return err
}

func runSteps(steps []func() (bool, error)) error {
	asked := make([]bool, len(steps))
	for i := 0; i < len(steps); {
		a, err := steps[i]()
		if errors.Is(err, prompt.ErrBack) {
			j := i - 1
			for j >= 0 && !asked[j] {
				j--
			}
			if j >= 0 {
				i = j
			}
			continue
		}
		if err != nil {
			return err
		}
		asked[i] = a
		i++
	}
	return nil
}

func wantDialog(d Deps, o addOpts) bool {
	if o.yes || (o.dryRun && len(o.names) > 0) {
		return false
	}
	return d.interactive()
}

func askProject(p prompt.Prompter, o addOpts, prev string) (string, error) {
	if o.projectSet {
		return resolveProject(o.project)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	def := cwd
	if prev != "" {
		def = prev
	}
	v, err := p.Input("project", "Project directory", def, func(s string) error {
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

func gatedLabel(m *harness.Manifest) string {
	miss, gated := m.Gate()
	if !gated {
		return ""
	}
	if miss.Kind == harness.MissEnv {
		return "needs $" + miss.Name
	}
	return "needs " + miss.Name + " on $PATH"
}

func askDomains(p prompt.Prompter, harnessRoot string, reg *harness.Registry, lock *lockfile.Lockfile, prev []string) ([]string, error) {
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
		opts = append(opts, prompt.Option{Value: n, Label: label, Selected: slices.Contains(prev, n)})
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

func allPlatformOptions(selected map[platform.ID]bool, note string, unignored map[platform.ID]string) []prompt.Option {
	var opts []prompt.Option
	for _, id := range platform.IDs() {
		l := string(id)
		if selected[id] && note != "" {
			l += " (" + note + ")"
		}
		if r := unignored[id]; r != "" {
			l += " [" + r + " not git-ignored]"
		}
		opts = append(opts, prompt.Option{Value: string(id), Label: l, Selected: selected[id]})
	}
	return opts
}

func unignoredRoots(project string, copyMode, force bool) map[platform.ID]string {
	out := map[platform.ID]string{}
	if copyMode || force {
		return out
	}
	for _, id := range platform.IDs() {
		in := &installer.Installer{ProjectRoot: project, Mode: lockfile.ModeLink, Platforms: []platform.ID{id}}
		var roots []string
		for _, c := range in.PlanRoots(lockfile.KindSkills, lockfile.KindAgents, lockfile.KindRules, lockfile.KindHooks).Conflicts {
			if c.Reason == installer.ReasonNotIgnored {
				roots = append(roots, c.Path)
			}
		}
		if len(roots) > 0 {
			out[id] = strings.Join(roots, ", ")
		}
	}
	return out
}

func askPlatforms(p prompt.Prompter, project string, copyMode, force bool, detected []platform.ID, real bool, prev []platform.ID) ([]platform.ID, error) {
	sel := map[platform.ID]bool{}
	for _, id := range detected {
		sel[id] = true
	}
	if prev != nil {
		sel = map[platform.ID]bool{}
		for _, id := range prev {
			sel[id] = true
		}
	}
	note := ""
	if real {
		note = "detected"
	}
	got, err := p.MultiSelect("platforms", "Platforms", allPlatformOptions(sel, note, unignoredRoots(project, copyMode, force)), func(s []string) error {
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

func askMode(p prompt.Prompter, prev lockfile.Mode) (lockfile.Mode, error) {
	if prev == "" {
		prev = lockfile.ModeLink
	}
	v, err := p.Select("mode", "Install mode", []prompt.Option{
		{Value: string(lockfile.ModeLink), Label: "symlink — edits in the checkout show up immediately", Selected: prev == lockfile.ModeLink},
		{Value: string(lockfile.ModeCopy), Label: "copy — static snapshot, independent of the checkout", Selected: prev == lockfile.ModeCopy},
	}, string(prev))
	if err != nil {
		return "", abortErr(err)
	}
	return lockfile.Mode(v), nil
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
	reg              *harness.Registry
	sel              selection
	lock             *lockfile.Lockfile
	plats            []platform.ID
	detected         bool
	dialog           bool
	mode             lockfile.Mode
	exclude          map[string]bool
	overwrite        map[string]bool
	adjust           []string
	hooks            []string
}

func addDialog(opts addOpts, d Deps) (*addPlan, error) {
	p := d.newPrompter()
	pl := &addPlan{dialog: true}
	var (
		projRaw   string
		prevDoms  []string
		prevPlats []platform.ID
		prevMode  lockfile.Mode
		platSrc   string
		rawPlats  []platform.ID
		rawMode   lockfile.Mode
		custom    customizeState
		prevRes   = map[string]string{}
		prevHooks []string
	)
	names := opts.names
	err := runSteps([]func() (bool, error){
		func() (bool, error) {
			var err error
			pl.project, err = askProject(p, opts, projRaw)
			if err == nil {
				projRaw = pl.project
			}
			return !opts.projectSet, err
		},
		func() (bool, error) {
			var err error
			if pl.harness, err = d.Locator.Locate(opts.harnessRoot); err != nil {
				return false, err
			}
			if pl.reg, err = harness.LoadRegistry(pl.harness); err != nil {
				return false, err
			}
			pl.lock, err = lockfile.Load(pl.project)
			return false, err
		},
		func() (bool, error) {
			if len(opts.names) > 0 {
				return false, nil
			}
			var err error
			if names, err = askDomains(p, pl.harness, pl.reg, pl.lock, prevDoms); err != nil {
				return true, err
			}
			prevDoms = names
			return true, nil
		},
		func() (bool, error) {
			var err error
			if pl.sel, err = resolveNames(pl.harness, pl.reg, names); err != nil {
				return false, err
			}
			if len(pl.sel.Domains) == 0 && len(pl.sel.Skills) == 0 {
				return false, errors.New("nothing to add")
			}
			return false, nil
		},
		func() (bool, error) {
			pl.exclude = nil
			doms := customizableDomains(pl.harness, pl.reg, pl.sel)
			if len(doms) == 0 {
				return false, nil
			}
			var err error
			pl.exclude, err = askCustomize(p, doms, &custom)
			return true, err
		},
		func() (bool, error) {
			det, src := detectPlatforms(pl.project, pl.lock)
			platSrc = "default"
			if len(src) > 0 {
				platSrc = "detected from " + strings.Join(uniqStrings(src), ", ")
			}
			var err error
			if len(opts.agents) > 0 {
				platSrc = "-a"
				pl.plats, err = platform.ParsePlatforms(opts.agents)
				rawPlats = pl.plats
				return false, err
			}
			if pl.plats, err = askPlatforms(p, pl.project, opts.copy, opts.force, det, len(src) > 0, prevPlats); err != nil {
				return true, err
			}
			prevPlats, rawPlats = pl.plats, pl.plats
			return true, nil
		},
		func() (bool, error) {
			pl.mode = lockfile.ModeLink
			if opts.copy {
				pl.mode = lockfile.ModeCopy
				rawMode = pl.mode
				return false, nil
			}
			var err error
			if pl.mode, err = askMode(p, prevMode); err != nil {
				return true, err
			}
			prevMode, rawMode = pl.mode, pl.mode
			return true, nil
		},
		func() (bool, error) {
			pl.plats, pl.mode = append([]platform.ID(nil), rawPlats...), rawMode
			pl.overwrite, pl.adjust = nil, nil
			return resolveConflicts(p, opts, pl, prevRes)
		},
		func() (bool, error) {
			pl.hooks = nil
			hopts, err := hookOptions(pl, prevHooks)
			if err != nil || len(hopts) == 0 {
				return false, err
			}
			got, err := p.MultiSelectOptional("hooks", "Register hooks (default: none)", hopts)
			if err != nil {
				return true, abortErr(err)
			}
			prevHooks, pl.hooks = got, got
			return true, nil
		},
		func() (bool, error) {
			if opts.dryRun {
				return false, nil
			}
			return true, confirm(p, "Install?", addSummary(opts, pl, platSrc, d.Locator))
		},
	})
	if err != nil {
		return nil, err
	}
	return pl, nil
}

func addSummary(opts addOpts, pl *addPlan, platSrc string, loc harness.Locator) string {
	var b strings.Builder
	fmt.Fprintf(&b, "harness:   %s\n", pl.harness)
	if opts.harnessRoot == "" {
		c, cok := loc.CwdRoot()
		e, eok := loc.ExecutableRoot()
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
	if ex := slices.Sorted(maps.Keys(pl.exclude)); len(ex) > 0 {
		fmt.Fprintf(&b, "deselect:  %s\n", strings.Join(ex, ", "))
		fmt.Fprintf(&b, "warning:   agents may reference deselected skills\n")
		for _, k := range ex {
			if skillInstalled(pl.lock, pl.project, k[strings.Index(k, "/")+1:]) {
				fmt.Fprintf(&b, "warning:   %s: already installed, deselected, kept — use remove\n", k)
			}
		}
	}
	fmt.Fprintf(&b, "platforms: %s (%s)\n", strings.Join(platformDirNames(pl.plats), ", "), platSrc)
	for _, a := range pl.adjust {
		fmt.Fprintf(&b, "adjusted:  %s\n", a)
	}
	for _, id := range pl.plats {
		if g := platform.CapGap(id); g != "" {
			fmt.Fprintf(&b, "warning:   %s: %s\n", id, g)
		}
	}
	for _, d := range health.Duplicates(pl.project, pl.lock) {
		fmt.Fprintf(&b, "warning:   %s\n", d)
	}
	if len(pl.hooks) == 0 {
		fmt.Fprintf(&b, "hooks:     none\n")
	} else {
		fmt.Fprintf(&b, "hooks:     %s\n", strings.Join(pl.hooks, ", "))
	}
	if slices.Contains(pl.hooks, loggerHook) {
		fmt.Fprintf(&b, "warning:   %s\n", loggerWarning)
	}
	mode := "symlink"
	if pl.mode == lockfile.ModeCopy {
		mode = "copy"
	}
	fmt.Fprintf(&b, "mode:      %s", mode)
	return b.String()
}

func removeChoices(lock *lockfile.Lockfile) []prompt.Option {
	domains := map[string]bool{}
	skills := map[string]bool{}
	whole := map[string]bool{}
	for _, e := range lock.Entries {
		if e.Domain == "as-skill" || e.Kind == lockfile.KindInstructions {
			continue
		}
		if !((e.Kind == lockfile.KindSkill || e.Kind == lockfile.KindSkills) && e.Name != "") {
			whole[e.Domain] = true
		}
	}
	for _, e := range lock.Entries {
		if e.Domain == "as-skill" || e.Kind == lockfile.KindInstructions {
			continue
		}
		if whole[e.Domain] {
			domains[e.Domain] = true
		} else if e.Name != "" {
			skills[e.Name] = true
		}
	}
	var opts []prompt.Option
	for _, d := range slices.Sorted(maps.Keys(domains)) {
		opts = append(opts, prompt.Option{Value: d, Label: d + " (domain)"})
	}
	for _, s := range slices.Sorted(maps.Keys(skills)) {
		if domains[s] {
			continue
		}
		opts = append(opts, prompt.Option{Value: s, Label: s + " (skill)"})
	}
	return opts
}

func askRemoveNames(p prompt.Prompter, lock *lockfile.Lockfile, project string, prev []string) ([]string, error) {
	opts := removeChoices(lock)
	for i := range opts {
		opts[i].Selected = slices.Contains(prev, opts[i].Value)
	}
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

func askRemovePlatforms(p prompt.Prompter, recorded, prev []platform.ID) ([]platform.ID, error) {
	var opts []prompt.Option
	for _, id := range recorded {
		opts = append(opts, prompt.Option{Value: string(id), Label: string(id), Selected: prev == nil || slices.Contains(prev, id)})
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

const (
	loggerHook    = "memory-bank-logger.sh"
	loggerWarning = "memory-bank-logger.sh sends the session transcript to the API and spawns a detached acceptEdits `claude -p` session"
)

func hookOptions(pl *addPlan, prev []string) ([]prompt.Option, error) {
	in := &installer.Installer{HarnessRoot: pl.harness}
	infos, err := in.DomainHooks(pl.reg, pl.sel.domainNames())
	if err != nil {
		return nil, err
	}
	var opts []prompt.Option
	for _, h := range infos {
		var on []string
		for _, id := range pl.plats {
			if slices.Contains(h.Platforms, id) {
				on = append(on, string(id))
			}
		}
		if len(on) == 0 {
			continue
		}
		label := fmt.Sprintf("%s — %s (%s)", h.Name, h.Event, strings.Join(on, ", "))
		if h.Name == loggerHook {
			label += " — SECURITY: " + loggerWarning
		}
		opts = append(opts, prompt.Option{Value: h.Name, Label: label, Selected: slices.Contains(prev, h.Name)})
	}
	return opts, nil
}

func registerSelected(in *installer.Installer, pl *addPlan) error {
	if len(pl.hooks) == 0 {
		return nil
	}
	for _, id := range pl.plats {
		names, err := in.DomainHookNames(pl.reg, pl.sel.domainNames(), id)
		if err != nil {
			return err
		}
		var chosen []string
		for _, n := range names {
			if slices.Contains(pl.hooks, n) {
				chosen = append(chosen, n)
			}
		}
		if len(chosen) == 0 {
			continue
		}
		if err := in.RegisterHooks(pl.reg, pl.sel.domainNames(), id, chosen); err != nil {
			return err
		}
	}
	return nil
}
