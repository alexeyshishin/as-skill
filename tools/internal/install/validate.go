package install

import (
	"fmt"
	"os"
	"path/filepath"

	"claude-harness/tools/internal/fsys"
)

func (in *Installer) ValidateInstall(p *Plan) error {
	if p.rm != nil {
		return nil
	}
	for _, op := range p.Ops {
		if err := in.validateDst(op); err != nil {
			return &OpError{op, stagePlace, err}
		}
	}
	return nil
}

func (in *Installer) validateDst(op Op) error {
	dst := filepath.Clean(op.Dst)
	if op.Type == OpBlock && dst == filepath.Join(in.ProjectRoot, agentsMD) {
		return nil
	}
	if _, err := os.Lstat(in.ProjectRoot); err != nil {
		for _, root := range PlatformRoots(in.ProjectRoot) {
			if fsys.StrictlyInside(root, dst) {
				return nil
			}
		}
		return fmt.Errorf("%s is outside the project's platform dirs", dst)
	}
	return InsideRoots(in.ProjectRoot, dst)
}
