// Package prompt is the interactive-question seam. Only huh.go imports huh.
package prompt

import (
	"errors"
	"os"

	"github.com/mattn/go-isatty"
)

// Option is one choice in a Select/MultiSelect question.
type Option struct {
	Value, Label string
	Selected     bool
}

// Prompter asks one question per call; id names the question for fakes/tests.
type Prompter interface {
	Input(id, title, def string, validate func(string) error) (string, error)
	MultiSelect(id, title string, opts []Option, validate func([]string) error) ([]string, error)
	// MultiSelectOptional accepts an empty submission as "none" (no hovered fallback).
	MultiSelectOptional(id, title string, opts []Option) ([]string, error)
	Select(id, title string, opts []Option, def string) (string, error)
	Confirm(id, title, desc string, def bool) (bool, error)
}

// ErrAborted is returned when the user cancels (Ctrl-C).
var ErrAborted = errors.New("aborted")

// ErrBack is returned when the user presses shift+tab (go to the previous step).
var ErrBack = errors.New("back")

var (
	isTerminal = func(fd uintptr) bool {
		return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
	}
	getenv = os.Getenv
)

// IsInteractive reports whether dialogs may run: stdin and stderr are
// terminals, CI is unset, TERM is not "dumb".
func IsInteractive() bool {
	if getenv("CI") != "" || getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(os.Stdin.Fd()) && isTerminal(os.Stderr.Fd())
}
