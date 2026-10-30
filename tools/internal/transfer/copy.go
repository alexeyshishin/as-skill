package transfer

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"claude-harness/tools/internal/lockfile"
)

func SkipTop(kind lockfile.Kind, name string) bool {
	return kind == lockfile.KindHooks && name == "README.md"
}

func findHarnessRoot(src string) string {
	abs, err := filepath.Abs(src)
	if err != nil {
		return src
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		if info, err := os.Stat(filepath.Join(dir, "domains")); err == nil && info.IsDir() {
			return dir
		}
		if filepath.Dir(dir) == dir {
			return abs
		}
	}
}

func CopyTree(kind lockfile.Kind, src, dst string, dryRun bool) (int, error) {
	return CopyTreeRoot(kind, src, dst, findHarnessRoot(src), dryRun)
}

func CopyTreeRoot(kind lockfile.Kind, src, dst, root string, dryRun bool) (int, error) {
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
	return copyDir(kind, src, dst, "", realRoot, map[string]bool{realSrc: true}, dryRun)
}

func inside(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func copyDir(kind lockfile.Kind, srcDir, dstDir, relBase, realRoot string, stack map[string]bool, dryRun bool) (int, error) {
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
		if SkipTop(kind, rel) {
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
			if !inside(realRoot, resolved) {
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
			n, err := copyDir(kind, realPath, target, rel, realRoot, stack, dryRun)
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

func WarnHooksNotExecutable(srcDir string) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sh") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Mode().Perm()&0o111 != 0 {
			continue
		}
		fmt.Fprintf(os.Stderr, "as-skill: warning: %s is not executable; the symlinked hook won't be either (fix it in domains/)\n", filepath.Join(srcDir, e.Name()))
	}
}
