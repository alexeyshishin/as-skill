package installer

import (
	"reflect"
	"testing"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/platform"
)

func realHarnessReg(t *testing.T) (*Installer, *harness.Registry) {
	t.Helper()
	h, err := harness.ResolveHarnessRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := harness.LoadRegistry(h)
	if err != nil {
		t.Fatal(err)
	}
	return &Installer{HarnessRoot: h}, reg
}

func TestDomainHookNamesPerPlatform(t *testing.T) {
	in, reg := realHarnessReg(t)
	for _, tc := range []struct {
		p    platform.ID
		want []string
	}{
		{platform.Claude, []string{"prettier-format.sh", "test-gate.sh", "memory-bank-load.sh", "memory-bank-logger.sh"}},
		{platform.Cursor, []string{"prettier-format.sh", "test-gate-record.sh", "test-gate.sh", "memory-bank-load.sh"}},
		{platform.OpenCode, nil},
	} {
		got, err := in.DomainHookNames(reg, []string{"code", "core"}, tc.p)
		if err != nil {
			t.Fatal(err)
		}
		if !sameSet(got, tc.want) {
			t.Errorf("%s: got %v want set %v", tc.p, got, tc.want)
		}
	}
	got, _ := in.DomainHookNames(reg, []string{"git"}, platform.Claude)
	if len(got) != 0 {
		t.Errorf("git has no hooks, got %v", got)
	}
}

func TestDomainHooksDeterministic(t *testing.T) {
	in, reg := realHarnessReg(t)
	a, _ := in.DomainHooks(reg, []string{"code", "core"})
	b, _ := in.DomainHooks(reg, []string{"code", "core"})
	if !reflect.DeepEqual(a, b) || len(a) == 0 {
		t.Fatalf("not deterministic: %v vs %v", a, b)
	}
}

func sameSet(a, b []string) bool {
	m := map[string]int{}
	for _, s := range a {
		m[s]++
	}
	for _, s := range b {
		m[s]--
	}
	for _, n := range m {
		if n != 0 {
			return false
		}
	}
	return true
}
