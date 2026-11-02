package install

import (
	"io"

	"claude-harness/tools/internal/iostreams"
)

type Reporter struct {
	IO *iostreams.IOStreams
}

func (r Reporter) Stdout() io.Writer { return r.IO.Stdout() }

func (r Reporter) Stderr() io.Writer { return r.IO.Stderr() }
