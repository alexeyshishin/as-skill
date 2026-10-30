package fsys

import "testing"

func TestWithin(t *testing.T) {
	cases := []struct {
		name, root, p    string
		within, strictly bool
	}{
		{"equal", "/a/b", "/a/b", true, false},
		{"child", "/a/b", "/a/b/c", true, true},
		{"nested", "/a/b", "/a/b/c/d", true, true},
		{"dotdot-prefixed child", "/a/b", "/a/b/..foo", true, true},
		{"parent", "/a/b", "/a", false, false},
		{"dotdot rel", "/a/b", "/a/b/..", false, false},
		{"dotdot x", "/a/b", "/a/b/../x", false, false},
		{"sibling with prefix", "/a/b", "/a/bc", false, false},
		{"unclean inside", "/a/b", "/a/b/../b/c", true, true},
		{"unclean equal", "/a/b/", "/a/b/../b", true, false},
		{"rel vs abs", "/a/b", "c", false, false},
		{"abs vs rel", "a", "/a/b", false, false},
		{"root slash", "/", "/x/y", true, true},
		{"root slash equal", "/", "/", true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Within(c.root, c.p); got != c.within {
				t.Errorf("Within(%q,%q)=%v want %v", c.root, c.p, got, c.within)
			}
			if got := StrictlyInside(c.root, c.p); got != c.strictly {
				t.Errorf("StrictlyInside(%q,%q)=%v want %v", c.root, c.p, got, c.strictly)
			}
		})
	}
}
