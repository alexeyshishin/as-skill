package cli

import (
	"testing"

	"claude-harness/tools/internal/harness"
)

func TestGateMessages(t *testing.T) {
	t.Setenv("AS_SKILL_GATE_X", "")
	t.Setenv("PATH", t.TempDir())
	env := &harness.Manifest{RequiresEnv: []string{"AS_SKILL_GATE_X"}}
	bin := &harness.Manifest{RequiresBin: []string{"nope-bin"}}
	ok := &harness.Manifest{}
	for _, c := range []struct {
		m             *harness.Manifest
		reason, label string
	}{
		{env, "$AS_SKILL_GATE_X is not set", "needs $AS_SKILL_GATE_X"},
		{bin, `the "nope-bin" binary is not on $PATH`, "needs nope-bin on $PATH"},
		{ok, "", ""},
	} {
		if got := gatedReason(c.m); got != c.reason {
			t.Errorf("gatedReason = %q, want %q", got, c.reason)
		}
		if got := gatedLabel(c.m); got != c.label {
			t.Errorf("gatedLabel = %q, want %q", got, c.label)
		}
	}
}
