# as-skill

🇷🇺 [Русская версия](README_RUS.md)

A CLI that installs `as-skill` domains and skills into the platform directories of a target project (`-a`, autodetected by default, otherwise `claude` → `.claude/`; see [Platforms](../README.md#platforms)).

## Build

Needs Go ≥ 1.25. The first build (`go build` or `install.sh`) downloads dependencies from the Go module proxy — a network connection is needed once; a clone without network and without a dependency cache will not build. From the repository root (`go.mod` lives there too, so that `domains/` is reachable):

```bash
go build -o as-skill ./tools
```

or run straight from source: `go run ./tools <command>`.

### `install.sh`

`source install.sh` from the repository root builds `as-skill`, adds it to the `PATH` of the current shell (sh/bash/zsh) and saves it for new terminals: the path is written to `~/.config/as-skill/env`, and a single guard line with no machine-specific path goes into the rc file (`~/.zshrc`; `~/.bash_profile` and `~/.bashrc`; otherwise `~/.profile`) — safe for dotfiles repositories. Open terminals pick up the PATH by running `source install.sh` again.

Without `source` (`./install.sh`) the binary is built but the parent shell's PATH does not change — the script will suggest running it via `source`.

### Platforms

`as-skill` is a plain Go binary with no platform-specific code: macOS, Linux, Windows.

- **Windows, `install.sh`** — this is an `sh` script: run it through Git Bash or WSL, or build directly: `go build -o as-skill.exe ./tools`.
- **Windows, symlink mode** — `os.Symlink` requires the `SeCreateSymbolicLinkPrivilege` privilege: you need **Developer Mode** enabled (Windows 10 1703+ / 11) or a terminal **run as administrator**. Otherwise the install fails with a permissions error — install with `--copy` (live edits from `domains/` are then not picked up; a reinstall is needed after changes).
- **fish** — `install.sh` edits only POSIX rc files and does not touch `config.fish`: it detects this from `$SHELL` and prints a hint. Add the PATH manually: `fish_add_path <path to the as-skill checkout>`.

## Usage

| Command | What it does |
|---------|--------------|
| `as-skill add <name...> [-a P] [-p PATH] [--copy] [-y] [--dry-run]` | Installs domains and/or skills. A name is a domain or a skill (a domain wins on a clash); `all` — all domains, those gated by `requires_env` are skipped with a warning. With no arguments in a terminal — a dialog; without a terminal, names and `-y` (or `--dry-run`) are required. Symlink by default, `--copy` — a snapshot |
| `as-skill remove [<name...>] [-a P] [-p PATH] [-y] [--dry-run]` | Removes what `add` installed, using lockfile records (a copy of the repository is not needed). Does not touch foreign or untracked content. Removes the hook registrations of the removed domain from `.claude/settings.json`; `settings.local.json` is not touched. A skill from a copy-domain is removed together with the domain. Without a copy of the repository, hook registrations and `AGENTS.md` blocks remain (with a warning). Old copy records without `items` cannot be removed — reinstall (`add --copy`) or delete the files manually. Paths outside platform directories are rejected |
| `as-skill list [domains\|skills\|platforms]` | What can be installed; what is installed in the `-p` project is marked `installed`; `platforms` — the platform matrix |
| `as-skill doctor [-p PATH] [-a P]` | A table of lockfile records (mode, platforms, path, health OK/MISSING/BROKEN) and checks: SOURCE-GONE, DRIFTED, UNTRACKED, STALE. Warnings that do not affect the exit code: DUPLICATE-SKILL, UNVERIFIED, HOOKS-TRUST. A non-zero exit on any finding — suitable for CI. Without a copy of the repository, lint and SOURCE-GONE are skipped |

Examples:

```bash
as-skill add all -y                                  # all domains, symlink
as-skill add code core -p <path> --copy -y           # code into another project, snapshot
as-skill add git core -p <path> -a codex,cursor -y   # other platforms
as-skill list                                        # what can be installed
as-skill doctor -p <path>                            # what is installed and in what state
as-skill remove code -p <path> -y                    # remove
```

Modes: **symlink** (default) — edits in `domains/` are visible in the project immediately; this is how this repository's own root `.claude/` lives (see [`CONTRIBUTING.md`](../CONTRIBUTING.md)); **`--copy`** — a static snapshot that does not depend on the checkout.

Advanced:

| Command | What it does |
|---------|--------------|
| `as-skill hooks enable\|disable [<name>...\|all] [-d D[,D]] -a P` | Registers/removes hooks in `.claude/settings.json`, `.codex/hooks.json`, `.cursor/hooks.json`; does not touch foreign entries. With `--domain` the names come from the domain sources, and `enable --dry-run --domain` prints `would register`; without `--domain` — from the project's `.<platform>/hooks`. Codex/cursor — per the documentation, not verified |
| `as-skill check` | Validates `domains/*/manifest.yaml` and `skills/*/SKILL.md`; skill names are unique across domains and are not equal to a domain name or `all` |

The repository copy is found automatically: `--harness-root` → up from cwd → up from the binary's real path. A binary built in a checkout works from any project. `-p` is the project you are equipping.

### Dialog

It opens if stdin and stderr are terminals, `CI` is not set and `TERM` is not `dumb` (`add git | cat` opens it too). Otherwise names and `-y` (or `--dry-run`) are required, or it exits with a hint. Prompts are in English. `shift+tab` — a step back (steps answered in advance are skipped), earlier answers remain as defaults.

Names and flags answer their question in advance (names — domains, `-a` — platforms, `-p` — project, `--copy` — mode); `-y` takes all defaults.

1. **Project directory** — cwd by default; `~` is expanded, the path must exist, `$HOME` is rejected.
2. **Domains to add** — multi-select with no preselection, at least one is required. Label `name — description (needs $VAR) [installed]`; domains gated by `requires_env` are rejected until the variable is set.
3. **Customize skills?** — Confirm, default No, if the selected domains have skills. Yes → multi-select "Skills to install from <domain>", all preselected; the domain's rules, agents and hooks are installed as usual. Dialog only: `-y` excludes nothing, separately named skills are not customized. An empty selection is rejected — to skip a domain's skills entirely, do not select the domain.
4. **Platforms** — all four; autodetected ones are preselected and marked `(detected)`, otherwise `claude` is preselected. A platform whose link directory is not in `.gitignore` is marked `[.cursor not git-ignored]` (it will be rejected in symlink mode); with `--copy` or `--force` there is no mark.
5. **Install mode** — `symlink` (default) or `copy`.
6. **Conflicts** — the dialog does a dry run of the install (`Installer.PlanInstall`) and asks:
   - a directory not in `.gitignore` (link mode) — a Select per platform: `drop platform <name>` (default; for a single platform it equals `abort`, so the default is `copy`) / `switch ALL platforms to copy` (the mode is one per run; the summary will show `adjusted:`) / `abort`. After each decision the dry run is repeated;
   - foreign untracked paths — Confirm `Overwrite these N paths?`, default No. Yes → exactly these paths are overwritten (the dialog does not enable `--force`), No → `aborted`.

   Reserved agent names and other errors are a hard error before anything is written.
7. **Hooks to register** (id `hooks`) — multi-select of the hooks of the selected domains, label `name — Event (platforms)`, **nothing by default**. A hook is `domains/<d>/hooks/*.sh` with a `# register:` line (`test-gate-lib.sh` is a library). The step is skipped if there are no such hooks or no platform supports them. Registration happens after the install and the lockfile write, on the platforms from the hook's `# platforms:` line (test-gate, prettier-format, memory-bank-load — claude/codex/cursor; test-gate-record — codex/cursor; memory-bank-logger — claude only). `memory-bank-logger` carries a warning: the session transcript goes to the API, and the background `claude -p` runs with acceptEdits. `-y` and a run without the dialog do **not** register hooks. With `--dry-run` the question is asked, then `would register <name> (<Event>) in <path>` is printed.
8. **Summary and "Install?"** (default Yes; for `remove` — "Remove?"). Lines: `harness:`, `project:`, `domains:`/`skills:`, `deselect:`, `platforms:`, `adjusted:`, `hooks:`, `mode:`, plus `warning:` — the checkout in cwd ≠ the binary's checkout; agents may reference deselected skills; a deselected skill is already installed and kept (`use remove`); platform capability gaps and duplicate skills; `memory-bank-logger`. Declining or Ctrl-C — nothing is written, `as-skill: error: aborted`, exit 1.

`as-skill remove` with no arguments in a terminal: a multi-select of the installed domains and skills from the lockfile; platforms are asked only if the lockfile has more than one; then a summary and confirmation.

### Legacy

Hidden from help: `install domain|domains|all|skill ...`, `uninstall ...`, `status` (table only, exit 0) and the flags `--platform`, `--force`, `--harness-root`. `install`/`uninstall` and `hooks enable|disable` reject `-p` equal to `$HOME`; `status` only reads and is not restricted.

## Packages

`internal/`: `cli` (dispatcher, flag parsing on `spf13/pflag`; imports `doctor`, `harness`, `install`, `lint`, `fsys`, `iostreams`, `lockfile`, `platform`, `prompt` directly); `doctor`, `lint`, `install` build on `harness`; `install` also on `render` and `hookcfg`; `render` imports `lockfile` and `platform`, `hookcfg` — only `platform`; `harness` — `frontmatter`, `fsys`, `lockfile`; the leaves are `platform`, `lockfile`, `fsys`, `frontmatter`, `iostreams`; plus `hooktest` (hook tests).

## Flags

Shared by `add`/`remove` unless noted otherwise:

| Flag | What it does | Default |
|------|--------------|---------|
| `-p, --project PATH` | the target project root: `~` is expanded, the path must exist, `$HOME` is rejected | `.` |
| `-d, --domain D` | `hooks` only: domains (comma-separated or repeated) whose sources the hooks are taken from | — |
| `-a, --agent P` | `claude\|codex\|cursor\|opencode\|all`, comma-separated or repeated; also for `doctor`/`hooks` | autodetected (below) |
| `--copy` | `add` only: a snapshot copy instead of a symlink | off (symlink by default) |
| `-y` | accept the defaults: no questions or confirmations (required without a terminal unless `--dry-run`) | off |
| `--dry-run` | writes nothing, only prints the plan | off |

The default platform is determined from the existing `.codex/`, `.cursor/`, `.opencode/`, `.claude/` (unless it contains only `skills-lock.json`) and from the platforms in the lockfile; if there is nothing, it is `claude`. The output prints the source (`platforms: detected from ...`).

Link mode for `.agents`, `.codex`, `.cursor`, `.opencode` inside a git repository requires the directory to be in `.gitignore` (`git check-ignore`); otherwise it is refused — add the directory or install with `--copy` (`--force` bypasses it). `add -y` does a dry run of the install first and on any conflict (including foreign untracked paths) fails with a hint (`-a claude`, `--copy` or remove the paths) without writing anything.

If a domain's `requires_env` names an unset variable (today only `obsidian` → `$OBSIDIAN_VAULT`), it is an error for an explicitly named domain/skill and a warning plus a skip for `all`. The mirror-image `requires_bin` (checked via `$PATH`) is not declared by any domain yet.
