package prompt

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

const (
	backHint    = "shift+tab back"
	projectHint = "tab complete · tab/↑/↓ choose · enter accept · shift+tab back"
)

type huhPrompter struct {
	in  io.Reader
	out io.Writer
}

func NewHuh() Prompter { return &huhPrompter{out: os.Stderr} }

const maxListRows = 10

type formModel struct {
	form    *huh.Form
	back    bool
	project *string
	list    []string
	open    bool
	sel     int
	off     int
	height  int
}

func (m *formModel) rows() int {
	if m.height == 0 {
		return maxListRows
	}
	return min(max(m.height-8, 0), maxListRows)
}

func (m *formModel) Init() tea.Cmd { return m.form.Init() }

func (m *formModel) hide() {
	m.list, m.open, m.sel, m.off = nil, false, -1, 0
}

func (m *formModel) setValue(v string) tea.Cmd {
	var cmds []tea.Cmd
	for _, step := range []tea.KeyMsg{{Type: tea.KeyCtrlE}, {Type: tea.KeyCtrlU}, {Type: tea.KeyRunes, Runes: []rune(v)}} {
		_, cmd := m.form.Update(step)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m *formModel) tab() tea.Cmd {
	if m.open && len(m.list) > 0 {
		m.move(1)
		return nil
	}
	next, matches := tabResult(*m.project)
	m.hide()
	var cmd tea.Cmd
	if next != *m.project {
		cmd = m.setValue(next)
	}
	if len(matches) >= 2 && m.rows() >= 2 {
		m.list, m.open = matches, true
	}
	return cmd
}

func (m *formModel) accept() tea.Cmd {
	v := m.list[m.sel]
	m.hide()
	return m.setValue(v)
}

func (m *formModel) move(delta int) {
	if !m.open {
		return
	}
	m.sel = wrapSel(m.sel, delta, len(m.list))
	m.off = scrollOff(m.off, m.sel, len(m.list), m.rows())
}

func (m *formModel) forward(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.form.Update(msg)
	if m.form.State != huh.StateNormal {
		return m, tea.Quit
	}
	return m, cmd
}

func (m *formModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, isKey := msg.(tea.KeyMsg)
	if isKey && k.String() == "shift+tab" {
		m.back = true
		return m, tea.Quit
	}
	if w, ok := msg.(tea.WindowSizeMsg); ok {
		m.height = w.Height
		if m.open {
			if m.rows() < 2 {
				m.hide()
			} else {
				m.off = scrollOff(m.off, m.sel, len(m.list), m.rows())
			}
		}
	}
	if m.project == nil || !isKey {
		return m.forward(msg)
	}
	switch {
	case k.Type == tea.KeyTab:
		return m, m.tab()
	case (k.Type == tea.KeyUp || k.Type == tea.KeyDown) && !k.Alt:
		if k.Type == tea.KeyDown {
			m.move(1)
		} else {
			m.move(-1)
		}
		return m, nil
	case k.Type == tea.KeyEnter:
		if m.open && m.sel >= 0 && m.sel < len(m.list) {
			return m, m.accept()
		}
		return m.forward(msg)
	}
	before := *m.project
	model, cmd := m.forward(msg)
	if *m.project != before {
		m.hide()
	}
	return model, cmd
}

func (m *formModel) View() string {
	if m.back || m.form.State != huh.StateNormal {
		return ""
	}
	view := m.form.View()
	if m.open {
		if list := renderMatches(m.list, m.sel, m.off, m.rows()); list != "" {
			view += "\n\n" + list
		}
	}
	return view
}

func newForm(f huh.Field) *huh.Form {
	km := huh.NewDefaultKeyMap()
	km.MultiSelect.Toggle = key.NewBinding(key.WithKeys(" ", "x"), key.WithHelp("space/x", "toggle"))
	form := huh.NewForm(huh.NewGroup(f)).WithKeyMap(km).WithAccessible(false)
	form.SubmitCmd = tea.Quit
	form.CancelCmd = tea.Quit
	return form
}

func (p *huhPrompter) run(f huh.Field, project *string) error {
	form := newForm(f)
	m := &formModel{form: form, project: project, sel: -1}
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
	var project *string
	if id == "project" {
		in = in.Description(projectHint)
		project = &v
	}
	if validate != nil {
		in = in.Validate(validate)
	}
	if err := p.run(in, project); err != nil {
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
	if err := p.run(ms, nil); err != nil {
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
	if err := p.run(huh.NewSelect[string]().Title(title).Description(backHint).Options(huhOptions(opts)...).Value(&v), nil); err != nil {
		return "", err
	}
	return v, nil
}

func (p *huhPrompter) Confirm(id, title, desc string, def bool) (bool, error) {
	v := def
	c := huh.NewConfirm().Title(title).Description(strings.TrimRight(desc+"\n"+backHint, "\n")).Value(&v)
	if err := p.run(c, nil); err != nil {
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
