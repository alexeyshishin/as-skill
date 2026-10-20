---
name: code-setup
description: Install this mini dev-loop harness into a target project via as-skill's copy mode. Installs the code domain (code-plan/code-build/code-review/code-debug/code-setup skills, their agents, the test-gate hook) plus core skills, then makes sure swarm-report/ exists. Run this from a clone of the harness repo — pass the absolute path to the project you want to equip.
---

# Skill: /setup

Install the harness into another project. You run in a clone of the harness repo; the
target is a different directory the user names.

## Invocation
`/setup <absolute-path-to-target-project>`

If no path is given, ask for one. Do not guess.

## Steps

1. **Resolve paths.**
   - `SRC` = this harness repo root (`$CLAUDE_PROJECT_DIR`).
   - `DST` = the path argument. Expand `~`. Abort if it does not exist or is not a
     directory: "Target `<path>` not found — give the absolute path to your project."
   - Refuse if `DST` equals `SRC` (don't install the harness into itself).

2. **Git check.** If `DST` is not a git repo (`git -C "$DST" rev-parse` fails), tell the
   user and offer to `git init` it. Do not init without a yes.

3. **Get `as-skill`.** If `SRC/as-skill` doesn't exist yet, build it first:
   `(cd "$SRC" && go build -o as-skill ./tools)`.

4. **Install the `code` domain in copy mode.**
   ```bash
   "$SRC/as-skill" install domain code --project "$DST" --with-core --copy
   ```
   `--copy` is deliberate, not the default — see Notes. This places, as real files
   (not symlinks):
   - `$DST/.claude/skills/{code-plan,code-build,code-review,code-debug,code-setup}/`
   - `$DST/.claude/agents/{code-planner,code-skeptic,code-reviewer,code-debugger}.md`
   - `$DST/.claude/hooks/test-gate.sh` (chmod +x'd automatically by copy mode)
   - `$DST/.claude/skills/{caveman,memory-bank,memory-bank-defrag,swarm-report}/`
     (core skills, via `--with-core`)

   `code-setup` itself gets copied along with the other four — `as-skill`'s domain
   install has no per-skill exclusion. That's harmless: the copy inside `DST` can't
   function as a re-installer there (it needs a live harness checkout as
   `--harness-root`), so leave it rather than hand-deleting it after every run.

5. **`swarm-report/`.** Create it if missing: `mkdir -p "$DST/swarm-report"`. No
   manual Memory Bank seeding needed — the `memory-bank` core skill (just installed
   in step 4) bootstraps `.memory-bank/index.md` itself the first time it's needed
   inside `DST` (see its own SKILL.md, "Bootstrapping a new Memory Bank").

6. **Stop hook wiring.** `test-gate.sh` only runs if `DST/.claude/settings.json`
   registers it as a Stop hook. If that file doesn't exist yet, create one that wires
   it in. If it exists, merge the hook entry by hand (dedupe by command string) —
   don't overwrite the user's existing hooks.

7. **Report.** Summarize what `as-skill` installed (its own stdout already lists each
   piece), note whether `swarm-report/` was newly created, and give the next step:
   > Installed (copy mode) into `<DST>`. Open it in Claude Code and run
   > `/plan "<your first feature>"`.

## Notes
- **Why `--copy`, not `as-skill`'s default (symlink).** `code-setup` targets projects
  unrelated to this checkout, elsewhere on the user's machine. Symlinking would
  couple that project's `.claude/` to this checkout's lifetime — move or delete it
  and the target breaks. `--copy` gives a static, independent snapshot instead. This
  is the same one-verb-plus-flag principle `AGENTS.md`'s "Where things deploy"
  section documents for `as-skill install` generally; `code-setup` just always
  chooses the `--copy` side of it, on purpose.
- Idempotent: re-running overwrites the copied skills/agents/hook with the current
  version (`as-skill`'s copy mode does this per file) but never touches
  `.memory-bank/` or `DST`'s own settings hooks.
- This skill only moves files (via `as-skill`) and ensures `swarm-report/` exists. It
  runs no build and edits no user code inside `DST`.
