package iostreams

import (
	"bytes"
	"io"
	"os"
)

type IOStreams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

type stdout struct{}

func (stdout) Write(p []byte) (int, error) { return os.Stdout.Write(p) }

type stderr struct{}

func (stderr) Write(p []byte) (int, error) { return os.Stderr.Write(p) }

type stdin struct{}

func (stdin) Read(p []byte) (int, error) { return os.Stdin.Read(p) }

func Std() *IOStreams {
	return &IOStreams{In: stdin{}, Out: stdout{}, Err: stderr{}}
}

func Test() (*IOStreams, *bytes.Buffer, *bytes.Buffer) {
	out, errb := &bytes.Buffer{}, &bytes.Buffer{}
	return &IOStreams{In: &bytes.Buffer{}, Out: out, Err: errb}, out, errb
}

func (s *IOStreams) Stdout() io.Writer {
	if s == nil || s.Out == nil {
		return stdout{}
	}
	return s.Out
}

func (s *IOStreams) Stderr() io.Writer {
	if s == nil || s.Err == nil {
		return stderr{}
	}
	return s.Err
}

func (s *IOStreams) Stdin() io.Reader {
	if s == nil || s.In == nil {
		return stdin{}
	}
	return s.In
}
