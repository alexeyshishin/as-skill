package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func Run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return errors.New("no command given")
	}
	switch args[0] {
	case "add":
		return runAdd(args[1:])
	case "remove":
		return runRemove(args[1:])
	case "install":
		return runInstall(args[1:])
	case "uninstall":
		return runUninstall(args[1:])
	case "list":
		return runList(args[1:])
	case "status":
		return runStatus(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "hooks":
		return runHooks(args[1:])
	case "check":
		return runCheck(args[1:])
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `as-skill — add claude-harness domains and skills to a project's platform dirs (default .claude/)

Usage:
  as-skill add <name...> -p <path> -y   add domains and/or skills (symlinks by default);
                                         a name is a domain, else a skill; `+"`all`"+` = every domain
  as-skill remove <name...> -p <path> -y
                                         remove what `+"`add`"+` placed (driven by skills-lock.json)
  as-skill list [domains|skills|platforms]
                                         show what's installable; entries installed in -p are
                                         marked; `+"`platforms`"+` prints the platform x kind matrix
  as-skill doctor [-p <path>]            entry table plus source-gone/hash-drift/untracked/stale
                                         checks; exits non-zero on any finding (CI gate)

Flags (add/remove):
  -a, --agent P    claude|codex|cursor|opencode|all, repeatable or comma-separated
                   (default: detected from the project, else claude)
  -p, --project P  target project root (default: current directory; not $HOME)
  --copy           add: copy a static snapshot instead of symlinking
  -y, --yes        take the defaults, no confirmation
  --dry-run        print what would happen, write nothing
  -h               this help

Advanced:
  as-skill hooks enable|disable <name|all> -a P [-p <path>] [--dry-run]
                                         register/unregister placed hook files
                                         (claude, codex, cursor; opencode: unsupported)
  as-skill check                         validate domains/*/manifest.yaml and skills/*/SKILL.md
                                         in this harness checkout

Examples:
  as-skill add code core -p ~/code/my-app -y
  as-skill add git -p . --copy -y
  as-skill add skill-name -a codex -p . -y
  as-skill remove git -p . -y
`)
}
