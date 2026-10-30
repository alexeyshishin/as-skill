// Package iostreams carries the process's input and output streams so command
// code can write through an injected value instead of os.Stdout/os.Stderr.
package iostreams

import (
	"bytes"
	"io"
	"os"
)

// IOStreams holds stdin, stdout and stderr. A nil field falls back to the
// process's standard stream.
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

// Std returns streams that re-read os.Stdout/os.Stderr/os.Stdin on every call,
// so swapping those variables later is honored.
func Std() *IOStreams {
	return &IOStreams{In: stdin{}, Out: stdout{}, Err: stderr{}}
}

// Test returns in-memory streams and their stdout and stderr buffers.
func Test() (*IOStreams, *bytes.Buffer, *bytes.Buffer) {
	out, errb := &bytes.Buffer{}, &bytes.Buffer{}
	return &IOStreams{In: &bytes.Buffer{}, Out: out, Err: errb}, out, errb
}

// Stdout returns the output writer, or the lazy os.Stdout when unset.
func (s *IOStreams) Stdout() io.Writer {
	if s == nil || s.Out == nil {
		return stdout{}
	}
	return s.Out
}

// Stderr returns the error writer, or the lazy os.Stderr when unset.
func (s *IOStreams) Stderr() io.Writer {
	if s == nil || s.Err == nil {
		return stderr{}
	}
	return s.Err
}

// Stdin returns the input reader, or the lazy os.Stdin when unset.
func (s *IOStreams) Stdin() io.Reader {
	if s == nil || s.In == nil {
		return stdin{}
	}
	return s.In
}
