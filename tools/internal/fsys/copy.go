package fsys

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func SkipTop(skipReadme bool, name string) bool {
	return skipReadme && name == "README.md"
}

func CopyTreeRoot(skipReadme bool, src, dst, root string, dryRun bool) (int, error) {
	info, err := os.Stat(src)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("%s is not a directory", src)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return 0, err
	}
	realSrc, err := filepath.EvalSymlinks(src)
	if err != nil {
		return 0, err
	}
	return copyDir(skipReadme, src, dst, "", realRoot, map[string]bool{realSrc: true}, dryRun)
}

func copyDir(skipReadme bool, srcDir, dstDir, relBase, realRoot string, stack map[string]bool, dryRun bool) (int, error) {
	if !dryRun {
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return 0, err
		}
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entries {
		rel := filepath.Join(relBase, e.Name())
		if SkipTop(skipReadme, rel) {
			continue
		}
		path := filepath.Join(srcDir, e.Name())
		target := filepath.Join(dstDir, e.Name())
		isDir := e.IsDir()
		realPath := path
		if e.Type()&fs.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return count, fmt.Errorf("resolving symlink %s: %w", path, err)
			}
			if !Within(realRoot, resolved) {
				return count, fmt.Errorf("symlink %s resolves to %s, outside the harness root %s", path, resolved, realRoot)
			}
			st, err := os.Stat(resolved)
			if err != nil {
				return count, err
			}
			realPath = resolved
			isDir = st.IsDir()
		}
		if isDir {
			real, err := filepath.EvalSymlinks(path)
			if err != nil {
				return count, err
			}
			if stack[real] {
				return count, fmt.Errorf("symlink cycle at %s -> %s", path, real)
			}
			stack[real] = true
			n, err := copyDir(skipReadme, realPath, target, rel, realRoot, stack, dryRun)
			delete(stack, real)
			count += n
			if err != nil {
				return count, err
			}
			continue
		}
		count++
		if dryRun {
			continue
		}
		if err := copyFile(realPath, target); err != nil {
			return count, err
		}
	}
	return count, nil
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(info.Mode().Perm())
}

func MakeShellScriptsExecutable(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sh") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		if err := os.Chmod(p, info.Mode().Perm()|0o111); err != nil {
			return err
		}
	}
	return nil
}

// NonExecutableScripts returns the *.sh files directly under srcDir that lack
// any exec bit.
func NonExecutableScripts(srcDir string) []string {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sh") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Mode().Perm()&0o111 != 0 {
			continue
		}
		out = append(out, filepath.Join(srcDir, e.Name()))
	}
	return out
}
