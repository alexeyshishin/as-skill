package cli

import (
	"errors"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/prompt"
)

type skillDomain struct {
	name   string
	skills []harness.SkillInfo
}

func customizableDomains(harnessRoot string, reg *harness.Registry, sel selection) []skillDomain {
	var out []skillDomain
	for _, d := range sel.Domains {
		if m, ok := reg.Domains[d.Name]; !ok || gatedReason(m) != "" {
			continue
		}
		sk, err := harness.Skills(harnessRoot, d.Name)
		if err != nil || len(sk) == 0 {
			continue
		}
		out = append(out, skillDomain{d.Name, sk})
	}
	return out
}

func skillLabel(s harness.SkillInfo) string {
	label := s.Name
	if s.Description != "" {
		d := []rune(s.Description)
		if len(d) > 50 {
			d = append(d[:49], '…')
		}
		label += " — " + string(d)
	}
	return label
}

var errNoSkills = errors.New("select at least one skill; to skip a domain's skills entirely, don't choose the domain")

func requireSkill(s []string) error {
	if len(s) == 0 {
		return errNoSkills
	}
	return nil
}

func askSkillSelection(p prompt.Prompter, doms []skillDomain, prev map[string]bool) (excl map[string]bool, back bool, err error) {
	cur := map[string]bool{}
	for k, v := range prev {
		cur[k] = v
	}
	for i := 0; i < len(doms); {
		d := doms[i]
		opts := make([]prompt.Option, len(d.skills))
		for j, s := range d.skills {
			opts[j] = prompt.Option{Value: s.Name, Label: skillLabel(s), Selected: !cur[d.name+"/"+s.Name]}
		}
		got, err := p.MultiSelect("skills:"+d.name, "Skills to install from "+d.name, opts, requireSkill)
		if errors.Is(err, prompt.ErrBack) {
			if i == 0 {
				return nil, true, nil
			}
			i--
			continue
		}
		if err != nil {
			return nil, false, abortErr(err)
		}
		if err := requireSkill(got); err != nil {
			return nil, false, err
		}
		keep := map[string]bool{}
		for _, g := range got {
			keep[g] = true
		}
		for _, s := range d.skills {
			if key := d.name + "/" + s.Name; keep[s.Name] {
				delete(cur, key)
			} else {
				cur[key] = true
			}
		}
		i++
	}
	excl = map[string]bool{}
	for _, d := range doms {
		for _, s := range d.skills {
			if key := d.name + "/" + s.Name; cur[key] {
				excl[key] = true
			}
		}
	}
	return excl, false, nil
}

type customizeState struct {
	on   bool
	excl map[string]bool
}

func askCustomize(p prompt.Prompter, doms []skillDomain, st *customizeState) (map[string]bool, error) {
	for {
		on, err := p.Confirm("customize", "Customize skills?",
			"Deselect individual skills inside the chosen domains; their rules, agents and hooks are still installed.", st.on)
		if err != nil {
			return nil, abortErr(err)
		}
		st.on = on
		if !on {
			return nil, nil
		}
		excl, back, err := askSkillSelection(p, doms, st.excl)
		if err != nil {
			return nil, err
		}
		if back {
			continue
		}
		st.excl = excl
		if len(excl) == 0 {
			return nil, nil
		}
		return excl, nil
	}
}
