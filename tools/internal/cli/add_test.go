package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"claude-harness/tools/internal/lockfile"
)

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if rel == "." || rel == filepath.Join(".claude", "skills-lock.json") {
			return nil
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			tg, _ := os.Readlink(p)
			out[rel] = "L:" + tg
		case d.IsDir():
			out[rel] = "D"
		default:
			b, _ := os.ReadFile(p)
			out[rel] = "F:" + string(b)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func normLock(t *testing.T, project string) string {
	t.Helper()
	lock, err := lockfile.Load(project)
	if err != nil {
		t.Fatal(err)
	}
	for i := range lock.Entries {

		lock.Entries[i].InstalledAt = zeroTime
	}
	b, _ := json.MarshalIndent(lock, "", "  ")
	return strings.ReplaceAll(string(b), project, "<P>")
}

func assertSameInstall(t *testing.T, a, b string) {
	t.Helper()
	ta, tb := snapshotTree(t, a), snapshotTree(t, b)
	for k, v := range ta {
		if w, ok := tb[strings.ReplaceAll(k, a, b)]; !ok || strings.ReplaceAll(v, a, "<P>") != strings.ReplaceAll(w, b, "<P>") {
			t.Errorf("tree differs at %s", k)
		}
	}
	for k := range tb {
		if _, ok := ta[k]; !ok {
			t.Errorf("only in legacy tree: %s", k)
		}
	}
	if la, lb := normLock(t, a), normLock(t, b); la != lb {
		t.Errorf("lockfile differs\n--- add ---\n%s\n--- legacy ---\n%s", la, lb)
	}
}

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runD(t, testDeps(), args...)
}

func runD(t *testing.T, d Deps, args ...string) (string, error) {
	t.Helper()
	return capture(t, func() error { return Dispatch(args, d) })
}

func TestAddParityWithLegacyInstall(t *testing.T) {
	h := realHarness(t)
	for _, mode := range []string{"", "--copy"} {
		t.Run("domains"+mode, func(t *testing.T) {
			a, b := t.TempDir(), t.TempDir()
			addArgs := []string{"add", "code", "core", "-y", "-p", a, "--harness-root", h}
			legArgs := []string{"install", "domains", "code", "core", "--project", b, "--harness-root", h}
			if mode != "" {
				addArgs, legArgs = append(addArgs, mode), append(legArgs, mode)
			}
			if _, err := run(t, addArgs...); err != nil {
				t.Fatal(err)
			}
			if _, err := run(t, legArgs...); err != nil {
				t.Fatal(err)
			}
			assertSameInstall(t, a, b)
		})
		t.Run("skill"+mode, func(t *testing.T) {
			a, b := t.TempDir(), t.TempDir()
			addArgs := []string{"add", "caveman", "-y", "-p", a, "--harness-root", h}
			legArgs := []string{"install", "skill", "caveman", "--project", b, "--harness-root", h}
			if mode != "" {
				addArgs, legArgs = append(addArgs, mode), append(legArgs, mode)
			}
			if _, err := run(t, addArgs...); err != nil {
				t.Fatal(err)
			}
			if _, err := run(t, legArgs...); err != nil {
				t.Fatal(err)
			}
			assertSameInstall(t, a, b)
		})
	}
}

func TestAddAllSkipsGated(t *testing.T) {
	h := realHarness(t)
	t.Setenv("OBSIDIAN_VAULT", "")
	p := t.TempDir()
	if _, err := run(t, "add", "all", "-y", "-p", p, "--harness-root", h, "--copy"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(p, ".claude", "skills", "obsidian-ingest")); err == nil {
		t.Fatal("gated domain installed")
	}
	if _, err := os.Lstat(filepath.Join(p, ".claude", "skills", "git-conventional-commits")); err != nil {
		t.Fatal(err)
	}
}

func TestAddGateWritesNothing(t *testing.T) {
	h := realHarness(t)
	for _, verb := range []string{"add", "remove"} {
		for _, args := range [][]string{
			{verb, "git", "--harness-root", h},
			{verb, "-y", "--harness-root", h},
			{verb, "--harness-root", h},
		} {
			p := t.TempDir()
			_, err := run(t, append(args, "-p", p)...)
			if err == nil {
				t.Fatalf("%v: want gate error", args)
			}
			if !strings.Contains(err.Error(), "as-skill "+verb+" code core -p <path> -y") || !strings.Contains(err.Error(), "default") {
				t.Errorf("%v: weak gate message: %v", args, err)
			}
			if ents, _ := os.ReadDir(p); len(ents) != 0 {
				t.Errorf("%v: wrote %d entries", args, len(ents))
			}
		}
	}
}

func TestAddDryRunPassesGateAndWritesNothing(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	out, err := run(t, "add", "git", "--dry-run", "-p", p, "--harness-root", h)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "would install") {
		t.Fatalf("no dry-run output:\n%s", out)
	}
	if ents, _ := os.ReadDir(p); len(ents) != 0 {
		t.Fatal("dry run wrote files")
	}
}

func TestProjectFlagRules(t *testing.T) {
	h := realHarness(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := filepath.Join(home, "proj")
	if err := os.Mkdir(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(home, "f")
	_ = os.WriteFile(file, nil, 0o644)
	for name, p := range map[string]string{"missing": filepath.Join(home, "nope"), "file": file, "home": home, "tilde-home": "~"} {
		if _, err := run(t, "add", "git", "-y", "-p", p, "--harness-root", h); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
	if _, err := run(t, "add", "git", "-y", "-p", "~/proj", "--harness-root", h); err != nil {
		t.Fatalf("~ expansion: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(proj, ".claude", "skills", "git-conventional-commits")); err != nil {
		t.Fatal(err)
	}
}

func TestAddAgentAliasAndDetected(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if err := os.Mkdir(filepath.Join(p, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "add", "git", "-y", "--copy", "-p", p, "--harness-root", h)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "platforms: detected from .codex") {
		t.Fatalf("no detection line:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(p, ".agents", "skills", "git-conventional-commits")); err != nil {
		t.Fatalf("codex skill not placed: %v", err)
	}
	q := t.TempDir()
	out, err = run(t, "add", "git", "-y", "--copy", "-a", "claude", "-p", q, "--harness-root", h)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "platforms:") {
		t.Fatalf("detection line with -a:\n%s", out)
	}
	r := t.TempDir()
	out, _ = run(t, "add", "git", "-y", "--copy", "-p", r, "--harness-root", h)
	if !strings.Contains(out, "platforms: default (claude)") {
		t.Fatalf("no default line:\n%s", out)
	}
}

func TestAddDetectedUnignoredRootFailsBeforeWrite(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	gitInitDir(t, p)
	if err := os.Mkdir(filepath.Join(p, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, ".cursor", "keep"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][]string{{"add", "-A"}, {"-c", "user.email=a@b.c", "-c", "user.name=n", "commit", "-qm", "x"}} {
		if out, err := exec.Command("git", append([]string{"-C", p}, c...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", c, err, out)
		}
	}
	_, err := run(t, "add", "git", "code", "-y", "-p", p, "--harness-root", h)
	if err == nil {
		t.Fatal("want guard failure")
	}
	if !strings.Contains(err.Error(), "-a claude") || !strings.Contains(err.Error(), "--copy") {
		t.Fatalf("message must name -a claude and --copy: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(p, ".claude")); serr == nil {
		t.Fatal("wrote .claude before failing")
	}
	if _, serr := os.Stat(filepath.Join(p, ".agents")); serr == nil {
		t.Fatal("wrote .agents before failing")
	}
	if _, err := run(t, "add", "git", "-y", "-a", "claude", "-p", p, "--harness-root", h); err != nil {
		t.Fatalf("-a claude should work: %v", err)
	}
	q := t.TempDir()
	gitInitDir(t, q)
	_ = os.Mkdir(filepath.Join(q, ".cursor"), 0o755)
	if _, err := run(t, "add", "git", "-y", "--copy", "-p", q, "--harness-root", h); err != nil {
		t.Fatalf("--copy should work: %v", err)
	}
}

func TestAddUnknownName(t *testing.T) {
	h := realHarness(t)
	_, err := run(t, "add", "nonsense", "-y", "-p", t.TempDir(), "--harness-root", h)
	if err == nil || !strings.Contains(err.Error(), "as-skill list skills") || !strings.Contains(err.Error(), "git") {
		t.Fatalf("got %v", err)
	}
}

func TestHelpText(t *testing.T) {
	out, err := run(t, "help")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"add", "remove", "list", "doctor", "Advanced:", "--copy", "--dry-run", "-a, --agent", "hooks", "check"} {
		if !strings.Contains(out, w) {
			t.Errorf("help lacks %q", w)
		}
	}
	for _, w := range []string{"install domain", "--force", "--platform", "--harness-root", "uninstall", "status"} {
		if strings.Contains(out, w) {
			t.Errorf("help must not mention %q", w)
		}
	}
}

func TestDoctorShowsTableNoDriftAfterMultiDomainCopy(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if _, err := run(t, "add", "code", "git", "core", "--copy", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	out, err := doctorCmd(t, h, p)
	if err != nil {
		t.Fatalf("doctor failed: %v\n%s", err, out)
	}
	if !strings.HasPrefix(out, "DOMAIN") {
		t.Fatalf("table must come first:\n%s", out)
	}
	if strings.Contains(out, "DRIFTED") {
		t.Fatalf("DRIFTED after multi-domain copy:\n%s", out)
	}
	st, err := run(t, "status", "-p", p)
	if err != nil || strings.Contains(st, "doctor:") || !strings.HasPrefix(st, "DOMAIN") {
		t.Fatalf("status must be table only: %v\n%s", err, st)
	}
	if _, err := doctorCmd(t, h, p, "-a", "claude"); err != nil {
		t.Fatalf("doctor -a: %v", err)
	}
}

func TestDoctorBadExplicitHarnessRootErrors(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if _, err := run(t, "add", "git", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "doctor", "-p", p, "--harness-root", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "--harness-root rejected") {
		t.Fatalf("want rejection error: %v\n%s", err, out)
	}
}

func TestDoctorWithoutHarness(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if _, err := run(t, "add", "git", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	d := testDeps()
	d.Resolve = func(string) (string, error) { return "", errors.New("no claude-harness checkout found") }
	out, err := runD(t, d, "doctor", "-p", p)
	if err != nil {
		t.Fatalf("doctor must not error without harness: %v\n%s", err, out)
	}
	if !strings.Contains(out, "harness checkout not found") {
		t.Fatalf("no note:\n%s", out)
	}
}

func TestListInstalledMarker(t *testing.T) {
	h := realHarness(t)
	p := t.TempDir()
	if _, err := run(t, "add", "git", "-y", "-p", p, "--harness-root", h); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "list", "-p", p)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(out, "\n") {
		marked := strings.HasSuffix(line, " installed")
		switch {
		case strings.HasPrefix(line, "  git "), strings.Contains(line, "git-conventional-commits"):
			if !marked {
				t.Errorf("want installed marker: %q", line)
			}
		case strings.HasPrefix(line, "  code "), strings.Contains(line, "code-plan"):
			if marked {
				t.Errorf("unexpected marker: %q", line)
			}
		}
	}
	d, err := run(t, "list", "domains")
	if err != nil || !strings.Contains(d, "domains:") || strings.Contains(d, "skills:") {
		t.Fatalf("list domains form: %v\n%s", err, d)
	}
}

var zeroTime time.Time
