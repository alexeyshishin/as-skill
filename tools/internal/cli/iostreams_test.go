package cli

import (
	"strings"
	"testing"

	"claude-harness/tools/internal/iostreams"
)

func TestMainListPlatformsWritesStdout(t *testing.T) {
	ios, out, errb := iostreams.Test()
	if err := Main([]string{"list", "platforms"}, ios); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 || errb.Len() != 0 {
		t.Fatalf("out=%q err=%q", out, errb)
	}
}

func TestMainBareInvocationUsageOnStderr(t *testing.T) {
	ios, out, errb := iostreams.Test()
	err := Main(nil, ios)
	if err == nil || err.Error() != "no command given" {
		t.Fatalf("err = %v", err)
	}
	if out.Len() != 0 || !strings.HasPrefix(errb.String(), "as-skill — ") {
		t.Fatalf("out=%q err=%q", out, errb)
	}
}
