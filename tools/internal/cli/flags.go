package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type addOpts struct {
	verb        string
	names       []string
	agents      []string
	project     string
	projectSet  bool
	harnessRoot string
	copy        bool
	yes         bool
	dryRun      bool
	force       bool
	help        bool
}

func parseAddArgs(verb string, args []string) (addOpts, error) {
	opts := addOpts{verb: verb, project: "."}
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasVal := a, "", false
		if strings.HasPrefix(a, "--") {
			if before, after, found := strings.Cut(a, "="); found {
				key, val, hasVal = before, after, true
			}
		}
		takeValue := func() (string, error) {
			if hasVal {
				return val, nil
			}
			i++
			if i >= len(args) {
				return "", fmt.Errorf("%s needs a value", key)
			}
			return args[i], nil
		}
		switch key {
		case "-a", "--agent", "--platform":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.agents = append(opts.agents, v)
		case "-p", "--project":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.project, opts.projectSet = v, true
		case "--harness-root":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.harnessRoot = v
		case "--copy":
			if verb == "remove" {
				return opts, fmt.Errorf("--copy applies to add only")
			}
			opts.copy = true
		case "-y", "--yes":
			opts.yes = true
		case "--dry-run":
			opts.dryRun = true
		case "--force":
			opts.force = true
		case "-h", "--help":
			opts.help = true
		default:
			if strings.HasPrefix(a, "-") {
				return opts, fmt.Errorf("unknown flag %q", a)
			}
			opts.names = append(opts.names, a)
		}
	}
	return opts, nil
}

func resolveProject(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, strings.TrimPrefix(p, "~"))
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("project %s: %w", abs, err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("project %s is not a directory", abs)
	}
	if home, err := os.UserHomeDir(); err == nil && sameDir(abs, home) {
		return "", fmt.Errorf("project %s is $HOME; global scope is not supported — pass a project directory with -p", abs)
	}
	return abs, nil
}

func sameDir(a, b string) bool {
	if ra, err := filepath.EvalSymlinks(a); err == nil {
		a = ra
	}
	if rb, err := filepath.EvalSymlinks(b); err == nil {
		b = rb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func gateError(o addOpts) error {
	var q []string
	if len(o.names) == 0 {
		q = append(q, "what: no domain or skill names given (default: none)")
	}
	if !o.projectSet {
		q = append(q, "project (-p): default current directory")
	}
	if len(o.agents) == 0 {
		q = append(q, "platforms (-a): default detected from the project, else claude")
	}
	if o.verb == "add" && !o.copy {
		q = append(q, "mode (--copy): default symlink")
	}
	q = append(q, "confirmation: pass -y to accept defaults, or --dry-run to preview")
	var b strings.Builder
	fmt.Fprintf(&b, "%s needs names and -y (or --dry-run); no interactive dialog yet. Unanswered:\n", o.verb)
	for _, s := range q {
		fmt.Fprintf(&b, "  - %s\n", s)
	}
	fmt.Fprintf(&b, "example: as-skill %s code core -p <path> -y", o.verb)
	return fmt.Errorf("%s", b.String())
}
