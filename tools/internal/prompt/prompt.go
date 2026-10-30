package prompt

import (
	"errors"
	"os"

	"github.com/mattn/go-isatty"
)

type Option struct {
	Value, Label string
	Selected     bool
}

type Prompter interface {
	Input(id, title, def string, validate func(string) error) (string, error)
	MultiSelect(id, title string, opts []Option, validate func([]string) error) ([]string, error)
	MultiSelectOptional(id, title string, opts []Option) ([]string, error)
	Select(id, title string, opts []Option, def string) (string, error)
	Confirm(id, title, desc string, def bool) (bool, error)
}

var ErrAborted = errors.New("aborted")

var ErrBack = errors.New("back")

var (
	isTerminal = func(fd uintptr) bool {
		return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
	}
	getenv = os.Getenv
)

func IsInteractive() bool {
	if getenv("CI") != "" || getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(os.Stdin.Fd()) && isTerminal(os.Stderr.Fd())
}
