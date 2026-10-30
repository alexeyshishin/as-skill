package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
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

type listValue struct{ p *[]string }

func (l listValue) String() string {
	if l.p == nil {
		return ""
	}
	return strings.Join(*l.p, ",")
}

func (l listValue) Set(v string) error {
	*l.p = append(*l.p, v)
	return nil
}

func (l listValue) Type() string { return "string" }

func newFlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("as-skill", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	fs.SortFlags = false
	return fs
}

func hiddenAlias(fs *pflag.FlagSet, name string, target *[]string) {
	fs.Var(listValue{target}, name, "")
	_ = fs.MarkHidden(name)
}

func parseFlags(fs *pflag.FlagSet, args []string) ([]string, error) {
	if err := fs.Parse(args); err != nil {
		return nil, flagError(args, err)
	}
	if fs.NArg() == 0 {
		return nil, nil
	}
	return fs.Args(), nil
}

func flagError(args []string, err error) error {
	var notExist *pflag.NotExistError
	var required *pflag.ValueRequiredError
	var syntax *pflag.InvalidSyntaxError
	switch {
	case errors.Is(err, pflag.ErrHelp):
		return fmt.Errorf("unknown flag %q", findArg(args, func(a string) bool {
			return a == "-h" || a == "--help" || strings.HasPrefix(a, "--help=")
		}, "--help"))
	case errors.As(err, &notExist):
		short := notExist.GetSpecifiedShortnames()
		name := notExist.GetSpecifiedName()
		return fmt.Errorf("unknown flag %q", findArg(args, func(a string) bool {
			if short != "" {
				return a == "-"+short
			}
			return a == "--"+name || strings.HasPrefix(a, "--"+name+"=")
		}, flagLabel(name, short)))
	case errors.As(err, &required):
		return fmt.Errorf("%s needs a value", flagLabel(required.GetSpecifiedName(), required.GetSpecifiedShortnames()))
	case errors.As(err, &syntax):
		return fmt.Errorf("unknown flag %q", syntax.GetSpecifiedFlag())
	}
	return err
}

func flagLabel(name, short string) string {
	if short != "" {
		return "-" + name
	}
	return "--" + name
}

func findArg(args []string, match func(string) bool, fallback string) string {
	for _, a := range args {
		if match(a) {
			return a
		}
	}
	return fallback
}

func parseAddArgs(verb string, args []string) (addOpts, error) {
	opts := addOpts{verb: verb, project: "."}
	fs := newFlagSet()
	fs.VarP(listValue{&opts.agents}, "agent", "a", "")
	hiddenAlias(fs, "platform", &opts.agents)
	fs.StringVarP(&opts.project, "project", "p", ".", "")
	fs.StringVar(&opts.harnessRoot, "harness-root", "", "")
	_ = fs.MarkHidden("harness-root")
	fs.BoolVar(&opts.copy, "copy", false, "")
	fs.BoolVarP(&opts.yes, "yes", "y", false, "")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "")
	fs.BoolVar(&opts.force, "force", false, "")
	_ = fs.MarkHidden("force")
	fs.BoolVarP(&opts.help, "help", "h", false, "")
	names, err := parseFlags(fs, args)
	if err != nil {
		return opts, err
	}
	if verb == "remove" && fs.Changed("copy") {
		return opts, fmt.Errorf("--copy applies to add only")
	}
	opts.projectSet = fs.Changed("project")
	opts.names = names
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
	fmt.Fprintf(&b, "%s needs names and -y (or --dry-run); the interactive dialog needs a terminal. Unanswered:\n", o.verb)
	for _, s := range q {
		fmt.Fprintf(&b, "  - %s\n", s)
	}
	fmt.Fprintf(&b, "example: as-skill %s code core -p <path> -y", o.verb)
	return fmt.Errorf("%s", b.String())
}
