package iostreams

import (
	"os"
	"testing"
)

func swap(t *testing.T, target **os.File) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := *target
	*target = w
	t.Cleanup(func() { *target = old; r.Close(); w.Close() })
	return r
}

func readN(t *testing.T, r *os.File, n int) string {
	t.Helper()
	b := make([]byte, n)
	m, err := r.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	return string(b[:m])
}

func TestStdFollowsSwappedFiles(t *testing.T) {
	s := Std()
	ro := swap(t, &os.Stdout)
	re := swap(t, &os.Stderr)
	s.Stdout().Write([]byte("out"))
	s.Stderr().Write([]byte("err"))
	if got := readN(t, ro, 3); got != "out" {
		t.Fatalf("stdout = %q", got)
	}
	if got := readN(t, re, 3); got != "err" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestZeroValueFallsBackToStd(t *testing.T) {
	var s IOStreams
	ro := swap(t, &os.Stdout)
	re := swap(t, &os.Stderr)
	s.Stdout().Write([]byte("a"))
	s.Stderr().Write([]byte("b"))
	if got := readN(t, ro, 1); got != "a" {
		t.Fatalf("stdout = %q", got)
	}
	if got := readN(t, re, 1); got != "b" {
		t.Fatalf("stderr = %q", got)
	}
	var np *IOStreams
	if np.Stdout() == nil || np.Stderr() == nil || np.Stdin() == nil {
		t.Fatal("nil receiver returned nil stream")
	}
}

func TestTestBuffersSeparate(t *testing.T) {
	s, out, errb := Test()
	s.Stdout().Write([]byte("o"))
	s.Stderr().Write([]byte("e"))
	if out.String() != "o" || errb.String() != "e" {
		t.Fatalf("out=%q err=%q", out, errb)
	}
}
