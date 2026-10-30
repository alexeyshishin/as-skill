package harness

import "testing"

func TestGateWith(t *testing.T) {
	m := &Manifest{RequiresEnv: []string{"A", "B"}, RequiresBin: []string{"x", "y"}}
	env := func(set ...string) func(string) string {
		return func(k string) string {
			for _, s := range set {
				if s == k {
					return "v"
				}
			}
			return ""
		}
	}
	bin := func(have ...string) func(string) bool {
		return func(k string) bool {
			for _, s := range have {
				if s == k {
					return true
				}
			}
			return false
		}
	}
	cases := []struct {
		name  string
		env   func(string) string
		bin   func(string) bool
		want  Miss
		gated bool
	}{
		{"all ok", env("A", "B"), bin("x", "y"), Miss{}, false},
		{"first env missing", env(), bin(), Miss{MissEnv, "A"}, true},
		{"second env missing", env("A"), bin("x", "y"), Miss{MissEnv, "B"}, true},
		{"env before bin", env("A"), bin("x"), Miss{MissEnv, "B"}, true},
		{"first bin missing", env("A", "B"), bin("y"), Miss{MissBin, "x"}, true},
		{"second bin missing", env("A", "B"), bin("x"), Miss{MissBin, "y"}, true},
	}
	for _, c := range cases {
		got, gated := m.GateWith(c.env, c.bin)
		if got != c.want || gated != c.gated {
			t.Errorf("%s: got %v,%v want %v,%v", c.name, got, gated, c.want, c.gated)
		}
	}
	if _, gated := (&Manifest{}).GateWith(env(), bin()); gated {
		t.Error("empty manifest gated")
	}
}

func TestGateDefaults(t *testing.T) {
	t.Setenv("AS_SKILL_GATE_TEST", "")
	m := &Manifest{RequiresEnv: []string{"AS_SKILL_GATE_TEST"}}
	if miss, ok := m.Gate(); !ok || miss.Name != "AS_SKILL_GATE_TEST" {
		t.Fatalf("got %v %v", miss, ok)
	}
	t.Setenv("AS_SKILL_GATE_TEST", "1")
	if _, ok := m.Gate(); ok {
		t.Fatal("gated with env set")
	}
}
