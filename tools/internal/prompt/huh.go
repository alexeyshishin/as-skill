package prompt

import (
	"errors"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
)

type huhPrompter struct {
	in  io.Reader // nil = terminal stdin
	out io.Writer
}

// NewHuh returns the terminal Prompter backed by charmbracelet/huh.
func NewHuh() Prompter { return &huhPrompter{out: os.Stderr} }

func (p *huhPrompter) run(f huh.Field) error {
	km := huh.NewDefaultKeyMap()
	km.MultiSelect.Toggle = key.NewBinding(key.WithKeys(" ", "x"), key.WithHelp("space/x", "toggle"))
	form := huh.NewForm(huh.NewGroup(f)).WithKeyMap(km).WithOutput(p.out).WithAccessible(false)
	if p.in != nil {
		form = form.WithInput(p.in)
	}
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return ErrAborted
		}
		return err
	}
	return nil
}

func (p *huhPrompter) Input(id, title, def string, validate func(string) error) (string, error) {
	v := def
	in := huh.NewInput().Title(title).Value(&v)
	if validate != nil {
		in = in.Validate(validate)
	}
	if err := p.run(in); err != nil {
		return "", err
	}
	return v, nil
}

func huhOptions(opts []Option) []huh.Option[string] {
	out := make([]huh.Option[string], len(opts))
	for i, o := range opts {
		out[i] = huh.NewOption(o.Label, o.Value).Selected(o.Selected)
	}
	return out
}

func (p *huhPrompter) MultiSelect(id, title string, opts []Option, validate func([]string) error) ([]string, error) {
	var vals []string
	var ms *huh.MultiSelect[string]
	hovered := ""
	ms = huh.NewMultiSelect[string]().Title(title).
		Description("space/x — toggle, enter — confirm (none toggled = highlighted one)").
		Options(huhOptions(opts)...).Value(&vals).
		Validate(func(v []string) error {
			v = dedupe(v)
			if len(v) == 0 {
				if h, ok := ms.Hovered(); ok {
					if validate != nil {
						if err := validate([]string{h}); err != nil {
							return err
						}
					}
					hovered = h
					return nil
				}
			}
			if validate != nil {
				return validate(v)
			}
			return nil
		})
	if err := p.run(ms); err != nil {
		return nil, err
	}
	vals = dedupe(vals)
	if len(vals) == 0 && hovered != "" {
		vals = []string{hovered}
	}
	return vals, nil
}

func (p *huhPrompter) Select(id, title string, opts []Option, def string) (string, error) {
	v := def
	if err := p.run(huh.NewSelect[string]().Title(title).Options(huhOptions(opts)...).Value(&v)); err != nil {
		return "", err
	}
	return v, nil
}

func (p *huhPrompter) Confirm(id, title, desc string, def bool) (bool, error) {
	v := def
	c := huh.NewConfirm().Title(title).Description(desc).Value(&v)
	if err := p.run(c); err != nil {
		return false, err
	}
	return v, nil
}

// dedupe drops repeated values, keeping first-seen order.
func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
