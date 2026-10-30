package cli

import (
	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/iostreams"
	"claude-harness/tools/internal/prompt"
)

// Deps are the injectable collaborators of the commands; zero fields fall back to the real ones.
type Deps struct {
	IO          *iostreams.IOStreams
	Prompter    func() prompt.Prompter
	Interactive func() bool
	Locator     harness.Locator
	// Resolve, when set, replaces Locator.Locate in doctor and remove.
	Resolve func(flag string) (string, error)
}

func (d Deps) newPrompter() prompt.Prompter {
	if d.Prompter == nil {
		return prompt.NewHuh()
	}
	return d.Prompter()
}

func (d Deps) interactive() bool {
	if d.Interactive == nil {
		return prompt.IsInteractive()
	}
	return d.Interactive()
}

func (d Deps) resolveHarness(flag string) (string, error) {
	if d.Resolve != nil {
		return d.Resolve(flag)
	}
	return d.Locator.Locate(flag)
}
