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
	fmt.Fprint(w, `as-skill — install claude-harness domains and skills into a project's .claude/

Usage:
  as-skill install domain  <name>              symlink one domain (default mode)
  as-skill install domains <name> [name...]    symlink several domains
  as-skill install all                         symlink every domain + core skills
  as-skill install skill   <name>              symlink one skill (domain-owned or core)
  as-skill install ... --copy                  copy a static snapshot instead of
                                                symlinking (same shapes as above; for
                                                sharing, or projects that shouldn't
                                                depend on this checkout's lifetime)
  as-skill uninstall domain|domains|all|skill ... remove whatever `+"`install`"+` placed
                                                (symlinks for link-mode entries, the
                                                real copied files/dirs for copy-mode
                                                entries; never touches foreign paths)
  as-skill list [domains|skills]                show what's installable
  as-skill status [--project PATH]              list lockfile entries: mode,
                                                path, health (OK/MISSING/BROKEN)
  as-skill doctor [--project PATH]
                  [--harness-root PATH]         status, plus source-gone/hash-
                                                drift/untracked checks; exits
                                                non-zero on any finding (CI gate)
  as-skill check                                validate domains/*/manifest.yaml
                                                and skills/*/SKILL.md in this
                                                harness checkout; no --project

Flags (install/uninstall):
  --project PATH        target project root, gets a .claude/ (default ".")
  --harness-root PATH   this repo's checkout (default: auto-detected upward from ".")
  --with-core            also install/uninstall core/skills/* (domain/domains/skill
                          modes; "all" always includes them)
  --copy                 install: copy a static snapshot instead of symlinking
                          (default: symlink)
  --dry-run              print what would happen, write nothing
  --force                install: overwrite existing content at the destination
                          even if as-skill isn't tracking it (default: refuse)

Examples:
  as-skill install domain git --project ~/code/my-app
  as-skill install domains git content --project .
  as-skill install all --project ~/code/my-app --copy
  as-skill install skill obsidian-ingest --project . --copy
  as-skill uninstall domain git --project .
`)
}
