package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"claude-harness/tools/internal/harness"
	"claude-harness/tools/internal/lockfile"
)

const claudeHomeVar = "CLAUDE_HOME"

var varPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func (in *Installer) resolveTarget(m *harness.Manifest, kind lockfile.Kind) (string, error) {
	if raw, ok := m.Targets[string(kind)]; ok {
		return expandTarget(raw, in.ProjectRoot)
	}
	return filepath.Join(in.ProjectRoot, ".claude", string(kind)), nil
}

func expandTarget(raw, projectRoot string) (string, error) {
	var missing string
	out := varPattern.ReplaceAllStringFunc(raw, func(tok string) string {
		name := varPattern.FindStringSubmatch(tok)[1]
		if name == claudeHomeVar {
			return filepath.Join(projectRoot, ".claude")
		}
		v, ok := os.LookupEnv(name)
		if !ok || v == "" {
			missing = name
			return ""
		}
		return v
	})
	if missing != "" {
		return "", fmt.Errorf("target %q references $%s, which is not set", raw, missing)
	}
	return filepath.Clean(out), nil
}
