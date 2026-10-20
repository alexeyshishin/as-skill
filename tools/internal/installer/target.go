package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"claude-harness/tools/internal/registry"
)

const claudeHomeVar = "CLAUDE_HOME"

var varPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func (in *Installer) resolveTarget(m *registry.Manifest, kind string) (string, error) {
	if raw, ok := m.Targets[kind]; ok {
		return expandTarget(raw, in.ProjectRoot)
	}
	return filepath.Join(in.ProjectRoot, ".claude", kind), nil
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
