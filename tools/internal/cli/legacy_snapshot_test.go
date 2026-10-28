package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyInstallStdoutSnapshot(t *testing.T) {
	h := realHarness(t)
	for _, mode := range []string{"link", "copy"} {
		t.Run(mode, func(t *testing.T) {
			p := t.TempDir()
			args := []string{"install", "domains", "code", "git", "core", "--project", p, "--harness-root", h}
			if mode == "copy" {
				args = append(args, "--copy")
			}
			out, err := run(t, args...)
			if err != nil {
				t.Fatal(err)
			}
			got := strings.ReplaceAll(strings.ReplaceAll(out, p, "<P>"), h, "<H>") + "--- lockfile ---\n" + strings.ReplaceAll(normLock(t, p), h, "<H>") + "\n"
			path := filepath.Join("testdata", "legacy_"+mode+".txt")
			if os.Getenv("UPDATE_GOLDEN") != "" {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("stdout snapshot differs from %s (UPDATE_GOLDEN=1 to regenerate)\n--- got ---\n%s", path, got)
			}
		})
	}
}
