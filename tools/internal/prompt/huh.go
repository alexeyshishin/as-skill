package prompt

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

const backHint = "shift+tab back"

type huhPrompter struct {
	in  io.Reader
	out io.Writer
}

func NewHuh() Prompter { return &huhPrompter{out: os.Stderr} }

type formModel struct {
	form *huh.Form
	back bool
}

func (m *formModel) Init() tea.Cmd { return m.form.Init() }

func (m *formModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "shift+tab" {
		m.back = true
		return m, tea.Quit
	}
	_, cmd := m.form.Update(msg)
	if m.form.State != huh.StateNormal {
		return m, tea.Quit
	}
	return m, cmd
}

func (m *formModel) View() string {
	if m.back || m.form.State != huh.StateNormal {
		return ""
	}
	return m.form.View()
}

func (p *huhPrompter) run(f huh.Field) error {
	km := huh.NewDefaultKeyMap()
	km.MultiSelect.Toggle = key.NewBinding(key.WithKeys(" ", "x"), key.WithHelp("space/x", "toggle"))
	form := huh.NewForm(huh.NewGroup(f)).WithKeyMap(km).WithAccessible(false)
	form.SubmitCmd = tea.Quit
	form.CancelCmd = tea.Quit
	m := &formModel{form: form}
	opts := []tea.ProgramOption{tea.WithOutput(p.out)}
	if p.in != nil {
		opts = append(opts, tea.WithInput(p.in))
	}
	if _, err := tea.NewProgram(m, opts...).Run(); err != nil {
		return err
	}
	switch {
	case m.back:
		return ErrBack
	case form.State == huh.StateAborted:
		return ErrAborted
	}
	return nil
}

func (p *huhPrompter) Input(id, title, def string, validate func(string) error) (string, error) {
	v := def
	in := huh.NewInput().Title(title).Description(backHint).Value(&v)
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
	return p.multiSelect(title, opts, validate, false)
}

func (p *huhPrompter) MultiSelectOptional(id, title string, opts []Option) ([]string, error) {
	return p.multiSelect(title, opts, nil, true)
}

func (p *huhPrompter) multiSelect(title string, opts []Option, validate func([]string) error, optional bool) ([]string, error) {
	var vals []string
	var ms *huh.MultiSelect[string]
	hovered := ""
	preselected := optional
	for _, o := range opts {
		if o.Selected {
			preselected = true
		}
	}
	ms = huh.NewMultiSelect[string]().Title(title).
		Description("space/x toggle · enter confirm · shift+tab back").
		Options(huhOptions(opts)...).Value(&vals).
		Validate(func(v []string) error {
			v = dedupe(v)
			if len(v) == 0 && !preselected {
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
	if err := p.run(huh.NewSelect[string]().Title(title).Description(backHint).Options(huhOptions(opts)...).Value(&v)); err != nil {
		return "", err
	}
	return v, nil
}

func (p *huhPrompter) Confirm(id, title, desc string, def bool) (bool, error) {
	v := def
	c := huh.NewConfirm().Title(title).Description(strings.TrimRight(desc+"\n"+backHint, "\n")).Value(&v)
	if err := p.run(c); err != nil {
		return false, err
	}
	return v, nil
}

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
