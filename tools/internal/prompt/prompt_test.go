package prompt

import "testing"

func TestIsInteractive(t *testing.T) {
	oldT, oldE := isTerminal, getenv
	defer func() { isTerminal, getenv = oldT, oldE }()
	cases := []struct {
		name string
		env  map[string]string
		tty  bool
		want bool
	}{
		{"tty", map[string]string{"TERM": "xterm"}, true, true},
		{"no tty", map[string]string{"TERM": "xterm"}, false, false},
		{"CI set", map[string]string{"TERM": "xterm", "CI": "true"}, true, false},
		{"dumb", map[string]string{"TERM": "dumb"}, true, false},
	}
	for _, c := range cases {
		isTerminal = func(uintptr) bool { return c.tty }
		getenv = func(k string) string { return c.env[k] }
		if got := IsInteractive(); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
