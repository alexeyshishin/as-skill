package cli

import (
	"os"
	"strings"

	"claude-harness/tools/internal/prompt"
)

type projectPrompt struct {
	prompt.Prompter
	project *string
}

func withProject(p prompt.Prompter, project *string) prompt.Prompter {
	return &projectPrompt{Prompter: p, project: project}
}

func (p *projectPrompt) title(id, title string) string {
	if p.project == nil || *p.project == "" || id == "project" || id == "confirm" {
		return title
	}
	return title + " · " + shortPath(*p.project)
}

func (p *projectPrompt) MultiSelect(id, title string, opts []prompt.Option, validate func([]string) error) ([]string, error) {
	return p.Prompter.MultiSelect(id, p.title(id, title), opts, validate)
}

func (p *projectPrompt) MultiSelectOptional(id, title string, opts []prompt.Option) ([]string, error) {
	return p.Prompter.MultiSelectOptional(id, p.title(id, title), opts)
}

func (p *projectPrompt) Select(id, title string, opts []prompt.Option, def string) (string, error) {
	return p.Prompter.Select(id, p.title(id, title), opts, def)
}

func (p *projectPrompt) Confirm(id, title, desc string, def bool) (bool, error) {
	return p.Prompter.Confirm(id, p.title(id, title), desc, def)
}

func shortPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(path, home+string(os.PathSeparator)); ok {
		return "~" + string(os.PathSeparator) + rest
	}
	return path
}
