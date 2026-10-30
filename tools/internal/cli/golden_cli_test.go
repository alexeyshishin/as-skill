package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var goldenDir string

var updateGolden = flag.Bool("update", false, "rewrite cli/testdata/golden/*.txt")

// captureBoth runs fn with os.Stdout and os.Stderr swapped to pipes drained
// concurrently; both are restored even if fn panics.
func captureBoth(t *testing.T, fn func() error) (stdout, stderr string, err error) {
	t.Helper()
	oldO, oldE := os.Stdout, os.Stderr
	ro, wo, e1 := os.Pipe()
	if e1 != nil {
		t.Fatal(e1)
	}
	re, we, e2 := os.Pipe()
	if e2 != nil {
		t.Fatal(e2)
	}
	outC, errC := make(chan string), make(chan string)
	drain := func(r *os.File, c chan string) {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		c <- b.String()
	}
	go drain(ro, outC)
	go drain(re, errC)
	func() {
		defer func() {
			wo.Close()
			we.Close()
			os.Stdout, os.Stderr = oldO, oldE
		}()
		os.Stdout, os.Stderr = wo, we
		err = fn()
	}()
	return <-outC, <-errC, err
}

func gWrite(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

// goldenHarness builds a small fake harness: domains alpha (skills, agents,
// rules, hooks), beta (skills, agents), gated (requires_env, skill).
func goldenHarness(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	markHarness(t, h)
	d := func(p ...string) string { return filepath.Join(append([]string{h, "domains"}, p...)...) }
	manifest := func(name, extra string) string {
		return "name: " + name + "\ndescription: " + name + " domain\n" + extra + "\ntargets:\n  rules:  ${CLAUDE_HOME}/rules\n  skills: ${CLAUDE_HOME}/skills\n  agents: ${CLAUDE_HOME}/agents\n"
	}
	gWrite(t, d("alpha", "manifest.yaml"), manifest("alpha", "requires_env: []"), 0o644)
	gWrite(t, d("alpha", "skills", "alpha-one", "SKILL.md"), "---\nname: alpha-one\ndescription: first alpha skill\n---\nbody one\n", 0o644)
	gWrite(t, d("alpha", "skills", "alpha-one", "README.md"), "# alpha-one\n", 0o644)
	gWrite(t, d("alpha", "skills", "alpha-two", "SKILL.md"), "---\nname: alpha-two\ndescription: second alpha skill\n---\nbody two\n", 0o644)
	gWrite(t, d("alpha", "skills", "alpha-two", "README.md"), "# alpha-two\n", 0o644)
	gWrite(t, d("alpha", "agents", "alpha-agent.md"), "---\nname: alpha-agent\ndescription: alpha agent\n---\nagent body\n", 0o644)
	gWrite(t, d("alpha", "agents", "plain.md"), "---\ndescription: agent without name\n---\nplain body\n", 0o644)
	gWrite(t, d("alpha", "rules", "alpha-rule.md"), "# alpha rule\n", 0o644)
	gWrite(t, d("alpha", "hooks", "gate.sh"), "#!/usr/bin/env bash\n# register: Stop\n# platforms: claude,codex,cursor\nexit 0\n", 0o755)
	gWrite(t, d("alpha", "hooks", "claude-only.sh"), "#!/usr/bin/env bash\n# register: SessionEnd\n# platforms: claude\nexit 0\n", 0o755)
	gWrite(t, d("beta", "manifest.yaml"), manifest("beta", "requires_env: []"), 0o644)
	gWrite(t, d("beta", "skills", "beta-one", "SKILL.md"), "---\nname: beta-one\ndescription: beta skill\n---\nbeta body\n", 0o644)
	gWrite(t, d("beta", "agents", "beta-agent.md"), "---\nname: beta-agent\ndescription: beta agent\n---\nbeta agent body\n", 0o644)
	gWrite(t, d("gated", "manifest.yaml"), manifest("gated", "requires_env:\n  - GOLDEN_GATE_VAR"), 0o644)
	gWrite(t, d("gated", "skills", "gated-one", "SKILL.md"), "---\nname: gated-one\ndescription: gated skill\n---\ngated body\n", 0o644)
	return h
}

type gcase struct {
	t     *testing.T
	h, p  string
	out   strings.Builder
	steps int
}

var (
	reTime = regexp.MustCompile(`\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?(Z|[+-]\d\d:\d\d)`)
	reHash = regexp.MustCompile(`\b[0-9a-f]{64}\b`)
)

func (c *gcase) norm(s string) string {
	home, _ := os.UserHomeDir()
	var pairs []string
	add := func(from, to string) {
		if from == "" {
			return
		}
		pairs = append(pairs, from, to)
		if r, err := filepath.EvalSymlinks(from); err == nil && r != from {
			pairs = append(pairs, r, to)
		}
	}
	// longest paths first so nested paths win
	add(c.p, "<P>")
	add(c.h, "<H>")
	add(home, "<HOME>")
	// strings.NewReplacer picks by argument order, so order matters: P/H are temp dirs outside HOME.
	s = strings.NewReplacer(pairs...).Replace(s)
	s = reTime.ReplaceAllString(s, "<TIME>")
	return reHash.ReplaceAllString(s, "<SHA256>")
}

// run executes `as-skill args...` and appends a normalized record.
func (c *gcase) run(args ...string) {
	c.t.Helper()
	so, se, err := captureBoth(c.t, func() error { return Run(args) })
	res := "ok"
	if err != nil {
		res = err.Error()
	}
	c.steps++
	fmt.Fprintf(&c.out, "### step %d: %s\n== stdout ==\n%s== stderr ==\n%s== result ==\n%s\n\n",
		c.steps, c.norm(strings.Join(args, " ")), c.norm(so), c.norm(se), c.norm(res))
}

// args appends the standard -p/--harness-root flags.
func (c *gcase) std(args ...string) []string {
	return append(args, "-p", c.p, "--harness-root", c.h)
}

func (c *gcase) tree() {
	var lines []string
	_ = filepath.Walk(c.p, func(path string, fi os.FileInfo, err error) error {
		if err != nil || path == c.p {
			return nil
		}
		rel, _ := filepath.Rel(c.p, path)
		if fi.Mode()&os.ModeSymlink != 0 {
			tgt, _ := os.Readlink(path)
			lines = append(lines, rel+" -> "+tgt)
		} else if !fi.IsDir() {
			lines = append(lines, rel)
		}
		return nil
	})
	fmt.Fprintf(&c.out, "### tree\n%s\n", c.norm(strings.Join(lines, "\n")))
}

func goldenCase(t *testing.T, name string, body func(c *gcase)) {
	t.Run(name, func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("GOLDEN_GATE_VAR", "x")
		os.Unsetenv("GOLDEN_GATE_VAR")
		c := &gcase{t: t, h: goldenHarness(t), p: t.TempDir()}
		body(c)
		got := c.out.String()
		path := filepath.Join(goldenDir, name+".txt")
		if *updateGolden {
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			return
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (run with -update)", err)
		}
		if got != string(want) {
			t.Fatalf("golden %s differs (-update to rewrite):\n%s", path, lineDiff(string(want), got))
		}
	})
}

func lineDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	var b strings.Builder
	n := max(len(w), len(g))
	shown := 0
	for i := 0; i < n && shown < 40; i++ {
		var a, c string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			c = g[i]
		}
		if a != c {
			fmt.Fprintf(&b, "line %d:\n- %s\n+ %s\n", i+1, a, c)
			shown++
		}
	}
	return b.String()
}

func TestGoldenCLI(t *testing.T) {
	var werr error
	goldenDir, werr = filepath.Abs(filepath.Join("testdata", "golden"))
	if werr != nil {
		t.Fatal(werr)
	}
	// add: platforms x mode
	for _, mode := range []string{"link", "copy"} {
		extra := []string{}
		if mode == "copy" {
			extra = []string{"--copy"}
		}
		goldenCase(t, "add_claude_"+mode, func(c *gcase) {
			c.run(c.std(append([]string{"add", "alpha", "beta", "-y", "-a", "claude"}, extra...)...)...)
			c.tree()
		})
		goldenCase(t, "add_all_platforms_"+mode, func(c *gcase) {
			c.run(c.std(append([]string{"add", "alpha", "beta", "-y", "-a", "all"}, extra...)...)...)
			c.tree()
		})
		goldenCase(t, "add_dry_run_"+mode, func(c *gcase) {
			c.run(c.std(append([]string{"add", "alpha", "beta", "--dry-run", "-a", "claude"}, extra...)...)...)
			c.run(c.std(append([]string{"add", "alpha", "--dry-run", "-a", "all"}, extra...)...)...)
			c.tree()
		})
	}
	goldenCase(t, "add_skill_only", func(c *gcase) {
		c.run(c.std("add", "alpha-one", "-y")...)
		c.run(c.std("add", "alpha-one", "beta-one", "-y", "--copy")...)
		c.tree()
	})
	goldenCase(t, "add_foreign_conflict", func(c *gcase) {
		gWrite(t, filepath.Join(c.p, ".claude", "skills", "alpha-one", "SKILL.md"), "foreign\n", 0o644)
		gWrite(t, filepath.Join(c.p, ".claude", "agents", "alpha-agent.md"), "foreign agent\n", 0o644)
		c.run(c.std("add", "alpha", "-y")...)
		c.run(c.std("add", "alpha", "-y", "--copy")...)
		c.tree()
	})
	goldenCase(t, "add_dup_skill_and_gaps", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y", "-a", "claude,cursor")...)
		c.run(c.std("add", "alpha", "-y", "-a", "claude,codex", "--copy")...)
		c.tree()
	})
	goldenCase(t, "add_gated", func(c *gcase) {
		c.run(c.std("add", "gated", "-y")...)
		c.run(c.std("add", "all", "-y")...)
		c.run(c.std("add", "gated-one", "-y")...)
		c.tree()
	})
	goldenCase(t, "add_errors", func(c *gcase) {
		c.run(c.std("add", "nope", "-y")...)
		c.run(c.std("add")...)
		c.run(c.std("add", "alpha", "-a", "bogus", "-y")...)
		c.run("add", "alpha", "-y", "-p", c.p, "--harness-root", filepath.Join(c.p, "nowhere"))
	})
	goldenCase(t, "add_hooks_dry_run", func(c *gcase) {
		// hook registration is dialog-only (default none, -y never): record what -y/--dry-run print.
		c.run(c.std("add", "alpha", "--copy", "-y", "--dry-run", "-a", "claude,codex")...)
		c.run(c.std("add", "alpha", "--copy", "-y", "-a", "claude,codex")...)
		c.tree()
	})

	// remove
	goldenCase(t, "remove_link", func(c *gcase) {
		c.run(c.std("add", "alpha", "beta", "-y")...)
		c.run(c.std("remove", "alpha", "-y")...)
		c.tree()
		c.run(c.std("remove", "beta", "-y", "--dry-run")...)
		c.run(c.std("remove", "beta", "-y")...)
		c.tree()
	})
	goldenCase(t, "remove_copy", func(c *gcase) {
		c.run(c.std("add", "alpha", "beta", "-y", "--copy")...)
		c.run(c.std("remove", "alpha", "-y")...)
		c.run(c.std("remove", "beta-one", "-y")...)
		c.tree()
	})
	goldenCase(t, "remove_codex", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y", "-a", "codex", "--copy")...)
		c.run(c.std("remove", "alpha", "-y", "-a", "codex")...)
		c.tree()
	})
	goldenCase(t, "remove_partial_platform", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y", "-a", "claude,codex", "--copy")...)
		c.run(c.std("remove", "alpha", "-y", "-a", "codex")...)
		c.tree()
		c.run(c.std("remove", "alpha", "-y", "-a", "claude")...)
		c.tree()
	})
	goldenCase(t, "remove_no_harness", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y")...)
		c.run("remove", "alpha", "-y", "-p", c.p, "--harness-root", filepath.Join(c.p, "nowhere"))
		// no flag: auto-resolution fails (package seam), remove falls back to no-harness mode
		old := resolveHarness
		resolveHarness = func(string) (string, error) { return "", errors.New("no harness (stub)") }
		defer func() { resolveHarness = old }()
		c.run("remove", "alpha", "-y", "-p", c.p)
		c.tree()
	})
	goldenCase(t, "remove_unknown", func(c *gcase) {
		c.run(c.std("remove", "ghost", "-y")...)
		c.run(c.std("add", "alpha", "-y")...)
		c.run(c.std("remove", "ghost", "-y")...)
		c.run(c.std("remove")...)
	})

	// list
	goldenCase(t, "list", func(c *gcase) {
		for _, a := range [][]string{{"list"}, {"list", "domains"}, {"list", "skills"}, {"list", "platforms"}} {
			c.run(c.std(a...)...)
		}
		c.run(c.std("add", "alpha-one", "beta", "-y")...)
		for _, a := range [][]string{{"list"}, {"list", "domains"}, {"list", "skills"}} {
			c.run(c.std(a...)...)
		}
		c.run(c.std("list", "bogus")...)
	})

	// doctor
	goldenCase(t, "doctor_clean", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y")...)
		c.run(c.std("add", "beta", "-y", "--copy")...)
		c.run(c.std("doctor")...)
	})
	goldenCase(t, "doctor_empty", func(c *gcase) {
		c.run(c.std("doctor")...)
	})
	goldenCase(t, "doctor_missing", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y", "--copy")...)
		if err := os.Remove(filepath.Join(c.p, ".claude", "skills", "alpha-one", "SKILL.md")); err != nil {
			t.Fatal(err)
		}
		c.run(c.std("doctor")...)
	})
	goldenCase(t, "doctor_drifted", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y", "--copy")...)
		gWrite(t, filepath.Join(c.p, ".claude", "skills", "alpha-one", "SKILL.md"), "hand edited\n", 0o644)
		c.run(c.std("doctor")...)
	})
	goldenCase(t, "doctor_untracked", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y")...)
		gWrite(t, filepath.Join(c.p, ".claude", "skills", "stray", "SKILL.md"), "---\nname: stray\n---\n", 0o644)
		c.run(c.std("doctor")...)
	})
	goldenCase(t, "doctor_source_gone", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y")...)
		if err := os.RemoveAll(filepath.Join(c.h, "domains", "alpha", "skills", "alpha-two")); err != nil {
			t.Fatal(err)
		}
		c.run(c.std("doctor")...)
	})
	goldenCase(t, "doctor_no_harness", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y", "--copy")...)
		c.run("doctor", "-p", c.p, "--harness-root", filepath.Join(c.p, "nowhere"))
		old := resolveHarness
		resolveHarness = func(string) (string, error) { return "", errors.New("no harness (stub)") }
		defer func() { resolveHarness = old }()
		c.run("doctor", "-p", c.p)
	})

	// check
	goldenCase(t, "check_clean", func(c *gcase) {
		t.Chdir(c.h)
		c.run("check")
	})
	goldenCase(t, "check_name_mismatch", func(c *gcase) {
		gWrite(t, filepath.Join(c.h, "domains", "beta", "skills", "beta-one", "SKILL.md"), "---\nname: other-name\ndescription: beta skill\n---\nbeta body\n", 0o644)
		t.Chdir(c.h)
		c.run("check")
	})

	// hooks
	goldenCase(t, "hooks_claude", func(c *gcase) {
		c.run(c.std("add", "alpha", "-y", "--copy")...)
		c.run(c.std("hooks", "enable", "all", "-a", "claude")...)
		c.run(c.std("hooks", "enable", "gate", "-a", "claude", "--dry-run")...)
		c.run(c.std("hooks", "disable", "gate", "-a", "claude")...)
		c.run(c.std("hooks", "enable", "-d", "alpha", "-a", "claude")...)
		c.run(c.std("hooks", "disable", "all", "-a", "claude")...)
		c.run(c.std("hooks", "enable", "claude-only", "-a", "codex")...)
		c.run(c.std("add", "alpha", "-y", "--copy", "-a", "codex")...)
		c.run(c.std("hooks", "enable", "all", "-a", "codex")...)
		c.run(c.std("hooks", "disable", "all", "-a", "codex")...)
		c.run(c.std("hooks", "bogus")...)
		c.tree()
	})

	// top level
	goldenCase(t, "top_level", func(c *gcase) {
		c.run()
		c.run("help")
		c.run("-h")
		c.run("frobnicate")
	})

	// legacy
	goldenCase(t, "legacy_install_status", func(c *gcase) {
		c.run(c.std("install", "domains", "alpha", "beta")...)
		c.run(c.std("status")...)
		c.run(c.std("uninstall", "domains", "beta")...)
		c.run(c.std("status")...)
		c.tree()
	})
}
